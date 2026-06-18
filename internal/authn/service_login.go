package authn

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/cache"
	platformjwt "github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/secpolicy"
	"github.com/maintainerd/auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type LoginService interface {
	LoginPublic(ctx context.Context, usernameOrEmail, password string, clientID, tenantID *string) (*LoginResponseDTO, error)
	Login(ctx context.Context, usernameOrEmail, password string, clientID, tenantID *string) (*LoginResponseDTO, error)
	CompleteMFALogin(ctx context.Context, challengeToken, method, code string, assertion []byte, clientID, tenantID *string) (*LoginResponseDTO, error)
	SendMFALoginSMS(ctx context.Context, challengeToken string) error
	BeginMFALoginWebAuthn(ctx context.Context, challengeToken string) (json.RawMessage, error)
	RefreshToken(ctx context.Context, refreshToken string, sessionID string) (*LoginResponseDTO, error)
	GetUserByEmail(ctx context.Context, email string, tenantID int64) (*User, error)
	Logout(ctx context.Context, accessToken string) error
	SetMFAFactorAuthenticator(a MFAFactorAuthenticator)
}

// MFAFactorAuthenticator verifies a login second factor and drives the SMS /
// WebAuthn ceremonies for it. Implemented by the mfa package; injected here so
// the login flow can elevate a freshly issued session to acr=2 without the
// authn package importing mfa.
type MFAFactorAuthenticator interface {
	VerifyFactor(ctx context.Context, userID int64, method, code string, assertion []byte) ([]string, error)
	SendSMSChallenge(ctx context.Context, userID int64) error
	BeginWebAuthnLogin(ctx context.Context, userID int64) (json.RawMessage, error)
	// EnrolledMFAMethods returns the user's usable MFA methods (totp, webauthn,
	// sms, backup_code) from their authoritative sources.
	EnrolledMFAMethods(ctx context.Context, userID int64) ([]string, error)
}

type MFATrustedDeviceAuthenticator interface {
	TrustedDeviceValid(ctx context.Context, userID int64, token string) (bool, error)
	IssueTrustedDevice(ctx context.Context, userID int64, periodDays int) (string, error)
}

type loginService struct {
	db                   *gorm.DB
	clientRepo           ClientRepository
	userRepo             UserRepository
	userTokenRepo        UserTokenRepository
	userIdentityRepo     UserIdentityRepository
	identityProviderRepo IdentityProviderRepository
	authEventService     authevent.AuthEventService
	sessionService       SessionService
	securitySettingRepo  secpolicy.SecuritySettingRepository // nil → skip expiry check
	mfaAuthenticator     MFAFactorAuthenticator              // nil → login MFA disabled
	jtiDenylist          cache.JTIDenylister
}

func NewLoginService(
	db *gorm.DB,
	clientRepo ClientRepository,
	userRepo UserRepository,
	userTokenRepo UserTokenRepository,
	userIdentityRepo UserIdentityRepository,
	identityProviderRepo IdentityProviderRepository,
	authEventService authevent.AuthEventService,
	sessionService SessionService,
	securitySettingRepo secpolicy.SecuritySettingRepository,
	jtiDenylist ...cache.JTIDenylister,
) LoginService {
	var denylist cache.JTIDenylister
	if len(jtiDenylist) > 0 {
		denylist = jtiDenylist[0]
	}
	return &loginService{
		db:                   db,
		clientRepo:           clientRepo,
		userRepo:             userRepo,
		userTokenRepo:        userTokenRepo,
		userIdentityRepo:     userIdentityRepo,
		identityProviderRepo: identityProviderRepo,
		authEventService:     authEventService,
		sessionService:       sessionService,
		securitySettingRepo:  securitySettingRepo,
		jtiDenylist:          denylist,
	}
}

// SetMFAFactorAuthenticator injects the MFA factor verifier used by the login
// MFA second step. Optional dependency (separate setter so the constructor
// signature and its many call sites stay unchanged); nil → login MFA disabled.
func (s *loginService) SetMFAFactorAuthenticator(a MFAFactorAuthenticator) {
	s.mfaAuthenticator = a
}

// findLoginUser resolves the login subject within a single tenant. Lookups are
// tenant-scoped because users are isolated per tenant — the same username/email
// can exist in multiple tenants, so an unscoped lookup could authenticate
// against the wrong tenant's account. A tenantID is therefore required.
func findLoginUser(repo UserRepository, usernameOrEmail string, tenantID int64) (*User, error) {
	user, err := repo.FindByUsernameAndTenantID(usernameOrEmail, tenantID)
	if err == nil && user != nil {
		return user, nil
	}
	if strings.Contains(usernameOrEmail, "@") {
		emailUser, emailErr := repo.FindByEmailAndTenantID(usernameOrEmail, tenantID)
		if emailErr == nil && emailUser != nil {
			return emailUser, nil
		}
		if err == nil {
			return emailUser, emailErr
		}
	}
	return user, err
}

func passwordChangeRequiredLoginResponse() *LoginResponseDTO {
	return &LoginResponseDTO{RequirePasswordChange: true}
}

func (s *loginService) enforceLoginEmailVerification(user *User, tenantID int64) error {
	if user == nil || user.IsEmailVerified || s.securitySettingRepo == nil {
		return nil
	}
	regPolicy := secpolicy.LoadRegistrationPolicy(s.securitySettingRepo, tenantID)
	if regPolicy.RequireEmailVerification {
		return apperror.NewUnauthorized("email is not verified")
	}
	return nil
}

// LoginPublic authenticates users for public-facing applications.
// clientID and tenantID are optional (pointer params).  Priority: clientID > tenantID > system default.
// Used by external applications on port 8081.
func (s *loginService) LoginPublic(ctx context.Context, usernameOrEmail, password string, clientID, tenantID *string) (result *LoginResponseDTO, err error) {
	_, span := otel.Tracer("service").Start(ctx, "login.public")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "login failed")
		} else {
			span.SetStatus(codes.Ok, "")
		}
		span.End()
	}()
	startTime := time.Now()

	// Resolve client early for rate-limiting scope and threat detection.
	resolvedClient, resolveErr := resolveClient(s.clientRepo, clientID, tenantID)
	clientIDStr := ""
	if clientID != nil {
		clientIDStr = *clientID
	}

	var lockoutPolicy *security.RateLimitConfig
	var tenantIDForRL int64
	if resolveErr == nil && resolvedClient != nil &&
		resolvedClient.IdentityProvider != nil {
		tenantIDForRL = resolvedClient.IdentityProvider.TenantID
		lockoutPolicy = secpolicy.LoadLockoutPolicy(s.securitySettingRepo, tenantIDForRL)
	}
	rateLimitIdentifier := fmt.Sprintf("%d:%s", tenantIDForRL, usernameOrEmail)

	// Rate limiting check (SOC2 CC6.1 - Logical Access Controls)
	if err := security.CheckRateLimitWithConfig(rateLimitIdentifier, lockoutPolicy); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_rate_limited",
			UserID:    usernameOrEmail,
			ClientID:  clientIDStr,
			Timestamp: startTime,
			Details:   err.Error(),
		})
		s.authEventService.Log(ctx, authevent.AuthEventInput{
			Category:    authevent.AuthEventCategoryAuthn,
			EventType:   authevent.AuthEventTypeLoginLock,
			Severity:    authevent.AuthEventSeverityWarn,
			Result:      authevent.AuthEventResultFailure,
			Description: ptr.Ptr(err.Error()),
		})
		return nil, err
	}

	// Threat detection — pre-auth velocity / brute-force / risk assessment.
	clientIP := middleware.ClientIPFromContext(ctx)
	threatPolicy := secpolicy.LoadThreatPolicy(s.securitySettingRepo, tenantIDForRL)
	threatDecision := security.AssessLoginThreat(ctx, tenantIDForRL, clientIP, "", threatPolicy)
	if threatDecision.Blocked {
		return nil, apperror.NewUnauthorized("login blocked by threat detection")
	}
	forceStepUp := threatPolicy != nil && threatPolicy.RiskBasedStepUpEnabled && threatDecision.RequiresStepUp

	var user *User
	var client *Client
	var userLookupErr error
	var userIdentitySub string

	// All database operations in transaction (read-only for consistency)
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)

		var txErr error
		client, txErr = resolveClient(txClientRepo, clientID, tenantID)
		if txErr != nil {
			security.LogSecurityEvent(security.SecurityEvent{
				EventType: "login_client_lookup_failure",
				UserID:    usernameOrEmail,
				ClientID:  clientIDStr,
				Timestamp: startTime,
				Details:   "Client lookup failed",
			})
			return apperror.NewUnauthorized("authentication failed")
		}

		if client == nil ||
			client.Status != shared.StatusActive ||
			client.Domain == nil || *client.Domain == "" ||
			client.IdentityProvider == nil {
			security.LogSecurityEvent(security.SecurityEvent{
				EventType: "login_invalid_client",
				UserID:    usernameOrEmail,
				ClientID:  clientIDStr,
				Timestamp: startTime,
				Details:   "Invalid or inactive client configuration",
			})
			return apperror.NewUnauthorized("authentication failed")
		}

		// Get user by username or tenant-scoped email
		user, userLookupErr = findLoginUser(txUserRepo, usernameOrEmail, client.IdentityProvider.TenantID)

		// Fetch user identity to get the Sub claim
		if userLookupErr == nil && user != nil {
			userIdentity, txErr := txUserIdentityRepo.FindByUserIDAndClientID(user.UserID, client.ClientID)
			if txErr == nil && userIdentity != nil {
				userIdentitySub = userIdentity.Sub
			}
		}

		return nil // No error, continue with authentication logic outside transaction
	})

	if err != nil {
		return nil, err
	}

	passwordValid := verifyLoginPassword(user, password, userLookupErr == nil)

	// Check if authentication succeeded
	if !passwordValid || user == nil || user.Password == nil {
		s.recordFailedLogin(ctx, rateLimitIdentifier, clientIDStr, client, startTime, lockoutPolicy)
		return nil, apperror.NewUnauthorized("invalid credentials")
	}

	// Check if user account is active
	if user.Status != shared.StatusActive {
		if user.Status == shared.StatusPending && !user.IsEmailVerified {
			return nil, apperror.NewUnauthorized("email is not verified")
		}

		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_inactive_user",
			UserID:    user.UserUUID.String(),
			ClientID:  clientIDStr,
			Timestamp: startTime,
			Details:   "Attempt to login with inactive user account",
		})

		s.authEventService.Log(ctx, authevent.AuthEventInput{
			TenantID:    client.IdentityProvider.TenantID,
			ActorUserID: &user.UserID,
			IPAddress:   middleware.ClientIPFromContext(ctx),
			UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
			Category:    authevent.AuthEventCategoryAuthn,
			EventType:   authevent.AuthEventTypeLoginFail,
			Severity:    authevent.AuthEventSeverityWarn,
			Result:      authevent.AuthEventResultFailure,
			Description: ptr.Ptr("Attempt to login with inactive account"),
		})

		return nil, apperror.NewUnauthorized("account is not active")
	}
	if err := s.enforceLoginEmailVerification(user, client.IdentityProvider.TenantID); err != nil {
		return nil, err
	}

	// Reset failed attempts on successful authentication.
	security.ResetFailedAttemptsWithConfig(rateLimitIdentifier, lockoutPolicy)

	// Check for compromised credentials at login (post-auth HIBP).
	s.checkCompromisedPassword(ctx, user, password, client.IdentityProvider.TenantID)

	// authevent.Log successful login
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "login_success",
		UserID:    user.UserUUID.String(),
		ClientID:  clientIDStr,
		Timestamp: startTime,
		Details:   fmt.Sprintf("Successful login for user %s", user.Username),
	})

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    client.IdentityProvider.TenantID,
		ActorUserID: &user.UserID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeLoginSuccess,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Successful login for user %s", user.Username)),
	})

	// Check password expiry and set ForcePasswordChange if needed
	s.checkPasswordExpiry(ctx, user, client.IdentityProvider.TenantID)

	if err := s.checkTemporaryPasswordExpiry(ctx, user, client.IdentityProvider.TenantID); err != nil {
		return nil, err
	}

	if user.ForcePasswordChange {
		return passwordChangeRequiredLoginResponse(), nil
	}

	if mfaResponse, mfaErr := s.loginMFAChallengeResponse(ctx, user, client.IdentityProvider.TenantID, forceStepUp); mfaErr != nil || mfaResponse != nil {
		return mfaResponse, mfaErr
	}

	// Record threat success after successful authentication.
	security.RecordLoginThreatSuccess(ctx, client.IdentityProvider.TenantID, user.UserID, middleware.ClientIPFromContext(ctx), middleware.UserAgentFromContext(ctx), threatPolicy)

	// Generate token response
	return s.generateTokenResponse(ctx, userIdentitySub, user, client)
}

// Login authenticates users for internal applications.
// If clientID and tenantID are provided, uses the specified auth client.
// If not provided, uses the default auth client.
// Used by internal applications on port 8080.
func (s *loginService) Login(ctx context.Context, usernameOrEmail, password string, clientID, tenantID *string) (result *LoginResponseDTO, err error) {
	_, span := otel.Tracer("service").Start(ctx, "login.internal")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "login failed")
		} else {
			span.SetStatus(codes.Ok, "")
		}
		span.End()
	}()
	startTime := time.Now()

	// Resolve tenant ID early for rate-limiting scope.
	var lockoutPolicy *security.RateLimitConfig
	var tenantIDForRL int64
	if resolvedClient, resolveErr := resolveClient(s.clientRepo, clientID, tenantID); resolveErr == nil && resolvedClient != nil && resolvedClient.IdentityProvider != nil {
		tenantIDForRL = resolvedClient.IdentityProvider.TenantID
		lockoutPolicy = secpolicy.LoadLockoutPolicy(s.securitySettingRepo, tenantIDForRL)
	}
	rateLimitIdentifier := fmt.Sprintf("%d:%s", tenantIDForRL, usernameOrEmail)

	// Rate limiting check (SOC2 CC6.1 - Logical Access Controls)
	if err := security.CheckRateLimitWithConfig(rateLimitIdentifier, lockoutPolicy); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_rate_limited",
			UserID:    usernameOrEmail,
			ClientID:  "internal",
			Timestamp: startTime,
			Details:   err.Error(),
		})
		s.authEventService.Log(ctx, authevent.AuthEventInput{
			Category:    authevent.AuthEventCategoryAuthn,
			EventType:   authevent.AuthEventTypeLoginLock,
			Severity:    authevent.AuthEventSeverityWarn,
			Result:      authevent.AuthEventResultFailure,
			Description: ptr.Ptr(err.Error()),
		})
		return nil, err
	}

	var user *User
	var client *Client
	var userIdentitySub string
	var userLookupErr error
	var threatPolicy *security.ThreatConfig
	var forceStepUp bool

	// All database operations in transaction for consistency
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)

		var txErr error
		client, txErr = resolveClient(txClientRepo, clientID, tenantID)
		if txErr != nil {
			security.LogSecurityEvent(security.SecurityEvent{
				EventType: "login_client_lookup_failure",
				UserID:    usernameOrEmail,
				ClientID:  "internal",
				Timestamp: startTime,
				Details:   "Client lookup failed",
			})
			return apperror.NewUnauthorized("authentication failed")
		}

		if client == nil ||
			client.Status != shared.StatusActive ||
			client.Domain == nil || *client.Domain == "" {
			security.LogSecurityEvent(security.SecurityEvent{
				EventType: "login_invalid_client",
				UserID:    usernameOrEmail,
				ClientID:  "internal",
				Timestamp: startTime,
				Details:   "Invalid or inactive default client configuration",
			})
			return apperror.NewUnauthorized("authentication failed")
		}

		// Get user by username or tenant-scoped email (timing-safe user lookup)
		user, userLookupErr = findLoginUser(txUserRepo, usernameOrEmail, client.IdentityProvider.TenantID)
		// Note: We don't return error here to maintain timing-safe behavior

		// Fetch user identity to get the Sub claim
		if user != nil {
			userIdentity, txErr := txUserIdentityRepo.FindByUserIDAndClientID(user.UserID, client.ClientID)
			if txErr == nil && userIdentity != nil {
				userIdentitySub = userIdentity.Sub
			}
		}

		return nil // No error, continue with authentication logic outside transaction
	})

	if err != nil {
		return nil, err
	}

	// Threat detection — post-transaction, once tenant is known.
	threatPolicy = secpolicy.LoadThreatPolicy(s.securitySettingRepo, client.IdentityProvider.TenantID)
	threatDecision := security.AssessLoginThreat(ctx, client.IdentityProvider.TenantID, middleware.ClientIPFromContext(ctx), "", threatPolicy)
	if threatDecision.Blocked {
		return nil, apperror.NewUnauthorized("login blocked by threat detection")
	}
	forceStepUp = threatPolicy != nil && threatPolicy.RiskBasedStepUpEnabled && threatDecision.RequiresStepUp

	passwordValid := verifyLoginPassword(user, password, userLookupErr == nil)

	// Check if authentication succeeded
	if !passwordValid || user == nil || user.Password == nil {
		s.recordFailedLogin(ctx, rateLimitIdentifier, "internal", client, startTime, lockoutPolicy)
		return nil, apperror.NewUnauthorized("invalid credentials")
	}

	// Check if user account is active
	if user.Status != shared.StatusActive {
		if user.Status == shared.StatusPending && !user.IsEmailVerified {
			return nil, apperror.NewUnauthorized("email is not verified")
		}

		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_inactive_user",
			UserID:    user.UserUUID.String(),
			ClientID:  "internal",
			Timestamp: startTime,
			Details:   "Attempt to login with inactive user account",
		})

		s.authEventService.Log(ctx, authevent.AuthEventInput{
			TenantID:    client.IdentityProvider.TenantID,
			ActorUserID: &user.UserID,
			IPAddress:   middleware.ClientIPFromContext(ctx),
			UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
			Category:    authevent.AuthEventCategoryAuthn,
			EventType:   authevent.AuthEventTypeLoginFail,
			Severity:    authevent.AuthEventSeverityWarn,
			Result:      authevent.AuthEventResultFailure,
			Description: ptr.Ptr("Attempt to login with inactive account"),
		})

		return nil, apperror.NewUnauthorized("account is not active")
	}
	if err := s.enforceLoginEmailVerification(user, client.IdentityProvider.TenantID); err != nil {
		return nil, err
	}

	// Reset failed attempts on successful authentication
	security.ResetFailedAttemptsWithConfig(rateLimitIdentifier, lockoutPolicy)

	// Check for compromised credentials at login (post-auth HIBP).
	s.checkCompromisedPassword(ctx, user, password, client.IdentityProvider.TenantID)

	// authevent.Log successful login
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "login_success",
		UserID:    user.UserUUID.String(),
		ClientID:  "internal",
		Timestamp: startTime,
		Details:   fmt.Sprintf("Successful internal login for user %s", user.Username),
	})

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    client.IdentityProvider.TenantID,
		ActorUserID: &user.UserID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeLoginSuccess,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Successful internal login for user %s", user.Username)),
	})

	// Check password expiry and set ForcePasswordChange if needed
	s.checkPasswordExpiry(ctx, user, client.IdentityProvider.TenantID)

	if err := s.checkTemporaryPasswordExpiry(ctx, user, client.IdentityProvider.TenantID); err != nil {
		return nil, err
	}

	if user.ForcePasswordChange {
		return passwordChangeRequiredLoginResponse(), nil
	}

	if mfaResponse, mfaErr := s.loginMFAChallengeResponse(ctx, user, client.IdentityProvider.TenantID, forceStepUp); mfaErr != nil || mfaResponse != nil {
		return mfaResponse, mfaErr
	}

	// Record threat success after successful authentication.
	security.RecordLoginThreatSuccess(ctx, client.IdentityProvider.TenantID, user.UserID, middleware.ClientIPFromContext(ctx), middleware.UserAgentFromContext(ctx), threatPolicy)

	// Generate token response
	return s.generateTokenResponse(ctx, userIdentitySub, user, client)
}

// GetUserByEmail looks up a user by email, scoped to the given tenant when
// tenantID > 0. Falls back to a global lookup only when no tenant is specified.
func (s *loginService) GetUserByEmail(ctx context.Context, email string, tenantID int64) (*User, error) {
	_, span := otel.Tracer("service").Start(ctx, "login.getUserByEmail")
	defer span.End()
	var user *User
	var err error
	if tenantID > 0 {
		user, err = s.userRepo.FindByEmailAndTenantID(email, tenantID)
	} else {
		user, err = s.userRepo.FindByEmail(email)
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get user by email failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return user, nil
}

func (s *loginService) Logout(ctx context.Context, accessToken string) error {
	_, span := otel.Tracer("service").Start(ctx, "login.logout")
	defer span.End()

	if accessToken == "" || s.sessionService == nil {
		return nil
	}

	parser := jwtlib.NewParser()
	token, _, err := parser.ParseUnverified(accessToken, jwtlib.MapClaims{})
	if err != nil {
		return nil
	}

	claims := token.Claims.(jwtlib.MapClaims)

	if err := s.denylistLogoutJTI(ctx, claims); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "access token denylist failed")
		return err
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return nil
	}

	userUUID, err := uuid.Parse(sub)
	if err != nil {
		return nil
	}

	user, err := s.userRepo.FindByUUID(userUUID)
	if err != nil || user == nil {
		return nil
	}

	if err := s.sessionService.RevokeAllSessions(ctx, user.UserID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "session revoke failed")
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (s *loginService) denylistLogoutJTI(ctx context.Context, claims jwtlib.MapClaims) error {
	if s.jtiDenylist == nil {
		return nil
	}

	jti, _ := claims["jti"].(string)
	if strings.TrimSpace(jti) == "" {
		return nil
	}

	ttl := jwtClaimTTL(claims["exp"])
	if ttl <= 0 {
		return nil
	}

	return s.jtiDenylist.DenyJTI(ctx, jti, ttl)
}

func jwtClaimTTL(expClaim any) time.Duration {
	var expUnix int64
	switch exp := expClaim.(type) {
	case float64:
		expUnix = int64(exp)
	case int64:
		expUnix = exp
	case int:
		expUnix = int64(exp)
	case json.Number:
		parsed, err := exp.Int64()
		if err != nil {
			return 0
		}
		expUnix = parsed
	default:
		return 0
	}
	return time.Until(time.Unix(expUnix, 0))
}

// checkPasswordExpiry marks ForcePasswordChange on the user if the policy has an
// ExpiryDays > 0 and the password was last changed more than ExpiryDays ago.
// A nil securitySettingRepo or a user with no PasswordChangedAt is a no-op.
func (s *loginService) checkPasswordExpiry(ctx context.Context, user *User, tenantID int64) {
	if s.securitySettingRepo == nil || user.PasswordChangedAt == nil {
		return
	}
	policy := secpolicy.LoadPasswordPolicy(s.securitySettingRepo, tenantID)
	if policy.ExpiryDays <= 0 {
		return
	}
	deadline := user.PasswordChangedAt.AddDate(0, 0, policy.ExpiryDays)
	if time.Now().After(deadline) {
		user.ForcePasswordChange = true
		if _, err := s.userRepo.UpdateByID(user.UserID, map[string]any{"force_password_change": true}); err != nil {
			slog.Warn("failed to set force_password_change", "user_id", user.UserID, "err", err)
		}
	}
}

func (s *loginService) checkTemporaryPasswordExpiry(ctx context.Context, user *User, tenantID int64) error {
	if user == nil || !user.ForcePasswordChange || user.TemporaryPasswordExpiresAt == nil {
		return nil
	}
	if time.Now().Before(*user.TemporaryPasswordExpiresAt) {
		return nil
	}
	slog.Warn("temporary password expired", "user_id", user.UserID, "tenant_id", tenantID)
	if s.authEventService != nil {
		s.authEventService.Log(ctx, authevent.AuthEventInput{
			TenantID:    tenantID,
			ActorUserID: &user.UserID,
			IPAddress:   middleware.ClientIPFromContext(ctx),
			UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
			Category:    authevent.AuthEventCategoryAuthn,
			EventType:   authevent.AuthEventTypeLoginFail,
			Severity:    authevent.AuthEventSeverityWarn,
			Result:      authevent.AuthEventResultFailure,
			Description: ptr.Ptr("Temporary password expired"),
		})
	}
	return apperror.NewUnauthorized("temporary password has expired")
}

func mfaStepUpTTLSeconds(policy *secpolicy.MFAPolicy) int64 {
	return policy.StepUpTTLSeconds()
}

type loginMFAPolicy struct {
	Required             bool     `json:"required"`
	EnforceMFA           bool     `json:"enforce_mfa"`
	Mode                 string   `json:"mode"`
	AllowedMethods       []string `json:"allowed_methods"`
	GracePeriodDays      int      `json:"grace_period_days"`
	PreferredMethod      string   `json:"preferred_method"`
	AdminGracePeriodDays int      `json:"admin_grace_period_days"`
}

func (s *loginService) loginMFAChallengeResponse(ctx context.Context, user *User, tenantID int64, forceStepUp bool) (*LoginResponseDTO, error) {
	if user == nil || s.mfaAuthenticator == nil {
		return nil, nil
	}

	// Check for a valid trusted-device token; skip MFA when recognized.
	trustedDeviceToken := trustedDeviceTokenFromContext(ctx)
	if trustedDeviceToken != "" && s.isTrustedDeviceValid(ctx, user.UserID, tenantID, trustedDeviceToken) {
		return nil, nil
	}

	var policy loginMFAPolicy
	if s.securitySettingRepo != nil {
		if setting, err := s.securitySettingRepo.FindByTenantID(tenantID); err == nil && setting != nil && len(setting.MFAConfig) > 0 {
			if jerr := json.Unmarshal(setting.MFAConfig, &policy); jerr != nil {
				policy = loginMFAPolicy{}
			}
		}
	}
	var mfaPolicy *secpolicy.MFAPolicy
	if mp := secpolicy.LoadMFAPolicy(s.securitySettingRepo, tenantID); mp != nil {
		mfaPolicy = mp
		policy.Mode = mp.Mode
		policy.AllowedMethods = mp.AllowedMethods
		policy.GracePeriodDays = mp.GracePeriodDays
		policy.PreferredMethod = mp.PreferredMethod
		policy.AdminGracePeriodDays = mp.AdminGracePeriodDays
	}
	if policy.Mode == "disabled" {
		return nil, nil
	}
	if policy.Mode == "enforced" || forceStepUp {
		policy.Required = true
	}
	policy.AllowedMethods = normalizeLoginMFAPolicyMethods(policy.AllowedMethods)

	// Ask the mfa service for the user's actual enrolled factors (the single
	// source of truth — covers TOTP, WebAuthn, the verified SMS phone, and backup
	// codes). Reading from the user record alone would miss SMS/WebAuthn.
	enrolled, err := s.mfaAuthenticator.EnrolledMFAMethods(ctx, user.UserID)
	if err != nil {
		return nil, nil
	}

	// Grace-period check: when mode=enforced and the user has no enrolled factor,
	// allow login at acr=1 if we're within the grace window from account creation.
	if policy.Mode == "enforced" && !hasPrimaryMFAFactor(enrolled) {
		graceDays := policy.GracePeriodDays
		if s.userHasAdminRole(user.UserID, tenantID) {
			graceDays = policy.AdminGracePeriodDays
		}
		if graceDays > 0 {
			deadline := user.CreatedAt.AddDate(0, 0, graceDays)
			if time.Now().Before(deadline) {
				return nil, nil
			}
		}
	}

	// Challenge for MFA when the tenant enforces it OR the user has a primary
	// factor enrolled. Backup codes alone never trigger MFA (recovery-only).
	if !policy.Required && !policy.EnforceMFA && !hasPrimaryMFAFactor(enrolled) {
		return nil, nil
	}

	allowedMethods := filterMFAMethodsByPolicy(enrolled, policy.AllowedMethods)
	allowedMethods = preferLoginMFAMethodFirst(allowedMethods, policy.PreferredMethod)
	if len(allowedMethods) == 0 {
		// Tenant forces MFA but the user has nothing usable enrolled → block.
		// For a self-enrolled user with no usable factor, fall through to a
		// normal (acr=1) login rather than locking them out.
		if policy.Required || policy.EnforceMFA {
			return nil, apperror.NewUnauthorized("MFA is required but no supported factors are enrolled")
		}
		return nil, nil
	}

	challengeToken, err := platformjwt.GenerateStepUpChallengeTokenWithContext(
		ctx,
		user.UserUUID.String(),
		time.Duration(mfaStepUpTTLSeconds(mfaPolicy))*time.Second,
		allowedMethods,
	)
	if err != nil {
		return nil, err
	}

	return &LoginResponseDTO{
		MFARequired:       true,
		MFAChallengeToken: &challengeToken,
		MFAAllowedMethods: allowedMethods,
	}, nil
}

// hasPrimaryMFAFactor reports whether the enrolled list contains a primary
// factor (anything other than backup_code). Backup codes are recovery-only and
// never keep MFA active on their own.
func hasPrimaryMFAFactor(enrolled []string) bool {
	for _, m := range enrolled {
		if m != "backup_code" {
			return true
		}
	}
	return false
}

// filterMFAMethodsByPolicy restricts the enrolled methods to those the tenant
// policy allows, preserving the enrolled order. An empty policy list means "no
// restriction" — all enrolled methods are offered.
func filterMFAMethodsByPolicy(enrolled []string, policyMethods []string) []string {
	if len(policyMethods) == 0 {
		return enrolled
	}
	allowed := make(map[string]bool, len(policyMethods))
	for _, m := range policyMethods {
		if m = strings.ToLower(strings.TrimSpace(m)); m != "" {
			allowed[m] = true
		}
	}
	out := make([]string, 0, len(enrolled))
	for _, m := range enrolled {
		if allowed[m] {
			out = append(out, m)
		}
	}
	return out
}

func verifyLoginPassword(user *User, password string, lookupOK bool) bool {
	if lookupOK && user != nil && user.Password != nil {
		return security.ComparePassword([]byte(*user.Password), []byte(password))
	}
	_ = bcrypt.CompareHashAndPassword(security.GetDummyBcryptHash(), []byte(password)) // intentional dummy hash comparison
	return false
}

func (s *loginService) recordFailedLogin(ctx context.Context, rateLimitIdentifier, clientID string, client *Client, at time.Time, lockoutPolicy *security.RateLimitConfig) {
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "login_failure",
		UserID:    rateLimitIdentifier,
		ClientID:  clientID,
		Timestamp: at,
		Details:   "Invalid credentials provided",
	})

	if client != nil {
		threatPolicy := secpolicy.LoadThreatPolicy(s.securitySettingRepo, client.IdentityProvider.TenantID)
		security.RecordLoginThreatFailure(ctx, client.IdentityProvider.TenantID, middleware.ClientIPFromContext(ctx), threatPolicy)
		s.authEventService.Log(ctx, authevent.AuthEventInput{
			TenantID:    client.IdentityProvider.TenantID,
			IPAddress:   middleware.ClientIPFromContext(ctx),
			UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
			Category:    authevent.AuthEventCategoryAuthn,
			EventType:   authevent.AuthEventTypeLoginFail,
			Severity:    authevent.AuthEventSeverityWarn,
			Result:      authevent.AuthEventResultFailure,
			Description: ptr.Ptr("Invalid credentials"),
		})
	}
}

func (s *loginService) generateTokenResponse(ctx context.Context, sub string, user *User, Client *Client) (*LoginResponseDTO, error) {
	// Plain password login is single-factor (acr=1).
	return s.generateTokenResponseWithAuth(ctx, sub, user, Client, []string{platformjwt.AMRPassword}, platformjwt.ACRLevel1)
}

// generateTokenResponseWithAuth issues a full session token set with the given
// amr/acr. Password login uses acr=1; the login MFA second step uses acr=2 so
// the whole session satisfies step-up routes without per-action re-prompts.
func (s *loginService) generateTokenResponseWithAuth(ctx context.Context, sub string, user *User, Client *Client, amr []string, acr string) (*LoginResponseDTO, error) {
	var sessionID string
	policy := resolveEffectiveSessionPolicy(s.securitySettingRepo, Client)
	tokenPolicy := resolveEffectiveTokenPolicy(s.securitySettingRepo, Client)

	// Create a session record and enforce concurrent session limit.
	if s.sessionService != nil {
		if err := enforceConcurrentLimitWithPolicy(ctx, s.sessionService, user.UserUUID, user.UserID, policy); err != nil {
			return nil, err
		}
		ipAddress := middleware.ClientIPFromContext(ctx)
		userAgent := middleware.UserAgentFromContext(ctx)
		sess, err := createSessionWithPolicy(ctx, s.sessionService, user.UserID, ipAddress, userAgent, policy)
		if err != nil {
			return nil, err
		}
		sessionID = sess.UserTokenUUID.String()
	}

	accessToken, idToken, refreshToken, err := generateTokenSetWithAuthContext(ctx, sub, user, Client, tokenAuthContextWithPolicy(amr, acr, sessionID, policy, tokenPolicy))
	if err != nil {
		return nil, err
	}

	resp := buildLoginTokenResponse(accessToken, idToken, refreshToken, time.Now().Unix())
	applyLoginCookiePolicy(resp, policy)
	if policy.AccessTokenTTLSeconds > 0 {
		resp.ExpiresIn = int64(policy.AccessTokenTTLSeconds)
	}
	resp.RequirePasswordChange = user.ForcePasswordChange
	if sessionID != "" {
		resp.SessionID = &sessionID
	}

	return resp, nil
}

// loginMFAMethodAllowed reports whether method is in the challenge token's
// allowed_methods claim (a []any of strings).
func loginMFAMethodAllowed(raw any, method string) bool {
	if method == "" {
		return false
	}
	values, ok := raw.([]any)
	if !ok {
		return false
	}
	for _, v := range values {
		if s, ok := v.(string); ok && s == method {
			return true
		}
	}
	return false
}

func preferLoginMFAMethodFirst(methods []string, preferred string) []string {
	if preferred == "recovery_code" {
		preferred = "backup_code"
	}
	if preferred == "" || len(methods) < 2 {
		return methods
	}
	out := make([]string, 0, len(methods))
	for _, method := range methods {
		if method == preferred {
			out = append(out, method)
			break
		}
	}
	if len(out) == 0 {
		return methods
	}
	for _, method := range methods {
		if method != preferred {
			out = append(out, method)
		}
	}
	return out
}

func normalizeLoginMFAPolicyMethods(methods []string) []string {
	normalized := make([]string, 0, len(methods))
	for _, method := range methods {
		if method == "recovery_code" {
			method = "backup_code"
		}
		normalized = append(normalized, method)
	}
	return normalized
}

// resolveMFAChallengeUser validates a login MFA challenge token and returns the
// referenced user. Shared by the verify / send-sms / webauthn-begin steps.
func (s *loginService) resolveMFAChallengeUser(challengeToken string) (*User, jwtlib.MapClaims, error) {
	claims, err := platformjwt.ValidateStepUpChallengeToken(challengeToken)
	if err != nil {
		return nil, nil, apperror.NewUnauthorized("invalid or expired MFA challenge")
	}
	userUUID, _ := claims["sub"].(string)
	if userUUID == "" {
		return nil, nil, apperror.NewUnauthorized("MFA challenge missing subject")
	}
	user, err := s.userRepo.FindByUUID(userUUID)
	if err != nil || user == nil {
		return nil, nil, apperror.NewUnauthorized("authentication failed")
	}
	return user, claims, nil
}

// CompleteMFALogin verifies the second factor for an in-flight login (identified
// by the challenge token from step one) and, on success, issues a full session
// elevated to acr=2.
func (s *loginService) CompleteMFALogin(ctx context.Context, challengeToken, method, code string, assertion []byte, clientID, tenantID *string) (*LoginResponseDTO, error) {
	if s.mfaAuthenticator == nil {
		return nil, apperror.NewInternal("MFA is not configured", nil)
	}

	user, claims, err := s.resolveMFAChallengeUser(challengeToken)
	if err != nil {
		return nil, err
	}
	if !loginMFAMethodAllowed(claims["allowed_methods"], method) {
		return nil, apperror.NewValidation(fmt.Sprintf("MFA method not allowed: %s", method))
	}
	if user.Status != shared.StatusActive {
		return nil, apperror.NewUnauthorized("account is not active")
	}

	// Resolve the client the same way login does.
	var client *Client
	client, err = resolveClient(s.clientRepo, clientID, tenantID)
	if err != nil || client == nil || client.Status != shared.StatusActive ||
		client.Domain == nil || *client.Domain == "" {
		return nil, apperror.NewUnauthorized("authentication failed")
	}

	identity, ierr := s.userIdentityRepo.FindByUserIDAndClientID(user.UserID, client.ClientID)
	if ierr != nil || identity == nil || identity.Sub == "" {
		return nil, apperror.NewUnauthorized("authentication failed")
	}

	amr, err := s.mfaAuthenticator.VerifyFactor(ctx, user.UserID, method, code, assertion)
	if err != nil {
		s.authEventService.Log(ctx, authevent.AuthEventInput{
			TenantID:    client.IdentityProvider.TenantID,
			ActorUserID: &user.UserID,
			IPAddress:   middleware.ClientIPFromContext(ctx),
			UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
			Category:    authevent.AuthEventCategoryAuthn,
			EventType:   authevent.AuthEventTypeLoginFail,
			Severity:    authevent.AuthEventSeverityWarn,
			Result:      authevent.AuthEventResultFailure,
			Description: ptr.Ptr(fmt.Sprintf("Login MFA verification failed via %s", method)),
		})
		return nil, err
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    client.IdentityProvider.TenantID,
		ActorUserID: &user.UserID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeLoginSuccess,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Login MFA completed via %s", method)),
	})

	resp, err := s.generateTokenResponseWithAuth(ctx, identity.Sub, user, client, amr, platformjwt.ACRLevel2)
	if err != nil {
		return nil, err
	}

	// Issue a trusted-device token when the client opted in.
	if rememberDeviceFromContext(ctx) {
		if deviceToken, tokenErr := s.issueTrustedDeviceToken(ctx, user.UserID, client.IdentityProvider.TenantID); tokenErr == nil {
			resp.TrustedDeviceToken = deviceToken
		}
	}

	return resp, nil
}

// SendMFALoginSMS sends an SMS OTP to the challenged user's phone during login.
func (s *loginService) SendMFALoginSMS(ctx context.Context, challengeToken string) error {
	if s.mfaAuthenticator == nil {
		return apperror.NewInternal("MFA is not configured", nil)
	}
	user, claims, err := s.resolveMFAChallengeUser(challengeToken)
	if err != nil {
		return err
	}
	if !loginMFAMethodAllowed(claims["allowed_methods"], "sms") {
		return apperror.NewValidation("MFA method not allowed: sms")
	}
	return s.mfaAuthenticator.SendSMSChallenge(ctx, user.UserID)
}

// BeginMFALoginWebAuthn starts a passkey assertion ceremony during login and
// returns the assertion options for the browser.
func (s *loginService) BeginMFALoginWebAuthn(ctx context.Context, challengeToken string) (json.RawMessage, error) {
	if s.mfaAuthenticator == nil {
		return nil, apperror.NewInternal("MFA is not configured", nil)
	}
	user, claims, err := s.resolveMFAChallengeUser(challengeToken)
	if err != nil {
		return nil, err
	}
	if !loginMFAMethodAllowed(claims["allowed_methods"], "webauthn") {
		return nil, apperror.NewValidation("MFA method not allowed: webauthn")
	}
	return s.mfaAuthenticator.BeginWebAuthnLogin(ctx, user.UserID)
}

func (s *loginService) issueTrustedDeviceToken(ctx context.Context, userID, tenantID int64) (string, error) {
	mfaPolicy := secpolicy.LoadMFAPolicy(s.securitySettingRepo, tenantID)
	if mfaPolicy == nil || mfaPolicy.Mode == "disabled" || mfaPolicy.TrustedDevicePeriodDays <= 0 {
		return "", nil
	}
	trusted, ok := s.mfaAuthenticator.(MFATrustedDeviceAuthenticator)
	if !ok {
		return "", nil
	}
	return trusted.IssueTrustedDevice(ctx, userID, mfaPolicy.TrustedDevicePeriodDays)
}

func (s *loginService) isTrustedDeviceValid(ctx context.Context, userID, tenantID int64, rawToken string) bool {
	mfaPolicy := secpolicy.LoadMFAPolicy(s.securitySettingRepo, tenantID)
	if mfaPolicy == nil || mfaPolicy.Mode == "disabled" || mfaPolicy.TrustedDevicePeriodDays <= 0 {
		return false
	}
	trusted, ok := s.mfaAuthenticator.(MFATrustedDeviceAuthenticator)
	if !ok {
		return false
	}
	valid, err := trusted.TrustedDeviceValid(ctx, userID, rawToken)
	return err == nil && valid
}

func (s *loginService) userHasAdminRole(userID, tenantID int64) bool {
	if s.userRepo == nil {
		return false
	}
	roles, err := s.userRepo.FindRoles(userID)
	if err != nil {
		return false
	}
	for _, role := range roles {
		if role.TenantID == tenantID && role.Name == shared.RoleSuperAdmin {
			return true
		}
	}
	return false
}

func (s *loginService) checkCompromisedPassword(ctx context.Context, user *User, password string, tenantID int64) {
	if user == nil || password == "" {
		return
	}
	threatPolicy := secpolicy.LoadThreatPolicy(s.securitySettingRepo, tenantID)
	if threatPolicy == nil || !threatPolicy.CompromisedCredentialMonitoringEnabled {
		return
	}
	if security.CheckHIBPPassword(ctx, []byte(password)) {
		slog.Warn("compromised password detected at login, forcing password change", "user_id", user.UserID)
		user.ForcePasswordChange = true
		if _, err := s.userRepo.UpdateByID(user.UserID, map[string]any{"force_password_change": true}); err != nil {
			slog.Warn("failed to set force_password_change for compromised credential", "user_id", user.UserID, "err", err)
		}
		s.authEventService.Log(ctx, authevent.AuthEventInput{
			TenantID:    tenantID,
			ActorUserID: &user.UserID,
			IPAddress:   middleware.ClientIPFromContext(ctx),
			UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
			Category:    authevent.AuthEventCategoryAuthn,
			EventType:   authevent.AuthEventTypeLoginSuccess,
			Severity:    authevent.AuthEventSeverityWarn,
			Result:      authevent.AuthEventResultSuccess,
			Description: ptr.Ptr("Login succeeded with compromised password — password change forced"),
		})
	}
}

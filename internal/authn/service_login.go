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
	LoginPublic(ctx context.Context, usernameOrEmail, password, clientID, providerID string) (*LoginResponseDTO, error)
	Login(ctx context.Context, usernameOrEmail, password string, clientID, providerID *string) (*LoginResponseDTO, error)
	CompleteMFALogin(ctx context.Context, challengeToken, method, code string, assertion []byte, clientID, providerID *string) (*LoginResponseDTO, error)
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

func findLoginUser(repo UserRepository, usernameOrEmail string, tenantID int64) (*User, error) {
	user, err := repo.FindByUsername(usernameOrEmail)
	if err == nil && user != nil {
		return user, nil
	}
	if strings.Contains(usernameOrEmail, "@") && tenantID > 0 {
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

// LoginPublic authenticates users for public-facing applications.
// Requires clientID and providerID to identify the auth client.
// Used by external applications on port 8081.
func (s *loginService) LoginPublic(ctx context.Context, usernameOrEmail, password, clientID, providerID string) (result *LoginResponseDTO, err error) {
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

	// Input validation is now handled at the DTO/handler level

	// Rate limiting check (SOC2 CC6.1 - Logical Access Controls)
	if err := security.CheckRateLimit(usernameOrEmail); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_rate_limited",
			UserID:    usernameOrEmail,
			ClientID:  clientID,
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
	var userLookupErr error
	var userIdentitySub string

	// All database operations in transaction (read-only for consistency)
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)
		txIdentityProviderRepo := s.identityProviderRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)

		// Look up identity provider by identifier to get auth container
		identityProvider, txErr := txIdentityProviderRepo.FindByIdentifier(providerID)
		if txErr != nil {
			security.LogSecurityEvent(security.SecurityEvent{
				EventType: "login_validation_failure",
				UserID:    usernameOrEmail,
				ClientID:  clientID,
				Timestamp: startTime,
				Details:   "Identity provider lookup failed",
			})
			return apperror.NewUnauthorized("authentication failed")
		}

		if identityProvider == nil {
			security.LogSecurityEvent(security.SecurityEvent{
				EventType: "login_validation_failure",
				UserID:    usernameOrEmail,
				ClientID:  clientID,
				Timestamp: startTime,
				Details:   "Identity provider not found",
			})
			return apperror.NewUnauthorized("authentication failed")
		}

		// Get and validate auth client with proper relationship preloading
		client, txErr = txClientRepo.FindByClientIDAndIdentityProvider(clientID, providerID)
		if txErr != nil {
			security.LogSecurityEvent(security.SecurityEvent{
				EventType: "login_client_lookup_failure",
				UserID:    usernameOrEmail,
				ClientID:  clientID,
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
				ClientID:  clientID,
				Timestamp: startTime,
				Details:   "Invalid or inactive client configuration",
			})
			return apperror.NewUnauthorized("authentication failed")
		}

		// Get user by username or tenant-scoped email (timing-safe user lookup)
		user, userLookupErr = findLoginUser(txUserRepo, usernameOrEmail, identityProvider.TenantID)

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
		s.recordFailedLogin(ctx, usernameOrEmail, clientID, client, startTime)
		return nil, apperror.NewUnauthorized("invalid credentials")
	}

	// Check if user account is active
	if user.Status != shared.StatusActive {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "login_inactive_user",
			UserID:    user.UserUUID.String(),
			ClientID:  clientID,
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

	// Reset failed attempts on successful authentication
	security.ResetFailedAttempts(usernameOrEmail)

	// authevent.Log successful login
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "login_success",
		UserID:    user.UserUUID.String(),
		ClientID:  clientID,
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

	if user.ForcePasswordChange {
		return passwordChangeRequiredLoginResponse(), nil
	}

	if mfaResponse, mfaErr := s.loginMFAChallengeResponse(ctx, user, client.IdentityProvider.TenantID); mfaErr != nil || mfaResponse != nil {
		return mfaResponse, mfaErr
	}

	// Generate token response
	return s.generateTokenResponse(ctx, userIdentitySub, user, client)
}

// Login authenticates users for internal applications.
// If clientID and providerID are provided, uses the specified auth client.
// If not provided, uses the default auth client.
// Used by internal applications on port 8080.
func (s *loginService) Login(ctx context.Context, usernameOrEmail, password string, clientID, providerID *string) (result *LoginResponseDTO, err error) {
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

	// Rate limiting check (SOC2 CC6.1 - Logical Access Controls)
	if err := security.CheckRateLimit(usernameOrEmail); err != nil {
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

	// All database operations in transaction for consistency
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)

		// Get auth client - either by client_id and provider_id or default
		var txErr error
		if clientID != nil && providerID != nil {
			// Get auth client by client_id and identity provider identifier
			client, txErr = txClientRepo.FindByClientIDAndIdentityProvider(*clientID, *providerID)
			if txErr != nil {
				security.LogSecurityEvent(security.SecurityEvent{
					EventType: "login_client_lookup_failure",
					UserID:    usernameOrEmail,
					ClientID:  *clientID,
					Timestamp: startTime,
					Details:   "Client lookup by client_id and provider_id failed",
				})
				return apperror.NewUnauthorized("authentication failed")
			}
		} else {
			// Get system client for no-client_id authentication (always the system tenant)
			client, txErr = txClientRepo.FindSystem()
			if txErr != nil {
				security.LogSecurityEvent(security.SecurityEvent{
					EventType: "login_client_lookup_failure",
					UserID:    usernameOrEmail,
					ClientID:  "internal",
					Timestamp: startTime,
					Details:   "Default client lookup failed",
				})
				return apperror.NewUnauthorized("authentication failed")
			}
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

	passwordValid := verifyLoginPassword(user, password, userLookupErr == nil)

	// Check if authentication succeeded
	if !passwordValid || user == nil || user.Password == nil {
		s.recordFailedLogin(ctx, usernameOrEmail, "internal", client, startTime)
		return nil, apperror.NewUnauthorized("invalid credentials")
	}

	// Check if user account is active
	if user.Status != shared.StatusActive {
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

	// Reset failed attempts on successful authentication
	security.ResetFailedAttempts(usernameOrEmail)

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

	if user.ForcePasswordChange {
		return passwordChangeRequiredLoginResponse(), nil
	}

	if mfaResponse, mfaErr := s.loginMFAChallengeResponse(ctx, user, client.IdentityProvider.TenantID); mfaErr != nil || mfaResponse != nil {
		return mfaResponse, mfaErr
	}

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

const loginMFAChallengeTTL = 5 * time.Minute

type loginMFAPolicy struct {
	Required       bool     `json:"required"`
	EnforceMFA     bool     `json:"enforce_mfa"`
	AllowedMethods []string `json:"allowed_methods"`
}

func (s *loginService) loginMFAChallengeResponse(ctx context.Context, user *User, tenantID int64) (*LoginResponseDTO, error) {
	if user == nil || s.mfaAuthenticator == nil {
		return nil, nil
	}

	// Load the tenant MFA policy if one exists. A missing security setting or
	// empty MFAConfig simply means "no tenant-level enforcement" — it must NOT
	// short-circuit the enrolled-factor trigger below.
	var policy loginMFAPolicy
	if s.securitySettingRepo != nil {
		if setting, err := s.securitySettingRepo.FindByTenantID(tenantID); err == nil && setting != nil && len(setting.MFAConfig) > 0 {
			if jerr := json.Unmarshal(setting.MFAConfig, &policy); jerr != nil {
				policy = loginMFAPolicy{}
			}
		}
	}

	// Ask the mfa service for the user's actual enrolled factors (the single
	// source of truth — covers TOTP, WebAuthn, the verified SMS phone, and backup
	// codes). Reading from the user record alone would miss SMS/WebAuthn.
	enrolled, err := s.mfaAuthenticator.EnrolledMFAMethods(ctx, user.UserID)
	if err != nil {
		// Don't lock the user out on a status-read error; fall back to acr=1.
		return nil, nil
	}

	// Challenge for MFA when the tenant enforces it OR the user has a primary
	// factor enrolled. Backup codes alone never trigger MFA (recovery-only).
	if !policy.Required && !policy.EnforceMFA && !hasPrimaryMFAFactor(enrolled) {
		return nil, nil
	}

	allowedMethods := filterMFAMethodsByPolicy(enrolled, policy.AllowedMethods)
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
		loginMFAChallengeTTL,
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

func (s *loginService) recordFailedLogin(ctx context.Context, usernameOrEmail, clientID string, client *Client, at time.Time) {
	security.RecordFailedAttempt(usernameOrEmail)

	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "login_failure",
		UserID:    usernameOrEmail,
		ClientID:  clientID,
		Timestamp: at,
		Details:   "Invalid credentials provided",
	})

	if client != nil {
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

	// Create a session record and enforce concurrent session limit.
	if s.sessionService != nil {
		if err := s.sessionService.EnforceConcurrentLimit(ctx, user.UserUUID, user.UserID); err != nil {
			return nil, err
		}
		ipAddress := middleware.ClientIPFromContext(ctx)
		userAgent := middleware.UserAgentFromContext(ctx)
		sess, err := s.sessionService.CreateSession(ctx, user.UserID, ipAddress, userAgent)
		if err != nil {
			return nil, err
		}
		sessionID = sess.UserTokenUUID.String()
	}

	accessToken, idToken, refreshToken, err := generateTokenSetWithAuthContext(ctx, sub, user, Client, tokenAuthContext{
		AMR:       amr,
		ACR:       acr,
		SessionID: sessionID,
	})
	if err != nil {
		return nil, err
	}

	resp := buildLoginTokenResponse(accessToken, idToken, refreshToken, time.Now().Unix())
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
func (s *loginService) CompleteMFALogin(ctx context.Context, challengeToken, method, code string, assertion []byte, clientID, providerID *string) (*LoginResponseDTO, error) {
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

	// Resolve the client the same way login does (explicit client or system).
	var client *Client
	if clientID != nil && providerID != nil {
		client, err = s.clientRepo.FindByClientIDAndIdentityProvider(*clientID, *providerID)
	} else {
		client, err = s.clientRepo.FindSystem()
	}
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

	return s.generateTokenResponseWithAuth(ctx, identity.Sub, user, client, amr, platformjwt.ACRLevel2)
}

// SendMFALoginSMS sends an SMS OTP to the challenged user's phone during login.
func (s *loginService) SendMFALoginSMS(ctx context.Context, challengeToken string) error {
	if s.mfaAuthenticator == nil {
		return apperror.NewInternal("MFA is not configured", nil)
	}
	user, _, err := s.resolveMFAChallengeUser(challengeToken)
	if err != nil {
		return err
	}
	return s.mfaAuthenticator.SendSMSChallenge(ctx, user.UserID)
}

// BeginMFALoginWebAuthn starts a passkey assertion ceremony during login and
// returns the assertion options for the browser.
func (s *loginService) BeginMFALoginWebAuthn(ctx context.Context, challengeToken string) (json.RawMessage, error) {
	if s.mfaAuthenticator == nil {
		return nil, apperror.NewInternal("MFA is not configured", nil)
	}
	user, _, err := s.resolveMFAChallengeUser(challengeToken)
	if err != nil {
		return nil, err
	}
	return s.mfaAuthenticator.BeginWebAuthnLogin(ctx, user.UserID)
}

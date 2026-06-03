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
	GetUserByEmail(ctx context.Context, email string, tenantID int64) (*User, error)
	Logout(ctx context.Context, accessToken string) error
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
	if s.securitySettingRepo == nil || user == nil {
		return nil, nil
	}

	setting, err := s.securitySettingRepo.FindDefaultByTenantID(tenantID)
	if err != nil || setting == nil || len(setting.MFAConfig) == 0 {
		return nil, nil
	}

	var policy loginMFAPolicy
	if err := json.Unmarshal(setting.MFAConfig, &policy); err != nil {
		return nil, nil
	}
	if !policy.Required && !policy.EnforceMFA {
		return nil, nil
	}

	allowedMethods := loginMFAAllowedMethods(user, policy.AllowedMethods)
	if len(allowedMethods) == 0 {
		return nil, apperror.NewUnauthorized("MFA is required but no supported factors are enrolled")
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

func loginMFAAllowedMethods(user *User, policyMethods []string) []string {
	policyAllows := map[string]bool{}
	for _, method := range policyMethods {
		method = strings.ToLower(strings.TrimSpace(method))
		if method != "" {
			policyAllows[method] = true
		}
	}
	if len(policyAllows) == 0 {
		policyAllows["totp"] = true
		policyAllows["backup_code"] = true
	}

	var methods []string
	if user.IsTOTPEnabled && policyAllows["totp"] {
		methods = append(methods, "totp")
	}
	if userHasAnyMFAFactor(user) && policyAllows["backup_code"] {
		methods = append(methods, "backup_code")
	}
	return methods
}

func userHasAnyMFAFactor(user *User) bool {
	return user.IsTOTPEnabled || user.IsWebAuthnEnabled || user.MFAEnabledAt != nil
}

func verifyLoginPassword(user *User, password string, lookupOK bool) bool {
	if lookupOK && user != nil && user.Password != nil {
		return security.ComparePassword([]byte(*user.Password), []byte(password))
	}
	bcrypt.CompareHashAndPassword(security.GetDummyBcryptHash(), []byte(password)) //nolint:errcheck
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
		AMR:       []string{platformjwt.AMRPassword},
		ACR:       platformjwt.ACRLevel1,
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

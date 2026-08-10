package authn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	platformjwt "github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// LoginService is the login surface actually mounted by the router
// (LoginPublicRoute). The former internal variant — Login, mounted only by a
// LoginRoute nothing ever called — is gone: it was an unreachable, near
// line-for-line duplicate of LoginPublic that still had to be kept in sync, so
// a fix applied to the live path could silently miss the copy.
type LoginService interface {
	LoginPublic(ctx context.Context, usernameOrEmail, password string, clientID, tenantID *string) (*LoginResponseDTO, error)
	CompleteMFALogin(ctx context.Context, challengeToken, method, code string, assertion []byte, clientID, tenantID *string) (*LoginResponseDTO, error)
	SendMFALoginSMS(ctx context.Context, challengeToken string) error
	SendMFALoginEmailOTP(ctx context.Context, challengeToken string) error
	BeginMFALoginWebAuthn(ctx context.Context, challengeToken string) (json.RawMessage, error)
	RefreshToken(ctx context.Context, refreshToken string, sessionID string) (*LoginResponseDTO, error)
	GetUserByEmail(ctx context.Context, email string, tenantID int64) (*User, error)
	Logout(ctx context.Context, accessToken string) error
	ForgetTrustedDevice(ctx context.Context, token string)
	SetMFAFactorAuthenticator(a MFAFactorAuthenticator)
	SetUserLockoutRepository(r UserLockoutRepository)
	SetTokenRevoker(r AccessTokenRevoker)
	MagicLinkMFAChallenge(ctx context.Context, user *User, tenantID int64) (*LoginResponseDTO, error)
	IssueMagicLinkSession(ctx context.Context, sub string, user *User, client *Client) (*LoginResponseDTO, error)
	EnforcePhoneVerification(ctx context.Context, user *User, tenantID int64) error
	SMSMFAChallenge(ctx context.Context, user *User, tenantID int64) (*LoginResponseDTO, error)
}

// MFAFactorAuthenticator verifies a login second factor and drives the SMS /
// WebAuthn ceremonies for it. Implemented by the mfa package; injected here so
// the login flow can elevate a freshly issued session to acr=2 without the
// authn package importing mfa.
type MFAFactorAuthenticator interface {
	VerifyFactor(ctx context.Context, userID int64, method, code string, assertion []byte) ([]string, error)
	SendSMSChallenge(ctx context.Context, userID int64) error
	SendEmailOTPChallenge(ctx context.Context, userID int64) error
	BeginWebAuthnLogin(ctx context.Context, userID int64) (json.RawMessage, error)
	// EnrolledMFAMethods returns the user's usable MFA methods (totp, webauthn,
	// sms, backup_code) from their authoritative sources.
	EnrolledMFAMethods(ctx context.Context, userID int64) ([]string, error)
}

type MFATrustedDeviceAuthenticator interface {
	// TrustedDeviceValid takes the tenant the login is happening in: trust is
	// granted per tenant, so a token issued in one must not be honoured in
	// another.
	TrustedDeviceValid(ctx context.Context, userID, tenantID int64, token string) (bool, error)
	IssueTrustedDevice(ctx context.Context, userID, tenantID int64, deviceID string, periodDays int) (string, error)
	RevokeTrustedDeviceByToken(ctx context.Context, token string) error
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
	lockoutRepo          UserLockoutRepository // nil → lockout tracking disabled
	tokenRevoker         AccessTokenRevoker    // nil → persistent revocation skipped
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

func (s *loginService) SetUserLockoutRepository(r UserLockoutRepository) {
	s.lockoutRepo = r
}

func (s *loginService) SetTokenRevoker(r AccessTokenRevoker) {
	s.tokenRevoker = r
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

// enforceLoginPhoneVerification blocks a login when the tenant requires phone
// verification and the user's phone is not yet verified. It mirrors the email
// gate and makes require_phone_verification a real control rather than the
// presence-only check it was before.
//
// It is applied to login methods that do NOT themselves prove the phone
// (password, magic-link). SMS OTP login is exempt and instead SETS
// IsPhoneVerified on success — a successful OTP IS proof of phone possession, so
// SMS login is the canonical way a user satisfies this requirement (no separate
// pre-auth phone-verification flow is introduced; the existing SMS OTP flow is
// reused).
func (s *loginService) enforceLoginPhoneVerification(user *User, tenantID int64) error {
	if user == nil || user.IsPhoneVerified || s.securitySettingRepo == nil {
		return nil
	}
	regPolicy := secpolicy.LoadRegistrationPolicy(s.securitySettingRepo, tenantID)
	if regPolicy.RequirePhoneVerification {
		return apperror.NewUnauthorized("phone is not verified")
	}
	return nil
}

// EnforcePhoneVerification exposes the phone-verification gate to the magic-link
// coordinator (magic-link proves email but not phone).
func (s *loginService) EnforcePhoneVerification(_ context.Context, user *User, tenantID int64) error {
	return s.enforceLoginPhoneVerification(user, tenantID)
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
	resolvedClient, resolveErr := resolvePublicClient(ctx, s.clientRepo, clientID, tenantID)
	clientIDStr := ""
	if clientID != nil {
		clientIDStr = *clientID
	}

	var lockoutPolicy *security.RateLimitConfig
	var tenantIDForRL int64
	if resolveErr == nil && resolvedClient != nil {
		tenantIDForRL = clientTenantID(resolvedClient)
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
		return nil, loginRateLimitError(err, lockoutPolicy)
	}

	// Threat detection — pre-auth velocity / brute-force / risk assessment.
	clientIP := middleware.ClientIPFromContext(ctx)
	threatPolicy := secpolicy.LoadThreatPolicy(s.securitySettingRepo, tenantIDForRL)
	threatDecision := security.AssessLoginThreat(ctx, tenantIDForRL, clientIP, "", threatPolicy)
	if threatDecision.Blocked {
		return nil, apperror.NewUnauthorized("login blocked by threat detection")
	}
	forceStepUp := threatPolicy != nil && threatPolicy.RiskBasedStepUpEnabled && threatDecision.RequiresStepUp

	// Lockout check — before password verification.
	if err := s.checkLockout(ctx, tenantIDForRL, usernameOrEmail, lockoutPolicy); err != nil {
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
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)

		var txErr error
		client, txErr = resolvePublicClient(ctx, txClientRepo, clientID, tenantID)
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
			clientTenantID(client) == 0 {
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
		user, userLookupErr = findLoginUser(txUserRepo, usernameOrEmail, clientTenantID(client))

		// Fetch user identity to get the Sub claim
		if userLookupErr == nil && user != nil {
			userIdentity, txErr := txUserIdentityRepo.FindByUserIDAndClientReachable(user.UserID, client.ClientID)
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
		s.recordLockoutFailure(ctx, tenantIDForRL, usernameOrEmail, lockoutPolicy)
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
			TenantID:    clientTenantID(client),
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
	if err := s.enforceLoginEmailVerification(user, clientTenantID(client)); err != nil {
		return nil, err
	}
	if err := s.enforceLoginPhoneVerification(user, clientTenantID(client)); err != nil {
		return nil, err
	}
	identity, err := s.ensureUserIdentityForClient(ctx, user, client)
	if err != nil {
		return nil, err
	}
	userIdentitySub = identity.Sub

	// Reset failed attempts on successful authentication.
	security.ResetFailedAttemptsWithConfig(rateLimitIdentifier, lockoutPolicy)
	s.clearLockout(ctx, tenantIDForRL, usernameOrEmail, lockoutPolicy)

	// Check for compromised credentials at login (post-auth HIBP).
	s.checkCompromisedPassword(ctx, user, password, clientTenantID(client))

	// authevent.Log successful login
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "login_success",
		UserID:    user.UserUUID.String(),
		ClientID:  clientIDStr,
		Timestamp: startTime,
		Details:   fmt.Sprintf("Successful login for user %s", user.Username),
	})

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    clientTenantID(client),
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
	s.checkPasswordExpiry(ctx, user, clientTenantID(client))

	if err := s.checkTemporaryPasswordExpiry(ctx, user, clientTenantID(client)); err != nil {
		return nil, err
	}

	if user.ForcePasswordChange {
		return passwordChangeRequiredLoginResponse(), nil
	}

	if mfaResponse, mfaErr := s.loginMFAChallengeResponse(ctx, user, clientTenantID(client), forceStepUp); mfaErr != nil || mfaResponse != nil {
		return mfaResponse, mfaErr
	}

	// Record threat success after successful authentication.
	security.RecordLoginThreatSuccess(ctx, clientTenantID(client), user.UserID, middleware.ClientIPFromContext(ctx), middleware.UserAgentFromContext(ctx), threatPolicy)

	// Generate token response
	return s.generateTokenResponse(ctx, userIdentitySub, user, client)
}

// GetUserByEmail looks up a user by email within an explicit tenant. A global
// fallback is intentionally forbidden because email addresses are unique only
// inside a tenant.
func (s *loginService) GetUserByEmail(ctx context.Context, email string, tenantID int64) (*User, error) {
	_, span := otel.Tracer("service").Start(ctx, "login.getUserByEmail")
	defer span.End()
	if tenantID <= 0 {
		return nil, apperror.NewValidation("tenant_id is required")
	}
	user, err := s.userRepo.FindByEmailAndTenantID(email, tenantID)
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

	// Persist the JTI to the DB revocation store for RFC 7009 compliance.
	// Best-effort: a failure here does not block logout (Redis denylist already guards re-use).
	s.revokeLogoutJTI(ctx, claims)

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return nil
	}

	// `sub` is a user_identities.sub, NOT users.user_uuid.
	//
	// For a built-in (system-IdP) identity it is an independently minted UUID; for
	// a federated identity it is the upstream provider's subject, which is often
	// not a UUID at all. This previously did uuid.Parse(sub) → FindByUUID(sub),
	// so it either bailed on the parse (every Google/Cognito user) or looked up a
	// user_uuid that never matches (every password user) — and returned nil having
	// revoked no session. Logout appeared to work only because the access token's
	// jti was denylisted above; the session row stayed live.
	//
	// FindBySubAndClientID is the same resolver the auth middleware uses, and it
	// also enforces that the identity is actually reachable from this client.
	clientID, _ := claims["client_id"].(string)
	user, err := s.userRepo.FindBySubAndClientID(sub, clientID)
	if err != nil || user == nil {
		return nil
	}

	// Revoke ONLY the session this token belongs to — never more.
	//
	// A logout is a per-session act. Console and identity share one browser, one
	// cookie domain and therefore one user_sessions row, so revoking it signs the
	// user out of both — which is the intent. A second browser or a phone holds a
	// DIFFERENT session and must be left alone; being silently signed out
	// somewhere else because you logged out here is alarming, not secure.
	// "Sign out everywhere" is the separate, explicit control for that
	// (DELETE /account/sessions), as is a password change or reset.
	//
	// This used to fall back to RevokeAllSessions when the token had no `sid`,
	// which is exactly that cross-browser sign-out — and OAuth-minted tokens
	// never carried a `sid`, so every console logout took the fallback. OAuth
	// tokens are now session-stamped at /authorize (see oauth.callerSessionUUID),
	// so the claim is present. When it is genuinely absent we revoke nothing: the
	// access token is already denylisted above, and guessing at a session by
	// nuking all of them is worse than doing nothing.
	sessionID, _ := claims["sid"].(string)
	sessionUUID, parseErr := uuid.Parse(sessionID)
	if parseErr != nil {
		span.SetStatus(codes.Ok, "no session bound to token; jti denylisted only")
		return nil
	}
	if err := s.sessionService.RevokeSession(ctx, user.UserID, sessionUUID); err != nil {
		// A session that is already gone is a successful logout, not an error —
		// double-submits and a console+identity pair both calling logout for the
		// same shared session are both normal.
		var notFound *apperror.NotFoundError
		if errors.As(err, &notFound) {
			span.SetStatus(codes.Ok, "session already revoked")
			return nil
		}
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

func jwtClaimExpiry(expClaim any) time.Time {
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
			return time.Time{}
		}
		expUnix = parsed
	default:
		return time.Time{}
	}
	return time.Unix(expUnix, 0)
}

func (s *loginService) revokeLogoutJTI(ctx context.Context, claims jwtlib.MapClaims) {
	if s.tokenRevoker == nil {
		return
	}
	jti, _ := claims["jti"].(string)
	if strings.TrimSpace(jti) == "" {
		return
	}
	expiresAt := jwtClaimExpiry(claims["exp"])
	if expiresAt.IsZero() {
		return
	}
	// The DB revocation row is tenant-scoped and FK-constrained to tenants, so it
	// needs the token's REAL tenant — not 0, which violated the foreign key on
	// every logout (SQLSTATE 23503). The tenant_id claim carries the tenant's
	// opaque UUID (RFC 9068 least-disclosure); resolve it to the internal PK. If
	// it can't resolve, skip the DB record — the Redis denylist (denylistLogoutJTI)
	// already guards token reuse, so this stays best-effort rather than writing an
	// invalid row.
	tenantUUID, _ := claims["tenant_id"].(string)
	tenantID := shared.TenantIDByUUIDString(ctx, tenantUUID)
	if tenantID == 0 {
		return
	}
	_ = s.tokenRevoker.Revoke(ctx, tenantID, jti, "access_token", "logout", expiresAt, nil, nil)
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
	// TOTPDigits is the authenticator code length, 6 or 8. Read here so the login
	// challenge can tell the client how long a code to expect; see
	// LoginResponseDTO.MFATOTPDigits.
	TOTPDigits int `json:"totp_digits"`
}

func (s *loginService) loginMFAChallengeResponse(ctx context.Context, user *User, tenantID int64, forceStepUp bool) (*LoginResponseDTO, error) {
	if user == nil || s.mfaAuthenticator == nil {
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
	// Capture the TENANT's own hard requirement (mode=enforced, the enforce_mfa
	// flag, or the explicit required flag) BEFORE folding in forceStepUp — a user
	// with no usable factor is blocked only for these. Risk-based step-up
	// (forceStepUp) is a best-effort ELEVATION, not a hard requirement: it must
	// challenge a user who can step up, but must never lock out a user who has no
	// factor to step up with. Blocking the latter would let an attacker trip the
	// risk signal to deny a non-MFA victim, and would strand every non-MFA user on
	// any flagged login.
	hardRequireMFA := policy.Mode == "enforced" || policy.EnforceMFA || policy.Required

	// Check for a valid trusted-device token; skip MFA when recognized. This is a
	// security-relevant step-down, so it is audited (SOC2/ISO/NIST expect MFA
	// bypasses to be logged).
	//
	// It runs AFTER the policy load and is subordinate to it. Previously it
	// returned before the policy was read and before forceStepUp was considered,
	// so a remembered browser walked straight past a tenant that hard-requires
	// MFA, and a risk-based step-up — raised precisely because this login looked
	// wrong — was defeated by a cookie the attacker had. "I trusted this browser
	// once" is a convenience signal; it cannot outrank the tenant's standing
	// requirement or a live risk signal.
	trustedDeviceToken := trustedDeviceTokenFromContext(ctx)
	if !hardRequireMFA && !forceStepUp &&
		trustedDeviceToken != "" && s.isTrustedDeviceValid(ctx, user.UserID, tenantID, trustedDeviceToken) {
		if s.authEventService != nil {
			s.authEventService.Log(ctx, authevent.AuthEventInput{
				TenantID:    tenantID,
				ActorUserID: &user.UserID,
				IPAddress:   middleware.ClientIPFromContext(ctx),
				UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
				Category:    authevent.AuthEventCategoryAuthn,
				EventType:   authevent.AuthEventTypeMFATrustedDeviceSkip,
				Severity:    authevent.AuthEventSeverityInfo,
				Result:      authevent.AuthEventResultSuccess,
				Description: ptr.Ptr("MFA step skipped: request presented a valid trusted-device token"),
			})
		}
		return nil, nil
	}

	if hardRequireMFA || forceStepUp {
		policy.Required = true
	}
	policy.AllowedMethods = normalizeLoginMFAPolicyMethods(policy.AllowedMethods)

	// Ask the mfa service for the user's actual enrolled factors (the single
	// source of truth — covers TOTP, WebAuthn, the verified SMS phone, and backup
	// codes). Reading from the user record alone would miss SMS/WebAuthn.
	//
	// A lookup failure FAILS CLOSED. Returning (nil, nil) here — the "proceed at
	// acr=1" contract used everywhere else in this function — meant a single
	// transient repository error let a user of a mode=enforced tenant in on a
	// password alone, and silently downgraded every MFA-enrolled user on an
	// optional-mode tenant. "The user has no second factor" and "we could not
	// read the user's second factors" are not the same answer, so the login is
	// refused, exactly as the passwordless sibling below already does.
	enrolled, err := s.mfaAuthenticator.EnrolledMFAMethods(ctx, user.UserID)
	if err != nil {
		return nil, apperror.NewInternal("failed to load enrolled MFA factors", err)
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
		// Tenant HARD-requires MFA but the user has nothing usable enrolled →
		// block. A risk-based step-up (forceStepUp) with no usable factor instead
		// falls through to a normal (acr=1) login rather than locking the user out:
		// the login was flagged for elevation, not denial (the tenant's block
		// threshold sits above the step-up band and is enforced upstream in
		// AssessLoginThreat), and the risk event is already logged/emitted by the
		// threat layer.
		if hardRequireMFA {
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
		MFATOTPDigits:     totpDigitsFromPolicy(policy),
	}, nil
}

// MagicLinkMFAChallenge applies the tenant's login MFA policy after a magic
// link has established the first factor. Unlike password login, email_otp is
// removed because it would reuse the same mailbox instead of proving a distinct
// factor. Enforced mode never bypasses MFA through a grace or trusted-device
// shortcut; users without a usable non-email factor are blocked.
func (s *loginService) MagicLinkMFAChallenge(ctx context.Context, user *User, tenantID int64) (*LoginResponseDTO, error) {
	// email_otp is excluded: it reuses the same mailbox the magic link arrived
	// in, so it proves no distinct second factor.
	return s.passwordlessMFAChallenge(ctx, user, tenantID, "email_otp", platformjwt.AMRMagicLink)
}

// SMSMFAChallenge applies the tenant's login MFA policy after an SMS one-time
// code has established the first factor. It mirrors MagicLinkMFAChallenge: an
// SMS passwordless login is a single possession factor, so when the tenant
// enforces MFA it must still be challenged for a SECOND factor rather than
// walking straight in at acr=1 — the gap that let mode=enforced be bypassed on
// this entry point entirely.
func (s *loginService) SMSMFAChallenge(ctx context.Context, user *User, tenantID int64) (*LoginResponseDTO, error) {
	// sms is excluded: it reuses the same phone the OTP arrived on, so offering
	// it as the "second" factor proves nothing new (same reasoning as email_otp
	// on the magic-link path).
	return s.passwordlessMFAChallenge(ctx, user, tenantID, "sms", platformjwt.AMRSMS)
}

// passwordlessMFAChallenge is the shared MFA-policy gate for passwordless
// first-factor logins (magic link, SMS). It returns a challenge when a second
// factor is required, (nil, nil) when the login may proceed at acr=1, or an
// error when the tenant enforces MFA but the user has no usable second factor.
//
// excludedMethod is the channel already used as the first factor, dropped from
// the allowed methods so it cannot masquerade as a distinct second factor.
func (s *loginService) passwordlessMFAChallenge(ctx context.Context, user *User, tenantID int64, excludedMethod, primaryAMR string) (*LoginResponseDTO, error) {
	if user == nil {
		return nil, apperror.NewUnauthorized("authentication failed")
	}

	policy := secpolicy.LoadMFAPolicy(s.securitySettingRepo, tenantID)
	if policy == nil || policy.Mode == "disabled" {
		return nil, nil
	}
	if s.mfaAuthenticator == nil {
		if policy.Mode == "enforced" {
			return nil, apperror.NewUnauthorized("MFA is required but no supported factors are enrolled")
		}
		return nil, nil
	}

	enrolled, err := s.mfaAuthenticator.EnrolledMFAMethods(ctx, user.UserID)
	if err != nil {
		return nil, apperror.NewInternal("failed to load enrolled MFA factors", err)
	}

	registeredPrimaryFactor := hasPrimaryMFAFactor(enrolled)
	if policy.Mode != "enforced" && !registeredPrimaryFactor {
		return nil, nil
	}

	allowedMethods := filterMFAMethodsByPolicy(enrolled, normalizeLoginMFAPolicyMethods(policy.AllowedMethods))
	allowedMethods = removeMFAMethod(allowedMethods, excludedMethod)
	allowedMethods = preferLoginMFAMethodFirst(allowedMethods, policy.PreferredMethod)
	if len(allowedMethods) == 0 {
		return nil, apperror.NewUnauthorized("MFA is required but no supported second factor is enrolled")
	}

	challengeToken, err := platformjwt.GenerateStepUpChallengeTokenForAuthMethodWithContext(
		ctx,
		user.UserUUID.String(),
		time.Duration(mfaStepUpTTLSeconds(policy))*time.Second,
		primaryAMR,
		allowedMethods,
	)
	if err != nil {
		return nil, err
	}

	return &LoginResponseDTO{
		MFARequired:       true,
		MFAChallengeToken: &challengeToken,
		MFAAllowedMethods: allowedMethods,
		MFATOTPDigits:     totpDigitsFromSecPolicy(policy),
	}, nil
}

func removeMFAMethod(methods []string, excluded string) []string {
	filtered := make([]string, 0, len(methods))
	for _, method := range methods {
		if method != excluded {
			filtered = append(filtered, method)
		}
	}
	return filtered
}

// IssueMagicLinkSession issues the normal policy-aware ACR-1 session for a
// passwordless login when no second factor is required.
func (s *loginService) IssueMagicLinkSession(ctx context.Context, sub string, user *User, client *Client) (*LoginResponseDTO, error) {
	return s.generateTokenResponseWithAuth(ctx, sub, user, client, []string{platformjwt.AMRMagicLink}, platformjwt.ACRLevel1)
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
		threatPolicy := secpolicy.LoadThreatPolicy(s.securitySettingRepo, clientTenantID(client))
		security.RecordLoginThreatFailure(ctx, clientTenantID(client), middleware.ClientIPFromContext(ctx), rateLimitIdentifier, threatPolicy)
		s.authEventService.Log(ctx, authevent.AuthEventInput{
			TenantID:    clientTenantID(client),
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

func (s *loginService) ensureUserIdentityForClient(ctx context.Context, user *User, client *Client) (*UserIdentity, error) {
	if user == nil || client == nil || client.ClientID == 0 || clientTenantID(client) == 0 {
		return nil, apperror.NewUnauthorized("authentication failed")
	}
	identity, err := s.userIdentityRepo.FindByUserIDAndClientReachable(user.UserID, client.ClientID)
	if err != nil {
		return nil, apperror.NewInternal("failed to load user identity", err)
	}
	if identity != nil {
		if identity.Sub == "" {
			return nil, apperror.NewUnauthorized("authentication failed")
		}
		return identity, nil
	}

	idpID := connectedSystemIdentityProviderID(client)
	if idpID == nil {
		return nil, apperror.NewUnauthorized("authentication failed")
	}
	created, err := s.userIdentityRepo.Create(&UserIdentity{
		TenantID:           clientTenantID(client),
		UserID:             user.UserID,
		IdentityProviderID: *idpID,
		Provider:           shared.ProviderMaintainerd,
		Sub:                uuid.NewString(),
		Metadata:           datatypes.JSON([]byte(`{}`)),
	})
	if err != nil {
		return nil, apperror.NewInternal("failed to create user identity", err)
	}
	return created, nil
}

// connectedSystemIdentityProviderID returns the id of the client's connected
// built-in system IdP, used to anchor a password-authenticated user's local
// identity. Selection keys off the system flag (is_system / provider_type ==
// system), NEVER the "maintainerd" provider string, so a connected EXTERNAL
// maintainerd (enterprise) IdP is never mistaken for the built-in one.
func connectedSystemIdentityProviderID(client *Client) *int64 {
	if client == nil {
		return nil
	}
	if client.ConnectedProviders != nil {
		for i := range *client.ConnectedProviders {
			conn := (*client.ConnectedProviders)[i]
			if !conn.Enabled || conn.IdentityProvider == nil {
				continue
			}
			if isSystemIdentityProvider(conn.IdentityProvider) {
				id := conn.IdentityProviderID
				if id == 0 {
					id = conn.IdentityProvider.IdentityProviderID
				}
				if id > 0 {
					return &id
				}
			}
		}
	}
	if client.IdentityProvider != nil && isSystemIdentityProvider(client.IdentityProvider) {
		id := client.IdentityProvider.IdentityProviderID
		if id == 0 {
			id = client.IdentityProviderID
		}
		if id > 0 {
			return &id
		}
	}
	return nil
}

// isSystemIdentityProvider reports whether an IdP is the built-in system
// provider by its authoritative flags, independent of the provider slug.
func isSystemIdentityProvider(idp *IdentityProvider) bool {
	return idp != nil && (idp.IsSystem || idp.ProviderType == shared.IDPTypeSystem)
}

// generateTokenResponseWithAuth issues a full session token set with the given
// amr/acr. Password login uses acr=1; the login MFA second step uses acr=2 so
// the whole session satisfies step-up routes without per-action re-prompts.
func (s *loginService) generateTokenResponseWithAuth(ctx context.Context, sub string, user *User, Client *Client, amr []string, acr string) (*LoginResponseDTO, error) {
	// Update last_login_at and atomically increment login_count.
	s.updateLoginTimestamps(ctx, user)

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
		// Record what actually authenticated this session. acr/amr are the
		// session's own facts — hardcoding "1" here made an MFA-completed login
		// indistinguishable from a password-only one.
		attrs := SessionAttributes{
			AMR:                amr,
			ACR:                acr,
			IdentityProviderID: connectedSystemIdentityProviderID(Client),
		}
		if Client != nil && Client.ClientID > 0 {
			cid := Client.ClientID
			attrs.ClientID = &cid
		}
		sess, err := createSessionWithPolicy(ctx, s.sessionService, user.UserID, clientTenantID(Client), ipAddress, userAgent, policy, attrs)
		if err != nil {
			return nil, err
		}
		sessionID = sess.UserSessionUUID.String()
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
	client, err = resolveClientForContext(ctx, s.clientRepo, clientID, tenantID)
	if err != nil || client == nil || client.Status != shared.StatusActive ||
		client.Domain == nil || *client.Domain == "" {
		return nil, apperror.NewUnauthorized("authentication failed")
	}

	amr, err := s.mfaAuthenticator.VerifyFactor(ctx, user.UserID, method, code, assertion)
	if err != nil {
		s.authEventService.Log(ctx, authevent.AuthEventInput{
			TenantID:    clientTenantID(client),
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
	if primaryAMR, _ := claims["primary_amr"].(string); primaryAMR != "" {
		amr = replacePrimaryAMR(amr, primaryAMR)
	}
	identity, err := s.ensureUserIdentityForClient(ctx, user, client)
	if err != nil {
		return nil, err
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    clientTenantID(client),
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
		if deviceToken, maxAge, tokenErr := s.issueTrustedDeviceToken(ctx, user.UserID, clientTenantID(client)); tokenErr == nil && deviceToken != "" {
			resp.TrustedDeviceToken = deviceToken
			resp.TrustedDeviceMaxAge = maxAge
			if s.authEventService != nil {
				s.authEventService.Log(ctx, authevent.AuthEventInput{
					TenantID:    clientTenantID(client),
					ActorUserID: &user.UserID,
					IPAddress:   middleware.ClientIPFromContext(ctx),
					UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
					Category:    authevent.AuthEventCategoryAuthn,
					EventType:   authevent.AuthEventTypeMFATrustedDeviceTrust,
					Severity:    authevent.AuthEventSeverityInfo,
					Result:      authevent.AuthEventResultSuccess,
					Description: ptr.Ptr("Device marked trusted; future logins from it may skip MFA until the trust window expires"),
				})
			}
		}
	}

	return resp, nil
}

func replacePrimaryAMR(amr []string, primaryAMR string) []string {
	replaced := false
	result := make([]string, 0, len(amr)+1)
	for _, method := range amr {
		if method == platformjwt.AMRPassword {
			if !replaced {
				result = append(result, primaryAMR)
				replaced = true
			}
			continue
		}
		result = append(result, method)
	}
	if !replaced {
		result = append([]string{primaryAMR}, result...)
	}
	return result
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

func (s *loginService) SendMFALoginEmailOTP(ctx context.Context, challengeToken string) error {
	if s.mfaAuthenticator == nil {
		return apperror.NewInternal("MFA is not configured", nil)
	}
	user, claims, err := s.resolveMFAChallengeUser(challengeToken)
	if err != nil {
		return err
	}
	if !loginMFAMethodAllowed(claims["allowed_methods"], "email_otp") {
		return apperror.NewValidation("MFA method not allowed: email_otp")
	}
	return s.mfaAuthenticator.SendEmailOTPChallenge(ctx, user.UserID)
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

// issueTrustedDeviceToken records the current device as trusted and returns the
// one-time plaintext secret plus the cookie/token lifetime in seconds. Returns
// an empty token (and 0 maxAge) when the tenant policy disables trusted devices.
func (s *loginService) issueTrustedDeviceToken(ctx context.Context, userID, tenantID int64) (string, int, error) {
	mfaPolicy := secpolicy.LoadMFAPolicy(s.securitySettingRepo, tenantID)
	if mfaPolicy == nil || mfaPolicy.Mode == "disabled" || mfaPolicy.TrustedDevicePeriodDays <= 0 {
		return "", 0, nil
	}
	trusted, ok := s.mfaAuthenticator.(MFATrustedDeviceAuthenticator)
	if !ok {
		return "", 0, nil
	}
	token, err := trusted.IssueTrustedDevice(ctx, userID, tenantID, deviceIDFromContext(ctx), mfaPolicy.TrustedDevicePeriodDays)
	if err != nil {
		return "", 0, err
	}
	return token, mfaPolicy.TrustedDevicePeriodDays * 24 * 60 * 60, nil
}

// ForgetTrustedDevice removes the trusted-device row for the presented token, so
// a browser can drop its own trust (e.g. on a shared computer). No-op on unknown
// or empty tokens.
func (s *loginService) ForgetTrustedDevice(ctx context.Context, token string) {
	if strings.TrimSpace(token) == "" {
		return
	}
	if trusted, ok := s.mfaAuthenticator.(MFATrustedDeviceAuthenticator); ok {
		_ = trusted.RevokeTrustedDeviceByToken(ctx, token)
	}
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
	valid, err := trusted.TrustedDeviceValid(ctx, userID, tenantID, rawToken)
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

// ──────────────────────────────────────────────────────────────────────────────
// Lockout helpers
// ──────────────────────────────────────────────────────────────────────────────

func (s *loginService) checkLockout(ctx context.Context, tenantID int64, identifier string, lockoutPolicy *security.RateLimitConfig) error {
	if s.lockoutRepo == nil || lockoutPolicy == nil {
		return nil
	}
	if !lockoutPolicy.Enabled {
		return nil
	}
	locked, err := s.lockoutRepo.IsLocked(ctx, tenantID, identifier)
	if err != nil {
		return nil
	}
	if locked {
		return apperror.NewUnauthorized("account is temporarily locked due to too many failed login attempts")
	}
	return nil
}

// recordLockoutFailure records one failed login and applies the tenant's full
// lockout policy through the DB path. When the failure tips the account into a
// fresh lock and the policy asks for it, the user notification fires exactly
// once via the same OnAccountLockout hook the settings wired for lockout.
func (s *loginService) recordLockoutFailure(ctx context.Context, tenantID int64, identifier string, policy *security.RateLimitConfig) {
	if s.lockoutRepo == nil || policy == nil || !policy.Enabled {
		return
	}
	result, err := s.lockoutRepo.RecordFailure(ctx, tenantID, identifier, middleware.ClientIPFromContext(ctx), lockoutRulesFromPolicy(policy))
	if err != nil {
		return
	}
	if result.JustLocked && policy.NotifyUserOnLockout && security.OnAccountLockout != nil {
		security.OnAccountLockout(ctx, identifier)
	}
}

func lockoutRulesFromPolicy(policy *security.RateLimitConfig) UserLockoutRules {
	return UserLockoutRules{
		MaxAttempts:       policy.MaxFailedAttempts,
		BaseDuration:      policy.LockoutDuration,
		ObservationWindow: policy.ObservationWindow,
		Progressive:       policy.ProgressiveLockout,
		MaxDuration:       policy.MaxLockoutDuration,
		ProgressionReset:  policy.ProgressionReset,
		AutoUnlock:        policy.AutoUnlock,
	}
}

// UserLockoutRules aliases the repository's LockoutRules so callers in this file
// read naturally. It is the same type.
type UserLockoutRules = LockoutRules

// clearLockout resets an identifier's lockout state on successful login, unless
// the tenant has opted out via reset_count_on_success=false — in which case the
// accumulated failure count is left to persist across successful logins.
func (s *loginService) clearLockout(ctx context.Context, tenantID int64, identifier string, policy *security.RateLimitConfig) {
	if s.lockoutRepo == nil {
		return
	}
	if policy != nil && !policy.ResetCountOnSuccess {
		return
	}
	_ = s.lockoutRepo.ClearLockout(ctx, tenantID, identifier)
}

func (s *loginService) updateLoginTimestamps(ctx context.Context, user *User) {
	if user == nil || user.UserID == 0 || s.userRepo == nil {
		return
	}
	now := time.Now()
	if _, err := s.userRepo.UpdateByID(user.UserID, map[string]any{
		"last_login_at": now,
		"login_count":   gorm.Expr("login_count + 1"),
	}); err != nil {
		slog.Warn("failed to update login timestamps", "user_id", user.UserID, "err", err)
	}
}

// loginRateLimitError maps a lockout or limiter failure onto the typed error the
// HTTP layer knows how to render.
//
// The bare error this used to return carried no type, so a locked-out login and
// a limiter outage both fell through to 500. That is wrong twice over: a client
// cannot tell "back off" from "we broke" and so cannot back off correctly, and
// every routine lockout showed up in monitoring as a server fault, which is how
// a real brute-force attack ends up buried in noise.
//
// The two cases are kept apart because they ask the caller for different things:
//
//   - The limiter is DOWN → 503. Nothing is wrong with this caller; the service
//     cannot currently meter anyone, so it declines rather than let the attempt
//     through unmetered (see security.limiterOutage, which fails closed).
//   - The account is LOCKED → 429 with Retry-After, so the client waits the
//     lockout out instead of hammering and extending it.
func loginRateLimitError(err error, policy *security.RateLimitConfig) error {
	if errors.Is(err, security.ErrRateLimiterUnavailable) {
		return apperror.NewServiceUnavailable(err.Error())
	}
	// Retry-After comes from the tenant's configured lockout duration, falling
	// back to the package default when the tenant has not set one — the same
	// value the limiter itself locked the account for.
	retryAfter := security.AccountLockoutTime
	if policy != nil && policy.LockoutDuration > 0 {
		retryAfter = policy.LockoutDuration
	}
	return apperror.NewTooManyRequestsAfter(err.Error(), retryAfter)
}

// totpDigitsFromPolicy reports the tenant's authenticator code length for the
// login second step, defaulting to 6 when unset.
//
// The step cannot look this up itself: the user has not authenticated yet, so
// the MFA status endpoint is closed to them. Sending it with the challenge is
// what lets the input match the code the authenticator shows.
func totpDigitsFromPolicy(policy loginMFAPolicy) int {
	return totpDigitsOrDefault(policy.TOTPDigits)
}

// totpDigitsFromSecPolicy is the same question asked of the secpolicy shape, the
// other type that reaches an MFA challenge on this path.
func totpDigitsFromSecPolicy(policy *secpolicy.MFAPolicy) int {
	if policy == nil {
		return loginTOTPDigitsDefault
	}
	return totpDigitsOrDefault(policy.TOTPDigits)
}

func totpDigitsOrDefault(digits int) int {
	if digits > 0 {
		return digits
	}
	return loginTOTPDigitsDefault
}

// loginTOTPDigitsDefault matches the seeded totp_digits policy.
const loginTOTPDigitsDefault = 6

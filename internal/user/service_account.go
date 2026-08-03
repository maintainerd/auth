package user

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/notifier"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/platform/email"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/platform/sms"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const emailChangeOTPLength = 6
const emailChangeOTPTTL = 1 * time.Hour

// Email-change OTPs are routed through user_otps (not stored on the users row)
// so they get the same single-use flag, failure accounting, and at-rest hashing
// as every other OTP. The pending new address is carried in the OTP row's
// metadata JSONB under "pending_email".
const emailChangeChannel = "email_change"
const emailChangeMaxFailed = 5
const emailChangePendingEmailKey = "pending_email"

const backupCodeCount = 10
const backupCodeLength = 8

// Phone-verification SMS OTP settings. A distinct channel keeps these OTPs from
// colliding with MFA's "sms" OTPs stored in the same user_otps table.
const phoneVerifyOTPLength = 6
const phoneVerifyOTPTTL = 10 * time.Minute
const phoneVerifyMaxFailed = 3
const phoneVerifyChannel = "phone_verify"

var (
	accountGenerateAccessTokenWithContext  = jwt.GenerateAccessTokenWithContext
	accountGenerateIDTokenWithContext      = jwt.GenerateIDTokenWithContext
	accountGenerateRefreshTokenWithContext = jwt.GenerateRefreshTokenWithContext
	// accountNewSMSProvider is the SMS provider factory. It is a package-level
	// indirection over sms.NewProviderFromDB so tests can swap it out.
	accountNewSMSProvider = sms.NewProviderFromDB
)

// AccountService handles self-service account management operations.
type AccountService interface {
	InitiateEmailChange(ctx context.Context, userID int64, newEmail, currentPassword string) error
	VerifyEmailChange(ctx context.Context, userID int64, otp string) error
	ChangeUsername(ctx context.Context, userID int64, newUsername, currentPassword string) error
	ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword string, callerSessionUUID *uuid.UUID) (*ChangePasswordResponseDTO, error)
	DeleteAccount(ctx context.Context, userID int64, currentPassword string) error
	ExportAccountData(ctx context.Context, userID int64) (*AccountExportDTO, error)
	GenerateBackupCodes(ctx context.Context, userID int64) (*GenerateBackupCodesResponseDTO, error)
	VerifyBackupCode(ctx context.Context, req VerifyBackupCodeDTO) (*LoginResponseDTO, error)
	SendPhoneVerification(ctx context.Context, userID int64, phone string) error
	VerifyPhone(ctx context.Context, userID int64, phone, code string) error
}

type accountService struct {
	db                   *gorm.DB
	userRepo             UserRepository
	userTokenRepo        UserTokenRepository
	profileRepo          ProfileRepository
	userSettingRepo      UserSettingRepository
	roleRepo             RoleRepository
	clientRepo           ClientRepository
	mfaBackupCodeRepo    UserMFABackupCodeRepository
	userIdentityRepo     UserIdentityRepository
	identityProviderRepo IdentityProviderRepository
	authEventService     authevent.AuthEventService
	securitySettingRepo  secpolicy.SecuritySettingRepository
	smsOtpRepo           notifier.UserOTPRepository
	// passwordHistoryRepo enforces PasswordPolicy.HistoryCount on self-service
	// password change. Nil is only tolerated when the tenant's HistoryCount is
	// 0 — see ChangePassword, which fails closed rather than skipping the check.
	passwordHistoryRepo UserPasswordHistoryRepository
	// sessionRepo is the canonical session store (user_sessions). Revoking
	// anywhere else does not end a login.
	sessionRepo SessionRevoker
	// refreshRevoker kills OAuth refresh tokens. Revoking sessions alone leaves
	// a long-lived credential minted under the OLD password fully spendable.
	// Nil disables it.
	refreshRevoker RefreshTokenRevoker
}

// RefreshTokenRevoker is the slice of the OAuth refresh-token store this
// service needs. Declared here rather than importing internal/oauth, which
// already imports this package — internal/app supplies the adapter.
type RefreshTokenRevoker interface {
	// WithTx joins the caller's transaction so the password update and the
	// revocation commit or roll back together.
	WithTx(tx *gorm.DB) RefreshTokenRevoker
	RevokeByUserID(userID int64) (int64, error)
}

// SessionRevoker is the slice of the canonical session store this service needs.
// Declared here rather than importing authn's repository interface so the user
// package keeps its existing dependency direction.
type SessionRevoker interface {
	// WithTx lets a revoke join the caller's transaction, so a password change
	// and its session revocation commit or roll back together.
	WithTx(tx *gorm.DB) SessionRevoker
	RevokeAllByUserID(userID int64, reason string) error
	RevokeAllExceptUUID(userID int64, keepSessionUUID uuid.UUID, reason string) error
}

func NewAccountService(
	db *gorm.DB,
	userRepo UserRepository,
	userTokenRepo UserTokenRepository,
	profileRepo ProfileRepository,
	userSettingRepo UserSettingRepository,
	roleRepo RoleRepository,
	clientRepo ClientRepository,
	mfaBackupCodeRepo UserMFABackupCodeRepository,
	userIdentityRepo UserIdentityRepository,
	identityProviderRepo IdentityProviderRepository,
	authEventService authevent.AuthEventService,
	securitySettingRepo secpolicy.SecuritySettingRepository,
	smsOtpRepo notifier.UserOTPRepository,
	passwordHistoryRepo UserPasswordHistoryRepository,
	sessionRepo SessionRevoker,
	refreshRevoker ...RefreshTokenRevoker,
) AccountService {
	svc := &accountService{
		passwordHistoryRepo:  passwordHistoryRepo,
		sessionRepo:          sessionRepo,
		db:                   db,
		userRepo:             userRepo,
		userTokenRepo:        userTokenRepo,
		profileRepo:          profileRepo,
		userSettingRepo:      userSettingRepo,
		roleRepo:             roleRepo,
		clientRepo:           clientRepo,
		mfaBackupCodeRepo:    mfaBackupCodeRepo,
		userIdentityRepo:     userIdentityRepo,
		identityProviderRepo: identityProviderRepo,
		authEventService:     authEventService,
		securitySettingRepo:  securitySettingRepo,
		smsOtpRepo:           smsOtpRepo,
	}
	if len(refreshRevoker) > 0 {
		svc.refreshRevoker = refreshRevoker[0]
	}
	return svc
}

// InitiateEmailChange verifies the current password and sends an OTP to the new email address.
func (s *accountService) InitiateEmailChange(ctx context.Context, userID int64, newEmail, currentPassword string) error {
	_, span := otel.Tracer("service").Start(ctx, "account.initiateEmailChange")
	defer span.End()

	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return apperror.NewNotFound("user not found")
	}

	// Verify current password
	if user.Password == nil {
		return apperror.NewValidation("account has no password set")
	}
	if !security.ComparePassword([]byte(*user.Password), []byte(currentPassword)) {
		return apperror.NewUnauthorized("invalid current password")
	}

	// Check new email is not already taken within the user's tenant
	existing, err := s.userRepo.FindByEmailAndTenantID(newEmail, user.TenantID)
	if err != nil {
		return apperror.NewInternal("failed to check email availability", err)
	}
	if existing != nil {
		return apperror.NewValidation("email address is already in use")
	}

	// Generate OTP and hash it for at-rest storage.
	otp, err := crypto.GenerateOTP(emailChangeOTPLength)
	if err != nil {
		return apperror.NewInternal("failed to generate OTP", err)
	}
	otpHash := crypto.HashAuthorizationCode(otp)

	// Clear any prior pending email-change OTPs for this user, then store a fresh
	// hashed OTP with the pending address in metadata. Routed through user_otps
	// (channel='email_change') rather than columns on the users row.
	s.db.Where("user_id = ? AND channel = ?", userID, emailChangeChannel).Delete(&notifier.UserOTP{})

	metadata, err := json.Marshal(map[string]string{emailChangePendingEmailKey: newEmail})
	if err != nil {
		return apperror.NewInternal("failed to encode email change metadata", err)
	}
	record := &notifier.UserOTP{
		UserID:    user.UserID,
		Channel:   emailChangeChannel,
		Recipient: newEmail,
		OTPHash:   otpHash,
		Metadata:  datatypes.JSON(metadata),
		ExpiresAt: time.Now().Add(emailChangeOTPTTL),
	}
	if _, err := s.smsOtpRepo.Create(record); err != nil {
		return apperror.NewInternal("failed to store pending email", err)
	}

	sendEmail := email.SendEmail

	// #nosec G118 -- best-effort background goroutine must outlive request context
	go func() {
		sendCtx := context.Background()
		if sendErr := s.sendEmailChangeOTP(sendCtx, sendEmail, user.TenantID, newEmail, otp); sendErr != nil {
			slog.Error("account: failed to send email change OTP", "error", sendErr, "user_id", userID)
		}
	}()

	span.SetStatus(codes.Ok, "")
	return nil
}

func (s *accountService) sendEmailChangeOTP(ctx context.Context, sendEmail func(context.Context, *gorm.DB, email.SendEmailParams) error, tenantID int64, toEmail, otp string) error {
	data := struct {
		OTP     string
		LogoURL string
	}{
		OTP:     otp,
		LogoURL: email.GetLogoURL(ctx, s.db),
	}

	rendered, err := email.RenderTemplate(s.db, "user:email:change", tenantID, data)
	if err != nil {
		return fmt.Errorf("failed to render email change template: %w", err)
	}
	return sendEmail(ctx, s.db, email.SendEmailParams{
		To:        toEmail,
		Subject:   rendered.Subject,
		BodyHTML:  rendered.BodyHTML,
		BodyPlain: rendered.BodyPlain,
	})
}

// VerifyEmailChange confirms the OTP and applies the pending email address.
func (s *accountService) VerifyEmailChange(ctx context.Context, userID int64, otp string) error {
	_, span := otel.Tracer("service").Start(ctx, "account.verifyEmailChange")
	defer span.End()

	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return apperror.NewNotFound("user not found")
	}

	// Find the most recent unused, unexpired email-change OTP for this user.
	// (Expired/used rows are excluded by the query, so they read as "not found".)
	otpRecord, lerr := s.smsOtpRepo.FindValidByUserAndChannel(user.UserID, emailChangeChannel)
	if lerr != nil {
		return apperror.NewInternal("failed to look up email change", lerr)
	}
	if otpRecord == nil {
		return apperror.NewValidation("no pending email change found")
	}

	expectedHash := crypto.HashAuthorizationCode(otp)
	if subtle.ConstantTimeCompare([]byte(otpRecord.OTPHash), []byte(expectedHash)) != 1 {
		_ = s.smsOtpRepo.RecordFailure(otpRecord.UserOTPID, emailChangeMaxFailed)
		return apperror.NewUnauthorized("invalid OTP")
	}

	var meta map[string]string
	if err := json.Unmarshal(otpRecord.Metadata, &meta); err != nil {
		return apperror.NewInternal("failed to decode email change metadata", err)
	}
	newEmail := meta[emailChangePendingEmailKey]
	if newEmail == "" {
		return apperror.NewValidation("pending email address is missing")
	}

	// Single-use: mark the OTP consumed before applying the change.
	if err := s.smsOtpRepo.MarkUsed(otpRecord.UserOTPID); err != nil {
		return apperror.NewInternal("failed to mark OTP used", err)
	}

	if err := s.userRepo.UpdateEmail(user.UserUUID, newEmail); err != nil {
		return apperror.NewInternal("failed to update email", err)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// ChangeUsername verifies the current password and updates the username.
func (s *accountService) ChangeUsername(ctx context.Context, userID int64, newUsername, currentPassword string) error {
	_, span := otel.Tracer("service").Start(ctx, "account.changeUsername")
	defer span.End()

	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return apperror.NewNotFound("user not found")
	}

	if user.Password == nil {
		return apperror.NewValidation("account has no password set")
	}
	if !security.ComparePassword([]byte(*user.Password), []byte(currentPassword)) {
		return apperror.NewUnauthorized("invalid current password")
	}

	// Check username not taken within the user's tenant
	existing, err := s.userRepo.FindByUsernameAndTenantID(newUsername, user.TenantID)
	if err != nil {
		return apperror.NewInternal("failed to check username availability", err)
	}
	if existing != nil && existing.UserID != userID {
		return apperror.NewValidation("username is already taken")
	}

	if err := s.userRepo.UpdateUsername(user.UserUUID, newUsername); err != nil {
		return apperror.NewInternal("failed to update username", err)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// ChangePassword rotates the authenticated user's own password.
//
// Until this existed the ONLY way to change a password was the unauthenticated
// emailed reset link, which meant checkPasswordHistory ran on exactly one code
// path and a signed-in user had no way to rotate a credential they believed was
// exposed without going through their inbox. The seeded permission
// "account:change-password:self" had been sitting there with nothing behind it.
//
// It deliberately mirrors the reset path's rules (tenant policy, history, hash,
// force_password_change clearing, temp-password clearing, session revocation)
// so the two cannot drift into enforcing different things. The one intentional
// difference is session handling — see below.
func (s *accountService) ChangePassword(
	ctx context.Context,
	userID int64,
	currentPassword, newPassword string,
	callerSessionUUID *uuid.UUID,
) (*ChangePasswordResponseDTO, error) {
	ctx, span := otel.Tracer("service").Start(ctx, "account.changePassword")
	defer span.End()

	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return nil, apperror.NewNotFound("user not found")
	}

	// A federated or passwordless account has nothing to rotate. 400, not 500.
	if user.Password == nil {
		return nil, apperror.NewValidation("account has no password set")
	}
	if !security.ComparePassword([]byte(*user.Password), []byte(currentPassword)) {
		s.logPasswordChangeEvent(ctx, user, false, "current password did not match")
		return nil, apperror.NewUnauthorized("invalid current password")
	}

	// Reject a no-op rotation explicitly. A tenant with HistoryCount = 0 would
	// otherwise happily "change" the password to itself and report success.
	if currentPassword == newPassword {
		return nil, apperror.NewValidation("new password must be different from the current password")
	}

	tenantID := user.TenantID
	policy := secpolicy.LoadPasswordPolicy(s.securitySettingRepo, tenantID)

	// Identity-aware validation: this is the one password flow that always knows
	// exactly whose password it is, so it can reject a password that merely
	// restates the account's own username or email.
	if err := security.ValidatePasswordPolicyForUser(ctx, newPassword, policy, security.PasswordUserContext{
		Username: user.Username,
		Email:    user.Email,
	}); err != nil {
		s.logPasswordChangeEvent(ctx, user, false, "new password rejected by policy")
		return nil, apperror.NewValidation(err.Error())
	}

	// Fail CLOSED. A nil history repo used to mean "skip the check", so a wiring
	// mistake would silently disable reuse protection for every tenant that
	// configured it.
	if policy.HistoryCount > 0 && s.passwordHistoryRepo == nil {
		return nil, apperror.NewInternal("password history is required by policy but is not configured", nil)
	}
	if s.passwordHistoryRepo != nil {
		if err := s.checkPasswordReuse(userID, policy.HistoryCount, newPassword); err != nil {
			s.logPasswordChangeEvent(ctx, user, false, "new password was recently used")
			return nil, err
		}
	}

	// Hashing is deliberately OUTSIDE the transaction below: argon2id is tuned to
	// take real time, and holding a row lock across it serializes every
	// concurrent password change on the same table.
	hashed, err := security.HashPasswordWithPolicy(ctx, []byte(newPassword), policy)
	if err != nil {
		return nil, apperror.NewInternal("failed to hash password", err)
	}

	// Session handling — the one place this intentionally differs from reset.
	//
	// ASVS V3 and NIST 800-63B require OTHER sessions to be invalidated on a
	// password change; neither requires logging out the session doing the
	// changing. Revoking the caller's own session punishes the user for
	// rotating, which is exactly the behaviour that stops people rotating. The
	// tenant knob (session config: revoke_sessions_on_password_change) still
	// decides WHETHER to revoke; it just applies to the others.
	revokeSessions := secpolicy.ShouldRevokeSessionsOnPasswordChange(s.securitySettingRepo, tenantID)
	if revokeSessions && s.sessionRepo == nil {
		return nil, apperror.NewInternal("cannot revoke sessions on password change: no session store configured", nil)
	}
	// No identifiable caller session means we cannot preserve one, so revoke
	// everything and tell the client to re-authenticate. Never silently skip.
	reauthRequired := revokeSessions && callerSessionUUID == nil

	now := time.Now()
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if _, err := s.userRepo.WithTx(tx).UpdateByID(user.UserID, map[string]any{
			"password":              string(hashed),
			"force_password_change": false,
			"password_changed_at":   now,
			// Mirrors the reset path: a temp password that has been replaced
			// must stop being subject to temp-password expiry.
			"temporary_password_expires_at": nil,
		}); err != nil {
			return apperror.NewInternal("failed to update password", err)
		}

		if s.passwordHistoryRepo != nil {
			if err := secpolicy.RecordPasswordHistory(
				s.passwordHistoryRepo.WithTx(tx), user.UserID, policy.HistoryCount, string(hashed),
			); err != nil {
				return apperror.NewInternal("failed to record password history", err)
			}
		}

		// A password change invalidates every OAuth refresh token, including the
		// caller's own — even when callerSessionUUID keeps their current session
		// alive. Refresh tokens are long-lived bearer credentials minted under the
		// OLD password; leaving them usable means changing your password after a
		// compromise does not actually lock the attacker out. The caller stays
		// signed in on this device via their session, so the UX cost is nil.
		revokeRefreshTokens := func(tx *gorm.DB) error {
			if s.refreshRevoker == nil {
				return nil
			}
			if _, err := s.refreshRevoker.WithTx(tx).RevokeByUserID(user.UserID); err != nil {
				return apperror.NewInternal("failed to revoke refresh tokens on password change", err)
			}
			return nil
		}

		if !revokeSessions {
			// Even when the tenant opts out of session revocation, a changed
			// credential must not leave old refresh tokens spendable.
			return revokeRefreshTokens(tx)
		}
		// WithTx: this must commit or roll back WITH the password update. Without
		// it the revoke ran on its own connection, so a rolled-back password
		// change could still have signed the user out everywhere.
		if callerSessionUUID == nil {
			if err := s.sessionRepo.WithTx(tx).RevokeAllByUserID(user.UserID, shared.SessionRevokePasswordChange); err != nil {
				return apperror.NewInternal("failed to revoke sessions on password change", err)
			}
			return revokeRefreshTokens(tx)
		}
		if err := s.sessionRepo.WithTx(tx).RevokeAllExceptUUID(user.UserID, *callerSessionUUID, shared.SessionRevokePasswordChange); err != nil {
			return apperror.NewInternal("failed to revoke other sessions on password change", err)
		}
		return revokeRefreshTokens(tx)
	})
	if txErr != nil {
		s.logPasswordChangeEvent(ctx, user, false, "password update failed")
		return nil, txErr
	}

	s.logPasswordChangeEvent(ctx, user, true, "password changed by the account owner")

	span.SetStatus(codes.Ok, "")
	return &ChangePasswordResponseDTO{
		OtherSessionsRevoked:     revokeSessions,
		ReauthenticationRequired: reauthRequired,
	}, nil
}

// checkPasswordReuse compares the candidate against the user's recent hashes.
func (s *accountService) checkPasswordReuse(userID int64, historyCount int, newPassword string) error {
	if historyCount <= 0 {
		return nil
	}
	hashes, err := s.passwordHistoryRepo.FindRecentHashes(userID, historyCount)
	if err != nil {
		// Fail closed: an unreadable history is not an empty history.
		return apperror.NewInternal("failed to read password history", err)
	}
	for _, h := range hashes {
		if security.ComparePassword([]byte(h), []byte(newPassword)) {
			return apperror.NewValidation("password was used recently and cannot be reused")
		}
	}
	return nil
}

// logPasswordChangeEvent emits the auth event for a password change attempt.
//
// It routes through authevent rather than sending an email: the two existing
// user-facing security notifications in this service (notify_user_on_lockout and
// new_device_notification_enabled) both emit auth events too, and the event
// system is what fans out to webhooks. Adding a bespoke email here would be a
// second, parallel notification mechanism.
//
// The description NEVER contains the password, its length, or any fragment.
func (s *accountService) logPasswordChangeEvent(ctx context.Context, user *User, success bool, description string) {
	if s.authEventService == nil || user == nil {
		return
	}
	eventType := authevent.AuthEventTypePasswordChange
	severity := authevent.AuthEventSeverityInfo
	result := authevent.AuthEventResultSuccess
	if !success {
		eventType = authevent.AuthEventTypePasswordChangeFail
		severity = authevent.AuthEventSeverityWarn
		result = authevent.AuthEventResultFailure
	}
	userID := user.UserID
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    user.TenantID,
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   eventType,
		Severity:    severity,
		Result:      result,
		Description: ptr.Ptr(description),
	})
}

// DeleteAccount verifies the current password, anonymizes direct PII, and
// revokes/removes account data that should not survive GDPR erasure.
func (s *accountService) DeleteAccount(ctx context.Context, userID int64, currentPassword string) error {
	_, span := otel.Tracer("service").Start(ctx, "account.deleteAccount")
	defer span.End()

	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return apperror.NewNotFound("user not found")
	}

	if user.Password == nil {
		return apperror.NewValidation("account has no password set")
	}
	if !security.ComparePassword([]byte(*user.Password), []byte(currentPassword)) {
		return apperror.NewUnauthorized("invalid current password")
	}

	anonymized := fmt.Sprintf("deleted-%d-%s", user.UserID, user.UserUUID.String()[:8])
	if _, err := s.userRepo.UpdateByID(user.UserID, map[string]any{
		"username":            anonymized,
		"email":               nil,
		"phone":               nil,
		"password":            nil,
		"is_email_verified":   false,
		"is_phone_verified":   false,
		"is_totp_enabled":     false,
		"is_webauthn_enabled": false,
		"status":              "deleted",
	}); err != nil {
		return apperror.NewInternal("failed to delete account", err)
	}
	if err := s.userTokenRepo.RevokeAllByUserID(user.UserID); err != nil {
		return apperror.NewInternal("failed to revoke account tokens", err)
	}
	if err := s.userTokenRepo.DeleteByUserID(user.UserID); err != nil {
		return apperror.NewInternal("failed to remove account tokens", err)
	}
	if err := s.profileRepo.DeleteByUserID(user.UserID); err != nil {
		return apperror.NewInternal("failed to remove profile data", err)
	}
	if err := s.userSettingRepo.DeleteByUserID(user.UserID); err != nil {
		return apperror.NewInternal("failed to remove user settings", err)
	}
	if err := s.userIdentityRepo.DeleteByUserID(user.UserID); err != nil {
		return apperror.NewInternal("failed to remove linked identities", err)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// ExportAccountData collects and returns all personal data for a user.
func (s *accountService) ExportAccountData(ctx context.Context, userID int64) (*AccountExportDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "account.exportData")
	defer span.End()

	user, err := s.userRepo.FindByID(userID, "Roles", "Profile", "UserSetting")
	if err != nil || user == nil {
		return nil, apperror.NewNotFound("user not found")
	}

	roleNames := make([]string, len(user.Roles))
	for i, role := range user.Roles {
		roleNames[i] = role.Name
	}

	export := &AccountExportDTO{
		UserUUID:  user.UserUUID.String(),
		Username:  user.Username,
		Email:     user.Email,
		Phone:     user.Phone,
		CreatedAt: user.CreatedAt,
		Roles:     roleNames,
	}

	if user.Profile != nil {
		export.Profile = user.Profile
	}
	if user.UserSetting != nil {
		export.Settings = user.UserSetting
	}

	span.SetStatus(codes.Ok, "")
	return export, nil
}

// GenerateBackupCodes deletes existing backup codes and generates 10 fresh ones.
func (s *accountService) GenerateBackupCodes(ctx context.Context, userID int64) (*GenerateBackupCodesResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "account.generateBackupCodes")
	defer span.End()

	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return nil, apperror.NewNotFound("user not found")
	}

	// Delete all existing backup codes for this user
	if err := s.mfaBackupCodeRepo.DeleteAllByUserID(userID); err != nil {
		return nil, apperror.NewInternal("failed to clear existing backup codes", err)
	}

	plaintextCodes := make([]string, backupCodeCount)
	records := make([]*UserMFABackupCode, backupCodeCount)

	for i := 0; i < backupCodeCount; i++ {
		code, err := crypto.GenerateRandomString(backupCodeLength)
		if err != nil {
			return nil, apperror.NewInternal("failed to generate backup code", err)
		}
		// Truncate to exactly backupCodeLength characters for consistent display
		if len(code) > backupCodeLength {
			code = code[:backupCodeLength]
		}
		plaintextCodes[i] = code
		records[i] = &UserMFABackupCode{
			UserID:   userID,
			CodeHash: crypto.HashAuthorizationCode(code),
		}
	}

	if err := s.mfaBackupCodeRepo.CreateBulk(records); err != nil {
		return nil, apperror.NewInternal("failed to store backup codes", err)
	}

	span.SetStatus(codes.Ok, "")
	return &GenerateBackupCodesResponseDTO{Codes: plaintextCodes}, nil
}

// VerifyBackupCode looks up a user by email, verifies a backup code, and issues tokens.
func (s *accountService) VerifyBackupCode(ctx context.Context, req VerifyBackupCodeDTO) (*LoginResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "account.verifyBackupCode")
	defer span.End()

	var user *User
	var client *Client
	var userIdentitySub string

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)
		txIdpRepo := s.identityProviderRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txBackupCodeRepo := s.mfaBackupCodeRepo.WithTx(tx)

		// Validate identity provider and client
		idp, txErr := txIdpRepo.FindByIdentifier(req.ProviderID)
		if txErr != nil || idp == nil {
			return apperror.NewUnauthorized("authentication failed")
		}

		client, txErr = txClientRepo.FindByClientIDAndIdentityProvider(req.ClientID, req.ProviderID)
		if txErr != nil || client == nil ||
			client.Status != shared.StatusActive ||
			client.Domain == nil || *client.Domain == "" {
			return apperror.NewUnauthorized("authentication failed")
		}

		// Check tenant MFA policy: backup-code recovery is an MFA factor;
		// when MFA is disabled for the tenant, recovery via backup codes is
		// also disabled.
		if policy := secpolicy.LoadMFAPolicy(s.securitySettingRepo, accountClientTenantID(client)); policy != nil && policy.Mode == "disabled" {
			return apperror.NewUnauthorized("backup code recovery is unavailable")
		}

		// Find user by email, scoped to the client's tenant.
		user, txErr = txUserRepo.FindByEmailAndTenantID(req.Email, accountClientTenantID(client))
		if txErr != nil {
			return apperror.NewInternal("failed to look up user", txErr)
		}
		if user == nil {
			return apperror.NewUnauthorized("invalid email or backup code")
		}
		if user.Status != shared.StatusActive {
			return apperror.NewUnauthorized("account is not active")
		}

		// Find and verify backup code
		codeHash := crypto.HashAuthorizationCode(req.Code)
		backupCode, txErr := txBackupCodeRepo.FindByUserIDAndCodeHash(user.UserID, codeHash)
		if txErr != nil {
			return apperror.NewInternal("failed to verify backup code", txErr)
		}
		if backupCode == nil {
			return apperror.NewUnauthorized("invalid email or backup code")
		}

		// Mark code as used (single-use)
		if txErr := txBackupCodeRepo.MarkUsed(backupCode.BackupCodeID); txErr != nil {
			return apperror.NewInternal("failed to mark backup code as used", txErr)
		}

		// Resolve user identity sub
		userIdentity, txErr := txUserIdentityRepo.FindByUserIDAndClientID(user.UserID, client.ClientID)
		if txErr != nil || userIdentity == nil {
			return apperror.NewUnauthorized("authentication failed")
		}
		userIdentitySub = userIdentity.Sub

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "backup code verification failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return s.generateTokenResponse(ctx, userIdentitySub, user, client)
}

// SendPhoneVerification generates an SMS OTP and sends it to the given phone so
// the authenticated user can prove ownership of the number. It mirrors the MFA
// SMS enrollment flow: prior pending OTPs for this user+channel are cleared, a
// fresh hashed OTP is stored, and delivery is best-effort — a missing/failing
// SMS provider is logged (with the OTP, for dev) and never hard-fails the call.
func (s *accountService) SendPhoneVerification(ctx context.Context, userID int64, phone string) error {
	_, span := otel.Tracer("service").Start(ctx, "account.sendPhoneVerification")
	defer span.End()

	if phone == "" {
		return apperror.NewValidation("phone is required")
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return apperror.NewNotFound("user not found")
	}

	otpCode, err := crypto.GenerateOTP(phoneVerifyOTPLength)
	if err != nil {
		return apperror.NewInternal("failed to generate SMS OTP", err)
	}
	otpHash := crypto.HashAuthorizationCode(otpCode)

	// Clear any prior pending phone-verification OTPs for this user.
	s.db.Where("user_id = ? AND channel = ?", userID, phoneVerifyChannel).Delete(&notifier.UserOTP{})

	record := &notifier.UserOTP{
		UserID:    userID,
		Channel:   phoneVerifyChannel,
		Recipient: phone,
		OTPHash:   otpHash,
		ExpiresAt: time.Now().Add(phoneVerifyOTPTTL),
	}
	if _, err := s.smsOtpRepo.Create(record); err != nil {
		return apperror.NewInternal("failed to store SMS OTP", err)
	}

	tenantID := user.TenantID
	provider, smsErr := accountNewSMSProvider(ctx, s.db, tenantID)
	if smsErr != nil {
		slog.Warn("SMS provider init failed; phone-verification OTP not delivered", "err", smsErr, "phone", phone, "otp", security.RedactedOTP(otpCode))
	} else if provider != nil {
		data := struct{ OTP string }{OTP: otpCode}
		msg, tplErr := sms.RenderTemplate(s.db, "sms:phone:verify", tenantID, data)
		if tplErr != nil {
			slog.Warn("SMS template render failed, using fallback", "err", tplErr)
			msg = fmt.Sprintf("Your phone verification code is: %s", otpCode)
		}
		if sendErr := provider.Send(ctx, phone, msg); sendErr != nil {
			slog.Error("SMS phone verification send failed", "err", sendErr, "phone", phone, "otp", security.RedactedOTP(otpCode))
		}
	} else {
		slog.Warn("no SMS provider configured; phone-verification OTP not delivered", "phone", phone, "otp", security.RedactedOTP(otpCode))
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// VerifyPhone confirms an SMS OTP sent by SendPhoneVerification and, on success,
// sets the authenticated user's phone and marks it verified. It mirrors the MFA
// VerifySMS flow: single-use OTP lookup by channel+recipient, constant-time hash
// comparison, failure accounting, and mark-used before applying the change.
func (s *accountService) VerifyPhone(ctx context.Context, userID int64, phone, code string) error {
	_, span := otel.Tracer("service").Start(ctx, "account.verifyPhone")
	defer span.End()

	otpRecord, lerr := s.smsOtpRepo.FindValid(phoneVerifyChannel, phone)
	if lerr != nil || otpRecord == nil {
		return apperror.NewUnauthorized("invalid or expired verification code")
	}

	if subtle.ConstantTimeCompare([]byte(otpRecord.OTPHash), []byte(crypto.HashAuthorizationCode(code))) != 1 {
		_ = s.smsOtpRepo.RecordFailure(otpRecord.UserOTPID, phoneVerifyMaxFailed)
		return apperror.NewUnauthorized("invalid verification code")
	}
	if err := s.smsOtpRepo.MarkUsed(otpRecord.UserOTPID); err != nil {
		return apperror.NewInternal("failed to mark verification code used", err)
	}

	if _, err := s.userRepo.UpdateByID(userID, map[string]any{
		"phone":             phone,
		"is_phone_verified": true,
		"phone_verified_at": time.Now(),
	}); err != nil {
		return apperror.NewInternal("failed to update phone", err)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// generateTokenResponse issues access, ID, and refresh tokens for the given user and client.
func (s *accountService) generateTokenResponse(ctx context.Context, sub string, user *User, client *Client) (*LoginResponseDTO, error) {
	accessToken, err := accountGenerateAccessTokenWithContext(
		ctx,
		sub,
		shared.DefaultTokenScope,
		*client.Domain,
		*client.Identifier,
		*client.Identifier,
		accountTokenRealm(client),
	)
	if err != nil {
		return nil, err
	}

	profile := &jwt.UserProfile{
		Email:         user.Email,
		EmailVerified: user.IsEmailVerified,
		Phone:         user.Phone,
		PhoneVerified: user.IsPhoneVerified,
	}

	idToken, err := accountGenerateIDTokenWithContext(ctx, sub, *client.Domain, *client.Identifier, accountTokenRealm(client), profile, "", nil)
	if err != nil {
		return nil, err
	}

	refreshToken, err := accountGenerateRefreshTokenWithContext(ctx, sub, *client.Domain, *client.Identifier, accountTokenRealm(client))
	if err != nil {
		return nil, err
	}

	return &LoginResponseDTO{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: refreshToken,
		ExpiresIn:    shared.DefaultAccessTokenExpiresIn,
		TokenType:    "Bearer",
		IssuedAt:     time.Now().Unix(),
	}, nil
}

func accountClientTenantID(client *Client) int64 {
	if client == nil {
		return 0
	}
	if client.TenantID > 0 {
		return client.TenantID
	}
	if client.IdentityProvider != nil {
		return client.IdentityProvider.TenantID
	}
	return 0
}

func accountTokenRealm(client *Client) string {
	if client == nil {
		return ""
	}
	if client.IdentityProvider != nil && client.IdentityProvider.Identifier != "" {
		return client.IdentityProvider.Identifier
	}
	if client.Identifier != nil {
		return *client.Identifier
	}
	return ""
}

package user

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
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
	"golang.org/x/crypto/bcrypt"
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

// backupCodeLength matches the mfa package's mfaBackupCodeLength. Both services
// write into user_mfa_backup_codes, so a code minted here must look and hash
// exactly like one minted there.
	//nolint:unused
const backupCodeLength = 10

// Backup-code recovery is a two-factor exchange (password + code), so wrong
// attempts are metered per account. The endpoint is registered outside the
// strict auth rate-limit group, and it hands out a full token set, so without
// this the only ceiling is the global 100/min/IP — enough to walk a code space
// and enough to brute a password with the code held constant. The key is
// deliberately NOT the login lockout key: an attacker must not be able to lock
// the victim out of normal sign-in by hammering recovery.
const recoveryBackupCodeThrottlePrefix = "recovery-backup-code:"

// Phone-verification SMS OTP settings. A distinct channel keeps these OTPs from
// colliding with MFA's "sms" OTPs stored in the same user_otps table.
const phoneVerifyOTPLength = 6
const phoneVerifyOTPTTL = 10 * time.Minute
const phoneVerifyMaxFailed = 3
const phoneVerifyChannel = "phone_verify"

var (
	accountGenerateAccessTokenWithContext = jwt.GenerateAccessTokenWithContext
	// Options variant: recovery logins must stamp a `sid` so the token they
	// return can be revoked like any other login's.
	accountGenerateAccessTokenWithOptionsContext = jwt.GenerateAccessTokenWithOptionsContext
	accountGenerateIDTokenWithContext            = jwt.GenerateIDTokenWithContext
	accountGenerateRefreshTokenWithContext       = jwt.GenerateRefreshTokenWithContext
	// accountNewSMSProvider is the SMS provider factory. It is a package-level
	// indirection over sms.NewProviderFromDB so tests can swap it out.
	accountNewSMSProvider = sms.NewProviderFromDB
)

// AccountService handles self-service account management operations.
type AccountService interface {
	// SetSessionCreator wires the store used to bind recovery logins to a real,
	// revocable session.
	SetSessionCreator(SessionCreator)
	InitiateEmailChange(ctx context.Context, userID int64, newEmail, currentPassword string) error
	VerifyEmailChange(ctx context.Context, userID int64, otp string) error
	ChangeUsername(ctx context.Context, userID int64, newUsername, currentPassword string) error
	ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword string, callerSessionUUID *uuid.UUID) (*ChangePasswordResponseDTO, error)
	// RevokeOtherSessions is "sign out my other devices" — see the method.
	RevokeOtherSessions(ctx context.Context, userID int64, keepSessionUUID uuid.UUID) error
	DeleteAccount(ctx context.Context, userID int64, currentPassword string) error
	ExportAccountData(ctx context.Context, userID int64) (*AccountExportDTO, error)
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
	// sessionCreator binds recovery logins to a real session. Nil makes those
	// logins fail closed rather than mint an unusable token.
	sessionCreator SessionCreator
}

// SetSessionCreator wires the session store used to bind recovery logins.
func (s *accountService) SetSessionCreator(c SessionCreator) { s.sessionCreator = c }

// RefreshTokenRevoker is the slice of the OAuth refresh-token store this
// service needs. Declared here rather than importing internal/oauth, which
// already imports this package — internal/app supplies the adapter.
type RefreshTokenRevoker interface {
	// WithTx joins the caller's transaction so the password update and the
	// revocation commit or roll back together.
	WithTx(tx *gorm.DB) RefreshTokenRevoker
	RevokeByUserID(userID int64) (int64, error)
}

// SessionCreator mints a canonical session row for a login this service
// completes. Backup-code recovery is an interactive login, so its token must be
// bound to a session like every other one: a token with no `sid` cannot be
// ended by logout, "sign out everywhere", or session revocation, and is
// rejected outright by the session middleware. internal/app supplies the
// adapter — importing authn here would invert the dependency.
type SessionCreator interface {
	CreateSession(ctx context.Context, userID, tenantID int64, ipAddress, userAgent string) (sessionUUID uuid.UUID, err error)
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

	// Sent synchronously so a delivery failure reaches the caller.
	//
	// This used to run in a detached goroutine, so the endpoint returned 200 and
	// the UI said "check your inbox" while the send failed and the only trace was
	// a log line. For an OTP the user cannot proceed without, that is the worst
	// possible outcome: no mail, no error, nothing to retry against. Every other
	// email flow here (password reset, magic link, verification) already returns
	// the send error; this one was the outlier.
	if sendErr := s.sendEmailChangeOTP(ctx, email.SendEmail, user.TenantID, newEmail, otp); sendErr != nil {
		slog.Error("account: failed to send email change OTP", "error", sendErr, "user_id", userID)
		span.SetStatus(codes.Error, "email change OTP send failed")
		// The stored OTP is left to expire; a retry simply issues a new one.
		return apperror.NewInternal("could not send the verification email, please try again", sendErr)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (s *accountService) sendEmailChangeOTP(ctx context.Context, sendEmail func(context.Context, *gorm.DB, email.SendEmailParams) error, tenantID int64, toEmail, otp string) error {
	data := struct {
		OTP     string
		LogoURL string
	}{
		OTP:     otp,
		LogoURL: email.GetLogoURL(ctx, s.db, tenantID),
	}

	rendered, err := email.RenderTemplate(s.db, "user:email:change", tenantID, data)
	if err != nil {
		return fmt.Errorf("failed to render email change template: %w", err)
	}
	return sendEmail(ctx, s.db, email.SendEmailParams{
		// Without this the provider lookup runs with tenant 0, misses, and silently
		// falls back to the SYSTEM tenant's SMTP config — so a tenant's mail would go
		// out through another tenant's server and From address.
		TenantID:  tenantID,
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

	// Apply the change BEFORE consuming the OTP.
	//
	// The order used to be the other way round, so a lost uniqueness race (two
	// accounts confirming the same new address) burned the loser's single-use OTP
	// on a write that never landed — they had to restart the whole flow through
	// their inbox for someone else's collision. The OTP is only spent once the
	// address is actually theirs; a duplicate MarkUsed on a retry is harmless
	// because the row is already the one we just consumed.
	previousEmail := user.Email
	if err := s.userRepo.UpdateEmail(user.UserUUID, newEmail); err != nil {
		// A unique index rejected the write: someone else took the address between
		// the availability pre-check and here. That is a 409, not a 500 — and
		// wrapping it in NewInternal made it one, because HandleServiceError tests
		// errors.As(&internal) BEFORE errors.Is(gorm.ErrDuplicatedKey), so the
		// duplicate-key backstop was unreachable behind an Internal wrapper.
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return apperror.NewConflict("email address is already in use")
		}
		return apperror.NewInternal("failed to update email", err)
	}

	// Single-use.
	if err := s.smsOtpRepo.MarkUsed(otpRecord.UserOTPID); err != nil {
		return apperror.NewInternal("failed to mark OTP used", err)
	}

	// Tell the address that just LOST the account. An attacker who reaches a live
	// session can move the sign-in identity to a mailbox they control; the
	// out-of-band notice to the previous address is the standard — and here the
	// only — signal that lets the real owner notice and react. Best-effort: the
	// change has already committed, so a delivery failure must not fail the call
	// or roll anything back.
	s.notifyPreviousEmailOfChange(ctx, user, previousEmail, newEmail)

	span.SetStatus(codes.Ok, "")
	return nil
}

// notifyPreviousEmailOfChange warns the old address that the account's sign-in
// email moved, and records an auth event the user can see in their security log.
//
// The email body is composed inline rather than rendered from a template: there
// is no seeded "email changed" security notice, and falling back to an unrelated
// template would send the wrong message. The auth event fires regardless of
// whether the mail goes out, so the signal survives a broken SMTP config.
func (s *accountService) notifyPreviousEmailOfChange(ctx context.Context, user *User, previousEmail, newEmail string) {
	if s.authEventService != nil {
		userID := user.UserID
		s.authEventService.Log(ctx, authevent.AuthEventInput{
			TenantID:    user.TenantID,
			ActorUserID: &userID,
			IPAddress:   middleware.ClientIPFromContext(ctx),
			UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
			Category:    authevent.AuthEventCategoryAuthn,
			EventType:   authevent.AuthEventTypeUserUpdated,
			Severity:    authevent.AuthEventSeverityWarn,
			Result:      authevent.AuthEventResultSuccess,
			// The new address is NOT recorded here: the event is readable by
			// whoever holds the account, which in the takeover case is the attacker.
			Description: ptr.Ptr("Account email address was changed"),
		})
	}

	if previousEmail == "" || previousEmail == newEmail {
		return
	}
	subject := "Your account email address was changed"
	bodyPlain := fmt.Sprintf(
		"The email address on your account was just changed from %s to %s.\n\n"+
			"If you did not make this change, contact your administrator immediately — "+
			"whoever made it can now receive your sign-in and password-reset mail.",
		previousEmail, newEmail)
	if err := email.SendEmail(ctx, s.db, email.SendEmailParams{
		TenantID:  user.TenantID,
		To:        previousEmail,
		Subject:   subject,
		BodyHTML:  fmt.Sprintf("<p>%s</p>", bodyPlain),
		BodyPlain: bodyPlain,
	}); err != nil {
		slog.Error("account: failed to notify previous email of address change", "error", err, "user_id", user.UserID)
	}
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
		// The pre-check above cannot close the race with a concurrent writer. A
		// lost race is a 409; wrapping it in NewInternal made it a 500, because
		// HandleServiceError tests errors.As(&internal) BEFORE
		// errors.Is(gorm.ErrDuplicatedKey) and never reaches the backstop.
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return apperror.NewConflict("username is already taken")
		}
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

// RevokeOtherSessions ends every session the user holds EXCEPT the one the
// request arrived on.
//
// It exists because the only bulk control was RevokeAllSessions, which also
// kills the caller's own session: a user who suspects one device is
// compromised had to sign themselves out of the device they were sitting at to
// deal with it, and any UI offering "sign out my other devices" was lying.
// RevokeAllExceptUUID already existed on the session store for exactly this
// shape and had no caller on the account path.
//
// OAuth refresh tokens are deliberately NOT revoked here, and this is the one
// place that is correct. The only revoker this service holds is user-wide
// (RevokeByUserID), so using it would take the caller's own refresh token with
// it — the browser that asked to sign out its *other* devices would be forced
// to re-login at its next refresh, which is the exact bug this method exists to
// avoid. Nothing is left spendable by skipping it: a refresh is rejected unless
// it carries a `sid` naming a still-live session (see the authn refresh
// service), and every other session's row is revoked below. Credential changes,
// which DO need the user-wide token revoke, keep going through ChangePassword /
// RevokeAllSessions.
func (s *accountService) RevokeOtherSessions(ctx context.Context, userID int64, keepSessionUUID uuid.UUID) error {
	_, span := otel.Tracer("service").Start(ctx, "account.revokeOtherSessions")
	defer span.End()

	// Fail closed on a missing caller session. uuid.Nil matches no row, so
	// RevokeAllExceptUUID would quietly degrade into a full sign-out-everywhere
	// and log the caller out of the browser they clicked in — the opposite of
	// what this endpoint promises. The caller must use RevokeAllSessions if that
	// is what they actually want.
	if keepSessionUUID == uuid.Nil {
		span.SetStatus(codes.Error, "no caller session")
		return apperror.NewValidation("this request is not bound to a session; use sign out everywhere instead")
	}

	if err := s.sessionRepo.RevokeAllExceptUUID(userID, keepSessionUUID, shared.SessionRevokeUserRevoke); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "revoke other sessions failed")
		return apperror.NewInternal("failed to revoke other sessions", err)
	}

	span.SetStatus(codes.Ok, "")
	return nil
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

// hashBackupCode hashes a backup code for storage.
//
// bcrypt, matching the mfa service, which writes and reads the same
// user_mfa_backup_codes table. This used to be crypto.HashAuthorizationCode — an
// UNSALTED SHA-256 — which broke two ways at once. Functionally: the mfa
// verifier does a bcrypt compare and the account verifier did a SHA-256 lookup,
// so codes issued by one path could never be redeemed by the other, while
// /mfa/backup-codes/count happily reported them as remaining. Cryptographically:
// an unsalted digest of a short random string is a rainbow-table lookup, and
	//nolint:unused
// these codes are full account-recovery credentials.
func hashBackupCode(code string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// isRedeemableBackupCodeHash reports whether a stored hash is in the one format
// the verifiers understand. The mfa package keeps its own copy for the same
// table — internal/user and internal/mfa are siblings, so neither imports the
// other.
func isRedeemableBackupCodeHash(hash string) bool {
	_, err := bcrypt.Cost([]byte(hash))
	return err == nil
}

// Backup-code GENERATION is owned exclusively by mfaService.RegenerateBackupCodes.
//
// This service used to carry its own copy behind POST /account/backup-codes,
// which no SPA ever called. It was not a harmless duplicate: it minted a
// hardcoded 10 codes and never consulted the tenant's MFA policy, so a tenant
// that forbids the backup_code method or sets a different recovery_codes_count
// had that decision bypassed by anyone calling the endpoint directly. Only
// verification — the recovery half below, which needs this service's login
// machinery — remains here.

// VerifyBackupCode recovers account access from a password plus one unused
// backup code, and issues tokens.
//
// The password is not optional. Without it this endpoint minted a full access +
// refresh token set from an email address and a single short code — a backup
// code is a recovery SECOND factor, not a standalone primary credential, so
// accepting it alone let anyone holding one walk past the tenant's enforced-MFA
// policy entirely. Attempts are metered per account (see
// recoveryBackupCodeThrottlePrefix) because the route sits outside the strict
// auth rate-limit group and records no login lockout.
func (s *accountService) VerifyBackupCode(ctx context.Context, req VerifyBackupCodeDTO) (*LoginResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "account.verifyBackupCode")
	defer span.End()

	var user *User
	var client *Client
	var userIdentitySub string

	// Keyed on the client + email the caller supplied, so the counter exists even
	// when no such account does — an unthrottled "no such user" path is itself an
	// enumeration and brute-force channel.
	throttleKey := recoveryBackupCodeThrottlePrefix + req.ClientID + ":" + req.Email
	if err := security.CheckRateLimit(throttleKey); err != nil {
		return nil, apperror.NewTooManyRequests("too many recovery attempts; try again later")
	}

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
			// Burn a bcrypt comparison against a dummy hash so a missing account
			// is not distinguishable from a wrong password by response time.
			_ = security.ComparePassword(security.GetDummyBcryptHash(), []byte(req.Password))
			return apperror.NewUnauthorized("invalid email, password, or backup code")
		}
		if user.Status != shared.StatusActive {
			return apperror.NewUnauthorized("account is not active")
		}

		// First factor: the account password. A federated/passwordless account has
		// nothing to check, so it cannot recover this way — it has no password to
		// pair with the code, and letting the code stand alone is the whole defect.
		if user.Password == nil {
			return apperror.NewUnauthorized("invalid email, password, or backup code")
		}
		if !security.ComparePassword([]byte(*user.Password), []byte(req.Password)) {
			return apperror.NewUnauthorized("invalid email, password, or backup code")
		}

		// Second factor: an unused backup code. bcrypt is salted, so the stored
		// hash cannot be looked up by value — every unused code is compared. This
		// replaces a lookup by unsalted SHA-256 that could never match anything the
		// SPAs actually issue (those are bcrypt), leaving /recovery/backup-code
		// permanently broken while the count endpoint claimed codes remained.
		bCodes, txErr := txBackupCodeRepo.FindUnusedByUserID(user.UserID)
		if txErr != nil {
			return apperror.NewInternal("failed to verify backup code", txErr)
		}
		var backupCode *UserMFABackupCode
		for i := range bCodes {
			if !isRedeemableBackupCodeHash(bCodes[i].CodeHash) {
				continue
			}
			if bcrypt.CompareHashAndPassword([]byte(bCodes[i].CodeHash), []byte(req.Code)) == nil {
				backupCode = &bCodes[i]
				break
			}
		}
		if backupCode == nil {
			return apperror.NewUnauthorized("invalid email, password, or backup code")
		}

		// Mark code as used (single-use)
		if txErr := txBackupCodeRepo.MarkUsed(backupCode.BackupCodeID); txErr != nil {
			return apperror.NewInternal("failed to mark backup code as used", txErr)
		}

		// Resolve user identity sub
		userIdentity, txErr := txUserIdentityRepo.FindByUserIDAndClientReachable(user.UserID, client.ClientID)
		if txErr != nil || userIdentity == nil {
			return apperror.NewUnauthorized("authentication failed")
		}
		userIdentitySub = userIdentity.Sub

		return nil
	})

	if err != nil {
		security.RecordFailedAttempt(throttleKey)
		span.RecordError(err)
		span.SetStatus(codes.Error, "backup code verification failed")
		return nil, err
	}
	security.ResetFailedAttempts(throttleKey)

	// Recovery completes a login, so it must produce a session-bound token. A
	// token with no `sid` survives logout and every session-revocation path, and
	// the session middleware now rejects it — so minting one would hand the user
	// a credential that fails on their very next request.
	if s.sessionCreator == nil {
		return nil, apperror.NewInternal("session store unavailable for recovery login", nil)
	}
	sessionUUID, sessErr := s.sessionCreator.CreateSession(
		ctx, user.UserID, accountClientTenantID(client),
		middleware.ClientIPFromContext(ctx), middleware.UserAgentFromContext(ctx),
	)
	if sessErr != nil {
		return nil, apperror.NewInternal("failed to start a session for recovery login", sessErr)
	}

	span.SetStatus(codes.Ok, "")
	return s.generateTokenResponse(ctx, userIdentitySub, user, client, sessionUUID.String())
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
func (s *accountService) generateTokenResponse(ctx context.Context, sub string, user *User, client *Client, sessionID string) (*LoginResponseDTO, error) {
	// SessionID is what makes the token revocable: logout, "sign out everywhere",
	// and session revocation all operate on user_sessions, and the session
	// middleware rejects a user token that carries no `sid`.
	accessToken, err := accountGenerateAccessTokenWithOptionsContext(
		ctx,
		sub,
		shared.DefaultTokenScope,
		jwt.TokenIssuerPtr(client.Domain),
		*client.Identifier,
		*client.Identifier,
		accountTokenRealm(client),
		&jwt.AccessTokenOptions{SessionID: sessionID},
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

	idToken, err := accountGenerateIDTokenWithContext(ctx, sub, jwt.TokenIssuerPtr(client.Domain), *client.Identifier, accountTokenRealm(client), profile, "", nil)
	if err != nil {
		return nil, err
	}

	refreshToken, err := accountGenerateRefreshTokenWithContext(ctx, sub, jwt.TokenIssuerPtr(client.Domain), *client.Identifier, accountTokenRealm(client))
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

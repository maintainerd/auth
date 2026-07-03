package user

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/notifier"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/platform/email"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/platform/sms"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

const emailChangeOTPLength = 6
const emailChangeOTPTTL = 1 * time.Hour
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
) AccountService {
	return &accountService{
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

	expiresAt := time.Now().Add(emailChangeOTPTTL)
	if err := s.userRepo.SetPendingEmail(user.UserUUID, newEmail, otpHash, expiresAt); err != nil {
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

	if user.PendingEmail == nil || user.EmailChangeOTP == nil || user.EmailChangeOTPExpiresAt == nil {
		return apperror.NewValidation("no pending email change found")
	}

	if time.Now().After(*user.EmailChangeOTPExpiresAt) {
		return apperror.NewValidation("email change OTP has expired")
	}

	expectedHash := crypto.HashAuthorizationCode(otp)
	if subtle.ConstantTimeCompare([]byte(*user.EmailChangeOTP), []byte(expectedHash)) != 1 {
		return apperror.NewUnauthorized("invalid OTP")
	}

	newEmail := *user.PendingEmail

	if err := s.userRepo.UpdateEmail(user.UserUUID, newEmail); err != nil {
		return apperror.NewInternal("failed to update email", err)
	}

	if err := s.userRepo.ClearEmailChange(user.UserUUID); err != nil {
		// Non-fatal — email was already updated
		slog.Warn("account: failed to clear email change fields", "error", err, "user_id", userID)
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
		"username":                    anonymized,
		"email":                       nil,
		"phone":                       nil,
		"password":                    nil,
		"pending_email":               nil,
		"email_change_otp":            nil,
		"email_change_otp_expires_at": nil,
		"is_email_verified":           false,
		"is_phone_verified":           false,
		"is_profile_completed":        false,
		"is_account_completed":        false,
		"is_totp_enabled":             false,
		"is_webauthn_enabled":         false,
		"mfa_enabled_at":              nil,
		"status":                      "deleted",
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
		slog.Warn("SMS provider init failed — logging OTP for dev", "err", smsErr, "phone", phone, "otp", otpCode)
	} else if provider != nil {
		data := struct{ OTP string }{OTP: otpCode}
		msg, tplErr := sms.RenderTemplate(s.db, "sms:phone:verify", tenantID, data)
		if tplErr != nil {
			slog.Warn("SMS template render failed, using fallback", "err", tplErr)
			msg = fmt.Sprintf("Your phone verification code is: %s", otpCode)
		}
		if sendErr := provider.Send(ctx, phone, msg); sendErr != nil {
			slog.Error("SMS phone verification send failed — logging OTP for dev", "err", sendErr, "phone", phone, "otp", otpCode)
		}
	} else {
		slog.Info("SMS OTP (no provider) — use for dev", "phone", phone, "otp", otpCode)
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

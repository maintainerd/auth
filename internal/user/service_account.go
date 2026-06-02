package user

import (
	"bytes"
	"context"
	"crypto/subtle"
	"fmt"
	"html/template"
	"log/slog"
	"time"

	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/email"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

const emailChangeOTPLength = 6
const emailChangeOTPTTL = 1 * time.Hour
const backupCodeCount = 10
const backupCodeLength = 8

// AccountService handles self-service account management operations.
type AccountService interface {
	InitiateEmailChange(ctx context.Context, userID int64, newEmail, currentPassword string) error
	VerifyEmailChange(ctx context.Context, userID int64, otp string) error
	ChangeUsername(ctx context.Context, userID int64, newUsername, currentPassword string) error
	DeleteAccount(ctx context.Context, userID int64, currentPassword string) error
	ExportAccountData(ctx context.Context, userID int64) (*AccountExportDTO, error)
	GenerateBackupCodes(ctx context.Context, userID int64) (*GenerateBackupCodesResponseDTO, error)
	VerifyBackupCode(ctx context.Context, req VerifyBackupCodeDTO) (*LoginResponseDTO, error)
}

type accountService struct {
	db                   *gorm.DB
	userRepo             UserRepository
	userTokenRepo        UserTokenRepository
	profileRepo          ProfileRepository
	userSettingRepo      UserSettingRepository
	roleRepo             RoleRepository
	clientRepo           ClientRepository
	backupCodeRepo       UserBackupCodeRepository
	userIdentityRepo     UserIdentityRepository
	identityProviderRepo IdentityProviderRepository
	authEventService     authevent.AuthEventService
}

func NewAccountService(
	db *gorm.DB,
	userRepo UserRepository,
	userTokenRepo UserTokenRepository,
	profileRepo ProfileRepository,
	userSettingRepo UserSettingRepository,
	roleRepo RoleRepository,
	clientRepo ClientRepository,
	backupCodeRepo UserBackupCodeRepository,
	userIdentityRepo UserIdentityRepository,
	identityProviderRepo IdentityProviderRepository,
	authEventService authevent.AuthEventService,
) AccountService {
	return &accountService{
		db:                   db,
		userRepo:             userRepo,
		userTokenRepo:        userTokenRepo,
		profileRepo:          profileRepo,
		userSettingRepo:      userSettingRepo,
		roleRepo:             roleRepo,
		clientRepo:           clientRepo,
		backupCodeRepo:       backupCodeRepo,
		userIdentityRepo:     userIdentityRepo,
		identityProviderRepo: identityProviderRepo,
		authEventService:     authEventService,
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

	// Check new email is not already taken
	existing, err := s.userRepo.FindByEmail(newEmail)
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

	// Send OTP email (best-effort)
	go func() {
		sendCtx := context.Background()
		if sendErr := s.sendEmailChangeOTP(sendCtx, newEmail, otp); sendErr != nil {
			slog.Error("account: failed to send email change OTP", "error", sendErr, "user_id", userID)
		}
	}()

	span.SetStatus(codes.Ok, "")
	return nil
}

func (s *accountService) sendEmailChangeOTP(ctx context.Context, toEmail, otp string) error {
	data := struct {
		OTP string
	}{OTP: otp}

	subject := "Your email change verification code"
	bodyHTML := fmt.Sprintf("<p>Your email change verification code is: <strong>%s</strong>. It expires in 1 hour.</p>", otp)

	// Try to use a template if available; fall back to inline HTML
	tmpl, err := template.New("email_change").Parse(bodyHTML)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}

	return email.SendEmail(ctx, email.SendEmailParams{
		To:       toEmail,
		Subject:  subject,
		BodyHTML: buf.String(),
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

	// Check username not taken
	existing, err := s.userRepo.FindByUsername(newUsername)
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
	if err := s.backupCodeRepo.DeleteAllByUserID(userID); err != nil {
		return nil, apperror.NewInternal("failed to clear existing backup codes", err)
	}

	plaintextCodes := make([]string, backupCodeCount)
	records := make([]*UserBackupCode, backupCodeCount)

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
		records[i] = &UserBackupCode{
			UserID:   userID,
			CodeHash: crypto.HashAuthorizationCode(code),
		}
	}

	if err := s.backupCodeRepo.CreateBulk(records); err != nil {
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
		txBackupCodeRepo := s.backupCodeRepo.WithTx(tx)

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

		// Find user by email
		user, txErr = txUserRepo.FindByEmail(req.Email)
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

// generateTokenResponse issues access, ID, and refresh tokens for the given user and client.
func (s *accountService) generateTokenResponse(ctx context.Context, sub string, user *User, client *Client) (*LoginResponseDTO, error) {
	accessToken, err := jwt.GenerateAccessTokenWithContext(
		ctx,
		sub,
		shared.DefaultTokenScope,
		*client.Domain,
		*client.Identifier,
		*client.Identifier,
		client.IdentityProvider.Identifier,
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

	idToken, err := jwt.GenerateIDTokenWithContext(ctx, sub, *client.Domain, *client.Identifier, client.IdentityProvider.Identifier, profile, "", nil)
	if err != nil {
		return nil, err
	}

	refreshToken, err := jwt.GenerateRefreshTokenWithContext(ctx, sub, *client.Domain, *client.Identifier, client.IdentityProvider.Identifier)
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

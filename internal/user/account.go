package user

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"time"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/email"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/crypto/bcrypt"
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
	ExportAccountData(ctx context.Context, userID int64) (*dto.AccountExportDTO, error)
	GenerateBackupCodes(ctx context.Context, userID int64) (*dto.GenerateBackupCodesResponseDTO, error)
	VerifyBackupCode(ctx context.Context, req dto.VerifyBackupCodeDTO) (*dto.LoginResponseDTO, error)
}

type accountService struct {
	db                   *gorm.DB
	userRepo             repository.UserRepository
	userTokenRepo        repository.UserTokenRepository
	profileRepo          repository.ProfileRepository
	userSettingRepo      repository.UserSettingRepository
	roleRepo             repository.RoleRepository
	clientRepo           repository.ClientRepository
	backupCodeRepo       repository.UserBackupCodeRepository
	userIdentityRepo     repository.UserIdentityRepository
	identityProviderRepo repository.IdentityProviderRepository
	authEventService     AuthEventService
}

func NewAccountService(
	db *gorm.DB,
	userRepo repository.UserRepository,
	userTokenRepo repository.UserTokenRepository,
	profileRepo repository.ProfileRepository,
	userSettingRepo repository.UserSettingRepository,
	roleRepo repository.RoleRepository,
	clientRepo repository.ClientRepository,
	backupCodeRepo repository.UserBackupCodeRepository,
	userIdentityRepo repository.UserIdentityRepository,
	identityProviderRepo repository.IdentityProviderRepository,
	authEventService AuthEventService,
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
	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(currentPassword)); err != nil {
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

	// Generate OTP
	otp, err := crypto.GenerateOTP(emailChangeOTPLength)
	if err != nil {
		return apperror.NewInternal("failed to generate OTP", err)
	}

	expiresAt := time.Now().Add(emailChangeOTPTTL)
	if err := s.userRepo.SetPendingEmail(user.UserUUID, newEmail, otp, expiresAt); err != nil {
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

	if *user.EmailChangeOTP != otp {
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
	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(currentPassword)); err != nil {
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

// DeleteAccount verifies the current password and soft-deletes the account.
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
	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(currentPassword)); err != nil {
		return apperror.NewUnauthorized("invalid current password")
	}

	if err := s.userRepo.SetStatus(user.UserUUID, "deleted"); err != nil {
		return apperror.NewInternal("failed to delete account", err)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// ExportAccountData collects and returns all personal data for a user.
func (s *accountService) ExportAccountData(ctx context.Context, userID int64) (*dto.AccountExportDTO, error) {
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

	export := &dto.AccountExportDTO{
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
func (s *accountService) GenerateBackupCodes(ctx context.Context, userID int64) (*dto.GenerateBackupCodesResponseDTO, error) {
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
	records := make([]*model.UserBackupCode, backupCodeCount)

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
		records[i] = &model.UserBackupCode{
			UserID:   userID,
			CodeHash: crypto.HashAuthorizationCode(code),
		}
	}

	if err := s.backupCodeRepo.CreateBulk(records); err != nil {
		return nil, apperror.NewInternal("failed to store backup codes", err)
	}

	span.SetStatus(codes.Ok, "")
	return &dto.GenerateBackupCodesResponseDTO{Codes: plaintextCodes}, nil
}

// VerifyBackupCode looks up a user by email, verifies a backup code, and issues tokens.
func (s *accountService) VerifyBackupCode(ctx context.Context, req dto.VerifyBackupCodeDTO) (*dto.LoginResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "account.verifyBackupCode")
	defer span.End()

	var user *model.User
	var client *model.Client
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
			client.Status != model.StatusActive ||
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
		if user.Status != model.StatusActive {
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
	return s.generateTokenResponse(userIdentitySub, user, client)
}

// generateTokenResponse issues access, ID, and refresh tokens for the given user and client.
func (s *accountService) generateTokenResponse(sub string, user *model.User, client *model.Client) (*dto.LoginResponseDTO, error) {
	accessToken, err := jwt.GenerateAccessToken(
		sub,
		"openid profile email",
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

	idToken, err := generateIDTokenFn(sub, *client.Domain, *client.Identifier, client.IdentityProvider.Identifier, profile, "", nil)
	if err != nil {
		return nil, err
	}

	refreshToken, err := generateRefreshTokenFn(sub, *client.Domain, *client.Identifier, client.IdentityProvider.Identifier)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponseDTO{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		IssuedAt:     time.Now().Unix(),
	}, nil
}

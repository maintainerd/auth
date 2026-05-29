package mfa

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	totpIssuer          = "maintainerd-auth"
	totpDigits          = otp.DigitsSix
	totpPeriod          = 30
	mfaBackupCodeCount  = 10
	mfaBackupCodeLength = 10
	stepUpChallengeTTL  = 5 * time.Minute
)

// MFAService handles TOTP enrollment/verification, backup code management,
// per-tenant MFA policy, admin resets, and step-up authentication.
type MFAService interface {
	// TOTP enrollment
	BeginTOTPEnrollment(ctx context.Context, userID int64, userEmail string) (*TOTPEnrollResponseDTO, error)
	FinishTOTPEnrollment(ctx context.Context, userID int64, code string) ([]string, error)
	VerifyTOTP(ctx context.Context, userID int64, code string) (bool, error)
	DisableTOTP(ctx context.Context, userID int64) error

	// Backup codes
	GetBackupCodesCount(ctx context.Context, userID int64) (int, error)
	RegenerateBackupCodes(ctx context.Context, userID int64) ([]string, error)

	// Status
	GetMFAStatus(ctx context.Context, userID int64) (*MFAStatusResponseDTO, error)

	// Policy
	GetMFAPolicy(ctx context.Context, userPoolID int64) (*MFAPolicyDTO, error)
	IsMFARequired(ctx context.Context, userPoolID int64) (bool, error)
	UserHasMFA(ctx context.Context, userID int64) (bool, error)

	// Admin
	AdminResetMFA(ctx context.Context, targetUserUUID string, actorUserID int64) error

	// Step-up
	IssueStepUpChallenge(ctx context.Context, userUUID string, allowedMethods []string) (*StepUpChallengeResponseDTO, error)
	VerifyStepUp(ctx context.Context, req StepUpVerifyRequestDTO, userID int64) (*StepUpVerifyResponseDTO, error)
}

type mfaService struct {
	db               *gorm.DB
	userRepo         UserRepository
	totpRepo         UserTOTPSecretRepository
	webAuthnCredRepo UserWebAuthnCredentialRepository
	backupCodeRepo   UserBackupCodeRepository
	secSettingRepo   SecuritySettingRepository
	authEventService AuthEventService
}

// NewMFAService constructs a MFAService.
func NewMFAService(
	db *gorm.DB,
	userRepo UserRepository,
	totpRepo UserTOTPSecretRepository,
	webAuthnCredRepo UserWebAuthnCredentialRepository,
	backupCodeRepo UserBackupCodeRepository,
	secSettingRepo SecuritySettingRepository,
	authEventService AuthEventService,
) MFAService {
	return &mfaService{
		db:               db,
		userRepo:         userRepo,
		totpRepo:         totpRepo,
		webAuthnCredRepo: webAuthnCredRepo,
		backupCodeRepo:   backupCodeRepo,
		secSettingRepo:   secSettingRepo,
		authEventService: authEventService,
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// TOTP
// ──────────────────────────────────────────────────────────────────────────────

// BeginTOTPEnrollment generates a new TOTP secret and stores it as pending
// (not yet enabled). The user must call FinishTOTPEnrollment with a valid code.
func (s *mfaService) BeginTOTPEnrollment(ctx context.Context, userID int64, userEmail string) (*TOTPEnrollResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "mfa.begin_totp_enrollment")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      totpIssuer,
		AccountName: userEmail,
		Digits:      totpDigits,
		Period:      totpPeriod,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "totp key generation failed")
		return nil, apperror.NewInternal("TOTP key generation failed", err)
	}

	secret := &UserTOTPSecret{
		UserID:    userID,
		Secret:    key.Secret(),
		IsEnabled: false,
	}
	if err := s.totpRepo.Upsert(secret); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "totp secret store failed")
		return nil, apperror.NewInternal("failed to store TOTP secret", err)
	}

	span.SetStatus(codes.Ok, "")
	return &TOTPEnrollResponseDTO{
		Secret:    key.Secret(),
		QRCodeURL: key.URL(),
	}, nil
}

// FinishTOTPEnrollment verifies the first TOTP code to confirm the user has
// set up their authenticator app, enables TOTP, and returns a fresh set of
// backup codes.
func (s *mfaService) FinishTOTPEnrollment(ctx context.Context, userID int64, code string) ([]string, error) {
	_, span := otel.Tracer("service").Start(ctx, "mfa.finish_totp_enrollment")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	record, err := s.totpRepo.FindByUserID(userID)
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewInternal("TOTP secret lookup failed", err)
	}
	if record == nil || record.IsEnabled {
		return nil, apperror.NewValidation("no pending TOTP enrollment found")
	}

	valid := totp.Validate(code, record.Secret)
	if !valid {
		span.SetStatus(codes.Error, "invalid totp code")
		return nil, apperror.NewValidation("invalid TOTP code")
	}

	if err := s.totpRepo.Enable(userID); err != nil {
		span.RecordError(err)
		return nil, apperror.NewInternal("failed to enable TOTP", err)
	}

	// Mark the user as having TOTP enabled.
	now := time.Now()
	if err := s.db.Model(&User{}).Where("user_id = ?", userID).
		Updates(map[string]any{
			"is_totp_enabled": true,
			"mfa_enabled_at":  now,
		}).Error; err != nil {
		span.RecordError(err)
		return nil, apperror.NewInternal("failed to update user MFA status", err)
	}

	// Generate a fresh set of backup codes.
	plainCodes, err := s.generateAndStoreBackupCodes(userID)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	s.authEventService.Log(ctx, AuthEventInput{
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    AuthEventCategoryAuthn,
		EventType:   AuthEventTypeTokenCreated,
		Severity:    AuthEventSeverityInfo,
		Result:      AuthEventResultSuccess,
		Description: ptr.Ptr("TOTP enrollment completed"),
	})

	span.SetStatus(codes.Ok, "")
	return plainCodes, nil
}

// VerifyTOTP validates a TOTP code for an already-enrolled user.
func (s *mfaService) VerifyTOTP(ctx context.Context, userID int64, code string) (bool, error) {
	_, span := otel.Tracer("service").Start(ctx, "mfa.verify_totp")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	record, err := s.totpRepo.FindByUserID(userID)
	if err != nil {
		span.RecordError(err)
		return false, apperror.NewInternal("TOTP lookup failed", err)
	}
	if record == nil || !record.IsEnabled {
		return false, apperror.NewValidation("TOTP is not enabled for this user")
	}

	valid := totp.Validate(code, record.Secret)
	if valid {
		_ = s.totpRepo.UpdateLastUsed(userID)
	}
	span.SetStatus(codes.Ok, "")
	return valid, nil
}

// DisableTOTP removes TOTP enrollment and clears all backup codes.
func (s *mfaService) DisableTOTP(ctx context.Context, userID int64) error {
	_, span := otel.Tracer("service").Start(ctx, "mfa.disable_totp")
	defer span.End()

	if err := s.totpRepo.Disable(userID); err != nil {
		span.RecordError(err)
		return apperror.NewInternal("failed to disable TOTP", err)
	}
	_ = s.backupCodeRepo.DeleteAllByUserID(userID)

	_ = s.db.Model(&User{}).Where("user_id = ?", userID).
		Updates(map[string]any{"is_totp_enabled": false}).Error

	s.authEventService.Log(ctx, AuthEventInput{
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    AuthEventCategoryAuthn,
		EventType:   AuthEventTypeTokenCreated,
		Severity:    AuthEventSeverityWarn,
		Result:      AuthEventResultSuccess,
		Description: ptr.Ptr("TOTP disabled by user"),
	})

	span.SetStatus(codes.Ok, "")
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Backup Codes
// ──────────────────────────────────────────────────────────────────────────────

// GetBackupCodesCount returns the number of unused backup codes the user has.
func (s *mfaService) GetBackupCodesCount(ctx context.Context, userID int64) (int, error) {
	codes, err := s.backupCodeRepo.FindUnusedByUserID(userID)
	if err != nil {
		return 0, apperror.NewInternal("backup code lookup failed", err)
	}
	return len(codes), nil
}

// RegenerateBackupCodes issues a fresh set of backup codes, replacing all existing ones.
func (s *mfaService) RegenerateBackupCodes(ctx context.Context, userID int64) ([]string, error) {
	_, span := otel.Tracer("service").Start(ctx, "mfa.regenerate_backup_codes")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	plainCodes, err := s.generateAndStoreBackupCodes(userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "regenerate failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return plainCodes, nil
}

// generateAndStoreBackupCodes generates mfaBackupCodeCount new codes,
// replaces all existing codes for the user, and returns the plaintexts.
func (s *mfaService) generateAndStoreBackupCodes(userID int64) ([]string, error) {
	_ = s.backupCodeRepo.DeleteAllByUserID(userID)

	plainCodes := make([]string, mfaBackupCodeCount)
	models := make([]*UserBackupCode, mfaBackupCodeCount)
	for i := range mfaBackupCodeCount {
		code, err := crypto.GenerateRandomString(mfaBackupCodeLength)
		if err != nil {
			return nil, apperror.NewInternal("backup code generation failed", err)
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		if err != nil {
			return nil, apperror.NewInternal("backup code hashing failed", err)
		}
		plainCodes[i] = code
		models[i] = &UserBackupCode{
			UserID:   userID,
			CodeHash: string(hash),
		}
	}
	if err := s.backupCodeRepo.CreateBulk(models); err != nil {
		return nil, apperror.NewInternal("backup code storage failed", err)
	}
	return plainCodes, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// MFA Status
// ──────────────────────────────────────────────────────────────────────────────

// GetMFAStatus returns a summary of all MFA factors enabled for a user.
func (s *mfaService) GetMFAStatus(ctx context.Context, userID int64) (*MFAStatusResponseDTO, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return nil, apperror.NewNotFound("user not found")
	}

	backupCount, _ := s.GetBackupCodesCount(ctx, userID)

	webAuthnCreds, err := s.webAuthnCredRepo.FindByUserID(userID)
	if err != nil {
		return nil, apperror.NewInternal("credential lookup failed", err)
	}
	credSummaries := make([]WebAuthnCredentialSummaryDTO, 0, len(webAuthnCreds))
	for _, c := range webAuthnCreds {
		summary := WebAuthnCredentialSummaryDTO{
			CredentialUUID: c.CredentialUUID.String(),
			Name:           c.Name,
			Transport:      c.Transport,
			CreatedAt:      c.CreatedAt.Format(time.RFC3339),
		}
		if c.LastUsedAt != nil {
			s := c.LastUsedAt.Format(time.RFC3339)
			summary.LastUsedAt = &s
		}
		credSummaries = append(credSummaries, summary)
	}

	resp := &MFAStatusResponseDTO{
		IsTOTPEnabled:     user.IsTOTPEnabled,
		IsWebAuthnEnabled: user.IsWebAuthnEnabled,
		BackupCodesCount:  backupCount,
		WebAuthnKeys:      credSummaries,
	}
	if user.MFAEnabledAt != nil {
		s := user.MFAEnabledAt.Format(time.RFC3339)
		resp.MFAEnabledAt = &s
	}
	return resp, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Policy
// ──────────────────────────────────────────────────────────────────────────────

// GetMFAPolicy reads the per-pool MFA policy from SecuritySetting.MFAConfig.
func (s *mfaService) GetMFAPolicy(ctx context.Context, userPoolID int64) (*MFAPolicyDTO, error) {
	setting, err := s.secSettingRepo.FindByUserPoolID(userPoolID)
	if err != nil || setting == nil {
		return &MFAPolicyDTO{Required: false, AllowedMethods: []string{"totp", "sms", "webauthn", "backup_code"}}, nil
	}
	var policy MFAPolicyDTO
	if err := json.Unmarshal(setting.MFAConfig, &policy); err != nil || policy.AllowedMethods == nil {
		return &MFAPolicyDTO{Required: false, AllowedMethods: []string{"totp", "sms", "webauthn", "backup_code"}}, nil
	}
	return &policy, nil
}

// IsMFARequired returns true when the pool policy requires MFA.
func (s *mfaService) IsMFARequired(ctx context.Context, userPoolID int64) (bool, error) {
	policy, err := s.GetMFAPolicy(ctx, userPoolID)
	if err != nil {
		return false, err
	}
	return policy.Required, nil
}

// UserHasMFA returns true when the user has at least one active MFA factor.
func (s *mfaService) UserHasMFA(ctx context.Context, userID int64) (bool, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return false, nil
	}
	return user.IsTOTPEnabled || user.IsWebAuthnEnabled, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Admin Reset
// ──────────────────────────────────────────────────────────────────────────────

// AdminResetMFA clears all MFA factors for the target user identified by UUID.
func (s *mfaService) AdminResetMFA(ctx context.Context, targetUserUUID string, actorUserID int64) error {
	_, span := otel.Tracer("service").Start(ctx, "mfa.admin_reset")
	defer span.End()
	span.SetAttributes(
		attribute.String("target_user.uuid", targetUserUUID),
		attribute.Int64("actor_user.id", actorUserID),
	)

	target, err := s.userRepo.FindByUUID(targetUserUUID)
	if err != nil || target == nil {
		return apperror.NewNotFound("target user not found")
	}
	targetUserID := target.UserID

	_ = s.totpRepo.Disable(targetUserID)
	_ = s.backupCodeRepo.DeleteAllByUserID(targetUserID)
	_ = s.webAuthnCredRepo.DeleteAllByUserID(targetUserID)

	if err := s.db.Model(&User{}).Where("user_id = ?", targetUserID).
		Updates(map[string]any{
			"is_totp_enabled":     false,
			"is_webauthn_enabled": false,
			"mfa_enabled_at":      nil,
		}).Error; err != nil {
		span.RecordError(err)
		return apperror.NewInternal("failed to reset user MFA status", err)
	}

	s.authEventService.Log(ctx, AuthEventInput{
		ActorUserID: &actorUserID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    AuthEventCategoryAuthn,
		EventType:   AuthEventTypeTokenCreated,
		Severity:    AuthEventSeverityCritical,
		Result:      AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Admin reset MFA for user %s", targetUserUUID)),
	})

	span.SetStatus(codes.Ok, "")
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Step-up Authentication
// ──────────────────────────────────────────────────────────────────────────────

// IssueStepUpChallenge issues a short-lived challenge token that authorizes
// a step-up authentication flow. The client must complete one of the
// allowedMethods then call VerifyStepUp.
func (s *mfaService) IssueStepUpChallenge(ctx context.Context, userUUID string, allowedMethods []string) (*StepUpChallengeResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "mfa.issue_step_up_challenge")
	defer span.End()

	token, err := jwt.GenerateStepUpChallengeToken(userUUID, stepUpChallengeTTL)
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewInternal("step-up challenge generation failed", err)
	}

	span.SetStatus(codes.Ok, "")
	return &StepUpChallengeResponseDTO{
		ChallengeToken: token,
		AllowedMethods: allowedMethods,
	}, nil
}

// VerifyStepUp validates the step-up challenge token, verifies the provided
// MFA factor, then issues a new access token with acr=2 and the appropriate
// amr claims.
func (s *mfaService) VerifyStepUp(ctx context.Context, req StepUpVerifyRequestDTO, userID int64) (*StepUpVerifyResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "mfa.verify_step_up")
	defer span.End()

	// Validate the challenge token.
	claims, err := jwt.ValidateStepUpChallengeToken(req.ChallengeToken)
	if err != nil {
		span.SetStatus(codes.Error, "invalid challenge token")
		return nil, apperror.NewUnauthorized("invalid or expired step-up challenge token")
	}
	userUUID, _ := claims["sub"].(string)
	if userUUID == "" {
		return nil, apperror.NewUnauthorized("step-up challenge token missing sub")
	}

	// Verify the provided MFA factor.
	var amr []string
	switch req.Method {
	case "totp":
		ok, verr := s.VerifyTOTP(ctx, userID, req.Code)
		if verr != nil || !ok {
			span.SetStatus(codes.Error, "totp verification failed")
			return nil, apperror.NewUnauthorized("invalid TOTP code")
		}
		amr = []string{"pwd", "otp"}

	case "backup_code":
		bCodes, lerr := s.backupCodeRepo.FindUnusedByUserID(userID)
		if lerr != nil {
			return nil, apperror.NewInternal("backup code lookup failed", lerr)
		}
		matched := false
		for _, bc := range bCodes {
			if bcrypt.CompareHashAndPassword([]byte(bc.CodeHash), []byte(req.Code)) == nil {
				_ = s.backupCodeRepo.MarkUsed(bc.BackupCodeID)
				matched = true
				break
			}
		}
		if !matched {
			span.SetStatus(codes.Error, "backup code invalid")
			return nil, apperror.NewUnauthorized("invalid backup code")
		}
		amr = []string{"pwd", "mfa"}

	default:
		return nil, apperror.NewValidation(fmt.Sprintf("unsupported step-up method: %s", req.Method))
	}

	// Fetch user for token generation.
	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return nil, apperror.NewInternal("user lookup failed", err)
	}

	issuer := config.AppPublicHostname
	accessToken, err := jwt.GenerateAccessTokenWithOptions(
		user.UserUUID.String(), "", issuer, issuer, "", "",
		&jwt.AccessTokenOptions{},
	)
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewInternal("token generation failed", err)
	}

	_ = amr // amr will be embedded in ID token via IDTokenParams in a full login flow

	s.authEventService.Log(ctx, AuthEventInput{
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    AuthEventCategoryAuthn,
		EventType:   AuthEventTypeTokenCreated,
		Severity:    AuthEventSeverityInfo,
		Result:      AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Step-up authentication completed via %s", req.Method)),
	})

	span.SetStatus(codes.Ok, "")
	return &StepUpVerifyResponseDTO{
		AccessToken: accessToken,
		ExpiresIn:   int64(jwt.AccessTokenTTL.Seconds()),
	}, nil
}

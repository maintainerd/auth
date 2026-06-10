package mfa

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/notifier"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/platform/sms"
	"github.com/maintainerd/auth/internal/secpolicy"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/hotp"
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
	smsStepUpOTPLength  = 6
	smsStepUpTTL        = 10 * time.Minute
	smsStepUpMaxFailed  = 3
)

var (
	generateBackupCodeString     = crypto.GenerateRandomString
	generateTOTPKey              = totp.Generate
	hashBackupCodePassword       = bcrypt.GenerateFromPassword
	generateStepUpChallengeToken = jwt.GenerateStepUpChallengeTokenWithContext
	validateStepUpChallengeToken = jwt.ValidateStepUpChallengeToken
	generateStepUpAccessToken    = jwt.GenerateAccessTokenWithOptionsContext
	checkMFARateLimit            = security.CheckRateLimit
)

// MFAService handles TOTP enrollment/verification, backup code management,
// per-tenant MFA policy, admin resets, and step-up authentication.
type MFAService interface {
	// TOTP enrollment
	BeginTOTPEnrollment(ctx context.Context, userID int64) (*TOTPEnrollResponseDTO, error)
	FinishTOTPEnrollment(ctx context.Context, userID int64, code string) ([]string, error)
	VerifyTOTP(ctx context.Context, userID int64, code string) (bool, error)
	DisableTOTP(ctx context.Context, userID int64) error

	// Backup codes
	GetBackupCodesCount(ctx context.Context, userID int64) (int, error)
	RegenerateBackupCodes(ctx context.Context, userID int64) ([]string, error)

	// Status
	GetMFAStatus(ctx context.Context, userID int64) (*MFAStatusResponseDTO, error)

	// Policy
	GetMFAPolicy(ctx context.Context, tenantID int64) (*MFAPolicyDTO, error)
	IsMFARequired(ctx context.Context, tenantID int64) (bool, error)
	UserHasMFA(ctx context.Context, userID int64) (bool, error)

	// SyncMFAState clears recovery state (backup codes, mfa_enabled_at) when no
	// primary MFA factor remains. Call after removing a factor that the service
	// does not own (e.g. a WebAuthn credential deleted via WebAuthnService).
	SyncMFAState(ctx context.Context, userID int64) error

	// EnrolledMFAMethods returns the user's usable MFA methods in canonical order
	// (totp, webauthn, sms, backup_code). It is the single source of truth for
	// "which factors does this user have", used by the authn login MFA step.
	EnrolledMFAMethods(ctx context.Context, userID int64) ([]string, error)

	// Admin
	AdminResetMFA(ctx context.Context, targetUserUUID string, actorUserID int64) error
	AdminResetMFAMethod(ctx context.Context, targetUserUUID, method string, actorUserID int64) error

	// Self-service — reset all of the caller's own MFA factors. Scoped to the
	// authenticated user (no target param), so a user can only reset their own.
	SelfResetMFA(ctx context.Context, userID int64) error

	// Step-up
	IssueStepUpChallenge(ctx context.Context, userUUID string, allowedMethods []string) (*StepUpChallengeResponseDTO, error)
	VerifyStepUp(ctx context.Context, req StepUpVerifyRequestDTO, userID int64) (*StepUpVerifyResponseDTO, error)
	SendStepUpSMS(ctx context.Context, userID int64) error

	// Login MFA (second step after password) — shared factor verification used
	// by the authn package to elevate a freshly issued session to acr=2.
	VerifyFactor(ctx context.Context, userID int64, method, code string, assertion []byte) ([]string, error)
	SendSMSChallenge(ctx context.Context, userID int64) error
	BeginWebAuthnLogin(ctx context.Context, userID int64) (json.RawMessage, error)

	// SMS MFA enrollment
	EnrollSMS(ctx context.Context, userID int64, phone string) error
	VerifySMS(ctx context.Context, userID int64, phone, code string) error
	DisableSMS(ctx context.Context, userID int64) error
}

type mfaService struct {
	db               *gorm.DB
	userRepo         UserRepository
	totpRepo         UserTOTPSecretRepository
	webAuthnCredRepo UserWebAuthnCredentialRepository
	webAuthnSvc      WebAuthnService
	backupCodeRepo   UserBackupCodeRepository
	smsPhoneRepo     UserSMSPhoneRepository
	smsOtpRepo       notifier.UserOTPRepository
	secSettingRepo   secpolicy.SecuritySettingRepository
	authEventService authevent.AuthEventService
}

// NewMFAService constructs a MFAService.
func NewMFAService(
	db *gorm.DB,
	userRepo UserRepository,
	totpRepo UserTOTPSecretRepository,
	webAuthnCredRepo UserWebAuthnCredentialRepository,
	webAuthnSvc WebAuthnService,
	backupCodeRepo UserBackupCodeRepository,
	smsPhoneRepo UserSMSPhoneRepository,
	smsOtpRepo notifier.UserOTPRepository,
	secSettingRepo secpolicy.SecuritySettingRepository,
	authEventService authevent.AuthEventService,
) MFAService {
	return &mfaService{
		db:               db,
		userRepo:         userRepo,
		totpRepo:         totpRepo,
		webAuthnCredRepo: webAuthnCredRepo,
		webAuthnSvc:      webAuthnSvc,
		backupCodeRepo:   backupCodeRepo,
		smsPhoneRepo:     smsPhoneRepo,
		smsOtpRepo:       smsOtpRepo,
		secSettingRepo:   secSettingRepo,
		authEventService: authEventService,
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// TOTP
// ──────────────────────────────────────────────────────────────────────────────

// BeginTOTPEnrollment generates a new TOTP secret and stores it as pending
// (not yet enabled). The user must call FinishTOTPEnrollment with a valid code.
func (s *mfaService) BeginTOTPEnrollment(ctx context.Context, userID int64) (*TOTPEnrollResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "mfa.begin_totp_enrollment")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return nil, apperror.NewNotFound("user not found")
	}

	key, err := generateTOTPKey(totp.GenerateOpts{
		Issuer:      totpIssuer,
		AccountName: user.Email,
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
	enc, encErr := crypto.EncryptAtRest(secret.Secret)
	if encErr != nil {
		span.RecordError(encErr)
		span.SetStatus(codes.Error, "totp secret encryption failed")
		return nil, apperror.NewInternal("failed to encrypt TOTP secret", encErr)
	}
	secret.Secret = enc
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

	dec := crypto.SafeDecryptAtRest(record.Secret)
	valid := totp.Validate(code, dec)
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

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeTokenCreated,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
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

	rateLimitKey := security.RateLimitKey(fmt.Sprintf("totp:%d", userID), "verify")
	if err := checkMFARateLimit(rateLimitKey); err != nil {
		span.SetStatus(codes.Error, "totp rate limited")
		return false, apperror.NewUnauthorized("too many attempts; try again later")
	}

	dec := crypto.SafeDecryptAtRest(record.Secret)
	step, valid, validationErr := validateTOTPAndStep(code, dec, time.Now())
	if validationErr != nil {
		span.RecordError(validationErr)
		return false, apperror.NewValidation("invalid TOTP code")
	}
	if valid {
		accepted, err := s.totpRepo.MarkStepUsed(userID, step)
		if err != nil {
			span.RecordError(err)
			return false, apperror.NewInternal("failed to update TOTP last-used step", err)
		}
		if !accepted {
			security.RecordFailedAttempt(rateLimitKey)
			span.SetStatus(codes.Error, "totp replay detected")
			return false, nil
		}
		security.ResetFailedAttempts(rateLimitKey)
	} else {
		security.RecordFailedAttempt(rateLimitKey)
	}
	span.SetStatus(codes.Ok, "")
	return valid, nil
}

func validateTOTPAndStep(passcode, secret string, at time.Time) (int64, bool, error) {
	counter := int64(math.Floor(float64(at.UTC().Unix()) / float64(totpPeriod)))
	candidates := []int64{counter}
	for i := int64(1); i <= 1; i++ {
		candidates = append(candidates, counter+i, counter-i)
	}
	for _, candidate := range candidates {
		if candidate < 0 {
			continue
		}
		ok, err := hotp.ValidateCustom(passcode, uint64(candidate), secret, hotp.ValidateOpts{
			Digits:    totpDigits,
			Algorithm: otp.AlgorithmSHA1,
		})
		if err != nil {
			return 0, false, err
		}
		if ok {
			return candidate, true, nil
		}
	}
	return 0, false, nil
}

// DisableTOTP removes TOTP enrollment and clears all backup codes.
func (s *mfaService) DisableTOTP(ctx context.Context, userID int64) error {
	_, span := otel.Tracer("service").Start(ctx, "mfa.disable_totp")
	defer span.End()

	if err := s.totpRepo.Disable(userID); err != nil {
		span.RecordError(err)
		return apperror.NewInternal("failed to disable TOTP", err)
	}
	if err := s.backupCodeRepo.DeleteAllByUserID(userID); err != nil {
		span.RecordError(err)
		return apperror.NewInternal("failed to delete backup codes", err)
	}

	if err := s.db.Model(&User{}).Where("user_id = ?", userID).
		Updates(map[string]any{"is_totp_enabled": false}).Error; err != nil {
		span.RecordError(err)
		return apperror.NewInternal("failed to update user TOTP state", err)
	}

	// Clear mfa_enabled_at (and any residual recovery state) if this was the
	// last primary factor.
	if err := s.SyncMFAState(ctx, userID); err != nil {
		span.RecordError(err)
		return err
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeTokenCreated,
		Severity:    authevent.AuthEventSeverityWarn,
		Result:      authevent.AuthEventResultSuccess,
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
	if err := s.backupCodeRepo.DeleteAllByUserID(userID); err != nil {
		return nil, apperror.NewInternal("failed to delete existing backup codes", err)
	}

	plainCodes := make([]string, mfaBackupCodeCount)
	models := make([]*UserBackupCode, mfaBackupCodeCount)
	for i := range mfaBackupCodeCount {
		code, err := generateBackupCodeString(mfaBackupCodeLength)
		if err != nil {
			return nil, apperror.NewInternal("backup code generation failed", err)
		}
		hash, err := hashBackupCodePassword([]byte(code), bcrypt.DefaultCost)
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
		IsSMSEnabled:      s.isSMSEnabled(ctx, userID),
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

// GetMFAPolicy reads the per-pool MFA policy from secpolicy.SecuritySetting.MFAConfig.
func (s *mfaService) GetMFAPolicy(ctx context.Context, tenantID int64) (*MFAPolicyDTO, error) {
	setting, err := s.secSettingRepo.FindByTenantID(tenantID)
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
func (s *mfaService) IsMFARequired(ctx context.Context, tenantID int64) (bool, error) {
	policy, _ := s.GetMFAPolicy(ctx, tenantID)
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

	if err := s.totpRepo.Disable(targetUserID); err != nil {
		span.RecordError(err)
		return apperror.NewInternal("failed to disable target TOTP", err)
	}
	if err := s.backupCodeRepo.DeleteAllByUserID(targetUserID); err != nil {
		span.RecordError(err)
		return apperror.NewInternal("failed to delete target backup codes", err)
	}
	if err := s.webAuthnCredRepo.DeleteAllByUserID(targetUserID); err != nil {
		span.RecordError(err)
		return apperror.NewInternal("failed to delete target WebAuthn credentials", err)
	}
	if err := s.smsPhoneRepo.DeleteByUserID(targetUserID); err != nil {
		span.RecordError(err)
		return apperror.NewInternal("failed to delete target SMS phone", err)
	}

	if err := s.db.Model(&User{}).Where("user_id = ?", targetUserID).
		Updates(map[string]any{
			"is_totp_enabled":     false,
			"is_webauthn_enabled": false,
			"mfa_enabled_at":      nil,
		}).Error; err != nil {
		span.RecordError(err)
		return apperror.NewInternal("failed to reset user MFA status", err)
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		ActorUserID: &actorUserID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeTokenCreated,
		Severity:    authevent.AuthEventSeverityCritical,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Admin reset MFA for user %s", targetUserUUID)),
	})

	span.SetStatus(codes.Ok, "")
	return nil
}

// Canonical MFA method identifiers accepted by AdminResetMFAMethod. These mirror
// the values returned by EnrolledMFAMethods.
const (
	mfaMethodTOTP       = "totp"
	mfaMethodWebAuthn   = "webauthn"
	mfaMethodSMS        = "sms"
	mfaMethodBackupCode = "backup_code"
)

// AdminResetMFAMethod removes a single MFA factor for a target user (admin only).
// method is one of "totp", "webauthn", "sms", or "backup_code". Unlike
// AdminResetMFA (which clears every factor), this lets an admin reset just the
// factor a user lost access to — e.g. wiping TOTP/SMS for a lost phone while
// leaving a registered passkey intact. After removing the factor it reconciles
// recovery state so the account is left clean when no primary factor remains.
func (s *mfaService) AdminResetMFAMethod(ctx context.Context, targetUserUUID, method string, actorUserID int64) error {
	_, span := otel.Tracer("service").Start(ctx, "mfa.admin_reset_method")
	defer span.End()
	span.SetAttributes(
		attribute.String("target_user.uuid", targetUserUUID),
		attribute.String("mfa.method", method),
		attribute.Int64("actor_user.id", actorUserID),
	)

	target, err := s.userRepo.FindByUUID(targetUserUUID)
	if err != nil || target == nil {
		return apperror.NewNotFound("target user not found")
	}
	targetUserID := target.UserID

	switch method {
	case mfaMethodTOTP:
		if err := s.totpRepo.Disable(targetUserID); err != nil {
			span.RecordError(err)
			return apperror.NewInternal("failed to disable target TOTP", err)
		}
		if err := s.db.Model(&User{}).Where("user_id = ?", targetUserID).
			Update("is_totp_enabled", false).Error; err != nil {
			span.RecordError(err)
			return apperror.NewInternal("failed to update target TOTP state", err)
		}
	case mfaMethodWebAuthn:
		if err := s.webAuthnCredRepo.DeleteAllByUserID(targetUserID); err != nil {
			span.RecordError(err)
			return apperror.NewInternal("failed to delete target WebAuthn credentials", err)
		}
		if err := s.db.Model(&User{}).Where("user_id = ?", targetUserID).
			Update("is_webauthn_enabled", false).Error; err != nil {
			span.RecordError(err)
			return apperror.NewInternal("failed to update target WebAuthn state", err)
		}
	case mfaMethodSMS:
		if err := s.smsPhoneRepo.DeleteByUserID(targetUserID); err != nil {
			span.RecordError(err)
			return apperror.NewInternal("failed to delete target SMS phone", err)
		}
	case mfaMethodBackupCode:
		if err := s.backupCodeRepo.DeleteAllByUserID(targetUserID); err != nil {
			span.RecordError(err)
			return apperror.NewInternal("failed to delete target backup codes", err)
		}
	default:
		return apperror.NewValidation("unsupported MFA method")
	}

	// Reconcile recovery/flag state — clears leftover backup codes and
	// mfa_enabled_at if this removal left the user with no primary factor.
	if err := s.SyncMFAState(ctx, targetUserID); err != nil {
		span.RecordError(err)
		return err
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		ActorUserID: &actorUserID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeTokenCreated,
		Severity:    authevent.AuthEventSeverityCritical,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Admin reset %s MFA for user %s", method, targetUserUUID)),
	})

	span.SetStatus(codes.Ok, "")
	return nil
}

// SelfResetMFA clears every MFA factor for the authenticated user. It is the
// self-service counterpart to AdminResetMFA: the caller's own user ID is the only
// account it touches (the handler derives userID from the session), so a user can
// never reset another account's MFA.
func (s *mfaService) SelfResetMFA(ctx context.Context, userID int64) error {
	_, span := otel.Tracer("service").Start(ctx, "mfa.self_reset")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	if err := s.totpRepo.Disable(userID); err != nil {
		span.RecordError(err)
		return apperror.NewInternal("failed to disable TOTP", err)
	}
	if err := s.backupCodeRepo.DeleteAllByUserID(userID); err != nil {
		span.RecordError(err)
		return apperror.NewInternal("failed to delete backup codes", err)
	}
	if err := s.webAuthnCredRepo.DeleteAllByUserID(userID); err != nil {
		span.RecordError(err)
		return apperror.NewInternal("failed to delete WebAuthn credentials", err)
	}
	if err := s.smsPhoneRepo.DeleteByUserID(userID); err != nil {
		span.RecordError(err)
		return apperror.NewInternal("failed to delete SMS phone", err)
	}

	if err := s.db.Model(&User{}).Where("user_id = ?", userID).
		Updates(map[string]any{
			"is_totp_enabled":     false,
			"is_webauthn_enabled": false,
			"mfa_enabled_at":      nil,
		}).Error; err != nil {
		span.RecordError(err)
		return apperror.NewInternal("failed to reset MFA status", err)
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeTokenCreated,
		Severity:    authevent.AuthEventSeverityWarn,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr("User reset their own MFA"),
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

	token, err := generateStepUpChallengeToken(ctx, userUUID, stepUpChallengeTTL, allowedMethods)
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

func (s *mfaService) SendStepUpSMS(ctx context.Context, userID int64) error {
	_, span := otel.Tracer("service").Start(ctx, "mfa.send_step_up_sms")
	defer span.End()

	phoneRecord, err := s.smsPhoneRepo.FindByUserID(userID)
	if err != nil {
		return apperror.NewInternal("failed to look up MFA phone", err)
	}
	if phoneRecord == nil || !phoneRecord.IsVerified {
		return apperror.NewValidation("no verified MFA phone on file")
	}

	if err := security.CheckRateLimit("mfa-sms-step-up:" + phoneRecord.Phone); err != nil {
		return err
	}

	otpCode, err := crypto.GenerateOTP(smsStepUpOTPLength)
	if err != nil {
		return apperror.NewInternal("failed to generate SMS OTP", err)
	}
	otpHash := crypto.HashAuthorizationCode(otpCode)

	s.db.Where("user_id = ?", userID).Delete(&notifier.UserOTP{})

	record := &notifier.UserOTP{
		UserID:    userID,
		Channel:   "sms",
		Recipient: phoneRecord.Phone,
		OTPHash:   otpHash,
		ExpiresAt: time.Now().Add(smsStepUpTTL),
	}
	if _, err := s.smsOtpRepo.Create(record); err != nil {
		return apperror.NewInternal("failed to store SMS OTP", err)
	}

	tenantID := mfaUserTenantID(ctx, s.db, userID)
	provider, smsErr := sms.NewProviderFromDB(ctx, s.db, tenantID)
	if smsErr != nil {
		slog.Warn("SMS provider init failed — logging OTP for dev", "err", smsErr, "phone", phoneRecord.Phone, "otp", otpCode)
	} else if provider != nil {
		if sendErr := provider.Send(ctx, phoneRecord.Phone, fmt.Sprintf("Your step-up code is: %s", otpCode)); sendErr != nil {
			slog.Error("SMS step-up send failed", "err", sendErr)
		}
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (s *mfaService) EnrollSMS(ctx context.Context, userID int64, phone string) error {
	_, span := otel.Tracer("service").Start(ctx, "mfa.enroll_sms")
	defer span.End()

	if phone == "" {
		return apperror.NewValidation("phone is required")
	}

	record, err := s.smsPhoneRepo.FindByUserID(userID)
	if err != nil {
		return apperror.NewInternal("failed to check existing MFA phone", err)
	}
	if record != nil && record.IsVerified {
		return apperror.NewConflict("SMS MFA is already enrolled — disable it first")
	}

	otpCode, err := crypto.GenerateOTP(smsStepUpOTPLength)
	if err != nil {
		return apperror.NewInternal("failed to generate SMS OTP", err)
	}
	otpHash := crypto.HashAuthorizationCode(otpCode)

	s.db.Where("user_id = ?", userID).Delete(&notifier.UserOTP{})
	otpRecord := &notifier.UserOTP{
		UserID:    userID,
		Channel:   "sms",
		Recipient: phone,
		OTPHash:   otpHash,
		ExpiresAt: time.Now().Add(smsStepUpTTL),
	}
	if _, err := s.smsOtpRepo.Create(otpRecord); err != nil {
		return apperror.NewInternal("failed to store SMS OTP", err)
	}

	tenantID := mfaUserTenantID(ctx, s.db, userID)
	provider, smsErr := sms.NewProviderFromDB(ctx, s.db, tenantID)
	if smsErr != nil {
		slog.Warn("SMS provider init failed — logging OTP for dev", "err", smsErr, "phone", phone, "otp", otpCode)
	} else if provider != nil {
		if sendErr := provider.Send(ctx, phone, fmt.Sprintf("Your MFA verification code is: %s", otpCode)); sendErr != nil {
			slog.Error("SMS enrollment send failed — logging OTP for dev", "err", sendErr, "phone", phone, "otp", otpCode)
		}
	} else {
		slog.Info("SMS OTP (no provider) — use for dev", "phone", phone, "otp", otpCode)
	}

	existing, _ := s.smsPhoneRepo.FindByUserID(userID)
	if existing != nil {
		existing.Phone = phone
		existing.IsVerified = false
		existing.VerifiedAt = nil
		if _, err := s.smsPhoneRepo.CreateOrUpdate(existing); err != nil {
			return apperror.NewInternal("failed to save MFA phone", err)
		}
	} else {
		if _, err := s.smsPhoneRepo.CreateOrUpdate(&UserSMSPhone{UserID: userID, Phone: phone}); err != nil {
			return apperror.NewInternal("failed to save MFA phone", err)
		}
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (s *mfaService) VerifySMS(ctx context.Context, userID int64, phone, code string) error {
	_, span := otel.Tracer("service").Start(ctx, "mfa.verify_sms")
	defer span.End()

	record, err := s.smsPhoneRepo.FindByUserID(userID)
	if err != nil {
		return apperror.NewInternal("failed to look up MFA phone", err)
	}
	if record == nil || record.Phone != phone {
		return apperror.NewValidation("no pending SMS enrollment for this phone")
	}

	otpRecord, lerr := s.smsOtpRepo.FindValid("sms", phone)
	if lerr != nil || otpRecord == nil {
		return apperror.NewUnauthorized("invalid or expired SMS code")
	}

	if subtle.ConstantTimeCompare([]byte(otpRecord.OTPHash), []byte(crypto.HashAuthorizationCode(code))) != 1 {
		_ = s.smsOtpRepo.RecordFailure(otpRecord.UserOTPID, smsStepUpMaxFailed)
		return apperror.NewUnauthorized("invalid SMS code")
	}
	if err := s.smsOtpRepo.MarkUsed(otpRecord.UserOTPID); err != nil {
		return apperror.NewInternal("failed to mark SMS OTP used", err)
	}

	now := time.Now()
	record.IsVerified = true
	record.VerifiedAt = &now
	if _, err := s.smsPhoneRepo.CreateOrUpdate(record); err != nil {
		return apperror.NewInternal("failed to verify MFA phone", err)
	}

	s.ensureMFAFlag(ctx, userID)

	span.SetStatus(codes.Ok, "")
	return nil
}

func (s *mfaService) DisableSMS(ctx context.Context, userID int64) error {
	if err := s.smsPhoneRepo.DeleteByUserID(userID); err != nil {
		return apperror.NewInternal("failed to disable SMS MFA", err)
	}
	// Clear recovery state if SMS was the last primary factor.
	return s.SyncMFAState(ctx, userID)
}

func (s *mfaService) ensureMFAFlag(ctx context.Context, userID int64) {
	now := time.Now()
	s.db.Model(&User{}).Where("user_id = ? AND mfa_enabled_at IS NULL", userID).
		Update("mfa_enabled_at", now)
}

// VerifyStepUp validates the step-up challenge token, verifies the provided
// MFA factor, then issues a new access token with acr=2 and the appropriate
// amr claims.
func (s *mfaService) VerifyStepUp(ctx context.Context, req StepUpVerifyRequestDTO, userID int64) (*StepUpVerifyResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "mfa.verify_step_up")
	defer span.End()

	// Validate the challenge token.
	claims, err := validateStepUpChallengeToken(req.ChallengeToken)
	if err != nil {
		span.SetStatus(codes.Error, "invalid challenge token")
		return nil, apperror.NewUnauthorized("invalid or expired step-up challenge token")
	}
	userUUID, _ := claims["sub"].(string)
	if userUUID == "" {
		return nil, apperror.NewUnauthorized("step-up challenge token missing sub")
	}

	tokenUser, err := s.userRepo.FindByUUID(userUUID)
	if err != nil || tokenUser == nil {
		return nil, apperror.NewUnauthorized("step-up challenge token references unknown user")
	}
	if tokenUser.UserID != userID {
		return nil, apperror.NewUnauthorized("step-up challenge token subject does not match authenticated user")
	}
	if !stepUpMethodAllowed(claims["allowed_methods"], req.Method) {
		return nil, apperror.NewValidation(fmt.Sprintf("step-up method not allowed: %s", req.Method))
	}

	// Verify the provided MFA factor.
	amr, err := s.verifyFactor(ctx, userID, req.Method, req.Code, req.Assertion)
	if err != nil {
		span.SetStatus(codes.Error, "factor verification failed")
		return nil, err
	}

	// Fetch user for token generation.
	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return nil, apperror.NewInternal("user lookup failed", err)
	}

	// Re-issue the elevated token as a faithful copy of the current session,
	// only bumping acr/amr. The subject MUST stay the original token's `sub`:
	// UserContextMiddleware resolves the caller via user_identities.sub, which
	// is the login subject — not necessarily the user UUID — so substituting the
	// UUID makes the retried request fail user lookup (401). client_id/provider_id
	// must also be carried (the generator rejects empty values).
	var sub, clientID, providerID, scope, sessionID string
	if claims := middleware.JWTClaimsFromContext(ctx); claims != nil {
		sub = claims.Sub
		clientID = claims.ClientID
		providerID = claims.ProviderID
		scope = claims.Scope
		sessionID = claims.SessionID
	}
	if sub == "" {
		sub = user.UserUUID.String()
	}

	issuer := config.AppPublicHostname
	accessToken, err := generateStepUpAccessToken(
		ctx,
		sub, scope, issuer, issuer, clientID, providerID,
		&jwt.AccessTokenOptions{
			AMR:       amr,
			ACR:       jwt.ACRLevel2,
			SessionID: sessionID,
		},
	)
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewInternal("token generation failed", err)
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeTokenCreated,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Step-up authentication completed via %s", req.Method)),
	})

	span.SetStatus(codes.Ok, "")
	return &StepUpVerifyResponseDTO{
		AccessToken: accessToken,
		ExpiresIn:   int64(jwt.AccessTokenTTL.Seconds()),
	}, nil
}

// verifyFactor verifies a single MFA factor (TOTP, backup code, SMS OTP, or a
// WebAuthn assertion) for userID and returns the amr values (per RFC 8176) to
// embed in the resulting token. It is shared by step-up elevation and the
// login MFA second step. `code` carries the typed proof; `assertion` carries
// the raw WebAuthn assertion JSON.
func (s *mfaService) verifyFactor(ctx context.Context, userID int64, method, code string, assertion []byte) ([]string, error) {
	switch method {
	case "totp":
		ok, verr := s.VerifyTOTP(ctx, userID, code)
		if verr != nil || !ok {
			return nil, apperror.NewUnauthorized("invalid TOTP code")
		}
		return []string{"pwd", "otp"}, nil

	case "backup_code":
		backupRateLimitKey := security.RateLimitKey(fmt.Sprintf("backup_code:%d", userID), "verify")
		if err := checkMFARateLimit(backupRateLimitKey); err != nil {
			return nil, apperror.NewUnauthorized("too many attempts; try again later")
		}
		bCodes, lerr := s.backupCodeRepo.FindUnusedByUserID(userID)
		if lerr != nil {
			return nil, apperror.NewInternal("backup code lookup failed", lerr)
		}
		for _, bc := range bCodes {
			if bcrypt.CompareHashAndPassword([]byte(bc.CodeHash), []byte(code)) == nil {
				if err := s.backupCodeRepo.MarkUsed(bc.BackupCodeID); err != nil {
					return nil, apperror.NewInternal("failed to mark backup code used", err)
				}
				return []string{"pwd", "mfa"}, nil
			}
		}
		security.RecordFailedAttempt(backupRateLimitKey)
		return nil, apperror.NewUnauthorized("invalid backup code")

	case "sms":
		smsRateLimitKey := security.RateLimitKey(fmt.Sprintf("sms_step_up:%d", userID), "verify")
		if err := checkMFARateLimit(smsRateLimitKey); err != nil {
			return nil, apperror.NewUnauthorized("too many attempts; try again later")
		}
		// SMS MFA uses the dedicated user_sms_phones record, NOT users.phone
		// (which is profile-only). The OTP was sent to and stored against this
		// MFA phone, so verification must look it up the same way.
		phone, perr := s.smsPhoneRepo.FindByUserID(userID)
		if perr != nil || phone == nil || !phone.IsVerified || phone.Phone == "" {
			return nil, apperror.NewUnauthorized("no verified phone on file")
		}
		record, lerr := s.smsOtpRepo.FindValid("sms", phone.Phone)
		if lerr != nil || record == nil {
			security.RecordFailedAttempt(smsRateLimitKey)
			return nil, apperror.NewUnauthorized("no valid SMS code found — request a new one")
		}
		if subtle.ConstantTimeCompare([]byte(record.OTPHash), []byte(crypto.HashAuthorizationCode(code))) != 1 {
			_ = s.smsOtpRepo.RecordFailure(record.UserOTPID, smsStepUpMaxFailed)
			security.RecordFailedAttempt(smsRateLimitKey)
			return nil, apperror.NewUnauthorized("invalid SMS code")
		}
		if err := s.smsOtpRepo.MarkUsed(record.UserOTPID); err != nil {
			return nil, apperror.NewInternal("failed to mark SMS OTP used", err)
		}
		return []string{"pwd", "sms"}, nil

	case "webauthn":
		if s.webAuthnSvc == nil {
			return nil, apperror.NewValidation("WebAuthn is not available")
		}
		if len(assertion) == 0 {
			return nil, apperror.NewValidation("WebAuthn assertion is required")
		}
		parsed, perr := parseWebAuthnRequestResponse(bytes.NewReader(assertion))
		if perr != nil {
			return nil, apperror.NewValidation("invalid WebAuthn assertion")
		}
		// FinishAuthentication validates the assertion against the session
		// created by the matching begin call, enforces sign-count regression
		// detection, and clears the ceremony session.
		cred, verr := s.webAuthnSvc.FinishAuthentication(ctx, userID, parsed)
		if verr != nil {
			return nil, apperror.NewUnauthorized("WebAuthn authentication failed")
		}
		// AMR per RFC 8176: password + user presence, plus a possession claim
		// reflecting the authenticator — a backup-eligible (synced) passkey is a
		// software key (swk); a device-bound credential is a hardware key (hwk).
		amr := []string{"pwd", "user"}
		if cred != nil && cred.IsBackupEligible {
			return append(amr, "swk"), nil
		}
		return append(amr, "hwk"), nil

	default:
		return nil, apperror.NewValidation(fmt.Sprintf("unsupported MFA method: %s", method))
	}
}

// VerifyFactor verifies a login second factor for userID and returns its amr
// values. Used by the authn login MFA step (after password) to elevate the
// freshly issued session to acr=2. The caller is responsible for having
// established that userID owns the in-flight login challenge.
func (s *mfaService) VerifyFactor(ctx context.Context, userID int64, method, code string, assertion []byte) ([]string, error) {
	return s.verifyFactor(ctx, userID, method, code, assertion)
}

// SendSMSChallenge sends an SMS OTP to userID's verified phone for the login
// MFA step. It reuses the same OTP store/rate limiting as step-up SMS.
func (s *mfaService) SendSMSChallenge(ctx context.Context, userID int64) error {
	return s.SendStepUpSMS(ctx, userID)
}

// BeginWebAuthnLogin starts a passkey assertion ceremony for userID and returns
// the assertion options as JSON for the browser. Used by the login MFA step.
func (s *mfaService) BeginWebAuthnLogin(ctx context.Context, userID int64) (json.RawMessage, error) {
	if s.webAuthnSvc == nil {
		return nil, apperror.NewValidation("WebAuthn is not available")
	}
	assertion, err := s.webAuthnSvc.BeginAuthentication(ctx, userID)
	if err != nil {
		return nil, err
	}
	return json.Marshal(assertion)
}

func stepUpMethodAllowed(raw any, method string) bool {
	if method == "" {
		return false
	}
	values, ok := raw.([]any)
	if !ok {
		return true
	}
	for _, value := range values {
		if s, ok := value.(string); ok && s == method {
			return true
		}
	}
	return false
}

func mfaUserTenantID(ctx context.Context, db *gorm.DB, userID int64) int64 {
	var tenantID int64
	if err := db.WithContext(ctx).
		Table("user_identities").
		Select("tenant_id").
		Where("user_id = ?", userID).
		Order("tenant_id ASC").
		Limit(1).
		Scan(&tenantID).Error; err != nil || tenantID == 0 {
		return 0
	}
	return tenantID
}

func (s *mfaService) isSMSEnabled(ctx context.Context, userID int64) bool {
	record, err := s.smsPhoneRepo.FindByUserID(userID)
	return err == nil && record != nil && record.IsVerified
}

// EnrolledMFAMethods returns the user's usable MFA methods, read from their
// authoritative sources: the user TOTP/WebAuthn flags, the verified SMS phone
// record, and the backup-code count. backup_code is included only as a fallback
// alongside a primary factor (a primary factor is always present when codes
// exist, since codes are generated on enrollment and purged on full removal).
func (s *mfaService) EnrolledMFAMethods(ctx context.Context, userID int64) ([]string, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return nil, apperror.NewNotFound("user not found")
	}
	methods := make([]string, 0, 4)
	if user.IsTOTPEnabled {
		methods = append(methods, "totp")
	}
	if user.IsWebAuthnEnabled {
		methods = append(methods, "webauthn")
	}
	if s.isSMSEnabled(ctx, userID) {
		methods = append(methods, "sms")
	}
	if count, _ := s.GetBackupCodesCount(ctx, userID); count > 0 {
		methods = append(methods, "backup_code")
	}
	return methods, nil
}

// SyncMFAState reconciles a user's recovery/flag state with their remaining
// primary factors. If no primary factor (TOTP, WebAuthn, or verified SMS phone)
// is active, it purges any leftover backup codes and clears mfa_enabled_at so
// the account is left in a clean "no MFA" state. Idempotent and a no-op while at
// least one primary factor remains.
func (s *mfaService) SyncMFAState(ctx context.Context, userID int64) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return apperror.NewInternal("user lookup failed", err)
	}
	if user.IsTOTPEnabled || user.IsWebAuthnEnabled || s.isSMSEnabled(ctx, userID) {
		return nil
	}
	if err := s.backupCodeRepo.DeleteAllByUserID(userID); err != nil {
		return apperror.NewInternal("failed to delete backup codes", err)
	}
	if err := s.db.Model(&User{}).Where("user_id = ?", userID).
		Update("mfa_enabled_at", nil).Error; err != nil {
		return apperror.NewInternal("failed to clear MFA state", err)
	}
	return nil
}

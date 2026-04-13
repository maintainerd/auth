package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SecuritySettingServiceDataResult struct {
	SecuritySettingUUID uuid.UUID
	UserPoolID          int64
	MFAConfig           map[string]any
	PasswordConfig      map[string]any
	SessionConfig       map[string]any
	ThreatConfig        map[string]any
	LockoutConfig       map[string]any
	RegistrationConfig  map[string]any
	TokenConfig         map[string]any
	Version             int
	CreatedBy           *int64
	UpdatedBy           *int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type SecuritySettingService interface {
	GetByUserPoolID(ctx context.Context, userPoolID int64) (*SecuritySettingServiceDataResult, error)
	GetMFAConfig(ctx context.Context, userPoolID int64) (map[string]any, error)
	GetPasswordConfig(ctx context.Context, userPoolID int64) (map[string]any, error)
	GetSessionConfig(ctx context.Context, userPoolID int64) (map[string]any, error)
	GetThreatConfig(ctx context.Context, userPoolID int64) (map[string]any, error)
	GetLockoutConfig(ctx context.Context, userPoolID int64) (map[string]any, error)
	GetRegistrationConfig(ctx context.Context, userPoolID int64) (map[string]any, error)
	GetTokenConfig(ctx context.Context, userPoolID int64) (map[string]any, error)
	UpdateMFAConfig(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	UpdatePasswordConfig(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	UpdateSessionConfig(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	UpdateThreatConfig(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	UpdateLockoutConfig(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	UpdateRegistrationConfig(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	UpdateTokenConfig(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
}

type securitySettingService struct {
	db                        *gorm.DB
	securitySettingRepo       repository.SecuritySettingRepository
	securitySettingsAuditRepo repository.SecuritySettingsAuditRepository
}

func NewSecuritySettingService(
	db *gorm.DB,
	securitySettingRepo repository.SecuritySettingRepository,
	securitySettingsAuditRepo repository.SecuritySettingsAuditRepository,
) SecuritySettingService {
	return &securitySettingService{
		db:                        db,
		securitySettingRepo:       securitySettingRepo,
		securitySettingsAuditRepo: securitySettingsAuditRepo,
	}
}

func toSecuritySettingServiceDataResult(ss *model.SecuritySetting) *SecuritySettingServiceDataResult {
	return &SecuritySettingServiceDataResult{
		SecuritySettingUUID: ss.SecuritySettingUUID,
		UserPoolID:          ss.UserPoolID,
		MFAConfig:           unmarshalJSON(ss.MFAConfig),
		PasswordConfig:      unmarshalJSON(ss.PasswordConfig),
		SessionConfig:       unmarshalJSON(ss.SessionConfig),
		ThreatConfig:        unmarshalJSON(ss.ThreatConfig),
		LockoutConfig:       unmarshalJSON(ss.LockoutConfig),
		RegistrationConfig:  unmarshalJSON(ss.RegistrationConfig),
		TokenConfig:         unmarshalJSON(ss.TokenConfig),
		Version:             ss.Version,
		CreatedBy:           ss.CreatedBy,
		UpdatedBy:           ss.UpdatedBy,
		CreatedAt:           ss.CreatedAt,
		UpdatedAt:           ss.UpdatedAt,
	}
}

func unmarshalJSON(data datatypes.JSON) map[string]any {
	var result map[string]any
	if len(data) > 0 {
		if err := json.Unmarshal(data, &result); err != nil {
			result = nil // fall through to empty-map default below
		}
	}
	if result == nil {
		result = make(map[string]any)
	}
	return result
}

func (s *securitySettingService) GetByUserPoolID(ctx context.Context, userPoolID int64) (*SecuritySettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.get")
	defer span.End()
	span.SetAttributes(attribute.Int64("user_pool.id", userPoolID))

	setting, err := s.securitySettingRepo.FindByUserPoolID(userPoolID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get security setting failed")
		return nil, err
	}
	if setting == nil {
		span.SetStatus(codes.Error, "get security setting failed")
		return nil, apperror.NewNotFoundWithReason("security settings not found")
	}
	span.SetStatus(codes.Ok, "")
	return toSecuritySettingServiceDataResult(setting), nil
}

func (s *securitySettingService) GetMFAConfig(ctx context.Context, userPoolID int64) (map[string]any, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.getMFA")
	defer span.End()
	span.SetAttributes(attribute.Int64("user_pool.id", userPoolID))

	setting, err := s.getOrCreateSecuritySetting(userPoolID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get mfa config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return unmarshalJSON(setting.MFAConfig), nil
}

func (s *securitySettingService) GetPasswordConfig(ctx context.Context, userPoolID int64) (map[string]any, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.getPassword")
	defer span.End()
	span.SetAttributes(attribute.Int64("user_pool.id", userPoolID))

	setting, err := s.getOrCreateSecuritySetting(userPoolID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get password config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return unmarshalJSON(setting.PasswordConfig), nil
}

func (s *securitySettingService) GetSessionConfig(ctx context.Context, userPoolID int64) (map[string]any, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.getSession")
	defer span.End()
	span.SetAttributes(attribute.Int64("user_pool.id", userPoolID))

	setting, err := s.getOrCreateSecuritySetting(userPoolID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get session config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return unmarshalJSON(setting.SessionConfig), nil
}

func (s *securitySettingService) GetThreatConfig(ctx context.Context, userPoolID int64) (map[string]any, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.getThreat")
	defer span.End()
	span.SetAttributes(attribute.Int64("user_pool.id", userPoolID))

	setting, err := s.getOrCreateSecuritySetting(userPoolID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get threat config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return unmarshalJSON(setting.ThreatConfig), nil
}

func (s *securitySettingService) GetLockoutConfig(ctx context.Context, userPoolID int64) (map[string]any, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.getLockout")
	defer span.End()
	span.SetAttributes(attribute.Int64("user_pool.id", userPoolID))

	setting, err := s.getOrCreateSecuritySetting(userPoolID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get lockout config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return unmarshalJSON(setting.LockoutConfig), nil
}

func (s *securitySettingService) GetRegistrationConfig(ctx context.Context, userPoolID int64) (map[string]any, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.getRegistration")
	defer span.End()
	span.SetAttributes(attribute.Int64("user_pool.id", userPoolID))

	setting, err := s.getOrCreateSecuritySetting(userPoolID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get registration config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return unmarshalJSON(setting.RegistrationConfig), nil
}

func (s *securitySettingService) GetTokenConfig(ctx context.Context, userPoolID int64) (map[string]any, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.getToken")
	defer span.End()
	span.SetAttributes(attribute.Int64("user_pool.id", userPoolID))

	setting, err := s.getOrCreateSecuritySetting(userPoolID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get token config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return unmarshalJSON(setting.TokenConfig), nil
}

func (s *securitySettingService) UpdateMFAConfig(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.updateMFA")
	defer span.End()
	span.SetAttributes(attribute.Int64("user_pool.id", userPoolID))

	result, err := s.updateConfig(userPoolID, "general", config, updatedBy, ipAddress, userAgent)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update mfa config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *securitySettingService) UpdatePasswordConfig(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.updatePassword")
	defer span.End()
	span.SetAttributes(attribute.Int64("user_pool.id", userPoolID))

	result, err := s.updateConfig(userPoolID, "password", config, updatedBy, ipAddress, userAgent)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update password config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *securitySettingService) UpdateSessionConfig(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.updateSession")
	defer span.End()
	span.SetAttributes(attribute.Int64("user_pool.id", userPoolID))

	result, err := s.updateConfig(userPoolID, "session", config, updatedBy, ipAddress, userAgent)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update session config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *securitySettingService) UpdateThreatConfig(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.updateThreat")
	defer span.End()
	span.SetAttributes(attribute.Int64("user_pool.id", userPoolID))

	result, err := s.updateConfig(userPoolID, "threat", config, updatedBy, ipAddress, userAgent)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update threat config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *securitySettingService) UpdateLockoutConfig(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.updateLockout")
	defer span.End()
	span.SetAttributes(attribute.Int64("user_pool.id", userPoolID))

	result, err := s.updateConfig(userPoolID, "lockout", config, updatedBy, ipAddress, userAgent)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update lockout config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *securitySettingService) UpdateRegistrationConfig(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.updateRegistration")
	defer span.End()
	span.SetAttributes(attribute.Int64("user_pool.id", userPoolID))

	result, err := s.updateConfig(userPoolID, "registration", config, updatedBy, ipAddress, userAgent)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update registration config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *securitySettingService) UpdateTokenConfig(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.updateToken")
	defer span.End()
	span.SetAttributes(attribute.Int64("user_pool.id", userPoolID))

	result, err := s.updateConfig(userPoolID, "token", config, updatedBy, ipAddress, userAgent)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update token config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *securitySettingService) getOrCreateSecuritySetting(userPoolID int64) (*model.SecuritySetting, error) {
	setting, err := s.securitySettingRepo.FindByUserPoolID(userPoolID)
	if err != nil {
		return nil, err
	}

	if setting == nil {
		// Create default security setting
		setting = &model.SecuritySetting{
			UserPoolID:         userPoolID,
			MFAConfig:          datatypes.JSON([]byte("{}")),
			PasswordConfig:     datatypes.JSON([]byte("{}")),
			SessionConfig:      datatypes.JSON([]byte("{}")),
			ThreatConfig:       datatypes.JSON([]byte("{}")),
			LockoutConfig:      datatypes.JSON([]byte("{}")),
			RegistrationConfig: datatypes.JSON([]byte("{}")),
			TokenConfig:        datatypes.JSON([]byte("{}")),
			Version:            1,
		}
		created, err := s.securitySettingRepo.Create(setting)
		if err != nil {
			return nil, err
		}
		return created, nil
	}

	return setting, nil
}

func (s *securitySettingService) updateConfig(userPoolID int64, configType string, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	var updatedSetting *model.SecuritySetting

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txSecuritySettingRepo := s.securitySettingRepo.WithTx(tx)
		txAuditRepo := s.securitySettingsAuditRepo.WithTx(tx)

		// Get or create security setting
		setting, err := txSecuritySettingRepo.FindByUserPoolID(userPoolID)
		if err != nil {
			return err
		}

		var oldConfigJSON datatypes.JSON
		var isNew bool

		if setting == nil {
			// Create new security setting
			isNew = true
			setting = &model.SecuritySetting{
				UserPoolID:         userPoolID,
				MFAConfig:          datatypes.JSON([]byte("{}")),
				PasswordConfig:     datatypes.JSON([]byte("{}")),
				SessionConfig:      datatypes.JSON([]byte("{}")),
				ThreatConfig:       datatypes.JSON([]byte("{}")),
				LockoutConfig:      datatypes.JSON([]byte("{}")),
				RegistrationConfig: datatypes.JSON([]byte("{}")),
				TokenConfig:        datatypes.JSON([]byte("{}")),
				Version:            1,
				CreatedBy:          &updatedBy,
			}
		}

		// Marshal new config
		configBytes, err := json.Marshal(config)
		if err != nil {
			return err
		}
		newConfigJSON := datatypes.JSON(configBytes)

		// Update the appropriate config field and capture old value
		switch configType {
		case "mfa":
			oldConfigJSON = setting.MFAConfig
			setting.MFAConfig = newConfigJSON
		case "password":
			oldConfigJSON = setting.PasswordConfig
			setting.PasswordConfig = newConfigJSON
		case "session":
			oldConfigJSON = setting.SessionConfig
			setting.SessionConfig = newConfigJSON
		case "threat":
			oldConfigJSON = setting.ThreatConfig
			setting.ThreatConfig = newConfigJSON
		case "lockout":
			oldConfigJSON = setting.LockoutConfig
			setting.LockoutConfig = newConfigJSON
		case "registration":
			oldConfigJSON = setting.RegistrationConfig
			setting.RegistrationConfig = newConfigJSON
		case "token":
			oldConfigJSON = setting.TokenConfig
			setting.TokenConfig = newConfigJSON
		default:
			return apperror.NewValidation("invalid config type")
		}

		setting.UpdatedBy = &updatedBy

		// Save setting
		var saved *model.SecuritySetting
		if isNew {
			saved, err = txSecuritySettingRepo.Create(setting)
		} else {
			saved, err = txSecuritySettingRepo.CreateOrUpdate(setting)
		}
		if err != nil {
			return err
		}

		// Increment version
		if err := txSecuritySettingRepo.IncrementVersion(saved.SecuritySettingID); err != nil {
			return err
		}

		// Create audit record
		audit := &model.SecuritySettingsAudit{
			UserPoolID:        userPoolID,
			SecuritySettingID: saved.SecuritySettingID,
			ChangeType:        "update_" + configType + "_config",
			OldConfig:         oldConfigJSON,
			NewConfig:         newConfigJSON,
			IPAddress:         ipAddress,
			UserAgent:         userAgent,
			CreatedBy:         &updatedBy,
		}
		if _, err := txAuditRepo.Create(audit); err != nil {
			return err
		}

		updatedSetting = saved
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Refresh to get updated version
	refreshed, err := s.securitySettingRepo.FindByUUID(updatedSetting.SecuritySettingUUID)
	if err != nil {
		return nil, err
	}

	return toSecuritySettingServiceDataResult(refreshed), nil
}

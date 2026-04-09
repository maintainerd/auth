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
	TenantID            int64
	GeneralConfig       map[string]any
	PasswordConfig      map[string]any
	SessionConfig       map[string]any
	ThreatConfig        map[string]any
	IPConfig            map[string]any
	Version             int
	CreatedBy           *int64
	UpdatedBy           *int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type SecuritySettingService interface {
	GetByTenantID(ctx context.Context, tenantID int64) (*SecuritySettingServiceDataResult, error)
	GetGeneralConfig(ctx context.Context, tenantID int64) (map[string]any, error)
	GetPasswordConfig(ctx context.Context, tenantID int64) (map[string]any, error)
	GetSessionConfig(ctx context.Context, tenantID int64) (map[string]any, error)
	GetThreatConfig(ctx context.Context, tenantID int64) (map[string]any, error)
	GetIPConfig(ctx context.Context, tenantID int64) (map[string]any, error)
	UpdateGeneralConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	UpdatePasswordConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	UpdateSessionConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	UpdateThreatConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	UpdateIPConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
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
		TenantID:            ss.TenantID,
		GeneralConfig:       unmarshalJSON(ss.GeneralConfig),
		PasswordConfig:      unmarshalJSON(ss.PasswordConfig),
		SessionConfig:       unmarshalJSON(ss.SessionConfig),
		ThreatConfig:        unmarshalJSON(ss.ThreatConfig),
		IPConfig:            unmarshalJSON(ss.IPConfig),
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

func (s *securitySettingService) GetByTenantID(ctx context.Context, tenantID int64) (*SecuritySettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.get")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	setting, err := s.securitySettingRepo.FindByTenantID(tenantID)
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

func (s *securitySettingService) GetGeneralConfig(ctx context.Context, tenantID int64) (map[string]any, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.getGeneral")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	setting, err := s.getOrCreateSecuritySetting(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get general config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return unmarshalJSON(setting.GeneralConfig), nil
}

func (s *securitySettingService) GetPasswordConfig(ctx context.Context, tenantID int64) (map[string]any, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.getPassword")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	setting, err := s.getOrCreateSecuritySetting(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get password config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return unmarshalJSON(setting.PasswordConfig), nil
}

func (s *securitySettingService) GetSessionConfig(ctx context.Context, tenantID int64) (map[string]any, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.getSession")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	setting, err := s.getOrCreateSecuritySetting(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get session config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return unmarshalJSON(setting.SessionConfig), nil
}

func (s *securitySettingService) GetThreatConfig(ctx context.Context, tenantID int64) (map[string]any, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.getThreat")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	setting, err := s.getOrCreateSecuritySetting(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get threat config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return unmarshalJSON(setting.ThreatConfig), nil
}

func (s *securitySettingService) GetIPConfig(ctx context.Context, tenantID int64) (map[string]any, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.getIP")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	setting, err := s.getOrCreateSecuritySetting(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get ip config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return unmarshalJSON(setting.IPConfig), nil
}

func (s *securitySettingService) UpdateGeneralConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.updateGeneral")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	result, err := s.updateConfig(tenantID, "general", config, updatedBy, ipAddress, userAgent)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update general config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *securitySettingService) UpdatePasswordConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.updatePassword")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	result, err := s.updateConfig(tenantID, "password", config, updatedBy, ipAddress, userAgent)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update password config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *securitySettingService) UpdateSessionConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.updateSession")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	result, err := s.updateConfig(tenantID, "session", config, updatedBy, ipAddress, userAgent)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update session config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *securitySettingService) UpdateThreatConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.updateThreat")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	result, err := s.updateConfig(tenantID, "threat", config, updatedBy, ipAddress, userAgent)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update threat config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *securitySettingService) UpdateIPConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "securitySetting.updateIP")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	result, err := s.updateConfig(tenantID, "ip", config, updatedBy, ipAddress, userAgent)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update ip config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *securitySettingService) getOrCreateSecuritySetting(tenantID int64) (*model.SecuritySetting, error) {
	setting, err := s.securitySettingRepo.FindByTenantID(tenantID)
	if err != nil {
		return nil, err
	}

	if setting == nil {
		// Create default security setting
		setting = &model.SecuritySetting{
			TenantID:       tenantID,
			GeneralConfig:  datatypes.JSON([]byte("{}")),
			PasswordConfig: datatypes.JSON([]byte("{}")),
			SessionConfig:  datatypes.JSON([]byte("{}")),
			ThreatConfig:   datatypes.JSON([]byte("{}")),
			IPConfig:       datatypes.JSON([]byte("{}")),
			Version:        1,
		}
		created, err := s.securitySettingRepo.Create(setting)
		if err != nil {
			return nil, err
		}
		return created, nil
	}

	return setting, nil
}

func (s *securitySettingService) updateConfig(tenantID int64, configType string, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	var updatedSetting *model.SecuritySetting

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txSecuritySettingRepo := s.securitySettingRepo.WithTx(tx)
		txAuditRepo := s.securitySettingsAuditRepo.WithTx(tx)

		// Get or create security setting
		setting, err := txSecuritySettingRepo.FindByTenantID(tenantID)
		if err != nil {
			return err
		}

		var oldConfigJSON datatypes.JSON
		var isNew bool

		if setting == nil {
			// Create new security setting
			isNew = true
			setting = &model.SecuritySetting{
				TenantID:       tenantID,
				GeneralConfig:  datatypes.JSON([]byte("{}")),
				PasswordConfig: datatypes.JSON([]byte("{}")),
				SessionConfig:  datatypes.JSON([]byte("{}")),
				ThreatConfig:   datatypes.JSON([]byte("{}")),
				IPConfig:       datatypes.JSON([]byte("{}")),
				Version:        1,
				CreatedBy:      &updatedBy,
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
		case "general":
			oldConfigJSON = setting.GeneralConfig
			setting.GeneralConfig = newConfigJSON
		case "password":
			oldConfigJSON = setting.PasswordConfig
			setting.PasswordConfig = newConfigJSON
		case "session":
			oldConfigJSON = setting.SessionConfig
			setting.SessionConfig = newConfigJSON
		case "threat":
			oldConfigJSON = setting.ThreatConfig
			setting.ThreatConfig = newConfigJSON
		case "ip":
			oldConfigJSON = setting.IPConfig
			setting.IPConfig = newConfigJSON
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
			TenantID:          tenantID,
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

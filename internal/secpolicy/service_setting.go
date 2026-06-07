package secpolicy

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/jsonutil"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SecuritySettingServiceDataResult struct {
	SecuritySettingUUID uuid.UUID
	TenantID            int64
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
	GetByTenantID(ctx context.Context, tenantID int64) (*SecuritySettingServiceDataResult, error)
	GetMFAConfig(ctx context.Context, tenantID int64) (map[string]any, error)
	GetPasswordConfig(ctx context.Context, tenantID int64) (map[string]any, error)
	GetSessionConfig(ctx context.Context, tenantID int64) (map[string]any, error)
	GetThreatConfig(ctx context.Context, tenantID int64) (map[string]any, error)
	GetLockoutConfig(ctx context.Context, tenantID int64) (map[string]any, error)
	GetRegistrationConfig(ctx context.Context, tenantID int64) (map[string]any, error)
	GetTokenConfig(ctx context.Context, tenantID int64) (map[string]any, error)
	UpdateMFAConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	UpdatePasswordConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	UpdateSessionConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	UpdateThreatConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	UpdateLockoutConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	UpdateRegistrationConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	UpdateTokenConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
}

type securitySettingService struct {
	db                        *gorm.DB
	securitySettingRepo       SecuritySettingRepository
	securitySettingsAuditRepo SecuritySettingsAuditRepository
}

type securityConfigDefinition struct {
	key        string
	spanGet    string
	spanUpdate string
	errGet     string
	errUpdate  string
	selectJSON func(*SecuritySetting) datatypes.JSON
	assignJSON func(*SecuritySetting, datatypes.JSON)
}

var securityConfigDefinitions = map[string]securityConfigDefinition{
	"mfa": {
		key: "mfa", spanGet: "securitySetting.getMFA", spanUpdate: "securitySetting.updateMFA",
		errGet: "get mfa config failed", errUpdate: "update mfa config failed",
		selectJSON: func(s *SecuritySetting) datatypes.JSON { return s.MFAConfig },
		assignJSON: func(s *SecuritySetting, v datatypes.JSON) { s.MFAConfig = v },
	},
	"password": {
		key: "password", spanGet: "securitySetting.getPassword", spanUpdate: "securitySetting.updatePassword",
		errGet: "get password config failed", errUpdate: "update password config failed",
		selectJSON: func(s *SecuritySetting) datatypes.JSON { return s.PasswordConfig },
		assignJSON: func(s *SecuritySetting, v datatypes.JSON) { s.PasswordConfig = v },
	},
	"session": {
		key: "session", spanGet: "securitySetting.getSession", spanUpdate: "securitySetting.updateSession",
		errGet: "get session config failed", errUpdate: "update session config failed",
		selectJSON: func(s *SecuritySetting) datatypes.JSON { return s.SessionConfig },
		assignJSON: func(s *SecuritySetting, v datatypes.JSON) { s.SessionConfig = v },
	},
	"threat": {
		key: "threat", spanGet: "securitySetting.getThreat", spanUpdate: "securitySetting.updateThreat",
		errGet: "get threat config failed", errUpdate: "update threat config failed",
		selectJSON: func(s *SecuritySetting) datatypes.JSON { return s.ThreatConfig },
		assignJSON: func(s *SecuritySetting, v datatypes.JSON) { s.ThreatConfig = v },
	},
	"lockout": {
		key: "lockout", spanGet: "securitySetting.getLockout", spanUpdate: "securitySetting.updateLockout",
		errGet: "get lockout config failed", errUpdate: "update lockout config failed",
		selectJSON: func(s *SecuritySetting) datatypes.JSON { return s.LockoutConfig },
		assignJSON: func(s *SecuritySetting, v datatypes.JSON) { s.LockoutConfig = v },
	},
	"registration": {
		key: "registration", spanGet: "securitySetting.getRegistration", spanUpdate: "securitySetting.updateRegistration",
		errGet: "get registration config failed", errUpdate: "update registration config failed",
		selectJSON: func(s *SecuritySetting) datatypes.JSON { return s.RegistrationConfig },
		assignJSON: func(s *SecuritySetting, v datatypes.JSON) { s.RegistrationConfig = v },
	},
	"token": {
		key: "token", spanGet: "securitySetting.getToken", spanUpdate: "securitySetting.updateToken",
		errGet: "get token config failed", errUpdate: "update token config failed",
		selectJSON: func(s *SecuritySetting) datatypes.JSON { return s.TokenConfig },
		assignJSON: func(s *SecuritySetting, v datatypes.JSON) { s.TokenConfig = v },
	},
}

func NewSecuritySettingService(
	db *gorm.DB,
	securitySettingRepo SecuritySettingRepository,
	securitySettingsAuditRepo SecuritySettingsAuditRepository,
) SecuritySettingService {
	return &securitySettingService{
		db:                        db,
		securitySettingRepo:       securitySettingRepo,
		securitySettingsAuditRepo: securitySettingsAuditRepo,
	}
}

func toSecuritySettingServiceDataResult(ss *SecuritySetting) *SecuritySettingServiceDataResult {
	return &SecuritySettingServiceDataResult{
		SecuritySettingUUID: ss.SecuritySettingUUID,
		TenantID:            ss.TenantID,
		MFAConfig:           jsonutil.JSONToMap(ss.MFAConfig),
		PasswordConfig:      jsonutil.JSONToMap(ss.PasswordConfig),
		SessionConfig:       jsonutil.JSONToMap(ss.SessionConfig),
		ThreatConfig:        jsonutil.JSONToMap(ss.ThreatConfig),
		LockoutConfig:       jsonutil.JSONToMap(ss.LockoutConfig),
		RegistrationConfig:  jsonutil.JSONToMap(ss.RegistrationConfig),
		TokenConfig:         jsonutil.JSONToMap(ss.TokenConfig),
		Version:             ss.Version,
		CreatedBy:           ss.CreatedBy,
		UpdatedBy:           ss.UpdatedBy,
		CreatedAt:           ss.CreatedAt,
		UpdatedAt:           ss.UpdatedAt,
	}
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

func (s *securitySettingService) GetMFAConfig(ctx context.Context, tenantID int64) (map[string]any, error) {
	return s.getConfig(ctx, tenantID, "mfa")
}

func (s *securitySettingService) GetPasswordConfig(ctx context.Context, tenantID int64) (map[string]any, error) {
	return s.getConfig(ctx, tenantID, "password")
}

func (s *securitySettingService) GetSessionConfig(ctx context.Context, tenantID int64) (map[string]any, error) {
	return s.getConfig(ctx, tenantID, "session")
}

func (s *securitySettingService) GetThreatConfig(ctx context.Context, tenantID int64) (map[string]any, error) {
	return s.getConfig(ctx, tenantID, "threat")
}

func (s *securitySettingService) GetLockoutConfig(ctx context.Context, tenantID int64) (map[string]any, error) {
	return s.getConfig(ctx, tenantID, "lockout")
}

func (s *securitySettingService) GetRegistrationConfig(ctx context.Context, tenantID int64) (map[string]any, error) {
	return s.getConfig(ctx, tenantID, "registration")
}

func (s *securitySettingService) GetTokenConfig(ctx context.Context, tenantID int64) (map[string]any, error) {
	return s.getConfig(ctx, tenantID, "token")
}

func (s *securitySettingService) getConfig(ctx context.Context, tenantID int64, configType string) (map[string]any, error) {
	def, ok := securityConfigDefinitions[configType]
	if !ok {
		return nil, apperror.NewValidation("invalid config type")
	}

	_, span := otel.Tracer("service").Start(ctx, def.spanGet)
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	setting, err := s.getOrCreateSecuritySetting(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, def.errGet)
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return jsonutil.JSONToMap(def.selectJSON(setting)), nil
}

func (s *securitySettingService) UpdateMFAConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	return s.updateConfigByDefinition(ctx, tenantID, "mfa", config, updatedBy, ipAddress, userAgent)
}

func (s *securitySettingService) UpdatePasswordConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	return s.updateConfigByDefinition(ctx, tenantID, "password", config, updatedBy, ipAddress, userAgent)
}

func (s *securitySettingService) UpdateSessionConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	return s.updateConfigByDefinition(ctx, tenantID, "session", config, updatedBy, ipAddress, userAgent)
}

func (s *securitySettingService) UpdateThreatConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	return s.updateConfigByDefinition(ctx, tenantID, "threat", config, updatedBy, ipAddress, userAgent)
}

func (s *securitySettingService) UpdateLockoutConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	return s.updateConfigByDefinition(ctx, tenantID, "lockout", config, updatedBy, ipAddress, userAgent)
}

func (s *securitySettingService) UpdateRegistrationConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	return s.updateConfigByDefinition(ctx, tenantID, "registration", config, updatedBy, ipAddress, userAgent)
}

func (s *securitySettingService) UpdateTokenConfig(ctx context.Context, tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	return s.updateConfigByDefinition(ctx, tenantID, "token", config, updatedBy, ipAddress, userAgent)
}

func (s *securitySettingService) updateConfigByDefinition(ctx context.Context, tenantID int64, configType string, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	def, ok := securityConfigDefinitions[configType]
	if !ok {
		return nil, apperror.NewValidation("invalid config type")
	}

	_, span := otel.Tracer("service").Start(ctx, def.spanUpdate)
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	result, err := s.updateConfig(tenantID, def, config, updatedBy, ipAddress, userAgent)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, def.errUpdate)
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *securitySettingService) getOrCreateSecuritySetting(tenantID int64) (*SecuritySetting, error) {
	setting, err := s.securitySettingRepo.FindByTenantID(tenantID)
	if err != nil {
		return nil, err
	}

	if setting == nil {
		// Create default security setting
		setting = &SecuritySetting{
			TenantID:           tenantID,
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

func (s *securitySettingService) updateConfig(tenantID int64, def securityConfigDefinition, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	var updatedSetting *SecuritySetting

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
			setting = &SecuritySetting{
				TenantID:           tenantID,
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

		oldConfigJSON = def.selectJSON(setting)
		def.assignJSON(setting, newConfigJSON)

		setting.UpdatedBy = &updatedBy

		// Save setting
		var saved *SecuritySetting
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
		audit := &SecuritySettingsAudit{
			TenantID:          tenantID,
			SecuritySettingID: saved.SecuritySettingID,
			ChangeType:        "update_" + def.key + "_config",
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

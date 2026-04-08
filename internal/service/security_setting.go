package service

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"github.com/maintainerd/auth/internal/apperror"
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
	GetByTenantID(tenantID int64) (*SecuritySettingServiceDataResult, error)
	GetGeneralConfig(tenantID int64) (map[string]any, error)
	GetPasswordConfig(tenantID int64) (map[string]any, error)
	GetSessionConfig(tenantID int64) (map[string]any, error)
	GetThreatConfig(tenantID int64) (map[string]any, error)
	GetIPConfig(tenantID int64) (map[string]any, error)
	UpdateGeneralConfig(tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	UpdatePasswordConfig(tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	UpdateSessionConfig(tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	UpdateThreatConfig(tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	UpdateIPConfig(tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
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

func (s *securitySettingService) GetByTenantID(tenantID int64) (*SecuritySettingServiceDataResult, error) {
	setting, err := s.securitySettingRepo.FindByTenantID(tenantID)
	if err != nil {
		return nil, err
	}
	if setting == nil {
		return nil, apperror.NewNotFoundWithReason("security settings not found")
	}
	return toSecuritySettingServiceDataResult(setting), nil
}

func (s *securitySettingService) GetGeneralConfig(tenantID int64) (map[string]any, error) {
	setting, err := s.getOrCreateSecuritySetting(tenantID)
	if err != nil {
		return nil, err
	}
	return unmarshalJSON(setting.GeneralConfig), nil
}

func (s *securitySettingService) GetPasswordConfig(tenantID int64) (map[string]any, error) {
	setting, err := s.getOrCreateSecuritySetting(tenantID)
	if err != nil {
		return nil, err
	}
	return unmarshalJSON(setting.PasswordConfig), nil
}

func (s *securitySettingService) GetSessionConfig(tenantID int64) (map[string]any, error) {
	setting, err := s.getOrCreateSecuritySetting(tenantID)
	if err != nil {
		return nil, err
	}
	return unmarshalJSON(setting.SessionConfig), nil
}

func (s *securitySettingService) GetThreatConfig(tenantID int64) (map[string]any, error) {
	setting, err := s.getOrCreateSecuritySetting(tenantID)
	if err != nil {
		return nil, err
	}
	return unmarshalJSON(setting.ThreatConfig), nil
}

func (s *securitySettingService) GetIPConfig(tenantID int64) (map[string]any, error) {
	setting, err := s.getOrCreateSecuritySetting(tenantID)
	if err != nil {
		return nil, err
	}
	return unmarshalJSON(setting.IPConfig), nil
}

func (s *securitySettingService) UpdateGeneralConfig(tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	return s.updateConfig(tenantID, "general", config, updatedBy, ipAddress, userAgent)
}

func (s *securitySettingService) UpdatePasswordConfig(tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	return s.updateConfig(tenantID, "password", config, updatedBy, ipAddress, userAgent)
}

func (s *securitySettingService) UpdateSessionConfig(tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	return s.updateConfig(tenantID, "session", config, updatedBy, ipAddress, userAgent)
}

func (s *securitySettingService) UpdateThreatConfig(tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	return s.updateConfig(tenantID, "threat", config, updatedBy, ipAddress, userAgent)
}

func (s *securitySettingService) UpdateIPConfig(tenantID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	return s.updateConfig(tenantID, "ip", config, updatedBy, ipAddress, userAgent)
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

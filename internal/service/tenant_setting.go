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
)

// TenantSettingServiceDataResult is the service-layer representation of a
// tenant_settings record.
type TenantSettingServiceDataResult struct {
	TenantSettingUUID uuid.UUID
	RateLimitConfig   map[string]any
	AuditConfig       map[string]any
	MaintenanceConfig map[string]any
	FeatureFlags      map[string]any
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// TenantSettingService defines business operations on tenant settings.
type TenantSettingService interface {
	Get(ctx context.Context, tenantID int64) (*TenantSettingServiceDataResult, error)
	GetRateLimitConfig(ctx context.Context, tenantID int64) (map[string]any, error)
	GetAuditConfig(ctx context.Context, tenantID int64) (map[string]any, error)
	GetMaintenanceConfig(ctx context.Context, tenantID int64) (map[string]any, error)
	GetFeatureFlags(ctx context.Context, tenantID int64) (map[string]any, error)
	UpdateRateLimitConfig(ctx context.Context, tenantID int64, config map[string]any) (*TenantSettingServiceDataResult, error)
	UpdateAuditConfig(ctx context.Context, tenantID int64, config map[string]any) (*TenantSettingServiceDataResult, error)
	UpdateMaintenanceConfig(ctx context.Context, tenantID int64, config map[string]any) (*TenantSettingServiceDataResult, error)
	UpdateFeatureFlags(ctx context.Context, tenantID int64, config map[string]any) (*TenantSettingServiceDataResult, error)
}

type tenantSettingService struct {
	tenantSettingRepo repository.TenantSettingRepository
}

// NewTenantSettingService creates a new TenantSettingService.
func NewTenantSettingService(tenantSettingRepo repository.TenantSettingRepository) TenantSettingService {
	return &tenantSettingService{tenantSettingRepo: tenantSettingRepo}
}

func toTenantSettingServiceDataResult(ts *model.TenantSetting) *TenantSettingServiceDataResult {
	return &TenantSettingServiceDataResult{
		TenantSettingUUID: ts.TenantSettingUUID,
		RateLimitConfig:   unmarshalJSON(ts.RateLimitConfig),
		AuditConfig:       unmarshalJSON(ts.AuditConfig),
		MaintenanceConfig: unmarshalJSON(ts.MaintenanceConfig),
		FeatureFlags:      unmarshalJSON(ts.FeatureFlags),
		CreatedAt:         ts.CreatedAt,
		UpdatedAt:         ts.UpdatedAt,
	}
}

// Get retrieves the full tenant setting record, auto-creating if missing.
func (s *tenantSettingService) Get(ctx context.Context, tenantID int64) (*TenantSettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantSetting.get")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	setting, err := s.getOrCreate(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get tenant setting failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toTenantSettingServiceDataResult(setting), nil
}

// GetRateLimitConfig retrieves the rate_limit_config JSONB section.
func (s *tenantSettingService) GetRateLimitConfig(ctx context.Context, tenantID int64) (map[string]any, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantSetting.getRateLimit")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	setting, err := s.getOrCreate(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get rate limit config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return unmarshalJSON(setting.RateLimitConfig), nil
}

// GetAuditConfig retrieves the audit_config JSONB section.
func (s *tenantSettingService) GetAuditConfig(ctx context.Context, tenantID int64) (map[string]any, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantSetting.getAudit")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	setting, err := s.getOrCreate(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get audit config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return unmarshalJSON(setting.AuditConfig), nil
}

// GetMaintenanceConfig retrieves the maintenance_config JSONB section.
func (s *tenantSettingService) GetMaintenanceConfig(ctx context.Context, tenantID int64) (map[string]any, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantSetting.getMaintenance")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	setting, err := s.getOrCreate(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get maintenance config failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return unmarshalJSON(setting.MaintenanceConfig), nil
}

// GetFeatureFlags retrieves the feature_flags JSONB section.
func (s *tenantSettingService) GetFeatureFlags(ctx context.Context, tenantID int64) (map[string]any, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantSetting.getFeatureFlags")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	setting, err := s.getOrCreate(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get feature flags failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return unmarshalJSON(setting.FeatureFlags), nil
}

// UpdateRateLimitConfig updates the rate_limit_config JSONB section.
func (s *tenantSettingService) UpdateRateLimitConfig(ctx context.Context, tenantID int64, config map[string]any) (*TenantSettingServiceDataResult, error) {
	return s.updateConfig(ctx, tenantID, "rate_limit", config)
}

// UpdateAuditConfig updates the audit_config JSONB section.
func (s *tenantSettingService) UpdateAuditConfig(ctx context.Context, tenantID int64, config map[string]any) (*TenantSettingServiceDataResult, error) {
	return s.updateConfig(ctx, tenantID, "audit", config)
}

// UpdateMaintenanceConfig updates the maintenance_config JSONB section.
func (s *tenantSettingService) UpdateMaintenanceConfig(ctx context.Context, tenantID int64, config map[string]any) (*TenantSettingServiceDataResult, error) {
	return s.updateConfig(ctx, tenantID, "maintenance", config)
}

// UpdateFeatureFlags updates the feature_flags JSONB section.
func (s *tenantSettingService) UpdateFeatureFlags(ctx context.Context, tenantID int64, config map[string]any) (*TenantSettingServiceDataResult, error) {
	return s.updateConfig(ctx, tenantID, "feature_flags", config)
}

func (s *tenantSettingService) updateConfig(ctx context.Context, tenantID int64, configType string, config map[string]any) (*TenantSettingServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantSetting.update"+configType)
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	setting, err := s.getOrCreate(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get tenant setting for update failed")
		return nil, err
	}

	configBytes, err := json.Marshal(config)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "marshal config failed")
		return nil, apperror.NewValidation("invalid config payload")
	}
	jsonData := datatypes.JSON(configBytes)

	switch configType {
	case "rate_limit":
		setting.RateLimitConfig = jsonData
	case "audit":
		setting.AuditConfig = jsonData
	case "maintenance":
		setting.MaintenanceConfig = jsonData
	case "feature_flags":
		setting.FeatureFlags = jsonData
	default:
		return nil, apperror.NewValidation("invalid config type")
	}

	updated, err := s.tenantSettingRepo.CreateOrUpdate(setting)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update tenant setting failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toTenantSettingServiceDataResult(updated), nil
}

func (s *tenantSettingService) getOrCreate(tenantID int64) (*model.TenantSetting, error) {
	setting, err := s.tenantSettingRepo.FindByTenantID(tenantID)
	if err != nil {
		return nil, err
	}
	if setting != nil {
		return setting, nil
	}

	setting = &model.TenantSetting{
		TenantID:          tenantID,
		RateLimitConfig:   datatypes.JSON([]byte("{}")),
		AuditConfig:       datatypes.JSON([]byte("{}")),
		MaintenanceConfig: datatypes.JSON([]byte("{}")),
		FeatureFlags:      datatypes.JSON([]byte("{}")),
	}
	created, err := s.tenantSettingRepo.Create(setting)
	if err != nil {
		return nil, apperror.NewInternal("failed to create default tenant settings", err)
	}
	return created, nil
}

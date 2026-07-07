package scim

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"go.opentelemetry.io/otel"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SCIMConfigurationService interface {
	Create(ctx context.Context, tenantID int64, input SCIMConfigurationCreateInput) (*SCIMConfigurationServiceDataResult, error)
	Update(ctx context.Context, scimUUID uuid.UUID, tenantID int64, input SCIMConfigurationUpdateInput) (*SCIMConfigurationServiceDataResult, error)
	GetByUUID(ctx context.Context, scimUUID uuid.UUID, tenantID int64) (*SCIMConfigurationServiceDataResult, error)
	Delete(ctx context.Context, scimUUID uuid.UUID, tenantID int64) error
	List(ctx context.Context, tenantID int64, filter SCIMConfigurationFilter) (*SCIMConfigurationListResult, error)
}

type SCIMConfigurationCreateInput struct {
	IdentityProviderID *int64
	DisplayName        string
	BaseURL            *string
	BearerToken        *string
	SyncUsers          bool
	SyncGroups         bool
	SyncDirection      string
	AttributeMapping   json.RawMessage
	IsActive           bool
}

type SCIMConfigurationUpdateInput struct {
	IdentityProviderID *int64
	DisplayName        *string
	BaseURL            *string
	BearerToken        *string
	SyncUsers          *bool
	SyncGroups         *bool
	SyncDirection      *string
	AttributeMapping   *json.RawMessage
	IsActive           *bool
}

type SCIMConfigurationListResult struct {
	Data       []SCIMConfigurationServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type scimConfigurationService struct {
	db   *gorm.DB
	repo SCIMConfigurationRepository
}

func NewSCIMConfigurationService(db *gorm.DB, repo SCIMConfigurationRepository) SCIMConfigurationService {
	return &scimConfigurationService{db: db, repo: repo}
}

func (s *scimConfigurationService) Create(ctx context.Context, tenantID int64, input SCIMConfigurationCreateInput) (*SCIMConfigurationServiceDataResult, error) {
	ctx, span := otel.Tracer("scim").Start(ctx, "SCIMConfigurationService.Create")
	defer span.End()

	attrMapping := datatypes.JSON("{}")
	if input.AttributeMapping != nil {
		attrMapping = datatypes.JSON(input.AttributeMapping)
	}

	cfg := &SCIMConfiguration{
		TenantID:           tenantID,
		IdentityProviderID: input.IdentityProviderID,
		DisplayName:        input.DisplayName,
		BaseURL:            input.BaseURL,
		SyncUsers:          input.SyncUsers,
		SyncGroups:         input.SyncGroups,
		SyncDirection:      input.SyncDirection,
		AttributeMapping:   attrMapping,
		IsActive:           input.IsActive,
	}

	if input.BearerToken != nil && *input.BearerToken != "" {
		hash := hashBearerToken(*input.BearerToken)
		cfg.BearerTokenHash = &hash
	}

	created, err := s.repo.Create(ctx, cfg)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperror.NewConflict("a SCIM configuration already exists for this tenant")
		}
		return nil, apperror.NewInternal("create scim configuration", err)
	}

	return toServiceDataResult(created), nil
}

func (s *scimConfigurationService) Update(ctx context.Context, scimUUID uuid.UUID, tenantID int64, input SCIMConfigurationUpdateInput) (*SCIMConfigurationServiceDataResult, error) {
	ctx, span := otel.Tracer("scim").Start(ctx, "SCIMConfigurationService.Update")
	defer span.End()

	cfg, err := s.repo.FindByUUID(ctx, scimUUID, tenantID)
	if err != nil {
		return nil, apperror.NewInternal("find scim configuration", err)
	}
	if cfg == nil {
		return nil, apperror.NewNotFound("scim_configuration")
	}

	if input.DisplayName != nil {
		cfg.DisplayName = *input.DisplayName
	}
	if input.BaseURL != nil {
		cfg.BaseURL = input.BaseURL
	}
	if input.BearerToken != nil {
		if *input.BearerToken != "" {
			hash := hashBearerToken(*input.BearerToken)
			cfg.BearerTokenHash = &hash
		} else {
			cfg.BearerTokenHash = nil
		}
	}
	if input.SyncUsers != nil {
		cfg.SyncUsers = *input.SyncUsers
	}
	if input.SyncGroups != nil {
		cfg.SyncGroups = *input.SyncGroups
	}
	if input.SyncDirection != nil {
		cfg.SyncDirection = *input.SyncDirection
	}
	if input.AttributeMapping != nil {
		cfg.AttributeMapping = datatypes.JSON(*input.AttributeMapping)
	}
	if input.IsActive != nil {
		cfg.IsActive = *input.IsActive
	}
	if input.IdentityProviderID != nil {
		cfg.IdentityProviderID = input.IdentityProviderID
	}

	updated, err := s.repo.Update(ctx, cfg)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperror.NewConflict("a SCIM configuration already exists for this tenant")
		}
		return nil, apperror.NewInternal("update scim configuration", err)
	}

	return toServiceDataResult(updated), nil
}

func (s *scimConfigurationService) GetByUUID(ctx context.Context, scimUUID uuid.UUID, tenantID int64) (*SCIMConfigurationServiceDataResult, error) {
	ctx, span := otel.Tracer("scim").Start(ctx, "SCIMConfigurationService.GetByUUID")
	defer span.End()

	cfg, err := s.repo.FindByUUID(ctx, scimUUID, tenantID)
	if err != nil {
		return nil, apperror.NewInternal("find scim configuration", err)
	}
	if cfg == nil {
		return nil, apperror.NewNotFound("scim_configuration")
	}
	return toServiceDataResult(cfg), nil
}

func (s *scimConfigurationService) Delete(ctx context.Context, scimUUID uuid.UUID, tenantID int64) error {
	ctx, span := otel.Tracer("scim").Start(ctx, "SCIMConfigurationService.Delete")
	defer span.End()

	cfg, err := s.repo.FindByUUID(ctx, scimUUID, tenantID)
	if err != nil {
		return apperror.NewInternal("find scim configuration", err)
	}
	if cfg == nil {
		return apperror.NewNotFound("scim_configuration")
	}
	if err := s.repo.Delete(ctx, cfg); err != nil {
		return apperror.NewInternal("delete scim configuration", err)
	}
	return nil
}

func (s *scimConfigurationService) List(ctx context.Context, tenantID int64, filter SCIMConfigurationFilter) (*SCIMConfigurationListResult, error) {
	ctx, span := otel.Tracer("scim").Start(ctx, "SCIMConfigurationService.List")
	defer span.End()

	result, err := s.repo.List(ctx, tenantID, filter)
	if err != nil {
		return nil, apperror.NewInternal("list scim configurations", err)
	}

	data := make([]SCIMConfigurationServiceDataResult, len(result.Data))
	for i, cfg := range result.Data {
		data[i] = *toServiceDataResult(&cfg)
	}

	return &SCIMConfigurationListResult{
		Data:       data,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func toServiceDataResult(cfg *SCIMConfiguration) *SCIMConfigurationServiceDataResult {
	r := &SCIMConfigurationServiceDataResult{
		SCIMConfigurationUUID: cfg.SCIMConfigurationUUID.String(),
		TenantID:              cfg.TenantID,
		IdentityProviderID:    cfg.IdentityProviderID,
		DisplayName:           cfg.DisplayName,
		BaseURL:               cfg.BaseURL,
		SyncUsers:             cfg.SyncUsers,
		SyncGroups:            cfg.SyncGroups,
		SyncDirection:         cfg.SyncDirection,
		AttributeMapping:      cfg.AttributeMapping,
		IsActive:              cfg.IsActive,
		CreatedAt:             cfg.CreatedAt.Format(time.RFC3339),
		UpdatedAt:             cfg.UpdatedAt.Format(time.RFC3339),
	}
	if cfg.LastSyncAt != nil {
		s := cfg.LastSyncAt.Format(time.RFC3339)
		r.LastSyncAt = &s
	}
	r.LastSyncStatus = cfg.LastSyncStatus
	r.LastSyncError = cfg.LastSyncError
	return r
}

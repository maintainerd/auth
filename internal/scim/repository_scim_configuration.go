package scim

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

type SCIMConfigurationRepository interface {
	FindByUUID(ctx context.Context, uuid uuid.UUID, tenantID int64) (*SCIMConfiguration, error)
	FindByTenantID(ctx context.Context, tenantID int64) (*SCIMConfiguration, error)
	FindByBearerTokenHash(ctx context.Context, hash string) (*SCIMConfiguration, error)
	Create(ctx context.Context, cfg *SCIMConfiguration) (*SCIMConfiguration, error)
	Update(ctx context.Context, cfg *SCIMConfiguration) (*SCIMConfiguration, error)
	Delete(ctx context.Context, cfg *SCIMConfiguration) error
	List(ctx context.Context, tenantID int64, filter SCIMConfigurationFilter) (*PaginationResult[SCIMConfiguration], error)
}

type SCIMConfigurationFilter struct {
	Search    string
	IsActive  *bool
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type scimConfigurationRepository struct {
	base *database.BaseRepository[SCIMConfiguration]
	db   *gorm.DB
}

func NewSCIMConfigurationRepository(db *gorm.DB) SCIMConfigurationRepository {
	return &scimConfigurationRepository{
		base: database.NewBaseRepository[SCIMConfiguration](db, "scim_configuration_uuid", "scim_configuration_id"),
		db:   db,
	}
}

func (r *scimConfigurationRepository) FindByUUID(ctx context.Context, scimUUID uuid.UUID, tenantID int64) (*SCIMConfiguration, error) {
	var cfg SCIMConfiguration
	err := r.db.WithContext(ctx).
		Where("scim_configuration_uuid = ? AND tenant_id = ?", scimUUID, tenantID).
		First(&cfg).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &cfg, nil
}

func (r *scimConfigurationRepository) FindByTenantID(ctx context.Context, tenantID int64) (*SCIMConfiguration, error) {
	var cfg SCIMConfiguration
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND is_active = ?", tenantID, true).
		First(&cfg).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &cfg, nil
}

func (r *scimConfigurationRepository) FindByBearerTokenHash(ctx context.Context, hash string) (*SCIMConfiguration, error) {
	var cfg SCIMConfiguration
	err := r.db.WithContext(ctx).
		Where("bearer_token_hash = ? AND is_active = ?", hash, true).
		First(&cfg).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &cfg, nil
}

func (r *scimConfigurationRepository) Create(ctx context.Context, cfg *SCIMConfiguration) (*SCIMConfiguration, error) {
	return r.base.Create(cfg)
}

func (r *scimConfigurationRepository) Update(ctx context.Context, cfg *SCIMConfiguration) (*SCIMConfiguration, error) {
	updateData := map[string]any{
		"identity_provider_id": cfg.IdentityProviderID,
		"display_name":         cfg.DisplayName,
		"base_url":             cfg.BaseURL,
		"bearer_token_hash":    cfg.BearerTokenHash,
		"sync_users":           cfg.SyncUsers,
		"sync_groups":          cfg.SyncGroups,
		"sync_direction":       cfg.SyncDirection,
		"attribute_mapping":    cfg.AttributeMapping,
		"is_active":            cfg.IsActive,
		"last_sync_at":         cfg.LastSyncAt,
		"last_sync_status":     cfg.LastSyncStatus,
		"last_sync_error":      cfg.LastSyncError,
	}
	return r.base.UpdateByUUID(cfg.SCIMConfigurationUUID, updateData)
}

func (r *scimConfigurationRepository) Delete(ctx context.Context, cfg *SCIMConfiguration) error {
	return r.base.DeleteByUUID(cfg.SCIMConfigurationUUID)
}

func (r *scimConfigurationRepository) List(ctx context.Context, tenantID int64, filter SCIMConfigurationFilter) (*PaginationResult[SCIMConfiguration], error) {
	q := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if filter.IsActive != nil {
		q = q.Where("is_active = ?", *filter.IsActive)
	}
	if filter.Search != "" {
		q = q.Where("display_name ILIKE ?", "%"+filter.Search+"%")
	}

	sortOrder := database.SanitizeOrder(filter.SortBy, filter.SortOrder, "created_at")
	q = q.Order(fmt.Sprintf("%s %s", sortOrder, "desc"))

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 20
	}

	return database.PaginateQuery[SCIMConfiguration](q, filter.Page, filter.Limit)
}

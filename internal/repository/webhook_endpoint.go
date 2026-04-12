package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

// WebhookEndpointRepositoryGetFilter holds query parameters for paginated
// webhook endpoint lookups.
type WebhookEndpointRepositoryGetFilter struct {
	TenantID  *int64
	Status    []string
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

// WebhookEndpointRepository defines persistence operations for the
// webhook_endpoints entity.
type WebhookEndpointRepository interface {
	BaseRepositoryMethods[model.WebhookEndpoint]
	WithTx(tx *gorm.DB) WebhookEndpointRepository
	FindByTenantID(tenantID int64) ([]model.WebhookEndpoint, error)
	FindByUUIDAndTenantID(webhookEndpointUUID uuid.UUID, tenantID int64) (*model.WebhookEndpoint, error)
	FindPaginated(filter WebhookEndpointRepositoryGetFilter) (*PaginationResult[model.WebhookEndpoint], error)
}

type webhookEndpointRepository struct {
	*BaseRepository[model.WebhookEndpoint]
}

// NewWebhookEndpointRepository creates a new WebhookEndpointRepository backed
// by the given database connection.
func NewWebhookEndpointRepository(db *gorm.DB) WebhookEndpointRepository {
	return &webhookEndpointRepository{
		BaseRepository: NewBaseRepository[model.WebhookEndpoint](db, "webhook_endpoint_uuid", "webhook_endpoint_id"),
	}
}

// WithTx returns a copy of the repository bound to the supplied transaction.
func (r *webhookEndpointRepository) WithTx(tx *gorm.DB) WebhookEndpointRepository {
	return &webhookEndpointRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByTenantID retrieves all webhook endpoints belonging to a tenant.
func (r *webhookEndpointRepository) FindByTenantID(tenantID int64) ([]model.WebhookEndpoint, error) {
	var endpoints []model.WebhookEndpoint
	err := r.DB().Where("tenant_id = ?", tenantID).Find(&endpoints).Error
	if err != nil {
		return nil, err
	}
	return endpoints, nil
}

// FindByUUIDAndTenantID retrieves a single webhook endpoint by UUID scoped to
// a tenant. Returns nil, nil when no record exists.
func (r *webhookEndpointRepository) FindByUUIDAndTenantID(webhookEndpointUUID uuid.UUID, tenantID int64) (*model.WebhookEndpoint, error) {
	var endpoint model.WebhookEndpoint
	err := r.DB().Where("webhook_endpoint_uuid = ? AND tenant_id = ?", webhookEndpointUUID, tenantID).First(&endpoint).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &endpoint, nil
}

// FindPaginated retrieves paginated webhook endpoints with filtering.
func (r *webhookEndpointRepository) FindPaginated(filter WebhookEndpointRepositoryGetFilter) (*PaginationResult[model.WebhookEndpoint], error) {
	query := r.DB().Model(&model.WebhookEndpoint{})

	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
	}
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit

	var endpoints []model.WebhookEndpoint
	if err := query.Offset(offset).Limit(filter.Limit).Find(&endpoints).Error; err != nil {
		return nil, err
	}

	totalPages := int(total) / filter.Limit
	if int(total)%filter.Limit > 0 {
		totalPages++
	}

	return &PaginationResult[model.WebhookEndpoint]{
		Data:       endpoints,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

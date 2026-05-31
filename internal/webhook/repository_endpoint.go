package webhook

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/database"
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
	BaseRepositoryMethods[WebhookEndpoint]
	WithTx(tx *gorm.DB) WebhookEndpointRepository
	FindByTenantID(tenantID int64) ([]WebhookEndpoint, error)
	FindActiveByTenantID(tenantID int64) ([]WebhookEndpoint, error)
	FindByUUIDAndTenantID(webhookEndpointUUID uuid.UUID, tenantID int64) (*WebhookEndpoint, error)
	FindPaginated(filter WebhookEndpointRepositoryGetFilter) (*PaginationResult[WebhookEndpoint], error)
	UpdateLastTriggeredAt(webhookEndpointID int64, t time.Time) error
}

type webhookEndpointRepository struct {
	*BaseRepository[WebhookEndpoint]
}

// NewWebhookEndpointRepository creates a new WebhookEndpointRepository backed
// by the given database connection.
func NewWebhookEndpointRepository(db *gorm.DB) WebhookEndpointRepository {
	return &webhookEndpointRepository{
		BaseRepository: database.NewBaseRepository[WebhookEndpoint](db, "webhook_endpoint_uuid", "webhook_endpoint_id"),
	}
}

// WithTx returns a copy of the repository bound to the supplied transaction.
func (r *webhookEndpointRepository) WithTx(tx *gorm.DB) WebhookEndpointRepository {
	return &webhookEndpointRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByTenantID retrieves all webhook endpoints belonging to a tenant.
func (r *webhookEndpointRepository) FindByTenantID(tenantID int64) ([]WebhookEndpoint, error) {
	var endpoints []WebhookEndpoint
	err := r.DB().Where("tenant_id = ?", tenantID).Find(&endpoints).Error
	if err != nil {
		return nil, err
	}
	return endpoints, nil
}

// FindByUUIDAndTenantID retrieves a single webhook endpoint by UUID scoped to
// a tenant. Returns nil, nil when no record exists.
func (r *webhookEndpointRepository) FindByUUIDAndTenantID(webhookEndpointUUID uuid.UUID, tenantID int64) (*WebhookEndpoint, error) {
	var endpoint WebhookEndpoint
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
func (r *webhookEndpointRepository) FindPaginated(filter WebhookEndpointRepositoryGetFilter) (*PaginationResult[WebhookEndpoint], error) {
	query := r.DB().Model(&WebhookEndpoint{})

	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
	}
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}

	query = query.Order(database.SanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	return database.PaginateQuery[WebhookEndpoint](query, filter.Page, filter.Limit)
}

// FindActiveByTenantID retrieves all active (non-deleted) webhook endpoints for a tenant.
func (r *webhookEndpointRepository) FindActiveByTenantID(tenantID int64) ([]WebhookEndpoint, error) {
	var endpoints []WebhookEndpoint
	err := r.DB().Where("tenant_id = ? AND status = ?", tenantID, "active").Find(&endpoints).Error
	if err != nil {
		return nil, err
	}
	return endpoints, nil
}

// UpdateLastTriggeredAt sets last_triggered_at for a webhook endpoint.
func (r *webhookEndpointRepository) UpdateLastTriggeredAt(webhookEndpointID int64, t time.Time) error {
	return r.DB().Model(&WebhookEndpoint{}).
		Where("webhook_endpoint_id = ?", webhookEndpointID).
		Update("last_triggered_at", t).Error
}

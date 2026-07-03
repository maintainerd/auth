package webhook

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	UpdateByUUID(uuid any, updatedData any) (*WebhookEndpoint, error)
	DeleteByUUID(uuid any) error
	FindByID(id any, preloads ...string) (*WebhookEndpoint, error)
	WithTx(tx *gorm.DB) WebhookEndpointRepository
	FindByTenantID(tenantID int64) ([]WebhookEndpoint, error)
	FindActiveByTenantID(tenantID int64) ([]WebhookEndpoint, error)
	FindByUUIDAndTenantID(webhookEndpointUUID uuid.UUID, tenantID int64) (*WebhookEndpoint, error)
	FindPaginated(filter WebhookEndpointRepositoryGetFilter) (*PaginationResult[WebhookEndpoint], error)
	UpdateLastTriggeredAt(webhookEndpointID int64, t time.Time) error
	// CountByTenantID returns the number of (non-deleted) endpoints for a tenant.
	CountByTenantID(tenantID int64) (int64, error)
	// IncrementConsecutiveFailures atomically bumps the failure counter and returns the new value.
	IncrementConsecutiveFailures(webhookEndpointID int64) (int, error)
	// ResetConsecutiveFailures zeroes the failure counter after a successful delivery.
	ResetConsecutiveFailures(webhookEndpointID int64) error
	// Quarantine marks an endpoint inactive after sustained failures.
	Quarantine(webhookEndpointID int64) error
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
	if filter.TenantID == nil || *filter.TenantID == 0 {
		return nil, fmt.Errorf("tenant_id is required")
	}

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
	err := r.DB().Where("tenant_id = ? AND status = ?", tenantID, shared.StatusActive).Find(&endpoints).Error
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

// CountByTenantID returns the number of non-deleted endpoints for a tenant.
func (r *webhookEndpointRepository) CountByTenantID(tenantID int64) (int64, error) {
	var count int64
	err := r.DB().Model(&WebhookEndpoint{}).Where("tenant_id = ?", tenantID).Count(&count).Error
	return count, err
}

// IncrementConsecutiveFailures atomically increments the failure counter and
// returns the new value, so callers can decide whether to quarantine.
func (r *webhookEndpointRepository) IncrementConsecutiveFailures(webhookEndpointID int64) (int, error) {
	var ep WebhookEndpoint
	err := r.DB().Model(&WebhookEndpoint{}).
		Clauses(clause.Returning{Columns: []clause.Column{{Name: "consecutive_failures"}}}).
		Where("webhook_endpoint_id = ?", webhookEndpointID).
		UpdateColumn("consecutive_failures", gorm.Expr("consecutive_failures + 1")).
		Scan(&ep).Error
	if err != nil {
		return 0, err
	}
	return ep.ConsecutiveFailures, nil
}

// ResetConsecutiveFailures zeroes the failure counter (called after a success).
func (r *webhookEndpointRepository) ResetConsecutiveFailures(webhookEndpointID int64) error {
	return r.DB().Model(&WebhookEndpoint{}).
		Where("webhook_endpoint_id = ? AND consecutive_failures > 0", webhookEndpointID).
		UpdateColumn("consecutive_failures", 0).Error
}

// Quarantine marks an endpoint inactive (quarantined) after sustained failures.
func (r *webhookEndpointRepository) Quarantine(webhookEndpointID int64) error {
	return r.DB().Model(&WebhookEndpoint{}).
		Where("webhook_endpoint_id = ?", webhookEndpointID).
		Update("status", "quarantined").Error
}

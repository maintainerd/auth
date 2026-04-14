package repository

import (
	"errors"
	"time"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

// AuthEventRepositoryGetFilter holds filter, sort, and pagination options
// for paginated auth event queries.
type AuthEventRepositoryGetFilter struct {
	TenantID     *int64
	ActorUserID  *int64
	TargetUserID *int64
	Category     *string
	EventType    *string
	Severity     *string
	Result       *string
	DateFrom     *time.Time
	DateTo       *time.Time
	SortBy       string
	SortOrder    string
	Page         int
	Limit        int
}

// AuthEventRepository defines persistence operations for auth events.
type AuthEventRepository interface {
	BaseRepositoryMethods[model.AuthEvent]
	WithTx(tx *gorm.DB) AuthEventRepository
	FindPaginated(filter AuthEventRepositoryGetFilter) (*PaginationResult[model.AuthEvent], error)
	FindByUUIDAndTenantID(uuid string, tenantID int64) (*model.AuthEvent, error)
	FindByDateRange(tenantID int64, from, to time.Time) ([]model.AuthEvent, error)
	DeleteOlderThan(cutoff time.Time) (int64, error)
	CountByEventType(eventType string, tenantID int64) (int64, error)
}

type authEventRepository struct {
	*BaseRepository[model.AuthEvent]
}

// NewAuthEventRepository creates a new AuthEventRepository backed by the supplied DB.
func NewAuthEventRepository(db *gorm.DB) AuthEventRepository {
	return &authEventRepository{
		BaseRepository: NewBaseRepository[model.AuthEvent](db, "auth_event_uuid", "auth_event_id"),
	}
}

// WithTx returns a copy of the repository bound to the given transaction.
func (r *authEventRepository) WithTx(tx *gorm.DB) AuthEventRepository {
	return &authEventRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindPaginated returns a page of auth events filtered by the supplied criteria.
func (r *authEventRepository) FindPaginated(filter AuthEventRepositoryGetFilter) (*PaginationResult[model.AuthEvent], error) {
	query := r.DB().Model(&model.AuthEvent{})

	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
	}
	if filter.ActorUserID != nil {
		query = query.Where("actor_user_id = ?", *filter.ActorUserID)
	}
	if filter.TargetUserID != nil {
		query = query.Where("target_user_id = ?", *filter.TargetUserID)
	}
	if filter.Category != nil && *filter.Category != "" {
		query = query.Where("category = ?", *filter.Category)
	}
	if filter.EventType != nil && *filter.EventType != "" {
		query = query.Where("event_type = ?", *filter.EventType)
	}
	if filter.Severity != nil && *filter.Severity != "" {
		query = query.Where("severity = ?", *filter.Severity)
	}
	if filter.Result != nil && *filter.Result != "" {
		query = query.Where("result = ?", *filter.Result)
	}
	if filter.DateFrom != nil {
		query = query.Where("created_at >= ?", *filter.DateFrom)
	}
	if filter.DateTo != nil {
		query = query.Where("created_at <= ?", *filter.DateTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	filter.Page, filter.Limit = normalizePagination(filter.Page, filter.Limit)
	offset := (filter.Page - 1) * filter.Limit

	var events []model.AuthEvent
	if err := query.Offset(offset).Limit(filter.Limit).Find(&events).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))
	return &PaginationResult[model.AuthEvent]{
		Data:       events,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

// FindByUUIDAndTenantID retrieves a single auth event by UUID scoped to a tenant.
func (r *authEventRepository) FindByUUIDAndTenantID(uuid string, tenantID int64) (*model.AuthEvent, error) {
	var event model.AuthEvent
	err := r.DB().
		Where("auth_event_uuid = ? AND tenant_id = ?", uuid, tenantID).
		First(&event).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &event, nil
}

// FindByDateRange returns all auth events within the given time range for a tenant.
func (r *authEventRepository) FindByDateRange(tenantID int64, from, to time.Time) ([]model.AuthEvent, error) {
	var events []model.AuthEvent
	err := r.DB().
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, from, to).
		Order("created_at DESC").
		Find(&events).Error
	return events, err
}

// DeleteOlderThan removes auth events older than the cutoff and returns the count deleted.
func (r *authEventRepository) DeleteOlderThan(cutoff time.Time) (int64, error) {
	result := r.DB().
		Where("created_at < ?", cutoff).
		Delete(&model.AuthEvent{})
	return result.RowsAffected, result.Error
}

// CountByEventType returns the number of events matching the event type within a tenant.
func (r *authEventRepository) CountByEventType(eventType string, tenantID int64) (int64, error) {
	var count int64
	err := r.DB().
		Model(&model.AuthEvent{}).
		Where("event_type = ? AND tenant_id = ?", eventType, tenantID).
		Count(&count).Error
	return count, err
}

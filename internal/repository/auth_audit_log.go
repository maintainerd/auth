package repository

import (
	"errors"
	"time"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

// AuthLogRepositoryGetFilter holds filter, sort, and pagination options
// for paginated auth log queries.
type AuthLogRepositoryGetFilter struct {
	TenantID  *int64
	UserID    *int64
	EventType *string
	DateFrom  *time.Time
	DateTo    *time.Time
	SortBy    string
	SortOrder string
	Page      int
	Limit     int
}

type AuthLogRepository interface {
	BaseRepositoryMethods[model.AuthLog]
	WithTx(tx *gorm.DB) AuthLogRepository
	FindPaginated(filter AuthLogRepositoryGetFilter) (*PaginationResult[model.AuthLog], error)
	FindByUUIDAndTenantID(uuid string, tenantID int64) (*model.AuthLog, error)
	FindByDateRange(tenantID int64, from, to time.Time) ([]model.AuthLog, error)
	DeleteOlderThan(cutoff time.Time) error
	CountByEventType(eventType string, tenantID int64) (int64, error)
}

type authLogRepository struct {
	*BaseRepository[model.AuthLog]
}

func NewAuthLogRepository(db *gorm.DB) AuthLogRepository {
	return &authLogRepository{
		BaseRepository: NewBaseRepository[model.AuthLog](db, "auth_log_uuid", "auth_log_id"),
	}
}

func (r *authLogRepository) WithTx(tx *gorm.DB) AuthLogRepository {
	return &authLogRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindPaginated returns a page of auth logs filtered by the supplied criteria.
// Replaces the former FindByUserID(userID, limit, offset) and
// FindByEventType(eventType, tenantID, limit, offset) raw-parameter methods.
func (r *authLogRepository) FindPaginated(filter AuthLogRepositoryGetFilter) (*PaginationResult[model.AuthLog], error) {
	query := r.DB().Model(&model.AuthLog{})

	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
	}
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.EventType != nil && *filter.EventType != "" {
		query = query.Where("event_type = ?", *filter.EventType)
	}
	if filter.DateFrom != nil {
		query = query.Where("created_at >= ?", *filter.DateFrom)
	}
	if filter.DateTo != nil {
		query = query.Where("created_at <= ?", *filter.DateTo)
	}

	// Count before pagination
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	// Pagination
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 20
	}
	offset := (filter.Page - 1) * filter.Limit

	var logs []model.AuthLog
	if err := query.Offset(offset).Limit(filter.Limit).Find(&logs).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))
	return &PaginationResult[model.AuthLog]{
		Data:       logs,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

// FindByUUIDAndTenantID retrieves an auth log by UUID and tenant ID.
func (r *authLogRepository) FindByUUIDAndTenantID(uuid string, tenantID int64) (*model.AuthLog, error) {
	var log model.AuthLog
	err := r.DB().
		Where("auth_log_uuid = ? AND tenant_id = ?", uuid, tenantID).
		First(&log).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &log, nil
}

func (r *authLogRepository) FindByDateRange(tenantID int64, from, to time.Time) ([]model.AuthLog, error) {
	var logs []model.AuthLog
	err := r.DB().
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, from, to).
		Order("created_at DESC").
		Find(&logs).Error
	return logs, err
}

func (r *authLogRepository) DeleteOlderThan(cutoff time.Time) error {
	return r.DB().
		Where("created_at < ?", cutoff).
		Delete(&model.AuthLog{}).Error
}

func (r *authLogRepository) CountByEventType(eventType string, tenantID int64) (int64, error) {
	var count int64
	err := r.DB().
		Model(&model.AuthLog{}).
		Where("event_type = ? AND tenant_id = ?", eventType, tenantID).
		Count(&count).Error
	return count, err
}

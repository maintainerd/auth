package authevent

import (
	"errors"
	"fmt"
	"time"

	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

// AuthEventRepositoryGetFilter holds filter, sort, and pagination options
// for paginated auth event queries.
type AuthEventRepositoryGetFilter struct {
	TenantID     *int64
	ActorUserID  *int64
	TargetUserID *int64
	UserUUID     *string
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
	Cursor       *int64
}

// AuthEventRepository defines persistence operations for auth events.
type AuthEventRepository interface {
	BaseRepositoryMethods[AuthEvent]
	UpdateByUUID(uuid any, updatedData any) (*AuthEvent, error)
	UpdateByID(id any, updatedData any) (*AuthEvent, error)
	DeleteByUUID(uuid any) error
	DeleteByID(id any) error
	WithTx(tx *gorm.DB) AuthEventRepository
	FindPaginated(filter AuthEventRepositoryGetFilter) (*PaginationResult[AuthEvent], error)
	FindByUUIDAndTenantID(uuid string, tenantID int64) (*AuthEvent, error)
	FindByDateRange(tenantID int64, from, to time.Time) ([]AuthEvent, error)
	DeleteOlderThan(cutoff time.Time) (int64, error)
	DeleteExpiredByAuditConfig(now time.Time, defaultRetentionDays int) (int64, error)
	CountByEventType(eventType string, tenantID int64) (int64, error)
}

type authEventRepository struct {
	*BaseRepository[AuthEvent]
}

// NewAuthEventRepository creates a new AuthEventRepository backed by the supplied DB.
func NewAuthEventRepository(db *gorm.DB) AuthEventRepository {
	return &authEventRepository{
		BaseRepository: database.NewBaseRepository[AuthEvent](db, "auth_event_uuid", "auth_event_id"),
	}
}

// WithTx returns a copy of the repository bound to the given transaction.
func (r *authEventRepository) WithTx(tx *gorm.DB) AuthEventRepository {
	return &authEventRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *authEventRepository) CreateOrUpdate(_ *AuthEvent) (*AuthEvent, error) {
	return nil, errors.New("auth events are append-only; use Create")
}

func (r *authEventRepository) UpdateByUUID(_ any, _ any) (*AuthEvent, error) {
	return nil, errors.New("auth events are immutable and cannot be updated")
}

func (r *authEventRepository) UpdateByID(_ any, _ any) (*AuthEvent, error) {
	return nil, errors.New("auth events are immutable and cannot be updated")
}

func (r *authEventRepository) DeleteByUUID(_ any) error {
	return errors.New("auth events can only be deleted by retention or tenant deletion")
}

func (r *authEventRepository) DeleteByID(_ any) error {
	return errors.New("auth events can only be deleted by retention or tenant deletion")
}

// FindPaginated returns a page of auth events filtered by the supplied criteria.
func (r *authEventRepository) FindPaginated(filter AuthEventRepositoryGetFilter) (*PaginationResult[AuthEvent], error) {
	if filter.TenantID == nil || *filter.TenantID == 0 {
		return nil, fmt.Errorf("tenant_id is required")
	}

	query := r.DB().Model(&AuthEvent{})

	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
	}
	if filter.ActorUserID != nil {
		query = query.Where("actor_user_id = ?", *filter.ActorUserID)
	}
	if filter.TargetUserID != nil {
		query = query.Where("target_user_id = ?", *filter.TargetUserID)
	}
	if filter.UserUUID != nil && *filter.UserUUID != "" {
		// Events where this user is either the actor or the subject.
		query = query.Where(
			"actor_user_id = (SELECT user_id FROM users WHERE user_uuid = ?) OR target_user_id = (SELECT user_id FROM users WHERE user_uuid = ?)",
			*filter.UserUUID, *filter.UserUUID,
		)
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

	query = query.Order(database.SanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	afterID := int64(0)
	if filter.Cursor != nil {
		afterID = *filter.Cursor
	}
	rows, nextCursor, err := database.PaginateKeyset[AuthEvent](query, afterID, filter.Limit, "auth_event_id", func(e AuthEvent) int64 { return e.AuthEventID })
	if err != nil {
		return nil, err
	}

	total, err := r.estimatedCount(query)
	if err != nil {
		return nil, err
	}

	return &PaginationResult[AuthEvent]{
		Data:       rows,
		Total:      total,
		Limit:      filter.Limit,
		NextCursor: nextCursor,
	}, nil
}

func (r *authEventRepository) estimatedCount(query *gorm.DB) (int64, error) {
	session := query.Session(&gorm.Session{NewDB: true})
	var total int64
	if err := session.Model(&AuthEvent{}).Select("COALESCE(reltuples::bigint, 0)").Table("pg_class").Where("relname = 'auth_events'").Scan(&total).Error; err != nil {
		return 0, nil
	}
	return total, nil
}

// FindByUUIDAndTenantID retrieves a single auth event by UUID scoped to a tenant.
func (r *authEventRepository) FindByUUIDAndTenantID(uuid string, tenantID int64) (*AuthEvent, error) {
	var event AuthEvent
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
func (r *authEventRepository) FindByDateRange(tenantID int64, from, to time.Time) ([]AuthEvent, error) {
	var events []AuthEvent
	err := r.DB().
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, from, to).
		Order("created_at DESC").
		Find(&events).Error
	return events, err
}

// DeleteOlderThan removes auth events older than the cutoff and returns the count deleted.
func (r *authEventRepository) DeleteOlderThan(cutoff time.Time) (int64, error) {
	var rowsAffected int64
	err := r.DB().Transaction(func(tx *gorm.DB) error {
		if err := allowAuthEventDelete(tx, "retention"); err != nil {
			return err
		}
		result := tx.
			Where("created_at < ?", cutoff).
			Delete(&AuthEvent{})
		rowsAffected = result.RowsAffected
		return result.Error
	})
	return rowsAffected, err
}

// DeleteExpiredByAuditConfig removes auth events according to each tenant's
// tenant_settings.audit_config.retention_days value. Missing/invalid values use
// the provided default retention.
func (r *authEventRepository) DeleteExpiredByAuditConfig(now time.Time, defaultRetentionDays int) (int64, error) {
	var rowsAffected int64
	err := r.DB().Transaction(func(tx *gorm.DB) error {
		if err := allowAuthEventDelete(tx, "retention"); err != nil {
			return err
		}
		result := tx.Exec(`
			DELETE FROM auth_events
			WHERE auth_events.created_at < ? - (
				  GREATEST(
					  COALESCE(
						  (
							  SELECT CASE
								  WHEN tenant_settings.audit_config->>'retention_days' ~ '^[0-9]+$'
									  THEN (tenant_settings.audit_config->>'retention_days')::integer
								  ELSE NULL
							  END
							  FROM tenant_settings
							  WHERE tenant_settings.tenant_id = auth_events.tenant_id
						  ),
						  ?
					  ),
					  1
				  ) * INTERVAL '1 day'
			  )
		`, now, defaultRetentionDays)
		rowsAffected = result.RowsAffected
		return result.Error
	})
	return rowsAffected, err
}

func allowAuthEventDelete(tx *gorm.DB, reason string) error {
	return tx.Exec("SELECT set_config('maintainerd.allow_auth_event_delete', ?, true)", reason).Error
}

// CountByEventType returns the number of events matching the event type within a tenant.
func (r *authEventRepository) CountByEventType(eventType string, tenantID int64) (int64, error) {
	var count int64
	err := r.DB().
		Model(&AuthEvent{}).
		Where("event_type = ? AND tenant_id = ?", eventType, tenantID).
		Count(&count).Error
	return count, err
}

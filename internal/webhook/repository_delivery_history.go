package webhook

import (
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

// DeliveryHistoryRepository defines persistence for webhook_delivery_history.
type DeliveryHistoryRepository interface {
	BaseRepositoryMethods[DeliveryHistory]
	FindByEventID(eventID uuid.UUID, endpointID int64) ([]DeliveryHistory, error)
	FindPendingRetries() ([]DeliveryHistory, error)
	FindByEndpointID(endpointID int64, limit int) ([]DeliveryHistory, error)
	FindByTenantID(tenantID int64, page, limit int) (*PaginationResult[DeliveryHistory], error)
	MoveToDeadLetter(deliveryHistoryID int64, errorReason string) error
	UpdateAttempt(deliveryHistoryID int64, attemptCount int, responseStatus *int, responseSummary string, errorReason string, nextRetry *time.Time, finalStatus string) error
	DeleteOlderThan(cutoff time.Time) (int64, error)
	DeleteBySubjectUUID(subjectUUID uuid.UUID) (int64, error)
	WithTx(tx *gorm.DB) DeliveryHistoryRepository
}

type deliveryHistoryRepository struct {
	*BaseRepository[DeliveryHistory]
}

func NewDeliveryHistoryRepository(db *gorm.DB) DeliveryHistoryRepository {
	return &deliveryHistoryRepository{
		BaseRepository: database.NewBaseRepository[DeliveryHistory](db, "delivery_history_uuid", "delivery_history_id"),
	}
}

func (r *deliveryHistoryRepository) WithTx(tx *gorm.DB) DeliveryHistoryRepository {
	return &deliveryHistoryRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *deliveryHistoryRepository) FindByEventID(eventID uuid.UUID, endpointID int64) ([]DeliveryHistory, error) {
	var history []DeliveryHistory
	query := r.DB().Where("event_id = ?", eventID)
	if endpointID > 0 {
		query = query.Where("webhook_endpoint_id = ?", endpointID)
	}
	err := query.Order("created_at DESC").Find(&history).Error
	return history, err
}

func (r *deliveryHistoryRepository) FindPendingRetries() ([]DeliveryHistory, error) {
	var history []DeliveryHistory
	err := r.DB().Where("final_status = ? AND next_retry_time <= ?", "pending", time.Now().UTC()).
		Order("next_retry_time ASC").
		Limit(100).
		Find(&history).Error
	return history, err
}

func (r *deliveryHistoryRepository) FindByEndpointID(endpointID int64, limit int) ([]DeliveryHistory, error) {
	var history []DeliveryHistory
	err := r.DB().Where("webhook_endpoint_id = ?", endpointID).
		Order("created_at DESC").
		Limit(limit).
		Find(&history).Error
	return history, err
}

func (r *deliveryHistoryRepository) FindByTenantID(tenantID int64, page, limit int) (*PaginationResult[DeliveryHistory], error) {
	query := r.DB().Model(&DeliveryHistory{}).Where("tenant_id = ?", tenantID).Order("created_at DESC")
	return database.PaginateQuery[DeliveryHistory](query, page, limit)
}

func (r *deliveryHistoryRepository) MoveToDeadLetter(deliveryHistoryID int64, errorReason string) error {
	return r.DB().Model(&DeliveryHistory{}).
		Where("delivery_history_id = ?", deliveryHistoryID).
		Updates(map[string]any{
			"final_status": "dead_letter",
			"error_reason": errorReason,
			"updated_at":   time.Now().UTC(),
		}).Error
}

func (r *deliveryHistoryRepository) UpdateAttempt(deliveryHistoryID int64, attemptCount int, responseStatus *int, responseSummary string, errorReason string, nextRetry *time.Time, finalStatus string) error {
	updates := map[string]any{
		"attempt_count":    attemptCount,
		"response_status":  responseStatus,
		"response_summary": responseSummary,
		"error_reason":     errorReason,
		"final_status":     finalStatus,
		"updated_at":       time.Now().UTC(),
	}
	if nextRetry != nil {
		updates["next_retry_time"] = nextRetry
	}
	return r.DB().Model(&DeliveryHistory{}).
		Where("delivery_history_id = ?", deliveryHistoryID).
		Updates(updates).Error
}

func (r *deliveryHistoryRepository) DeleteOlderThan(cutoff time.Time) (int64, error) {
	result := r.DB().Where("created_at < ? AND final_status IN ?", cutoff, []string{"success", "dead_letter"}).Delete(&DeliveryHistory{})
	return result.RowsAffected, result.Error
}

func (r *deliveryHistoryRepository) DeleteBySubjectUUID(subjectUUID uuid.UUID) (int64, error) {
	result := r.DB().Where("event_id IN (SELECT event_id FROM integration_event_outbox WHERE subject_uuid = ?)", subjectUUID).Delete(&DeliveryHistory{})
	return result.RowsAffected, result.Error
}

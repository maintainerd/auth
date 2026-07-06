package user

import (
	"errors"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

// DataErasureRequestRepository defines persistence operations for the
// data_erasure_requests entity.
type DataErasureRequestRepository interface {
	BaseRepositoryMethods[DataErasureRequest]
	WithTx(tx *gorm.DB) DataErasureRequestRepository
	// FindActiveByUserID returns the most recent pending/in_progress request for
	// a user (used to make erasure requests idempotent). Returns nil, nil when
	// none exists.
	FindActiveByUserID(userID int64) (*DataErasureRequest, error)
	// FindDueForProcessing returns pending requests whose scheduled_at has passed
	// and which are not under legal hold, up to limit rows.
	FindDueForProcessing(now time.Time, limit int) ([]DataErasureRequest, error)
	// MarkInProgress flips a request to in_progress and records started_at.
	MarkInProgress(id int64, startedAt time.Time) error
	// MarkCompleted flips a request to completed and records completed_at.
	MarkCompleted(id int64, completedAt time.Time) error
	// MarkPending reverts a request to pending (used when processing fails so the
	// worker retries on the next tick). 'failed' is intentionally not used — it
	// is not an allowed status per chk_data_erasure_requests_status.
	MarkPending(id int64) error
}

type dataErasureRequestRepository struct {
	*BaseRepository[DataErasureRequest]
}

// NewDataErasureRequestRepository creates a new repository backed by db.
func NewDataErasureRequestRepository(db *gorm.DB) DataErasureRequestRepository {
	return &dataErasureRequestRepository{
		BaseRepository: database.NewBaseRepository[DataErasureRequest](
			db, "data_erasure_request_uuid", "data_erasure_request_id",
		),
	}
}

// WithTx returns a copy of the repository bound to the supplied transaction.
func (r *dataErasureRequestRepository) WithTx(tx *gorm.DB) DataErasureRequestRepository {
	return &dataErasureRequestRepository{BaseRepository: r.BaseRepository.WithTx(tx)}
}

func (r *dataErasureRequestRepository) FindActiveByUserID(userID int64) (*DataErasureRequest, error) {
	var req DataErasureRequest
	err := r.DB().
		Where("user_id = ? AND status IN ?", userID, []string{"pending", "in_progress"}).
		Order("created_at DESC").
		First(&req).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &req, nil
}

func (r *dataErasureRequestRepository) FindDueForProcessing(now time.Time, limit int) ([]DataErasureRequest, error) {
	var requests []DataErasureRequest
	err := r.DB().
		Where("status = ? AND scheduled_at <= ? AND legal_hold = ?", "pending", now, false).
		Order("scheduled_at ASC").
		Limit(limit).
		Find(&requests).Error
	if err != nil {
		return nil, err
	}
	return requests, nil
}

func (r *dataErasureRequestRepository) MarkInProgress(id int64, startedAt time.Time) error {
	return r.DB().Model(&DataErasureRequest{}).
		Where("data_erasure_request_id = ?", id).
		Updates(map[string]any{"status": "in_progress", "started_at": startedAt}).Error
}

func (r *dataErasureRequestRepository) MarkCompleted(id int64, completedAt time.Time) error {
	return r.DB().Model(&DataErasureRequest{}).
		Where("data_erasure_request_id = ?", id).
		Updates(map[string]any{"status": "completed", "completed_at": completedAt}).Error
}

func (r *dataErasureRequestRepository) MarkPending(id int64) error {
	return r.DB().Model(&DataErasureRequest{}).
		Where("data_erasure_request_id = ?", id).
		Update("status", "pending").Error
}

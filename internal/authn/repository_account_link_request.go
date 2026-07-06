package authn

import (
	"errors"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

// AccountLinkRequestRepository defines persistence operations for the
// account_link_requests entity.
type AccountLinkRequestRepository interface {
	BaseRepositoryMethods[AccountLinkRequest]
	WithTx(tx *gorm.DB) AccountLinkRequestRepository
	// FindByToken looks up a single request by its confirmation_token regardless
	// of status (used by the confirm handler + Initiate collision check).
	FindByToken(token string) (*AccountLinkRequest, error)
	// MarkConfirmed flips a pending request to confirmed and records confirmed_at.
	MarkConfirmed(id int64, at time.Time) error
	// MarkExpired flips a pending request to expired and records rejected_at.
	MarkExpired(id int64, at time.Time) error
	// ExpireStale marks pending requests whose expires_at has passed as expired.
	ExpireStale(now time.Time) (int64, error)
}

type accountLinkRequestRepository struct {
	*BaseRepository[AccountLinkRequest]
}

// NewAccountLinkRequestRepository creates a new repository backed by db.
func NewAccountLinkRequestRepository(db *gorm.DB) AccountLinkRequestRepository {
	return &accountLinkRequestRepository{
		BaseRepository: database.NewBaseRepository[AccountLinkRequest](
			db, "account_link_request_uuid", "account_link_request_id",
		),
	}
}

// WithTx returns a copy of the repository bound to the supplied transaction.
func (r *accountLinkRequestRepository) WithTx(tx *gorm.DB) AccountLinkRequestRepository {
	return &accountLinkRequestRepository{BaseRepository: r.BaseRepository.WithTx(tx)}
}

func (r *accountLinkRequestRepository) FindByToken(token string) (*AccountLinkRequest, error) {
	var req AccountLinkRequest
	err := r.DB().Where("confirmation_token = ?", token).First(&req).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &req, nil
}

func (r *accountLinkRequestRepository) ExpireStale(now time.Time) (int64, error) {
	result := r.DB().Model(&AccountLinkRequest{}).
		Where("status = ? AND expires_at < ?", "pending", now).
		Update("status", "expired")
	return result.RowsAffected, result.Error
}

func (r *accountLinkRequestRepository) MarkConfirmed(id int64, at time.Time) error {
	return r.DB().Model(&AccountLinkRequest{}).
		Where("account_link_request_id = ?", id).
		Updates(map[string]any{"status": "confirmed", "confirmed_at": at}).Error
}

func (r *accountLinkRequestRepository) MarkExpired(id int64, at time.Time) error {
	return r.DB().Model(&AccountLinkRequest{}).
		Where("account_link_request_id = ?", id).
		Updates(map[string]any{"status": "expired", "rejected_at": at}).Error
}

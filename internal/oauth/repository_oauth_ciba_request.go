package oauth

import (
	"errors"
	"time"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

// OAuthCIBARequestRepository defines data access operations for
// Client-Initiated Backchannel Authentication requests.
type OAuthCIBARequestRepository interface {
	BaseRepositoryMethods[model.OAuthCIBARequest]
	WithTx(tx *gorm.DB) OAuthCIBARequestRepository
	FindByAuthReqIDHash(hash string) (*model.OAuthCIBARequest, error)
	UpdateStatus(id int64, status string) error
	UpdateApproval(id int64, userID int64) error
	UpdateLastPollAt(id int64) error
	MarkNotificationSent(id int64) error
	DeleteExpired(before time.Time) (int64, error)
}

type oauthCIBARequestRepository struct {
	*BaseRepository[model.OAuthCIBARequest]
}

// NewOAuthCIBARequestRepository creates a new OAuthCIBARequestRepository.
func NewOAuthCIBARequestRepository(db *gorm.DB) OAuthCIBARequestRepository {
	return &oauthCIBARequestRepository{
		BaseRepository: NewBaseRepository[model.OAuthCIBARequest](db, "oauth_ciba_request_uuid", "oauth_ciba_request_id"),
	}
}

func (r *oauthCIBARequestRepository) WithTx(tx *gorm.DB) OAuthCIBARequestRepository {
	return &oauthCIBARequestRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByAuthReqIDHash looks up a CIBA request by the SHA-256 hash of the
// auth_req_id. Returns nil, nil when not found.
func (r *oauthCIBARequestRepository) FindByAuthReqIDHash(hash string) (*model.OAuthCIBARequest, error) {
	var req model.OAuthCIBARequest
	err := r.DB().
		Preload("Client").
		Preload("Client.IdentityProvider").
		Preload("User").
		Where("auth_req_id_hash = ?", hash).
		First(&req).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &req, nil
}

// UpdateStatus sets the status on a CIBA request.
func (r *oauthCIBARequestRepository) UpdateStatus(id int64, status string) error {
	return r.DB().Model(&model.OAuthCIBARequest{}).
		Where("oauth_ciba_request_id = ?", id).
		Update("status", status).Error
}

// UpdateApproval sets status=approved and records the approving user.
func (r *oauthCIBARequestRepository) UpdateApproval(id int64, userID int64) error {
	return r.DB().Model(&model.OAuthCIBARequest{}).
		Where("oauth_ciba_request_id = ?", id).
		Updates(map[string]any{
			"status":  model.CIBAStatusApproved,
			"user_id": userID,
		}).Error
}

// UpdateLastPollAt records when the client last polled.
func (r *oauthCIBARequestRepository) UpdateLastPollAt(id int64) error {
	now := time.Now()
	return r.DB().Model(&model.OAuthCIBARequest{}).
		Where("oauth_ciba_request_id = ?", id).
		Update("last_poll_at", now).Error
}

// MarkNotificationSent sets the notification_sent_at timestamp.
func (r *oauthCIBARequestRepository) MarkNotificationSent(id int64) error {
	now := time.Now()
	return r.DB().Model(&model.OAuthCIBARequest{}).
		Where("oauth_ciba_request_id = ?", id).
		Update("notification_sent_at", now).Error
}

// DeleteExpired removes CIBA requests that expired before the given cutoff.
func (r *oauthCIBARequestRepository) DeleteExpired(before time.Time) (int64, error) {
	result := r.DB().
		Where("expires_at < ?", before).
		Delete(&model.OAuthCIBARequest{})
	return result.RowsAffected, result.Error
}

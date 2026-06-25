package oauth

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

// OAuthCIBARequestRepository defines data access operations for
// Client-Initiated Backchannel Authentication requests.
type OAuthCIBARequestRepository interface {
	BaseRepositoryMethods[OAuthCIBARequest]
	WithTx(tx *gorm.DB) OAuthCIBARequestRepository
	FindByAuthReqIDHash(hash string) (*OAuthCIBARequest, error)
	UpdateStatus(id int64, status string) error
	UpdateApproval(id int64, userID int64) error
	UpdateApprovalContext(id int64, userID int64, acr string, amr []string) error
	UpdateLastPollAt(id int64) error
	MarkNotificationSent(id int64) error
	DeleteExpired(before time.Time) (int64, error)
}

type oauthCIBARequestRepository struct {
	*BaseRepository[OAuthCIBARequest]
}

// NewOAuthCIBARequestRepository creates a new OAuthCIBARequestRepository.
func NewOAuthCIBARequestRepository(db *gorm.DB) OAuthCIBARequestRepository {
	return &oauthCIBARequestRepository{
		BaseRepository: database.NewBaseRepository[OAuthCIBARequest](db, "oauth_ciba_request_uuid", "oauth_ciba_request_id"),
	}
}

func (r *oauthCIBARequestRepository) WithTx(tx *gorm.DB) OAuthCIBARequestRepository {
	return &oauthCIBARequestRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByAuthReqIDHash looks up a CIBA request by the SHA-256 hash of the
// auth_req_id. Returns nil, nil when not found.
func (r *oauthCIBARequestRepository) FindByAuthReqIDHash(hash string) (*OAuthCIBARequest, error) {
	var req OAuthCIBARequest
	err := r.DB().
		Preload("Client").
		Preload("Client.Tenant").
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
	return r.DB().Model(&OAuthCIBARequest{}).
		Where("oauth_ciba_request_id = ?", id).
		Update("status", status).Error
}

// UpdateApproval sets status=approved and records the approving user.
func (r *oauthCIBARequestRepository) UpdateApproval(id int64, userID int64) error {
	return r.UpdateApprovalContext(id, userID, "", nil)
}

func (r *oauthCIBARequestRepository) UpdateApprovalContext(id int64, userID int64, acr string, amr []string) error {
	updates := map[string]any{
		"status":  CIBAStatusApproved,
		"user_id": userID,
	}
	if acr != "" {
		updates["auth_acr"] = acr
	}
	if len(amr) > 0 {
		amrJSON, _ := json.Marshal(amr)
		updates["auth_amr"] = amrJSON
	}
	return r.DB().Model(&OAuthCIBARequest{}).
		Where("oauth_ciba_request_id = ?", id).
		Updates(updates).Error
}

// UpdateLastPollAt records when the client last polled.
func (r *oauthCIBARequestRepository) UpdateLastPollAt(id int64) error {
	now := time.Now()
	return r.DB().Model(&OAuthCIBARequest{}).
		Where("oauth_ciba_request_id = ?", id).
		Update("last_poll_at", now).Error
}

// MarkNotificationSent sets the notification_sent_at timestamp.
func (r *oauthCIBARequestRepository) MarkNotificationSent(id int64) error {
	now := time.Now()
	return r.DB().Model(&OAuthCIBARequest{}).
		Where("oauth_ciba_request_id = ?", id).
		Update("notification_sent_at", now).Error
}

// DeleteExpired removes CIBA requests that expired before the given cutoff.
func (r *oauthCIBARequestRepository) DeleteExpired(before time.Time) (int64, error) {
	result := r.DB().
		Where("expires_at < ?", before).
		Delete(&OAuthCIBARequest{})
	return result.RowsAffected, result.Error
}

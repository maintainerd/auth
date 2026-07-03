package oauth

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

type OAuthAuthorizeRequestRepository interface {
	BaseRepositoryMethods[OAuthAuthorizeRequest]
	WithTx(tx *gorm.DB) OAuthAuthorizeRequestRepository
	FindByUUID(uuid uuid.UUID) (*OAuthAuthorizeRequest, error)
	Consume(id int64, tenantID int64, at time.Time) error
	DeleteExpired(before time.Time) (int64, error)
}

type oauthAuthorizeRequestRepository struct {
	*BaseRepository[OAuthAuthorizeRequest]
}

func NewOAuthAuthorizeRequestRepository(db *gorm.DB) OAuthAuthorizeRequestRepository {
	return &oauthAuthorizeRequestRepository{
		BaseRepository: database.NewBaseRepository[OAuthAuthorizeRequest](db, "oauth_authorize_request_uuid", "oauth_authorize_request_id"),
	}
}

func (r *oauthAuthorizeRequestRepository) WithTx(tx *gorm.DB) OAuthAuthorizeRequestRepository {
	return &oauthAuthorizeRequestRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *oauthAuthorizeRequestRepository) FindByUUID(id uuid.UUID) (*OAuthAuthorizeRequest, error) {
	var req OAuthAuthorizeRequest
	err := r.DB().
		Where("oauth_authorize_request_uuid = ? AND status = ? AND consumed_at IS NULL", id, "pending").
		First(&req).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &req, nil
}

func (r *oauthAuthorizeRequestRepository) Consume(id int64, tenantID int64, at time.Time) error {
	result := r.DB().Model(&OAuthAuthorizeRequest{}).
		Where("oauth_authorize_request_id = ? AND status = ? AND consumed_at IS NULL", id, "pending").
		Updates(map[string]any{
			"tenant_id":   tenantID,
			"status":      "consumed",
			"consumed_at": at,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAlreadyConsumed
	}
	return nil
}

func (r *oauthAuthorizeRequestRepository) DeleteExpired(before time.Time) (int64, error) {
	var total int64
	for {
		result := r.DB().
			Where("expires_at < ?", before).
			Limit(10000).
			Delete(&OAuthAuthorizeRequest{})
		if result.Error != nil {
			return total, result.Error
		}
		if result.RowsAffected == 0 {
			break
		}
		total += result.RowsAffected
	}
	return total, nil
}

var ErrAuthorizeRequestAlreadyConsumed = errors.New("authorize request already consumed")

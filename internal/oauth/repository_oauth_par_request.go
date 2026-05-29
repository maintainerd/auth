package oauth

import (
	"errors"
	"time"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

// OAuthPARRequestRepository defines data access operations for Pushed
// Authorization Requests (RFC 9126).
type OAuthPARRequestRepository interface {
	BaseRepositoryMethods[model.OAuthPARRequest]
	WithTx(tx *gorm.DB) OAuthPARRequestRepository
	FindByRequestURIHash(hash string) (*model.OAuthPARRequest, error)
	MarkUsed(id int64) error
	DeleteExpired(before time.Time) (int64, error)
}

type oauthPARRequestRepository struct {
	*BaseRepository[model.OAuthPARRequest]
}

// NewOAuthPARRequestRepository creates a new OAuthPARRequestRepository.
func NewOAuthPARRequestRepository(db *gorm.DB) OAuthPARRequestRepository {
	return &oauthPARRequestRepository{
		BaseRepository: NewBaseRepository[model.OAuthPARRequest](db, "oauth_par_request_uuid", "oauth_par_request_id"),
	}
}

func (r *oauthPARRequestRepository) WithTx(tx *gorm.DB) OAuthPARRequestRepository {
	return &oauthPARRequestRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByRequestURIHash looks up a PAR request by the SHA-256 hash of its
// request_uri token. Returns nil, nil when no matching record exists.
func (r *oauthPARRequestRepository) FindByRequestURIHash(hash string) (*model.OAuthPARRequest, error) {
	var req model.OAuthPARRequest
	err := r.DB().
		Preload("Client").
		Preload("Client.ClientURIs").
		Where("request_uri_hash = ? AND is_used = false", hash).
		First(&req).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &req, nil
}

// MarkUsed marks a PAR request as consumed so it cannot be replayed.
func (r *oauthPARRequestRepository) MarkUsed(id int64) error {
	return r.DB().Model(&model.OAuthPARRequest{}).
		Where("oauth_par_request_id = ?", id).
		Update("is_used", true).Error
}

// DeleteExpired removes PAR requests that expired before the given cutoff.
func (r *oauthPARRequestRepository) DeleteExpired(before time.Time) (int64, error) {
	result := r.DB().
		Where("expires_at < ?", before).
		Delete(&model.OAuthPARRequest{})
	return result.RowsAffected, result.Error
}

package oauth

import (
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

type oauthTokenRevocationRepository struct {
	*BaseRepository[OAuthTokenRevocation]
}

func NewOAuthTokenRevocationRepository(db *gorm.DB) OAuthTokenRevocationRepository {
	return &oauthTokenRevocationRepository{
		BaseRepository: database.NewBaseRepository[OAuthTokenRevocation](db, "oauth_token_revocation_uuid", "oauth_token_revocation_id"),
	}
}

func (r *oauthTokenRevocationRepository) Revoke(revocation *OAuthTokenRevocation) error {
	return r.DB().Create(revocation).Error
}

func (r *oauthTokenRevocationRepository) IsRevoked(tenantID int64, jti string) (bool, error) {
	var count int64
	err := r.DB().Model(&OAuthTokenRevocation{}).
		Where("tenant_id = ? AND jti = ? AND expires_at > ?", tenantID, jti, time.Now()).
		Count(&count).Error
	return count > 0, err
}

func (r *oauthTokenRevocationRepository) DeleteExpired() (int64, error) {
	result := r.DB().
		Where("expires_at < ?", time.Now()).
		Delete(&OAuthTokenRevocation{})
	return result.RowsAffected, result.Error
}

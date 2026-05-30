package oauth

import (
	"errors"
	"time"

	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

// OAuthAuthorizationCodeRepository defines data access operations for
// authorization codes.
type OAuthAuthorizationCodeRepository interface {
	BaseRepositoryMethods[OAuthAuthorizationCode]
	WithTx(tx *gorm.DB) OAuthAuthorizationCodeRepository
	FindByCodeHash(codeHash string) (*OAuthAuthorizationCode, error)
	MarkUsed(codeID int64) error
	DeleteExpired(before time.Time) (int64, error)
}

type oauthAuthorizationCodeRepository struct {
	*BaseRepository[OAuthAuthorizationCode]
}

// NewOAuthAuthorizationCodeRepository creates a new OAuthAuthorizationCodeRepository.
func NewOAuthAuthorizationCodeRepository(db *gorm.DB) OAuthAuthorizationCodeRepository {
	return &oauthAuthorizationCodeRepository{
		BaseRepository: database.NewBaseRepository[OAuthAuthorizationCode](db, "oauth_authorization_code_uuid", "oauth_authorization_code_id"),
	}
}

func (r *oauthAuthorizationCodeRepository) WithTx(tx *gorm.DB) OAuthAuthorizationCodeRepository {
	return &oauthAuthorizationCodeRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByCodeHash looks up an authorization code by its SHA-256 hash.
// Returns nil, nil when no matching code exists.
func (r *oauthAuthorizationCodeRepository) FindByCodeHash(codeHash string) (*OAuthAuthorizationCode, error) {
	var code OAuthAuthorizationCode
	err := r.DB().
		Preload("Client").
		Preload("Client.IdentityProvider").
		Where("code_hash = ?", codeHash).
		First(&code).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &code, nil
}

// MarkUsed marks an authorization code as consumed so it cannot be reused.
func (r *oauthAuthorizationCodeRepository) MarkUsed(codeID int64) error {
	now := time.Now()
	return r.DB().Model(&OAuthAuthorizationCode{}).
		Where("oauth_authorization_code_id = ?", codeID).
		Updates(map[string]any{
			"is_used": true,
			"used_at": now,
		}).Error
}

// DeleteExpired removes authorization codes that expired before the given
// cutoff time. Returns the number of rows deleted.
func (r *oauthAuthorizationCodeRepository) DeleteExpired(before time.Time) (int64, error) {
	result := r.DB().
		Where("expires_at < ?", before).
		Delete(&OAuthAuthorizationCode{})
	return result.RowsAffected, result.Error
}

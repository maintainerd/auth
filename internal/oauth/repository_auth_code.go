package oauth

import (
	"errors"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
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
		Preload("Client.Tenant").
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
// The WHERE clause includes used = false so a concurrent redemption attempt
// finds 0 rows affected and receives an error rather than silently succeeding.
func (r *oauthAuthorizationCodeRepository) MarkUsed(codeID int64) error {
	now := time.Now()
	result := r.DB().Model(&OAuthAuthorizationCode{}).
		Where("oauth_authorization_code_id = ? AND used = false", codeID).
		Updates(map[string]any{
			"used":    true,
			"used_at": now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperror.NewConflict("authorization code already used")
	}
	return nil
}

// DeleteExpired removes authorization codes that expired before the given
// cutoff time. Returns the number of rows deleted.
func (r *oauthAuthorizationCodeRepository) DeleteExpired(before time.Time) (int64, error) {
	var total int64
	for {
		result := r.DB().
			Where("expires_at < ?", before).
			Limit(10000).
			Delete(&OAuthAuthorizationCode{})
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

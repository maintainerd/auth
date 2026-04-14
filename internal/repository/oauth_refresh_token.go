package repository

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

// OAuthRefreshTokenRepository defines data access operations for OAuth refresh
// tokens with family-based rotation tracking.
type OAuthRefreshTokenRepository interface {
	BaseRepositoryMethods[model.OAuthRefreshToken]
	WithTx(tx *gorm.DB) OAuthRefreshTokenRepository
	FindByTokenHash(tokenHash string) (*model.OAuthRefreshToken, error)
	FindActiveByUserAndClient(userID, clientID int64) ([]model.OAuthRefreshToken, error)
	RevokeByID(tokenID int64) error
	RevokeByFamily(familyID uuid.UUID) (int64, error)
	RevokeByUserAndClient(userID, clientID int64) (int64, error)
	RevokeByUserID(userID int64) (int64, error)
	UpdateLastUsed(tokenID int64) error
	DeleteExpired(before time.Time) (int64, error)
	CountByUserAndClient(userID, clientID int64) (int64, error)
}

type oauthRefreshTokenRepository struct {
	*BaseRepository[model.OAuthRefreshToken]
}

// NewOAuthRefreshTokenRepository creates a new OAuthRefreshTokenRepository.
func NewOAuthRefreshTokenRepository(db *gorm.DB) OAuthRefreshTokenRepository {
	return &oauthRefreshTokenRepository{
		BaseRepository: NewBaseRepository[model.OAuthRefreshToken](db, "oauth_refresh_token_uuid", "oauth_refresh_token_id"),
	}
}

func (r *oauthRefreshTokenRepository) WithTx(tx *gorm.DB) OAuthRefreshTokenRepository {
	return &oauthRefreshTokenRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByTokenHash looks up a refresh token by its SHA-256 hash.
// Returns nil, nil when no matching token exists.
func (r *oauthRefreshTokenRepository) FindByTokenHash(tokenHash string) (*model.OAuthRefreshToken, error) {
	var token model.OAuthRefreshToken
	err := r.DB().
		Preload("Client").
		Preload("Client.IdentityProvider").
		Where("token_hash = ?", tokenHash).
		First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

// FindActiveByUserAndClient returns all non-revoked, non-expired refresh tokens
// for a user-client pair.
func (r *oauthRefreshTokenRepository) FindActiveByUserAndClient(userID, clientID int64) ([]model.OAuthRefreshToken, error) {
	var tokens []model.OAuthRefreshToken
	err := r.DB().
		Where("user_id = ? AND client_id = ? AND is_revoked = false AND expires_at > ?", userID, clientID, time.Now()).
		Find(&tokens).Error
	return tokens, err
}

// RevokeByID revokes a single refresh token.
func (r *oauthRefreshTokenRepository) RevokeByID(tokenID int64) error {
	now := time.Now()
	return r.DB().Model(&model.OAuthRefreshToken{}).
		Where("oauth_refresh_token_id = ? AND is_revoked = false", tokenID).
		Updates(map[string]any{
			"is_revoked": true,
			"revoked_at": now,
		}).Error
}

// RevokeByFamily revokes all refresh tokens in a family. Used for reuse
// detection — when a rotated-out token is presented again, the entire family
// is considered compromised. Returns the number of tokens revoked.
func (r *oauthRefreshTokenRepository) RevokeByFamily(familyID uuid.UUID) (int64, error) {
	now := time.Now()
	result := r.DB().Model(&model.OAuthRefreshToken{}).
		Where("family_id = ? AND is_revoked = false", familyID).
		Updates(map[string]any{
			"is_revoked": true,
			"revoked_at": now,
		})
	return result.RowsAffected, result.Error
}

// RevokeByUserAndClient revokes all refresh tokens for a user-client pair.
// Returns the number of tokens revoked.
func (r *oauthRefreshTokenRepository) RevokeByUserAndClient(userID, clientID int64) (int64, error) {
	now := time.Now()
	result := r.DB().Model(&model.OAuthRefreshToken{}).
		Where("user_id = ? AND client_id = ? AND is_revoked = false", userID, clientID).
		Updates(map[string]any{
			"is_revoked": true,
			"revoked_at": now,
		})
	return result.RowsAffected, result.Error
}

// RevokeByUserID revokes all refresh tokens for a user across all clients.
// Returns the number of tokens revoked.
func (r *oauthRefreshTokenRepository) RevokeByUserID(userID int64) (int64, error) {
	now := time.Now()
	result := r.DB().Model(&model.OAuthRefreshToken{}).
		Where("user_id = ? AND is_revoked = false", userID).
		Updates(map[string]any{
			"is_revoked": true,
			"revoked_at": now,
		})
	return result.RowsAffected, result.Error
}

// UpdateLastUsed records when a refresh token was last used at token exchange.
func (r *oauthRefreshTokenRepository) UpdateLastUsed(tokenID int64) error {
	return r.DB().Model(&model.OAuthRefreshToken{}).
		Where("oauth_refresh_token_id = ?", tokenID).
		Update("last_used_at", time.Now()).Error
}

// DeleteExpired removes refresh tokens that expired before the given cutoff.
// Returns the number of rows deleted.
func (r *oauthRefreshTokenRepository) DeleteExpired(before time.Time) (int64, error) {
	result := r.DB().
		Where("expires_at < ?", before).
		Delete(&model.OAuthRefreshToken{})
	return result.RowsAffected, result.Error
}

// CountByUserAndClient returns the total number of active refresh tokens for
// a given user-client pair. Used to enforce token count limits.
func (r *oauthRefreshTokenRepository) CountByUserAndClient(userID, clientID int64) (int64, error) {
	var count int64
	err := r.DB().Model(&model.OAuthRefreshToken{}).
		Where("user_id = ? AND client_id = ? AND is_revoked = false AND expires_at > ?", userID, clientID, time.Now()).
		Count(&count).Error
	return count, err
}

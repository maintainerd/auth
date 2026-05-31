package user

import (
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/database"
	"github.com/maintainerd/auth/internal/shared"
	"gorm.io/gorm"
)

type UserTokenRepository interface {
	BaseRepositoryMethods[UserToken]
	WithTx(tx *gorm.DB) UserTokenRepository
	FindByUserID(userID int64) ([]UserToken, error)
	FindActiveTokensByUserID(userID int64) ([]UserToken, error)
	FindByUserIDAndTokenType(userID int64, tokenType string) ([]UserToken, error)
	RevokeByUUID(tokenUUID uuid.UUID) error
	RevokeAllByUserID(userID int64) error
	DeleteByUserID(userID int64) error
	DeleteExpiredTokens(before time.Time) error

	// Session-specific methods
	FindActiveSessions(userID int64) ([]UserToken, error)
	FindActiveSessionByUUID(userID int64, sessionUUID uuid.UUID) (*UserToken, error)
	CountActiveSessions(userID int64) (int64, error)
	TouchSession(sessionUUID uuid.UUID, now time.Time) error
	RevokeSessionByUUID(userID int64, sessionUUID uuid.UUID) error
	RevokeAllSessionsByUserID(userID int64) error
}

type userTokenRepository struct {
	*BaseRepository[UserToken]
}

func NewUserTokenRepository(db *gorm.DB) UserTokenRepository {
	return &userTokenRepository{
		BaseRepository: database.NewBaseRepository[UserToken](db, "user_token_uuid", "user_token_id"),
	}
}

func (r *userTokenRepository) WithTx(tx *gorm.DB) UserTokenRepository {
	return &userTokenRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *userTokenRepository) FindByUserID(userID int64) ([]UserToken, error) {
	var tokens []UserToken
	err := r.DB().Where("user_id = ?", userID).Find(&tokens).Error
	return tokens, err
}

func (r *userTokenRepository) FindActiveTokensByUserID(userID int64) ([]UserToken, error) {
	var tokens []UserToken
	err := r.DB().
		Where("user_id = ? AND is_revoked = false AND (expires_at IS NULL OR expires_at > ?)", userID, time.Now()).
		Find(&tokens).Error
	return tokens, err
}

func (r *userTokenRepository) FindByUserIDAndTokenType(userID int64, tokenType string) ([]UserToken, error) {
	var tokens []UserToken
	err := r.DB().
		Where("user_id = ? AND token_type = ?", userID, tokenType).
		Find(&tokens).Error
	return tokens, err
}

func (r *userTokenRepository) RevokeByUUID(tokenUUID uuid.UUID) error {
	return r.DB().Model(&UserToken{}).
		Where("user_token_uuid = ?", tokenUUID).
		Update("is_revoked", true).Error
}

func (r *userTokenRepository) RevokeAllByUserID(userID int64) error {
	return r.DB().Model(&UserToken{}).
		Where("user_id = ?", userID).
		Update("is_revoked", true).Error
}

func (r *userTokenRepository) DeleteByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&UserToken{}).Error
}

func (r *userTokenRepository) DeleteExpiredTokens(before time.Time) error {
	return r.DB().
		Where("expires_at IS NOT NULL AND expires_at < ?", before).
		Delete(&UserToken{}).Error
}

// FindActiveSessions returns all non-revoked, non-expired session tokens for
// the given user (token_type = 'user:session'), ordered oldest-first.
func (r *userTokenRepository) FindActiveSessions(userID int64) ([]UserToken, error) {
	var tokens []UserToken
	now := time.Now()
	err := r.DB().
		Where("user_id = ? AND token_type = ? AND is_revoked = false AND (absolute_expires_at IS NULL OR absolute_expires_at > ?)",
			userID, shared.TokenTypeSession, now).
		Order("created_at ASC").
		Find(&tokens).Error
	return tokens, err
}

// FindActiveSessionByUUID looks up a single active session by UUID with
// ownership check (must belong to userID).
func (r *userTokenRepository) FindActiveSessionByUUID(userID int64, sessionUUID uuid.UUID) (*UserToken, error) {
	var token UserToken
	now := time.Now()
	err := r.DB().
		Where("user_id = ? AND user_token_uuid = ? AND token_type = ? AND is_revoked = false AND (absolute_expires_at IS NULL OR absolute_expires_at > ?)",
			userID, sessionUUID, shared.TokenTypeSession, now).
		First(&token).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &token, err
}

// CountActiveSessions returns the number of active (non-revoked, non-expired)
// session tokens for the given user.
func (r *userTokenRepository) CountActiveSessions(userID int64) (int64, error) {
	var count int64
	now := time.Now()
	err := r.DB().Model(&UserToken{}).
		Where("user_id = ? AND token_type = ? AND is_revoked = false AND (absolute_expires_at IS NULL OR absolute_expires_at > ?)",
			userID, shared.TokenTypeSession, now).
		Count(&count).Error
	return count, err
}

// TouchSession updates last_used_at for the session identified by sessionUUID.
// This implements the sliding idle timeout: callers invoke this on every
// authenticated request.
func (r *userTokenRepository) TouchSession(sessionUUID uuid.UUID, now time.Time) error {
	return r.DB().Model(&UserToken{}).
		Where("user_token_uuid = ? AND token_type = ? AND is_revoked = false", sessionUUID, shared.TokenTypeSession).
		Update("last_used_at", now).Error
}

// RevokeSessionByUUID revokes a single session token with an ownership check.
// Returns nil when the session is not found (idempotent).
func (r *userTokenRepository) RevokeSessionByUUID(userID int64, sessionUUID uuid.UUID) error {
	return r.DB().Model(&UserToken{}).
		Where("user_id = ? AND user_token_uuid = ? AND token_type = ?", userID, sessionUUID, shared.TokenTypeSession).
		Update("is_revoked", true).Error
}

// RevokeAllSessionsByUserID revokes all session tokens for a user without
// touching non-session token types (e.g. email verification, password reset).
func (r *userTokenRepository) RevokeAllSessionsByUserID(userID int64) error {
	return r.DB().Model(&UserToken{}).
		Where("user_id = ? AND token_type = ?", userID, shared.TokenTypeSession).
		Update("is_revoked", true).Error
}

package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type UserTokenRepository interface {
	BaseRepositoryMethods[model.UserToken]
	WithTx(tx *gorm.DB) UserTokenRepository
	FindByUserID(userID int64) ([]model.UserToken, error)
	FindActiveTokensByUserID(userID int64) ([]model.UserToken, error)
	FindByUserIDAndTokenType(userID int64, tokenType string) ([]model.UserToken, error)
	RevokeByUUID(tokenUUID uuid.UUID) error
	RevokeAllByUserID(userID int64) error
	DeleteByUserID(userID int64) error
	DeleteExpiredTokens(before time.Time) error
}

type userTokenRepository struct {
	*BaseRepository[model.UserToken]
	db *gorm.DB
}

func NewUserTokenRepository(db *gorm.DB) UserTokenRepository {
	return &userTokenRepository{
		BaseRepository: NewBaseRepository[model.UserToken](db, "token_uuid", "token_id"),
		db:             db,
	}
}

func (r *userTokenRepository) WithTx(tx *gorm.DB) UserTokenRepository {
	return &userTokenRepository{
		BaseRepository: NewBaseRepository[model.UserToken](tx, "token_uuid", "token_id"),
		db:             tx,
	}
}

func (r *userTokenRepository) FindByUserID(userID int64) ([]model.UserToken, error) {
	var tokens []model.UserToken
	err := r.db.Where("user_id = ?", userID).Find(&tokens).Error
	return tokens, err
}

func (r *userTokenRepository) FindActiveTokensByUserID(userID int64) ([]model.UserToken, error) {
	var tokens []model.UserToken
	err := r.db.
		Where("user_id = ? AND is_revoked = false AND (expires_at IS NULL OR expires_at > ?)", userID, time.Now()).
		Find(&tokens).Error
	return tokens, err
}

func (r *userTokenRepository) FindByUserIDAndTokenType(userID int64, tokenType string) ([]model.UserToken, error) {
	var tokens []model.UserToken
	err := r.db.
		Where("user_id = ? AND token_type = ?", userID, tokenType).
		Find(&tokens).Error
	return tokens, err
}

func (r *userTokenRepository) RevokeByUUID(tokenUUID uuid.UUID) error {
	return r.db.Model(&model.UserToken{}).
		Where("token_uuid = ?", tokenUUID).
		Update("is_revoked", true).Error
}

func (r *userTokenRepository) RevokeAllByUserID(userID int64) error {
	return r.db.Model(&model.UserToken{}).
		Where("user_id = ?", userID).
		Update("is_revoked", true).Error
}

func (r *userTokenRepository) DeleteByUserID(userID int64) error {
	return r.db.Where("user_id = ?", userID).Delete(&model.UserToken{}).Error
}

func (r *userTokenRepository) DeleteExpiredTokens(before time.Time) error {
	return r.db.
		Where("expires_at IS NOT NULL AND expires_at < ?", before).
		Delete(&model.UserToken{}).Error
}

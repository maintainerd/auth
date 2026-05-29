package mfa

import (
	"errors"
	"time"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type UserTOTPSecretRepository interface {
	BaseRepositoryMethods[model.UserTOTPSecret]
	WithTx(tx *gorm.DB) UserTOTPSecretRepository
	FindByUserID(userID int64) (*model.UserTOTPSecret, error)
	Upsert(secret *model.UserTOTPSecret) error
	Enable(userID int64) error
	Disable(userID int64) error
	UpdateLastUsed(userID int64) error
	DeleteByUserID(userID int64) error
}

type userTOTPSecretRepository struct {
	*BaseRepository[model.UserTOTPSecret]
}

func NewUserTOTPSecretRepository(db *gorm.DB) UserTOTPSecretRepository {
	return &userTOTPSecretRepository{
		BaseRepository: NewBaseRepository[model.UserTOTPSecret](db, "totp_secret_uuid", "totp_secret_id"),
	}
}

func (r *userTOTPSecretRepository) WithTx(tx *gorm.DB) UserTOTPSecretRepository {
	return &userTOTPSecretRepository{BaseRepository: r.BaseRepository.WithTx(tx)}
}

func (r *userTOTPSecretRepository) FindByUserID(userID int64) (*model.UserTOTPSecret, error) {
	var s model.UserTOTPSecret
	err := r.DB().Where("user_id = ?", userID).First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *userTOTPSecretRepository) Upsert(secret *model.UserTOTPSecret) error {
	var existing model.UserTOTPSecret
	err := r.DB().Where("user_id = ?", secret.UserID).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.DB().Create(secret).Error
		}
		return err
	}
	return r.DB().Model(&existing).Updates(map[string]any{
		"secret":     secret.Secret,
		"is_enabled": secret.IsEnabled,
		"updated_at": time.Now(),
	}).Error
}

func (r *userTOTPSecretRepository) Enable(userID int64) error {
	now := time.Now()
	return r.DB().Model(&model.UserTOTPSecret{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"is_enabled":  true,
			"enrolled_at": now,
			"updated_at":  now,
		}).Error
}

func (r *userTOTPSecretRepository) Disable(userID int64) error {
	return r.DB().Model(&model.UserTOTPSecret{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"is_enabled": false,
			"updated_at": time.Now(),
		}).Error
}

func (r *userTOTPSecretRepository) UpdateLastUsed(userID int64) error {
	now := time.Now()
	return r.DB().Model(&model.UserTOTPSecret{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"last_used_at": now,
			"updated_at":   now,
		}).Error
}

func (r *userTOTPSecretRepository) DeleteByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&model.UserTOTPSecret{}).Error
}

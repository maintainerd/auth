package mfa

import (
	"errors"
	"time"

	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type UserMFATOTPSecretRepository interface {
	BaseRepositoryMethods[UserMFATOTPSecret]
	WithTx(tx *gorm.DB) UserMFATOTPSecretRepository
	FindByUserID(userID int64) (*UserMFATOTPSecret, error)
	Upsert(secret *UserMFATOTPSecret) error
	Enable(userID int64) error
	Disable(userID int64) error
	UpdateLastUsed(userID int64) error
	MarkStepUsed(userID int64, step int64) (bool, error)
	DeleteByUserID(userID int64) error
}

type userMFATOTPSecretRepository struct {
	*BaseRepository[UserMFATOTPSecret]
}

func NewUserMFATOTPSecretRepository(db *gorm.DB) UserMFATOTPSecretRepository {
	return &userMFATOTPSecretRepository{
		BaseRepository: database.NewBaseRepository[UserMFATOTPSecret](db, "totp_secret_uuid", "totp_secret_id"),
	}
}

func (r *userMFATOTPSecretRepository) WithTx(tx *gorm.DB) UserMFATOTPSecretRepository {
	return &userMFATOTPSecretRepository{BaseRepository: r.BaseRepository.WithTx(tx)}
}

func (r *userMFATOTPSecretRepository) FindByUserID(userID int64) (*UserMFATOTPSecret, error) {
	var s UserMFATOTPSecret
	err := r.DB().Where("user_id = ?", userID).First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *userMFATOTPSecretRepository) Upsert(secret *UserMFATOTPSecret) error {
	var existing UserMFATOTPSecret
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

func (r *userMFATOTPSecretRepository) Enable(userID int64) error {
	now := time.Now()
	return r.DB().Model(&UserMFATOTPSecret{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"is_enabled":  true,
			"enrolled_at": now,
			"updated_at":  now,
		}).Error
}

func (r *userMFATOTPSecretRepository) Disable(userID int64) error {
	return r.DB().Model(&UserMFATOTPSecret{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"is_enabled": false,
			"updated_at": time.Now(),
		}).Error
}

func (r *userMFATOTPSecretRepository) UpdateLastUsed(userID int64) error {
	now := time.Now()
	return r.DB().Model(&UserMFATOTPSecret{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"last_used_at": now,
			"updated_at":   now,
		}).Error
}

func (r *userMFATOTPSecretRepository) MarkStepUsed(userID int64, step int64) (bool, error) {
	now := time.Now()
	result := r.DB().Model(&UserMFATOTPSecret{}).
		Where("user_id = ? AND (last_used_step IS NULL OR last_used_step < ?)", userID, step).
		Updates(map[string]any{
			"last_used_step": step,
			"last_used_at":   now,
			"updated_at":     now,
		})
	return result.RowsAffected > 0, result.Error
}

func (r *userMFATOTPSecretRepository) DeleteByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&UserMFATOTPSecret{}).Error
}

package notifier

import (
	"errors"
	"time"

	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type UserOTPRepository interface {
	BaseRepositoryMethods[UserOTP]
	WithTx(tx *gorm.DB) UserOTPRepository
	FindValid(channel, recipient string) (*UserOTP, error)
	RecordFailure(id int64, maxAttempts int) error
	MarkUsed(id int64) error
}

type userOTPRepository struct {
	*BaseRepository[UserOTP]
}

func NewUserOTPRepository(db *gorm.DB) UserOTPRepository {
	return &userOTPRepository{
		BaseRepository: database.NewBaseRepository[UserOTP](db, "user_otp_uuid", "user_otp_id"),
	}
}

func (r *userOTPRepository) WithTx(tx *gorm.DB) UserOTPRepository {
	return &userOTPRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *userOTPRepository) FindValid(channel, recipient string) (*UserOTP, error) {
	var otp UserOTP
	err := r.DB().
		Where("channel = ? AND recipient = ? AND used = false AND expires_at > ?", channel, recipient, time.Now()).
		Order("created_at DESC").
		First(&otp).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &otp, nil
}

func (r *userOTPRepository) RecordFailure(id int64, maxAttempts int) error {
	return r.DB().Model(&UserOTP{}).
		Where("user_otp_id = ?", id).
		Updates(map[string]any{
			"failed_attempts": gorm.Expr("failed_attempts + 1"),
			"used":            gorm.Expr("CASE WHEN failed_attempts + 1 >= ? THEN TRUE ELSE used END", maxAttempts),
		}).Error
}

func (r *userOTPRepository) MarkUsed(id int64) error {
	return r.DB().Model(&UserOTP{}).
		Where("user_otp_id = ?", id).
		Update("used", true).Error
}

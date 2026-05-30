package notifier

import (
	"errors"
	"time"

	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type SMSOtpRepository interface {
	BaseRepositoryMethods[SMSOtp]
	WithTx(tx *gorm.DB) SMSOtpRepository
	FindValidByPhone(phone string) (*SMSOtp, error)
	MarkUsed(id int64) error
}

type smsOtpRepository struct {
	*BaseRepository[SMSOtp]
}

func NewSMSOtpRepository(db *gorm.DB) SMSOtpRepository {
	return &smsOtpRepository{
		BaseRepository: database.NewBaseRepository[SMSOtp](db, "sms_otp_uuid", "sms_otp_id"),
	}
}

func (r *smsOtpRepository) WithTx(tx *gorm.DB) SMSOtpRepository {
	return &smsOtpRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *smsOtpRepository) FindValidByPhone(phone string) (*SMSOtp, error) {
	var otp SMSOtp
	err := r.DB().
		Where("phone = ? AND used = false AND expires_at > ?", phone, time.Now()).
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

func (r *smsOtpRepository) MarkUsed(id int64) error {
	return r.DB().Model(&SMSOtp{}).
		Where("sms_otp_id = ?", id).
		Update("used", true).Error
}

package notifier

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// EmailConfigRepository defines persistence operations for the email_config
// entity.
type EmailConfigRepository interface {
	BaseRepositoryMethods[EmailConfig]
	WithTx(tx *gorm.DB) EmailConfigRepository
	FindByTenantID(tenantID int64) (*EmailConfig, error)
}

type emailConfigRepository struct {
	*BaseRepository[EmailConfig]
}

// NewEmailConfigRepository creates a new EmailConfigRepository backed by the
// given database connection.
func NewEmailConfigRepository(db *gorm.DB) EmailConfigRepository {
	return &emailConfigRepository{
		BaseRepository: NewBaseRepository[EmailConfig](db, "email_config_uuid", "email_config_id"),
	}
}

// WithTx returns a copy of the repository bound to the supplied transaction.
func (r *emailConfigRepository) WithTx(tx *gorm.DB) EmailConfigRepository {
	return &emailConfigRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByTenantID retrieves the single email_config record for a tenant.
// Returns nil, nil when no record exists.
func (r *emailConfigRepository) FindByTenantID(tenantID int64) (*EmailConfig, error) {
	var config EmailConfig
	err := r.DB().Where("tenant_id = ?", tenantID).First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

// SMSConfigRepository defines persistence operations for the sms_config
// entity.
type SMSConfigRepository interface {
	BaseRepositoryMethods[SMSConfig]
	WithTx(tx *gorm.DB) SMSConfigRepository
	FindByTenantID(tenantID int64) (*SMSConfig, error)
}

type smsConfigRepository struct {
	*BaseRepository[SMSConfig]
}

// NewSMSConfigRepository creates a new SMSConfigRepository backed by the
// given database connection.
func NewSMSConfigRepository(db *gorm.DB) SMSConfigRepository {
	return &smsConfigRepository{
		BaseRepository: NewBaseRepository[SMSConfig](db, "sms_config_uuid", "sms_config_id"),
	}
}

// WithTx returns a copy of the repository bound to the supplied transaction.
func (r *smsConfigRepository) WithTx(tx *gorm.DB) SMSConfigRepository {
	return &smsConfigRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByTenantID retrieves the single sms_config record for a tenant. Returns
// nil, nil when no record exists.
func (r *smsConfigRepository) FindByTenantID(tenantID int64) (*SMSConfig, error) {
	var config SMSConfig
	err := r.DB().Where("tenant_id = ?", tenantID).First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

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
		BaseRepository: NewBaseRepository[SMSOtp](db, "sms_otp_uuid", "sms_otp_id"),
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

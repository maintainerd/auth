package repository

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

// SMSConfigRepository defines persistence operations for the sms_config
// entity.
type SMSConfigRepository interface {
	BaseRepositoryMethods[model.SMSConfig]
	WithTx(tx *gorm.DB) SMSConfigRepository
	FindByTenantID(tenantID int64) (*model.SMSConfig, error)
}

type smsConfigRepository struct {
	*BaseRepository[model.SMSConfig]
}

// NewSMSConfigRepository creates a new SMSConfigRepository backed by the
// given database connection.
func NewSMSConfigRepository(db *gorm.DB) SMSConfigRepository {
	return &smsConfigRepository{
		BaseRepository: NewBaseRepository[model.SMSConfig](db, "sms_config_uuid", "sms_config_id"),
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
func (r *smsConfigRepository) FindByTenantID(tenantID int64) (*model.SMSConfig, error) {
	var config model.SMSConfig
	err := r.DB().Where("tenant_id = ?", tenantID).First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

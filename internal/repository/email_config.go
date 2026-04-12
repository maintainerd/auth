package repository

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

// EmailConfigRepository defines persistence operations for the email_config
// entity.
type EmailConfigRepository interface {
	BaseRepositoryMethods[model.EmailConfig]
	WithTx(tx *gorm.DB) EmailConfigRepository
	FindByTenantID(tenantID int64) (*model.EmailConfig, error)
}

type emailConfigRepository struct {
	*BaseRepository[model.EmailConfig]
}

// NewEmailConfigRepository creates a new EmailConfigRepository backed by the
// given database connection.
func NewEmailConfigRepository(db *gorm.DB) EmailConfigRepository {
	return &emailConfigRepository{
		BaseRepository: NewBaseRepository[model.EmailConfig](db, "email_config_uuid", "email_config_id"),
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
func (r *emailConfigRepository) FindByTenantID(tenantID int64) (*model.EmailConfig, error) {
	var config model.EmailConfig
	err := r.DB().Where("tenant_id = ?", tenantID).First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

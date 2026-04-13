package repository

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

// TenantSettingRepository defines persistence operations for the
// tenant_settings entity.
type TenantSettingRepository interface {
	BaseRepositoryMethods[model.TenantSetting]
	WithTx(tx *gorm.DB) TenantSettingRepository
	FindByTenantID(tenantID int64) (*model.TenantSetting, error)
}

type tenantSettingRepository struct {
	*BaseRepository[model.TenantSetting]
}

// NewTenantSettingRepository creates a new TenantSettingRepository backed by
// the given database connection.
func NewTenantSettingRepository(db *gorm.DB) TenantSettingRepository {
	return &tenantSettingRepository{
		BaseRepository: NewBaseRepository[model.TenantSetting](db, "tenant_setting_uuid", "tenant_setting_id"),
	}
}

// WithTx returns a copy of the repository bound to the supplied transaction.
func (r *tenantSettingRepository) WithTx(tx *gorm.DB) TenantSettingRepository {
	return &tenantSettingRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByTenantID retrieves the single tenant_settings record for a tenant.
// Returns nil, nil when no record exists.
func (r *tenantSettingRepository) FindByTenantID(tenantID int64) (*model.TenantSetting, error) {
	var setting model.TenantSetting
	err := r.DB().Where("tenant_id = ?", tenantID).First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &setting, nil
}

package secpolicy

import (
	"errors"

	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type SecuritySettingRepositoryGetFilter struct {
	UserPoolID *int64
	Version    *int
	CreatedBy  *int64
	UpdatedBy  *int64
	Page       int
	Limit      int
	SortBy     string
	SortOrder  string
}

type SecuritySettingRepository interface {
	BaseRepositoryMethods[SecuritySetting]
	WithTx(tx *gorm.DB) SecuritySettingRepository
	FindByUserPoolID(tenantID int64) (*SecuritySetting, error)
	// FindDefaultByTenantID returns the security setting for a tenant's default
	// user pool, joining user_pools internally. Returns nil when not found.
	FindDefaultByTenantID(tenantID int64) (*SecuritySetting, error)
	FindPaginated(filter SecuritySettingRepositoryGetFilter) (*PaginationResult[SecuritySetting], error)
	IncrementVersion(securitySettingID int64) error
}

type securitySettingRepository struct {
	*BaseRepository[SecuritySetting]
}

func NewSecuritySettingRepository(db *gorm.DB) SecuritySettingRepository {
	return &securitySettingRepository{
		BaseRepository: database.NewBaseRepository[SecuritySetting](db, "security_setting_uuid", "security_setting_id"),
	}
}

func (r *securitySettingRepository) WithTx(tx *gorm.DB) SecuritySettingRepository {
	return &securitySettingRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *securitySettingRepository) FindDefaultByTenantID(tenantID int64) (*SecuritySetting, error) {
	var ss SecuritySetting
	err := r.DB().
		Joins("JOIN user_pools ON user_pools.user_pool_id = security_settings.user_pool_id").
		Where("user_pools.tenant_id = ? AND user_pools.is_system = true AND user_pools.deleted_at IS NULL", tenantID).
		First(&ss).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ss, nil
}

func (r *securitySettingRepository) FindByUserPoolID(tenantID int64) (*SecuritySetting, error) {
	var securitySetting SecuritySetting
	err := r.DB().Where("user_pool_id = ?", tenantID).First(&securitySetting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &securitySetting, nil
}

func (r *securitySettingRepository) FindPaginated(filter SecuritySettingRepositoryGetFilter) (*PaginationResult[SecuritySetting], error) {
	query := r.DB().Model(&SecuritySetting{})

	if filter.UserPoolID != nil {
		query = query.Where("user_pool_id = ?", *filter.UserPoolID)
	}
	if filter.Version != nil {
		query = query.Where("version = ?", *filter.Version)
	}
	if filter.CreatedBy != nil {
		query = query.Where("created_by = ?", *filter.CreatedBy)
	}
	if filter.UpdatedBy != nil {
		query = query.Where("updated_by = ?", *filter.UpdatedBy)
	}

	query = query.Order(database.SanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	return database.PaginateQuery[SecuritySetting](query, filter.Page, filter.Limit)
}

func (r *securitySettingRepository) IncrementVersion(securitySettingID int64) error {
	return r.DB().Model(&SecuritySetting{}).
		Where("security_setting_id = ?", securitySettingID).
		UpdateColumn("version", gorm.Expr("version + ?", 1)).Error
}

package secpolicy

import (
	"errors"
	"fmt"

	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

type SecuritySettingRepositoryGetFilter struct {
	TenantID  *int64
	Version   *int
	CreatedBy *int64
	UpdatedBy *int64
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type SecuritySettingRepository interface {
	BaseRepositoryMethods[SecuritySetting]
	FindByUUID(uuid any, preloads ...string) (*SecuritySetting, error)
	WithTx(tx *gorm.DB) SecuritySettingRepository
	// FindByTenantID returns the tenant's security setting, or nil when not found.
	FindByTenantID(tenantID int64) (*SecuritySetting, error)
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

func (r *securitySettingRepository) FindByTenantID(tenantID int64) (*SecuritySetting, error) {
	var securitySetting SecuritySetting
	err := r.DB().Where("tenant_id = ?", tenantID).First(&securitySetting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &securitySetting, nil
}

func (r *securitySettingRepository) FindPaginated(filter SecuritySettingRepositoryGetFilter) (*PaginationResult[SecuritySetting], error) {
	if filter.TenantID == nil || *filter.TenantID == 0 {
		return nil, fmt.Errorf("tenant_id is required")
	}

	query := r.DB().Model(&SecuritySetting{})

	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
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

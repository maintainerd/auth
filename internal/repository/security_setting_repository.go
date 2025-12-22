package repository

import (
	"github.com/maintainerd/auth/internal/model"
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
	BaseRepositoryMethods[model.SecuritySetting]
	WithTx(tx *gorm.DB) SecuritySettingRepository
	FindByTenantID(tenantID int64) (*model.SecuritySetting, error)
	FindPaginated(filter SecuritySettingRepositoryGetFilter) (*PaginationResult[model.SecuritySetting], error)
	IncrementVersion(securitySettingID int64) error
}

type securitySettingRepository struct {
	*BaseRepository[model.SecuritySetting]
	db *gorm.DB
}

func NewSecuritySettingRepository(db *gorm.DB) SecuritySettingRepository {
	return &securitySettingRepository{
		BaseRepository: NewBaseRepository[model.SecuritySetting](db, "security_setting_uuid", "security_setting_id"),
		db:             db,
	}
}

func (r *securitySettingRepository) WithTx(tx *gorm.DB) SecuritySettingRepository {
	return &securitySettingRepository{
		BaseRepository: NewBaseRepository[model.SecuritySetting](tx, "security_setting_uuid", "security_setting_id"),
		db:             tx,
	}
}

func (r *securitySettingRepository) FindByTenantID(tenantID int64) (*model.SecuritySetting, error) {
	var securitySetting model.SecuritySetting
	err := r.db.Where("tenant_id = ?", tenantID).First(&securitySetting).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &securitySetting, nil
}

func (r *securitySettingRepository) FindPaginated(filter SecuritySettingRepositoryGetFilter) (*PaginationResult[model.SecuritySetting], error) {
	query := r.db.Model(&model.SecuritySetting{})

	// Apply filters
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

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply sorting
	if filter.SortBy != "" {
		order := filter.SortBy
		if filter.SortOrder != "" {
			order += " " + filter.SortOrder
		}
		query = query.Order(order)
	} else {
		query = query.Order("created_at DESC")
	}

	// Apply pagination
	offset := (filter.Page - 1) * filter.Limit
	var securitySettings []model.SecuritySetting
	if err := query.Offset(offset).Limit(filter.Limit).Find(&securitySettings).Error; err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int(total) / filter.Limit
	if int(total)%filter.Limit > 0 {
		totalPages++
	}

	return &PaginationResult[model.SecuritySetting]{
		Data:       securitySettings,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *securitySettingRepository) IncrementVersion(securitySettingID int64) error {
	return r.db.Model(&model.SecuritySetting{}).
		Where("security_setting_id = ?", securitySettingID).
		UpdateColumn("version", gorm.Expr("version + ?", 1)).Error
}

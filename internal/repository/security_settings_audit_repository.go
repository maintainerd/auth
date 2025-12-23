package repository

import (
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type SecuritySettingsAuditRepositoryGetFilter struct {
	TenantID          *int64
	SecuritySettingID *int64
	ChangeType        *string
	CreatedBy         *int64
	Page              int
	Limit             int
	SortBy            string
	SortOrder         string
}

type SecuritySettingsAuditRepository interface {
	BaseRepositoryMethods[model.SecuritySettingsAudit]
	WithTx(tx *gorm.DB) SecuritySettingsAuditRepository
	FindBySecuritySettingID(securitySettingID int64) ([]model.SecuritySettingsAudit, error)
	FindByTenantID(tenantID int64) ([]model.SecuritySettingsAudit, error)
	FindPaginated(filter SecuritySettingsAuditRepositoryGetFilter) (*PaginationResult[model.SecuritySettingsAudit], error)
}

type securitySettingsAuditRepository struct {
	*BaseRepository[model.SecuritySettingsAudit]
	db *gorm.DB
}

func NewSecuritySettingsAuditRepository(db *gorm.DB) SecuritySettingsAuditRepository {
	return &securitySettingsAuditRepository{
		BaseRepository: NewBaseRepository[model.SecuritySettingsAudit](db, "security_settings_audit_uuid", "security_settings_audit_id"),
		db:             db,
	}
}

func (r *securitySettingsAuditRepository) WithTx(tx *gorm.DB) SecuritySettingsAuditRepository {
	return &securitySettingsAuditRepository{
		BaseRepository: NewBaseRepository[model.SecuritySettingsAudit](tx, "security_settings_audit_uuid", "security_settings_audit_id"),
		db:             tx,
	}
}

func (r *securitySettingsAuditRepository) FindBySecuritySettingID(securitySettingID int64) ([]model.SecuritySettingsAudit, error) {
	var audits []model.SecuritySettingsAudit
	err := r.db.Where("security_setting_id = ?", securitySettingID).Order("created_at DESC").Find(&audits).Error
	if err != nil {
		return nil, err
	}
	return audits, nil
}

func (r *securitySettingsAuditRepository) FindByTenantID(tenantID int64) ([]model.SecuritySettingsAudit, error) {
	var audits []model.SecuritySettingsAudit
	err := r.db.Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&audits).Error
	if err != nil {
		return nil, err
	}
	return audits, nil
}

func (r *securitySettingsAuditRepository) FindPaginated(filter SecuritySettingsAuditRepositoryGetFilter) (*PaginationResult[model.SecuritySettingsAudit], error) {
	query := r.db.Model(&model.SecuritySettingsAudit{})

	// Apply filters
	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
	}
	if filter.SecuritySettingID != nil {
		query = query.Where("security_setting_id = ?", *filter.SecuritySettingID)
	}
	if filter.ChangeType != nil {
		query = query.Where("change_type = ?", *filter.ChangeType)
	}
	if filter.CreatedBy != nil {
		query = query.Where("created_by = ?", *filter.CreatedBy)
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
	var audits []model.SecuritySettingsAudit
	if err := query.Offset(offset).Limit(filter.Limit).Find(&audits).Error; err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int(total) / filter.Limit
	if int(total)%filter.Limit > 0 {
		totalPages++
	}

	return &PaginationResult[model.SecuritySettingsAudit]{
		Data:       audits,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

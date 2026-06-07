package secpolicy

import (
	"github.com/maintainerd/auth/internal/platform/database"
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
	BaseRepositoryMethods[SecuritySettingsAudit]
	WithTx(tx *gorm.DB) SecuritySettingsAuditRepository
	FindBySecuritySettingID(securitySettingID int64) ([]SecuritySettingsAudit, error)
	FindByTenantID(tenantID int64) ([]SecuritySettingsAudit, error)
	FindPaginated(filter SecuritySettingsAuditRepositoryGetFilter) (*PaginationResult[SecuritySettingsAudit], error)
}

type securitySettingsAuditRepository struct {
	*BaseRepository[SecuritySettingsAudit]
}

func NewSecuritySettingsAuditRepository(db *gorm.DB) SecuritySettingsAuditRepository {
	return &securitySettingsAuditRepository{
		BaseRepository: database.NewBaseRepository[SecuritySettingsAudit](db, "security_settings_audit_uuid", "security_settings_audit_id"),
	}
}

func (r *securitySettingsAuditRepository) WithTx(tx *gorm.DB) SecuritySettingsAuditRepository {
	return &securitySettingsAuditRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *securitySettingsAuditRepository) FindBySecuritySettingID(securitySettingID int64) ([]SecuritySettingsAudit, error) {
	var audits []SecuritySettingsAudit
	err := r.DB().Where("security_setting_id = ?", securitySettingID).Order("created_at DESC").Find(&audits).Error
	if err != nil {
		return nil, err
	}
	return audits, nil
}

func (r *securitySettingsAuditRepository) FindByTenantID(tenantID int64) ([]SecuritySettingsAudit, error) {
	var audits []SecuritySettingsAudit
	err := r.DB().Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&audits).Error
	if err != nil {
		return nil, err
	}
	return audits, nil
}

func (r *securitySettingsAuditRepository) FindPaginated(filter SecuritySettingsAuditRepositoryGetFilter) (*PaginationResult[SecuritySettingsAudit], error) {
	query := r.DB().Model(&SecuritySettingsAudit{})

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

	query = query.Order(database.SanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	return database.PaginateQuery[SecuritySettingsAudit](query, filter.Page, filter.Limit)
}

package secpolicy

import (
	"errors"

	"gorm.io/gorm"
)

// IPRestrictionRuleRepositoryGetFilter holds filter, pagination, and sorting
// parameters for paginated IP restriction rule queries.
type IPRestrictionRuleRepositoryGetFilter struct {
	TenantID    *int64
	Type        *string
	Status      []string
	IPAddress   *string
	Description *string
	CreatedBy   *int64
	UpdatedBy   *int64
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

// IPRestrictionRuleRepository defines persistence operations for IP restriction
// rules.
type IPRestrictionRuleRepository interface {
	BaseRepositoryMethods[IPRestrictionRule]
	WithTx(tx *gorm.DB) IPRestrictionRuleRepository
	FindByTenantID(tenantID int64) ([]IPRestrictionRule, error)
	FindByTenantIDAndStatus(tenantID int64, status string) ([]IPRestrictionRule, error)
	FindByTenantIDAndType(tenantID int64, ruleType string) ([]IPRestrictionRule, error)
	FindPaginated(filter IPRestrictionRuleRepositoryGetFilter) (*PaginationResult[IPRestrictionRule], error)
}

type ipRestrictionRuleRepository struct {
	*BaseRepository[IPRestrictionRule]
}

// NewIPRestrictionRuleRepository creates a new IPRestrictionRuleRepository
// backed by the given database connection.
func NewIPRestrictionRuleRepository(db *gorm.DB) IPRestrictionRuleRepository {
	return &ipRestrictionRuleRepository{
		BaseRepository: NewBaseRepository[IPRestrictionRule](db, "ip_restriction_rule_uuid", "ip_restriction_rule_id"),
	}
}

// WithTx returns a copy of the repository that uses the given transaction.
func (r *ipRestrictionRuleRepository) WithTx(tx *gorm.DB) IPRestrictionRuleRepository {
	return &ipRestrictionRuleRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByTenantID returns all IP restriction rules belonging to the given tenant.
func (r *ipRestrictionRuleRepository) FindByTenantID(tenantID int64) ([]IPRestrictionRule, error) {
	var rules []IPRestrictionRule
	err := r.DB().Where("tenant_id = ?", tenantID).Find(&rules).Error
	if err != nil {
		return nil, err
	}
	return rules, nil
}

// FindByTenantIDAndStatus returns all IP restriction rules for a tenant
// filtered by status.
func (r *ipRestrictionRuleRepository) FindByTenantIDAndStatus(tenantID int64, status string) ([]IPRestrictionRule, error) {
	var rules []IPRestrictionRule
	err := r.DB().Where("tenant_id = ? AND status = ?", tenantID, status).Find(&rules).Error
	if err != nil {
		return nil, err
	}
	return rules, nil
}

// FindByTenantIDAndType returns all IP restriction rules for a tenant
// filtered by rule type.
func (r *ipRestrictionRuleRepository) FindByTenantIDAndType(tenantID int64, ruleType string) ([]IPRestrictionRule, error) {
	var rules []IPRestrictionRule
	err := r.DB().Where("tenant_id = ? AND type = ?", tenantID, ruleType).Find(&rules).Error
	if err != nil {
		return nil, err
	}
	return rules, nil
}

// FindPaginated returns a paginated, filtered, and sorted list of IP
// restriction rules.
func (r *ipRestrictionRuleRepository) FindPaginated(filter IPRestrictionRuleRepositoryGetFilter) (*PaginationResult[IPRestrictionRule], error) {
	query := r.DB().Model(&IPRestrictionRule{})

	// Apply filters
	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
	}
	if filter.Type != nil {
		query = query.Where("type = ?", *filter.Type)
	}
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if filter.IPAddress != nil {
		query = query.Where("ip_address ILIKE ?", "%"+*filter.IPAddress+"%")
	}
	if filter.Description != nil {
		query = query.Where("description ILIKE ?", "%"+*filter.Description+"%")
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

	// Apply sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	// Pagination guards prevent division-by-zero and negative offsets
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit
	var rules []IPRestrictionRule
	if err := query.Offset(offset).Limit(filter.Limit).Find(&rules).Error; err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int(total) / filter.Limit
	if int(total)%filter.Limit > 0 {
		totalPages++
	}

	return &PaginationResult[IPRestrictionRule]{
		Data:       rules,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

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
		BaseRepository: NewBaseRepository[SecuritySetting](db, "security_setting_uuid", "security_setting_id"),
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
		Where("user_pools.tenant_id = ? AND user_pools.is_default = true AND user_pools.deleted_at IS NULL", tenantID).
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

	// Apply filters
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

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	// Pagination guards prevent division-by-zero and negative offsets
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit
	var securitySettings []SecuritySetting
	if err := query.Offset(offset).Limit(filter.Limit).Find(&securitySettings).Error; err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int(total) / filter.Limit
	if int(total)%filter.Limit > 0 {
		totalPages++
	}

	return &PaginationResult[SecuritySetting]{
		Data:       securitySettings,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *securitySettingRepository) IncrementVersion(securitySettingID int64) error {
	return r.DB().Model(&SecuritySetting{}).
		Where("security_setting_id = ?", securitySettingID).
		UpdateColumn("version", gorm.Expr("version + ?", 1)).Error
}

type SecuritySettingsAuditRepositoryGetFilter struct {
	UserPoolID        *int64
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
	FindByUserPoolID(tenantID int64) ([]SecuritySettingsAudit, error)
	FindPaginated(filter SecuritySettingsAuditRepositoryGetFilter) (*PaginationResult[SecuritySettingsAudit], error)
}

type securitySettingsAuditRepository struct {
	*BaseRepository[SecuritySettingsAudit]
}

func NewSecuritySettingsAuditRepository(db *gorm.DB) SecuritySettingsAuditRepository {
	return &securitySettingsAuditRepository{
		BaseRepository: NewBaseRepository[SecuritySettingsAudit](db, "security_settings_audit_uuid", "security_settings_audit_id"),
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

func (r *securitySettingsAuditRepository) FindByUserPoolID(tenantID int64) ([]SecuritySettingsAudit, error) {
	var audits []SecuritySettingsAudit
	err := r.DB().Where("user_pool_id = ?", tenantID).Order("created_at DESC").Find(&audits).Error
	if err != nil {
		return nil, err
	}
	return audits, nil
}

func (r *securitySettingsAuditRepository) FindPaginated(filter SecuritySettingsAuditRepositoryGetFilter) (*PaginationResult[SecuritySettingsAudit], error) {
	query := r.DB().Model(&SecuritySettingsAudit{})

	// Apply filters
	if filter.UserPoolID != nil {
		query = query.Where("user_pool_id = ?", *filter.UserPoolID)
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

	// Apply sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	// Pagination guards prevent division-by-zero and negative offsets
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit
	var audits []SecuritySettingsAudit
	if err := query.Offset(offset).Limit(filter.Limit).Find(&audits).Error; err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int(total) / filter.Limit
	if int(total)%filter.Limit > 0 {
		totalPages++
	}

	return &PaginationResult[SecuritySettingsAudit]{
		Data:       audits,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

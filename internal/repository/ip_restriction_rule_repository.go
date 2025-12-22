package repository

import (
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type IpRestrictionRuleRepositoryGetFilter struct {
	TenantID    *int64
	Type        *string
	Status      []string
	IpAddress   *string
	Description *string
	CreatedBy   *int64
	UpdatedBy   *int64
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type IpRestrictionRuleRepository interface {
	BaseRepositoryMethods[model.IpRestrictionRule]
	WithTx(tx *gorm.DB) IpRestrictionRuleRepository
	FindByTenantID(tenantID int64) ([]model.IpRestrictionRule, error)
	FindByTenantIDAndStatus(tenantID int64, status string) ([]model.IpRestrictionRule, error)
	FindByTenantIDAndType(tenantID int64, ruleType string) ([]model.IpRestrictionRule, error)
	FindPaginated(filter IpRestrictionRuleRepositoryGetFilter) (*PaginationResult[model.IpRestrictionRule], error)
}

type ipRestrictionRuleRepository struct {
	*BaseRepository[model.IpRestrictionRule]
	db *gorm.DB
}

func NewIpRestrictionRuleRepository(db *gorm.DB) IpRestrictionRuleRepository {
	return &ipRestrictionRuleRepository{
		BaseRepository: NewBaseRepository[model.IpRestrictionRule](db, "ip_restriction_rule_uuid", "ip_restriction_rule_id"),
		db:             db,
	}
}

func (r *ipRestrictionRuleRepository) WithTx(tx *gorm.DB) IpRestrictionRuleRepository {
	return &ipRestrictionRuleRepository{
		BaseRepository: NewBaseRepository[model.IpRestrictionRule](tx, "ip_restriction_rule_uuid", "ip_restriction_rule_id"),
		db:             tx,
	}
}

func (r *ipRestrictionRuleRepository) FindByTenantID(tenantID int64) ([]model.IpRestrictionRule, error) {
	var rules []model.IpRestrictionRule
	err := r.db.Where("tenant_id = ?", tenantID).Find(&rules).Error
	if err != nil {
		return nil, err
	}
	return rules, nil
}

func (r *ipRestrictionRuleRepository) FindByTenantIDAndStatus(tenantID int64, status string) ([]model.IpRestrictionRule, error) {
	var rules []model.IpRestrictionRule
	err := r.db.Where("tenant_id = ? AND status = ?", tenantID, status).Find(&rules).Error
	if err != nil {
		return nil, err
	}
	return rules, nil
}

func (r *ipRestrictionRuleRepository) FindByTenantIDAndType(tenantID int64, ruleType string) ([]model.IpRestrictionRule, error) {
	var rules []model.IpRestrictionRule
	err := r.db.Where("tenant_id = ? AND type = ?", tenantID, ruleType).Find(&rules).Error
	if err != nil {
		return nil, err
	}
	return rules, nil
}

func (r *ipRestrictionRuleRepository) FindPaginated(filter IpRestrictionRuleRepositoryGetFilter) (*PaginationResult[model.IpRestrictionRule], error) {
	query := r.db.Model(&model.IpRestrictionRule{})

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
	if filter.IpAddress != nil {
		query = query.Where("ip_address ILIKE ?", "%"+*filter.IpAddress+"%")
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
	var rules []model.IpRestrictionRule
	if err := query.Offset(offset).Limit(filter.Limit).Find(&rules).Error; err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int(total) / filter.Limit
	if int(total)%filter.Limit > 0 {
		totalPages++
	}

	return &PaginationResult[model.IpRestrictionRule]{
		Data:       rules,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

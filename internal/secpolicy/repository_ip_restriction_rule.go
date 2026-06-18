package secpolicy

import (
	"fmt"

	"github.com/maintainerd/auth/internal/platform/database"
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
	FindByUUID(uuid any, preloads ...string) (*IPRestrictionRule, error)
	UpdateByUUID(uuid any, updatedData any) (*IPRestrictionRule, error)
	DeleteByUUID(uuid any) error
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
		BaseRepository: database.NewBaseRepository[IPRestrictionRule](db, "ip_restriction_rule_uuid", "ip_restriction_rule_id"),
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
	if filter.TenantID == nil || *filter.TenantID == 0 {
		return nil, fmt.Errorf("tenant_id is required")
	}

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
	query = database.ApplyILike(query, "ip_address", filter.IPAddress)
	query = database.ApplyILike(query, "description", filter.Description)
	if filter.CreatedBy != nil {
		query = query.Where("created_by = ?", *filter.CreatedBy)
	}
	if filter.UpdatedBy != nil {
		query = query.Where("updated_by = ?", *filter.UpdatedBy)
	}

	query = query.Order(database.SanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	return database.PaginateQuery[IPRestrictionRule](query, filter.Page, filter.Limit)
}

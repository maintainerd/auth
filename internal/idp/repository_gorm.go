package idp

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IdentityProviderRepositoryGetFilter struct {
	Name         *string
	DisplayName  *string
	Provider     []string
	ProviderType *string
	Identifier   *string
	TenantID     *int64
	Status       []string
	IsDefault    *bool
	IsSystem     *bool
	Page         int
	Limit        int
	SortBy       string
	SortOrder    string
}

type IdentityProviderRepository interface {
	BaseRepositoryMethods[IdentityProvider]
	WithTx(tx *gorm.DB) IdentityProviderRepository
	FindByName(name string, tenantID int64) (*IdentityProvider, error)
	FindByIdentifier(identifier string) (*IdentityProvider, error)
	FindDefaultByTenantID(tenantID int64) (*IdentityProvider, error)
	// FindByTenantAndProvider returns the active provider record matching the
	// tenant and provider slug (e.g. "google", "cognito").
	FindByTenantAndProvider(tenantID int64, provider string) (*IdentityProvider, error)
	// FindAllByTenantID returns every provider configured for a tenant.
	FindAllByTenantID(tenantID int64) ([]IdentityProvider, error)
	FindPaginated(filter IdentityProviderRepositoryGetFilter) (*PaginationResult[IdentityProvider], error)
}

type identityProviderRepository struct {
	*BaseRepository[IdentityProvider]
}

func NewIdentityProviderRepository(db *gorm.DB) IdentityProviderRepository {
	return &identityProviderRepository{
		BaseRepository: NewBaseRepository[IdentityProvider](db, "identity_provider_uuid", "identity_provider_id"),
	}
}

func (r *identityProviderRepository) WithTx(tx *gorm.DB) IdentityProviderRepository {
	return &identityProviderRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *identityProviderRepository) FindByName(name string, tenantID int64) (*IdentityProvider, error) {
	var provider IdentityProvider
	err := r.DB().
		Where("name = ? AND tenant_id = ?", name, tenantID).
		First(&provider).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &provider, err
}

func (r *identityProviderRepository) FindByIdentifier(identifier string) (*IdentityProvider, error) {
	var provider IdentityProvider
	err := r.DB().
		Where("identifier = ?", identifier).
		First(&provider).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &provider, nil
}

func (r *identityProviderRepository) FindDefaultByTenantID(tenantID int64) (*IdentityProvider, error) {
	var provider IdentityProvider
	err := r.DB().
		Where("tenant_id = ? AND is_default = true", tenantID).
		First(&provider).Error
	return &provider, err
}

func (r *identityProviderRepository) FindPaginated(filter IdentityProviderRepositoryGetFilter) (*PaginationResult[IdentityProvider], error) {
	query := r.DB().Model(&IdentityProvider{})

	// Filters with LIKE
	if filter.Name != nil {
		query = query.Where("name ILIKE ?", "%"+*filter.Name+"%")
	}
	if filter.DisplayName != nil {
		query = query.Where("display_name ILIKE ?", "%"+*filter.DisplayName+"%")
	}

	// Filters with exact match
	if len(filter.Provider) > 0 {
		query = query.Where("provider IN ?", filter.Provider)
	}
	if filter.ProviderType != nil {
		query = query.Where("provider_type = ?", *filter.ProviderType)
	}
	if filter.Identifier != nil {
		query = query.Where("identifier = ?", *filter.Identifier)
	}
	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
	}
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
	}
	if filter.IsSystem != nil {
		query = query.Where("is_system = ?", *filter.IsSystem)
	}

	// Sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	// Count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Pagination guards prevent division-by-zero and negative offsets
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit
	var apis []IdentityProvider
	if err := query.Preload("Tenant").Limit(filter.Limit).Offset(offset).Find(&apis).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[IdentityProvider]{
		Data:       apis,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *identityProviderRepository) FindByTenantAndProvider(tenantID int64, provider string) (*IdentityProvider, error) {
	var idp IdentityProvider
	err := r.DB().
		Where("tenant_id = ? AND provider = ? AND deleted_at IS NULL", tenantID, provider).
		First(&idp).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &idp, nil
}

func (r *identityProviderRepository) FindAllByTenantID(tenantID int64) ([]IdentityProvider, error) {
	var idps []IdentityProvider
	err := r.DB().
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Find(&idps).Error
	return idps, err
}

type SignupFlowRepositoryGetFilter struct {
	Name       *string
	Identifier *string
	Status     []string
	TenantID   *int64
	ClientID   *int64
	Page       int
	Limit      int
	SortBy     string
	SortOrder  string
}

type SignupFlowRepository interface {
	BaseRepositoryMethods[SignupFlow]
	WithTx(tx *gorm.DB) SignupFlowRepository
	FindPaginated(filter SignupFlowRepositoryGetFilter) (*PaginationResult[SignupFlow], error)
	FindByUUIDAndTenantID(signupFlowUUID uuid.UUID, tenantID int64, preloads ...string) (*SignupFlow, error)
	FindByIdentifierAndClientID(identifier string, clientID int64) (*SignupFlow, error)
	FindByName(name string) (*SignupFlow, error)
}

type signupFlowRepository struct {
	*BaseRepository[SignupFlow]
}

func NewSignupFlowRepository(db *gorm.DB) SignupFlowRepository {
	return &signupFlowRepository{
		BaseRepository: NewBaseRepository[SignupFlow](db, "signup_flow_uuid", "signup_flow_id"),
	}
}

func (r *signupFlowRepository) WithTx(tx *gorm.DB) SignupFlowRepository {
	return &signupFlowRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *signupFlowRepository) FindPaginated(filter SignupFlowRepositoryGetFilter) (*PaginationResult[SignupFlow], error) {
	var signupFlows []SignupFlow
	var total int64

	query := r.DB().Model(&SignupFlow{})

	// Apply filters
	if filter.Name != nil && *filter.Name != "" {
		query = query.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(*filter.Name)+"%")
	}
	if filter.Identifier != nil && *filter.Identifier != "" {
		query = query.Where("LOWER(identifier) LIKE ?", "%"+strings.ToLower(*filter.Identifier)+"%")
	}
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
	}
	if filter.ClientID != nil {
		query = query.Where("client_id = ?", *filter.ClientID)
	}

	// Count total before pagination
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	// Apply pagination
	offset := (filter.Page - 1) * filter.Limit
	query = query.Offset(offset).Limit(filter.Limit)

	// Execute query with preloads
	if err := query.Preload("Client").Find(&signupFlows).Error; err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int(total) / filter.Limit
	if int(total)%filter.Limit > 0 {
		totalPages++
	}

	return &PaginationResult[SignupFlow]{
		Data:       signupFlows,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *signupFlowRepository) FindByIdentifierAndClientID(identifier string, clientID int64) (*SignupFlow, error) {
	var signupFlow SignupFlow
	err := r.DB().Where("identifier = ? AND client_id = ?", identifier, clientID).First(&signupFlow).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &signupFlow, nil
}

func (r *signupFlowRepository) FindByUUIDAndTenantID(signupFlowUUID uuid.UUID, tenantID int64, preloads ...string) (*SignupFlow, error) {
	var signupFlow SignupFlow
	query := r.DB().Where("signup_flow_uuid = ? AND tenant_id = ?", signupFlowUUID, tenantID)

	for _, preload := range preloads {
		query = query.Preload(preload)
	}

	err := query.First(&signupFlow).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &signupFlow, nil
}

func (r *signupFlowRepository) FindByName(name string) (*SignupFlow, error) {
	var signupFlow SignupFlow
	err := r.DB().Where("name = ?", name).First(&signupFlow).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &signupFlow, nil
}

type SignupFlowRoleRepository interface {
	BaseRepositoryMethods[SignupFlowRole]
	WithTx(tx *gorm.DB) SignupFlowRoleRepository
	FindBySignupFlowID(signupFlowID int64) ([]SignupFlowRole, error)
	FindBySignupFlowIDPaginated(signupFlowID int64, page, limit int) ([]SignupFlowRole, int64, error)
	DeleteBySignupFlowIDAndRoleID(signupFlowID, roleID int64) error
	FindBySignupFlowIDAndRoleID(signupFlowID, roleID int64) (*SignupFlowRole, error)
}

type signupFlowRoleRepository struct {
	*BaseRepository[SignupFlowRole]
}

func NewSignupFlowRoleRepository(db *gorm.DB) SignupFlowRoleRepository {
	return &signupFlowRoleRepository{
		BaseRepository: NewBaseRepository[SignupFlowRole](db, "signup_flow_role_uuid", "signup_flow_role_id"),
	}
}

func (r *signupFlowRoleRepository) WithTx(tx *gorm.DB) SignupFlowRoleRepository {
	return &signupFlowRoleRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *signupFlowRoleRepository) FindBySignupFlowID(signupFlowID int64) ([]SignupFlowRole, error) {
	var signupFlowRoles []SignupFlowRole
	err := r.DB().Where("signup_flow_id = ?", signupFlowID).Preload("Role").Find(&signupFlowRoles).Error
	if err != nil {
		return nil, err
	}
	return signupFlowRoles, nil
}

func (r *signupFlowRoleRepository) FindBySignupFlowIDPaginated(signupFlowID int64, page, limit int) ([]SignupFlowRole, int64, error) {
	var signupFlowRoles []SignupFlowRole
	var total int64

	query := r.DB().Where("signup_flow_id = ?", signupFlowID)

	// Get total count
	if err := query.Model(&SignupFlowRole{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated data
	offset := (page - 1) * limit
	err := query.Preload("Role").Offset(offset).Limit(limit).Find(&signupFlowRoles).Error
	if err != nil {
		return nil, 0, err
	}

	return signupFlowRoles, total, nil
}

func (r *signupFlowRoleRepository) DeleteBySignupFlowIDAndRoleID(signupFlowID, roleID int64) error {
	return r.DB().Where("signup_flow_id = ? AND role_id = ?", signupFlowID, roleID).Delete(&SignupFlowRole{}).Error
}

func (r *signupFlowRoleRepository) FindBySignupFlowIDAndRoleID(signupFlowID, roleID int64) (*SignupFlowRole, error) {
	var signupFlowRole SignupFlowRole
	err := r.DB().Where("signup_flow_id = ? AND role_id = ?", signupFlowID, roleID).First(&signupFlowRole).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &signupFlowRole, nil
}

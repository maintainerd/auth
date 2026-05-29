package idp

import (
	"errors"

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

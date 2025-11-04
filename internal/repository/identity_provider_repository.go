package repository

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type IdentityProviderRepositoryGetFilter struct {
	Name         *string
	DisplayName  *string
	ProviderType *string
	Identifier   *string
	TenantID     *int64
	IsActive     *bool
	IsDefault    *bool
	Page         int
	Limit        int
	SortBy       string
	SortOrder    string
}

type IdentityProviderRepository interface {
	BaseRepositoryMethods[model.IdentityProvider]
	WithTx(tx *gorm.DB) IdentityProviderRepository
	FindByName(name string, tenantID int64) (*model.IdentityProvider, error)
	FindByIdentifier(identifier string) (*model.IdentityProvider, error)
	FindDefaultByTenantID(tenantID int64) (*model.IdentityProvider, error)
	FindPaginated(filter IdentityProviderRepositoryGetFilter) (*PaginationResult[model.IdentityProvider], error)
}

type identityProviderRepository struct {
	*BaseRepository[model.IdentityProvider]
	db *gorm.DB
}

func NewIdentityProviderRepository(db *gorm.DB) IdentityProviderRepository {
	return &identityProviderRepository{
		BaseRepository: NewBaseRepository[model.IdentityProvider](db, "identity_provider_uuid", "identity_provider_id"),
		db:             db,
	}
}

func (r *identityProviderRepository) WithTx(tx *gorm.DB) IdentityProviderRepository {
	return &identityProviderRepository{
		BaseRepository: NewBaseRepository[model.IdentityProvider](tx, "identity_provider_uuid", "identity_provider_id"),
		db:             tx,
	}
}

func (r *identityProviderRepository) FindByName(name string, tenantID int64) (*model.IdentityProvider, error) {
	var provider model.IdentityProvider
	err := r.db.
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

func (r *identityProviderRepository) FindByIdentifier(identifier string) (*model.IdentityProvider, error) {
	var provider model.IdentityProvider
	err := r.db.
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

func (r *identityProviderRepository) FindDefaultByTenantID(tenantID int64) (*model.IdentityProvider, error) {
	var provider model.IdentityProvider
	err := r.db.
		Where("tenant_id = ? AND is_default = true", tenantID).
		First(&provider).Error
	return &provider, err
}

func (r *identityProviderRepository) FindPaginated(filter IdentityProviderRepositoryGetFilter) (*PaginationResult[model.IdentityProvider], error) {
	query := r.db.Model(&model.IdentityProvider{})

	// Filters with LIKE
	if filter.Name != nil {
		query = query.Where("name ILIKE ?", "%"+*filter.Name+"%")
	}
	if filter.DisplayName != nil {
		query = query.Where("display_name ILIKE ?", "%"+*filter.DisplayName+"%")
	}

	// Filters with exact match
	if filter.ProviderType != nil {
		query = query.Where("provider_type = ?", *filter.ProviderType)
	}
	if filter.Identifier != nil {
		query = query.Where("identifier = ?", *filter.Identifier)
	}
	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
	}

	// Sorting
	orderBy := filter.SortBy + " " + filter.SortOrder
	query = query.Order(orderBy)

	// Count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Pagination
	offset := (filter.Page - 1) * filter.Limit
	var apis []model.IdentityProvider
	if err := query.Preload("AuthContainer").Limit(filter.Limit).Offset(offset).Find(&apis).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[model.IdentityProvider]{
		Data:       apis,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

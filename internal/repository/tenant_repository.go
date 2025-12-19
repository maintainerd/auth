package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type TenantRepositoryGetFilter struct {
	Name        *string
	Description *string
	Identifier  *string
	Status      []string
	IsPublic    *bool
	IsDefault   *bool
	IsSystem    *bool
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type TenantRepository interface {
	BaseRepositoryMethods[model.Tenant]
	WithTx(tx *gorm.DB) TenantRepository
	FindByName(name string) (*model.Tenant, error)
	FindByIdentifier(identifier string) (*model.Tenant, error)
	FindDefault() (*model.Tenant, error)
	FindPaginated(filter TenantRepositoryGetFilter) (*PaginationResult[model.Tenant], error)
	SetStatusByUUID(tenantUUID uuid.UUID, status string) error
	SetDefaultStatusByUUID(tenantUUID uuid.UUID, isDefault bool) error
	SetSystemStatusByUUID(tenantUUID uuid.UUID, isSystem bool) error
}

type tenantRepository struct {
	*BaseRepository[model.Tenant]
	db *gorm.DB
}

func NewTenantRepository(db *gorm.DB) TenantRepository {
	return &tenantRepository{
		BaseRepository: NewBaseRepository[model.Tenant](db, "tenant_uuid", "tenant_id"),
		db:             db,
	}
}

func (r *tenantRepository) WithTx(tx *gorm.DB) TenantRepository {
	return &tenantRepository{
		BaseRepository: NewBaseRepository[model.Tenant](tx, "tenant_uuid", "tenant_id"),
		db:             tx,
	}
}

func (r *tenantRepository) FindByName(name string) (*model.Tenant, error) {
	var tenant model.Tenant
	err := r.db.Where("name = ?", name).First(&tenant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tenant, nil
}

func (r *tenantRepository) FindByIdentifier(identifier string) (*model.Tenant, error) {
	var tenant model.Tenant
	err := r.db.Where("identifier = ?", identifier).First(&tenant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tenant, nil
}

func (r *tenantRepository) FindDefault() (*model.Tenant, error) {
	var tenant model.Tenant
	err := r.db.Where("is_default = ?", true).First(&tenant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tenant, nil
}

func (r *tenantRepository) FindPaginated(filter TenantRepositoryGetFilter) (*PaginationResult[model.Tenant], error) {
	query := r.db.Model(&model.Tenant{})

	// Filters with LIKE
	if filter.Name != nil {
		query = query.Where("name ILIKE ?", "%"+*filter.Name+"%")
	}
	if filter.Description != nil {
		query = query.Where("description ILIKE ?", "%"+*filter.Description+"%")
	}
	if filter.Identifier != nil {
		query = query.Where("identifier ILIKE ?", "%"+*filter.Identifier+"%")
	}

	// Filters with exact match
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if filter.IsPublic != nil {
		query = query.Where("is_public = ?", *filter.IsPublic)
	}
	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
	}
	if filter.IsSystem != nil {
		query = query.Where("is_system = ?", *filter.IsSystem)
	}

	// Sorting
	if filter.SortBy != "" {
		order := "ASC"
		if filter.SortOrder == "desc" {
			order = "DESC"
		}
		query = query.Order(filter.SortBy + " " + order)
	} else {
		query = query.Order("created_at DESC")
	}

	// Count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Pagination
	offset := (filter.Page - 1) * filter.Limit
	var tenants []model.Tenant
	if err := query.Limit(filter.Limit).Offset(offset).Find(&tenants).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[model.Tenant]{
		Data:       tenants,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *tenantRepository) SetStatusByUUID(tenantUUID uuid.UUID, status string) error {
	return r.db.Model(&model.Tenant{}).Where("tenant_uuid = ?", tenantUUID).Update("status", status).Error
}

func (r *tenantRepository) SetDefaultStatusByUUID(tenantUUID uuid.UUID, isDefault bool) error {
	return r.db.Model(&model.Tenant{}).Where("tenant_uuid = ?", tenantUUID).Update("is_default", isDefault).Error
}

func (r *tenantRepository) SetSystemStatusByUUID(tenantUUID uuid.UUID, isSystem bool) error {
	return r.db.Model(&model.Tenant{}).Where("tenant_uuid = ?", tenantUUID).Update("is_system", isSystem).Error
}

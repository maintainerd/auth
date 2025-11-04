package repository

import (
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type TenantServiceRepositoryGetFilter struct {
	TenantID  *int64
	ServiceID *int64
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type TenantServiceRepository interface {
	BaseRepositoryMethods[model.TenantService]
	WithTx(tx *gorm.DB) TenantServiceRepository
	FindPaginated(filter TenantServiceRepositoryGetFilter) (*PaginationResult[model.TenantService], error)
	FindByTenantAndService(tenantID int64, serviceID int64) (*model.TenantService, error)
	DeleteByTenantAndService(tenantID int64, serviceID int64) error
}

type tenantServiceRepository struct {
	*BaseRepository[model.TenantService]
	db *gorm.DB
}

func NewTenantServiceRepository(db *gorm.DB) TenantServiceRepository {
	return &tenantServiceRepository{
		BaseRepository: NewBaseRepository[model.TenantService](db, "", "tenant_service_id"),
		db:             db,
	}
}

func (r *tenantServiceRepository) WithTx(tx *gorm.DB) TenantServiceRepository {
	return &tenantServiceRepository{
		BaseRepository: NewBaseRepository[model.TenantService](tx, "", "tenant_service_id"),
		db:             tx,
	}
}

func (r *tenantServiceRepository) FindByTenantAndService(tenantID int64, serviceID int64) (*model.TenantService, error) {
	var tenantService model.TenantService
	err := r.db.Where("tenant_id = ? AND service_id = ?", tenantID, serviceID).First(&tenantService).Error
	if err != nil {
		return nil, err
	}
	return &tenantService, nil
}

func (r *tenantServiceRepository) DeleteByTenantAndService(tenantID int64, serviceID int64) error {
	return r.db.Where("tenant_id = ? AND service_id = ?", tenantID, serviceID).Delete(&model.TenantService{}).Error
}

func (r *tenantServiceRepository) FindPaginated(filter TenantServiceRepositoryGetFilter) (*PaginationResult[model.TenantService], error) {
	query := r.db.Model(&model.TenantService{})

	// Filters
	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
	}
	if filter.ServiceID != nil {
		query = query.Where("service_id = ?", *filter.ServiceID)
	}

	// Sorting
	if filter.SortBy != "" {
		order := "ASC"
		if filter.SortOrder == "desc" {
			order = "DESC"
		}
		query = query.Order(filter.SortBy + " " + order)
	} else {
		query = query.Order("tenant_service_id DESC")
	}

	// Pagination
	offset := (filter.Page - 1) * filter.Limit
	if filter.Limit > 0 {
		query = query.Offset(offset).Limit(filter.Limit)
	}

	var tenantServices []model.TenantService
	var total int64

	// Count total records
	countQuery := r.db.Model(&model.TenantService{})
	if filter.TenantID != nil {
		countQuery = countQuery.Where("tenant_id = ?", *filter.TenantID)
	}
	if filter.ServiceID != nil {
		countQuery = countQuery.Where("service_id = ?", *filter.ServiceID)
	}
	countQuery.Count(&total)

	// Execute query with preloads
	err := query.Preload("Tenant").Preload("Service").Find(&tenantServices).Error
	if err != nil {
		return nil, err
	}

	return &PaginationResult[model.TenantService]{
		Data:  tenantServices,
		Total: total,
		Page:  filter.Page,
		Limit: filter.Limit,
	}, nil
}

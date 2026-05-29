package tenant

import (
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
	BaseRepositoryMethods[TenantService]
	WithTx(tx *gorm.DB) TenantServiceRepository
	FindPaginated(filter TenantServiceRepositoryGetFilter) (*PaginationResult[TenantService], error)
	FindByTenantAndService(tenantID int64, serviceID int64) (*TenantService, error)
	DeleteByTenantAndService(tenantID int64, serviceID int64) error
}

type tenantServiceRepository struct {
	*BaseRepository[TenantService]
}

func NewTenantServiceRepository(db *gorm.DB) TenantServiceRepository {
	return &tenantServiceRepository{
		BaseRepository: NewBaseRepository[TenantService](db, "", "tenant_service_id"),
	}
}

func (r *tenantServiceRepository) WithTx(tx *gorm.DB) TenantServiceRepository {
	return &tenantServiceRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *tenantServiceRepository) FindByTenantAndService(tenantID int64, serviceID int64) (*TenantService, error) {
	var tenantService TenantService
	err := r.DB().Where("tenant_id = ? AND service_id = ?", tenantID, serviceID).First(&tenantService).Error
	if err != nil {
		return nil, err
	}
	return &tenantService, nil
}

func (r *tenantServiceRepository) DeleteByTenantAndService(tenantID int64, serviceID int64) error {
	return r.DB().Where("tenant_id = ? AND service_id = ?", tenantID, serviceID).Delete(&TenantService{}).Error
}

func (r *tenantServiceRepository) FindPaginated(filter TenantServiceRepositoryGetFilter) (*PaginationResult[TenantService], error) {
	query := r.DB().Model(&TenantService{})

	// Filters
	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
	}
	if filter.ServiceID != nil {
		query = query.Where("service_id = ?", *filter.ServiceID)
	}

	// Sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "tenant_service_id DESC"))

	// Pagination
	filter.Page, filter.Limit = normalizePagination(filter.Page, filter.Limit)
	offset := (filter.Page - 1) * filter.Limit
	if filter.Limit > 0 {
		query = query.Offset(offset).Limit(filter.Limit)
	}

	var tenantServices []TenantService
	var total int64

	// Count total records
	countQuery := r.DB().Model(&TenantService{})
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

	return &PaginationResult[TenantService]{
		Data:  tenantServices,
		Total: total,
		Page:  filter.Page,
		Limit: filter.Limit,
	}, nil
}

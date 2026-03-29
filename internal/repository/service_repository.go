package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type ServiceRepositoryGetFilter struct {
	Name        *string
	DisplayName *string
	Description *string
	Version     *string
	TenantID    *int64
	IsSystem    *bool
	Status      []string
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type ServiceRepository interface {
	BaseRepositoryMethods[model.Service]
	WithTx(tx *gorm.DB) ServiceRepository
	FindByName(serviceName string) (*model.Service, error)
	FindByNameAndTenantID(serviceName string, tenantID int64) (*model.Service, error)
	FindByTenantID(tenantID int64) ([]model.Service, error)
	FindPaginated(filter ServiceRepositoryGetFilter) (*PaginationResult[model.Service], error)
	FindServicesByPolicyUUID(policyUUID uuid.UUID, filter ServiceRepositoryGetFilter) (*PaginationResult[model.Service], error)
	SetStatusByUUID(serviceUUID uuid.UUID, status string) error
	CountPoliciesByServiceID(serviceID int64) (int64, error)
}

type serviceRepository struct {
	*BaseRepository[model.Service]
}

func NewServiceRepository(db *gorm.DB) ServiceRepository {
	return &serviceRepository{
		BaseRepository: NewBaseRepository[model.Service](db, "service_uuid", "service_id"),
	}
}

func (r *serviceRepository) WithTx(tx *gorm.DB) ServiceRepository {
	return &serviceRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *serviceRepository) FindByName(serviceName string) (*model.Service, error) {
	var service model.Service
	err := r.DB().Where("name = ?", serviceName).First(&service).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &service, nil
}

func (r *serviceRepository) FindByNameAndTenantID(serviceName string, tenantID int64) (*model.Service, error) {
	var service model.Service
	err := r.DB().
		Joins("JOIN tenant_services ON services.service_id = tenant_services.service_id").
		Where("services.name = ? AND tenant_services.tenant_id = ?", serviceName, tenantID).
		First(&service).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &service, nil
}

func (r *serviceRepository) FindByTenantID(tenantID int64) ([]model.Service, error) {
	var services []model.Service
	err := r.DB().
		Joins("JOIN tenant_services ON services.service_id = tenant_services.service_id").
		Where("tenant_services.tenant_id = ?", tenantID).
		Find(&services).Error
	return services, err
}

func (r *serviceRepository) FindPaginated(filter ServiceRepositoryGetFilter) (*PaginationResult[model.Service], error) {
	query := r.DB().Model(&model.Service{})

	// Filters with LIKE
	if filter.Name != nil {
		query = query.Where("name ILIKE ?", "%"+*filter.Name+"%")
	}
	if filter.DisplayName != nil {
		query = query.Where("display_name ILIKE ?", "%"+*filter.DisplayName+"%")
	}
	if filter.Description != nil {
		query = query.Where("description ILIKE ?", "%"+*filter.Description+"%")
	}
	if filter.Version != nil {
		query = query.Where("version ILIKE ?", "%"+*filter.Version+"%")
	}

	// Filters with exact match
	if filter.TenantID != nil {
		query = query.Joins("JOIN tenant_services ON services.service_id = tenant_services.service_id").
			Where("tenant_services.tenant_id = ?", *filter.TenantID)
	}
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
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

	// Pagination
	offset := (filter.Page - 1) * filter.Limit
	var services []model.Service
	if err := query.Limit(filter.Limit).Offset(offset).Find(&services).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[model.Service]{
		Data:       services,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *serviceRepository) FindServicesByPolicyUUID(policyUUID uuid.UUID, filter ServiceRepositoryGetFilter) (*PaginationResult[model.Service], error) {
	query := r.DB().Model(&model.Service{}).
		Joins("INNER JOIN service_policies ON services.service_id = service_policies.service_id").
		Joins("INNER JOIN policies ON service_policies.policy_id = policies.policy_id").
		Where("policies.policy_uuid = ?", policyUUID)

	// Apply filters with LIKE
	if filter.Name != nil {
		query = query.Where("services.name ILIKE ?", "%"+*filter.Name+"%")
	}
	if filter.DisplayName != nil {
		query = query.Where("services.display_name ILIKE ?", "%"+*filter.DisplayName+"%")
	}
	if filter.Description != nil {
		query = query.Where("services.description ILIKE ?", "%"+*filter.Description+"%")
	}
	if filter.Version != nil {
		query = query.Where("services.version ILIKE ?", "%"+*filter.Version+"%")
	}

	// Status filter (multiple values)
	if len(filter.Status) > 0 {
		query = query.Where("services.status IN ?", filter.Status)
	}

	// Boolean filters
	if filter.IsSystem != nil {
		query = query.Where("services.is_system = ?", *filter.IsSystem)
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrderPrefixed("services.", filter.SortBy, filter.SortOrder, "services.created_at DESC"))

	// Pagination
	offset := (filter.Page - 1) * filter.Limit
	var services []model.Service
	if err := query.Limit(filter.Limit).Offset(offset).Find(&services).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[model.Service]{
		Data:       services,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *serviceRepository) SetStatusByUUID(serviceUUID uuid.UUID, status string) error {
	return r.DB().Model(&model.Service{}).
		Where("service_uuid = ?", serviceUUID).
		Update("status", status).Error
}

func (r *serviceRepository) CountPoliciesByServiceID(serviceID int64) (int64, error) {
	var count int64
	err := r.DB().Model(&model.ServicePolicy{}).
		Where("service_id = ?", serviceID).
		Count(&count).Error
	return count, err
}

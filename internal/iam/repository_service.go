package iam

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/database"
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
	BaseRepositoryMethods[Service]
	FindByUUID(uuid any, preloads ...string) (*Service, error)
	DeleteByUUID(uuid any) error
	WithTx(tx *gorm.DB) ServiceRepository
	FindByName(serviceName string) (*Service, error)
	FindByNameAndTenantID(serviceName string, tenantID int64) (*Service, error)
	FindByTenantID(tenantID int64) ([]Service, error)
	FindPaginated(filter ServiceRepositoryGetFilter) (*PaginationResult[Service], error)
	FindServicesByPolicyUUID(policyUUID uuid.UUID, filter ServiceRepositoryGetFilter) (*PaginationResult[Service], error)
	SetStatusByUUID(serviceUUID uuid.UUID, status string) error
	CountPoliciesByServiceID(serviceID int64) (int64, error)
}

type serviceRepository struct {
	*BaseRepository[Service]
}

func NewServiceRepository(db *gorm.DB) ServiceRepository {
	return &serviceRepository{
		BaseRepository: database.NewBaseRepository[Service](db, "service_uuid", "service_id"),
	}
}

func (r *serviceRepository) WithTx(tx *gorm.DB) ServiceRepository {
	return &serviceRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *serviceRepository) FindByName(serviceName string) (*Service, error) {
	var service Service
	err := r.DB().Where("name = ?", serviceName).First(&service).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &service, nil
}

func (r *serviceRepository) FindByNameAndTenantID(serviceName string, tenantID int64) (*Service, error) {
	var service Service
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

func (r *serviceRepository) FindByTenantID(tenantID int64) ([]Service, error) {
	var services []Service
	err := r.DB().
		Joins("JOIN tenant_services ON services.service_id = tenant_services.service_id").
		Where("tenant_services.tenant_id = ?", tenantID).
		Find(&services).Error
	return services, err
}

func (r *serviceRepository) FindPaginated(filter ServiceRepositoryGetFilter) (*PaginationResult[Service], error) {
	if filter.TenantID == nil || *filter.TenantID == 0 {
		return nil, fmt.Errorf("tenant_id is required")
	}

	query := r.DB().Model(&Service{})

	// Filters with LIKE
	query = database.ApplyILike(query, "name", filter.Name)
	query = database.ApplyILike(query, "display_name", filter.DisplayName)
	query = database.ApplyILike(query, "description", filter.Description)
	query = database.ApplyILike(query, "version", filter.Version)

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
	query = query.Order(database.SanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	return database.PaginateQuery[Service](query, filter.Page, filter.Limit)
}

func (r *serviceRepository) FindServicesByPolicyUUID(policyUUID uuid.UUID, filter ServiceRepositoryGetFilter) (*PaginationResult[Service], error) {
	query := r.DB().Model(&Service{}).
		Joins("INNER JOIN service_policies ON services.service_id = service_policies.service_id").
		Joins("INNER JOIN policies ON service_policies.policy_id = policies.policy_id").
		Where("policies.policy_uuid = ?", policyUUID)

	// Apply filters with LIKE
	query = database.ApplyILike(query, "services.name", filter.Name)
	query = database.ApplyILike(query, "services.display_name", filter.DisplayName)
	query = database.ApplyILike(query, "services.description", filter.Description)
	query = database.ApplyILike(query, "services.version", filter.Version)

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
	query = query.Order(database.SanitizeOrderPrefixed("services.", filter.SortBy, filter.SortOrder, "services.created_at DESC"))

	return database.PaginateQuery[Service](query, filter.Page, filter.Limit)
}

func (r *serviceRepository) SetStatusByUUID(serviceUUID uuid.UUID, status string) error {
	return r.DB().Model(&Service{}).
		Where("service_uuid = ?", serviceUUID).
		Update("status", status).Error
}

func (r *serviceRepository) CountPoliciesByServiceID(serviceID int64) (int64, error) {
	var count int64
	err := r.DB().Model(&ServicePolicy{}).
		Where("service_id = ?", serviceID).
		Count(&count).Error
	return count, err
}

package iam

import (
	"errors"

	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

type ServicePolicyRepositoryGetFilter struct {
	ServiceID *int64
	PolicyID  *int64
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type ServicePolicyRepository interface {
	BaseRepositoryMethods[ServicePolicy]
	WithTx(tx *gorm.DB) ServicePolicyRepository
	FindPaginated(filter ServicePolicyRepositoryGetFilter) (*PaginationResult[ServicePolicy], error)
	FindByServiceAndPolicy(serviceID int64, policyID int64) (*ServicePolicy, error)
	DeleteByServiceAndPolicy(serviceID int64, policyID int64) error
	FindPoliciesByServiceID(serviceID int64) ([]Policy, error)
	FindServicesByPolicyID(policyID int64) ([]Service, error)
}

type servicePolicyRepository struct {
	*BaseRepository[ServicePolicy]
}

func NewServicePolicyRepository(db *gorm.DB) ServicePolicyRepository {
	return &servicePolicyRepository{
		BaseRepository: database.NewBaseRepository[ServicePolicy](db, "service_policy_uuid", "service_policy_id"),
	}
}

func (r *servicePolicyRepository) WithTx(tx *gorm.DB) ServicePolicyRepository {
	return &servicePolicyRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *servicePolicyRepository) FindByServiceAndPolicy(serviceID int64, policyID int64) (*ServicePolicy, error) {
	var servicePolicy ServicePolicy
	err := r.DB().Where("service_id = ? AND policy_id = ?", serviceID, policyID).First(&servicePolicy).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &servicePolicy, nil
}

func (r *servicePolicyRepository) DeleteByServiceAndPolicy(serviceID int64, policyID int64) error {
	return r.DB().Where("service_id = ? AND policy_id = ?", serviceID, policyID).Delete(&ServicePolicy{}).Error
}

func (r *servicePolicyRepository) FindPoliciesByServiceID(serviceID int64) ([]Policy, error) {
	var policies []Policy
	err := r.DB().Table("policies").
		Joins("INNER JOIN service_policies ON policies.policy_id = service_policies.policy_id").
		Where("service_policies.service_id = ?", serviceID).
		Find(&policies).Error
	return policies, err
}

func (r *servicePolicyRepository) FindServicesByPolicyID(policyID int64) ([]Service, error) {
	var services []Service
	err := r.DB().Table("services").
		Joins("INNER JOIN service_policies ON services.service_id = service_policies.service_id").
		Where("service_policies.policy_id = ?", policyID).
		Find(&services).Error
	return services, err
}

func (r *servicePolicyRepository) FindPaginated(filter ServicePolicyRepositoryGetFilter) (*PaginationResult[ServicePolicy], error) {
	query := r.DB().Model(&ServicePolicy{})

	// Apply filters
	if filter.ServiceID != nil {
		query = query.Where("service_id = ?", *filter.ServiceID)
	}
	if filter.PolicyID != nil {
		query = query.Where("policy_id = ?", *filter.PolicyID)
	}

	query = query.Order(database.SanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))
	return database.PaginateQuery[ServicePolicy](query, filter.Page, filter.Limit)
}

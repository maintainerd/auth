package repository

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
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
	BaseRepositoryMethods[model.ServicePolicy]
	WithTx(tx *gorm.DB) ServicePolicyRepository
	FindPaginated(filter ServicePolicyRepositoryGetFilter) (*PaginationResult[model.ServicePolicy], error)
	FindByServiceAndPolicy(serviceID int64, policyID int64) (*model.ServicePolicy, error)
	DeleteByServiceAndPolicy(serviceID int64, policyID int64) error
	FindPoliciesByServiceID(serviceID int64) ([]model.Policy, error)
	FindServicesByPolicyID(policyID int64) ([]model.Service, error)
}

type servicePolicyRepository struct {
	*BaseRepository[model.ServicePolicy]
	db *gorm.DB
}

func NewServicePolicyRepository(db *gorm.DB) ServicePolicyRepository {
	return &servicePolicyRepository{
		BaseRepository: NewBaseRepository[model.ServicePolicy](db, "service_policy_uuid", "service_policy_id"),
		db:             db,
	}
}

func (r *servicePolicyRepository) WithTx(tx *gorm.DB) ServicePolicyRepository {
	return &servicePolicyRepository{
		BaseRepository: NewBaseRepository[model.ServicePolicy](tx, "service_policy_uuid", "service_policy_id"),
		db:             tx,
	}
}

func (r *servicePolicyRepository) FindByServiceAndPolicy(serviceID int64, policyID int64) (*model.ServicePolicy, error) {
	var servicePolicy model.ServicePolicy
	err := r.db.Where("service_id = ? AND policy_id = ?", serviceID, policyID).First(&servicePolicy).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &servicePolicy, nil
}

func (r *servicePolicyRepository) DeleteByServiceAndPolicy(serviceID int64, policyID int64) error {
	return r.db.Where("service_id = ? AND policy_id = ?", serviceID, policyID).Delete(&model.ServicePolicy{}).Error
}

func (r *servicePolicyRepository) FindPoliciesByServiceID(serviceID int64) ([]model.Policy, error) {
	var policies []model.Policy
	err := r.db.Table("policies").
		Joins("INNER JOIN service_policies ON policies.policy_id = service_policies.policy_id").
		Where("service_policies.service_id = ?", serviceID).
		Find(&policies).Error
	return policies, err
}

func (r *servicePolicyRepository) FindServicesByPolicyID(policyID int64) ([]model.Service, error) {
	var services []model.Service
	err := r.db.Table("services").
		Joins("INNER JOIN service_policies ON services.service_id = service_policies.service_id").
		Where("service_policies.policy_id = ?", policyID).
		Find(&services).Error
	return services, err
}

func (r *servicePolicyRepository) FindPaginated(filter ServicePolicyRepositoryGetFilter) (*PaginationResult[model.ServicePolicy], error) {
	query := r.db.Model(&model.ServicePolicy{})

	// Apply filters
	if filter.ServiceID != nil {
		query = query.Where("service_id = ?", *filter.ServiceID)
	}
	if filter.PolicyID != nil {
		query = query.Where("policy_id = ?", *filter.PolicyID)
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply sorting
	if filter.SortBy != "" {
		order := filter.SortBy
		if filter.SortOrder == "desc" {
			order += " DESC"
		} else {
			order += " ASC"
		}
		query = query.Order(order)
	} else {
		query = query.Order("created_at DESC")
	}

	// Apply pagination
	offset := (filter.Page - 1) * filter.Limit
	var servicePolicies []model.ServicePolicy
	if err := query.Limit(filter.Limit).Offset(offset).Find(&servicePolicies).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[model.ServicePolicy]{
		Data:       servicePolicies,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

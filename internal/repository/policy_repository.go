package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type PolicyRepositoryGetFilter struct {
	TenantID    int64
	Name        *string
	Description *string
	Version     *string
	Status      []string
	IsDefault   *bool
	IsSystem    *bool
	ServiceID   *uuid.UUID
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type PolicyRepository interface {
	BaseRepositoryMethods[model.Policy]
	WithTx(tx *gorm.DB) PolicyRepository
	FindByUUIDAndTenantID(policyUUID uuid.UUID, tenantID int64) (*model.Policy, error)
	FindByName(policyName string, tenantID int64) (*model.Policy, error)
	FindByNameAndVersion(policyName string, version string, tenantID int64) (*model.Policy, error)
	FindDefaultPolicies(tenantID int64) ([]model.Policy, error)
	FindSystemPolicies(tenantID int64) ([]model.Policy, error)
	FindPaginated(filter PolicyRepositoryGetFilter) (*PaginationResult[model.Policy], error)
	SetStatusByUUID(policyUUID uuid.UUID, tenantID int64, status string) error
	SetDefaultStatusByUUID(policyUUID uuid.UUID, tenantID int64, isDefault bool) error
	SetSystemStatusByUUID(policyUUID uuid.UUID, tenantID int64, isSystem bool) error
	DeleteByUUIDAndTenantID(policyUUID uuid.UUID, tenantID int64) error
}

type policyRepository struct {
	*BaseRepository[model.Policy]
	db *gorm.DB
}

func NewPolicyRepository(db *gorm.DB) PolicyRepository {
	return &policyRepository{
		BaseRepository: NewBaseRepository[model.Policy](db, "policy_uuid", "policy_id"),
		db:             db,
	}
}

func (r *policyRepository) WithTx(tx *gorm.DB) PolicyRepository {
	return &policyRepository{
		BaseRepository: NewBaseRepository[model.Policy](tx, "policy_uuid", "policy_id"),
		db:             tx,
	}
}

func (r *policyRepository) FindByUUIDAndTenantID(policyUUID uuid.UUID, tenantID int64) (*model.Policy, error) {
	var policy model.Policy
	err := r.db.Where("policy_uuid = ? AND tenant_id = ?", policyUUID, tenantID).First(&policy).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &policy, nil
}

func (r *policyRepository) FindByName(policyName string, tenantID int64) (*model.Policy, error) {
	var policy model.Policy
	err := r.db.Where("name = ? AND tenant_id = ?", policyName, tenantID).First(&policy).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &policy, nil
}

func (r *policyRepository) FindByNameAndVersion(policyName string, version string, tenantID int64) (*model.Policy, error) {
	var policy model.Policy
	err := r.db.Where("name = ? AND version = ? AND tenant_id = ?", policyName, version, tenantID).First(&policy).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &policy, nil
}

func (r *policyRepository) FindDefaultPolicies(tenantID int64) ([]model.Policy, error) {
	var policies []model.Policy
	err := r.db.Where("is_default = ? AND tenant_id = ?", true, tenantID).Find(&policies).Error
	return policies, err
}

func (r *policyRepository) FindSystemPolicies(tenantID int64) ([]model.Policy, error) {
	var policies []model.Policy
	err := r.db.Where("is_system = ? AND tenant_id = ?", true, tenantID).Find(&policies).Error
	return policies, err
}

func (r *policyRepository) SetStatusByUUID(policyUUID uuid.UUID, tenantID int64, status string) error {
	return r.db.Model(&model.Policy{}).
		Where("policy_uuid = ? AND tenant_id = ?", policyUUID, tenantID).
		Update("status", status).Error
}

func (r *policyRepository) SetDefaultStatusByUUID(policyUUID uuid.UUID, tenantID int64, isDefault bool) error {
	return r.db.Model(&model.Policy{}).
		Where("policy_uuid = ? AND tenant_id = ?", policyUUID, tenantID).
		Update("is_default", isDefault).Error
}

func (r *policyRepository) SetSystemStatusByUUID(policyUUID uuid.UUID, tenantID int64, isSystem bool) error {
	return r.db.Model(&model.Policy{}).
		Where("policy_uuid = ? AND tenant_id = ?", policyUUID, tenantID).
		Update("is_system", isSystem).Error
}

func (r *policyRepository) FindPaginated(filter PolicyRepositoryGetFilter) (*PaginationResult[model.Policy], error) {
	query := r.db.Model(&model.Policy{})

	// Filter by tenant_id
	query = query.Where("tenant_id = ?", filter.TenantID)

	// Apply filters
	if filter.Name != nil {
		query = query.Where("name ILIKE ?", "%"+*filter.Name+"%")
	}
	if filter.Description != nil {
		query = query.Where("description ILIKE ?", "%"+*filter.Description+"%")
	}
	if filter.Version != nil {
		query = query.Where("version ILIKE ?", "%"+*filter.Version+"%")
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
	if filter.ServiceID != nil {
		// Join with service_policies and services tables to filter by service UUID
		query = query.Joins("INNER JOIN service_policies ON policies.policy_id = service_policies.policy_id").
			Joins("INNER JOIN services ON service_policies.service_id = services.service_id").
			Where("services.service_uuid = ?", *filter.ServiceID)
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

	// Pagination
	offset := (filter.Page - 1) * filter.Limit
	var policies []model.Policy
	if err := query.Limit(filter.Limit).Offset(offset).Find(&policies).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[model.Policy]{
		Data:       policies,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *policyRepository) DeleteByUUIDAndTenantID(policyUUID uuid.UUID, tenantID int64) error {
	return r.db.Where("policy_uuid = ? AND tenant_id = ?", policyUUID, tenantID).Delete(&model.Policy{}).Error
}

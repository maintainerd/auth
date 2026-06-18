package iam

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type PolicyRepositoryGetFilter struct {
	TenantID    int64
	Name        *string
	Description *string
	Version     *string
	Status      []string
	IsSystem    *bool
	ServiceID   *uuid.UUID
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type PolicyRepository interface {
	BaseRepositoryMethods[Policy]
	UpdateByUUID(uuid any, updatedData any) (*Policy, error)
	WithTx(tx *gorm.DB) PolicyRepository
	FindByUUIDAndTenantID(policyUUID uuid.UUID, tenantID int64) (*Policy, error)
	FindByName(policyName string, tenantID int64) (*Policy, error)
	FindByNameAndVersion(policyName string, version string, tenantID int64) (*Policy, error)
	FindSystemPolicies(tenantID int64) ([]Policy, error)
	FindPaginated(filter PolicyRepositoryGetFilter) (*PaginationResult[Policy], error)
	SetStatusByUUID(policyUUID uuid.UUID, tenantID int64, status string) error
	SetSystemStatusByUUID(policyUUID uuid.UUID, tenantID int64, isSystem bool) error
	DeleteByUUIDAndTenantID(policyUUID uuid.UUID, tenantID int64) error
}

type policyRepository struct {
	*BaseRepository[Policy]
}

func NewPolicyRepository(db *gorm.DB) PolicyRepository {
	return &policyRepository{
		BaseRepository: database.NewBaseRepository[Policy](db, "policy_uuid", "policy_id"),
	}
}

func (r *policyRepository) WithTx(tx *gorm.DB) PolicyRepository {
	return &policyRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *policyRepository) FindByUUIDAndTenantID(policyUUID uuid.UUID, tenantID int64) (*Policy, error) {
	var policy Policy
	err := r.DB().Where("policy_uuid = ? AND tenant_id = ?", policyUUID, tenantID).First(&policy).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &policy, nil
}

func (r *policyRepository) FindByName(policyName string, tenantID int64) (*Policy, error) {
	var policy Policy
	err := r.DB().Where("name = ? AND tenant_id = ?", policyName, tenantID).First(&policy).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &policy, nil
}

func (r *policyRepository) FindByNameAndVersion(policyName string, version string, tenantID int64) (*Policy, error) {
	var policy Policy
	err := r.DB().Where("name = ? AND version = ? AND tenant_id = ?", policyName, version, tenantID).First(&policy).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &policy, nil
}

func (r *policyRepository) FindSystemPolicies(tenantID int64) ([]Policy, error) {
	var policies []Policy
	err := r.DB().Where("is_system = ? AND tenant_id = ?", true, tenantID).Find(&policies).Error
	return policies, err
}

func (r *policyRepository) SetStatusByUUID(policyUUID uuid.UUID, tenantID int64, status string) error {
	return r.DB().Model(&Policy{}).
		Where("policy_uuid = ? AND tenant_id = ?", policyUUID, tenantID).
		Update("status", status).Error
}

func (r *policyRepository) SetSystemStatusByUUID(policyUUID uuid.UUID, tenantID int64, isSystem bool) error {
	return r.DB().Model(&Policy{}).
		Where("policy_uuid = ? AND tenant_id = ?", policyUUID, tenantID).
		Update("is_system", isSystem).Error
}

func (r *policyRepository) FindPaginated(filter PolicyRepositoryGetFilter) (*PaginationResult[Policy], error) {
	query := r.DB().Model(&Policy{})

	// Filter by tenant_id
	query = query.Where("tenant_id = ?", filter.TenantID)

	// Apply filters
	query = database.ApplyILike(query, "name", filter.Name)
	query = database.ApplyILike(query, "description", filter.Description)
	query = database.ApplyILike(query, "version", filter.Version)
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
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

	query = query.Order(database.SanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))
	return database.PaginateQuery[Policy](query, filter.Page, filter.Limit)
}

func (r *policyRepository) DeleteByUUIDAndTenantID(policyUUID uuid.UUID, tenantID int64) error {
	return r.DB().Where("policy_uuid = ? AND tenant_id = ?", policyUUID, tenantID).Delete(&Policy{}).Error
}

package iam

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
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
	FindPaginated(filter PolicyRepositoryGetFilter) (*PaginationResult[Policy], error)
	SetStatusByUUID(policyUUID uuid.UUID, tenantID int64, status string) error
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

// FindSystemPolicies and SetSystemStatusByUUID are gone: both were zero-caller.
// is_system decides whether a policy can be edited or deleted at all, so a setter
// for it should not sit in the interface until something actually needs one.
func (r *policyRepository) SetStatusByUUID(policyUUID uuid.UUID, tenantID int64, status string) error {
	return r.DB().Model(&Policy{}).
		Where("policy_uuid = ? AND tenant_id = ?", policyUUID, tenantID).
		Update("status", status).Error
}

// policySortColumns is this table's own sort allowlist. The global set in
// platform/database is a union across every table, so ordering by a column
// `policies` does not have reaches Postgres as an undefined column (42703) and
// surfaces as a 500 rather than a 400.
var policySortColumns = map[string]struct{}{
	"created_at": {}, "updated_at": {}, "name": {}, "version": {},
	"status": {}, "is_system": {}, "tenant_id": {},
}

func (r *policyRepository) FindPaginated(filter PolicyRepositoryGetFilter) (*PaginationResult[Policy], error) {
	query := r.DB().Model(&Policy{})

	// EVERY predicate is table-qualified. The service_id filter below joins
	// `services`, which also has tenant_id, name, description, status and is_system —
	// so an unqualified predicate becomes "column reference is ambiguous" (42702) and
	// GET /policies?service_id=… was an unconditional 500.
	query = query.Where("policies.tenant_id = ?", filter.TenantID)

	query = database.ApplyILike(query, "policies.name", filter.Name)
	query = database.ApplyILike(query, "policies.description", filter.Description)
	query = database.ApplyILike(query, "policies.version", filter.Version)
	if len(filter.Status) > 0 {
		query = query.Where("policies.status IN ?", filter.Status)
	}
	if filter.IsSystem != nil {
		query = query.Where("policies.is_system = ?", *filter.IsSystem)
	}
	if filter.ServiceID != nil {
		// Join with service_policies and services tables to filter by service UUID.
		// Scoped to live rows: a soft-deleted service must not keep surfacing its
		// policies in this filter.
		query = query.Joins("INNER JOIN service_policies ON policies.policy_id = service_policies.policy_id").
			Joins("INNER JOIN services ON service_policies.service_id = services.service_id AND services.deleted_at IS NULL").
			Where("services.service_uuid = ?", *filter.ServiceID)
	}

	query = query.Order(database.SanitizeOrderInPrefixed(policySortColumns, "policies.", filter.SortBy, filter.SortOrder, "policies.created_at DESC"))
	return database.PaginateQuery[Policy](query, filter.Page, filter.Limit)
}

func (r *policyRepository) DeleteByUUIDAndTenantID(policyUUID uuid.UUID, tenantID int64) error {
	return r.DB().Where("policy_uuid = ? AND tenant_id = ?", policyUUID, tenantID).Delete(&Policy{}).Error
}

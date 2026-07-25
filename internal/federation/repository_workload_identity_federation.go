package federation

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

// workloadIdentityFederationSortColumns is this table's own sort allowlist. The
// global set in platform/database is a union across every table, so it contains
// columns this table does not have — status, email, identifier, is_default,
// event_type and 16 others. Ordering by one of those reached Postgres as an
// undefined column and surfaced as a 500 rather than a 400.
var workloadIdentityFederationSortColumns = map[string]struct{}{
	"created_at": {}, "updated_at": {}, "name": {}, "is_active": {},
	"issuer_url": {}, "audience": {}, "subject_claim": {},
	"subject_pattern": {}, "tenant_id": {}, "client_id": {},
}

// WorkloadIdentityFederationRepositoryGetFilter holds query parameters for
// paginated workload identity federation lookups.
type WorkloadIdentityFederationRepositoryGetFilter struct {
	TenantID  *int64
	Name      *string
	IsActive  *bool
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

// WorkloadIdentityFederationRepository defines persistence operations for the
// workload_identity_federations entity.
type WorkloadIdentityFederationRepository interface {
	BaseRepositoryMethods[WorkloadIdentityFederation]
	WithTx(tx *gorm.DB) WorkloadIdentityFederationRepository
	UpdateByUUID(uuid any, updatedData any) (*WorkloadIdentityFederation, error)
	DeleteByUUID(uuid any) error
	FindByUUIDAndTenantID(federationUUID uuid.UUID, tenantID int64) (*WorkloadIdentityFederation, error)
	FindPaginated(filter WorkloadIdentityFederationRepositoryGetFilter) (*PaginationResult[WorkloadIdentityFederation], error)
	// FindActiveByIssuer returns every active (non-deleted) federation that
	// trusts the given issuer_url, across tenants. The token-exchange flow uses
	// this to resolve which federation (if any) a presented external token maps
	// to before verifying its signature.
	FindActiveByIssuer(issuerURL string) ([]WorkloadIdentityFederation, error)
}

type workloadIdentityFederationRepository struct {
	*BaseRepository[WorkloadIdentityFederation]
}

// NewWorkloadIdentityFederationRepository creates a new repository backed by db.
func NewWorkloadIdentityFederationRepository(db *gorm.DB) WorkloadIdentityFederationRepository {
	return &workloadIdentityFederationRepository{
		BaseRepository: database.NewBaseRepository[WorkloadIdentityFederation](
			db, "workload_identity_federation_uuid", "workload_identity_federation_id",
		),
	}
}

// WithTx returns a copy of the repository bound to the supplied transaction.
func (r *workloadIdentityFederationRepository) WithTx(tx *gorm.DB) WorkloadIdentityFederationRepository {
	return &workloadIdentityFederationRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByUUIDAndTenantID retrieves a single federation by UUID scoped to a
// tenant. Returns nil, nil when no record exists.
func (r *workloadIdentityFederationRepository) FindByUUIDAndTenantID(federationUUID uuid.UUID, tenantID int64) (*WorkloadIdentityFederation, error) {
	var federation WorkloadIdentityFederation
	err := r.DB().
		Where("workload_identity_federation_uuid = ? AND tenant_id = ?", federationUUID, tenantID).
		First(&federation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &federation, nil
}

// FindPaginated retrieves paginated federations with tenant scoping.
func (r *workloadIdentityFederationRepository) FindPaginated(filter WorkloadIdentityFederationRepositoryGetFilter) (*PaginationResult[WorkloadIdentityFederation], error) {
	if filter.TenantID == nil || *filter.TenantID == 0 {
		return nil, fmt.Errorf("tenant_id is required")
	}

	query := r.DB().Model(&WorkloadIdentityFederation{}).
		Where("tenant_id = ?", *filter.TenantID)

	query = database.ApplyILike(query, "name", filter.Name)
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	query = query.Order(database.SanitizeOrderIn(workloadIdentityFederationSortColumns, filter.SortBy, filter.SortOrder, "created_at DESC"))

	return database.PaginateQuery[WorkloadIdentityFederation](query, filter.Page, filter.Limit)
}

// FindActiveByIssuer returns all active, non-deleted federations trusting the
// given issuer_url.
func (r *workloadIdentityFederationRepository) FindActiveByIssuer(issuerURL string) ([]WorkloadIdentityFederation, error) {
	var federations []WorkloadIdentityFederation
	// Deterministic order. The exchange path resolves a tenant from these rows, so
	// leaving the order to Postgres meant the winning federation could change after
	// a routine UPDATE rewrote a tuple — a silent, config-free change of behaviour
	// on an unauthenticated endpoint.
	err := r.DB().
		Where("issuer_url = ? AND is_active = ?", issuerURL, true).
		Order("workload_identity_federation_id ASC").
		Find(&federations).Error
	if err != nil {
		return nil, err
	}
	return federations, nil
}

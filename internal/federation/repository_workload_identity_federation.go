package federation

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

// WorkloadIdentityFederationRepositoryGetFilter holds query parameters for
// paginated workload identity federation lookups.
type WorkloadIdentityFederationRepositoryGetFilter struct {
	TenantID  *int64
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
		Where("tenant_id = ?", *filter.TenantID).
		Order(database.SanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	return database.PaginateQuery[WorkloadIdentityFederation](query, filter.Page, filter.Limit)
}

// FindActiveByIssuer returns all active, non-deleted federations trusting the
// given issuer_url.
func (r *workloadIdentityFederationRepository) FindActiveByIssuer(issuerURL string) ([]WorkloadIdentityFederation, error) {
	var federations []WorkloadIdentityFederation
	err := r.DB().
		Where("issuer_url = ? AND is_active = ?", issuerURL, true).
		Find(&federations).Error
	if err != nil {
		return nil, err
	}
	return federations, nil
}

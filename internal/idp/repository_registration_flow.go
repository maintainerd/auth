package idp

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

// registrationFlowSortColumns is this table's own sort allowlist. The global set
// in platform/database is a union across every table and still contains columns
// registration_flows does not have (notably "identifier", removed when the flow
// name became the public selector) — ordering by one of those would reach
// Postgres as an undefined column and surface as a 500.
var registrationFlowSortColumns = map[string]struct{}{
	"created_at": {}, "updated_at": {}, "name": {}, "status": {},
	"is_system": {}, "client_id": {}, "tenant_id": {},
}

type RegistrationFlowRepositoryGetFilter struct {
	Name      *string
	Search    *string
	Status    []string
	TenantID  *int64
	ClientID  *int64
	IsSystem  *bool
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type RegistrationFlowRepository interface {
	BaseRepositoryMethods[RegistrationFlow]
	DeleteByUUID(uuid any) error
	FindByID(id any, preloads ...string) (*RegistrationFlow, error)
	WithTx(tx *gorm.DB) RegistrationFlowRepository
	FindPaginated(filter RegistrationFlowRepositoryGetFilter) (*PaginationResult[RegistrationFlow], error)
	FindByUUIDAndTenantID(registrationFlowUUID uuid.UUID, tenantID int64, preloads ...string) (*RegistrationFlow, error)
	// FindByNameAndClientTenant resolves the public registration-link selector
	// (the flow name). Both the client AND the tenant are part of the predicate:
	// the client alone proves existence, not ownership, and this is the highest
	// privilege path in the domain (it decides which roles a new user gets).
	FindByNameAndClientTenant(name string, clientID, tenantID int64) (*RegistrationFlow, error)
	// FindByNameAndTenantID backs the uniqueness pre-check. It is tenant-scoped
	// to match uq_registration_flows_tenant_name exactly, so a collision
	// surfaces as 409 rather than a driver-level 500.
	FindByNameAndTenantID(name string, tenantID int64) (*RegistrationFlow, error)
}

type registrationFlowRepository struct {
	*BaseRepository[RegistrationFlow]
}

func NewRegistrationFlowRepository(db *gorm.DB) RegistrationFlowRepository {
	return &registrationFlowRepository{
		BaseRepository: database.NewBaseRepository[RegistrationFlow](db, "registration_flow_uuid", "registration_flow_id"),
	}
}

func (r *registrationFlowRepository) WithTx(tx *gorm.DB) RegistrationFlowRepository {
	return &registrationFlowRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *registrationFlowRepository) FindPaginated(filter RegistrationFlowRepositoryGetFilter) (*PaginationResult[RegistrationFlow], error) {
	if filter.TenantID == nil || *filter.TenantID == 0 {
		return nil, fmt.Errorf("tenant_id is required")
	}

	query := r.DB().Model(&RegistrationFlow{})

	// Free-text search spans name and description. The name is also the value an
	// integrator pastes back from a broken registration URL, so it is the
	// primary support lookup.
	if filter.Search != nil && strings.TrimSpace(*filter.Search) != "" {
		term := "%" + strings.ToLower(strings.TrimSpace(*filter.Search)) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ?", term, term)
	} else if filter.Name != nil && *filter.Name != "" {
		query = database.ApplyILike(query, "name", filter.Name)
	}
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
	}
	if filter.ClientID != nil {
		query = query.Where("client_id = ?", *filter.ClientID)
	}
	if filter.IsSystem != nil {
		query = query.Where("is_system = ?", *filter.IsSystem)
	}

	// Apply sorting — protected against SQL injection via allowlist
	query = query.Order(database.SanitizeOrderIn(registrationFlowSortColumns, filter.SortBy, filter.SortOrder, "created_at DESC")).Preload("Client")

	return database.PaginateQuery[RegistrationFlow](query, filter.Page, filter.Limit)
}

func (r *registrationFlowRepository) FindByNameAndClientTenant(name string, clientID, tenantID int64) (*RegistrationFlow, error) {
	var registrationFlow RegistrationFlow
	err := r.DB().
		Where("name = ? AND client_id = ? AND tenant_id = ?", name, clientID, tenantID).
		First(&registrationFlow).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &registrationFlow, nil
}

func (r *registrationFlowRepository) FindByUUIDAndTenantID(registrationFlowUUID uuid.UUID, tenantID int64, preloads ...string) (*RegistrationFlow, error) {
	var registrationFlow RegistrationFlow
	query := r.DB().Where("registration_flow_uuid = ? AND tenant_id = ?", registrationFlowUUID, tenantID)

	for _, preload := range preloads {
		query = query.Preload(preload)
	}

	err := query.First(&registrationFlow).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &registrationFlow, nil
}

func (r *registrationFlowRepository) FindByNameAndTenantID(name string, tenantID int64) (*RegistrationFlow, error) {
	var registrationFlow RegistrationFlow
	err := r.DB().Where("name = ? AND tenant_id = ?", name, tenantID).First(&registrationFlow).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &registrationFlow, nil
}

package idp

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

type RegistrationFlowRepositoryGetFilter struct {
	Name       *string
	Identifier *string
	Status     []string
	TenantID   *int64
	ClientID   *int64
	Page       int
	Limit      int
	SortBy     string
	SortOrder  string
}

type RegistrationFlowRepository interface {
	BaseRepositoryMethods[RegistrationFlow]
	DeleteByUUID(uuid any) error
	FindByID(id any, preloads ...string) (*RegistrationFlow, error)
	WithTx(tx *gorm.DB) RegistrationFlowRepository
	FindPaginated(filter RegistrationFlowRepositoryGetFilter) (*PaginationResult[RegistrationFlow], error)
	FindByUUIDAndTenantID(registrationFlowUUID uuid.UUID, tenantID int64, preloads ...string) (*RegistrationFlow, error)
	FindByIdentifierAndClientID(identifier string, clientID int64) (*RegistrationFlow, error)
	FindByName(name string) (*RegistrationFlow, error)
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

	// Apply filters
	if filter.Name != nil && *filter.Name != "" {
		query = database.ApplyILike(query, "name", filter.Name)
	}
	if filter.Identifier != nil && *filter.Identifier != "" {
		query = database.ApplyILike(query, "identifier", filter.Identifier)
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

	// Apply sorting — protected against SQL injection via allowlist
	query = query.Order(database.SanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC")).Preload("Client")

	return database.PaginateQuery[RegistrationFlow](query, filter.Page, filter.Limit)
}

func (r *registrationFlowRepository) FindByIdentifierAndClientID(identifier string, clientID int64) (*RegistrationFlow, error) {
	var registrationFlow RegistrationFlow
	err := r.DB().Where("identifier = ? AND client_id = ?", identifier, clientID).First(&registrationFlow).Error
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

func (r *registrationFlowRepository) FindByName(name string) (*RegistrationFlow, error) {
	var registrationFlow RegistrationFlow
	err := r.DB().Where("name = ?", name).First(&registrationFlow).Error
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

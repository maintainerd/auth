package idp

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type AuthFlowRepositoryGetFilter struct {
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

type AuthFlowRepository interface {
	BaseRepositoryMethods[AuthFlow]
	DeleteByUUID(uuid any) error
	WithTx(tx *gorm.DB) AuthFlowRepository
	FindPaginated(filter AuthFlowRepositoryGetFilter) (*PaginationResult[AuthFlow], error)
	FindByUUIDAndTenantID(authFlowUUID uuid.UUID, tenantID int64, preloads ...string) (*AuthFlow, error)
	FindByIdentifierAndClientID(identifier string, clientID int64) (*AuthFlow, error)
	FindByClientID(clientID int64) ([]AuthFlow, error)
	FindByName(name string) (*AuthFlow, error)
	FindByNameAndTenantID(name string, tenantID int64) (*AuthFlow, error)
}

type authFlowRepository struct {
	*BaseRepository[AuthFlow]
}

func NewAuthFlowRepository(db *gorm.DB) AuthFlowRepository {
	return &authFlowRepository{
		BaseRepository: database.NewBaseRepository[AuthFlow](db, "auth_flow_uuid", "auth_flow_id"),
	}
}

func (r *authFlowRepository) WithTx(tx *gorm.DB) AuthFlowRepository {
	return &authFlowRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *authFlowRepository) FindPaginated(filter AuthFlowRepositoryGetFilter) (*PaginationResult[AuthFlow], error) {
	if filter.TenantID == nil || *filter.TenantID == 0 {
		return nil, fmt.Errorf("tenant_id is required")
	}

	query := r.DB().Model(&AuthFlow{})

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
	query = query.Order(database.SanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC")).Preload("Client").Preload("Branding")

	return database.PaginateQuery[AuthFlow](query, filter.Page, filter.Limit)
}

func (r *authFlowRepository) FindByIdentifierAndClientID(identifier string, clientID int64) (*AuthFlow, error) {
	var authFlow AuthFlow
	err := r.DB().Where("identifier = ? AND client_id = ?", identifier, clientID).First(&authFlow).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &authFlow, nil
}

func (r *authFlowRepository) FindByClientID(clientID int64) ([]AuthFlow, error) {
	var flows []AuthFlow
	err := r.DB().Where("client_id = ?", clientID).Order("auth_flow_id ASC").Find(&flows).Error
	return flows, err
}

func (r *authFlowRepository) FindByUUIDAndTenantID(authFlowUUID uuid.UUID, tenantID int64, preloads ...string) (*AuthFlow, error) {
	var authFlow AuthFlow
	query := r.DB().Where("auth_flow_uuid = ? AND tenant_id = ?", authFlowUUID, tenantID)

	for _, preload := range preloads {
		query = query.Preload(preload)
	}

	err := query.First(&authFlow).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &authFlow, nil
}

func (r *authFlowRepository) FindByName(name string) (*AuthFlow, error) {
	var authFlow AuthFlow
	err := r.DB().Where("name = ?", name).First(&authFlow).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &authFlow, nil
}

func (r *authFlowRepository) FindByNameAndTenantID(name string, tenantID int64) (*AuthFlow, error) {
	var authFlow AuthFlow
	err := r.DB().Where("name = ? AND tenant_id = ?", name, tenantID).First(&authFlow).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &authFlow, nil
}

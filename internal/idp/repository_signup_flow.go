package idp

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type SignupFlowRepositoryGetFilter struct {
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

type SignupFlowRepository interface {
	BaseRepositoryMethods[SignupFlow]
	WithTx(tx *gorm.DB) SignupFlowRepository
	FindPaginated(filter SignupFlowRepositoryGetFilter) (*PaginationResult[SignupFlow], error)
	FindByUUIDAndTenantID(signupFlowUUID uuid.UUID, tenantID int64, preloads ...string) (*SignupFlow, error)
	FindByIdentifierAndClientID(identifier string, clientID int64) (*SignupFlow, error)
	FindByName(name string) (*SignupFlow, error)
}

type signupFlowRepository struct {
	*BaseRepository[SignupFlow]
}

func NewSignupFlowRepository(db *gorm.DB) SignupFlowRepository {
	return &signupFlowRepository{
		BaseRepository: database.NewBaseRepository[SignupFlow](db, "signup_flow_uuid", "signup_flow_id"),
	}
}

func (r *signupFlowRepository) WithTx(tx *gorm.DB) SignupFlowRepository {
	return &signupFlowRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *signupFlowRepository) FindPaginated(filter SignupFlowRepositoryGetFilter) (*PaginationResult[SignupFlow], error) {
	query := r.DB().Model(&SignupFlow{})

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

	return database.PaginateQuery[SignupFlow](query, filter.Page, filter.Limit)
}

func (r *signupFlowRepository) FindByIdentifierAndClientID(identifier string, clientID int64) (*SignupFlow, error) {
	var signupFlow SignupFlow
	err := r.DB().Where("identifier = ? AND client_id = ?", identifier, clientID).First(&signupFlow).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &signupFlow, nil
}

func (r *signupFlowRepository) FindByUUIDAndTenantID(signupFlowUUID uuid.UUID, tenantID int64, preloads ...string) (*SignupFlow, error) {
	var signupFlow SignupFlow
	query := r.DB().Where("signup_flow_uuid = ? AND tenant_id = ?", signupFlowUUID, tenantID)

	for _, preload := range preloads {
		query = query.Preload(preload)
	}

	err := query.First(&signupFlow).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &signupFlow, nil
}

func (r *signupFlowRepository) FindByName(name string) (*SignupFlow, error) {
	var signupFlow SignupFlow
	err := r.DB().Where("name = ?", name).First(&signupFlow).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &signupFlow, nil
}

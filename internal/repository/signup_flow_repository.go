package repository

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
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
	BaseRepositoryMethods[model.SignupFlow]
	WithTx(tx *gorm.DB) SignupFlowRepository
	FindPaginated(filter SignupFlowRepositoryGetFilter) (*PaginationResult[model.SignupFlow], error)
	FindByUUIDAndTenantID(signupFlowUUID uuid.UUID, tenantID int64, preloads ...string) (*model.SignupFlow, error)
	FindByIdentifierAndClientID(identifier string, clientID int64) (*model.SignupFlow, error)
	FindByName(name string) (*model.SignupFlow, error)
}

type signupFlowRepository struct {
	*BaseRepository[model.SignupFlow]
}

func NewSignupFlowRepository(db *gorm.DB) SignupFlowRepository {
	return &signupFlowRepository{
		BaseRepository: NewBaseRepository[model.SignupFlow](db, "signup_flow_uuid", "signup_flow_id"),
	}
}

func (r *signupFlowRepository) WithTx(tx *gorm.DB) SignupFlowRepository {
	return &signupFlowRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *signupFlowRepository) FindPaginated(filter SignupFlowRepositoryGetFilter) (*PaginationResult[model.SignupFlow], error) {
	var signupFlows []model.SignupFlow
	var total int64

	query := r.DB().Model(&model.SignupFlow{})

	// Apply filters
	if filter.Name != nil && *filter.Name != "" {
		query = query.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(*filter.Name)+"%")
	}
	if filter.Identifier != nil && *filter.Identifier != "" {
		query = query.Where("LOWER(identifier) LIKE ?", "%"+strings.ToLower(*filter.Identifier)+"%")
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

	// Count total before pagination
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	// Apply pagination
	offset := (filter.Page - 1) * filter.Limit
	query = query.Offset(offset).Limit(filter.Limit)

	// Execute query with preloads
	if err := query.Preload("Client").Find(&signupFlows).Error; err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int(total) / filter.Limit
	if int(total)%filter.Limit > 0 {
		totalPages++
	}

	return &PaginationResult[model.SignupFlow]{
		Data:       signupFlows,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *signupFlowRepository) FindByIdentifierAndClientID(identifier string, clientID int64) (*model.SignupFlow, error) {
	var signupFlow model.SignupFlow
	err := r.DB().Where("identifier = ? AND client_id = ?", identifier, clientID).First(&signupFlow).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &signupFlow, nil
}

func (r *signupFlowRepository) FindByUUIDAndTenantID(signupFlowUUID uuid.UUID, tenantID int64, preloads ...string) (*model.SignupFlow, error) {
	var signupFlow model.SignupFlow
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

func (r *signupFlowRepository) FindByName(name string) (*model.SignupFlow, error) {
	var signupFlow model.SignupFlow
	err := r.DB().Where("name = ?", name).First(&signupFlow).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &signupFlow, nil
}

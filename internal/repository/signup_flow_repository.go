package repository

import (
	"strings"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type SignupFlowRepositoryGetFilter struct {
	Name         *string
	Identifier   *string
	Status       []string
	AuthClientID *int64
	Page         int
	Limit        int
	SortBy       string
	SortOrder    string
}

type SignupFlowRepositoryGetResult struct {
	Data       []model.SignupFlow
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type SignupFlowRepository interface {
	BaseRepositoryMethods[model.SignupFlow]
	WithTx(tx *gorm.DB) SignupFlowRepository
	FindPaginated(filter SignupFlowRepositoryGetFilter) (*SignupFlowRepositoryGetResult, error)
	FindByIdentifierAndAuthClientID(identifier string, authClientID int64) (*model.SignupFlow, error)
	FindByName(name string) (*model.SignupFlow, error)
}

type signupFlowRepository struct {
	*BaseRepository[model.SignupFlow]
	db *gorm.DB
}

func NewSignupFlowRepository(db *gorm.DB) SignupFlowRepository {
	return &signupFlowRepository{
		BaseRepository: NewBaseRepository[model.SignupFlow](db, "signup_flow_uuid", "signup_flow_id"),
		db:             db,
	}
}

func (r *signupFlowRepository) WithTx(tx *gorm.DB) SignupFlowRepository {
	return &signupFlowRepository{
		BaseRepository: NewBaseRepository[model.SignupFlow](tx, "signup_flow_uuid", "signup_flow_id"),
		db:             tx,
	}
}

func (r *signupFlowRepository) FindPaginated(filter SignupFlowRepositoryGetFilter) (*SignupFlowRepositoryGetResult, error) {
	var signupFlows []model.SignupFlow
	var total int64

	query := r.db.Model(&model.SignupFlow{})

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
	if filter.AuthClientID != nil {
		query = query.Where("auth_client_id = ?", *filter.AuthClientID)
	}

	// Count total before pagination
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply sorting
	if filter.SortBy != "" {
		order := "ASC"
		if filter.SortOrder == "desc" {
			order = "DESC"
		}
		query = query.Order(filter.SortBy + " " + order)
	} else {
		query = query.Order("created_at DESC")
	}

	// Apply pagination
	offset := (filter.Page - 1) * filter.Limit
	query = query.Offset(offset).Limit(filter.Limit)

	// Execute query with preloads
	if err := query.Preload("AuthClient").Find(&signupFlows).Error; err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int(total) / filter.Limit
	if int(total)%filter.Limit > 0 {
		totalPages++
	}

	return &SignupFlowRepositoryGetResult{
		Data:       signupFlows,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *signupFlowRepository) FindByIdentifierAndAuthClientID(identifier string, authClientID int64) (*model.SignupFlow, error) {
	var signupFlow model.SignupFlow
	err := r.db.Where("identifier = ? AND auth_client_id = ?", identifier, authClientID).First(&signupFlow).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &signupFlow, nil
}

func (r *signupFlowRepository) FindByName(name string) (*model.SignupFlow, error) {
	var signupFlow model.SignupFlow
	err := r.db.Where("name = ?", name).First(&signupFlow).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &signupFlow, nil
}

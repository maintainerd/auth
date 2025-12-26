package repository

import (
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type LoginTemplateRepositoryGetFilter struct {
	Name      *string
	Status    []string
	Template  *string
	IsDefault *bool
	IsSystem  *bool
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type LoginTemplateRepository interface {
	BaseRepositoryMethods[model.LoginTemplate]
	FindByName(name string) (*model.LoginTemplate, error)
	FindPaginated(filter LoginTemplateRepositoryGetFilter) (*PaginationResult[model.LoginTemplate], error)
}

type loginTemplateRepository struct {
	*BaseRepository[model.LoginTemplate]
	db *gorm.DB
}

func NewLoginTemplateRepository(db *gorm.DB) LoginTemplateRepository {
	return &loginTemplateRepository{
		BaseRepository: NewBaseRepository[model.LoginTemplate](db, "login_template_uuid", "login_template_id"),
		db:             db,
	}
}

// FindByName retrieves an active login template by its name
func (r *loginTemplateRepository) FindByName(name string) (*model.LoginTemplate, error) {
	var template model.LoginTemplate
	err := r.db.
		Where("name = ? AND status = ?", name, "active").
		First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// FindPaginated retrieves paginated login templates with filtering
func (r *loginTemplateRepository) FindPaginated(filter LoginTemplateRepositoryGetFilter) (*PaginationResult[model.LoginTemplate], error) {
	query := r.db.Model(&model.LoginTemplate{})

	// Apply filters
	if filter.Name != nil && *filter.Name != "" {
		query = query.Where("name ILIKE ?", "%"+*filter.Name+"%")
	}
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if filter.Template != nil && *filter.Template != "" {
		query = query.Where("template = ?", *filter.Template)
	}
	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
	}
	if filter.IsSystem != nil {
		query = query.Where("is_system = ?", *filter.IsSystem)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply sorting
	sortBy := "created_at"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}
	sortOrder := "desc"
	if filter.SortOrder != "" {
		sortOrder = filter.SortOrder
	}
	query = query.Order(sortBy + " " + sortOrder)

	// Apply pagination
	page := 1
	if filter.Page > 0 {
		page = filter.Page
	}
	limit := 10
	if filter.Limit > 0 {
		limit = filter.Limit
	}
	offset := (page - 1) * limit
	query = query.Offset(offset).Limit(limit)

	// Execute query
	var templates []model.LoginTemplate
	if err := query.Find(&templates).Error; err != nil {
		return nil, err
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return &PaginationResult[model.LoginTemplate]{
		Data:       templates,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

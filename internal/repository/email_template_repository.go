package repository

import (
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type EmailTemplateRepositoryGetFilter struct {
	Name      *string
	Status    []string
	IsDefault *bool
	IsSystem  *bool
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type EmailTemplateRepository interface {
	BaseRepositoryMethods[model.EmailTemplate]
	FindByName(name string) (*model.EmailTemplate, error)
	FindPaginated(filter EmailTemplateRepositoryGetFilter) (*PaginationResult[model.EmailTemplate], error)
}

type emailTemplateRepository struct {
	*BaseRepository[model.EmailTemplate]
	db *gorm.DB
}

func NewEmailTemplateRepository(db *gorm.DB) EmailTemplateRepository {
	return &emailTemplateRepository{
		BaseRepository: NewBaseRepository[model.EmailTemplate](db, "email_template_uuid", "email_template_id"),
		db:             db,
	}
}

// FindByName retrieves an active email template by its name
func (r *emailTemplateRepository) FindByName(name string) (*model.EmailTemplate, error) {
	var template model.EmailTemplate
	err := r.db.
		Where("name = ? AND status = ?", name, "active").
		First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// FindPaginated retrieves paginated email templates with filtering
func (r *emailTemplateRepository) FindPaginated(filter EmailTemplateRepositoryGetFilter) (*PaginationResult[model.EmailTemplate], error) {
	query := r.db.Model(&model.EmailTemplate{})

	// Apply filters
	if filter.Name != nil && *filter.Name != "" {
		query = query.Where("name ILIKE ?", "%"+*filter.Name+"%")
	}
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
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
	var templates []model.EmailTemplate
	if err := query.Find(&templates).Error; err != nil {
		return nil, err
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return &PaginationResult[model.EmailTemplate]{
		Data:       templates,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

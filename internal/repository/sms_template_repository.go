package repository

import (
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type SmsTemplateRepositoryGetFilter struct {
	Name      *string
	Status    []string
	IsDefault *bool
	IsSystem  *bool
	Encoding  *string
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type SmsTemplateRepository interface {
	BaseRepositoryMethods[model.SmsTemplate]
	FindByName(name string) (*model.SmsTemplate, error)
	FindPaginated(filter SmsTemplateRepositoryGetFilter) (*PaginationResult[model.SmsTemplate], error)
}

type smsTemplateRepository struct {
	*BaseRepository[model.SmsTemplate]
	db *gorm.DB
}

func NewSmsTemplateRepository(db *gorm.DB) SmsTemplateRepository {
	return &smsTemplateRepository{
		BaseRepository: NewBaseRepository[model.SmsTemplate](db, "sms_template_uuid", "sms_template_id"),
		db:             db,
	}
}

// FindByName retrieves an active SMS template by its name
func (r *smsTemplateRepository) FindByName(name string) (*model.SmsTemplate, error) {
	var template model.SmsTemplate
	err := r.db.
		Where("name = ? AND status = ?", name, "active").
		First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// FindPaginated retrieves paginated SMS templates with filtering
func (r *smsTemplateRepository) FindPaginated(filter SmsTemplateRepositoryGetFilter) (*PaginationResult[model.SmsTemplate], error) {
	query := r.db.Model(&model.SmsTemplate{})

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
	var templates []model.SmsTemplate
	if err := query.Find(&templates).Error; err != nil {
		return nil, err
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return &PaginationResult[model.SmsTemplate]{
		Data:       templates,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

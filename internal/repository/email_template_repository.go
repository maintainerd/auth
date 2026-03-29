package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type EmailTemplateRepositoryGetFilter struct {
	Name      *string
	Status    []string
	TenantID  *int64
	IsDefault *bool
	IsSystem  *bool
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type EmailTemplateRepository interface {
	BaseRepositoryMethods[model.EmailTemplate]
	FindByUUIDAndTenantID(emailTemplateUUID uuid.UUID, tenantID int64, preloads ...string) (*model.EmailTemplate, error)
	FindByName(name string) (*model.EmailTemplate, error)
	FindPaginated(filter EmailTemplateRepositoryGetFilter) (*PaginationResult[model.EmailTemplate], error)
}

type emailTemplateRepository struct {
	*BaseRepository[model.EmailTemplate]
}

func NewEmailTemplateRepository(db *gorm.DB) EmailTemplateRepository {
	return &emailTemplateRepository{
		BaseRepository: NewBaseRepository[model.EmailTemplate](db, "email_template_uuid", "email_template_id"),
	}
}

// FindByUUIDAndTenantID retrieves an email template by UUID and tenant ID
func (r *emailTemplateRepository) FindByUUIDAndTenantID(emailTemplateUUID uuid.UUID, tenantID int64, preloads ...string) (*model.EmailTemplate, error) {
	var template model.EmailTemplate
	query := r.DB().Where("email_template_uuid = ? AND tenant_id = ?", emailTemplateUUID, tenantID)

	for _, preload := range preloads {
		query = query.Preload(preload)
	}

	err := query.First(&template).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &template, nil
}

// FindByName retrieves an active email template by its name
func (r *emailTemplateRepository) FindByName(name string) (*model.EmailTemplate, error) {
	var template model.EmailTemplate
	err := r.DB().
		Where("name = ? AND status = ?", name, "active").
		First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// FindPaginated retrieves paginated email templates with filtering
func (r *emailTemplateRepository) FindPaginated(filter EmailTemplateRepositoryGetFilter) (*PaginationResult[model.EmailTemplate], error) {
	query := r.DB().Model(&model.EmailTemplate{})

	// Apply filters
	if filter.Name != nil && *filter.Name != "" {
		query = query.Where("name ILIKE ?", "%"+*filter.Name+"%")
	}
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
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

	// Apply sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

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

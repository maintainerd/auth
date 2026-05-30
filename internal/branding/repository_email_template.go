package branding

import (
	"errors"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
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
	BaseRepositoryMethods[EmailTemplate]
	FindByUUIDAndTenantID(emailTemplateUUID uuid.UUID, tenantID int64, preloads ...string) (*EmailTemplate, error)
	FindByName(name string) (*EmailTemplate, error)
	FindPaginated(filter EmailTemplateRepositoryGetFilter) (*PaginationResult[EmailTemplate], error)
}

type emailTemplateRepository struct {
	*BaseRepository[EmailTemplate]
}

func NewEmailTemplateRepository(db *gorm.DB) EmailTemplateRepository {
	return &emailTemplateRepository{
		BaseRepository: NewBaseRepository[EmailTemplate](db, "email_template_uuid", "email_template_id"),
	}
}

// FindByUUIDAndTenantID retrieves an email template by UUID and tenant ID
func (r *emailTemplateRepository) FindByUUIDAndTenantID(emailTemplateUUID uuid.UUID, tenantID int64, preloads ...string) (*EmailTemplate, error) {
	var template EmailTemplate
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
func (r *emailTemplateRepository) FindByName(name string) (*EmailTemplate, error) {
	var template EmailTemplate
	err := r.DB().
		Where("name = ? AND status = ?", name, shared.StatusActive).
		First(&template).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &template, nil
}

// FindPaginated retrieves paginated email templates with filtering
func (r *emailTemplateRepository) FindPaginated(filter EmailTemplateRepositoryGetFilter) (*PaginationResult[EmailTemplate], error) {
	query := r.DB().Model(&EmailTemplate{})

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
	var templates []EmailTemplate
	if err := query.Find(&templates).Error; err != nil {
		return nil, err
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return &PaginationResult[EmailTemplate]{
		Data:       templates,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

package repository

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type LoginTemplateRepositoryGetFilter struct {
	Name      *string
	Status    []string
	Template  *string
	TenantID  *int64
	IsDefault *bool
	IsSystem  *bool
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type LoginTemplateRepository interface {
	BaseRepositoryMethods[model.LoginTemplate]
	FindByUUIDAndTenantID(loginTemplateUUID uuid.UUID, tenantID int64, preloads ...string) (*model.LoginTemplate, error)
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

// FindByUUIDAndTenantID retrieves a login template by UUID and tenant ID
func (r *loginTemplateRepository) FindByUUIDAndTenantID(loginTemplateUUID uuid.UUID, tenantID int64, preloads ...string) (*model.LoginTemplate, error) {
	var template model.LoginTemplate
	query := r.db.Where("login_template_uuid = ? AND tenant_id = ?", loginTemplateUUID, tenantID)

	for _, preload := range preloads {
		query = query.Preload(preload)
	}

	err := query.First(&template).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &template, nil
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

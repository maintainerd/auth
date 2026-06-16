package branding

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/database"
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
	FindByNameAndTenantID(name string, tenantID int64) (*EmailTemplate, error)
	FindPaginated(filter EmailTemplateRepositoryGetFilter) (*PaginationResult[EmailTemplate], error)
}

type emailTemplateRepository struct {
	*BaseRepository[EmailTemplate]
}

func NewEmailTemplateRepository(db *gorm.DB) EmailTemplateRepository {
	return &emailTemplateRepository{
		BaseRepository: database.NewBaseRepository[EmailTemplate](db, "email_template_uuid", "email_template_id"),
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

func (r *emailTemplateRepository) FindByNameAndTenantID(name string, tenantID int64) (*EmailTemplate, error) {
	var template EmailTemplate
	err := r.DB().
		Where("name = ? AND tenant_id = ? AND status = ?", name, tenantID, shared.StatusActive).
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
	query = database.ApplyILike(query, "name", filter.Name)
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

	// Apply sorting — protected against SQL injection via allowlist
	query = query.Order(database.SanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	return database.PaginateQuery[EmailTemplate](query, filter.Page, filter.Limit)
}

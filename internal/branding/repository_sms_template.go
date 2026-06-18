package branding

import (
	"errors"
	"fmt"

	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type SMSTemplateRepositoryGetFilter struct {
	TenantID  *int64
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

type SMSTemplateRepository interface {
	BaseRepositoryMethods[SMSTemplate]
	UpdateByUUID(uuid any, updatedData any) (*SMSTemplate, error)
	DeleteByUUID(uuid any) error
	FindByUUIDAndTenantID(uuid string, tenantID int64) (*SMSTemplate, error)
	FindPaginated(filter SMSTemplateRepositoryGetFilter) (*PaginationResult[SMSTemplate], error)
}

type smsTemplateRepository struct {
	*BaseRepository[SMSTemplate]
}

func NewSMSTemplateRepository(db *gorm.DB) SMSTemplateRepository {
	return &smsTemplateRepository{
		BaseRepository: database.NewBaseRepository[SMSTemplate](db, "sms_template_uuid", "sms_template_id"),
	}
}

// FindByUUIDAndTenantID retrieves an SMS template by UUID and tenant ID
func (r *smsTemplateRepository) FindByUUIDAndTenantID(uuid string, tenantID int64) (*SMSTemplate, error) {
	var template SMSTemplate
	err := r.DB().
		Where("sms_template_uuid = ? AND tenant_id = ?", uuid, tenantID).
		First(&template).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &template, nil
}

// FindPaginated retrieves paginated SMS templates with filtering
func (r *smsTemplateRepository) FindPaginated(filter SMSTemplateRepositoryGetFilter) (*PaginationResult[SMSTemplate], error) {
	if filter.TenantID == nil || *filter.TenantID == 0 {
		return nil, fmt.Errorf("tenant_id is required")
	}

	query := r.DB().Model(&SMSTemplate{})

	// Apply filters
	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
	}
	query = database.ApplyILike(query, "name", filter.Name)
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
	}
	if filter.IsSystem != nil {
		query = query.Where("is_system = ?", *filter.IsSystem)
	}

	// Apply sorting — protected against SQL injection via allowlist
	query = query.Order(database.SanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	return database.PaginateQuery[SMSTemplate](query, filter.Page, filter.Limit)
}

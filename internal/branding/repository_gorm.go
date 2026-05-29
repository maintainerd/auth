package branding

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"gorm.io/gorm"
)

// BrandingRepository defines persistence operations for the branding entity.
type BrandingRepository interface {
	BaseRepositoryMethods[Branding]
	WithTx(tx *gorm.DB) BrandingRepository
	FindByTenantID(tenantID int64) (*Branding, error)
}

type brandingRepository struct {
	*BaseRepository[Branding]
}

// NewBrandingRepository creates a new BrandingRepository backed by the given
// database connection.
func NewBrandingRepository(db *gorm.DB) BrandingRepository {
	return &brandingRepository{
		BaseRepository: NewBaseRepository[Branding](db, "branding_uuid", "branding_id"),
	}
}

// WithTx returns a copy of the repository bound to the supplied transaction.
func (r *brandingRepository) WithTx(tx *gorm.DB) BrandingRepository {
	return &brandingRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByTenantID retrieves the single branding record for a tenant. Returns
// nil, nil when no record exists.
func (r *brandingRepository) FindByTenantID(tenantID int64) (*Branding, error) {
	var branding Branding
	err := r.DB().Where("tenant_id = ?", tenantID).First(&branding).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &branding, nil
}

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
	BaseRepositoryMethods[LoginTemplate]
	FindByUUIDAndTenantID(loginTemplateUUID uuid.UUID, tenantID int64, preloads ...string) (*LoginTemplate, error)
	FindByName(name string) (*LoginTemplate, error)
	FindPaginated(filter LoginTemplateRepositoryGetFilter) (*PaginationResult[LoginTemplate], error)
}

type loginTemplateRepository struct {
	*BaseRepository[LoginTemplate]
}

func NewLoginTemplateRepository(db *gorm.DB) LoginTemplateRepository {
	return &loginTemplateRepository{
		BaseRepository: NewBaseRepository[LoginTemplate](db, "login_template_uuid", "login_template_id"),
	}
}

// FindByUUIDAndTenantID retrieves a login template by UUID and tenant ID
func (r *loginTemplateRepository) FindByUUIDAndTenantID(loginTemplateUUID uuid.UUID, tenantID int64, preloads ...string) (*LoginTemplate, error) {
	var template LoginTemplate
	query := r.DB().Where("login_template_uuid = ? AND tenant_id = ?", loginTemplateUUID, tenantID)

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

// FindByName retrieves an active login template by its name
func (r *loginTemplateRepository) FindByName(name string) (*LoginTemplate, error) {
	var template LoginTemplate
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

// FindPaginated retrieves paginated login templates with filtering
func (r *loginTemplateRepository) FindPaginated(filter LoginTemplateRepositoryGetFilter) (*PaginationResult[LoginTemplate], error) {
	query := r.DB().Model(&LoginTemplate{})

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
	var templates []LoginTemplate
	if err := query.Find(&templates).Error; err != nil {
		return nil, err
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return &PaginationResult[LoginTemplate]{
		Data:       templates,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

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
	FindByName(name string) (*SMSTemplate, error)
	FindByUUIDAndTenantID(uuid string, tenantID int64) (*SMSTemplate, error)
	FindPaginated(filter SMSTemplateRepositoryGetFilter) (*PaginationResult[SMSTemplate], error)
}

type smsTemplateRepository struct {
	*BaseRepository[SMSTemplate]
}

func NewSMSTemplateRepository(db *gorm.DB) SMSTemplateRepository {
	return &smsTemplateRepository{
		BaseRepository: NewBaseRepository[SMSTemplate](db, "sms_template_uuid", "sms_template_id"),
	}
}

// FindByName retrieves an active SMS template by its name
func (r *smsTemplateRepository) FindByName(name string) (*SMSTemplate, error) {
	var template SMSTemplate
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
	query := r.DB().Model(&SMSTemplate{})

	// Apply filters
	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
	}
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
	var templates []SMSTemplate
	if err := query.Find(&templates).Error; err != nil {
		return nil, err
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return &PaginationResult[SMSTemplate]{
		Data:       templates,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

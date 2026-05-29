package tenant

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TenantRepositoryGetFilter struct {
	Name        *string
	DisplayName *string
	Description *string
	Identifier  *string
	Status      []string
	IsPublic    *bool
	IsSystem    *bool
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type TenantRepository interface {
	BaseRepositoryMethods[Tenant]
	WithTx(tx *gorm.DB) TenantRepository
	FindByName(name string) (*Tenant, error)
	FindByIdentifier(identifier string) (*Tenant, error)
	FindSystem() (*Tenant, error)
	FindPaginated(filter TenantRepositoryGetFilter) (*PaginationResult[Tenant], error)
	SetStatusByUUID(tenantUUID uuid.UUID, status string) error
	SetSystemStatusByUUID(tenantUUID uuid.UUID, isSystem bool) error
}

type tenantRepository struct {
	*BaseRepository[Tenant]
}

func NewTenantRepository(db *gorm.DB) TenantRepository {
	return &tenantRepository{
		BaseRepository: NewBaseRepository[Tenant](db, "tenant_uuid", "tenant_id"),
	}
}

func (r *tenantRepository) WithTx(tx *gorm.DB) TenantRepository {
	return &tenantRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *tenantRepository) FindByName(name string) (*Tenant, error) {
	var tenant Tenant
	err := r.DB().Where("name = ?", name).First(&tenant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tenant, nil
}

func (r *tenantRepository) FindByIdentifier(identifier string) (*Tenant, error) {
	var tenant Tenant
	err := r.DB().Where("identifier = ?", identifier).First(&tenant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tenant, nil
}

// FindSystem returns the unique system tenant (is_system = true).
// There is always exactly one system tenant; it cannot be deleted.
func (r *tenantRepository) FindSystem() (*Tenant, error) {
	var tenant Tenant
	err := r.DB().Where("is_system = ?", true).First(&tenant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tenant, nil
}

func (r *tenantRepository) FindPaginated(filter TenantRepositoryGetFilter) (*PaginationResult[Tenant], error) {
	query := r.DB().Model(&Tenant{})

	// Filters with LIKE
	if filter.Name != nil {
		query = query.Where("name ILIKE ?", "%"+*filter.Name+"%")
	}
	if filter.DisplayName != nil {
		query = query.Where("display_name ILIKE ?", "%"+*filter.DisplayName+"%")
	}
	if filter.Description != nil {
		query = query.Where("description ILIKE ?", "%"+*filter.Description+"%")
	}
	if filter.Identifier != nil {
		query = query.Where("identifier ILIKE ?", "%"+*filter.Identifier+"%")
	}

	// Filters with exact match
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if filter.IsPublic != nil {
		query = query.Where("is_public = ?", *filter.IsPublic)
	}
	if filter.IsSystem != nil {
		query = query.Where("is_system = ?", *filter.IsSystem)
	}

	// Sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	// Count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Pagination guards prevent division-by-zero and negative offsets
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit
	var tenants []Tenant
	if err := query.Limit(filter.Limit).Offset(offset).Find(&tenants).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[Tenant]{
		Data:       tenants,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *tenantRepository) SetStatusByUUID(tenantUUID uuid.UUID, status string) error {
	return r.DB().Model(&Tenant{}).Where("tenant_uuid = ?", tenantUUID).Update("status", status).Error
}

func (r *tenantRepository) SetSystemStatusByUUID(tenantUUID uuid.UUID, isSystem bool) error {
	return r.DB().Model(&Tenant{}).Where("tenant_uuid = ?", tenantUUID).Update("is_system", isSystem).Error
}

type TenantMemberRepository interface {
	BaseRepositoryMethods[TenantMember]
	WithTx(tx *gorm.DB) TenantMemberRepository
	FindByTenantMemberUUID(uuid uuid.UUID) (*TenantMember, error)
	FindByTenantAndUser(tenantID int64, userID int64) (*TenantMember, error)
	FindAllByTenant(tenantID int64) ([]TenantMember, error)
	FindAllByUser(userID int64) ([]TenantMember, error)
}

type tenantMemberRepository struct {
	*BaseRepository[TenantMember]
}

func NewTenantMemberRepository(db *gorm.DB) TenantMemberRepository {
	return &tenantMemberRepository{
		BaseRepository: NewBaseRepository[TenantMember](db, "tenant_member_uuid", "tenant_member_id"),
	}
}

func (r *tenantMemberRepository) WithTx(tx *gorm.DB) TenantMemberRepository {
	return &tenantMemberRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *tenantMemberRepository) FindByTenantMemberUUID(uuid uuid.UUID) (*TenantMember, error) {
	var tu TenantMember
	err := r.DB().Where("tenant_member_uuid = ?", uuid).First(&tu).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tu, nil
}

func (r *tenantMemberRepository) FindByTenantAndUser(tenantID int64, userID int64) (*TenantMember, error) {
	var tu TenantMember
	err := r.DB().Where("tenant_id = ? AND user_id = ?", tenantID, userID).First(&tu).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tu, nil
}

func (r *tenantMemberRepository) FindAllByTenant(tenantID int64) ([]TenantMember, error) {
	var tus []TenantMember
	err := r.DB().Where("tenant_id = ?", tenantID).Find(&tus).Error
	if err != nil {
		return nil, err
	}
	return tus, nil
}

func (r *tenantMemberRepository) FindAllByUser(userID int64) ([]TenantMember, error) {
	var tus []TenantMember
	err := r.DB().Where("user_id = ?", userID).Find(&tus).Error
	if err != nil {
		return nil, err
	}
	return tus, nil
}

type TenantServiceRepositoryGetFilter struct {
	TenantID  *int64
	ServiceID *int64
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type TenantServiceRepository interface {
	BaseRepositoryMethods[TenantService]
	WithTx(tx *gorm.DB) TenantServiceRepository
	FindPaginated(filter TenantServiceRepositoryGetFilter) (*PaginationResult[TenantService], error)
	FindByTenantAndService(tenantID int64, serviceID int64) (*TenantService, error)
	DeleteByTenantAndService(tenantID int64, serviceID int64) error
}

type tenantServiceRepository struct {
	*BaseRepository[TenantService]
}

func NewTenantServiceRepository(db *gorm.DB) TenantServiceRepository {
	return &tenantServiceRepository{
		BaseRepository: NewBaseRepository[TenantService](db, "", "tenant_service_id"),
	}
}

func (r *tenantServiceRepository) WithTx(tx *gorm.DB) TenantServiceRepository {
	return &tenantServiceRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *tenantServiceRepository) FindByTenantAndService(tenantID int64, serviceID int64) (*TenantService, error) {
	var tenantService TenantService
	err := r.DB().Where("tenant_id = ? AND service_id = ?", tenantID, serviceID).First(&tenantService).Error
	if err != nil {
		return nil, err
	}
	return &tenantService, nil
}

func (r *tenantServiceRepository) DeleteByTenantAndService(tenantID int64, serviceID int64) error {
	return r.DB().Where("tenant_id = ? AND service_id = ?", tenantID, serviceID).Delete(&TenantServiceLink{}).Error
}

func (r *tenantServiceRepository) FindPaginated(filter TenantServiceRepositoryGetFilter) (*PaginationResult[TenantService], error) {
	query := r.DB().Model(&TenantServiceLink{})

	// Filters
	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
	}
	if filter.ServiceID != nil {
		query = query.Where("service_id = ?", *filter.ServiceID)
	}

	// Sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "tenant_service_id DESC"))

	// Pagination
	filter.Page, filter.Limit = normalizePagination(filter.Page, filter.Limit)
	offset := (filter.Page - 1) * filter.Limit
	if filter.Limit > 0 {
		query = query.Offset(offset).Limit(filter.Limit)
	}

	var tenantServices []TenantService
	var total int64

	// Count total records
	countQuery := r.DB().Model(&TenantServiceLink{})
	if filter.TenantID != nil {
		countQuery = countQuery.Where("tenant_id = ?", *filter.TenantID)
	}
	if filter.ServiceID != nil {
		countQuery = countQuery.Where("service_id = ?", *filter.ServiceID)
	}
	countQuery.Count(&total)

	// Execute query with preloads
	err := query.Preload("Tenant").Preload("Service").Find(&tenantServices).Error
	if err != nil {
		return nil, err
	}

	return &PaginationResult[TenantService]{
		Data:  tenantServices,
		Total: total,
		Page:  filter.Page,
		Limit: filter.Limit,
	}, nil
}

// TenantSettingRepository defines persistence operations for the
// tenant_settings entity.
type TenantSettingRepository interface {
	BaseRepositoryMethods[TenantSetting]
	WithTx(tx *gorm.DB) TenantSettingRepository
	FindByTenantID(tenantID int64) (*TenantSetting, error)
}

type tenantSettingRepository struct {
	*BaseRepository[TenantSetting]
}

// NewTenantSettingRepository creates a new TenantSettingRepository backed by
// the given database connection.
func NewTenantSettingRepository(db *gorm.DB) TenantSettingRepository {
	return &tenantSettingRepository{
		BaseRepository: NewBaseRepository[TenantSetting](db, "tenant_setting_uuid", "tenant_setting_id"),
	}
}

// WithTx returns a copy of the repository bound to the supplied transaction.
func (r *tenantSettingRepository) WithTx(tx *gorm.DB) TenantSettingRepository {
	return &tenantSettingRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByTenantID retrieves the single tenant_settings record for a tenant.
// Returns nil, nil when no record exists.
func (r *tenantSettingRepository) FindByTenantID(tenantID int64) (*TenantSetting, error) {
	var setting TenantSetting
	err := r.DB().Where("tenant_id = ?", tenantID).First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &setting, nil
}

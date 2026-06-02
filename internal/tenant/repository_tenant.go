package tenant

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/database"
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
	DeleteCascade(ctx context.Context, tx *gorm.DB, tenantID int64, cascadeModels []any) error
}

type tenantRepository struct {
	*BaseRepository[Tenant]
}

func NewTenantRepository(db *gorm.DB) TenantRepository {
	return &tenantRepository{
		BaseRepository: database.NewBaseRepository[Tenant](db, "tenant_uuid", "tenant_id"),
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

	query = database.ApplyILike(query, "name", filter.Name)
	query = database.ApplyILike(query, "display_name", filter.DisplayName)
	query = database.ApplyILike(query, "description", filter.Description)
	query = database.ApplyILike(query, "identifier", filter.Identifier)
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if filter.IsPublic != nil {
		query = query.Where("is_public = ?", *filter.IsPublic)
	}
	if filter.IsSystem != nil {
		query = query.Where("is_system = ?", *filter.IsSystem)
	}

	query = query.Order(database.SanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	return database.PaginateQuery[Tenant](query, filter.Page, filter.Limit)
}

func (r *tenantRepository) SetStatusByUUID(tenantUUID uuid.UUID, status string) error {
	return r.DB().Model(&Tenant{}).Where("tenant_uuid = ?", tenantUUID).Update("status", status).Error
}

func (r *tenantRepository) SetSystemStatusByUUID(tenantUUID uuid.UUID, isSystem bool) error {
	return r.DB().Model(&Tenant{}).Where("tenant_uuid = ?", tenantUUID).Update("is_system", isSystem).Error
}

func (r *tenantRepository) DeleteCascade(ctx context.Context, tx *gorm.DB, tenantID int64, cascadeModels []any) error {
	for _, m := range cascadeModels {
		if err := tx.WithContext(ctx).Where("tenant_id = ?", tenantID).Delete(m).Error; err != nil {
			return err
		}
	}
	return nil
}

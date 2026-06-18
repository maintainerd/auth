package tenant

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type TenantMemberRepository interface {
	BaseRepositoryMethods[TenantMember]
	DeleteByUUID(uuid any) error
	WithTx(tx *gorm.DB) TenantMemberRepository
	FindByTenantMemberUUID(uuid uuid.UUID) (*TenantMember, error)
	FindByTenantAndUser(tenantID int64, userID int64) (*TenantMember, error)
	FindByTenant(filter TenantMemberRepositoryListFilter) (*PaginationResult[TenantMember], error)
	FindAllByUser(userID int64) ([]TenantMember, error)
}

type TenantMemberRepositoryListFilter struct {
	TenantID  int64
	Role      *string
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type tenantMemberRepository struct {
	*BaseRepository[TenantMember]
}

func NewTenantMemberRepository(db *gorm.DB) TenantMemberRepository {
	return &tenantMemberRepository{
		BaseRepository: database.NewBaseRepository[TenantMember](db, "tenant_member_uuid", "tenant_member_id"),
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

func (r *tenantMemberRepository) FindByTenant(filter TenantMemberRepositoryListFilter) (*PaginationResult[TenantMember], error) {
	query := r.DB().Model(&TenantMember{}).Where("tenant_id = ?", filter.TenantID)
	if filter.Role != nil {
		query = query.Where("role = ?", *filter.Role)
	}

	query = query.Order(database.SanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	return database.PaginateQuery[TenantMember](query, filter.Page, filter.Limit)
}

func (r *tenantMemberRepository) FindAllByUser(userID int64) ([]TenantMember, error) {
	var tus []TenantMember
	err := r.DB().Where("user_id = ?", userID).Find(&tus).Error
	if err != nil {
		return nil, err
	}
	return tus, nil
}

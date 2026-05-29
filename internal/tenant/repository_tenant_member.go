package tenant

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

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

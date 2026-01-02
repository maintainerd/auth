package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type TenantMemberRepository interface {
	BaseRepositoryMethods[model.TenantMember]
	WithTx(tx *gorm.DB) TenantMemberRepository
	FindByTenantMemberUUID(uuid uuid.UUID) (*model.TenantMember, error)
	FindByTenantAndUser(tenantID int64, userID int64) (*model.TenantMember, error)
	FindAllByTenant(tenantID int64) ([]model.TenantMember, error)
	FindAllByUser(userID int64) ([]model.TenantMember, error)
}

type tenantMemberRepository struct {
	*BaseRepository[model.TenantMember]
	db *gorm.DB
}

func NewTenantMemberRepository(db *gorm.DB) TenantMemberRepository {
	return &tenantMemberRepository{
		BaseRepository: NewBaseRepository[model.TenantMember](db, "tenant_member_uuid", "tenant_member_id"),
		db:             db,
	}
}

func (r *tenantMemberRepository) WithTx(tx *gorm.DB) TenantMemberRepository {
	return &tenantMemberRepository{
		BaseRepository: NewBaseRepository[model.TenantMember](tx, "tenant_member_uuid", "tenant_member_id"),
		db:             tx,
	}
}

func (r *tenantMemberRepository) FindByTenantMemberUUID(uuid uuid.UUID) (*model.TenantMember, error) {
	var tu model.TenantMember
	err := r.db.Where("tenant_member_uuid = ?", uuid).First(&tu).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tu, nil
}

func (r *tenantMemberRepository) FindByTenantAndUser(tenantID int64, userID int64) (*model.TenantMember, error) {
	var tu model.TenantMember
	err := r.db.Where("tenant_id = ? AND user_id = ?", tenantID, userID).First(&tu).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tu, nil
}

func (r *tenantMemberRepository) FindAllByTenant(tenantID int64) ([]model.TenantMember, error) {
	var tus []model.TenantMember
	err := r.db.Where("tenant_id = ?", tenantID).Find(&tus).Error
	if err != nil {
		return nil, err
	}
	return tus, nil
}

func (r *tenantMemberRepository) FindAllByUser(userID int64) ([]model.TenantMember, error) {
	var tus []model.TenantMember
	err := r.db.Where("user_id = ?", userID).Find(&tus).Error
	if err != nil {
		return nil, err
	}
	return tus, nil
}

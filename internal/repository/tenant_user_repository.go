package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type TenantUserRepository interface {
	BaseRepositoryMethods[model.TenantUser]
	WithTx(tx *gorm.DB) TenantUserRepository
	FindByTenantUserUUID(uuid uuid.UUID) (*model.TenantUser, error)
	FindByTenantAndUser(tenantID int64, userID int64) (*model.TenantUser, error)
	FindAllByTenant(tenantID int64) ([]model.TenantUser, error)
	FindAllByUser(userID int64) ([]model.TenantUser, error)
}

type tenantUserRepository struct {
	*BaseRepository[model.TenantUser]
	db *gorm.DB
}

func NewTenantUserRepository(db *gorm.DB) TenantUserRepository {
	return &tenantUserRepository{
		BaseRepository: NewBaseRepository[model.TenantUser](db, "tenant_user_uuid", "tenant_user_id"),
		db:             db,
	}
}

func (r *tenantUserRepository) WithTx(tx *gorm.DB) TenantUserRepository {
	return &tenantUserRepository{
		BaseRepository: NewBaseRepository[model.TenantUser](tx, "tenant_user_uuid", "tenant_user_id"),
		db:             tx,
	}
}

func (r *tenantUserRepository) FindByTenantUserUUID(uuid uuid.UUID) (*model.TenantUser, error) {
	var tu model.TenantUser
	err := r.db.Where("tenant_user_uuid = ?", uuid).First(&tu).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tu, nil
}

func (r *tenantUserRepository) FindByTenantAndUser(tenantID int64, userID int64) (*model.TenantUser, error) {
	var tu model.TenantUser
	err := r.db.Where("tenant_id = ? AND user_id = ?", tenantID, userID).First(&tu).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tu, nil
}

func (r *tenantUserRepository) FindAllByTenant(tenantID int64) ([]model.TenantUser, error) {
	var tenantUsers []model.TenantUser
	err := r.db.Where("tenant_id = ?", tenantID).Find(&tenantUsers).Error
	if err != nil {
		return nil, err
	}
	return tenantUsers, nil
}

func (r *tenantUserRepository) FindAllByUser(userID int64) ([]model.TenantUser, error) {
	var tenantUsers []model.TenantUser
	err := r.db.Where("user_id = ?", userID).Find(&tenantUsers).Error
	if err != nil {
		return nil, err
	}
	return tenantUsers, nil
}

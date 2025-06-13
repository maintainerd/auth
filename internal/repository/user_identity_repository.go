package repository

import (
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type UserIdentityRepository interface {
	BaseRepositoryMethods[model.UserIdentity]
	FindByUserID(userID int64) ([]model.UserIdentity, error)
	FindByProviderAndUserID(providerName string, providerUserID string) (*model.UserIdentity, error)
	FindByEmail(email string) ([]model.UserIdentity, error)
	DeleteByUserID(userID int64) error
}

type userIdentityRepository struct {
	*BaseRepository[model.UserIdentity]
	db *gorm.DB
}

func NewUserIdentityRepository(db *gorm.DB) UserIdentityRepository {
	return &userIdentityRepository{
		BaseRepository: NewBaseRepository[model.UserIdentity](db, "user_identity_uuid", "user_identity_id"),
		db:             db,
	}
}

func (r *userIdentityRepository) FindByUserID(userID int64) ([]model.UserIdentity, error) {
	var identities []model.UserIdentity
	err := r.db.Where("user_id = ?", userID).Find(&identities).Error
	return identities, err
}

func (r *userIdentityRepository) FindByProviderAndUserID(providerName string, providerUserID string) (*model.UserIdentity, error) {
	var ui model.UserIdentity
	err := r.db.
		Where("provider_name = ? AND provider_user_id = ?", providerName, providerUserID).
		First(&ui).Error
	return &ui, err
}

func (r *userIdentityRepository) FindByEmail(email string) ([]model.UserIdentity, error) {
	var identities []model.UserIdentity
	err := r.db.Where("email = ?", email).Find(&identities).Error
	return identities, err
}

func (r *userIdentityRepository) DeleteByUserID(userID int64) error {
	return r.db.Where("user_id = ?", userID).Delete(&model.UserIdentity{}).Error
}

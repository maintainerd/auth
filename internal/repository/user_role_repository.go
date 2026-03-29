package repository

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type UserRoleRepository interface {
	BaseRepositoryMethods[model.UserRole]
	WithTx(tx *gorm.DB) UserRoleRepository
	FindByUserID(userID int64) ([]model.UserRole, error)
	FindByUserIDAndRoleID(userID int64, roleID int64) (*model.UserRole, error)
	FindDefaultRolesByUserID(userID int64) ([]model.UserRole, error)
	DeleteByUserID(userID int64) error
	DeleteByUserIDAndRoleID(userID int64, roleID int64) error
}

type userRoleRepository struct {
	*BaseRepository[model.UserRole]
}

func NewUserRoleRepository(db *gorm.DB) UserRoleRepository {
	return &userRoleRepository{
		BaseRepository: NewBaseRepository[model.UserRole](db, "user_role_uuid", "user_role_id"),
	}
}

func (r *userRoleRepository) WithTx(tx *gorm.DB) UserRoleRepository {
	return &userRoleRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *userRoleRepository) FindByUserID(userID int64) ([]model.UserRole, error) {
	var userRoles []model.UserRole
	err := r.DB().Where("user_id = ?", userID).Find(&userRoles).Error
	return userRoles, err
}

func (r *userRoleRepository) FindByUserIDAndRoleID(userID int64, roleID int64) (*model.UserRole, error) {
	var ur model.UserRole
	err := r.DB().
		Where("user_id = ? AND role_id = ?", userID, roleID).
		First(&ur).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ur, nil
}

func (r *userRoleRepository) FindDefaultRolesByUserID(userID int64) ([]model.UserRole, error) {
	var userRoles []model.UserRole
	err := r.DB().
		Where("user_id = ? AND is_default = true", userID).
		Find(&userRoles).Error
	return userRoles, err
}

func (r *userRoleRepository) DeleteByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&model.UserRole{}).Error
}

func (r *userRoleRepository) DeleteByUserIDAndRoleID(userID int64, roleID int64) error {
	return r.DB().
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&model.UserRole{}).Error
}

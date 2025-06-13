package repository

import (
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type UserRoleRepository interface {
	BaseRepositoryMethods[model.UserRole]
	FindByUserID(userID int64) ([]model.UserRole, error)
	FindByUserIDAndRoleID(userID int64, roleID int64) (*model.UserRole, error)
	FindDefaultRolesByUserID(userID int64) ([]model.UserRole, error)
	DeleteByUserID(userID int64) error
	DeleteByUserIDAndRoleID(userID int64, roleID int64) error
}

type userRoleRepository struct {
	*BaseRepository[model.UserRole]
	db *gorm.DB
}

func NewUserRoleRepository(db *gorm.DB) UserRoleRepository {
	return &userRoleRepository{
		BaseRepository: NewBaseRepository[model.UserRole](db, "user_role_uuid", "user_role_id"),
		db:             db,
	}
}

func (r *userRoleRepository) FindByUserID(userID int64) ([]model.UserRole, error) {
	var userRoles []model.UserRole
	err := r.db.Where("user_id = ?", userID).Find(&userRoles).Error
	return userRoles, err
}

func (r *userRoleRepository) FindByUserIDAndRoleID(userID int64, roleID int64) (*model.UserRole, error) {
	var ur model.UserRole
	err := r.db.
		Where("user_id = ? AND role_id = ?", userID, roleID).
		First(&ur).Error
	return &ur, err
}

func (r *userRoleRepository) FindDefaultRolesByUserID(userID int64) ([]model.UserRole, error) {
	var userRoles []model.UserRole
	err := r.db.
		Where("user_id = ? AND is_default = true", userID).
		Find(&userRoles).Error
	return userRoles, err
}

func (r *userRoleRepository) DeleteByUserID(userID int64) error {
	return r.db.Where("user_id = ?", userID).Delete(&model.UserRole{}).Error
}

func (r *userRoleRepository) DeleteByUserIDAndRoleID(userID int64, roleID int64) error {
	return r.db.
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&model.UserRole{}).Error
}

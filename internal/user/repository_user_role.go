package user

import (
	"errors"

	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

type UserRoleRepository interface {
	BaseRepositoryMethods[UserRole]
	FindAll(preloads ...string) ([]UserRole, error)
	FindByUUID(uuid any, preloads ...string) (*UserRole, error)
	FindByUUIDs(uuids []string, preloads ...string) ([]UserRole, error)
	FindByID(id any, preloads ...string) (*UserRole, error)
	UpdateByUUID(uuid any, updatedData any) (*UserRole, error)
	UpdateByID(id any, updatedData any) (*UserRole, error)
	DeleteByUUID(uuid any) error
	DeleteByID(id any) error
	Paginate(conditions map[string]any, page int, limit int, preloads ...string) (*PaginationResult[UserRole], error)
	WithTx(tx *gorm.DB) UserRoleRepository
	FindByUserID(userID int64) ([]UserRole, error)
	FindByUserIDAndRoleID(userID int64, roleID int64) (*UserRole, error)
	FindDefaultRolesByUserID(userID int64) ([]UserRole, error)
	DeleteByUserID(userID int64) error
	DeleteByUserIDAndRoleID(userID int64, roleID int64) error
}

type userRoleRepository struct {
	*BaseRepository[UserRole]
}

func NewUserRoleRepository(db *gorm.DB) UserRoleRepository {
	return &userRoleRepository{
		BaseRepository: database.NewBaseRepository[UserRole](db, "user_role_uuid", "user_role_id"),
	}
}

func (r *userRoleRepository) WithTx(tx *gorm.DB) UserRoleRepository {
	return &userRoleRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *userRoleRepository) FindByUserID(userID int64) ([]UserRole, error) {
	var userRoles []UserRole
	err := r.DB().Where("user_id = ?", userID).Find(&userRoles).Error
	return userRoles, err
}

func (r *userRoleRepository) FindByUserIDAndRoleID(userID int64, roleID int64) (*UserRole, error) {
	var ur UserRole
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

func (r *userRoleRepository) FindDefaultRolesByUserID(userID int64) ([]UserRole, error) {
	var userRoles []UserRole
	err := r.DB().
		Where("user_id = ? AND is_default = true", userID).
		Find(&userRoles).Error
	return userRoles, err
}

func (r *userRoleRepository) DeleteByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&UserRole{}).Error
}

func (r *userRoleRepository) DeleteByUserIDAndRoleID(userID int64, roleID int64) error {
	return r.DB().
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&UserRole{}).Error
}

package repository

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type UserRepository interface {
	BaseRepositoryMethods[model.User]
	FindByUsername(username string, authContainerID int64) (*model.User, error)
	FindByEmail(email string, authContainerID int64) (*model.User, error)
	FindByUsernameOrEmail(identifier string, authContainerID int64) (*model.User, error)
	SetEmailVerified(userUUID uuid.UUID, verified bool) error
	SetActiveStatus(userUUID uuid.UUID, active bool) error
}

type userRepository struct {
	*BaseRepository[model.User]
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		BaseRepository: NewBaseRepository[model.User](db, "user_uuid", "user_id"),
		db:             db,
	}
}

func (r *userRepository) FindByUsername(username string, authContainerID int64) (*model.User, error) {
	var user model.User
	err := r.db.
		Where("username = ? AND auth_container_id = ?", username, authContainerID).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) FindByEmail(email string, authContainerID int64) (*model.User, error) {
	var user model.User
	err := r.db.
		Where("email = ? AND auth_container_id = ?", email, authContainerID).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) FindByUsernameOrEmail(identifier string, authContainerID int64) (*model.User, error) {
	var user model.User
	err := r.db.
		Where("(username = ? OR email = ?) AND auth_container_id = ?", identifier, identifier, authContainerID).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) SetEmailVerified(userUUID uuid.UUID, verified bool) error {
	return r.db.Model(&model.User{}).
		Where("user_uuid = ?", userUUID).
		Update("is_email_verified", verified).Error
}

func (r *userRepository) SetActiveStatus(userUUID uuid.UUID, active bool) error {
	return r.db.Model(&model.User{}).
		Where("user_uuid = ?", userUUID).
		Update("is_active", active).Error
}

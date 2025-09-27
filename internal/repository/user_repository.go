package repository

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type UserRepositoryGetFilter struct {
	Username        *string
	Email           *string
	Phone           *string
	IsActive        *bool
	AuthContainerID *int64
	Page            int
	Limit           int
	SortBy          string
	SortOrder       string
}

type UserRepository interface {
	BaseRepositoryMethods[model.User]
	WithTx(tx *gorm.DB) UserRepository
	FindByUsername(username string, authContainerID int64) (*model.User, error)
	FindByEmail(email string, authContainerID int64) (*model.User, error)
	FindSuperAdmin() (*model.User, error)
	FindRoles(userID int64) ([]model.Role, error)
	FindBySubAndClientID(sub string, authClientID string) (*model.User, error)
	FindPaginated(filter UserRepositoryGetFilter) (*PaginationResult[model.User], error)
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

func (r *userRepository) WithTx(tx *gorm.DB) UserRepository {
	return &userRepository{
		BaseRepository: NewBaseRepository[model.User](tx, "user_uuid", "user_id"),
		db:             tx,
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

func (r *userRepository) FindSuperAdmin() (*model.User, error) {
	var user model.User
	err := r.db.
		Joins("JOIN auth_containers ON users.auth_container_id = auth_containers.auth_container_id").
		Joins("JOIN user_roles ON users.user_id = user_roles.user_id").
		Joins("JOIN roles ON user_roles.role_id = roles.role_id").
		Where("auth_containers.is_active = true AND auth_containers.is_default = true").
		Where("roles.name = ?", "super-admin").
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) FindRoles(userID int64) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.
		Model(&model.Role{}).
		Select("roles.*").
		Joins("JOIN user_roles ur ON ur.role_id = roles.role_id").
		Where("ur.user_id = ?", userID).
		Find(&roles).Error
	return roles, err
}

func (r *userRepository) FindBySubAndClientID(sub string, authClientID string) (*model.User, error) {
	var user model.User
	err := r.db.
		Preload("AuthContainer").
		Preload("AuthContainer.Organization").
		Preload("UserIdentities.AuthClient").
		Preload("Roles.Permissions").
		Joins("JOIN user_identities ON users.user_id = user_identities.user_id").
		Joins("JOIN auth_clients ON user_identities.auth_client_id = auth_clients.auth_client_id").
		Where("user_identities.sub = ? AND auth_clients.client_id = ?", sub, authClientID).
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

func (r *userRepository) FindPaginated(filter UserRepositoryGetFilter) (*PaginationResult[model.User], error) {
	var users []model.User
	var total int64

	query := r.db.Model(&model.User{})

	// Apply filters
	if filter.Username != nil {
		query = query.Where("username ILIKE ?", "%"+*filter.Username+"%")
	}
	if filter.Email != nil {
		query = query.Where("email ILIKE ?", "%"+*filter.Email+"%")
	}
	if filter.Phone != nil {
		query = query.Where("phone ILIKE ?", "%"+*filter.Phone+"%")
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.AuthContainerID != nil {
		query = query.Where("auth_container_id = ?", *filter.AuthContainerID)
	}

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply sorting
	if filter.SortBy != "" {
		order := filter.SortBy
		if filter.SortOrder == "desc" {
			order += " DESC"
		} else {
			order += " ASC"
		}
		query = query.Order(order)
	} else {
		query = query.Order("created_at DESC")
	}

	// Apply pagination
	offset := (filter.Page - 1) * filter.Limit
	if err := query.Offset(offset).Limit(filter.Limit).Find(&users).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[model.User]{
		Data:       users,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

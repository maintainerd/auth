package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"gorm.io/gorm"
)

type RoleServiceDataResult struct {
	RoleUUID    uuid.UUID
	Name        string
	Description string
	IsDefault   bool
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type RoleServiceGetFilter struct {
	Name            *string
	Description     *string
	IsDefault       *bool
	IsActive        *bool
	AuthContainerID int64
	Page            int
	Limit           int
	SortBy          string
	SortOrder       string
}

type RoleServiceGetResult struct {
	Data       []RoleServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type RoleService interface {
	Get(filter RoleServiceGetFilter) (*RoleServiceGetResult, error)
	GetByUUID(roleUUID uuid.UUID) (*RoleServiceDataResult, error)
	Create(name string, description string, IsDefault bool, IsActive bool, authContainerId int64) (*RoleServiceDataResult, error)
	Update(roleUUID uuid.UUID, name string, description string, IsDefault bool, IsActive bool, authContainerId int64) (*RoleServiceDataResult, error)
	DeleteByUUID(roleUUID uuid.UUID) (*RoleServiceDataResult, error)
}

type roleService struct {
	db       *gorm.DB
	roleRepo repository.RoleRepository
}

func NewRoleService(
	db *gorm.DB,
	roleRepo repository.RoleRepository,
) RoleService {
	return &roleService{
		db:       db,
		roleRepo: roleRepo,
	}
}

func (s *roleService) Get(filter RoleServiceGetFilter) (*RoleServiceGetResult, error) {
	roleFilter := repository.RoleRepositoryGetFilter{
		Name:            filter.Name,
		Description:     filter.Description,
		IsDefault:       filter.IsDefault,
		IsActive:        filter.IsActive,
		AuthContainerID: filter.AuthContainerID,
		Page:            filter.Page,
		Limit:           filter.Limit,
		SortBy:          filter.SortBy,
		SortOrder:       filter.SortOrder,
	}

	// Query paginated roles
	result, err := s.roleRepo.FindPaginated(roleFilter)
	if err != nil {
		return nil, err
	}

	roles := make([]RoleServiceDataResult, len(result.Data))
	for i, r := range result.Data {
		roles[i] = RoleServiceDataResult{
			RoleUUID:    r.RoleUUID,
			Name:        r.Name,
			Description: r.Description,
			IsDefault:   r.IsDefault,
			IsActive:    r.IsActive,
			CreatedAt:   r.CreatedAt,
			UpdatedAt:   r.UpdatedAt,
		}
	}

	return &RoleServiceGetResult{
		Data:       roles,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *roleService) GetByUUID(roleUUID uuid.UUID) (*RoleServiceDataResult, error) {
	role, err := s.roleRepo.FindByUUID(roleUUID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.New("role not found")
	}

	return &RoleServiceDataResult{
		RoleUUID:    role.RoleUUID,
		Name:        role.Name,
		Description: role.Description,
		IsDefault:   role.IsDefault,
		IsActive:    role.IsActive,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}, nil
}

func (s *roleService) Create(name string, description string, isDefault bool, isActive bool, authContainerId int64) (*RoleServiceDataResult, error) {
	var createdRole *model.Role

	// Transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var existingRole *model.Role
		var err error

		// Check if role already exist
		existingRole, err = s.roleRepo.FindByNameAndAuthContainerID(name, authContainerId)
		if err != nil {
			return err
		}
		if existingRole != nil {
			return errors.New(name + " role already exist")
		}

		// Create role
		newRole := &model.Role{
			Name:            name,
			Description:     description,
			IsDefault:       isDefault,
			IsActive:        isActive,
			AuthContainerID: authContainerId,
		}

		if err := tx.Create(newRole).Error; err != nil {
			return err
		}

		createdRole = newRole

		return nil // commit transaction
	})

	if err != nil {
		return nil, err
	}

	return &RoleServiceDataResult{
		RoleUUID:    createdRole.RoleUUID,
		Name:        createdRole.Name,
		Description: createdRole.Description,
		IsDefault:   createdRole.IsDefault,
		IsActive:    createdRole.IsActive,
		CreatedAt:   createdRole.CreatedAt,
		UpdatedAt:   createdRole.UpdatedAt,
	}, nil
}

func (s *roleService) Update(roleUUID uuid.UUID, name string, description string, isDefault bool, isActive bool, authContainerId int64) (*RoleServiceDataResult, error) {
	var updatedRole *model.Role

	// Transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var role *model.Role
		var err error

		// Find existing role
		role, err = s.roleRepo.FindByUUID(roleUUID)
		if err != nil {
			return err
		}
		if role == nil {
			return errors.New("role not found")
		}

		if role.Name != name {

			existingRole, err := s.roleRepo.FindByNameAndAuthContainerID(name, authContainerId)
			if err != nil {
				return err
			}
			if existingRole != nil && existingRole.RoleUUID != roleUUID {
				return errors.New(name + " role already exist")
			}
		}

		// Create role
		role.Name = name
		role.Description = description
		role.IsDefault = isDefault
		role.IsActive = isActive

		if err := tx.Save(role).Error; err != nil {
			return err
		}

		updatedRole = role

		return nil // commit transaction
	})

	if err != nil {
		return nil, err
	}

	return &RoleServiceDataResult{
		RoleUUID:    updatedRole.RoleUUID,
		Name:        updatedRole.Name,
		Description: updatedRole.Description,
		IsDefault:   updatedRole.IsDefault,
		IsActive:    updatedRole.IsActive,
		CreatedAt:   updatedRole.CreatedAt,
		UpdatedAt:   updatedRole.UpdatedAt,
	}, nil
}

func (s *roleService) DeleteByUUID(roleUUID uuid.UUID) (*RoleServiceDataResult, error) {
	deletedRole, err := s.roleRepo.DeleteByUUID(roleUUID)
	if err != nil {
		return nil, err
	}
	if deletedRole == nil {
		return nil, errors.New("role not found")
	}

	return &RoleServiceDataResult{
		RoleUUID:    deletedRole.RoleUUID,
		Name:        deletedRole.Name,
		Description: deletedRole.Description,
		IsDefault:   deletedRole.IsDefault,
		IsActive:    deletedRole.IsActive,
		CreatedAt:   deletedRole.CreatedAt,
		UpdatedAt:   deletedRole.UpdatedAt,
	}, nil
}

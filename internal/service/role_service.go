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
	Permissions *[]PermissionServiceDataResult
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
	Create(name string, description string, IsDefault bool, IsActive bool, authContainerUUID string, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error)
	Update(roleUUID uuid.UUID, name string, description string, IsDefault bool, IsActive bool, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error)
	SetActiveStatusByUUID(roleUUID uuid.UUID, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error)
	DeleteByUUID(roleUUID uuid.UUID, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error)
	AddRolePermissions(roleUUID uuid.UUID, permissionUUIDs []uuid.UUID, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error)
	RemoveRolePermissions(roleUUID uuid.UUID, permissionUUID uuid.UUID, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error)
}

type roleService struct {
	db                 *gorm.DB
	roleRepo           repository.RoleRepository
	permissionRepo     repository.PermissionRepository
	rolePermissionRepo repository.RolePermissionRepository
	userRepo           repository.UserRepository
	authContainerRepo  repository.AuthContainerRepository
}

func NewRoleService(
	db *gorm.DB,
	roleRepo repository.RoleRepository,
	permissionRepo repository.PermissionRepository,
	rolePermissionRepo repository.RolePermissionRepository,
	userRepo repository.UserRepository,
	authContainerRepo repository.AuthContainerRepository,
) RoleService {
	return &roleService{
		db:                 db,
		roleRepo:           roleRepo,
		permissionRepo:     permissionRepo,
		rolePermissionRepo: rolePermissionRepo,
		userRepo:           userRepo,
		authContainerRepo:  authContainerRepo,
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
		roles[i] = *toRoleServiceDataResult(&r)
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

	return toRoleServiceDataResult(role), nil
}

func (s *roleService) Create(name string, description string, isDefault bool, isActive bool, authContainerUUID string, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error) {
	var createdRole *model.Role

	// Transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)
		txAuthContainerRepo := s.authContainerRepo.WithTx(tx)

		// Parse auth container UUID
		authContainerUUIDParsed, err := uuid.Parse(authContainerUUID)
		if err != nil {
			return errors.New("invalid auth container UUID")
		}

		// Validate auth container exists
		targetAuthContainer, err := txAuthContainerRepo.FindByUUID(authContainerUUIDParsed, "Organization")
		if err != nil || targetAuthContainer == nil {
			return errors.New("auth container not found")
		}

		// Get actor user with auth container and organization info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "AuthContainer.Organization")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate auth container access permissions
		if err := ValidateAuthContainerAccess(actorUser, targetAuthContainer); err != nil {
			return err
		}

		// Check if role already exist
		existingRole, err := txRoleRepo.FindByNameAndAuthContainerID(name, targetAuthContainer.AuthContainerID)
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
			AuthContainerID: targetAuthContainer.AuthContainerID,
		}

		_, err = txRoleRepo.CreateOrUpdate(newRole)
		if err != nil {
			return err
		}

		createdRole = newRole

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toRoleServiceDataResult(createdRole), nil
}

func (s *roleService) Update(roleUUID uuid.UUID, name string, description string, isDefault bool, isActive bool, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error) {
	var updatedRole *model.Role

	// Transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Find existing role
		role, err := txRoleRepo.FindByUUID(roleUUID, "AuthContainer.Organization")
		if err != nil {
			return err
		}
		if role == nil {
			return errors.New("role not found")
		}

		// Get actor user with auth container and organization info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "AuthContainer.Organization")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate auth container access permissions
		if err := ValidateAuthContainerAccess(actorUser, role.AuthContainer); err != nil {
			return err
		}

		// Check if role is a default record
		if role.IsDefault {
			return errors.New("default role is not allowed to be updated")
		}

		// If role name is changed, check if duplicate
		if role.Name != name {
			existingRole, err := txRoleRepo.FindByNameAndAuthContainerID(name, role.AuthContainerID)
			if err != nil {
				return err
			}
			if existingRole != nil && existingRole.RoleUUID != roleUUID {
				return errors.New(name + " role already exists")
			}
		}

		// Update role
		role.Name = name
		role.Description = description
		role.IsDefault = isDefault
		role.IsActive = isActive

		_, err = txRoleRepo.CreateOrUpdate(role)
		if err != nil {
			return err
		}

		updatedRole = role

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toRoleServiceDataResult(updatedRole), nil
}

func (s *roleService) SetActiveStatusByUUID(roleUUID uuid.UUID, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error) {
	var updatedRole *model.Role

	// Transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Find existing role
		role, err := txRoleRepo.FindByUUID(roleUUID, "AuthContainer.Organization")
		if err != nil {
			return err
		}
		if role == nil {
			return errors.New("role not found")
		}

		// Get actor user with auth container and organization info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "AuthContainer.Organization")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate auth container access permissions
		if err := ValidateAuthContainerAccess(actorUser, role.AuthContainer); err != nil {
			return err
		}

		// Check if role is a default record
		if role.IsDefault {
			return errors.New("default role is not allowed to be updated")
		}

		// Update role
		role.IsActive = !role.IsActive

		_, err = txRoleRepo.CreateOrUpdate(role)
		if err != nil {
			return err
		}

		updatedRole = role

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toRoleServiceDataResult(updatedRole), nil
}

func (s *roleService) DeleteByUUID(roleUUID uuid.UUID, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error) {
	// Check role existence
	role, err := s.roleRepo.FindByUUID(roleUUID, "AuthContainer.Organization")
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.New("role not found")
	}

	// Get actor user with auth container and organization info
	actorUser, err := s.userRepo.FindByUUID(actorUserUUID, "AuthContainer.Organization")
	if err != nil || actorUser == nil {
		return nil, errors.New("actor user not found")
	}

	// Validate auth container access permissions
	if err := ValidateAuthContainerAccess(actorUser, role.AuthContainer); err != nil {
		return nil, err
	}

	// Check if role is a default record
	if role.IsDefault {
		return nil, errors.New("default role is not allowed to be deleted")
	}

	// Delete role
	err = s.roleRepo.DeleteByUUID(roleUUID)
	if err != nil {
		return nil, err
	}

	return toRoleServiceDataResult(role), nil
}

func (s *roleService) AddRolePermissions(roleUUID uuid.UUID, permissionUUIDs []uuid.UUID, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error) {
	var roleWithPermissions *model.Role

	// Transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRoleRepo := s.roleRepo.WithTx(tx)
		txPermissionRepo := s.permissionRepo.WithTx(tx)
		txRolePermissionRepo := s.rolePermissionRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Find existing role
		role, err := txRoleRepo.FindByUUID(roleUUID, "AuthContainer.Organization")
		if err != nil {
			return err
		}
		if role == nil {
			return errors.New("role not found")
		}

		// Get actor user with auth container and organization info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "AuthContainer.Organization")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate auth container access permissions
		if err := ValidateAuthContainerAccess(actorUser, role.AuthContainer); err != nil {
			return err
		}

		// Check if role is a default record
		if role.IsDefault {
			return errors.New("default role is not allowed to be updated")
		}

		// Convert UUIDs to strings for the repository method
		permissionUUIDStrings := make([]string, len(permissionUUIDs))
		for i, uuid := range permissionUUIDs {
			permissionUUIDStrings[i] = uuid.String()
		}

		// Find permissions by UUIDs
		permissions, err := txPermissionRepo.FindByUUIDs(permissionUUIDStrings)
		if err != nil {
			return err
		}

		// Validate that all permissions were found
		if len(permissions) != len(permissionUUIDs) {
			return errors.New("one or more permissions not found")
		}

		// Create role-permission associations using the dedicated repository
		for _, permission := range permissions {
			// Check if association already exists
			existing, err := txRolePermissionRepo.FindByRoleAndPermission(role.RoleID, permission.PermissionID)
			if err != nil {
				return err
			}

			// Skip if association already exists
			if existing != nil {
				continue
			}

			// Create new role-permission association
			rolePermission := &model.RolePermission{
				RoleID:       role.RoleID,
				PermissionID: permission.PermissionID,
			}

			_, err = txRolePermissionRepo.Create(rolePermission)
			if err != nil {
				return err
			}
		}

		// Fetch the role with permissions for the response
		roleWithPermissions, err = txRoleRepo.FindByUUID(roleUUID, "Permissions")
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toRoleServiceDataResult(roleWithPermissions), nil
}

func (s *roleService) RemoveRolePermissions(roleUUID uuid.UUID, permissionUUID uuid.UUID, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error) {
	var roleWithPermissions *model.Role

	// Transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRoleRepo := s.roleRepo.WithTx(tx)
		txPermissionRepo := s.permissionRepo.WithTx(tx)
		txRolePermissionRepo := s.rolePermissionRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Find existing role
		role, err := txRoleRepo.FindByUUID(roleUUID, "AuthContainer.Organization")
		if err != nil {
			return err
		}
		if role == nil {
			return errors.New("role not found")
		}

		// Get actor user with auth container and organization info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "AuthContainer.Organization")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate auth container access permissions
		if err := ValidateAuthContainerAccess(actorUser, role.AuthContainer); err != nil {
			return err
		}

		// Check if role is a default record
		if role.IsDefault {
			return errors.New("default role is not allowed to be updated")
		}

		// Find permission by UUID
		permission, err := txPermissionRepo.FindByUUID(permissionUUID.String())
		if err != nil {
			return err
		}
		if permission == nil {
			return errors.New("permission not found")
		}

		// Check if association exists
		existing, err := txRolePermissionRepo.FindByRoleAndPermission(role.RoleID, permission.PermissionID)
		if err != nil {
			return err
		}

		// Skip if association doesn't exist
		if existing == nil {
			// Association doesn't exist, but we'll still return success for idempotency
			roleWithPermissions, err = txRoleRepo.FindByUUID(roleUUID, "Permissions")
			if err != nil {
				return err
			}
			return nil
		}

		// Remove the role-permission association
		err = txRolePermissionRepo.RemoveByRoleAndPermission(role.RoleID, permission.PermissionID)
		if err != nil {
			return err
		}

		// Fetch the role with permissions for the response
		roleWithPermissions, err = txRoleRepo.FindByUUID(roleUUID, "Permissions")
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toRoleServiceDataResult(roleWithPermissions), nil
}

// Reponse builder
func toRoleServiceDataResult(role *model.Role) *RoleServiceDataResult {
	if role == nil {
		return nil
	}

	result := &RoleServiceDataResult{
		RoleUUID:    role.RoleUUID,
		Name:        role.Name,
		Description: role.Description,
		IsDefault:   role.IsDefault,
		IsActive:    role.IsActive,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}

	if role.Permissions != nil {
		permissions := make([]PermissionServiceDataResult, len(role.Permissions))
		for i, p := range role.Permissions {
			permissions[i] = *toPermissionServiceDataResult(&p)
		}
		result.Permissions = &permissions
	}

	return result
}

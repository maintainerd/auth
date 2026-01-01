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
	IsSystem    bool
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type RoleServiceGetFilter struct {
	Name        *string
	Description *string
	IsDefault   *bool
	IsSystem    *bool
	Status      *string
	TenantID    int64
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type RoleServiceGetResult struct {
	Data       []RoleServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type RoleServiceGetPermissionsFilter struct {
	RoleUUID  uuid.UUID
	Status    *string
	TenantID  int64
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type RoleServiceGetPermissionsResult struct {
	Data       []PermissionServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type RoleService interface {
	Get(filter RoleServiceGetFilter) (*RoleServiceGetResult, error)
	GetByUUID(roleUUID uuid.UUID, tenantID int64) (*RoleServiceDataResult, error)
	GetRolePermissions(filter RoleServiceGetPermissionsFilter) (*RoleServiceGetPermissionsResult, error)
	Create(name string, description string, isDefault bool, isSystem bool, status string, tenantUUID string, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error)
	Update(roleUUID uuid.UUID, tenantID int64, name string, description string, isDefault bool, isSystem bool, status string, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error)
	SetStatusByUUID(roleUUID uuid.UUID, tenantID int64, status string, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error)
	DeleteByUUID(roleUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error)
	AddRolePermissions(roleUUID uuid.UUID, tenantID int64, permissionUUIDs []uuid.UUID, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error)
	RemoveRolePermissions(roleUUID uuid.UUID, tenantID int64, permissionUUID uuid.UUID, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error)
}

type roleService struct {
	db                 *gorm.DB
	roleRepo           repository.RoleRepository
	permissionRepo     repository.PermissionRepository
	rolePermissionRepo repository.RolePermissionRepository
	userRepo           repository.UserRepository
	tenantRepo         repository.TenantRepository
}

func NewRoleService(
	db *gorm.DB,
	roleRepo repository.RoleRepository,
	permissionRepo repository.PermissionRepository,
	rolePermissionRepo repository.RolePermissionRepository,
	userRepo repository.UserRepository,
	tenantRepo repository.TenantRepository,
) RoleService {
	return &roleService{
		db:                 db,
		roleRepo:           roleRepo,
		permissionRepo:     permissionRepo,
		rolePermissionRepo: rolePermissionRepo,
		userRepo:           userRepo,
		tenantRepo:         tenantRepo,
	}
}

func (s *roleService) Get(filter RoleServiceGetFilter) (*RoleServiceGetResult, error) {
	roleFilter := repository.RoleRepositoryGetFilter{
		Name:        filter.Name,
		Description: filter.Description,
		IsDefault:   filter.IsDefault,
		IsSystem:    filter.IsSystem,
		Status:      filter.Status,
		TenantID:    filter.TenantID,
		Page:        filter.Page,
		Limit:       filter.Limit,
		SortBy:      filter.SortBy,
		SortOrder:   filter.SortOrder,
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

func (s *roleService) GetByUUID(roleUUID uuid.UUID, tenantID int64) (*RoleServiceDataResult, error) {
	role, err := s.roleRepo.FindByUUID(roleUUID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.New("role not found")
	}

	// Validate tenant ownership
	if role.TenantID != tenantID {
		return nil, errors.New("role not found or access denied")
	}

	return toRoleServiceDataResult(role), nil
}

func (s *roleService) GetRolePermissions(filter RoleServiceGetPermissionsFilter) (*RoleServiceGetPermissionsResult, error) {
	// Verify role exists and belongs to tenant
	role, err := s.roleRepo.FindByUUID(filter.RoleUUID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.New("role not found")
	}
	if role.TenantID != filter.TenantID {
		return nil, errors.New("role not found")
	}

	// Build repository filter
	repoFilter := repository.RoleRepositoryGetPermissionsFilter{
		RoleUUID:  filter.RoleUUID,
		Status:    filter.Status,
		Page:      filter.Page,
		Limit:     filter.Limit,
		SortBy:    filter.SortBy,
		SortOrder: filter.SortOrder,
	}

	// Query paginated permissions
	result, err := s.roleRepo.GetPermissionsByRoleUUID(repoFilter)
	if err != nil {
		return nil, err
	}

	// Map to service result
	permissions := make([]PermissionServiceDataResult, len(result.Data))
	for i, p := range result.Data {
		permissions[i] = *toPermissionServiceDataResult(&p)
	}

	return &RoleServiceGetPermissionsResult{
		Data:       permissions,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *roleService) Create(name string, description string, isDefault bool, isSystem bool, status string, tenantUUID string, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error) {
	var createdRole *model.Role

	// Transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)
		txTenantRepo := s.tenantRepo.WithTx(tx)

		// Parse tenant UUID
		tenantUUIDParsed, err := uuid.Parse(tenantUUID)
		if err != nil {
			return errors.New("invalid tenant UUID")
		}

		// Validate tenant exists
		targetTenant, err := txTenantRepo.FindByUUID(tenantUUIDParsed)
		if err != nil || targetTenant == nil {
			return errors.New("tenant not found")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, targetTenant); err != nil {
			return err
		}

		// Check if role already exist
		existingRole, err := txRoleRepo.FindByNameAndTenantID(name, targetTenant.TenantID)
		if err != nil {
			return err
		}
		if existingRole != nil {
			return errors.New(name + " role already exist")
		}

		// Create role
		newRole := &model.Role{
			Name:        name,
			Description: description,
			IsDefault:   isDefault,
			IsSystem:    isSystem,
			Status:      status,
			TenantID:    targetTenant.TenantID,
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

func (s *roleService) Update(roleUUID uuid.UUID, tenantID int64, name string, description string, isDefault bool, isSystem bool, status string, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error) {
	var updatedRole *model.Role

	// Transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Find existing role
		role, err := txRoleRepo.FindByUUID(roleUUID, "Tenant")
		if err != nil {
			return err
		}
		if role == nil {
			return errors.New("role not found")
		}

		// Validate tenant ownership
		if role.TenantID != tenantID {
			return errors.New("role not found or access denied")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, role.Tenant); err != nil {
			return err
		}

		// Check if role is a system record
		if role.IsSystem {
			return errors.New("system role is not allowed to be updated")
		}

		// If role name is changed, check if duplicate
		if role.Name != name {
			existingRole, err := txRoleRepo.FindByNameAndTenantID(name, role.TenantID)
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
		role.IsSystem = isSystem
		role.Status = status

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

func (s *roleService) SetStatusByUUID(roleUUID uuid.UUID, tenantID int64, status string, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error) {
	var updatedRole *model.Role

	// Transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Find existing role
		role, err := txRoleRepo.FindByUUID(roleUUID, "Tenant")
		if err != nil {
			return err
		}
		if role == nil {
			return errors.New("role not found")
		}

		// Validate tenant ownership
		if role.TenantID != tenantID {
			return errors.New("role not found or access denied")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, role.Tenant); err != nil {
			return err
		}

		// Check if role is a system record
		if role.IsSystem {
			return errors.New("system role is not allowed to be updated")
		}

		// Update role
		role.Status = status

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

func (s *roleService) DeleteByUUID(roleUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error) {
	// Check role existence
	role, err := s.roleRepo.FindByUUID(roleUUID, "Tenant")
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.New("role not found")
	}

	// Validate tenant ownership
	if role.TenantID != tenantID {
		return nil, errors.New("role not found or access denied")
	}

	// Get actor user with tenant info
	actorUser, err := s.userRepo.FindByUUID(actorUserUUID, "Tenant")
	if err != nil || actorUser == nil {
		return nil, errors.New("actor user not found")
	}

	// Validate tenant access permissions
	if err := ValidateTenantAccess(actorUser, role.Tenant); err != nil {
		return nil, err
	}

	// Check if role is a system record
	if role.IsSystem {
		return nil, errors.New("system role is not allowed to be deleted")
	}

	// Delete role
	err = s.roleRepo.DeleteByUUID(roleUUID)
	if err != nil {
		return nil, err
	}

	return toRoleServiceDataResult(role), nil
}

func (s *roleService) AddRolePermissions(roleUUID uuid.UUID, tenantID int64, permissionUUIDs []uuid.UUID, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error) {
	var roleWithPermissions *model.Role

	// Transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRoleRepo := s.roleRepo.WithTx(tx)
		txPermissionRepo := s.permissionRepo.WithTx(tx)
		txRolePermissionRepo := s.rolePermissionRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Find existing role
		role, err := txRoleRepo.FindByUUID(roleUUID, "Tenant")
		if err != nil {
			return err
		}
		if role == nil {
			return errors.New("role not found")
		}

		// Validate tenant ownership
		if role.TenantID != tenantID {
			return errors.New("role not found or access denied")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, role.Tenant); err != nil {
			return err
		}

		// Check if role is a system record
		if role.IsSystem {
			return errors.New("system role is not allowed to be updated")
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

func (s *roleService) RemoveRolePermissions(roleUUID uuid.UUID, tenantID int64, permissionUUID uuid.UUID, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error) {
	var roleWithPermissions *model.Role

	// Transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRoleRepo := s.roleRepo.WithTx(tx)
		txPermissionRepo := s.permissionRepo.WithTx(tx)
		txRolePermissionRepo := s.rolePermissionRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Find existing role
		role, err := txRoleRepo.FindByUUID(roleUUID, "Tenant")
		if err != nil {
			return err
		}
		if role == nil {
			return errors.New("role not found")
		}

		// Validate tenant ownership
		if role.TenantID != tenantID {
			return errors.New("role not found or access denied")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, role.Tenant); err != nil {
			return err
		}

		// Check if role is a system record
		if role.IsSystem {
			return errors.New("system role is not allowed to be updated")
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
		IsSystem:    role.IsSystem,
		Status:      role.Status,
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

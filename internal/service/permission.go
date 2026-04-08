package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"gorm.io/gorm"
)

type PermissionServiceDataResult struct {
	PermissionUUID uuid.UUID
	Name           string
	Description    string
	API            *APIServiceDataResult
	Status         string
	IsDefault      bool
	IsSystem       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type PermissionServiceGetFilter struct {
	TenantID    int64
	Name        *string
	Description *string
	APIUUID     *string
	RoleUUID    *string
	ClientUUID  *string
	Status      *string
	IsDefault   *bool
	IsSystem    *bool
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type PermissionServiceGetResult struct {
	Data       []PermissionServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type PermissionService interface {
	Get(filter PermissionServiceGetFilter) (*PermissionServiceGetResult, error)
	GetByUUID(permissionUUID uuid.UUID, tenantID int64) (*PermissionServiceDataResult, error)
	Create(tenantID int64, name string, description string, status string, isSystem bool, apiUUID string) (*PermissionServiceDataResult, error)
	Update(permissionUUID uuid.UUID, tenantID int64, name string, description string, status string) (*PermissionServiceDataResult, error)
	SetActiveStatusByUUID(permissionUUID uuid.UUID, tenantID int64) (*PermissionServiceDataResult, error)
	SetStatus(permissionUUID uuid.UUID, tenantID int64, status string) (*PermissionServiceDataResult, error)
	DeleteByUUID(permissionUUID uuid.UUID, tenantID int64) (*PermissionServiceDataResult, error)
}

type permissionService struct {
	db             *gorm.DB
	permissionRepo repository.PermissionRepository
	apiRepo        repository.APIRepository
	roleRepo       repository.RoleRepository
	clientRepo     repository.ClientRepository
}

func NewPermissionService(
	db *gorm.DB,
	permissionRepo repository.PermissionRepository,
	apiRepo repository.APIRepository,
	roleRepo repository.RoleRepository,
	clientRepo repository.ClientRepository,
) PermissionService {
	return &permissionService{
		db:             db,
		permissionRepo: permissionRepo,
		apiRepo:        apiRepo,
		roleRepo:       roleRepo,
		clientRepo:     clientRepo,
	}
}

func (s *permissionService) Get(filter PermissionServiceGetFilter) (*PermissionServiceGetResult, error) {
	var apiID *int64
	var roleID *int64

	// Get api if uuid exist
	if filter.APIUUID != nil {
		api, err := s.apiRepo.FindByUUID(*filter.APIUUID)
		if err != nil || api == nil {
			return nil, apperror.NewNotFound("api not found")
		}
		apiID = &api.APIID
	}

	// Get role if uuid exist
	if filter.RoleUUID != nil {
		role, err := s.roleRepo.FindByUUID(*filter.RoleUUID)
		if err != nil || role == nil {
			return nil, apperror.NewNotFound("role not found")
		}
		roleID = &role.RoleID
	}

	// Note: ClientUUID filtering is no longer supported in the new hierarchical structure
	// Auth client permissions are now managed through the client_apis -> client_permissions relationship
	// Use the auth client API endpoints instead: /clients/{client_uuid}/apis/{api_uuid}/permissions
	if filter.ClientUUID != nil {
		return nil, apperror.NewValidation("auth client filtering is no longer supported - use auth client API endpoints instead")
	}

	// Build query filter
	queryFilter := repository.PermissionRepositoryGetFilter{
		TenantID:    filter.TenantID,
		Name:        filter.Name,
		Description: filter.Description,
		APIID:       apiID,
		RoleID:      roleID,
		Status:      filter.Status,
		IsDefault:   filter.IsDefault,
		IsSystem:    filter.IsSystem,
		Page:        filter.Page,
		Limit:       filter.Limit,
		SortBy:      filter.SortBy,
		SortOrder:   filter.SortOrder,
	}

	result, err := s.permissionRepo.FindPaginated(queryFilter)
	if err != nil {
		return nil, err
	}

	// Build response data
	resData := make([]PermissionServiceDataResult, len(result.Data))
	for i, rdata := range result.Data {
		resData[i] = *toPermissionServiceDataResult(&rdata)
	}

	return &PermissionServiceGetResult{
		Data:       resData,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *permissionService) GetByUUID(permissionUUID uuid.UUID, tenantID int64) (*PermissionServiceDataResult, error) {
	permission, err := s.permissionRepo.FindByUUIDAndTenantID(permissionUUID, tenantID)
	if err != nil {
		return nil, err
	}
	if permission == nil {
		return nil, apperror.NewNotFoundWithReason("permission not found or access denied")
	}

	return toPermissionServiceDataResult(permission), nil
}

func (s *permissionService) Create(tenantID int64, name string, description string, status string, isSystem bool, apiUUID string) (*PermissionServiceDataResult, error) {
	var createdPermission *model.Permission

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txPermissionRepo := s.permissionRepo.WithTx(tx)
		txAPIRepo := s.apiRepo.WithTx(tx)

		// Check if permission already exists
		existingPermission, err := txPermissionRepo.FindByName(name, tenantID)
		if err != nil {
			return err
		}
		if existingPermission != nil {
			return apperror.NewConflict(name + " permission already exists")
		}

		// Parse and validate API UUID
		apiUUIDParsed, err := uuid.Parse(apiUUID)
		if err != nil {
			return apperror.NewValidation("invalid api uuid")
		}

		// Check if api exists and belongs to the same tenant
		api, err := txAPIRepo.FindByUUIDAndTenantID(apiUUIDParsed, tenantID)
		if err != nil {
			return err
		}
		if api == nil {
			return apperror.NewNotFoundWithReason("api not found or access denied")
		}

		// Create permission
		newPermission := &model.Permission{
			Name:        name,
			Description: description,
			TenantID:    tenantID,
			APIID:       api.APIID,
			Status:      status,
			IsDefault:   false, // System-managed field, always default to false for user-created permissions
			IsSystem:    isSystem,
		}

		_, err = txPermissionRepo.CreateOrUpdate(newPermission)
		if err != nil {
			return err
		}

		// Fetch permission with api preloaded
		createdPermission, err = txPermissionRepo.FindByUUIDAndTenantID(newPermission.PermissionUUID, tenantID)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toPermissionServiceDataResult(createdPermission), nil
}

func (s *permissionService) Update(permissionUUID uuid.UUID, tenantID int64, name string, description string, status string) (*PermissionServiceDataResult, error) {
	var updatedPermission *model.Permission

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txPermissionRepo := s.permissionRepo.WithTx(tx)

		// Get permission and validate tenant ownership
		permission, err := txPermissionRepo.FindByUUIDAndTenantID(permissionUUID, tenantID)
		if err != nil {
			return err
		}
		if permission == nil {
			return apperror.NewNotFoundWithReason("permission not found or access denied")
		}

		// Check if default
		if permission.IsDefault {
			return apperror.NewValidation("default permission cannot cannot be updated")
		}

		// Check if permission already exist
		if permission.Name != name {
			existingPermission, err := txPermissionRepo.FindByName(name, tenantID)
			if err != nil {
				return err
			}
			if existingPermission != nil && existingPermission.PermissionUUID != permissionUUID {
				return apperror.NewConflict(name + " permission already exists")
			}
		}

		// Set values
		permission.Name = name
		permission.Description = description
		permission.Status = status

		// Update
		_, err = txPermissionRepo.CreateOrUpdate(permission)
		if err != nil {
			return err
		}

		updatedPermission = permission

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toPermissionServiceDataResult(updatedPermission), nil
}

func (s *permissionService) SetActiveStatusByUUID(permissionUUID uuid.UUID, tenantID int64) (*PermissionServiceDataResult, error) {
	var updatedPermission *model.Permission

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txPermissionRepo := s.permissionRepo.WithTx(tx)

		// Get permission and validate tenant ownership
		permission, err := txPermissionRepo.FindByUUIDAndTenantID(permissionUUID, tenantID)
		if err != nil {
			return err
		}
		if permission == nil {
			return apperror.NewNotFoundWithReason("permission not found or access denied")
		}

		// Check if default
		if permission.IsDefault {
			return apperror.NewValidation("default permission cannot be updated")
		}

		if permission.Status == model.StatusActive {
			permission.Status = model.StatusInactive
		} else {
			permission.Status = model.StatusActive
		}

		_, err = txPermissionRepo.CreateOrUpdate(permission)
		if err != nil {
			return err
		}

		updatedPermission = permission

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toPermissionServiceDataResult(updatedPermission), nil
}

func (s *permissionService) SetStatus(permissionUUID uuid.UUID, tenantID int64, status string) (*PermissionServiceDataResult, error) {
	var updatedPermission *model.Permission

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txPermissionRepo := s.permissionRepo.WithTx(tx)

		// Get permission and validate tenant ownership
		permission, err := txPermissionRepo.FindByUUIDAndTenantID(permissionUUID, tenantID)
		if err != nil {
			return err
		}
		if permission == nil {
			return apperror.NewNotFoundWithReason("permission not found or access denied")
		}

		// Set status
		permission.Status = status

		// Update
		_, err = txPermissionRepo.CreateOrUpdate(permission)
		if err != nil {
			return err
		}

		updatedPermission = permission
		return nil
	})

	if err != nil {
		return nil, err
	}

	return toPermissionServiceDataResult(updatedPermission), nil
}

func (s *permissionService) DeleteByUUID(permissionUUID uuid.UUID, tenantID int64) (*PermissionServiceDataResult, error) {
	// Get permission and validate tenant ownership
	permission, err := s.permissionRepo.FindByUUIDAndTenantID(permissionUUID, tenantID)
	if err != nil {
		return nil, err
	}
	if permission == nil {
		return nil, apperror.NewNotFoundWithReason("permission not found or access denied")
	}

	// Check if default
	if permission.IsDefault {
		return nil, apperror.NewValidation("default permission cannot be deleted")
	}

	err = s.permissionRepo.DeleteByUUIDAndTenantID(permissionUUID, tenantID)
	if err != nil {
		return nil, err
	}

	return toPermissionServiceDataResult(permission), nil
}

// Reponse builder
func toPermissionServiceDataResult(permission *model.Permission) *PermissionServiceDataResult {
	if permission == nil {
		return nil
	}

	result := &PermissionServiceDataResult{
		PermissionUUID: permission.PermissionUUID,
		Name:           permission.Name,
		Description:    permission.Description,
		Status:         permission.Status,
		IsDefault:      permission.IsDefault,
		IsSystem:       permission.IsSystem,
		CreatedAt:      permission.CreatedAt,
		UpdatedAt:      permission.UpdatedAt,
	}

	if permission.API != nil {
		result.API = &APIServiceDataResult{
			APIUUID:     permission.API.APIUUID,
			Name:        permission.API.Name,
			DisplayName: permission.API.DisplayName,
			Description: permission.API.Description,
			APIType:     permission.API.APIType,
			Identifier:  permission.API.Identifier,
			Status:      permission.API.Status,
			CreatedAt:   permission.API.CreatedAt,
			UpdatedAt:   permission.API.UpdatedAt,
		}
	}

	return result
}

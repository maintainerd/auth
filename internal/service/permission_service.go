package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"gorm.io/gorm"
)

type PermissionServiceDataResult struct {
	PermissionUUID uuid.UUID
	Name           string
	Description    string
	API            *APIServiceDataResult
	IsActive       bool
	IsDefault      bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type PermissionServiceGetFilter struct {
	Name           *string
	Description    *string
	APIUUID        *string
	RoleUUID       *string
	AuthClientUUID *string
	IsActive       *bool
	IsDefault      *bool
	Page           int
	Limit          int
	SortBy         string
	SortOrder      string
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
	GetByUUID(permissionUUID uuid.UUID) (*PermissionServiceDataResult, error)
	Create(name string, description string, isActive bool, isDefault bool, apiUUID string) (*PermissionServiceDataResult, error)
	Update(permissionUUID uuid.UUID, name string, description string, isActive bool, isDefault bool) (*PermissionServiceDataResult, error)
	SetActiveStatusByUUID(permissionUUID uuid.UUID) (*PermissionServiceDataResult, error)
	DeleteByUUID(permissionUUID uuid.UUID) (*PermissionServiceDataResult, error)
}

type permissionService struct {
	db             *gorm.DB
	permissionRepo repository.PermissionRepository
	apiRepo        repository.APIRepository
	roleRepo       repository.RoleRepository
	authClientRepo repository.AuthClientRepository
}

func NewPermissionService(
	db *gorm.DB,
	permissionRepo repository.PermissionRepository,
	apiRepo repository.APIRepository,
	roleRepo repository.RoleRepository,
	authClientRepo repository.AuthClientRepository,
) PermissionService {
	return &permissionService{
		db:             db,
		permissionRepo: permissionRepo,
		apiRepo:        apiRepo,
		roleRepo:       roleRepo,
		authClientRepo: authClientRepo,
	}
}

func (s *permissionService) Get(filter PermissionServiceGetFilter) (*PermissionServiceGetResult, error) {
	var apiID *int64
	var roleID *int64
	var authClientID *int64

	// Get api if uuid exist
	if filter.APIUUID != nil {
		api, err := s.apiRepo.FindByUUID(*filter.APIUUID)
		if err != nil || api == nil {
			return nil, errors.New("api not found")
		}
		apiID = &api.APIID
	}

	// Get role if uuid exist
	if filter.RoleUUID != nil {
		role, err := s.roleRepo.FindByUUID(*filter.RoleUUID)
		if err != nil || role == nil {
			return nil, errors.New("role not found")
		}
		roleID = &role.RoleID
	}

	// Get auth client if uuid exist
	if filter.AuthClientUUID != nil {
		authClient, err := s.authClientRepo.FindByUUID(*filter.AuthClientUUID)
		if err != nil || authClient == nil {
			return nil, errors.New("auth client not found")
		}
		authClientID = &authClient.AuthClientID
	}

	// Build query filter
	queryFilter := repository.PermissionRepositoryGetFilter{
		Name:         filter.Name,
		Description:  filter.Description,
		APIID:        apiID,
		RoleID:       roleID,
		AuthClientID: authClientID,
		IsActive:     filter.IsActive,
		IsDefault:    filter.IsDefault,
		Page:         filter.Page,
		Limit:        filter.Limit,
		SortBy:       filter.SortBy,
		SortOrder:    filter.SortOrder,
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

func (s *permissionService) GetByUUID(permissionUUID uuid.UUID) (*PermissionServiceDataResult, error) {
	permission, err := s.permissionRepo.FindByUUID(permissionUUID, "API")
	if err != nil || permission == nil {
		return nil, errors.New("permission not found")
	}

	return toPermissionServiceDataResult(permission), nil
}

func (s *permissionService) Create(name string, description string, isActive bool, isDefault bool, apiUUID string) (*PermissionServiceDataResult, error) {
	var createdPermission *model.Permission

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txPermissionRepo := s.permissionRepo.WithTx(tx)
		txAPIRepo := s.apiRepo.WithTx(tx)

		// Check if permission already exists
		existingPermission, err := txPermissionRepo.FindByName(name)
		if err != nil {
			return err
		}
		if existingPermission != nil {
			return errors.New(name + " permission already exists")
		}

		// Check if api exist
		api, err := txAPIRepo.FindByUUID(apiUUID)
		if err != nil || api == nil {
			return errors.New("api not found")
		}

		// Create permission
		newPermission := &model.Permission{
			Name:        name,
			Description: description,
			APIID:       api.APIID,
			IsActive:    isActive,
			IsDefault:   isDefault,
		}

		_, err = txPermissionRepo.CreateOrUpdate(newPermission)
		if err != nil {
			return err
		}

		// Fetch permission with api preloaded
		createdPermission, err = txPermissionRepo.FindByUUID(newPermission.PermissionUUID, "API")
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

func (s *permissionService) Update(permissionUUID uuid.UUID, name string, description string, isActive bool, isDefault bool) (*PermissionServiceDataResult, error) {
	var updatedPermission *model.Permission

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txPermissionRepo := s.permissionRepo.WithTx(tx)

		// Get permission
		permission, err := txPermissionRepo.FindByUUID(permissionUUID, "API")
		if err != nil || permission == nil {
			return errors.New("permission not found")
		}

		// Check if default
		if permission.IsDefault {
			return errors.New("default permission cannot cannot be updated")
		}

		// Check if permission already exist
		if permission.Name != name {
			existingPermission, err := txPermissionRepo.FindByName(name)
			if err != nil {
				return err
			}
			if existingPermission != nil && existingPermission.PermissionUUID != permissionUUID {
				return errors.New(name + " permission already exists")
			}
		}

		// Set values
		permission.Name = name
		permission.Description = description
		permission.IsActive = isActive

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

func (s *permissionService) SetActiveStatusByUUID(permissionUUID uuid.UUID) (*PermissionServiceDataResult, error) {
	var updatedPermission *model.Permission

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txPermissionRepo := s.permissionRepo.WithTx(tx)

		// Get permission
		permission, err := txPermissionRepo.FindByUUID(permissionUUID, "API")
		if err != nil || permission == nil {
			return errors.New("permission not found")
		}

		// Check if default
		if permission.IsDefault {
			return errors.New("default permission cannot be updated")
		}

		permission.IsActive = !permission.IsActive

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

func (s *permissionService) DeleteByUUID(permissionUUID uuid.UUID) (*PermissionServiceDataResult, error) {
	// Get permission
	permission, err := s.permissionRepo.FindByUUID(permissionUUID, "API")
	if err != nil || permission == nil {
		return nil, errors.New("permission not found")
	}

	// Check if default
	if permission.IsDefault {
		return nil, errors.New("default permission cannot be deleted")
	}

	err = s.permissionRepo.DeleteByUUID(permissionUUID)
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
		IsActive:       permission.IsActive,
		IsDefault:      permission.IsDefault,
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
			IsDefault:   permission.API.IsDefault,
			CreatedAt:   permission.API.CreatedAt,
			UpdatedAt:   permission.API.UpdatedAt,
		}
	}

	return result
}

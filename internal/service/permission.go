package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/cache"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	Get(ctx context.Context, filter PermissionServiceGetFilter) (*PermissionServiceGetResult, error)
	GetByUUID(ctx context.Context, permissionUUID uuid.UUID, tenantID int64) (*PermissionServiceDataResult, error)
	Create(ctx context.Context, tenantID int64, name string, description string, status string, isSystem bool, apiUUID string) (*PermissionServiceDataResult, error)
	Update(ctx context.Context, permissionUUID uuid.UUID, tenantID int64, name string, description string, status string) (*PermissionServiceDataResult, error)
	SetActiveStatusByUUID(ctx context.Context, permissionUUID uuid.UUID, tenantID int64) (*PermissionServiceDataResult, error)
	SetStatus(ctx context.Context, permissionUUID uuid.UUID, tenantID int64, status string) (*PermissionServiceDataResult, error)
	DeleteByUUID(ctx context.Context, permissionUUID uuid.UUID, tenantID int64) (*PermissionServiceDataResult, error)
}

type permissionService struct {
	db               *gorm.DB
	permissionRepo   repository.PermissionRepository
	apiRepo          repository.APIRepository
	roleRepo         repository.RoleRepository
	clientRepo       repository.ClientRepository
	cacheInvalidator cache.Invalidator
}

func NewPermissionService(
	db *gorm.DB,
	permissionRepo repository.PermissionRepository,
	apiRepo repository.APIRepository,
	roleRepo repository.RoleRepository,
	clientRepo repository.ClientRepository,
	cacheInvalidator cache.Invalidator,
) PermissionService {
	return &permissionService{
		db:               db,
		permissionRepo:   permissionRepo,
		apiRepo:          apiRepo,
		roleRepo:         roleRepo,
		clientRepo:       clientRepo,
		cacheInvalidator: cacheInvalidator,
	}
}

func (s *permissionService) Get(ctx context.Context, filter PermissionServiceGetFilter) (*PermissionServiceGetResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "permission.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", filter.TenantID))
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "list permissions failed")
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

func (s *permissionService) GetByUUID(ctx context.Context, permissionUUID uuid.UUID, tenantID int64) (*PermissionServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "permission.get")
	defer span.End()
	span.SetAttributes(attribute.String("permission.uuid", permissionUUID.String()), attribute.Int64("tenant.id", tenantID))
	permission, err := s.permissionRepo.FindByUUIDAndTenantID(permissionUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get permission failed")
		return nil, err
	}
	if permission == nil {
		return nil, apperror.NewNotFoundWithReason("permission not found or access denied")
	}
	span.SetStatus(codes.Ok, "")
	return toPermissionServiceDataResult(permission), nil
}

func (s *permissionService) Create(ctx context.Context, tenantID int64, name string, description string, status string, isSystem bool, apiUUID string) (*PermissionServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "permission.create")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "create permission failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toPermissionServiceDataResult(createdPermission), nil
}

func (s *permissionService) Update(ctx context.Context, permissionUUID uuid.UUID, tenantID int64, name string, description string, status string) (*PermissionServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "permission.update")
	defer span.End()
	span.SetAttributes(attribute.String("permission.uuid", permissionUUID.String()), attribute.Int64("tenant.id", tenantID))
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "update permission failed")
		return nil, err
	}

	s.cacheInvalidator.InvalidateAllUsers(ctx)
	span.SetStatus(codes.Ok, "")
	return toPermissionServiceDataResult(updatedPermission), nil
}

func (s *permissionService) SetActiveStatusByUUID(ctx context.Context, permissionUUID uuid.UUID, tenantID int64) (*PermissionServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "permission.setActiveStatus")
	defer span.End()
	span.SetAttributes(attribute.String("permission.uuid", permissionUUID.String()), attribute.Int64("tenant.id", tenantID))
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "set active status failed")
		return nil, err
	}

	s.cacheInvalidator.InvalidateAllUsers(ctx)
	span.SetStatus(codes.Ok, "")
	return toPermissionServiceDataResult(updatedPermission), nil
}

func (s *permissionService) SetStatus(ctx context.Context, permissionUUID uuid.UUID, tenantID int64, status string) (*PermissionServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "permission.setStatus")
	defer span.End()
	span.SetAttributes(attribute.String("permission.uuid", permissionUUID.String()), attribute.Int64("tenant.id", tenantID))
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "set status failed")
		return nil, err
	}

	s.cacheInvalidator.InvalidateAllUsers(ctx)
	span.SetStatus(codes.Ok, "")
	return toPermissionServiceDataResult(updatedPermission), nil
}

func (s *permissionService) DeleteByUUID(ctx context.Context, permissionUUID uuid.UUID, tenantID int64) (*PermissionServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "permission.delete")
	defer span.End()
	span.SetAttributes(attribute.String("permission.uuid", permissionUUID.String()), attribute.Int64("tenant.id", tenantID))
	// Get permission and validate tenant ownership
	permission, err := s.permissionRepo.FindByUUIDAndTenantID(permissionUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete permission failed")
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete permission failed")
		return nil, err
	}

	s.cacheInvalidator.InvalidateAllUsers(ctx)
	span.SetStatus(codes.Ok, "")
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

package iam

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/event"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	Get(ctx context.Context, filter RoleServiceGetFilter) (*RoleServiceGetResult, error)
	GetByUUID(ctx context.Context, roleUUID uuid.UUID, tenantID int64) (*RoleServiceDataResult, error)
	GetRolePermissions(ctx context.Context, filter RoleServiceGetPermissionsFilter) (*RoleServiceGetPermissionsResult, error)
	Create(ctx context.Context, name string, description string, isDefault bool, isSystem bool, status string, tenantUUID string, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error)
	Update(ctx context.Context, roleUUID uuid.UUID, tenantID int64, name string, description string, isDefault bool, isSystem bool, status string, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error)
	SetStatusByUUID(ctx context.Context, roleUUID uuid.UUID, tenantID int64, status string, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error)
	DeleteByUUID(ctx context.Context, roleUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error)
	AddRolePermissions(ctx context.Context, roleUUID uuid.UUID, tenantID int64, permissionUUIDs []uuid.UUID, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error)
	RemoveRolePermissions(ctx context.Context, roleUUID uuid.UUID, tenantID int64, permissionUUID uuid.UUID, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error)
}

type roleService struct {
	db                 *gorm.DB
	roleRepo           RoleRepository
	permissionRepo     PermissionRepository
	rolePermissionRepo RolePermissionRepository
	userRepo           UserRepository
	tenantRepo         TenantRepository
	authzInvalidator   AuthorizationTokenInvalidator
	authEventService   authevent.AuthEventService
	eventService       event.EventService
}

func NewRoleService(
	db *gorm.DB,
	roleRepo RoleRepository,
	permissionRepo PermissionRepository,
	rolePermissionRepo RolePermissionRepository,
	userRepo UserRepository,
	tenantRepo TenantRepository,
	cacheInvalidator cache.Invalidator,
	authEventService authevent.AuthEventService,
	eventService event.EventService,
	authzInvalidator ...AuthorizationTokenInvalidator,
) RoleService {
	invalidator := AuthorizationTokenInvalidator(noopAuthorizationTokenInvalidator{})
	if len(authzInvalidator) > 0 && authzInvalidator[0] != nil {
		invalidator = authzInvalidator[0]
	}
	return &roleService{
		db:                 db,
		roleRepo:           roleRepo,
		permissionRepo:     permissionRepo,
		rolePermissionRepo: rolePermissionRepo,
		userRepo:           userRepo,
		tenantRepo:         tenantRepo,
		authzInvalidator:   invalidator,
		authEventService:   coalesceAuthEventService(authEventService),
		eventService:       eventService,
	}
}

func (s *roleService) Get(ctx context.Context, filter RoleServiceGetFilter) (*RoleServiceGetResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "role.list")
	defer span.End()

	roleFilter := RoleRepositoryGetFilter(filter)

	// Query paginated roles
	result, err := s.roleRepo.FindPaginated(roleFilter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list roles failed")
		return nil, err
	}

	roles := make([]RoleServiceDataResult, len(result.Data))
	for i, r := range result.Data {
		roles[i] = *toRoleServiceDataResult(&r)
	}

	span.SetStatus(codes.Ok, "")
	return &RoleServiceGetResult{
		Data:       roles,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *roleService) GetByUUID(ctx context.Context, roleUUID uuid.UUID, tenantID int64) (*RoleServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "role.get")
	defer span.End()
	span.SetAttributes(attribute.String("role.uuid", roleUUID.String()), attribute.Int64("tenant.id", tenantID))

	role, err := s.roleRepo.FindByUUID(roleUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get role failed")
		return nil, err
	}
	if role == nil {
		span.SetStatus(codes.Error, "get role failed")
		return nil, apperror.NewNotFound("role not found")
	}

	// Validate tenant ownership
	if role.TenantID != tenantID {
		span.SetStatus(codes.Error, "get role failed")
		return nil, apperror.NewNotFoundWithReason("role not found or access denied")
	}

	span.SetStatus(codes.Ok, "")
	return toRoleServiceDataResult(role), nil
}

func (s *roleService) GetRolePermissions(ctx context.Context, filter RoleServiceGetPermissionsFilter) (*RoleServiceGetPermissionsResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "role.listPermissions")
	defer span.End()
	span.SetAttributes(attribute.String("role.uuid", filter.RoleUUID.String()), attribute.Int64("tenant.id", filter.TenantID))

	// Verify role exists and belongs to tenant
	role, err := s.roleRepo.FindByUUID(filter.RoleUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list role permissions failed")
		return nil, err
	}
	if role == nil {
		span.SetStatus(codes.Error, "list role permissions failed")
		return nil, apperror.NewNotFound("role not found")
	}
	if role.TenantID != filter.TenantID {
		span.SetStatus(codes.Error, "list role permissions failed")
		return nil, apperror.NewNotFound("role not found")
	}

	// Build repository filter
	repoFilter := RoleRepositoryGetPermissionsFilter{
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "list role permissions failed")
		return nil, err
	}

	// Map to service result
	permissions := make([]PermissionServiceDataResult, len(result.Data))
	for i, p := range result.Data {
		permissions[i] = *toPermissionServiceDataResult(&p)
	}

	span.SetStatus(codes.Ok, "")
	return &RoleServiceGetPermissionsResult{
		Data:       permissions,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *roleService) Create(ctx context.Context, name string, description string, isDefault bool, isSystem bool, status string, tenantUUID string, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "role.create")
	defer span.End()
	span.SetAttributes(attribute.String("tenant.uuid", tenantUUID))

	var createdRole *Role
	var capturedTenantID, capturedActorID int64

	// Transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRoleRepo := s.roleRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)
		txTenantRepo := s.tenantRepo.WithTx(tx)

		// Parse tenant UUID
		tenantUUIDParsed, err := uuid.Parse(tenantUUID)
		if err != nil {
			return apperror.NewValidation("invalid tenant UUID")
		}

		// Validate tenant exists
		targetTenant, err := txTenantRepo.FindByUUID(tenantUUIDParsed)
		if err != nil || targetTenant == nil {
			return apperror.NewNotFound("tenant not found")
		}

		// Get actor user with user identities for tenant validation
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, targetTenant); err != nil {
			return err
		}
		capturedTenantID = targetTenant.TenantID
		capturedActorID = actorUser.UserID

		// Check if role already exist
		existingRole, err := txRoleRepo.FindByNameAndTenantID(name, targetTenant.TenantID)
		if err != nil {
			return err
		}
		if existingRole != nil {
			return apperror.NewConflict(name + " role already exist")
		}

		// Create role
		newRole := &Role{
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

		// Emit role.created integration event inside the transaction
		if s.eventService != nil {
			s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeRoleCreated, 1, targetTenant.TenantID,
			).SetActor(&actorUser.UserID).SetSubject(&createdRole.RoleUUID, "role"))
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create role failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    capturedTenantID,
		ActorUserID: &capturedActorID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthz,
		EventType:   authevent.AuthEventTypeAuthzAdmin,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Role created: %s", createdRole.Name)),
	})
	return toRoleServiceDataResult(createdRole), nil
}

func (s *roleService) Update(ctx context.Context, roleUUID uuid.UUID, tenantID int64, name string, description string, isDefault bool, isSystem bool, status string, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "role.update")
	defer span.End()
	span.SetAttributes(attribute.String("role.uuid", roleUUID.String()), attribute.Int64("tenant.id", tenantID))

	var updatedRole *Role
	var capturedActorID int64

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
			return apperror.NewNotFound("role not found")
		}

		// Validate tenant ownership
		if role.TenantID != tenantID {
			return apperror.NewNotFoundWithReason("role not found or access denied")
		}

		// Get actor user with user identities for tenant validation
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, role.Tenant); err != nil {
			return err
		}
		capturedActorID = actorUser.UserID

		// Check if role is a system record
		if role.IsSystem {
			return apperror.NewValidation("system role is not allowed to be updated")
		}

		// If role name is changed, check if duplicate
		if role.Name != name {
			existingRole, err := txRoleRepo.FindByNameAndTenantID(name, role.TenantID)
			if err != nil {
				return err
			}
			if existingRole != nil && existingRole.RoleUUID != roleUUID {
				return apperror.NewConflict(name + " role already exists")
			}
		}

		// Track changed fields
		var changed []string
		if role.Name != name {
			changed = append(changed, "name")
		}
		if role.Description != description {
			changed = append(changed, "description")
		}
		if role.IsDefault != isDefault {
			changed = append(changed, "is_default")
		}
		if role.IsSystem != isSystem {
			changed = append(changed, "is_system")
		}
		if role.Status != status {
			changed = append(changed, "status")
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

		// Emit role.updated integration event inside the transaction
		if s.eventService != nil && len(changed) > 0 {
			s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeRoleUpdated, 1, tenantID,
			).SetActor(&actorUser.UserID).
				SetSubject(&updatedRole.RoleUUID, "role").
				SetChangedFields(changed...))
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update role failed")
		return nil, err
	}

	if err := s.authzInvalidator.InvalidateRoleChange(ctx, updatedRole.RoleID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalidate role sessions failed")
		return nil, apperror.NewInternal("failed to invalidate affected role sessions", err)
	}

	span.SetStatus(codes.Ok, "")
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
		ActorUserID: &capturedActorID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthz,
		EventType:   authevent.AuthEventTypeAuthzAdmin,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Role updated: %s", updatedRole.Name)),
	})
	return toRoleServiceDataResult(updatedRole), nil
}

func (s *roleService) SetStatusByUUID(ctx context.Context, roleUUID uuid.UUID, tenantID int64, status string, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "role.setStatus")
	defer span.End()
	span.SetAttributes(attribute.String("role.uuid", roleUUID.String()), attribute.Int64("tenant.id", tenantID))

	var updatedRole *Role
	var capturedActorID int64

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
			return apperror.NewNotFound("role not found")
		}

		// Validate tenant ownership
		if role.TenantID != tenantID {
			return apperror.NewNotFoundWithReason("role not found or access denied")
		}

		// Get actor user with user identities for tenant validation
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, role.Tenant); err != nil {
			return err
		}
		capturedActorID = actorUser.UserID

		// Check if role is a system record
		if role.IsSystem {
			return apperror.NewValidation("system role is not allowed to be updated")
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "set role status failed")
		return nil, err
	}

	if err := s.authzInvalidator.InvalidateRoleChange(ctx, updatedRole.RoleID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalidate role sessions failed")
		return nil, apperror.NewInternal("failed to invalidate affected role sessions", err)
	}

	span.SetStatus(codes.Ok, "")
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
		ActorUserID: &capturedActorID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthz,
		EventType:   authevent.AuthEventTypeAuthzAdmin,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Role status set to %s: %s", status, updatedRole.Name)),
	})
	return toRoleServiceDataResult(updatedRole), nil
}

func (s *roleService) DeleteByUUID(ctx context.Context, roleUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "role.delete")
	defer span.End()
	span.SetAttributes(attribute.String("role.uuid", roleUUID.String()), attribute.Int64("tenant.id", tenantID))

	// Check role existence
	role, err := s.roleRepo.FindByUUID(roleUUID, "Tenant")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete role failed")
		return nil, err
	}
	if role == nil {
		span.SetStatus(codes.Error, "delete role failed")
		return nil, apperror.NewNotFound("role not found")
	}

	// Validate tenant ownership
	if role.TenantID != tenantID {
		span.SetStatus(codes.Error, "delete role failed")
		return nil, apperror.NewNotFoundWithReason("role not found or access denied")
	}

	// Get actor user with user identities for tenant validation
	actorUser, err := s.userRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
	if err != nil || actorUser == nil {
		span.SetStatus(codes.Error, "delete role failed")
		return nil, apperror.NewNotFoundWithReason("actor user not found")
	}

	// Validate tenant access permissions
	if err := ValidateTenantAccess(actorUser, role.Tenant); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete role failed")
		return nil, err
	}

	// Check if role is a system record
	if role.IsSystem {
		span.SetStatus(codes.Error, "delete role failed")
		return nil, apperror.NewValidation("system role is not allowed to be deleted")
	}

	// Delete role + emit role.deleted atomically; cache invalidation and the
	// audit log run after the commit.
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.roleRepo.WithTx(tx).DeleteByUUID(roleUUID); err != nil {
			return err
		}
		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeRoleDeleted, 1, tenantID,
			).SetActor(&actorUser.UserID).SetSubject(&role.RoleUUID, "role")); emitErr != nil {
				return emitErr
			}
		}
		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete role failed")
		return nil, err
	}

	if err := s.authzInvalidator.InvalidateRoleChange(ctx, role.RoleID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalidate role sessions failed")
		return nil, apperror.NewInternal("failed to invalidate affected role sessions", err)
	}

	span.SetStatus(codes.Ok, "")
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
		ActorUserID: &actorUser.UserID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthz,
		EventType:   authevent.AuthEventTypeAuthzAdmin,
		Severity:    authevent.AuthEventSeverityWarn,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Role deleted: %s", role.Name)),
	})
	return toRoleServiceDataResult(role), nil
}

func (s *roleService) AddRolePermissions(ctx context.Context, roleUUID uuid.UUID, tenantID int64, permissionUUIDs []uuid.UUID, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "role.addPermissions")
	defer span.End()
	span.SetAttributes(attribute.String("role.uuid", roleUUID.String()), attribute.Int64("tenant.id", tenantID))

	var roleWithPermissions *Role
	var capturedActorID int64

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
			return apperror.NewNotFound("role not found")
		}

		// Validate tenant ownership
		if role.TenantID != tenantID {
			return apperror.NewNotFoundWithReason("role not found or access denied")
		}

		// Get actor user with user identities for tenant validation
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, role.Tenant); err != nil {
			return err
		}
		capturedActorID = actorUser.UserID

		// Check if role is a system record
		if role.IsSystem {
			return apperror.NewValidation("system role is not allowed to be updated")
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
			return apperror.NewNotFoundWithReason("one or more permissions not found")
		}

		// Create role-permission associations using the dedicated repository
		for _, permission := range permissions {
			if permission.TenantID != role.TenantID {
				return apperror.NewNotFoundWithReason("one or more permissions not found or access denied")
			}

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
			rolePermission := &RolePermission{
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

		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeRolePermissionsChanged, 1, tenantID,
			).SetActor(&capturedActorID).SetSubject(&roleWithPermissions.RoleUUID, "role").
				SetChangedFields("permissions")); emitErr != nil {
				return emitErr
			}
		}
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "add role permissions failed")
		return nil, err
	}

	if err := s.authzInvalidator.InvalidateRoleChange(ctx, roleWithPermissions.RoleID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalidate role sessions failed")
		return nil, apperror.NewInternal("failed to invalidate affected role sessions", err)
	}

	span.SetStatus(codes.Ok, "")
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
		ActorUserID: &capturedActorID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthz,
		EventType:   authevent.AuthEventTypeAuthzChange,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Permissions added to role: %s", roleWithPermissions.Name)),
	})
	return toRoleServiceDataResult(roleWithPermissions), nil
}

func (s *roleService) RemoveRolePermissions(ctx context.Context, roleUUID uuid.UUID, tenantID int64, permissionUUID uuid.UUID, actorUserUUID uuid.UUID) (*RoleServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "role.removePermissions")
	defer span.End()
	span.SetAttributes(attribute.String("role.uuid", roleUUID.String()), attribute.Int64("tenant.id", tenantID))

	var roleWithPermissions *Role
	var capturedActorID int64

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
			return apperror.NewNotFound("role not found")
		}

		// Validate tenant ownership
		if role.TenantID != tenantID {
			return apperror.NewNotFoundWithReason("role not found or access denied")
		}

		// Get actor user with user identities for tenant validation
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, role.Tenant); err != nil {
			return err
		}
		capturedActorID = actorUser.UserID

		// Check if role is a system record
		if role.IsSystem {
			return apperror.NewValidation("system role is not allowed to be updated")
		}

		// Find permission by UUID
		permission, err := txPermissionRepo.FindByUUID(permissionUUID.String())
		if err != nil {
			return err
		}
		if permission == nil {
			return apperror.NewNotFound("permission not found")
		}
		if permission.TenantID != role.TenantID {
			return apperror.NewNotFoundWithReason("permission not found or access denied")
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

		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeRolePermissionsChanged, 1, tenantID,
			).SetActor(&capturedActorID).SetSubject(&roleWithPermissions.RoleUUID, "role").
				SetChangedFields("permissions")); emitErr != nil {
				return emitErr
			}
		}
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "remove role permissions failed")
		return nil, err
	}

	if err := s.authzInvalidator.InvalidateRoleChange(ctx, roleWithPermissions.RoleID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalidate role sessions failed")
		return nil, apperror.NewInternal("failed to invalidate affected role sessions", err)
	}

	span.SetStatus(codes.Ok, "")
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
		ActorUserID: &capturedActorID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthz,
		EventType:   authevent.AuthEventTypeAuthzChange,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Permission removed from role: %s", roleWithPermissions.Name)),
	})
	return toRoleServiceDataResult(roleWithPermissions), nil
}

// Reponse builder
func toRoleServiceDataResult(role *Role) *RoleServiceDataResult {
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

func ValidateTenantAccess(actor *User, target *Tenant) error {
	if actor == nil {
		return apperror.NewUnauthorized("actor user not found")
	}
	if target == nil {
		return apperror.NewNotFoundWithReason("tenant not found")
	}
	if len(actor.UserIdentities) == 0 {
		return apperror.NewForbidden("actor user has no identities")
	}
	for _, identity := range actor.UserIdentities {
		if identity.TenantID == target.TenantID {
			return nil
		}
		if identity.Tenant != nil && identity.Tenant.IsSystem {
			return nil
		}
	}
	return apperror.NewForbidden("tenant access denied")
}

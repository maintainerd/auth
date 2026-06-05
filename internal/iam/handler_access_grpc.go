package iam

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/auth/internal/shared"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
)

type PermissionGRPCHandler struct {
	authv1.UnimplementedPermissionServiceServer
	tenantService     TenantResolver
	permissionService PermissionService
}

func NewPermissionGRPCHandler(tenantService TenantResolver, permissionService PermissionService) *PermissionGRPCHandler {
	return &PermissionGRPCHandler{tenantService: tenantService, permissionService: permissionService}
}

func (h *PermissionGRPCHandler) ListPermissions(ctx context.Context, req *authv1.ListPermissionsRequest) (*authv1.ListPermissionsResponse, error) {
	scope, err := resolveIAMTenant(ctx, h.tenantService, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	dto := PermissionFilterDTO{
		Name:                 iamOptionalString(req.GetName()),
		Description:          iamOptionalString(req.GetDescription()),
		APIUUID:              iamOptionalString(req.GetApiUuid()),
		RoleUUID:             iamOptionalString(req.GetRoleUuid()),
		ClientUUID:           iamOptionalString(req.GetClientUuid()),
		Status:               iamOptionalString(req.GetStatus()),
		IsDefault:            req.IsDefault,
		IsSystem:             req.IsSystem,
		PaginationRequestDTO: iamPaginationDTO(req.GetPagination()),
	}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.permissionService.Get(ctx, PermissionServiceGetFilter{
		TenantID:    scope.TenantID,
		Name:        dto.Name,
		Description: dto.Description,
		APIUUID:     dto.APIUUID,
		RoleUUID:    dto.RoleUUID,
		ClientUUID:  dto.ClientUUID,
		Status:      dto.Status,
		IsDefault:   dto.IsDefault,
		IsSystem:    dto.IsSystem,
		Page:        dto.Page,
		Limit:       dto.Limit,
		SortBy:      dto.SortBy,
		SortOrder:   dto.SortOrder,
	})
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.Permission, len(result.Data))
	for i := range result.Data {
		rows[i] = permissionProto(&result.Data[i])
	}
	return &authv1.ListPermissionsResponse{Permissions: rows, Page: iamPageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *PermissionGRPCHandler) GetPermission(ctx context.Context, req *authv1.GetPermissionRequest) (*authv1.GetPermissionResponse, error) {
	scope, id, err := resolveIAMTenantAndUUID(ctx, h.tenantService, req.GetTenantUuid(), req.GetPermissionUuid(), "Permission UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.permissionService.GetByUUID(ctx, id, scope.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetPermissionResponse{Permission: permissionProto(result)}, nil
}

func (h *PermissionGRPCHandler) CreatePermission(ctx context.Context, req *authv1.CreatePermissionRequest) (*authv1.CreatePermissionResponse, error) {
	scope, err := resolveIAMTenant(ctx, h.tenantService, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	dto := PermissionCreateRequestDTO{Name: req.GetName(), Description: req.GetDescription(), Status: req.GetStatus(), APIUUID: req.GetApiUuid()}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.permissionService.Create(ctx, scope.TenantID, dto.Name, dto.Description, dto.Status, false, dto.APIUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CreatePermissionResponse{Permission: permissionProto(result)}, nil
}

func (h *PermissionGRPCHandler) UpdatePermission(ctx context.Context, req *authv1.UpdatePermissionRequest) (*authv1.UpdatePermissionResponse, error) {
	scope, id, err := resolveIAMTenantAndUUID(ctx, h.tenantService, req.GetTenantUuid(), req.GetPermissionUuid(), "Permission UUID")
	if err != nil {
		return nil, err
	}
	dto := PermissionUpdateRequestDTO{Name: req.GetName(), Description: req.GetDescription(), Status: req.GetStatus()}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.permissionService.Update(ctx, id, scope.TenantID, dto.Name, dto.Description, dto.Status)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdatePermissionResponse{Permission: permissionProto(result)}, nil
}

func (h *PermissionGRPCHandler) SetPermissionStatus(ctx context.Context, req *authv1.SetPermissionStatusRequest) (*authv1.SetPermissionStatusResponse, error) {
	scope, id, err := resolveIAMTenantAndUUID(ctx, h.tenantService, req.GetTenantUuid(), req.GetPermissionUuid(), "Permission UUID")
	if err != nil {
		return nil, err
	}
	dto := PermissionStatusUpdateDTO{Status: req.GetStatus()}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.permissionService.SetStatus(ctx, id, scope.TenantID, dto.Status)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.SetPermissionStatusResponse{Permission: permissionProto(result)}, nil
}

func (h *PermissionGRPCHandler) DeletePermission(ctx context.Context, req *authv1.DeletePermissionRequest) (*authv1.DeletePermissionResponse, error) {
	scope, id, err := resolveIAMTenantAndUUID(ctx, h.tenantService, req.GetTenantUuid(), req.GetPermissionUuid(), "Permission UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.permissionService.DeleteByUUID(ctx, id, scope.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeletePermissionResponse{Permission: permissionProto(result)}, nil
}

type PolicyGRPCHandler struct {
	authv1.UnimplementedPolicyServiceServer
	tenantService TenantResolver
	policyService PolicyService
}

func NewPolicyGRPCHandler(tenantService TenantResolver, policyService PolicyService) *PolicyGRPCHandler {
	return &PolicyGRPCHandler{tenantService: tenantService, policyService: policyService}
}

func (h *PolicyGRPCHandler) ListPolicies(ctx context.Context, req *authv1.ListPoliciesRequest) (*authv1.ListPoliciesResponse, error) {
	scope, err := resolveIAMTenant(ctx, h.tenantService, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	var serviceID *uuid.UUID
	if req.GetServiceUuid() != "" {
		parsed, err := iamParseUUID(req.GetServiceUuid(), "Service UUID")
		if err != nil {
			return nil, err
		}
		serviceID = &parsed
	}
	dto := PolicyFilterDTO{
		Name:                 iamOptionalString(req.GetName()),
		Description:          iamOptionalString(req.GetDescription()),
		Version:              iamOptionalString(req.GetVersion()),
		Status:               req.GetStatus(),
		IsSystem:             req.IsSystem,
		ServiceID:            serviceID,
		PaginationRequestDTO: iamPaginationDTO(req.GetPagination()),
	}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.policyService.Get(ctx, PolicyServiceGetFilter{
		TenantID:    scope.TenantID,
		Name:        dto.Name,
		Description: dto.Description,
		Version:     dto.Version,
		Status:      dto.Status,
		IsSystem:    dto.IsSystem,
		ServiceID:   dto.ServiceID,
		Page:        dto.Page,
		Limit:       dto.Limit,
		SortBy:      dto.SortBy,
		SortOrder:   dto.SortOrder,
	})
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.Policy, len(result.Data))
	for i := range result.Data {
		rows[i] = policyProto(&result.Data[i])
	}
	return &authv1.ListPoliciesResponse{Policies: rows, Page: iamPageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *PolicyGRPCHandler) GetPolicy(ctx context.Context, req *authv1.GetPolicyRequest) (*authv1.GetPolicyResponse, error) {
	scope, id, err := resolveIAMTenantAndUUID(ctx, h.tenantService, req.GetTenantUuid(), req.GetPolicyUuid(), "Policy UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.policyService.GetByUUID(ctx, id, scope.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetPolicyResponse{Policy: policyProto(result)}, nil
}

func (h *PolicyGRPCHandler) ListPolicyServices(ctx context.Context, req *authv1.ListPolicyServicesRequest) (*authv1.ListPolicyServicesResponse, error) {
	scope, policyUUID, err := resolveIAMTenantAndUUID(ctx, h.tenantService, req.GetTenantUuid(), req.GetPolicyUuid(), "Policy UUID")
	if err != nil {
		return nil, err
	}
	dto := PolicyServicesFilterDTO{
		Name:                 iamOptionalString(req.GetName()),
		DisplayName:          iamOptionalString(req.GetDisplayName()),
		Description:          iamOptionalString(req.GetDescription()),
		PaginationRequestDTO: iamPaginationDTO(req.GetPagination()),
	}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.policyService.GetServicesByPolicyUUID(ctx, policyUUID, scope.TenantID, PolicyServiceServicesFilter{
		Name:        dto.Name,
		DisplayName: dto.DisplayName,
		Description: dto.Description,
		Page:        dto.Page,
		Limit:       dto.Limit,
		SortBy:      dto.SortBy,
		SortOrder:   dto.SortOrder,
	})
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.Service, len(result.Data))
	for i := range result.Data {
		rows[i] = policyServiceProto(&result.Data[i])
	}
	return &authv1.ListPolicyServicesResponse{Services: rows, Page: iamPageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *PolicyGRPCHandler) CreatePolicy(ctx context.Context, req *authv1.CreatePolicyRequest) (*authv1.CreatePolicyResponse, error) {
	scope, err := resolveIAMTenant(ctx, h.tenantService, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	document, err := policyDocumentJSON(req.GetDocument())
	if err != nil {
		return nil, err
	}
	dto := PolicyCreateRequestDTO{Name: req.GetName(), Description: req.Description, Document: document, Version: req.GetVersion(), Status: req.GetStatus()}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.policyService.Create(ctx, scope.TenantID, dto.Name, dto.Description, dto.Document, dto.Version, dto.Status, false)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CreatePolicyResponse{Policy: policyProto(result)}, nil
}

func (h *PolicyGRPCHandler) UpdatePolicy(ctx context.Context, req *authv1.UpdatePolicyRequest) (*authv1.UpdatePolicyResponse, error) {
	scope, id, err := resolveIAMTenantAndUUID(ctx, h.tenantService, req.GetTenantUuid(), req.GetPolicyUuid(), "Policy UUID")
	if err != nil {
		return nil, err
	}
	document, err := policyDocumentJSON(req.GetDocument())
	if err != nil {
		return nil, err
	}
	dto := PolicyUpdateRequestDTO{Name: req.GetName(), Description: req.Description, Document: document, Version: req.GetVersion(), Status: req.GetStatus()}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.policyService.Update(ctx, id, scope.TenantID, dto.Name, dto.Description, dto.Document, dto.Version, dto.Status)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdatePolicyResponse{Policy: policyProto(result)}, nil
}

func (h *PolicyGRPCHandler) SetPolicyStatus(ctx context.Context, req *authv1.SetPolicyStatusRequest) (*authv1.SetPolicyStatusResponse, error) {
	scope, id, err := resolveIAMTenantAndUUID(ctx, h.tenantService, req.GetTenantUuid(), req.GetPolicyUuid(), "Policy UUID")
	if err != nil {
		return nil, err
	}
	dto := PolicyStatusUpdateDTO{Status: req.GetStatus()}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.policyService.SetStatusByUUID(ctx, id, scope.TenantID, dto.Status)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.SetPolicyStatusResponse{Policy: policyProto(result)}, nil
}

func (h *PolicyGRPCHandler) DeletePolicy(ctx context.Context, req *authv1.DeletePolicyRequest) (*authv1.DeletePolicyResponse, error) {
	scope, id, err := resolveIAMTenantAndUUID(ctx, h.tenantService, req.GetTenantUuid(), req.GetPolicyUuid(), "Policy UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.policyService.DeleteByUUID(ctx, id, scope.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeletePolicyResponse{Policy: policyProto(result)}, nil
}

type RoleGRPCHandler struct {
	authv1.UnimplementedRoleServiceServer
	tenantService TenantResolver
	roleService   RoleService
}

func NewRoleGRPCHandler(tenantService TenantResolver, roleService RoleService) *RoleGRPCHandler {
	return &RoleGRPCHandler{tenantService: tenantService, roleService: roleService}
}

func (h *RoleGRPCHandler) ListRoles(ctx context.Context, req *authv1.ListRolesRequest) (*authv1.ListRolesResponse, error) {
	scope, err := resolveIAMTenant(ctx, h.tenantService, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	dto := RoleFilterDTO{
		Name:                 iamOptionalString(req.GetName()),
		Description:          iamOptionalString(req.GetDescription()),
		IsDefault:            req.IsDefault,
		IsSystem:             req.IsSystem,
		Status:               iamOptionalString(req.GetStatus()),
		PaginationRequestDTO: iamPaginationDTO(req.GetPagination()),
	}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.roleService.Get(ctx, RoleServiceGetFilter{
		Name:        dto.Name,
		Description: dto.Description,
		IsDefault:   dto.IsDefault,
		IsSystem:    dto.IsSystem,
		Status:      dto.Status,
		TenantID:    scope.TenantID,
		Page:        dto.Page,
		Limit:       dto.Limit,
		SortBy:      dto.SortBy,
		SortOrder:   dto.SortOrder,
	})
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.Role, len(result.Data))
	for i := range result.Data {
		rows[i] = roleProto(&result.Data[i])
	}
	return &authv1.ListRolesResponse{Roles: rows, Page: iamPageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *RoleGRPCHandler) GetRole(ctx context.Context, req *authv1.GetRoleRequest) (*authv1.GetRoleResponse, error) {
	scope, id, err := resolveIAMTenantAndUUID(ctx, h.tenantService, req.GetTenantUuid(), req.GetRoleUuid(), "Role UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.roleService.GetByUUID(ctx, id, scope.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetRoleResponse{Role: roleProto(result)}, nil
}

func (h *RoleGRPCHandler) CreateRole(ctx context.Context, req *authv1.CreateRoleRequest) (*authv1.CreateRoleResponse, error) {
	scope, actor, err := resolveIAMTenantAndActor(ctx, h.tenantService, req.GetTenantUuid(), req.GetActorUserUuid())
	if err != nil {
		return nil, err
	}
	dto := RoleCreateOrUpdateRequestDTO{Name: req.GetName(), Description: req.GetDescription(), Status: req.GetStatus()}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.roleService.Create(ctx, dto.Name, dto.Description, false, false, dto.Status, scope.TenantUUID.String(), actor)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CreateRoleResponse{Role: roleProto(result)}, nil
}

func (h *RoleGRPCHandler) UpdateRole(ctx context.Context, req *authv1.UpdateRoleRequest) (*authv1.UpdateRoleResponse, error) {
	scope, roleUUID, actor, err := resolveIAMTenantRoleActor(ctx, h.tenantService, req.GetTenantUuid(), req.GetRoleUuid(), req.GetActorUserUuid())
	if err != nil {
		return nil, err
	}
	dto := RoleCreateOrUpdateRequestDTO{Name: req.GetName(), Description: req.GetDescription(), Status: req.GetStatus()}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.roleService.Update(ctx, roleUUID, scope.TenantID, dto.Name, dto.Description, false, false, dto.Status, actor)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdateRoleResponse{Role: roleProto(result)}, nil
}

func (h *RoleGRPCHandler) SetRoleStatus(ctx context.Context, req *authv1.SetRoleStatusRequest) (*authv1.SetRoleStatusResponse, error) {
	scope, roleUUID, actor, err := resolveIAMTenantRoleActor(ctx, h.tenantService, req.GetTenantUuid(), req.GetRoleUuid(), req.GetActorUserUuid())
	if err != nil {
		return nil, err
	}
	if req.GetStatus() != shared.StatusActive && req.GetStatus() != shared.StatusInactive {
		return nil, apperror.ToGRPCError(apperror.NewValidation("Status must be 'active' or 'inactive'"))
	}
	result, err := h.roleService.SetStatusByUUID(ctx, roleUUID, scope.TenantID, req.GetStatus(), actor)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.SetRoleStatusResponse{Role: roleProto(result)}, nil
}

func (h *RoleGRPCHandler) DeleteRole(ctx context.Context, req *authv1.DeleteRoleRequest) (*authv1.DeleteRoleResponse, error) {
	scope, roleUUID, actor, err := resolveIAMTenantRoleActor(ctx, h.tenantService, req.GetTenantUuid(), req.GetRoleUuid(), req.GetActorUserUuid())
	if err != nil {
		return nil, err
	}
	result, err := h.roleService.DeleteByUUID(ctx, roleUUID, scope.TenantID, actor)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeleteRoleResponse{Role: roleProto(result)}, nil
}

func (h *RoleGRPCHandler) ListRolePermissions(ctx context.Context, req *authv1.ListRolePermissionsRequest) (*authv1.ListRolePermissionsResponse, error) {
	scope, roleUUID, err := resolveIAMTenantAndUUID(ctx, h.tenantService, req.GetTenantUuid(), req.GetRoleUuid(), "Role UUID")
	if err != nil {
		return nil, err
	}
	dto := PaginationRequestDTO{Page: iamPaginationDTO(req.GetPagination()).Page, Limit: iamPaginationDTO(req.GetPagination()).Limit, SortBy: iamPaginationDTO(req.GetPagination()).SortBy, SortOrder: iamPaginationDTO(req.GetPagination()).SortOrder}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.roleService.GetRolePermissions(ctx, RoleServiceGetPermissionsFilter{
		RoleUUID:  roleUUID,
		Status:    iamOptionalString(req.GetStatus()),
		TenantID:  scope.TenantID,
		Page:      dto.Page,
		Limit:     dto.Limit,
		SortBy:    dto.SortBy,
		SortOrder: dto.SortOrder,
	})
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.Permission, len(result.Data))
	for i := range result.Data {
		rows[i] = permissionProto(&result.Data[i])
	}
	return &authv1.ListRolePermissionsResponse{Permissions: rows, Page: iamPageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *RoleGRPCHandler) AddRolePermissions(ctx context.Context, req *authv1.AddRolePermissionsRequest) (*authv1.AddRolePermissionsResponse, error) {
	scope, roleUUID, actor, err := resolveIAMTenantRoleActor(ctx, h.tenantService, req.GetTenantUuid(), req.GetRoleUuid(), req.GetActorUserUuid())
	if err != nil {
		return nil, err
	}
	permissionUUIDs, err := parseIAMUUIDs(req.GetPermissionUuids(), "Permission UUID")
	if err != nil {
		return nil, err
	}
	dto := RoleAddPermissionsRequestDTO{Permissions: permissionUUIDs}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.roleService.AddRolePermissions(ctx, roleUUID, scope.TenantID, permissionUUIDs, actor)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.AddRolePermissionsResponse{Role: roleProto(result)}, nil
}

func (h *RoleGRPCHandler) RemoveRolePermission(ctx context.Context, req *authv1.RemoveRolePermissionRequest) (*authv1.RemoveRolePermissionResponse, error) {
	scope, roleUUID, actor, err := resolveIAMTenantRoleActor(ctx, h.tenantService, req.GetTenantUuid(), req.GetRoleUuid(), req.GetActorUserUuid())
	if err != nil {
		return nil, err
	}
	permissionUUID, err := iamParseUUID(req.GetPermissionUuid(), "Permission UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.roleService.RemoveRolePermissions(ctx, roleUUID, scope.TenantID, permissionUUID, actor)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.RemoveRolePermissionResponse{Role: roleProto(result)}, nil
}

type AuthorizationGRPCHandler struct {
	authv1.UnimplementedAuthorizationServiceServer
	authorizationService ServiceAuthorizationService
}

func NewAuthorizationGRPCHandler(authorizationService ServiceAuthorizationService) *AuthorizationGRPCHandler {
	return &AuthorizationGRPCHandler{authorizationService: authorizationService}
}

func (h *AuthorizationGRPCHandler) Authorize(ctx context.Context, req *authv1.AuthorizeRequest) (*authv1.AuthorizeResponse, error) {
	decision := h.authorizationService.Authorize(ctx, AuthzRequest{Principal: req.GetPrincipal(), Action: req.GetAction(), Resource: req.GetResource()})
	return &authv1.AuthorizeResponse{Allowed: decision.Allowed, Reason: decision.Reason}, nil
}

func resolveIAMTenant(ctx context.Context, tenantService TenantResolver, tenantUUID string) (*tenantScope, error) {
	parsed, err := iamParseUUID(tenantUUID, "Tenant UUID")
	if err != nil {
		return nil, err
	}
	result, err := tenantService.GetByUUID(ctx, parsed)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &tenantScope{TenantID: result.TenantID, TenantUUID: result.TenantUUID}, nil
}

func resolveIAMTenantAndUUID(ctx context.Context, tenantService TenantResolver, tenantUUID string, value string, label string) (*tenantScope, uuid.UUID, error) {
	scope, err := resolveIAMTenant(ctx, tenantService, tenantUUID)
	if err != nil {
		return nil, uuid.Nil, err
	}
	parsed, err := iamParseUUID(value, label)
	if err != nil {
		return nil, uuid.Nil, err
	}
	return scope, parsed, nil
}

func resolveIAMTenantAndActor(ctx context.Context, tenantService TenantResolver, tenantUUID string, actorValue string) (*tenantScope, uuid.UUID, error) {
	scope, err := resolveIAMTenant(ctx, tenantService, tenantUUID)
	if err != nil {
		return nil, uuid.Nil, err
	}
	actor, err := iamParseUUID(actorValue, "Actor user UUID")
	if err != nil {
		return nil, uuid.Nil, err
	}
	return scope, actor, nil
}

func resolveIAMTenantRoleActor(ctx context.Context, tenantService TenantResolver, tenantUUID string, roleValue string, actorValue string) (*tenantScope, uuid.UUID, uuid.UUID, error) {
	scope, roleUUID, err := resolveIAMTenantAndUUID(ctx, tenantService, tenantUUID, roleValue, "Role UUID")
	if err != nil {
		return nil, uuid.Nil, uuid.Nil, err
	}
	actor, err := iamParseUUID(actorValue, "Actor user UUID")
	if err != nil {
		return nil, uuid.Nil, uuid.Nil, err
	}
	return scope, roleUUID, actor, nil
}

type tenantScope struct {
	TenantID   int64
	TenantUUID uuid.UUID
}

func parseIAMUUIDs(values []string, label string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, len(values))
	for i, value := range values {
		parsed, err := iamParseUUID(value, label)
		if err != nil {
			return nil, err
		}
		result[i] = parsed
	}
	return result, nil
}

func permissionProto(result *PermissionServiceDataResult) *authv1.Permission {
	if result == nil {
		return nil
	}
	return &authv1.Permission{
		PermissionUuid: result.PermissionUUID.String(),
		Name:           result.Name,
		Description:    result.Description,
		Api:            apiProto(result.API),
		Status:         result.Status,
		IsDefault:      result.IsDefault,
		IsSystem:       result.IsSystem,
		CreatedAt:      timestamppb.New(result.CreatedAt),
		UpdatedAt:      timestamppb.New(result.UpdatedAt),
	}
}

func policyProto(result *PolicyServiceDataResult) *authv1.Policy {
	if result == nil {
		return nil
	}
	return &authv1.Policy{
		PolicyUuid:  result.PolicyUUID.String(),
		Name:        result.Name,
		Description: result.Description,
		Document:    policyDocumentProto(result.Document),
		Version:     result.Version,
		Status:      result.Status,
		IsSystem:    result.IsSystem,
		CreatedAt:   timestamppb.New(result.CreatedAt),
		UpdatedAt:   timestamppb.New(result.UpdatedAt),
	}
}

func policyServiceProto(result *PolicyServiceServiceDataResult) *authv1.Service {
	if result == nil {
		return nil
	}
	return &authv1.Service{
		ServiceUuid: result.ServiceUUID.String(),
		Name:        result.Name,
		DisplayName: result.DisplayName,
		Description: result.Description,
		Version:     result.Version,
		Status:      result.Status,
		IsSystem:    result.IsSystem,
		ApiCount:    result.APICount,
		PolicyCount: result.PolicyCount,
		CreatedAt:   timestamppb.New(result.CreatedAt),
		UpdatedAt:   timestamppb.New(result.UpdatedAt),
	}
}

func roleProto(result *RoleServiceDataResult) *authv1.Role {
	if result == nil {
		return nil
	}
	permissions := make([]*authv1.Permission, 0)
	if result.Permissions != nil {
		permissions = make([]*authv1.Permission, len(*result.Permissions))
		for i := range *result.Permissions {
			permissions[i] = permissionProto(&(*result.Permissions)[i])
		}
	}
	return &authv1.Role{
		RoleUuid:    result.RoleUUID.String(),
		Name:        result.Name,
		Description: result.Description,
		Permissions: permissions,
		IsDefault:   result.IsDefault,
		IsSystem:    result.IsSystem,
		Status:      result.Status,
		CreatedAt:   timestamppb.New(result.CreatedAt),
		UpdatedAt:   timestamppb.New(result.UpdatedAt),
	}
}

func policyDocumentJSON(document *structpb.Struct) (datatypes.JSON, error) {
	if document == nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation("Policy document is required"))
	}
	payload, _ := json.Marshal(document.AsMap())
	return datatypes.JSON(payload), nil
}

func policyDocumentProto(document datatypes.JSON) *structpb.Struct {
	if len(document) == 0 {
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	var raw map[string]any
	if err := json.Unmarshal(document, &raw); err != nil {
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	result, _ := structpb.NewStruct(raw)
	return result
}

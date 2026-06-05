package iam

import (
	"context"

	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/auth/internal/shared"
)

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

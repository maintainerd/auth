package iam

import (
	"context"

	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
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

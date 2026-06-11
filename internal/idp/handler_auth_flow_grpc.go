package idp

import (
	"context"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthFlowGRPCHandler struct {
	authv1.UnimplementedSignupFlowServiceServer
	tenantResolver  TenantResolver
	authFlowService AuthFlowService
}

func NewAuthFlowGRPCHandler(tenantResolver TenantResolver, authFlowService AuthFlowService) *AuthFlowGRPCHandler {
	return &AuthFlowGRPCHandler{tenantResolver: tenantResolver, authFlowService: authFlowService}
}

func (h *AuthFlowGRPCHandler) ListSignupFlows(ctx context.Context, req *authv1.ListSignupFlowsRequest) (*authv1.ListSignupFlowsResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	var clientUUID *uuid.UUID
	if req.GetClientUuid() != "" {
		parsed, err := grpcUUID(req.GetClientUuid(), "Client UUID")
		if err != nil {
			return nil, err
		}
		clientUUID = &parsed
	}
	dto := grpcPagination(req.GetPagination())
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.authFlowService.GetAll(ctx, tenant.TenantID, grpcStr(req.GetName()), grpcStr(req.GetIdentifier()), req.GetStatus(), clientUUID, dto.Page, dto.Limit, dto.SortBy, dto.SortOrder)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.SignupFlow, len(result.Data))
	for i := range result.Data {
		rows[i] = toAuthFlowProto(&result.Data[i])
	}
	return &authv1.ListSignupFlowsResponse{SignupFlows: rows, Page: grpcPageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *AuthFlowGRPCHandler) GetSignupFlow(ctx context.Context, req *authv1.GetSignupFlowRequest) (*authv1.GetSignupFlowResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	sfUUID, err := grpcUUID(req.GetSignupFlowUuid(), "AuthFlow UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.authFlowService.GetByUUID(ctx, sfUUID, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetSignupFlowResponse{SignupFlow: toAuthFlowProto(result)}, nil
}

func (h *AuthFlowGRPCHandler) CreateSignupFlow(ctx context.Context, req *authv1.CreateSignupFlowRequest) (*authv1.CreateSignupFlowResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	clientUUID, err := grpcUUID(req.GetClientUuid(), "Client UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.authFlowService.Create(ctx, tenant.TenantID, req.GetName(), req.GetDescription(), req.GetStatus(), clientUUID, nil, nil, nil)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CreateSignupFlowResponse{SignupFlow: toAuthFlowProto(result)}, nil
}

func (h *AuthFlowGRPCHandler) UpdateSignupFlow(ctx context.Context, req *authv1.UpdateSignupFlowRequest) (*authv1.UpdateSignupFlowResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	sfUUID, err := grpcUUID(req.GetSignupFlowUuid(), "AuthFlow UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.authFlowService.Update(ctx, sfUUID, tenant.TenantID, req.GetName(), req.GetDescription(), req.GetStatus(), nil, nil, nil)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdateSignupFlowResponse{SignupFlow: toAuthFlowProto(result)}, nil
}

func (h *AuthFlowGRPCHandler) SetSignupFlowStatus(ctx context.Context, req *authv1.SetSignupFlowStatusRequest) (*authv1.SetSignupFlowStatusResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	sfUUID, err := grpcUUID(req.GetSignupFlowUuid(), "AuthFlow UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.authFlowService.UpdateStatus(ctx, sfUUID, tenant.TenantID, req.GetStatus())
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.SetSignupFlowStatusResponse{SignupFlow: toAuthFlowProto(result)}, nil
}

func (h *AuthFlowGRPCHandler) DeleteSignupFlow(ctx context.Context, req *authv1.DeleteSignupFlowRequest) (*authv1.DeleteSignupFlowResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	sfUUID, err := grpcUUID(req.GetSignupFlowUuid(), "AuthFlow UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.authFlowService.Delete(ctx, sfUUID, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeleteSignupFlowResponse{SignupFlow: toAuthFlowProto(result)}, nil
}

func (h *AuthFlowGRPCHandler) AssignSignupFlowRoles(ctx context.Context, req *authv1.AssignSignupFlowRolesRequest) (*authv1.AssignSignupFlowRolesResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	sfUUID, err := grpcUUID(req.GetSignupFlowUuid(), "AuthFlow UUID")
	if err != nil {
		return nil, err
	}
	roleUUIDs := make([]uuid.UUID, len(req.GetRoleUuids()))
	for i, r := range req.GetRoleUuids() {
		parsed, err := grpcUUID(r, "Role UUID")
		if err != nil {
			return nil, err
		}
		roleUUIDs[i] = parsed
	}
	roles, err := h.authFlowService.AssignRoles(ctx, sfUUID, tenant.TenantID, roleUUIDs)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.SignupFlowRole, len(roles))
	for i := range roles {
		rows[i] = toAuthFlowRoleProto(&roles[i])
	}
	return &authv1.AssignSignupFlowRolesResponse{Roles: rows}, nil
}

func (h *AuthFlowGRPCHandler) ListSignupFlowRoles(ctx context.Context, req *authv1.ListSignupFlowRolesRequest) (*authv1.ListSignupFlowRolesResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	sfUUID, err := grpcUUID(req.GetSignupFlowUuid(), "AuthFlow UUID")
	if err != nil {
		return nil, err
	}
	dto := grpcPagination(req.GetPagination())
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.authFlowService.GetRoles(ctx, sfUUID, tenant.TenantID, dto.Page, dto.Limit)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.SignupFlowRole, len(result.Data))
	for i := range result.Data {
		rows[i] = toAuthFlowRoleProto(&result.Data[i])
	}
	return &authv1.ListSignupFlowRolesResponse{Roles: rows, Page: grpcPageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *AuthFlowGRPCHandler) RemoveSignupFlowRole(ctx context.Context, req *authv1.RemoveSignupFlowRoleRequest) (*authv1.RemoveSignupFlowRoleResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	sfUUID, err := grpcUUID(req.GetSignupFlowUuid(), "AuthFlow UUID")
	if err != nil {
		return nil, err
	}
	roleUUID, err := grpcUUID(req.GetRoleUuid(), "Role UUID")
	if err != nil {
		return nil, err
	}
	if err := h.authFlowService.RemoveRole(ctx, sfUUID, tenant.TenantID, roleUUID); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.RemoveSignupFlowRoleResponse{Removed: true}, nil
}

func (h *AuthFlowGRPCHandler) resolveTenant(ctx context.Context, tenantUUID string) (*TenantServiceDataResult, error) {
	parsed, err := grpcUUID(tenantUUID, "Tenant UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.tenantResolver.GetByUUID(ctx, parsed)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return result, nil
}

func toAuthFlowProto(result *AuthFlowServiceDataResult) *authv1.SignupFlow {
	if result == nil {
		return nil
	}
	return &authv1.SignupFlow{
		SignupFlowUuid: result.AuthFlowUUID.String(),
		Name:         result.Name,
		Description:  result.Description,
		Identifier:   result.Identifier,
		Status:       result.Status,
		ClientUuid:   result.ClientUUID.String(),
		CreatedAt:    timestamppb.New(result.CreatedAt),
		UpdatedAt:    timestamppb.New(result.UpdatedAt),
	}
}

func toAuthFlowRoleProto(result *AuthFlowRoleServiceDataResult) *authv1.SignupFlowRole {
	if result == nil {
		return nil
	}
	return &authv1.SignupFlowRole{
		SignupFlowRoleUuid: result.AuthFlowRoleUUID.String(),
		RoleUuid:         result.RoleUUID.String(),
		RoleName:         result.RoleName,
		RoleDescription:  result.RoleDescription,
		RoleIsDefault:    result.RoleIsDefault,
		RoleIsSystem:     result.RoleIsSystem,
		RoleStatus:       result.RoleStatus,
		CreatedAt:        timestamppb.New(result.CreatedAt),
		UpdatedAt:        timestamppb.New(result.UpdatedAt),
	}
}

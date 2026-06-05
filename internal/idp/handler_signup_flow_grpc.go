package idp

import (
	"context"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SignupFlowGRPCHandler struct {
	authv1.UnimplementedSignupFlowServiceServer
	tenantResolver   TenantResolver
	signupFlowService SignupFlowService
}

func NewSignupFlowGRPCHandler(tenantResolver TenantResolver, signupFlowService SignupFlowService) *SignupFlowGRPCHandler {
	return &SignupFlowGRPCHandler{tenantResolver: tenantResolver, signupFlowService: signupFlowService}
}

func (h *SignupFlowGRPCHandler) ListSignupFlows(ctx context.Context, req *authv1.ListSignupFlowsRequest) (*authv1.ListSignupFlowsResponse, error) {
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
	result, err := h.signupFlowService.GetAll(ctx, tenant.TenantID, grpcStr(req.GetName()), grpcStr(req.GetIdentifier()), req.GetStatus(), clientUUID, dto.Page, dto.Limit, dto.SortBy, dto.SortOrder)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.SignupFlow, len(result.Data))
	for i := range result.Data {
		rows[i] = toSignupFlowProto(&result.Data[i])
	}
	return &authv1.ListSignupFlowsResponse{SignupFlows: rows, Page: grpcPageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *SignupFlowGRPCHandler) GetSignupFlow(ctx context.Context, req *authv1.GetSignupFlowRequest) (*authv1.GetSignupFlowResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	sfUUID, err := grpcUUID(req.GetSignupFlowUuid(), "SignupFlow UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.signupFlowService.GetByUUID(ctx, sfUUID, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetSignupFlowResponse{SignupFlow: toSignupFlowProto(result)}, nil
}

func (h *SignupFlowGRPCHandler) CreateSignupFlow(ctx context.Context, req *authv1.CreateSignupFlowRequest) (*authv1.CreateSignupFlowResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	clientUUID, err := grpcUUID(req.GetClientUuid(), "Client UUID")
	if err != nil {
		return nil, err
	}
	config := structpbToMap(req.GetConfig())
	result, err := h.signupFlowService.Create(ctx, tenant.TenantID, req.GetName(), req.GetDescription(), config, req.GetStatus(), clientUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CreateSignupFlowResponse{SignupFlow: toSignupFlowProto(result)}, nil
}

func (h *SignupFlowGRPCHandler) UpdateSignupFlow(ctx context.Context, req *authv1.UpdateSignupFlowRequest) (*authv1.UpdateSignupFlowResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	sfUUID, err := grpcUUID(req.GetSignupFlowUuid(), "SignupFlow UUID")
	if err != nil {
		return nil, err
	}
	config := structpbToMap(req.GetConfig())
	result, err := h.signupFlowService.Update(ctx, sfUUID, tenant.TenantID, req.GetName(), req.GetDescription(), config, req.GetStatus())
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdateSignupFlowResponse{SignupFlow: toSignupFlowProto(result)}, nil
}

func (h *SignupFlowGRPCHandler) SetSignupFlowStatus(ctx context.Context, req *authv1.SetSignupFlowStatusRequest) (*authv1.SetSignupFlowStatusResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	sfUUID, err := grpcUUID(req.GetSignupFlowUuid(), "SignupFlow UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.signupFlowService.UpdateStatus(ctx, sfUUID, tenant.TenantID, req.GetStatus())
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.SetSignupFlowStatusResponse{SignupFlow: toSignupFlowProto(result)}, nil
}

func (h *SignupFlowGRPCHandler) DeleteSignupFlow(ctx context.Context, req *authv1.DeleteSignupFlowRequest) (*authv1.DeleteSignupFlowResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	sfUUID, err := grpcUUID(req.GetSignupFlowUuid(), "SignupFlow UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.signupFlowService.Delete(ctx, sfUUID, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeleteSignupFlowResponse{SignupFlow: toSignupFlowProto(result)}, nil
}

func (h *SignupFlowGRPCHandler) AssignSignupFlowRoles(ctx context.Context, req *authv1.AssignSignupFlowRolesRequest) (*authv1.AssignSignupFlowRolesResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	sfUUID, err := grpcUUID(req.GetSignupFlowUuid(), "SignupFlow UUID")
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
	roles, err := h.signupFlowService.AssignRoles(ctx, sfUUID, tenant.TenantID, roleUUIDs)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.SignupFlowRole, len(roles))
	for i := range roles {
		rows[i] = toSignupFlowRoleProto(&roles[i])
	}
	return &authv1.AssignSignupFlowRolesResponse{Roles: rows}, nil
}

func (h *SignupFlowGRPCHandler) ListSignupFlowRoles(ctx context.Context, req *authv1.ListSignupFlowRolesRequest) (*authv1.ListSignupFlowRolesResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	sfUUID, err := grpcUUID(req.GetSignupFlowUuid(), "SignupFlow UUID")
	if err != nil {
		return nil, err
	}
	dto := grpcPagination(req.GetPagination())
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.signupFlowService.GetRoles(ctx, sfUUID, tenant.TenantID, dto.Page, dto.Limit)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.SignupFlowRole, len(result.Data))
	for i := range result.Data {
		rows[i] = toSignupFlowRoleProto(&result.Data[i])
	}
	return &authv1.ListSignupFlowRolesResponse{Roles: rows, Page: grpcPageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *SignupFlowGRPCHandler) RemoveSignupFlowRole(ctx context.Context, req *authv1.RemoveSignupFlowRoleRequest) (*authv1.RemoveSignupFlowRoleResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	sfUUID, err := grpcUUID(req.GetSignupFlowUuid(), "SignupFlow UUID")
	if err != nil {
		return nil, err
	}
	roleUUID, err := grpcUUID(req.GetRoleUuid(), "Role UUID")
	if err != nil {
		return nil, err
	}
	if err := h.signupFlowService.RemoveRole(ctx, sfUUID, tenant.TenantID, roleUUID); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.RemoveSignupFlowRoleResponse{Removed: true}, nil
}

func (h *SignupFlowGRPCHandler) resolveTenant(ctx context.Context, tenantUUID string) (*TenantServiceDataResult, error) {
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

func toSignupFlowProto(result *SignupFlowServiceDataResult) *authv1.SignupFlow {
	if result == nil {
		return nil
	}
	return &authv1.SignupFlow{
		SignupFlowUuid: result.SignupFlowUUID.String(),
		Name:           result.Name,
		Description:    result.Description,
		Identifier:     result.Identifier,
		Config:         mapToStructpb(result.Config),
		Status:         result.Status,
		ClientUuid:     result.ClientUUID.String(),
		CreatedAt:      timestamppb.New(result.CreatedAt),
		UpdatedAt:      timestamppb.New(result.UpdatedAt),
	}
}

func toSignupFlowRoleProto(result *SignupFlowRoleServiceDataResult) *authv1.SignupFlowRole {
	if result == nil {
		return nil
	}
	return &authv1.SignupFlowRole{
		SignupFlowRoleUuid: result.SignupFlowRoleUUID.String(),
		RoleUuid:           result.RoleUUID.String(),
		RoleName:           result.RoleName,
		RoleDescription:    result.RoleDescription,
		RoleIsDefault:      result.RoleIsDefault,
		RoleIsSystem:       result.RoleIsSystem,
		RoleStatus:         result.RoleStatus,
		CreatedAt:          timestamppb.New(result.CreatedAt),
		UpdatedAt:          timestamppb.New(result.UpdatedAt),
	}
}

func structpbToMap(s *structpb.Struct) map[string]any {
	if s == nil {
		return nil
	}
	return s.AsMap()
}

func mapToStructpb(m map[string]any) *structpb.Struct {
	if m == nil {
		return nil
	}
	result, _ := structpb.NewStruct(m)
	return result
}

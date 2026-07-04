package idp

import (
	"context"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
)

type RegistrationFlowGRPCHandler struct {
	authv1.UnimplementedRegistrationFlowServiceServer
	tenantResolver          TenantResolver
	registrationFlowService RegistrationFlowService
}

func NewRegistrationFlowGRPCHandler(tenantResolver TenantResolver, registrationFlowService RegistrationFlowService) *RegistrationFlowGRPCHandler {
	return &RegistrationFlowGRPCHandler{tenantResolver: tenantResolver, registrationFlowService: registrationFlowService}
}

func (h *RegistrationFlowGRPCHandler) ListRegistrationFlows(ctx context.Context, req *authv1.ListRegistrationFlowsRequest) (*authv1.ListRegistrationFlowsResponse, error) {
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
	result, err := h.registrationFlowService.GetAll(ctx, tenant.TenantID, grpcStr(req.GetName()), grpcStr(req.GetIdentifier()), req.GetStatus(), clientUUID, dto.Page, dto.Limit, dto.SortBy, dto.SortOrder)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.RegistrationFlow, len(result.Data))
	for i := range result.Data {
		rows[i] = toRegistrationFlowProto(&result.Data[i])
	}
	return &authv1.ListRegistrationFlowsResponse{RegistrationFlows: rows, Page: grpcPageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *RegistrationFlowGRPCHandler) GetRegistrationFlow(ctx context.Context, req *authv1.GetRegistrationFlowRequest) (*authv1.GetRegistrationFlowResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	sfUUID, err := grpcUUID(req.GetRegistrationFlowUuid(), "RegistrationFlow UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.registrationFlowService.GetByUUID(ctx, sfUUID, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetRegistrationFlowResponse{RegistrationFlow: toRegistrationFlowProto(result)}, nil
}

func (h *RegistrationFlowGRPCHandler) CreateRegistrationFlow(ctx context.Context, req *authv1.CreateRegistrationFlowRequest) (*authv1.CreateRegistrationFlowResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	clientUUID, err := grpcUUID(req.GetClientUuid(), "Client UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.registrationFlowService.Create(ctx, tenant.TenantID, req.GetName(), req.GetDescription(), req.GetStatus(), clientUUID, nil, nil, false, datatypes.JSON([]byte("[]")))
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CreateRegistrationFlowResponse{RegistrationFlow: toRegistrationFlowProto(result)}, nil
}

func (h *RegistrationFlowGRPCHandler) UpdateRegistrationFlow(ctx context.Context, req *authv1.UpdateRegistrationFlowRequest) (*authv1.UpdateRegistrationFlowResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	sfUUID, err := grpcUUID(req.GetRegistrationFlowUuid(), "RegistrationFlow UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.registrationFlowService.Update(ctx, sfUUID, tenant.TenantID, req.GetName(), req.GetDescription(), req.GetStatus(), nil, false, datatypes.JSON([]byte("[]")))
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdateRegistrationFlowResponse{RegistrationFlow: toRegistrationFlowProto(result)}, nil
}

func (h *RegistrationFlowGRPCHandler) SetRegistrationFlowStatus(ctx context.Context, req *authv1.SetRegistrationFlowStatusRequest) (*authv1.SetRegistrationFlowStatusResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	sfUUID, err := grpcUUID(req.GetRegistrationFlowUuid(), "RegistrationFlow UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.registrationFlowService.UpdateStatus(ctx, sfUUID, tenant.TenantID, req.GetStatus())
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.SetRegistrationFlowStatusResponse{RegistrationFlow: toRegistrationFlowProto(result)}, nil
}

func (h *RegistrationFlowGRPCHandler) DeleteRegistrationFlow(ctx context.Context, req *authv1.DeleteRegistrationFlowRequest) (*authv1.DeleteRegistrationFlowResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	sfUUID, err := grpcUUID(req.GetRegistrationFlowUuid(), "RegistrationFlow UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.registrationFlowService.Delete(ctx, sfUUID, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeleteRegistrationFlowResponse{RegistrationFlow: toRegistrationFlowProto(result)}, nil
}

func (h *RegistrationFlowGRPCHandler) AssignRegistrationFlowRoles(ctx context.Context, req *authv1.AssignRegistrationFlowRolesRequest) (*authv1.AssignRegistrationFlowRolesResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	sfUUID, err := grpcUUID(req.GetRegistrationFlowUuid(), "RegistrationFlow UUID")
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
	roles, err := h.registrationFlowService.AssignRoles(ctx, sfUUID, tenant.TenantID, roleUUIDs)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.RegistrationFlowRole, len(roles))
	for i := range roles {
		rows[i] = toRegistrationFlowRoleProto(&roles[i])
	}
	return &authv1.AssignRegistrationFlowRolesResponse{Roles: rows}, nil
}

func (h *RegistrationFlowGRPCHandler) ListRegistrationFlowRoles(ctx context.Context, req *authv1.ListRegistrationFlowRolesRequest) (*authv1.ListRegistrationFlowRolesResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	sfUUID, err := grpcUUID(req.GetRegistrationFlowUuid(), "RegistrationFlow UUID")
	if err != nil {
		return nil, err
	}
	dto := grpcPagination(req.GetPagination())
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.registrationFlowService.GetRoles(ctx, sfUUID, tenant.TenantID, dto.Page, dto.Limit)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.RegistrationFlowRole, len(result.Data))
	for i := range result.Data {
		rows[i] = toRegistrationFlowRoleProto(&result.Data[i])
	}
	return &authv1.ListRegistrationFlowRolesResponse{Roles: rows, Page: grpcPageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *RegistrationFlowGRPCHandler) RemoveRegistrationFlowRole(ctx context.Context, req *authv1.RemoveRegistrationFlowRoleRequest) (*authv1.RemoveRegistrationFlowRoleResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	sfUUID, err := grpcUUID(req.GetRegistrationFlowUuid(), "RegistrationFlow UUID")
	if err != nil {
		return nil, err
	}
	roleUUID, err := grpcUUID(req.GetRoleUuid(), "Role UUID")
	if err != nil {
		return nil, err
	}
	if err := h.registrationFlowService.RemoveRole(ctx, sfUUID, tenant.TenantID, roleUUID); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.RemoveRegistrationFlowRoleResponse{Removed: true}, nil
}

func (h *RegistrationFlowGRPCHandler) resolveTenant(ctx context.Context, tenantUUID string) (*TenantServiceDataResult, error) {
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

func toRegistrationFlowProto(result *RegistrationFlowServiceDataResult) *authv1.RegistrationFlow {
	if result == nil {
		return nil
	}
	return &authv1.RegistrationFlow{
		RegistrationFlowUuid: result.RegistrationFlowUUID.String(),
		Name:                 result.Name,
		Description:          result.Description,
		Identifier:           result.Identifier,
		Status:               result.Status,
		ClientUuid:           result.ClientUUID.String(),
		CreatedAt:            timestamppb.New(result.CreatedAt),
		UpdatedAt:            timestamppb.New(result.UpdatedAt),
	}
}

func toRegistrationFlowRoleProto(result *RegistrationFlowRoleServiceDataResult) *authv1.RegistrationFlowRole {
	if result == nil {
		return nil
	}
	return &authv1.RegistrationFlowRole{
		RegistrationFlowRoleUuid: result.RegistrationFlowRoleUUID.String(),
		RoleUuid:                 result.RoleUUID.String(),
		RoleName:                 result.RoleName,
		RoleDescription:          result.RoleDescription,
		RoleIsDefault:            result.RoleIsDefault,
		RoleIsSystem:             result.RoleIsSystem,
		RoleStatus:               result.RoleStatus,
		CreatedAt:                timestamppb.New(result.CreatedAt),
		UpdatedAt:                timestamppb.New(result.UpdatedAt),
	}
}

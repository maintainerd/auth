package iam

import (
	"context"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/auth/internal/tenant"
)

type ServiceGRPCHandler struct {
	authv1.UnimplementedServiceServiceServer
	tenantService  TenantResolver
	serviceService ServiceService
}

func NewServiceGRPCHandler(tenantService TenantResolver, serviceService ServiceService) *ServiceGRPCHandler {
	return &ServiceGRPCHandler{tenantService: tenantService, serviceService: serviceService}
}

func (h *ServiceGRPCHandler) ListServices(ctx context.Context, req *authv1.ListServicesRequest) (*authv1.ListServicesResponse, error) {
	scope, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	dto := ServiceFilterDTO{
		Name:                 iamOptionalString(req.GetName()),
		DisplayName:          iamOptionalString(req.GetDisplayName()),
		Description:          iamOptionalString(req.GetDescription()),
		Version:              iamOptionalString(req.GetVersion()),
		Status:               req.GetStatus(),
		IsSystem:             req.IsSystem,
		PaginationRequestDTO: iamPaginationDTO(req.GetPagination()),
	}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.serviceService.Get(ctx, ServiceServiceGetFilter{
		Name:        dto.Name,
		DisplayName: dto.DisplayName,
		Description: dto.Description,
		Version:     dto.Version,
		Status:      dto.Status,
		IsSystem:    dto.IsSystem,
		TenantID:    &scope.TenantID,
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
		rows[i] = serviceProto(&result.Data[i])
	}
	return &authv1.ListServicesResponse{Services: rows, Page: iamPageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *ServiceGRPCHandler) GetService(ctx context.Context, req *authv1.GetServiceRequest) (*authv1.GetServiceResponse, error) {
	scope, id, err := h.resolveTenantAndUUID(ctx, req.GetTenantUuid(), req.GetServiceUuid(), "Service UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.serviceService.GetByUUID(ctx, id, scope.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetServiceResponse{Service: serviceProto(result)}, nil
}

func (h *ServiceGRPCHandler) CreateService(ctx context.Context, req *authv1.CreateServiceRequest) (*authv1.CreateServiceResponse, error) {
	scope, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	dto := ServiceCreateOrUpdateRequestDTO{Name: req.GetName(), DisplayName: req.GetDisplayName(), Description: req.GetDescription(), Version: req.GetVersion(), Status: req.GetStatus()}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.serviceService.Create(ctx, dto.Name, dto.DisplayName, dto.Description, dto.Version, false, dto.Status, scope.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CreateServiceResponse{Service: serviceProto(result)}, nil
}

func (h *ServiceGRPCHandler) UpdateService(ctx context.Context, req *authv1.UpdateServiceRequest) (*authv1.UpdateServiceResponse, error) {
	scope, id, err := h.resolveTenantAndUUID(ctx, req.GetTenantUuid(), req.GetServiceUuid(), "Service UUID")
	if err != nil {
		return nil, err
	}
	dto := ServiceCreateOrUpdateRequestDTO{Name: req.GetName(), DisplayName: req.GetDisplayName(), Description: req.GetDescription(), Version: req.GetVersion(), Status: req.GetStatus()}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.serviceService.Update(ctx, id, scope.TenantID, dto.Name, dto.DisplayName, dto.Description, dto.Version, false, dto.Status)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdateServiceResponse{Service: serviceProto(result)}, nil
}

func (h *ServiceGRPCHandler) SetServiceStatus(ctx context.Context, req *authv1.SetServiceStatusRequest) (*authv1.SetServiceStatusResponse, error) {
	scope, id, err := h.resolveTenantAndUUID(ctx, req.GetTenantUuid(), req.GetServiceUuid(), "Service UUID")
	if err != nil {
		return nil, err
	}
	dto := ServiceStatusUpdateRequestDTO{Status: req.GetStatus()}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.serviceService.SetStatusByUUID(ctx, id, scope.TenantID, dto.Status)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.SetServiceStatusResponse{Service: serviceProto(result)}, nil
}

func (h *ServiceGRPCHandler) DeleteService(ctx context.Context, req *authv1.DeleteServiceRequest) (*authv1.DeleteServiceResponse, error) {
	scope, id, err := h.resolveTenantAndUUID(ctx, req.GetTenantUuid(), req.GetServiceUuid(), "Service UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.serviceService.DeleteByUUID(ctx, id, scope.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeleteServiceResponse{Service: serviceProto(result)}, nil
}

func (h *ServiceGRPCHandler) AssignServicePolicy(ctx context.Context, req *authv1.AssignServicePolicyRequest) (*authv1.AssignServicePolicyResponse, error) {
	scope, serviceUUID, policyUUID, err := h.resolveServicePolicyScope(ctx, req.GetTenantUuid(), req.GetServiceUuid(), req.GetPolicyUuid())
	if err != nil {
		return nil, err
	}
	if err := h.serviceService.AssignPolicy(ctx, serviceUUID, policyUUID, scope.TenantID); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.AssignServicePolicyResponse{Assigned: true}, nil
}

func (h *ServiceGRPCHandler) RemoveServicePolicy(ctx context.Context, req *authv1.RemoveServicePolicyRequest) (*authv1.RemoveServicePolicyResponse, error) {
	scope, serviceUUID, policyUUID, err := h.resolveServicePolicyScope(ctx, req.GetTenantUuid(), req.GetServiceUuid(), req.GetPolicyUuid())
	if err != nil {
		return nil, err
	}
	if err := h.serviceService.RemovePolicy(ctx, serviceUUID, policyUUID, scope.TenantID); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.RemoveServicePolicyResponse{Removed: true}, nil
}

func (h *ServiceGRPCHandler) resolveTenant(ctx context.Context, tenantUUID string) (*tenant.TenantServiceDataResult, error) {
	parsed, err := iamParseUUID(tenantUUID, "Tenant UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.tenantService.GetByUUID(ctx, parsed)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return result, nil
}

func (h *ServiceGRPCHandler) resolveTenantAndUUID(ctx context.Context, tenantUUID string, value string, label string) (*tenant.TenantServiceDataResult, uuid.UUID, error) {
	scope, err := h.resolveTenant(ctx, tenantUUID)
	if err != nil {
		return nil, uuid.Nil, err
	}
	parsed, err := iamParseUUID(value, label)
	if err != nil {
		return nil, uuid.Nil, err
	}
	return scope, parsed, nil
}

func (h *ServiceGRPCHandler) resolveServicePolicyScope(ctx context.Context, tenantUUID string, serviceValue string, policyValue string) (*tenant.TenantServiceDataResult, uuid.UUID, uuid.UUID, error) {
	scope, serviceUUID, err := h.resolveTenantAndUUID(ctx, tenantUUID, serviceValue, "Service UUID")
	if err != nil {
		return nil, uuid.Nil, uuid.Nil, err
	}
	policyUUID, err := iamParseUUID(policyValue, "Policy UUID")
	if err != nil {
		return nil, uuid.Nil, uuid.Nil, err
	}
	return scope, serviceUUID, policyUUID, nil
}

package iam

import (
	"context"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/auth/internal/platform/pagination"
	"github.com/maintainerd/auth/internal/tenant"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TenantResolver interface {
	GetByUUID(ctx context.Context, tenantUUID uuid.UUID) (*tenant.TenantServiceDataResult, error)
}

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

type APIGRPCHandler struct {
	authv1.UnimplementedAPIServiceServer
	tenantService TenantResolver
	apiService    APIService
}

func NewAPIGRPCHandler(tenantService TenantResolver, apiService APIService) *APIGRPCHandler {
	return &APIGRPCHandler{tenantService: tenantService, apiService: apiService}
}

func (h *APIGRPCHandler) ListAPIs(ctx context.Context, req *authv1.ListAPIsRequest) (*authv1.ListAPIsResponse, error) {
	scope, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	dto := APIFilterDTO{
		Name:                 iamOptionalString(req.GetName()),
		DisplayName:          iamOptionalString(req.GetDisplayName()),
		APIType:              iamOptionalString(req.GetApiType()),
		Identifier:           iamOptionalString(req.GetIdentifier()),
		ServiceUUID:          iamOptionalString(req.GetServiceUuid()),
		Status:               req.GetStatus(),
		IsSystem:             req.IsSystem,
		PaginationRequestDTO: iamPaginationDTO(req.GetPagination()),
	}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	var serviceID *int64
	if dto.ServiceUUID != nil {
		serviceUUID, err := iamParseUUID(*dto.ServiceUUID, "Service UUID")
		if err != nil {
			return nil, err
		}
		resolved, err := h.apiService.GetServiceIDByUUID(ctx, serviceUUID)
		if err != nil {
			return nil, apperror.ToGRPCError(err)
		}
		serviceID = &resolved
	}
	result, err := h.apiService.Get(ctx, APIServiceGetFilter{
		TenantID:    scope.TenantID,
		Name:        dto.Name,
		DisplayName: dto.DisplayName,
		APIType:     dto.APIType,
		Identifier:  dto.Identifier,
		ServiceID:   serviceID,
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
	rows := make([]*authv1.API, len(result.Data))
	for i := range result.Data {
		rows[i] = apiProto(&result.Data[i])
	}
	return &authv1.ListAPIsResponse{Apis: rows, Page: iamPageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *APIGRPCHandler) GetAPI(ctx context.Context, req *authv1.GetAPIRequest) (*authv1.GetAPIResponse, error) {
	scope, id, err := h.resolveTenantAndUUID(ctx, req.GetTenantUuid(), req.GetApiUuid(), "API UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.apiService.GetByUUID(ctx, id, scope.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetAPIResponse{Api: apiProto(result)}, nil
}

func (h *APIGRPCHandler) CreateAPI(ctx context.Context, req *authv1.CreateAPIRequest) (*authv1.CreateAPIResponse, error) {
	scope, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	dto := APICreateRequestDTO{Name: req.GetName(), DisplayName: req.GetDisplayName(), Description: req.GetDescription(), APIType: req.GetApiType(), Status: req.GetStatus(), ServiceUUID: req.GetServiceUuid()}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.apiService.Create(ctx, scope.TenantID, dto.Name, dto.DisplayName, dto.Description, dto.APIType, dto.Status, false, dto.ServiceUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CreateAPIResponse{Api: apiProto(result)}, nil
}

func (h *APIGRPCHandler) UpdateAPI(ctx context.Context, req *authv1.UpdateAPIRequest) (*authv1.UpdateAPIResponse, error) {
	scope, id, err := h.resolveTenantAndUUID(ctx, req.GetTenantUuid(), req.GetApiUuid(), "API UUID")
	if err != nil {
		return nil, err
	}
	dto := APIUpdateRequestDTO{Name: req.GetName(), DisplayName: req.GetDisplayName(), Description: req.GetDescription(), APIType: req.GetApiType(), Status: req.GetStatus(), ServiceUUID: req.GetServiceUuid()}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.apiService.Update(ctx, id, scope.TenantID, dto.Name, dto.DisplayName, dto.Description, dto.APIType, dto.Status, dto.ServiceUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdateAPIResponse{Api: apiProto(result)}, nil
}

func (h *APIGRPCHandler) SetAPIStatus(ctx context.Context, req *authv1.SetAPIStatusRequest) (*authv1.SetAPIStatusResponse, error) {
	scope, id, err := h.resolveTenantAndUUID(ctx, req.GetTenantUuid(), req.GetApiUuid(), "API UUID")
	if err != nil {
		return nil, err
	}
	dto := APIStatusUpdateDTO{Status: req.GetStatus()}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.apiService.SetStatusByUUID(ctx, id, scope.TenantID, dto.Status)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.SetAPIStatusResponse{Api: apiProto(result)}, nil
}

func (h *APIGRPCHandler) DeleteAPI(ctx context.Context, req *authv1.DeleteAPIRequest) (*authv1.DeleteAPIResponse, error) {
	scope, id, err := h.resolveTenantAndUUID(ctx, req.GetTenantUuid(), req.GetApiUuid(), "API UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.apiService.DeleteByUUID(ctx, id, scope.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeleteAPIResponse{Api: apiProto(result)}, nil
}

func (h *APIGRPCHandler) resolveTenant(ctx context.Context, tenantUUID string) (*tenant.TenantServiceDataResult, error) {
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

func (h *APIGRPCHandler) resolveTenantAndUUID(ctx context.Context, tenantUUID string, value string, label string) (*tenant.TenantServiceDataResult, uuid.UUID, error) {
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

func serviceProto(result *ServiceServiceDataResult) *authv1.Service {
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

func apiProto(result *APIServiceDataResult) *authv1.API {
	if result == nil {
		return nil
	}
	return &authv1.API{
		ApiUuid:     result.APIUUID.String(),
		Name:        result.Name,
		DisplayName: result.DisplayName,
		Description: result.Description,
		ApiType:     result.APIType,
		Identifier:  result.Identifier,
		Status:      result.Status,
		IsSystem:    result.IsSystem,
		Service:     serviceProto(result.Service),
		CreatedAt:   timestamppb.New(result.CreatedAt),
		UpdatedAt:   timestamppb.New(result.UpdatedAt),
	}
}

func iamParseUUID(value string, label string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation(label + " is required"))
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation("Invalid " + label))
	}
	return parsed, nil
}

func iamPaginationDTO(req *authv1.Pagination) PaginationRequestDTO {
	if req == nil {
		return PaginationRequestDTO{Page: 1, Limit: pagination.DefaultPageSize}
	}
	page := int(req.GetPage())
	limit := int(req.GetLimit())
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = pagination.DefaultPageSize
	}
	return PaginationRequestDTO{Page: page, Limit: limit, SortBy: req.GetSortBy(), SortOrder: req.GetSortOrder()}
}

func iamPageProto(total int64, page int, limit int, totalPages int) *authv1.PageMetadata {
	return &authv1.PageMetadata{Total: total, Page: int32(page), Limit: int32(limit), TotalPages: int32(totalPages)}
}

func iamOptionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

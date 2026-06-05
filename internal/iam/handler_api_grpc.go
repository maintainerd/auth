package iam

import (
	"context"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/auth/internal/tenant"
)

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

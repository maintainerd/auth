package iam

import (
	"context"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
)

type PolicyGRPCHandler struct {
	authv1.UnimplementedPolicyServiceServer
	tenantService TenantResolver
	policyService PolicyService
}

func NewPolicyGRPCHandler(tenantService TenantResolver, policyService PolicyService) *PolicyGRPCHandler {
	return &PolicyGRPCHandler{tenantService: tenantService, policyService: policyService}
}

func (h *PolicyGRPCHandler) ListPolicies(ctx context.Context, req *authv1.ListPoliciesRequest) (*authv1.ListPoliciesResponse, error) {
	scope, err := resolveIAMTenant(ctx, h.tenantService, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	var serviceID *uuid.UUID
	if req.GetServiceId() != "" {
		parsed, err := iamParseUUID(req.GetServiceId(), "Service UUID")
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
	scope, id, err := resolveIAMTenantAndUUID(ctx, h.tenantService, req.GetTenantId(), req.GetPolicyId(), "Policy UUID")
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
	scope, policyUUID, err := resolveIAMTenantAndUUID(ctx, h.tenantService, req.GetTenantId(), req.GetPolicyId(), "Policy UUID")
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
	scope, err := resolveIAMTenant(ctx, h.tenantService, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	// The tenant boundary is enforced BEFORE the ledger claim so a caller that may
	// not act on this tenant cannot consume — or occupy — a key it may not spend.
	document, err := policyDocumentJSON(req.GetDocument())
	if err != nil {
		return nil, err
	}
	dto := PolicyCreateRequestDTO{Name: req.GetName(), Description: iamOptionalString(req.GetDescription()), Document: document, Version: req.GetVersion(), Status: req.GetStatus()}
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
	scope, id, err := resolveIAMTenantAndUUID(ctx, h.tenantService, req.GetTenantId(), req.GetPolicyId(), "Policy UUID")
	if err != nil {
		return nil, err
	}
	document, err := policyDocumentJSON(req.GetDocument())
	if err != nil {
		return nil, err
	}
	dto := PolicyUpdateRequestDTO{Name: req.GetName(), Description: iamOptionalString(req.GetDescription()), Document: document, Version: req.GetVersion(), Status: req.GetStatus()}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	// The control plane authenticates as a service, not a user, so the change is
	// attributed to the calling client. Without this the gRPC path — which writes no
	// management_audit_log row either — would leave policy changes entirely
	// unattributed.
	result, err := h.policyService.Update(ctx, id, scope.TenantID, dto.Name, dto.Description,
		dto.Document, dto.Version, dto.Status, policyChangeContextFromGRPC(ctx))
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdatePolicyResponse{Policy: policyProto(result)}, nil
}

func (h *PolicyGRPCHandler) SetPolicyStatus(ctx context.Context, req *authv1.SetPolicyStatusRequest) (*authv1.SetPolicyStatusResponse, error) {
	scope, id, err := resolveIAMTenantAndUUID(ctx, h.tenantService, req.GetTenantId(), req.GetPolicyId(), "Policy UUID")
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
	scope, id, err := resolveIAMTenantAndUUID(ctx, h.tenantService, req.GetTenantId(), req.GetPolicyId(), "Policy UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.policyService.DeleteByUUID(ctx, id, scope.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeletePolicyResponse{Policy: policyProto(result)}, nil
}

// policyChangeContextFromGRPC attributes a policy change to the calling principal.
func policyChangeContextFromGRPC(ctx context.Context) PolicyChangeContext {
	claims := middleware.JWTClaimsFromContext(ctx)
	if claims == nil {
		return PolicyChangeContext{}
	}
	reason := "changed via the gRPC control plane by " + claims.Service
	return PolicyChangeContext{Reason: &reason}
}

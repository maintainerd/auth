package secpolicy

import (
	"context"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/auth/internal/platform/pagination"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TenantResolver interface {
	GetByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
}

type IPRestrictionRuleGRPCHandler struct {
	authv1.UnimplementedIPRestrictionRuleServiceServer
	tenantResolver TenantResolver
	svc            IPRestrictionRuleService
}

func NewIPRestrictionRuleGRPCHandler(tenantResolver TenantResolver, svc IPRestrictionRuleService) *IPRestrictionRuleGRPCHandler {
	return &IPRestrictionRuleGRPCHandler{tenantResolver: tenantResolver, svc: svc}
}

func (h *IPRestrictionRuleGRPCHandler) ListIPRestrictionRules(ctx context.Context, req *authv1.ListIPRestrictionRulesRequest) (*authv1.ListIPRestrictionRulesResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	pg := grpcPagination(req.GetPagination())
	result, err := h.svc.GetAll(ctx, tenant.TenantID, optionalStr(req.GetType()), req.GetStatus(), optionalStr(req.GetIpAddress()), optionalStr(req.GetDescription()), pg.Page, pg.Limit, pg.SortBy, pg.SortOrder)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.IPRestrictionRule, len(result.Data))
	for i := range result.Data {
		rows[i] = ipRuleProto(&result.Data[i])
	}
	return &authv1.ListIPRestrictionRulesResponse{Rules: rows, Page: pageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *IPRestrictionRuleGRPCHandler) GetIPRestrictionRule(ctx context.Context, req *authv1.GetIPRestrictionRuleRequest) (*authv1.GetIPRestrictionRuleResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	ruleUUID, err := parseUUID(req.GetIpRestrictionRuleUuid(), "IP restriction rule UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.svc.GetByUUID(ctx, tenant.TenantID, ruleUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetIPRestrictionRuleResponse{Rule: ipRuleProto(result)}, nil
}

func (h *IPRestrictionRuleGRPCHandler) CreateIPRestrictionRule(ctx context.Context, req *authv1.CreateIPRestrictionRuleRequest) (*authv1.CreateIPRestrictionRuleResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	result, err := h.svc.Create(ctx, tenant.TenantID, req.GetDescription(), req.GetType(), req.GetIpAddress(), req.GetStatus(), req.GetCreatedBy())
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CreateIPRestrictionRuleResponse{Rule: ipRuleProto(result)}, nil
}

func (h *IPRestrictionRuleGRPCHandler) UpdateIPRestrictionRule(ctx context.Context, req *authv1.UpdateIPRestrictionRuleRequest) (*authv1.UpdateIPRestrictionRuleResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	ruleUUID, err := parseUUID(req.GetIpRestrictionRuleUuid(), "IP restriction rule UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.svc.Update(ctx, tenant.TenantID, ruleUUID, req.GetDescription(), req.GetType(), req.GetIpAddress(), req.GetStatus(), req.GetUpdatedBy())
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdateIPRestrictionRuleResponse{Rule: ipRuleProto(result)}, nil
}

func (h *IPRestrictionRuleGRPCHandler) SetIPRestrictionRuleStatus(ctx context.Context, req *authv1.SetIPRestrictionRuleStatusRequest) (*authv1.SetIPRestrictionRuleStatusResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	ruleUUID, err := parseUUID(req.GetIpRestrictionRuleUuid(), "IP restriction rule UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.svc.UpdateStatus(ctx, tenant.TenantID, ruleUUID, req.GetStatus(), req.GetUpdatedBy())
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.SetIPRestrictionRuleStatusResponse{Rule: ipRuleProto(result)}, nil
}

func (h *IPRestrictionRuleGRPCHandler) DeleteIPRestrictionRule(ctx context.Context, req *authv1.DeleteIPRestrictionRuleRequest) (*authv1.DeleteIPRestrictionRuleResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	ruleUUID, err := parseUUID(req.GetIpRestrictionRuleUuid(), "IP restriction rule UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.svc.Delete(ctx, tenant.TenantID, ruleUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeleteIPRestrictionRuleResponse{Rule: ipRuleProto(result)}, nil
}

func (h *IPRestrictionRuleGRPCHandler) resolveTenant(ctx context.Context, tenantUUID string) (*TenantServiceDataResult, error) {
	parsed, err := parseUUID(tenantUUID, "Tenant UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.tenantResolver.GetByUUID(ctx, parsed)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return result, nil
}

func parseUUID(value string, label string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation(label + " is required"))
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation("Invalid " + label))
	}
	return parsed, nil
}

func optionalStr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func grpcPagination(req *authv1.Pagination) PaginationRequestDTO {
	if req == nil {
		return PaginationRequestDTO{Page: 1, Limit: pagination.DefaultPageSize}
	}
	page, limit := int(req.GetPage()), int(req.GetLimit())
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = pagination.DefaultPageSize
	}
	return PaginationRequestDTO{Page: page, Limit: limit, SortBy: req.GetSortBy(), SortOrder: req.GetSortOrder()}
}

func pageProto(total int64, page, limit, totalPages int) *authv1.PageMetadata {
	return &authv1.PageMetadata{Total: total, Page: int32(page), Limit: int32(limit), TotalPages: int32(totalPages)}
}

func ipRuleProto(r *IPRestrictionRuleServiceDataResult) *authv1.IPRestrictionRule {
	if r == nil {
		return nil
	}
	return &authv1.IPRestrictionRule{
		IpRestrictionRuleUuid: r.IPRestrictionRuleUUID.String(), Description: r.Description,
		Type: r.Type, IpAddress: r.IPAddress, Status: r.Status,
		CreatedAt: timestamppb.New(r.CreatedAt), UpdatedAt: timestamppb.New(r.UpdatedAt),
	}
}

package webhook

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

type WebhookEndpointGRPCHandler struct {
	authv1.UnimplementedWebhookEndpointServiceServer
	tenantResolver TenantResolver
	svc            WebhookEndpointService
}

func NewWebhookEndpointGRPCHandler(r TenantResolver, svc WebhookEndpointService) *WebhookEndpointGRPCHandler {
	return &WebhookEndpointGRPCHandler{tenantResolver: r, svc: svc}
}

func (h *WebhookEndpointGRPCHandler) ListWebhookEndpoints(ctx context.Context, req *authv1.ListWebhookEndpointsRequest) (*authv1.ListWebhookEndpointsResponse, error) {
	t, err := h.tenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	pg := paginationDTO(req.GetPagination())
	r, err := h.svc.GetAll(ctx, t.TenantID, req.GetStatus(), pg.Page, pg.Limit, pg.SortBy, pg.SortOrder)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.WebhookEndpoint, len(r.Data))
	for i := range r.Data {
		rows[i] = webhookProto(&r.Data[i])
	}
	return &authv1.ListWebhookEndpointsResponse{Endpoints: rows, Page: pageProto(r.Total, r.Page, r.Limit, r.TotalPages)}, nil
}

func (h *WebhookEndpointGRPCHandler) GetWebhookEndpoint(ctx context.Context, req *authv1.GetWebhookEndpointRequest) (*authv1.GetWebhookEndpointResponse, error) {
	t, err := h.tenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	id, err := parseUUID(req.GetWebhookEndpointUuid(), "Webhook endpoint UUID")
	if err != nil {
		return nil, err
	}
	r, err := h.svc.GetByUUID(ctx, t.TenantID, id)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetWebhookEndpointResponse{Endpoint: webhookProto(r)}, nil
}

func (h *WebhookEndpointGRPCHandler) CreateWebhookEndpoint(ctx context.Context, req *authv1.CreateWebhookEndpointRequest) (*authv1.CreateWebhookEndpointResponse, error) {
	t, err := h.tenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	r, err := h.svc.Create(ctx, t.TenantID, req.GetUrl(), true,
		intPtr(req.MaxRetries), intPtr(req.TimeoutSeconds), req.GetDescription(), req.GetStatus())
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CreateWebhookEndpointResponse{Endpoint: webhookProto(r)}, nil
}

func (h *WebhookEndpointGRPCHandler) UpdateWebhookEndpoint(ctx context.Context, req *authv1.UpdateWebhookEndpointRequest) (*authv1.UpdateWebhookEndpointResponse, error) {
	t, err := h.tenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	id, err := parseUUID(req.GetWebhookEndpointUuid(), "Webhook endpoint UUID")
	if err != nil {
		return nil, err
	}
	r, err := h.svc.Update(ctx, t.TenantID, id, req.GetUrl(), false, true,
		intPtr(req.MaxRetries), intPtr(req.TimeoutSeconds), req.GetDescription(), req.GetStatus())
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdateWebhookEndpointResponse{Endpoint: webhookProto(r)}, nil
}

func (h *WebhookEndpointGRPCHandler) SetWebhookEndpointStatus(ctx context.Context, req *authv1.SetWebhookEndpointStatusRequest) (*authv1.SetWebhookEndpointStatusResponse, error) {
	t, err := h.tenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	id, err := parseUUID(req.GetWebhookEndpointUuid(), "Webhook endpoint UUID")
	if err != nil {
		return nil, err
	}
	r, err := h.svc.UpdateStatus(ctx, t.TenantID, id, req.GetStatus())
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.SetWebhookEndpointStatusResponse{Endpoint: webhookProto(r)}, nil
}

func (h *WebhookEndpointGRPCHandler) DeleteWebhookEndpoint(ctx context.Context, req *authv1.DeleteWebhookEndpointRequest) (*authv1.DeleteWebhookEndpointResponse, error) {
	t, err := h.tenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	id, err := parseUUID(req.GetWebhookEndpointUuid(), "Webhook endpoint UUID")
	if err != nil {
		return nil, err
	}
	r, err := h.svc.Delete(ctx, t.TenantID, id)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeleteWebhookEndpointResponse{Endpoint: webhookProto(r)}, nil
}

func (h *WebhookEndpointGRPCHandler) tenant(ctx context.Context, tuuid string) (*TenantServiceDataResult, error) {
	p, err := parseUUID(tuuid, "Tenant UUID")
	if err != nil {
		return nil, err
	}
	r, err := h.tenantResolver.GetByUUID(ctx, p)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return r, nil
}

func parseUUID(value, label string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation(label + " is required"))
	}
	p, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation("Invalid " + label))
	}
	return p, nil
}

func paginationDTO(req *authv1.Pagination) PaginationRequestDTO {
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

func intPtr(v *int32) *int {
	if v == nil {
		return nil
	}
	n := int(*v)
	return &n
}

func webhookProto(r *WebhookEndpointServiceDataResult) *authv1.WebhookEndpoint {
	if r == nil {
		return nil
	}
	var lt *timestamppb.Timestamp
	if r.LastTriggeredAt != nil {
		lt = timestamppb.New(*r.LastTriggeredAt)
	}
	return &authv1.WebhookEndpoint{
		WebhookEndpointUuid: r.WebhookEndpointUUID.String(), Url: r.URL, Description: r.Description,
		MaxRetries: int32(r.MaxRetries), TimeoutSeconds: int32(r.TimeoutSeconds), Status: r.Status,
		LastTriggeredAt: lt, CreatedAt: timestamppb.New(r.CreatedAt), UpdatedAt: timestamppb.New(r.UpdatedAt),
	}
}

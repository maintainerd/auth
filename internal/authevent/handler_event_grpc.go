package authevent

import (
	"context"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/auth/internal/platform/pagination"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TenantResolver interface {
	GetByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
}

type AuthEventGRPCHandler struct {
	authv1.UnimplementedAuthEventServiceServer
	tenantResolver TenantResolver
	svc            AuthEventService
}

func NewAuthEventGRPCHandler(r TenantResolver, svc AuthEventService) *AuthEventGRPCHandler {
	return &AuthEventGRPCHandler{tenantResolver: r, svc: svc}
}

func (h *AuthEventGRPCHandler) ListAuthEvents(ctx context.Context, req *authv1.ListAuthEventsRequest) (*authv1.ListAuthEventsResponse, error) {
	t, err := h.tenant(ctx, req.GetTenantUuid())
	if err != nil { return nil, err }
	pg := paginationDTO(req.GetPagination())
	tid := t.TenantID
	r, err := h.svc.FindPaginated(ctx, AuthEventRepositoryGetFilter{
		TenantID: &tid, EventType: strPtr(req.GetEventType()), Category: strPtr(req.GetCategory()),
		Severity: strPtr(req.GetSeverity()), Result: strPtr(req.GetResult()),
		Page: pg.Page, Limit: pg.Limit, SortBy: pg.SortBy, SortOrder: pg.SortOrder,
	})
	if err != nil { return nil, apperror.ToGRPCError(err) }
	rows := make([]*authv1.AuthEvent, len(r.Data))
	for i := range r.Data { rows[i] = eventProto(&r.Data[i]) }
	return &authv1.ListAuthEventsResponse{Events: rows, Page: pageProto(r.Total, r.Page, r.Limit, r.TotalPages)}, nil
}

func (h *AuthEventGRPCHandler) GetAuthEvent(ctx context.Context, req *authv1.GetAuthEventRequest) (*authv1.GetAuthEventResponse, error) {
	t, err := h.tenant(ctx, req.GetTenantUuid())
	if err != nil { return nil, err }
	id, err := parseUUID(req.GetAuthEventUuid(), "Auth event UUID")
	if err != nil { return nil, err }
	r, err := h.svc.FindByUUID(ctx, t.TenantID, id)
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return &authv1.GetAuthEventResponse{Event: eventProto(r)}, nil
}

func (h *AuthEventGRPCHandler) CountAuthEventsByType(ctx context.Context, req *authv1.CountAuthEventsByTypeRequest) (*authv1.CountAuthEventsByTypeResponse, error) {
	t, err := h.tenant(ctx, req.GetTenantUuid())
	if err != nil { return nil, err }
	count, err := h.svc.CountByEventType(ctx, req.GetEventType(), t.TenantID)
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return &authv1.CountAuthEventsByTypeResponse{Count: count}, nil
}

func (h *AuthEventGRPCHandler) tenant(ctx context.Context, tuuid string) (*TenantServiceDataResult, error) {
	p, err := parseUUID(tuuid, "Tenant UUID")
	if err != nil { return nil, err }
	r, err := h.tenantResolver.GetByUUID(ctx, p)
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return r, nil
}

func parseUUID(value, label string) (uuid.UUID, error) {
	if value == "" { return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation(label+" is required")) }
	p, err := uuid.Parse(value)
	if err != nil { return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation("Invalid "+label)) }
	return p, nil
}

func paginationDTO(req *authv1.Pagination) PaginationRequestDTO {
	if req == nil { return PaginationRequestDTO{Page: 1, Limit: pagination.DefaultPageSize} }
	page, limit := int(req.GetPage()), int(req.GetLimit())
	if page == 0 { page = 1 }; if limit == 0 { limit = pagination.DefaultPageSize }
	return PaginationRequestDTO{Page: page, Limit: limit, SortBy: req.GetSortBy(), SortOrder: req.GetSortOrder()}
}

func pageProto(total int64, page, limit, totalPages int) *authv1.PageMetadata {
	return &authv1.PageMetadata{Total: total, Page: int32(page), Limit: int32(limit), TotalPages: int32(totalPages)}
}

func eventProto(r *AuthEventServiceDataResult) *authv1.AuthEvent {
	if r == nil { return nil }
	var meta *structpb.Struct
	if len(r.Metadata) > 0 { meta, _ = structpb.NewStruct(map[string]any{}) }
	return &authv1.AuthEvent{
		AuthEventUuid: r.AuthEventUUID.String(), Category: r.Category, EventType: r.EventType,
		Severity: r.Severity, Result: r.Result, IpAddress: r.IPAddress,
		UserAgent: r.UserAgent, Description: r.Description, ErrorReason: r.ErrorReason,
		TraceId: r.TraceID, Metadata: meta, CreatedAt: timestamppb.New(r.CreatedAt),
	}
}

func strPtr(v string) *string { if v == "" { return nil }; return &v }

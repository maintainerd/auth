package authevent

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/pagination"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// AuthEventHandler handles admin endpoints for querying auth events.
type AuthEventHandler struct {
	authEventService AuthEventService
}

// NewAuthEventHandler creates a new AuthEventHandler.
func NewAuthEventHandler(authEventService AuthEventService) *AuthEventHandler {
	return &AuthEventHandler{authEventService: authEventService}
}

// GetAll returns a paginated list of auth events for the authenticated tenant.
//
// GET /auth-events
func (h *AuthEventHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	repoFilter, ok := h.repositoryFilterFromRequest(w, r, tenant.TenantID, pagination.ParseQuery(r))
	if !ok {
		return
	}

	result, err := h.authEventService.FindPaginated(r.Context(), repoFilter)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get auth events", err)
		return
	}

	response := PaginatedResponseDTO[AuthEventResponseDTO]{
		Rows:       toAuthEventResponseDTOList(result.Data),
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "Auth events retrieved successfully")
}

// Export returns auth events as CSV or JSON for the authenticated tenant.
//
// GET /auth-events/export?format=csv
func (h *AuthEventHandler) Export(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	exporter, ok := h.authEventService.(interface {
		Export(context.Context, AuthEventRepositoryGetFilter, string) (*AuthEventExport, error)
	})
	if !ok {
		resp.Error(w, http.StatusNotImplemented, "Auth event export is not available")
		return
	}

	paginationRequest := pagination.PaginationRequestDTO{
		Page:      1,
		Limit:     1,
		SortBy:    r.URL.Query().Get("sort_by"),
		SortOrder: r.URL.Query().Get("sort_order"),
	}
	repoFilter, ok := h.repositoryFilterFromRequest(w, r, tenant.TenantID, paginationRequest)
	if !ok {
		return
	}

	export, err := exporter.Export(r.Context(), repoFilter, r.URL.Query().Get("format"))
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to export auth events", err)
		return
	}

	// PCI 10.2.3: exporting the audit trail is itself a security-relevant access
	// event (a bulk dump of security data with actor/IP/PII) — record it durably.
	auth := middleware.AuthFromRequest(r)
	var actorID *int64
	if auth.User != nil {
		actorID = &auth.User.UserID
	}
	h.authEventService.Log(r.Context(), AuthEventInput{
		TenantID:    tenant.TenantID,
		ActorUserID: actorID,
		IPAddress:   middleware.ClientIPFromContext(r.Context()),
		Category:    AuthEventCategorySystem,
		EventType:   AuthEventTypeAuditExport,
		Severity:    AuthEventSeverityInfo,
		Result:      AuthEventResultSuccess,
		Description: ptr.Ptr("Auth-event audit trail exported"),
	})

	w.Header().Set("Content-Type", export.ContentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+export.Filename+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(export.Data)
}

// Get returns a single auth event by UUID for the authenticated tenant.
//
// GET /auth-events/{auth_event_uuid}
func (h *AuthEventHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	authEventUUIDStr := chi.URLParam(r, "auth_event_uuid")
	authEventUUID, err := uuid.Parse(authEventUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid auth event UUID")
		return
	}

	event, err := h.authEventService.FindByUUID(r.Context(), tenant.TenantID, authEventUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Auth event not found", err)
		return
	}

	resp.Success(w, toAuthEventResponseDTO(*event), "Auth event retrieved successfully")
}

// CountByType returns the number of events matching a type for the tenant.
//
// GET /auth-events/count
func (h *AuthEventHandler) CountByType(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	eventType := r.URL.Query().Get("event_type")
	if eventType == "" {
		resp.Error(w, http.StatusBadRequest, "event_type query parameter is required")
		return
	}

	count, err := h.authEventService.CountByEventType(r.Context(), eventType, tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to count auth events", err)
		return
	}

	resp.Success(w, map[string]int64{"count": count}, "Auth event count retrieved successfully")
}

func (h *AuthEventHandler) repositoryFilterFromRequest(
	w http.ResponseWriter,
	r *http.Request,
	tenantID int64,
	paginationRequest pagination.PaginationRequestDTO,
) (AuthEventRepositoryGetFilter, bool) {
	q := r.URL.Query()
	filter := AuthEventFilterDTO{
		Category:             ptr.PtrOrNil(q.Get("category")),
		EventType:            ptr.PtrOrNil(q.Get("event_type")),
		Severity:             ptr.PtrOrNil(q.Get("severity")),
		Result:               ptr.PtrOrNil(q.Get("result")),
		DateFrom:             ptr.PtrOrNil(q.Get("date_from")),
		DateTo:               ptr.PtrOrNil(q.Get("date_to")),
		PaginationRequestDTO: paginationRequest,
	}

	if err := filter.Validate(); err != nil {
		resp.ValidationError(w, err)
		return AuthEventRepositoryGetFilter{}, false
	}

	repoFilter := AuthEventRepositoryGetFilter{
		TenantID:  &tenantID,
		UserUUID:  ptr.PtrOrNil(q.Get("user")),
		Category:  filter.Category,
		EventType: filter.EventType,
		Severity:  filter.Severity,
		Result:    filter.Result,
		SortBy:    filter.SortBy,
		SortOrder: filter.SortOrder,
		Page:      filter.Page,
		Limit:     filter.Limit,
	}

	if filter.DateFrom != nil {
		if t, err := time.Parse(time.RFC3339, *filter.DateFrom); err == nil {
			repoFilter.DateFrom = &t
		}
	}
	if filter.DateTo != nil {
		if t, err := time.Parse(time.RFC3339, *filter.DateTo); err == nil {
			repoFilter.DateTo = &t
		}
	}

	return repoFilter, true
}

func toAuthEventResponseDTO(e AuthEventServiceDataResult) AuthEventResponseDTO {
	var metadata *map[string]any
	if e.Metadata != nil {
		var m map[string]any
		if err := json.Unmarshal(e.Metadata, &m); err == nil && len(m) > 0 {
			metadata = &m
		}
	}

	return AuthEventResponseDTO{
		AuthEventID:  e.AuthEventUUID.String(),
		TenantID:     e.TenantID,
		ActorUserID:  e.ActorUserID,
		TargetUserID: e.TargetUserID,
		IPAddress:    e.IPAddress,
		UserAgent:    e.UserAgent,
		Category:     e.Category,
		EventType:    e.EventType,
		Severity:     e.Severity,
		Result:       e.Result,
		Description:  e.Description,
		ErrorReason:  e.ErrorReason,
		TraceID:      e.TraceID,
		Metadata:     metadata,
		CreatedAt:    e.CreatedAt,
	}
}

func toAuthEventResponseDTOList(events []AuthEventServiceDataResult) []AuthEventResponseDTO {
	result := make([]AuthEventResponseDTO, len(events))
	for i, e := range events {
		result[i] = toAuthEventResponseDTO(e)
	}
	return result
}

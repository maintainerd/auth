package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/ptr"
	"github.com/maintainerd/auth/internal/repository"
	resp "github.com/maintainerd/auth/internal/rest/response"
	"github.com/maintainerd/auth/internal/service"
)

// AuthEventHandler handles admin endpoints for querying auth events.
type AuthEventHandler struct {
	authEventService service.AuthEventService
}

// NewAuthEventHandler creates a new AuthEventHandler.
func NewAuthEventHandler(authEventService service.AuthEventService) *AuthEventHandler {
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

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	filter := dto.AuthEventFilterDTO{
		Category:  ptr.PtrOrNil(q.Get("category")),
		EventType: ptr.PtrOrNil(q.Get("event_type")),
		Severity:  ptr.PtrOrNil(q.Get("severity")),
		Result:    ptr.PtrOrNil(q.Get("result")),
		DateFrom:  ptr.PtrOrNil(q.Get("date_from")),
		DateTo:    ptr.PtrOrNil(q.Get("date_to")),
		PaginationRequestDTO: dto.PaginationRequestDTO{
			Page:      page,
			Limit:     limit,
			SortBy:    q.Get("sort_by"),
			SortOrder: q.Get("sort_order"),
		},
	}

	if err := filter.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	repoFilter := repository.AuthEventRepositoryGetFilter{
		TenantID:  &tenant.TenantID,
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

	result, err := h.authEventService.FindPaginated(r.Context(), repoFilter)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get auth events", err)
		return
	}

	response := dto.PaginatedResponseDTO[dto.AuthEventResponseDTO]{
		Rows:       toAuthEventResponseDTOList(result.Data),
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "Auth events retrieved successfully")
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

func toAuthEventResponseDTO(e service.AuthEventServiceDataResult) dto.AuthEventResponseDTO {
	var metadata *map[string]any
	if e.Metadata != nil {
		var m map[string]any
		if err := json.Unmarshal(e.Metadata, &m); err == nil && len(m) > 0 {
			metadata = &m
		}
	}

	return dto.AuthEventResponseDTO{
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

func toAuthEventResponseDTOList(events []service.AuthEventServiceDataResult) []dto.AuthEventResponseDTO {
	result := make([]dto.AuthEventResponseDTO, len(events))
	for i, e := range events {
		result[i] = toAuthEventResponseDTO(e)
	}
	return result
}

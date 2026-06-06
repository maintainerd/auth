package event

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/middleware"
	resp "github.com/maintainerd/auth/internal/platform/response"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Event Route Service (broker routing)
// ---------------------------------------------------------------------------

// EventRouteService manages per-tenant broker route configurations.
type EventRouteService interface {
	ListByTenant(ctx context.Context, tenantID int64) ([]EventRouteServiceResult, error)
	Create(ctx context.Context, tenantID int64, eventTypeID int64, destination string) (*EventRouteServiceResult, error)
	Update(ctx context.Context, tenantID int64, eventRouteUUID uuid.UUID, destination string, enabled bool) (*EventRouteServiceResult, error)
	Delete(ctx context.Context, tenantID int64, eventRouteUUID uuid.UUID) error
}

type EventRouteServiceResult struct {
	EventRouteUUID string
	TenantID       int64
	EventTypeID    int64
	EventTypeKey   string
	Channel        string
	Destination    string
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type eventRouteServiceImpl struct {
	eventRouteRepo EventRouteRepository
	eventTypeRepo  EventTypeRepository
	writeGate      *WriteGate
	db             *gorm.DB
}

func NewEventRouteService(db *gorm.DB, eventRouteRepo EventRouteRepository, eventTypeRepo EventTypeRepository, writeGate *WriteGate) EventRouteService {
	return &eventRouteServiceImpl{eventRouteRepo: eventRouteRepo, eventTypeRepo: eventTypeRepo, writeGate: writeGate, db: db}
}

func (s *eventRouteServiceImpl) ListByTenant(ctx context.Context, tenantID int64) ([]EventRouteServiceResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "eventRoute.list")
	defer span.End()

	routes, err := s.eventRouteRepo.FindByTenantID(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list event routes failed")
		return nil, err
	}

	result := make([]EventRouteServiceResult, len(routes))
	for i, r := range routes {
		key := ""
		if et, _ := s.eventTypeRepo.FindByID(r.EventTypeID); et != nil {
			key = et.Key
		}
		result[i] = EventRouteServiceResult{
			EventRouteUUID: r.EventRouteUUID.String(),
			TenantID:       r.TenantID,
			EventTypeID:    r.EventTypeID,
			EventTypeKey:   key,
			Channel:        r.Channel,
			Destination:    r.Destination,
			Enabled:        r.Enabled,
			CreatedAt:      r.CreatedAt,
			UpdatedAt:      r.UpdatedAt,
		}
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *eventRouteServiceImpl) Create(ctx context.Context, tenantID int64, eventTypeID int64, destination string) (*EventRouteServiceResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "eventRoute.create")
	defer span.End()

	et, err := s.eventTypeRepo.FindByID(eventTypeID)
	if err != nil || et == nil {
		span.SetStatus(codes.Error, "event type not found")
		return nil, apperror.NewNotFound("event type")
	}
	// Do not allow routing a retired/disabled event type.
	if !et.IsActive {
		span.SetStatus(codes.Error, "event type is not active")
		return nil, apperror.NewValidation("event type is not active and cannot be routed")
	}

	route := &EventRoute{
		TenantID:    tenantID,
		EventTypeID: eventTypeID,
		Destination: destination,
		Channel:     "rabbitmq",
		Enabled:     true,
	}

	created, err := s.eventRouteRepo.Create(route)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create event route failed")
		return nil, err
	}

	s.writeGate.InvalidateTenant(ctx, tenantID)
	span.SetStatus(codes.Ok, "")
	return &EventRouteServiceResult{
		EventRouteUUID: created.EventRouteUUID.String(),
		TenantID:       created.TenantID,
		EventTypeID:    created.EventTypeID,
		EventTypeKey:   et.Key,
		Channel:        created.Channel,
		Destination:    created.Destination,
		Enabled:        created.Enabled,
		CreatedAt:      created.CreatedAt,
		UpdatedAt:      created.UpdatedAt,
	}, nil
}

func (s *eventRouteServiceImpl) Update(ctx context.Context, tenantID int64, eventRouteUUID uuid.UUID, destination string, enabled bool) (*EventRouteServiceResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "eventRoute.update")
	defer span.End()

	route, err := s.eventRouteRepo.FindByUUID(eventRouteUUID)
	if err != nil || route == nil {
		span.SetStatus(codes.Error, "event route not found")
		return nil, apperror.NewNotFound("event route")
	}
	if route.TenantID != tenantID {
		return nil, apperror.NewNotFound("event route")
	}

	route.Destination = destination
	route.Enabled = enabled

	updated, err := s.eventRouteRepo.UpdateByUUID(eventRouteUUID, route)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update event route failed")
		return nil, err
	}

	s.writeGate.InvalidateTenant(ctx, tenantID)
	span.SetStatus(codes.Ok, "")

	key := ""
	if et, _ := s.eventTypeRepo.FindByID(updated.EventTypeID); et != nil {
		key = et.Key
	}
	return &EventRouteServiceResult{
		EventRouteUUID: updated.EventRouteUUID.String(),
		TenantID:       updated.TenantID,
		EventTypeID:    updated.EventTypeID,
		EventTypeKey:   key,
		Channel:        updated.Channel,
		Destination:    updated.Destination,
		Enabled:        updated.Enabled,
		CreatedAt:      updated.CreatedAt,
		UpdatedAt:      updated.UpdatedAt,
	}, nil
}

func (s *eventRouteServiceImpl) Delete(ctx context.Context, tenantID int64, eventRouteUUID uuid.UUID) error {
	_, span := otel.Tracer("service").Start(ctx, "eventRoute.delete")
	defer span.End()

	route, err := s.eventRouteRepo.FindByUUID(eventRouteUUID)
	if err != nil || route == nil {
		span.SetStatus(codes.Error, "event route not found")
		return apperror.NewNotFound("event route")
	}
	if route.TenantID != tenantID {
		return apperror.NewNotFound("event route")
	}

	if err := s.eventRouteRepo.DeleteByUUID(eventRouteUUID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete event route failed")
		return err
	}

	s.writeGate.InvalidateTenant(ctx, tenantID)
	span.SetStatus(codes.Ok, "")
	return nil
}

// ---------------------------------------------------------------------------
// Management HTTP handler
// ---------------------------------------------------------------------------

// ManagementHandler handles event route and replay admin HTTP requests.
type ManagementHandler struct {
	eventRouteService EventRouteService
}

// NewManagementHandler creates a new ManagementHandler.
func NewManagementHandler(eventRouteService EventRouteService) *ManagementHandler {
	return &ManagementHandler{eventRouteService: eventRouteService}
}

// eventRouteRequestDTO is the request body for creating/updating an event route.
type eventRouteRequestDTO struct {
	EventTypeID int64  `json:"event_type_id"`
	Destination string `json:"destination"`
	Enabled     *bool  `json:"enabled"`
}

type eventRouteResponseDTO struct {
	UUID         string `json:"uuid"`
	EventTypeID  int64  `json:"event_type_id"`
	EventTypeKey string `json:"event_type_key"`
	Channel      string `json:"channel"`
	Destination  string `json:"destination"`
	Enabled      bool   `json:"enabled"`
}

// ListEventRoutes returns all broker routes for the current tenant.
//
// GET /event-routes
func (h *ManagementHandler) ListEventRoutes(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	routes, err := h.eventRouteService.ListByTenant(r.Context(), tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to list event routes", err)
		return
	}

	result := make([]eventRouteResponseDTO, len(routes))
	for i, rt := range routes {
		result[i] = eventRouteResponseDTO{
			UUID:         rt.EventRouteUUID,
			EventTypeID:  rt.EventTypeID,
			EventTypeKey: rt.EventTypeKey,
			Channel:      rt.Channel,
			Destination:  rt.Destination,
			Enabled:      rt.Enabled,
		}
	}

	resp.Success(w, result, "Event routes retrieved successfully")
}

// CreateEventRoute creates a new broker route for the tenant.
//
// POST /event-routes
func (h *ManagementHandler) CreateEventRoute(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	var req eventRouteRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if req.EventTypeID <= 0 || req.Destination == "" {
		resp.Error(w, http.StatusBadRequest, "event_type_id and destination are required")
		return
	}

	result, err := h.eventRouteService.Create(r.Context(), tenant.TenantID, req.EventTypeID, req.Destination)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create event route", err)
		return
	}

	resp.Created(w, eventRouteResponseDTO{
		UUID:         result.EventRouteUUID,
		EventTypeID:  result.EventTypeID,
		EventTypeKey: result.EventTypeKey,
		Channel:      result.Channel,
		Destination:  result.Destination,
		Enabled:      result.Enabled,
	}, "Event route created successfully")
}

// UpdateEventRoute updates an existing broker route.
//
// PUT /event-routes/{event_route_uuid}
func (h *ManagementHandler) UpdateEventRoute(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	routeUUIDStr := chi.URLParam(r, "event_route_uuid")
	routeUUID, err := uuid.Parse(routeUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid event route UUID")
		return
	}

	var req eventRouteRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	result, err := h.eventRouteService.Update(r.Context(), tenant.TenantID, routeUUID, req.Destination, enabled)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update event route", err)
		return
	}

	resp.Success(w, eventRouteResponseDTO{
		UUID:         result.EventRouteUUID,
		EventTypeID:  result.EventTypeID,
		EventTypeKey: result.EventTypeKey,
		Channel:      result.Channel,
		Destination:  result.Destination,
		Enabled:      result.Enabled,
	}, "Event route updated successfully")
}

// DeleteEventRoute deletes a broker route.
//
// DELETE /event-routes/{event_route_uuid}
func (h *ManagementHandler) DeleteEventRoute(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	routeUUIDStr := chi.URLParam(r, "event_route_uuid")
	routeUUID, err := uuid.Parse(routeUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid event route UUID")
		return
	}

	if err := h.eventRouteService.Delete(r.Context(), tenant.TenantID, routeUUID); err != nil {
		resp.HandleServiceError(w, r, "Failed to delete event route", err)
		return
	}

	resp.Success(w, nil, "Event route deleted successfully")
}

// unused import guard

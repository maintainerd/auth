package webhook

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/event"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// SubscriptionHandler manages per-endpoint event type subscriptions.
type SubscriptionHandler struct {
	endpointEventRepo WebhookEndpointEventRepository
	endpointRepo      WebhookEndpointRepository
	eventTypeRepo     event.EventTypeRepository
}

// NewSubscriptionHandler creates a new SubscriptionHandler.
func NewSubscriptionHandler(
	endpointEventRepo WebhookEndpointEventRepository,
	endpointRepo WebhookEndpointRepository,
	eventTypeRepo event.EventTypeRepository,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		endpointEventRepo: endpointEventRepo,
		endpointRepo:      endpointRepo,
		eventTypeRepo:     eventTypeRepo,
	}
}

type subscriptionRequestDTO struct {
	EventTypeUUID string `json:"event_type_uuid"`
}

type subscriptionDTO struct {
	EventTypeUUID string `json:"event_type_uuid"`
	EventTypeKey  string `json:"event_type_key"`
}

// ListSubscriptions returns the event type UUIDs this webhook endpoint subscribes to.
//
// GET /webhook-endpoints/{webhook_endpoint_uuid}/subscriptions
func (h *SubscriptionHandler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	epUUIDStr := chi.URLParam(r, "webhook_endpoint_uuid")
	epUUID, err := uuid.Parse(epUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid webhook endpoint UUID")
		return
	}

	ep, err := h.endpointRepo.FindByUUIDAndTenantID(epUUID, tenant.TenantID)
	if err != nil || ep == nil {
		resp.Error(w, http.StatusNotFound, "Webhook endpoint not found")
		return
	}

	subs, err := h.endpointEventRepo.FindByEndpointID(ep.WebhookEndpointID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to list subscriptions", err)
		return
	}

	result := make([]subscriptionDTO, 0, len(subs))
	for _, s := range subs {
		et, _ := h.eventTypeRepo.FindByID(s.EventTypeID)
		if et == nil {
			continue
		}
		result = append(result, subscriptionDTO{
			EventTypeUUID: et.EventTypeUUID.String(),
			EventTypeKey:  et.Key,
		})
	}

	resp.Success(w, result, "Subscriptions retrieved successfully")
}

// AddSubscription adds an event type subscription to a webhook endpoint.
//
// POST /webhook-endpoints/{webhook_endpoint_uuid}/subscriptions
func (h *SubscriptionHandler) AddSubscription(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	epUUIDStr := chi.URLParam(r, "webhook_endpoint_uuid")
	epUUID, err := uuid.Parse(epUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid webhook endpoint UUID")
		return
	}

	ep, err := h.endpointRepo.FindByUUIDAndTenantID(epUUID, tenant.TenantID)
	if err != nil || ep == nil {
		resp.Error(w, http.StatusNotFound, "Webhook endpoint not found")
		return
	}

	var req subscriptionRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if req.EventTypeUUID == "" {
		resp.Error(w, http.StatusBadRequest, "event_type_uuid is required")
		return
	}

	et, err := h.eventTypeRepo.FindByUUID(req.EventTypeUUID)
	if err != nil || et == nil {
		resp.Error(w, http.StatusNotFound, "Event type not found")
		return
	}
	// Tenant isolation: the event type must belong to the caller's tenant,
	// otherwise an endpoint could be subscribed to another tenant's event type.
	if et.TenantID != tenant.TenantID {
		resp.Error(w, http.StatusNotFound, "Event type not found")
		return
	}

	entry := WebhookEndpointEvent{
		WebhookEndpointID: ep.WebhookEndpointID,
		EventTypeID:       et.EventTypeID,
	}

	_, err = h.endpointEventRepo.Create(&entry)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to add subscription", err)
		return
	}

	resp.Created(w, subscriptionDTO{
		EventTypeUUID: et.EventTypeUUID.String(),
		EventTypeKey:  et.Key,
	}, "Subscription added successfully")
}

// RemoveSubscription removes an event type subscription from a webhook endpoint.
//
// DELETE /webhook-endpoints/{webhook_endpoint_uuid}/subscriptions
func (h *SubscriptionHandler) RemoveSubscription(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	epUUIDStr := chi.URLParam(r, "webhook_endpoint_uuid")
	epUUID, err := uuid.Parse(epUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid webhook endpoint UUID")
		return
	}

	ep, err := h.endpointRepo.FindByUUIDAndTenantID(epUUID, tenant.TenantID)
	if err != nil || ep == nil {
		resp.Error(w, http.StatusNotFound, "Webhook endpoint not found")
		return
	}

	var req subscriptionRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if req.EventTypeUUID == "" {
		resp.Error(w, http.StatusBadRequest, "event_type_uuid is required")
		return
	}

	et, err := h.eventTypeRepo.FindByUUID(req.EventTypeUUID)
	if err != nil || et == nil {
		resp.Error(w, http.StatusNotFound, "Event type not found")
		return
	}
	if et.TenantID != tenant.TenantID {
		resp.Error(w, http.StatusNotFound, "Event type not found")
		return
	}

	if err := h.endpointEventRepo.DeleteByEndpointIDAndEventTypeID(ep.WebhookEndpointID, et.EventTypeID); err != nil {
		resp.HandleServiceError(w, r, "Failed to remove subscription", err)
		return
	}

	resp.Success(w, nil, "Subscription removed")
}

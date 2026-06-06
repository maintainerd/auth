package webhook

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/middleware"
	resp "github.com/maintainerd/auth/internal/platform/response"
)

// SubscriptionHandler manages per-endpoint event type subscriptions.
type SubscriptionHandler struct {
	endpointEventRepo WebhookEndpointEventRepository
	endpointRepo      WebhookEndpointRepository
}

// NewSubscriptionHandler creates a new SubscriptionHandler.
func NewSubscriptionHandler(
	endpointEventRepo WebhookEndpointEventRepository,
	endpointRepo WebhookEndpointRepository,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		endpointEventRepo: endpointEventRepo,
		endpointRepo:      endpointRepo,
	}
}

type subscriptionRequestDTO struct {
	EventTypeID int64 `json:"event_type_id"`
}

type subscriptionResponseDTO struct {
	WebhookEndpointID int64  `json:"webhook_endpoint_id"`
	EventTypeID       int64  `json:"event_type_id"`
	Message           string `json:"message"`
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

	if req.EventTypeID <= 0 {
		resp.Error(w, http.StatusBadRequest, "event_type_id is required")
		return
	}

	entry := WebhookEndpointEvent{
		WebhookEndpointID: ep.WebhookEndpointID,
		EventTypeID:       req.EventTypeID,
	}

	_, err = h.endpointEventRepo.Create(&entry)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to add subscription", err)
		return
	}

	resp.Created(w, subscriptionResponseDTO{
		WebhookEndpointID: ep.WebhookEndpointID,
		EventTypeID:       req.EventTypeID,
		Message:           "Subscription added",
	}, "Subscription added successfully")
}

// RemoveSubscription removes an event type subscription from a webhook endpoint.
//
// DELETE /webhook-endpoints/{webhook_endpoint_uuid}/subscriptions/{event_type_id}
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

	if err := h.endpointEventRepo.DeleteByEndpointIDAndEventTypeID(ep.WebhookEndpointID, req.EventTypeID); err != nil {
		resp.HandleServiceError(w, r, "Failed to remove subscription", err)
		return
	}

	resp.Success(w, nil, "Subscription removed")
}

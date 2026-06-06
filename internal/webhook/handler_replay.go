package webhook

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/middleware"
	resp "github.com/maintainerd/auth/internal/platform/response"
)

const (
	maxEndpointsPerTenant = 50
)

// RateLimitAndCapMiddleware checks that a tenant hasn't exceeded the
// maximum number of webhook endpoints before allowing creation.
// Returns 429 Too Many Requests when the cap is hit.
func RateLimitAndCapMiddleware(repo WebhookEndpointRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Defensive: if the repo was not wired, do not panic — log and pass
			// through (the cap simply isn't enforced until wiring is fixed).
			if repo == nil {
				slog.Error("webhook rate limit: endpoint repo not wired; cap not enforced")
				next.ServeHTTP(w, r)
				return
			}
			tenant := middleware.AuthFromRequest(r).Tenant
			if tenant == nil {
				next.ServeHTTP(w, r)
				return
			}

			count, err := repo.CountByTenantID(tenant.TenantID)
			if err != nil {
				slog.Error("webhook rate limit: failed to count endpoints",
					"tenant_id", tenant.TenantID, "err", err)
				// Fail closed: do not allow creation when the cap cannot be verified.
				resp.Error(w, http.StatusServiceUnavailable,
					"Unable to verify webhook endpoint quota, try again later")
				return
			}

			if count >= maxEndpointsPerTenant {
				resp.Error(w, http.StatusTooManyRequests,
					"Maximum number of webhook endpoints reached for this tenant")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ReplayHandler handles replay of webhook deliveries.
type ReplayHandler struct {
	deliveryHistoryRepo DeliveryHistoryRepository
	endpointRepo        WebhookEndpointRepository
	replayFn            func(ctx context.Context, endpoint WebhookEndpoint, eventID uuid.UUID, isReplay bool) error
}

// NewReplayHandler creates a new ReplayHandler.
func NewReplayHandler(
	deliveryHistoryRepo DeliveryHistoryRepository,
	endpointRepo WebhookEndpointRepository,
	replayFn func(ctx context.Context, endpoint WebhookEndpoint, eventID uuid.UUID, isReplay bool) error,
) *ReplayHandler {
	return &ReplayHandler{
		deliveryHistoryRepo: deliveryHistoryRepo,
		endpointRepo:        endpointRepo,
		replayFn:            replayFn,
	}
}

type replayRequestDTO struct {
	EventID           string `json:"event_id"`
	WebhookEndpointID string `json:"webhook_endpoint_id,omitempty"`
}

// ReplayDelivery replays a specific delivery or re-triggers all deliveries for an event.
//
// POST /webhook-endpoints/replay
func (h *ReplayHandler) ReplayDelivery(w http.ResponseWriter, r *http.Request) {
	// Guards the unwired-route case (router passes a nil handler) so the method
	// value does not nil-panic when invoked.
	if h == nil {
		resp.Error(w, http.StatusServiceUnavailable, "Webhook replay is not available on this deployment")
		return
	}
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	var req replayRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	eventID, err := uuid.Parse(req.EventID)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid event_id UUID")
		return
	}

	var endpointUUID uuid.UUID
	if req.WebhookEndpointID != "" {
		endpointUUID, err = uuid.Parse(req.WebhookEndpointID)
		if err != nil {
			resp.Error(w, http.StatusBadRequest, "Invalid webhook_endpoint_id UUID")
			return
		}
	}

	if !uuidIsNil(endpointUUID) {
		ep, err := h.endpointRepo.FindByUUIDAndTenantID(endpointUUID, tenant.TenantID)
		if err != nil || ep == nil {
			resp.Error(w, http.StatusNotFound, "Webhook endpoint not found")
			return
		}

		if err := h.replayFn(r.Context(), *ep, eventID, true); err != nil {
			resp.Error(w, http.StatusInternalServerError, "Replay failed: "+err.Error())
			return
		}
	} else {
		// Replay to all active endpoints for the tenant
		endpoints, err := h.endpointRepo.FindActiveByTenantID(tenant.TenantID)
		if err != nil {
			resp.HandleServiceError(w, r, "Failed to find endpoints", err)
			return
		}

		replayed := 0
		for _, ep := range endpoints {
			if err := h.replayFn(r.Context(), ep, eventID, true); err != nil {
				slog.Warn("webhook replay: delivery failed",
					"endpoint", ep.WebhookEndpointID,
					"event_id", eventID,
					"err", err,
				)
				continue
			}
			replayed++
		}
		resp.Success(w, map[string]any{
			"replayed": replayed,
			"total":    len(endpoints),
			"event_id": eventID.String(),
		}, "Replay completed")
		return
	}

	resp.Success(w, nil, "Replay initiated")
}

func uuidIsNil(id uuid.UUID) bool {
	return id == uuid.Nil
}

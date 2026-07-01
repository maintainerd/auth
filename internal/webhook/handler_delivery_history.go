package webhook

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/pagination"
	resp "github.com/maintainerd/auth/internal/platform/response"
)

type DeliveryHistoryHandler struct {
	deliveryHistoryRepo DeliveryHistoryRepository
	webhookEndpointRepo WebhookEndpointRepository
}

func NewDeliveryHistoryHandler(
	deliveryHistoryRepo DeliveryHistoryRepository,
	webhookEndpointRepo WebhookEndpointRepository,
) *DeliveryHistoryHandler {
	return &DeliveryHistoryHandler{
		deliveryHistoryRepo: deliveryHistoryRepo,
		webhookEndpointRepo: webhookEndpointRepo,
	}
}

func (h *DeliveryHistoryHandler) GetDeliveries(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	endpointUUID, err := uuid.Parse(chi.URLParam(r, "webhook_endpoint_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid webhook endpoint UUID")
		return
	}

	endpoint, err := h.webhookEndpointRepo.FindByUUIDAndTenantID(endpointUUID, tenant.TenantID)
	if err != nil || endpoint == nil {
		resp.Error(w, http.StatusNotFound, "Webhook endpoint not found")
		return
	}

	pageReq := pagination.ParseQuery(r)

	result, err := h.deliveryHistoryRepo.FindByEndpointID(endpoint.WebhookEndpointID, pageReq.Limit)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to retrieve delivery history", err)
		return
	}

	rows := make([]DeliveryHistoryResponseDTO, len(result))
	for i, dh := range result {
		rows[i] = toDeliveryHistoryResponse(&dh)
	}
	resp.Success(w, rows, "Delivery history retrieved successfully")
}

func toDeliveryHistoryResponse(dh *DeliveryHistory) DeliveryHistoryResponseDTO {
	return DeliveryHistoryResponseDTO{
		DeliveryHistoryUUID: dh.DeliveryHistoryUUID,
		EventType:           dh.EventType,
		AttemptCount:        dh.AttemptCount,
		FinalStatus:         dh.FinalStatus,
		ResponseStatus:      dh.ResponseStatus,
		ResponseSummary:     dh.ResponseSummary,
		IsReplay:            dh.IsReplay,
		CreatedAt:           dh.CreatedAt,
	}
}

package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/model"
	resp "github.com/maintainerd/auth/internal/rest/response"
	"github.com/maintainerd/auth/internal/service"
)

// WebhookEndpointHandler handles HTTP requests for webhook endpoint management.
type WebhookEndpointHandler struct {
	webhookEndpointService service.WebhookEndpointService
}

// NewWebhookEndpointHandler creates a new WebhookEndpointHandler.
func NewWebhookEndpointHandler(webhookEndpointService service.WebhookEndpointService) *WebhookEndpointHandler {
	return &WebhookEndpointHandler{webhookEndpointService: webhookEndpointService}
}

// GetAll retrieves all webhook endpoints for the tenant with optional filtering and pagination.
//
// GET /webhook-endpoints
func (h *WebhookEndpointHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	var status []string
	if v := q.Get("status"); v != "" {
		status = append(status, v)
	}

	filter := dto.WebhookEndpointFilterDTO{
		Status: status,
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

	result, err := h.webhookEndpointService.GetAll(
		r.Context(), tenant.TenantID,
		filter.Status,
		filter.Page, filter.Limit,
		filter.SortBy, filter.SortOrder,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get webhook endpoints", err)
		return
	}

	response := dto.PaginatedResponseDTO[dto.WebhookEndpointResponseDTO]{
		Rows:       toWebhookEndpointResponseDTOList(result.Data),
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}

	resp.Success(w, response, "Webhook endpoints retrieved successfully")
}

// Get retrieves a specific webhook endpoint by UUID.
//
// GET /webhook-endpoints/{webhook_endpoint_uuid}
func (h *WebhookEndpointHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	webhookUUIDStr := chi.URLParam(r, "webhook_endpoint_uuid")
	webhookUUID, err := uuid.Parse(webhookUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid webhook endpoint UUID")
		return
	}

	result, err := h.webhookEndpointService.GetByUUID(r.Context(), tenant.TenantID, webhookUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Webhook endpoint not found", err)
		return
	}

	resp.Success(w, toWebhookEndpointResponseDTO(*result), "Webhook endpoint retrieved successfully")
}

// Create creates a new webhook endpoint for the tenant.
//
// POST /webhook-endpoints
func (h *WebhookEndpointHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	var req dto.WebhookEndpointCreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	status := model.StatusActive
	if req.Status != nil {
		status = *req.Status
	}

	result, err := h.webhookEndpointService.Create(
		r.Context(), tenant.TenantID,
		req.URL, req.Secret, req.Events,
		req.MaxRetries, req.TimeoutSeconds,
		req.Description, status,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to create webhook endpoint", err)
		return
	}

	resp.Created(w, toWebhookEndpointResponseDTO(*result), "Webhook endpoint created successfully")
}

// Update updates an existing webhook endpoint.
//
// PUT /webhook-endpoints/{webhook_endpoint_uuid}
func (h *WebhookEndpointHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	webhookUUIDStr := chi.URLParam(r, "webhook_endpoint_uuid")
	webhookUUID, err := uuid.Parse(webhookUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid webhook endpoint UUID")
		return
	}

	var req dto.WebhookEndpointUpdateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	status := model.StatusActive
	if req.Status != nil {
		status = *req.Status
	}

	result, err := h.webhookEndpointService.Update(
		r.Context(), tenant.TenantID, webhookUUID,
		req.URL, req.Secret, req.Events,
		req.MaxRetries, req.TimeoutSeconds,
		req.Description, status,
	)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update webhook endpoint", err)
		return
	}

	resp.Success(w, toWebhookEndpointResponseDTO(*result), "Webhook endpoint updated successfully")
}

// Delete soft-deletes a webhook endpoint.
//
// DELETE /webhook-endpoints/{webhook_endpoint_uuid}
func (h *WebhookEndpointHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	webhookUUIDStr := chi.URLParam(r, "webhook_endpoint_uuid")
	webhookUUID, err := uuid.Parse(webhookUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid webhook endpoint UUID")
		return
	}

	result, err := h.webhookEndpointService.Delete(r.Context(), tenant.TenantID, webhookUUID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to delete webhook endpoint", err)
		return
	}

	resp.Success(w, toWebhookEndpointResponseDTO(*result), "Webhook endpoint deleted successfully")
}

// UpdateStatus updates the status of a webhook endpoint.
//
// PATCH /webhook-endpoints/{webhook_endpoint_uuid}/status
func (h *WebhookEndpointHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	webhookUUIDStr := chi.URLParam(r, "webhook_endpoint_uuid")
	webhookUUID, err := uuid.Parse(webhookUUIDStr)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid webhook endpoint UUID")
		return
	}

	var req dto.WebhookEndpointUpdateStatusRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	result, err := h.webhookEndpointService.UpdateStatus(r.Context(), tenant.TenantID, webhookUUID, req.Status)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update webhook endpoint status", err)
		return
	}

	resp.Success(w, toWebhookEndpointResponseDTO(*result), "Webhook endpoint status updated successfully")
}

func toWebhookEndpointResponseDTO(we service.WebhookEndpointServiceDataResult) dto.WebhookEndpointResponseDTO {
	var lastTriggered *string
	if we.LastTriggeredAt != nil {
		formatted := we.LastTriggeredAt.Format("2006-01-02T15:04:05Z07:00")
		lastTriggered = &formatted
	}

	return dto.WebhookEndpointResponseDTO{
		WebhookEndpointID: we.WebhookEndpointUUID.String(),
		URL:               we.URL,
		Events:            we.Events,
		MaxRetries:        we.MaxRetries,
		TimeoutSeconds:    we.TimeoutSeconds,
		Status:            we.Status,
		Description:       we.Description,
		LastTriggeredAt:   lastTriggered,
		CreatedAt:         we.CreatedAt,
		UpdatedAt:         we.UpdatedAt,
	}
}

func toWebhookEndpointResponseDTOList(endpoints []service.WebhookEndpointServiceDataResult) []dto.WebhookEndpointResponseDTO {
	result := make([]dto.WebhookEndpointResponseDTO, len(endpoints))
	for i, ep := range endpoints {
		result[i] = toWebhookEndpointResponseDTO(ep)
	}
	return result
}

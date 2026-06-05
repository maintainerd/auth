package event

import (
	"encoding/json"
	"net/http"

	"github.com/maintainerd/auth/internal/platform/middleware"
	resp "github.com/maintainerd/auth/internal/platform/response"
)

// ConfigHandler handles HTTP requests for event configuration management.
type ConfigHandler struct {
	eventTypeService             EventTypeService
	tenantEventTypeConfigService TenantEventTypeConfigService
}

// NewConfigHandler creates a new ConfigHandler.
func NewConfigHandler(
	eventTypeService EventTypeService,
	tenantEventTypeConfigService TenantEventTypeConfigService,
) *ConfigHandler {
	return &ConfigHandler{
		eventTypeService:             eventTypeService,
		tenantEventTypeConfigService: tenantEventTypeConfigService,
	}
}

// eventTypeResponseDTO is the JSON representation of an event type.
type eventTypeResponseDTO struct {
	UUID        string `json:"uuid"`
	Key         string `json:"key"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Version     int    `json:"version"`
	IsActive    bool   `json:"is_active"`
}

// ListEventTypes returns all active event types available for subscription.
func (h *ConfigHandler) ListEventTypes(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	types, err := h.eventTypeService.ListActive(r.Context())
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to list event types", err)
		return
	}

	result := make([]eventTypeResponseDTO, len(types))
	for i, t := range types {
		result[i] = eventTypeResponseDTO{
			UUID:        t.EventTypeUUID.String(),
			Key:         t.Key,
			Category:    t.Category,
			Description: t.Description,
			Version:     t.Version,
			IsActive:    t.IsActive,
		}
	}

	resp.Success(w, result, "Event types retrieved successfully")
}

// tenantEventTypeRequestDTO is the request body for toggling a tenant event type.
type tenantEventTypeRequestDTO struct {
	EventTypeID int64 `json:"event_type_id"`
	Enabled     bool  `json:"enabled"`
}

// GetTenantEventTypes returns per-tenant event type configurations.
func (h *ConfigHandler) GetTenantEventTypes(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	configs, err := h.tenantEventTypeConfigService.GetByTenant(r.Context(), tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get tenant event types", err)
		return
	}

	resp.Success(w, configs, "Tenant event types retrieved successfully")
}

// SetTenantEventType enables or disables an event type for the tenant.
func (h *ConfigHandler) SetTenantEventType(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	var req tenantEventTypeRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if req.EventTypeID <= 0 {
		resp.ValidationError(w, nil)
		return
	}

	result, err := h.tenantEventTypeConfigService.SetEnabled(r.Context(), tenant.TenantID, req.EventTypeID, req.Enabled)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update tenant event type", err)
		return
	}

	resp.Success(w, result, "Tenant event type updated successfully")
}

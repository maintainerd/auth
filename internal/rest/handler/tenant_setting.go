package handler

import (
	"encoding/json"
	"net/http"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	resp "github.com/maintainerd/auth/internal/rest/response"
	"github.com/maintainerd/auth/internal/service"
)

// TenantSettingHandler handles tenant-level settings endpoints with JSONB
// sub-configs (rate_limit, audit, maintenance, feature_flags).
type TenantSettingHandler struct {
	tenantSettingService service.TenantSettingService
}

// NewTenantSettingHandler creates a new TenantSettingHandler.
func NewTenantSettingHandler(tenantSettingService service.TenantSettingService) *TenantSettingHandler {
	return &TenantSettingHandler{tenantSettingService: tenantSettingService}
}

// GetRateLimitConfig retrieves the rate limit configuration for the tenant.
//
// GET /tenant-settings/rate-limit
func (h *TenantSettingHandler) GetRateLimitConfig(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	config, err := h.tenantSettingService.GetRateLimitConfig(r.Context(), tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get rate limit config", err)
		return
	}

	resp.Success(w, dto.TenantSettingConfigResponseDTO(config), "Rate limit config retrieved successfully")
}

// UpdateRateLimitConfig updates the rate limit configuration for the tenant.
//
// PUT /tenant-settings/rate-limit
func (h *TenantSettingHandler) UpdateRateLimitConfig(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	var req dto.TenantSettingUpdateConfigRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	_, err := h.tenantSettingService.UpdateRateLimitConfig(r.Context(), tenant.TenantID, map[string]any(req))
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update rate limit config", err)
		return
	}

	config, err := h.tenantSettingService.GetRateLimitConfig(r.Context(), tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get updated config", err)
		return
	}

	resp.Success(w, dto.TenantSettingConfigResponseDTO(config), "Rate limit config updated successfully")
}

// GetAuditConfig retrieves the audit configuration for the tenant.
//
// GET /tenant-settings/audit
func (h *TenantSettingHandler) GetAuditConfig(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	config, err := h.tenantSettingService.GetAuditConfig(r.Context(), tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get audit config", err)
		return
	}

	resp.Success(w, dto.TenantSettingConfigResponseDTO(config), "Audit config retrieved successfully")
}

// UpdateAuditConfig updates the audit configuration for the tenant.
//
// PUT /tenant-settings/audit
func (h *TenantSettingHandler) UpdateAuditConfig(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	var req dto.TenantSettingUpdateConfigRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	_, err := h.tenantSettingService.UpdateAuditConfig(r.Context(), tenant.TenantID, map[string]any(req))
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update audit config", err)
		return
	}

	config, err := h.tenantSettingService.GetAuditConfig(r.Context(), tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get updated config", err)
		return
	}

	resp.Success(w, dto.TenantSettingConfigResponseDTO(config), "Audit config updated successfully")
}

// GetMaintenanceConfig retrieves the maintenance configuration for the tenant.
//
// GET /tenant-settings/maintenance
func (h *TenantSettingHandler) GetMaintenanceConfig(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	config, err := h.tenantSettingService.GetMaintenanceConfig(r.Context(), tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get maintenance config", err)
		return
	}

	resp.Success(w, dto.TenantSettingConfigResponseDTO(config), "Maintenance config retrieved successfully")
}

// UpdateMaintenanceConfig updates the maintenance configuration for the tenant.
//
// PUT /tenant-settings/maintenance
func (h *TenantSettingHandler) UpdateMaintenanceConfig(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	var req dto.TenantSettingUpdateConfigRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	_, err := h.tenantSettingService.UpdateMaintenanceConfig(r.Context(), tenant.TenantID, map[string]any(req))
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update maintenance config", err)
		return
	}

	config, err := h.tenantSettingService.GetMaintenanceConfig(r.Context(), tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get updated config", err)
		return
	}

	resp.Success(w, dto.TenantSettingConfigResponseDTO(config), "Maintenance config updated successfully")
}

// GetFeatureFlags retrieves the feature flags for the tenant.
//
// GET /tenant-settings/feature-flags
func (h *TenantSettingHandler) GetFeatureFlags(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	config, err := h.tenantSettingService.GetFeatureFlags(r.Context(), tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get feature flags", err)
		return
	}

	resp.Success(w, dto.TenantSettingConfigResponseDTO(config), "Feature flags retrieved successfully")
}

// UpdateFeatureFlags updates the feature flags for the tenant.
//
// PUT /tenant-settings/feature-flags
func (h *TenantSettingHandler) UpdateFeatureFlags(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	var req dto.TenantSettingUpdateConfigRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	_, err := h.tenantSettingService.UpdateFeatureFlags(r.Context(), tenant.TenantID, map[string]any(req))
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update feature flags", err)
		return
	}

	config, err := h.tenantSettingService.GetFeatureFlags(r.Context(), tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get updated config", err)
		return
	}

	resp.Success(w, dto.TenantSettingConfigResponseDTO(config), "Feature flags updated successfully")
}

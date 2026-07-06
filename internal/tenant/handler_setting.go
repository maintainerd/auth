package tenant

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
	"gorm.io/datatypes"
)

// TenantSettingHandler handles tenant-level settings endpoints with JSONB
// sub-configs (rate_limit, audit, maintenance).
type TenantSettingHandler struct {
	tenantSettingService TenantSettingService
	authEventService     authevent.AuthEventService
	auditLogger          auditlog.ManagementAuditLogger
}

// NewTenantSettingHandler creates a new TenantSettingHandler.
func NewTenantSettingHandler(tenantSettingService TenantSettingService, authEventService ...authevent.AuthEventService) *TenantSettingHandler {
	var eventService authevent.AuthEventService
	if len(authEventService) > 0 {
		eventService = authEventService[0]
	}
	return &TenantSettingHandler{tenantSettingService: tenantSettingService, authEventService: eventService}
}

func (h *TenantSettingHandler) SetAuditLogger(l auditlog.ManagementAuditLogger) { h.auditLogger = l }

func (h *TenantSettingHandler) logAudit(r *http.Request, tenantID int64, actorUserID *int64, action, resourceType, resourceID string, resourceUUID *uuid.UUID, changes, outcome string) {
	if h.auditLogger == nil {
		return
	}
	_ = h.auditLogger.Log(r.Context(), auditlog.LogEntry{
		TenantID:     tenantID,
		ActorUserID:  actorUserID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceUUID: resourceUUID,
		Changes:      changes,
		Outcome:      outcome,
	})
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

	resp.Success(w, TenantSettingConfigResponseDTO(config), "Rate limit config retrieved successfully")
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

	var req TenantSettingUpdateConfigRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}
	if err := req.ValidateRateLimitConfig(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	result, err := h.tenantSettingService.UpdateRateLimitConfig(r.Context(), tenant.TenantID, map[string]any(req))
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update rate limit config", err)
		return
	}

	auth := middleware.AuthFromRequest(r)
	var actorUserID *int64
	if auth.User != nil {
		actorUserID = &auth.User.UserID
	}
	changesJSON, _ := json.Marshal(map[string]any{"update": map[string]any(req), "after": result.RateLimitConfig})
	h.logAudit(r, tenant.TenantID, actorUserID, "tenant_setting.update_rate_limit", "tenant_setting", strconv.FormatInt(tenant.TenantID, 10), nil, string(changesJSON), "success")

	resp.Success(w, TenantSettingConfigResponseDTO(result.RateLimitConfig), "Rate limit config updated successfully")
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

	resp.Success(w, TenantSettingConfigResponseDTO(config), "Audit config retrieved successfully")
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

	var req TenantSettingUpdateConfigRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}
	if err := req.ValidateAuditConfig(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	result, err := h.tenantSettingService.UpdateAuditConfig(r.Context(), tenant.TenantID, map[string]any(req))
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update audit config", err)
		return
	}

	auth := middleware.AuthFromRequest(r)
	var actorUserID *int64
	if auth.User != nil {
		actorUserID = &auth.User.UserID
	}
	changesJSON, _ := json.Marshal(map[string]any{"update": map[string]any(req), "after": result.AuditConfig})
	h.logAudit(r, tenant.TenantID, actorUserID, "tenant_setting.update_audit", "tenant_setting", strconv.FormatInt(tenant.TenantID, 10), nil, string(changesJSON), "success")

	resp.Success(w, TenantSettingConfigResponseDTO(result.AuditConfig), "Audit config updated successfully")
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

	resp.Success(w, TenantSettingConfigResponseDTO(config), "Maintenance config retrieved successfully")
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

	var req TenantSettingUpdateConfigRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}
	if err := req.ValidateMaintenanceConfig(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	result, err := h.tenantSettingService.UpdateMaintenanceConfig(r.Context(), tenant.TenantID, map[string]any(req))
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to update maintenance config", err)
		return
	}

	h.logMaintenanceConfigUpdated(r, result.MaintenanceConfig)

	auth := middleware.AuthFromRequest(r)
	var actorUserID *int64
	if auth.User != nil {
		actorUserID = &auth.User.UserID
	}
	changesJSON, _ := json.Marshal(map[string]any{"update": map[string]any(req), "after": result.MaintenanceConfig})
	h.logAudit(r, tenant.TenantID, actorUserID, "tenant_setting.update_maintenance", "tenant_setting", strconv.FormatInt(tenant.TenantID, 10), nil, string(changesJSON), "success")

	resp.Success(w, TenantSettingConfigResponseDTO(result.MaintenanceConfig), "Maintenance config updated successfully")
}

func (h *TenantSettingHandler) logMaintenanceConfigUpdated(r *http.Request, config map[string]any) {
	if h.authEventService == nil {
		return
	}
	auth := middleware.AuthFromRequest(r)
	if auth.Tenant == nil {
		return
	}

	var actorUserID *int64
	if auth.User != nil && auth.User.UserID > 0 {
		actorUserID = &auth.User.UserID
	}
	userAgent := r.UserAgent()
	description := "Maintenance config updated"
	metadata, _ := json.Marshal(map[string]any{"config": config})

	h.authEventService.Log(r.Context(), authevent.AuthEventInput{
		TenantID:    auth.Tenant.TenantID,
		ActorUserID: actorUserID,
		IPAddress:   requestIP(r),
		UserAgent:   &userAgent,
		Category:    authevent.AuthEventCategorySystem,
		EventType:   authevent.AuthEventTypeMaintenanceConfigUpdated,
		Severity:    authevent.AuthEventSeverityWarn,
		Result:      authevent.AuthEventResultSuccess,
		Description: &description,
		Metadata:    datatypes.JSON(metadata),
	})
}

func requestIP(r *http.Request) string {
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		value := r.Header.Get(header)
		if value == "" {
			continue
		}
		ip := strings.TrimSpace(strings.Split(value, ",")[0])
		if net.ParseIP(ip) != nil {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && net.ParseIP(host) != nil {
		return host
	}
	if net.ParseIP(r.RemoteAddr) != nil {
		return r.RemoteAddr
	}
	return "0.0.0.0"
}

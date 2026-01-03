package resthandler

import (
	"encoding/json"
	"net/http"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/util"
)

// SecuritySettingHandler handles security configuration operations.
//
// This handler manages tenant-scoped security settings across different categories
// (general, password, session, threat, and IP configurations). All operations are
// tenant-isolated - middleware validates tenant access and stores it in the request
// context. The handler retrieves the tenant from context and delegates to the service
// layer for business logic and data persistence.
type SecuritySettingHandler struct {
	securitySettingService service.SecuritySettingService
}

// NewSecuritySettingHandler creates a new security setting handler instance.
func NewSecuritySettingHandler(securitySettingService service.SecuritySettingService) *SecuritySettingHandler {
	return &SecuritySettingHandler{
		securitySettingService: securitySettingService,
	}
}

// GetGeneralConfig retrieves general security configuration for the tenant.
//
// GET /security-settings/general
//
// Returns the current general security configuration settings for the authenticated
// tenant. The tenant is extracted from the request context (validated by middleware).
func (h *SecuritySettingHandler) GetGeneralConfig(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Fetch general configuration for the tenant
	config, err := h.securitySettingService.GetGeneralConfig(tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get general config", err.Error())
		return
	}

	// Build response
	response := dto.SecuritySettingConfigResponseDto(config)

	util.Success(w, response, "General config retrieved successfully")
}

// GetPasswordConfig retrieves password security configuration for the tenant.
//
// GET /security-settings/password
//
// Returns the current password security policy settings (complexity, expiration,
// history, etc.) for the authenticated tenant.
func (h *SecuritySettingHandler) GetPasswordConfig(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Fetch password configuration for the tenant
	config, err := h.securitySettingService.GetPasswordConfig(tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get password config", err.Error())
		return
	}

	// Build response
	response := dto.SecuritySettingConfigResponseDto(config)

	util.Success(w, response, "Password config retrieved successfully")
}

// GetSessionConfig retrieves session security configuration for the tenant.
//
// GET /security-settings/session
//
// Returns the current session management settings (timeout, concurrent sessions,
// idle timeout, etc.) for the authenticated tenant.
func (h *SecuritySettingHandler) GetSessionConfig(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Fetch session configuration for the tenant
	config, err := h.securitySettingService.GetSessionConfig(tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get session config", err.Error())
		return
	}

	// Build response
	response := dto.SecuritySettingConfigResponseDto(config)

	util.Success(w, response, "Session config retrieved successfully")
}

// GetThreatConfig retrieves threat security configuration for the tenant.
//
// GET /security-settings/threat
//
// Returns the current threat protection settings (brute force protection, rate limiting,
// suspicious activity detection, etc.) for the authenticated tenant.
func (h *SecuritySettingHandler) GetThreatConfig(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Fetch threat configuration for the tenant
	config, err := h.securitySettingService.GetThreatConfig(tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get threat config", err.Error())
		return
	}

	// Build response
	response := dto.SecuritySettingConfigResponseDto(config)

	util.Success(w, response, "Threat config retrieved successfully")
}

// GetIpConfig retrieves IP security configuration for the tenant.
//
// GET /security-settings/ip
//
// Returns the current IP-based security settings (IP whitelisting, geolocation
// restrictions, etc.) for the authenticated tenant.
func (h *SecuritySettingHandler) GetIpConfig(w http.ResponseWriter, r *http.Request) {
	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Fetch IP configuration for the tenant
	config, err := h.securitySettingService.GetIpConfig(tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get IP config", err.Error())
		return
	}

	// Build response
	response := dto.SecuritySettingConfigResponseDto(config)

	util.Success(w, response, "IP config retrieved successfully")
}

// UpdateGeneralConfig updates general security configuration for the tenant.
//
// PUT /security-settings/general
//
// Updates general security settings for the authenticated tenant. This operation
// is audited, capturing the user who made the change along with their IP address
// and user agent for compliance tracking.
func (h *SecuritySettingHandler) UpdateGeneralConfig(w http.ResponseWriter, r *http.Request) {
	// Get user from context (needed for audit tracking)
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Decode and validate request body
	var req dto.SecuritySettingUpdateConfigRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Extract client IP and user agent for audit trail
	clientIP := r.Context().Value(middleware.ClientIPKey)
	userAgentCtx := r.Context().Value(middleware.UserAgentKey)

	clientIPStr := ""
	userAgentStr := ""
	if clientIP != nil {
		clientIPStr = clientIP.(string)
	}
	if userAgentCtx != nil {
		userAgentStr = userAgentCtx.(string)
	}

	// Update general configuration (creates audit record)
	_, err := h.securitySettingService.UpdateGeneralConfig(tenant.TenantID, map[string]interface{}(req), user.UserID, clientIPStr, userAgentStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update general config", err.Error())
		return
	}

	// Fetch and return the updated configuration
	config, err := h.securitySettingService.GetGeneralConfig(tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get updated config", err.Error())
		return
	}

	// Build response
	response := dto.SecuritySettingConfigResponseDto(config)

	util.Success(w, response, "General config updated successfully")
}

// UpdatePasswordConfig updates password security configuration for the tenant.
//
// PUT /security-settings/password
//
// Updates password policy settings for the authenticated tenant (complexity requirements,
// expiration rules, history tracking, etc.). This operation is audited.
func (h *SecuritySettingHandler) UpdatePasswordConfig(w http.ResponseWriter, r *http.Request) {
	// Get user from context (needed for audit tracking)
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Decode and validate request body
	var req dto.SecuritySettingUpdateConfigRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Extract client IP and user agent for audit trail
	clientIP := r.Context().Value(middleware.ClientIPKey)
	userAgentCtx := r.Context().Value(middleware.UserAgentKey)

	clientIPStr := ""
	userAgentStr := ""
	if clientIP != nil {
		clientIPStr = clientIP.(string)
	}
	if userAgentCtx != nil {
		userAgentStr = userAgentCtx.(string)
	}

	// Update password configuration (creates audit record)
	_, err := h.securitySettingService.UpdatePasswordConfig(tenant.TenantID, map[string]interface{}(req), user.UserID, clientIPStr, userAgentStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update password config", err.Error())
		return
	}

	// Fetch and return the updated configuration
	config, err := h.securitySettingService.GetPasswordConfig(tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get updated config", err.Error())
		return
	}

	// Build response
	response := dto.SecuritySettingConfigResponseDto(config)

	util.Success(w, response, "Password config updated successfully")
}

// UpdateSessionConfig updates session security configuration for the tenant.
//
// PUT /security-settings/session
//
// Updates session management settings for the authenticated tenant (timeouts, concurrent
// session limits, idle timeout policies, etc.). This operation is audited.
func (h *SecuritySettingHandler) UpdateSessionConfig(w http.ResponseWriter, r *http.Request) {
	// Get user from context (needed for audit tracking)
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Decode and validate request body
	var req dto.SecuritySettingUpdateConfigRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Extract client IP and user agent for audit trail
	clientIP := r.Context().Value(middleware.ClientIPKey)
	userAgentCtx := r.Context().Value(middleware.UserAgentKey)

	clientIPStr := ""
	userAgentStr := ""
	if clientIP != nil {
		clientIPStr = clientIP.(string)
	}
	if userAgentCtx != nil {
		userAgentStr = userAgentCtx.(string)
	}

	// Update session configuration (creates audit record)
	_, err := h.securitySettingService.UpdateSessionConfig(tenant.TenantID, map[string]interface{}(req), user.UserID, clientIPStr, userAgentStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update session config", err.Error())
		return
	}

	// Fetch and return the updated configuration
	config, err := h.securitySettingService.GetSessionConfig(tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get updated config", err.Error())
		return
	}

	// Build response
	response := dto.SecuritySettingConfigResponseDto(config)

	util.Success(w, response, "Session config updated successfully")
}

// UpdateThreatConfig updates threat security configuration for the tenant.
//
// PUT /security-settings/threat
//
// Updates threat protection settings for the authenticated tenant (brute force detection,
// rate limiting, suspicious activity thresholds, etc.). This operation is audited.
func (h *SecuritySettingHandler) UpdateThreatConfig(w http.ResponseWriter, r *http.Request) {
	// Get user from context (needed for audit tracking)
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Decode and validate request body
	var req dto.SecuritySettingUpdateConfigRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Extract client IP and user agent for audit trail
	clientIP := r.Context().Value(middleware.ClientIPKey)
	userAgentCtx := r.Context().Value(middleware.UserAgentKey)

	clientIPStr := ""
	userAgentStr := ""
	if clientIP != nil {
		clientIPStr = clientIP.(string)
	}
	if userAgentCtx != nil {
		userAgentStr = userAgentCtx.(string)
	}

	// Update threat configuration (creates audit record)
	_, err := h.securitySettingService.UpdateThreatConfig(tenant.TenantID, map[string]interface{}(req), user.UserID, clientIPStr, userAgentStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update threat config", err.Error())
		return
	}

	// Fetch and return the updated configuration
	config, err := h.securitySettingService.GetThreatConfig(tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get updated config", err.Error())
		return
	}

	// Build response
	response := dto.SecuritySettingConfigResponseDto(config)

	util.Success(w, response, "Threat config updated successfully")
}

// UpdateIpConfig updates IP security configuration for the tenant.
//
// PUT /security-settings/ip
//
// Updates IP-based security settings for the authenticated tenant (IP whitelisting,
// geolocation restrictions, VPN/proxy detection, etc.). This operation is audited.
func (h *SecuritySettingHandler) UpdateIpConfig(w http.ResponseWriter, r *http.Request) {
	// Get user from context (needed for audit tracking)
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	// Get tenant from context (middleware already validated access)
	tenant, ok := r.Context().Value(middleware.TenantContextKey).(*model.Tenant)
	if !ok || tenant == nil {
		util.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	// Decode and validate request body
	var req dto.SecuritySettingUpdateConfigRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Extract client IP and user agent for audit trail
	clientIP := r.Context().Value(middleware.ClientIPKey)
	userAgentCtx := r.Context().Value(middleware.UserAgentKey)

	clientIPStr := ""
	userAgentStr := ""
	if clientIP != nil {
		clientIPStr = clientIP.(string)
	}
	if userAgentCtx != nil {
		userAgentStr = userAgentCtx.(string)
	}

	// Update IP configuration (creates audit record)
	_, err := h.securitySettingService.UpdateIpConfig(tenant.TenantID, map[string]interface{}(req), user.UserID, clientIPStr, userAgentStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update IP config", err.Error())
		return
	}

	// Fetch and return the updated configuration
	config, err := h.securitySettingService.GetIpConfig(tenant.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get updated config", err.Error())
		return
	}

	// Build response
	response := dto.SecuritySettingConfigResponseDto(config)

	util.Success(w, response, "IP config updated successfully")
}

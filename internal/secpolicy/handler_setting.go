package secpolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/maintainerd/auth/internal/platform/middleware"
	resp "github.com/maintainerd/auth/internal/platform/response"
)

var configDisplayLabels = map[string]string{
	"mfa":          "General",
	"password":     "Password",
	"session":      "Session",
	"threat":       "Threat",
	"lockout":      "IP",
	"registration": "Registration",
	"token":        "Token",
}

// SecuritySettingHandler handles security configuration operations.
//
// This handler manages tenant-scoped security settings across different categories
// (general, password, session, threat, and IP configurations). All operations are
// tenant-isolated - middleware validates tenant access and stores it in the request
// context. The handler retrieves the tenant from context and delegates to the service
// layer for business logic and data persistence.
type SecuritySettingHandler struct {
	securitySettingService SecuritySettingService
}

// NewSecuritySettingHandler creates a new security setting handler instance.
func NewSecuritySettingHandler(securitySettingService SecuritySettingService) *SecuritySettingHandler {
	return &SecuritySettingHandler{
		securitySettingService: securitySettingService,
	}
}

// GetMFAConfig retrieves general security configuration for the tenant.
//
// GET /security-settings/general
//
// Returns the current general security configuration settings for the authenticated
// tenant. The tenant is extracted from the request context (validated by middleware).
func (h *SecuritySettingHandler) GetMFAConfig(w http.ResponseWriter, r *http.Request) {
	h.getConfig(w, r, "mfa")
}

// GetPasswordConfig retrieves password security configuration for the tenant.
//
// GET /security-settings/password
//
// Returns the current password security policy settings (complexity, expiration,
// history, etc.) for the authenticated tenant.
func (h *SecuritySettingHandler) GetPasswordConfig(w http.ResponseWriter, r *http.Request) {
	h.getConfig(w, r, "password")
}

// GetSessionConfig retrieves session security configuration for the tenant.
//
// GET /security-settings/session
//
// Returns the current session management settings (timeout, concurrent sessions,
// idle timeout, etc.) for the authenticated tenant.
func (h *SecuritySettingHandler) GetSessionConfig(w http.ResponseWriter, r *http.Request) {
	h.getConfig(w, r, "session")
}

// GetThreatConfig retrieves threat security configuration for the tenant.
//
// GET /security-settings/threat
//
// Returns the current threat protection settings (brute force protection, rate limiting,
// suspicious activity detection, etc.) for the authenticated tenant.
func (h *SecuritySettingHandler) GetThreatConfig(w http.ResponseWriter, r *http.Request) {
	h.getConfig(w, r, "threat")
}

// GetLockoutConfig retrieves IP security configuration for the tenant.
//
// GET /security-settings/ip
//
// Returns the current IP-based security settings (IP whitelisting, geolocation
// restrictions, etc.) for the authenticated tenant.
func (h *SecuritySettingHandler) GetLockoutConfig(w http.ResponseWriter, r *http.Request) {
	h.getConfig(w, r, "lockout")
}

// UpdateMFAConfig updates general security configuration for the tenant.
//
// PUT /security-settings/general
//
// Updates general security settings for the authenticated tenant. This operation
// is audited, capturing the user who made the change along with their IP address
// and user agent for compliance tracking.
func (h *SecuritySettingHandler) UpdateMFAConfig(w http.ResponseWriter, r *http.Request) {
	h.updateConfig(w, r, "mfa")
}

// UpdatePasswordConfig updates password security configuration for the tenant.
//
// PUT /security-settings/password
//
// Updates password policy settings for the authenticated tenant (complexity requirements,
// expiration rules, history tracking, etc.). This operation is audited.
func (h *SecuritySettingHandler) UpdatePasswordConfig(w http.ResponseWriter, r *http.Request) {
	h.updateConfig(w, r, "password")
}

// UpdateSessionConfig updates session security configuration for the tenant.
//
// PUT /security-settings/session
//
// Updates session management settings for the authenticated tenant (timeouts, concurrent
// session limits, idle timeout policies, etc.). This operation is audited.
func (h *SecuritySettingHandler) UpdateSessionConfig(w http.ResponseWriter, r *http.Request) {
	h.updateConfig(w, r, "session")
}

// UpdateThreatConfig updates threat security configuration for the tenant.
//
// PUT /security-settings/threat
//
// Updates threat protection settings for the authenticated tenant (brute force detection,
// rate limiting, suspicious activity thresholds, etc.). This operation is audited.
func (h *SecuritySettingHandler) UpdateThreatConfig(w http.ResponseWriter, r *http.Request) {
	h.updateConfig(w, r, "threat")
}

// UpdateLockoutConfig updates IP security configuration for the tenant.
//
// PUT /security-settings/ip
//
// Updates IP-based security settings for the authenticated tenant (IP whitelisting,
// geolocation restrictions, VPN/proxy detection, etc.). This operation is audited.
func (h *SecuritySettingHandler) UpdateLockoutConfig(w http.ResponseWriter, r *http.Request) {
	h.updateConfig(w, r, "lockout")
}

// GetRegistrationConfig retrieves registration security configuration for the tenant.
//
// GET /security-settings/registration
func (h *SecuritySettingHandler) GetRegistrationConfig(w http.ResponseWriter, r *http.Request) {
	h.getConfig(w, r, "registration")
}

// GetTokenConfig retrieves token security configuration for the tenant.
//
// GET /security-settings/token
func (h *SecuritySettingHandler) GetTokenConfig(w http.ResponseWriter, r *http.Request) {
	h.getConfig(w, r, "token")
}

// UpdateRegistrationConfig updates registration security configuration for the tenant.
//
// PUT /security-settings/registration
func (h *SecuritySettingHandler) UpdateRegistrationConfig(w http.ResponseWriter, r *http.Request) {
	h.updateConfig(w, r, "registration")
}

// UpdateTokenConfig updates token security configuration for the tenant.
//
// PUT /security-settings/token
func (h *SecuritySettingHandler) UpdateTokenConfig(w http.ResponseWriter, r *http.Request) {
	h.updateConfig(w, r, "token")
}

func (h *SecuritySettingHandler) getConfig(w http.ResponseWriter, r *http.Request, configType string) {
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	config, err := h.fetchConfigByType(r.Context(), tenant.TenantID, configType)
	if err != nil {
		label := configDisplayLabels[configType]
		resp.HandleServiceError(w, r, "Failed to get "+label+" config", err)
		return
	}

	label := configDisplayLabels[configType]
	response := SecuritySettingConfigResponseDTO(config)
	resp.Success(w, response, label+" config retrieved successfully")
}

func (h *SecuritySettingHandler) updateConfig(w http.ResponseWriter, r *http.Request, configType string) {
	user := middleware.AuthFromRequest(r).User
	tenant := middleware.AuthFromRequest(r).Tenant
	if tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Tenant not found in context")
		return
	}

	var req SecuritySettingUpdateConfigRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	clientIPStr := middleware.ClientIPFromContext(r.Context())
	userAgentStr := middleware.UserAgentFromContext(r.Context())

	if err := h.updateConfigByType(r.Context(), tenant.TenantID, configType, map[string]any(req), user.UserID, clientIPStr, userAgentStr); err != nil {
		label := configDisplayLabels[configType]
		resp.HandleServiceError(w, r, "Failed to update "+label+" config", err)
		return
	}

	config, err := h.fetchConfigByType(r.Context(), tenant.TenantID, configType)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to get updated config", err)
		return
	}

	label := configDisplayLabels[configType]
	response := SecuritySettingConfigResponseDTO(config)
	resp.Success(w, response, label+" config updated successfully")
}

func (h *SecuritySettingHandler) fetchConfigByType(ctx context.Context, tenantID int64, configType string) (map[string]any, error) {
	switch configType {
	case "mfa":
		return h.securitySettingService.GetMFAConfig(ctx, tenantID)
	case "password":
		return h.securitySettingService.GetPasswordConfig(ctx, tenantID)
	case "session":
		return h.securitySettingService.GetSessionConfig(ctx, tenantID)
	case "threat":
		return h.securitySettingService.GetThreatConfig(ctx, tenantID)
	case "lockout":
		return h.securitySettingService.GetLockoutConfig(ctx, tenantID)
	case "registration":
		return h.securitySettingService.GetRegistrationConfig(ctx, tenantID)
	case "token":
		return h.securitySettingService.GetTokenConfig(ctx, tenantID)
	default:
		return nil, fmt.Errorf("invalid config type: %s", configType)
	}
}

func (h *SecuritySettingHandler) updateConfigByType(ctx context.Context, tenantID int64, configType string, config map[string]any, updatedBy int64, ipAddress, userAgent string) error {
	var err error
	switch configType {
	case "mfa":
		_, err = h.securitySettingService.UpdateMFAConfig(ctx, tenantID, config, updatedBy, ipAddress, userAgent)
	case "password":
		_, err = h.securitySettingService.UpdatePasswordConfig(ctx, tenantID, config, updatedBy, ipAddress, userAgent)
	case "session":
		_, err = h.securitySettingService.UpdateSessionConfig(ctx, tenantID, config, updatedBy, ipAddress, userAgent)
	case "threat":
		_, err = h.securitySettingService.UpdateThreatConfig(ctx, tenantID, config, updatedBy, ipAddress, userAgent)
	case "lockout":
		_, err = h.securitySettingService.UpdateLockoutConfig(ctx, tenantID, config, updatedBy, ipAddress, userAgent)
	case "registration":
		_, err = h.securitySettingService.UpdateRegistrationConfig(ctx, tenantID, config, updatedBy, ipAddress, userAgent)
	case "token":
		_, err = h.securitySettingService.UpdateTokenConfig(ctx, tenantID, config, updatedBy, ipAddress, userAgent)
	default:
		return fmt.Errorf("invalid config type: %s", configType)
	}
	return err
}

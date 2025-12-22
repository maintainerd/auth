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

type SecuritySettingHandler struct {
	securitySettingService service.SecuritySettingService
}

func NewSecuritySettingHandler(securitySettingService service.SecuritySettingService) *SecuritySettingHandler {
	return &SecuritySettingHandler{
		securitySettingService: securitySettingService,
	}
}

// GetGeneralConfig retrieves general security configuration
func (h *SecuritySettingHandler) GetGeneralConfig(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	config, err := h.securitySettingService.GetGeneralConfig(user.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get general config", err.Error())
		return
	}

	response := dto.SecuritySettingConfigResponseDto(config)

	util.Success(w, response, "General config retrieved successfully")
}

// GetPasswordConfig retrieves password security configuration
func (h *SecuritySettingHandler) GetPasswordConfig(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	config, err := h.securitySettingService.GetPasswordConfig(user.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get password config", err.Error())
		return
	}

	response := dto.SecuritySettingConfigResponseDto(config)

	util.Success(w, response, "Password config retrieved successfully")
}

// GetSessionConfig retrieves session security configuration
func (h *SecuritySettingHandler) GetSessionConfig(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	config, err := h.securitySettingService.GetSessionConfig(user.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get session config", err.Error())
		return
	}

	response := dto.SecuritySettingConfigResponseDto(config)

	util.Success(w, response, "Session config retrieved successfully")
}

// GetThreatConfig retrieves threat security configuration
func (h *SecuritySettingHandler) GetThreatConfig(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	config, err := h.securitySettingService.GetThreatConfig(user.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get threat config", err.Error())
		return
	}

	response := dto.SecuritySettingConfigResponseDto(config)

	util.Success(w, response, "Threat config retrieved successfully")
}

// GetIpConfig retrieves IP security configuration
func (h *SecuritySettingHandler) GetIpConfig(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	config, err := h.securitySettingService.GetIpConfig(user.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get IP config", err.Error())
		return
	}

	response := dto.SecuritySettingConfigResponseDto(config)

	util.Success(w, response, "IP config retrieved successfully")
}

// UpdateGeneralConfig updates general security configuration
func (h *SecuritySettingHandler) UpdateGeneralConfig(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	var req dto.SecuritySettingUpdateConfigRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Get IP address and user agent
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

	_, err := h.securitySettingService.UpdateGeneralConfig(user.TenantID, map[string]interface{}(req), user.UserID, clientIPStr, userAgentStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update general config", err.Error())
		return
	}

	// Return updated config
	config, err := h.securitySettingService.GetGeneralConfig(user.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get updated config", err.Error())
		return
	}

	response := dto.SecuritySettingConfigResponseDto(config)

	util.Success(w, response, "General config updated successfully")
}

// UpdatePasswordConfig updates password security configuration
func (h *SecuritySettingHandler) UpdatePasswordConfig(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	var req dto.SecuritySettingUpdateConfigRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Get IP address and user agent
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

	_, err := h.securitySettingService.UpdatePasswordConfig(user.TenantID, map[string]interface{}(req), user.UserID, clientIPStr, userAgentStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update password config", err.Error())
		return
	}

	// Return updated config
	config, err := h.securitySettingService.GetPasswordConfig(user.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get updated config", err.Error())
		return
	}

	response := dto.SecuritySettingConfigResponseDto(config)

	util.Success(w, response, "Password config updated successfully")
}

// UpdateSessionConfig updates session security configuration
func (h *SecuritySettingHandler) UpdateSessionConfig(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	var req dto.SecuritySettingUpdateConfigRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Get IP address and user agent
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

	_, err := h.securitySettingService.UpdateSessionConfig(user.TenantID, map[string]interface{}(req), user.UserID, clientIPStr, userAgentStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update session config", err.Error())
		return
	}

	// Return updated config
	config, err := h.securitySettingService.GetSessionConfig(user.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get updated config", err.Error())
		return
	}

	response := dto.SecuritySettingConfigResponseDto(config)

	util.Success(w, response, "Session config updated successfully")
}

// UpdateThreatConfig updates threat security configuration
func (h *SecuritySettingHandler) UpdateThreatConfig(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	var req dto.SecuritySettingUpdateConfigRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Get IP address and user agent
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

	_, err := h.securitySettingService.UpdateThreatConfig(user.TenantID, map[string]interface{}(req), user.UserID, clientIPStr, userAgentStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update threat config", err.Error())
		return
	}

	// Return updated config
	config, err := h.securitySettingService.GetThreatConfig(user.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get updated config", err.Error())
		return
	}

	response := dto.SecuritySettingConfigResponseDto(config)

	util.Success(w, response, "Threat config updated successfully")
}

// UpdateIpConfig updates IP security configuration
func (h *SecuritySettingHandler) UpdateIpConfig(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(middleware.UserContextKey).(*model.User)

	var req dto.SecuritySettingUpdateConfigRequestDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		util.ValidationError(w, err)
		return
	}

	// Get IP address and user agent
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

	_, err := h.securitySettingService.UpdateIpConfig(user.TenantID, map[string]interface{}(req), user.UserID, clientIPStr, userAgentStr)
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Failed to update IP config", err.Error())
		return
	}

	// Return updated config
	config, err := h.securitySettingService.GetIpConfig(user.TenantID)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to get updated config", err.Error())
		return
	}

	response := dto.SecuritySettingConfigResponseDto(config)

	util.Success(w, response, "IP config updated successfully")
}

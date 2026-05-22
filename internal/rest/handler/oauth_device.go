package handler

import (
	"encoding/json"
	"net/http"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	resp "github.com/maintainerd/auth/internal/rest/response"
	"github.com/maintainerd/auth/internal/service"
)

// OAuthDeviceHandler handles the Device Authorization Grant (RFC 8628).
type OAuthDeviceHandler struct {
	deviceService service.OAuthDeviceService
}

// NewOAuthDeviceHandler creates a new OAuthDeviceHandler.
func NewOAuthDeviceHandler(deviceService service.OAuthDeviceService) *OAuthDeviceHandler {
	return &OAuthDeviceHandler{deviceService: deviceService}
}

// Authorize handles POST /oauth/device_authorization.
func (h *OAuthDeviceHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		resp.Error(w, http.StatusBadRequest, "invalid form data")
		return
	}

	req := dto.OAuthDeviceAuthorizationRequestDTO{
		ClientID: r.FormValue("client_id"),
		Scope:    r.FormValue("scope"),
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	creds := extractOAuthClientCredentials(r, req.ClientID, r.FormValue("client_secret"))

	result, oerr := h.deviceService.Authorize(r.Context(), req, creds)
	if oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

// VerifyUserCode handles POST /oauth/device (authenticated user submits user_code).
func (h *OAuthDeviceHandler) VerifyUserCode(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}

	if err := r.ParseForm(); err != nil {
		resp.Error(w, http.StatusBadRequest, "invalid form data")
		return
	}

	req := dto.OAuthDeviceVerifyRequestDTO{
		UserCode: r.FormValue("user_code"),
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	if oerr := h.deviceService.VerifyUserCode(r.Context(), req, user.UserID); oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	resp.Success(w, nil, "device authorized")
}

// ExchangeDeviceToken handles POST /oauth/token with grant_type=urn:ietf:params:oauth:grant-type:device_code.
func (h *OAuthDeviceHandler) ExchangeDeviceToken(w http.ResponseWriter, r *http.Request) {
	req := dto.OAuthDeviceTokenRequestDTO{
		DeviceCode: r.FormValue("device_code"),
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	creds := extractOAuthClientCredentials(r, r.FormValue("client_id"), r.FormValue("client_secret"))

	result, oerr := h.deviceService.ExchangeToken(r.Context(), req, creds)
	if oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

// DenyUserCode handles POST /oauth/device/deny (authenticated user denies the request).
func (h *OAuthDeviceHandler) DenyUserCode(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}

	if err := r.ParseForm(); err != nil {
		resp.Error(w, http.StatusBadRequest, "invalid form data")
		return
	}

	req := dto.OAuthDeviceVerifyRequestDTO{
		UserCode: r.FormValue("user_code"),
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	if oerr := h.deviceService.DenyUserCode(r.Context(), req, user.UserID); oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	resp.Success(w, nil, "device authorization denied")
}

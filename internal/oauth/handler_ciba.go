package oauth

import (
	"net/http"

	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// OAuthCIBAHandler handles Client-Initiated Backchannel Authentication.
type OAuthCIBAHandler struct {
	cibaService OAuthCIBAService
}

// NewOAuthCIBAHandler creates a new OAuthCIBAHandler.
func NewOAuthCIBAHandler(cibaService OAuthCIBAService) *OAuthCIBAHandler {
	return &OAuthCIBAHandler{cibaService: cibaService}
}

// Initiate handles POST /oauth/ciba.
func (h *OAuthCIBAHandler) Initiate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		resp.Error(w, http.StatusBadRequest, "invalid form data")
		return
	}

	req := OAuthCIBARequestDTO{
		Scope:          r.FormValue("scope"),
		LoginHint:      r.FormValue("login_hint"),
		BindingMessage: r.FormValue("binding_message"),
		ClientID:       r.FormValue("client_id"),
		ClientSecret:   r.FormValue("client_secret"),
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	creds := extractOAuthClientCredentials(r, r.FormValue("client_id"), r.FormValue("client_secret"))

	result, oerr := h.cibaService.Initiate(r.Context(), req, creds)
	if oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	writeOAuthJSON(w, http.StatusOK, result)
}

// ExchangeToken handles POST /oauth/token with grant_type=urn:openid:params:grant-type:ciba.
func (h *OAuthCIBAHandler) ExchangeToken(w http.ResponseWriter, r *http.Request) {
	req := OAuthCIBATokenRequestDTO{
		AuthReqID:    r.FormValue("auth_req_id"),
		ClientID:     r.FormValue("client_id"),
		ClientSecret: r.FormValue("client_secret"),
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	creds := extractOAuthClientCredentials(r, r.FormValue("client_id"), r.FormValue("client_secret"))

	result, oerr := h.cibaService.ExchangeToken(r.Context(), req, creds)
	if oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	writeOAuthJSON(w, http.StatusOK, result)
}

// ApproveRequest handles POST /oauth/ciba/approve (authenticated user approves).
func (h *OAuthCIBAHandler) ApproveRequest(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}

	if err := r.ParseForm(); err != nil {
		resp.Error(w, http.StatusBadRequest, "invalid form data")
		return
	}

	authReqID := r.FormValue("auth_req_id")
	if authReqID == "" {
		resp.Error(w, http.StatusBadRequest, "auth_req_id is required")
		return
	}

	if oerr := h.cibaService.ApproveRequest(r.Context(), authReqID, user.UserID); oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	resp.Success(w, nil, "request approved")
}

// DenyRequest handles POST /oauth/ciba/deny (authenticated user denies).
func (h *OAuthCIBAHandler) DenyRequest(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}

	if err := r.ParseForm(); err != nil {
		resp.Error(w, http.StatusBadRequest, "invalid form data")
		return
	}

	authReqID := r.FormValue("auth_req_id")
	if authReqID == "" {
		resp.Error(w, http.StatusBadRequest, "auth_req_id is required")
		return
	}

	if oerr := h.cibaService.DenyRequest(r.Context(), authReqID, user.UserID); oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	resp.Success(w, nil, "request denied")
}

package oauth

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// OAuthRegisterHandler handles Dynamic Client Registration (RFC 7591).
type OAuthRegisterHandler struct {
	registerService OAuthRegisterService
}

// NewOAuthRegisterHandler creates a new OAuthRegisterHandler.
func NewOAuthRegisterHandler(registerService OAuthRegisterService) *OAuthRegisterHandler {
	return &OAuthRegisterHandler{registerService: registerService}
}

// Register handles POST /oauth/register (RFC 7591 §3).
//
// The RFC's "initial access token" is the caller's own access token: the route
// is mounted behind JWT auth and the client:create permission, so registration
// is an authenticated, tenant-scoped operation rather than something anyone on
// the internet can do.
func (h *OAuthRegisterHandler) Register(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := registrationTenantID(r)
	if !ok {
		resp.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req OAuthClientRegistrationRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	result, oerr := h.registerService.Register(r.Context(), req, tenantID)
	if oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	writeOAuthJSON(w, http.StatusCreated, result)
}

// Read handles GET /oauth/register/{client_id} (RFC 7592 §2.1).
func (h *OAuthRegisterHandler) Read(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := registrationTenantID(r)
	if !ok {
		resp.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}

	clientIdentifier := strings.TrimSpace(chi.URLParam(r, "client_id"))
	if clientIdentifier == "" {
		resp.Error(w, http.StatusBadRequest, "client_id is required")
		return
	}

	result, oerr := h.registerService.Read(r.Context(), clientIdentifier, tenantID)
	if oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	writeOAuthJSON(w, http.StatusOK, result)
}

// registrationTenantID resolves the tenant a registration is scoped to. It comes
// from the authenticated caller and from nowhere else — a request-body or
// query-parameter tenant would let any caller register a client into any tenant.
func registrationTenantID(r *http.Request) (int64, bool) {
	auth := middleware.AuthFromRequest(r)
	if auth.User == nil || auth.Tenant == nil || auth.Tenant.TenantID <= 0 {
		return 0, false
	}
	return auth.Tenant.TenantID, true
}

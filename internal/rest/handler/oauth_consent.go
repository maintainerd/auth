package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/middleware"
	resp "github.com/maintainerd/auth/internal/rest/response"
	"github.com/maintainerd/auth/internal/service"
)

// OAuthConsentHandler handles consent grant management endpoints (list and
// revoke existing grants).
type OAuthConsentHandler struct {
	consentService service.OAuthConsentService
}

// NewOAuthConsentHandler creates a new OAuthConsentHandler.
func NewOAuthConsentHandler(consentService service.OAuthConsentService) *OAuthConsentHandler {
	return &OAuthConsentHandler{consentService: consentService}
}

// ListGrants handles GET /oauth/consent/grants. Returns all consent grants
// for the authenticated user.
func (h *OAuthConsentHandler) ListGrants(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	grants, err := h.consentService.ListGrants(r.Context(), user.UserID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to retrieve consent grants", err)
		return
	}

	resp.Success(w, grants, "Consent grants retrieved")
}

// RevokeGrant handles DELETE /oauth/consent/grants/{grant_uuid}. Removes an
// existing consent grant.
func (h *OAuthConsentHandler) RevokeGrant(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	grantUUID, err := uuid.Parse(chi.URLParam(r, "grant_uuid"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid grant UUID")
		return
	}

	if err := h.consentService.RevokeGrant(r.Context(), grantUUID, user.UserID); err != nil {
		resp.HandleServiceError(w, r, "Failed to revoke consent grant", err)
		return
	}

	resp.Success(w, nil, "Consent grant revoked")
}

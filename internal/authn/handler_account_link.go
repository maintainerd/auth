package authn

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// AccountLinkHandler handles the social-login account-linking confirmation
// endpoint on the public port (8081).
type AccountLinkHandler struct {
	service AccountLinkRequestService
}

// NewAccountLinkHandler creates a new AccountLinkHandler.
func NewAccountLinkHandler(service AccountLinkRequestService) *AccountLinkHandler {
	return &AccountLinkHandler{service: service}
}

// AccountLinkConfirmResponseDTO is the JSON response of a successful confirmation.
type AccountLinkConfirmResponseDTO struct {
	UUID         string `json:"account_link_request_uuid"`
	Provider     string `json:"provider"`
	Status       string `json:"status"`
	LinkedUserID string `json:"-"`
}

// Confirm finalizes a pending account-link request. The caller must be
// authenticated as the existing account being linked (re-auth gate).
//
// POST /account-link/{token}/confirm
func (h *AccountLinkHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth.User == nil || auth.Tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	token := chi.URLParam(r, "token")
	if err := validateAccountLinkToken(token); err != nil {
		resp.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.service.Confirm(r.Context(), token, auth.User.UserID, auth.Tenant.TenantID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to confirm account link", err)
		return
	}

	resp.Success(w, AccountLinkConfirmResponseDTO{
		UUID:     result.UUID,
		Provider: result.ProviderName,
		Status:   "confirmed",
	}, "Account linked successfully")
}

package oauth

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// OAuthSigningKeyHandler exposes the signing-key lifecycle on the internal
// control plane.
//
// There was no surface at all: SigningKeyRepository.RetireByKID and
// MarkCompromised were implemented with zero callers, rotation only ever
// happened on a 24h timer, and the seeded `security:rotate-keys` permission
// guarded nothing. An operator who learned a signing key had leaked had no way
// to disown it short of editing the database by hand.
type OAuthSigningKeyHandler struct {
	keySvc KeyRotationService
}

// NewOAuthSigningKeyHandler creates a new OAuthSigningKeyHandler.
func NewOAuthSigningKeyHandler(keySvc KeyRotationService) *OAuthSigningKeyHandler {
	return &OAuthSigningKeyHandler{keySvc: keySvc}
}

// ListKeys handles GET /oauth/signing-keys.
func (h *OAuthSigningKeyHandler) ListKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := h.keySvc.ListKeys(r.Context(), 0)
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, "Failed to list signing keys")
		return
	}
	resp.Success(w, keys, "Signing keys retrieved")
}

// Rotate handles POST /oauth/signing-keys/rotate.
func (h *OAuthSigningKeyHandler) Rotate(w http.ResponseWriter, r *http.Request) {
	if err := h.keySvc.Rotate(r.Context()); err != nil {
		resp.Error(w, http.StatusInternalServerError, "Failed to rotate signing key")
		return
	}
	resp.Success(w, nil, "Signing key rotated")
}

// Retire handles POST /oauth/signing-keys/{kid}/retire.
func (h *OAuthSigningKeyHandler) Retire(w http.ResponseWriter, r *http.Request) {
	kid := strings.TrimSpace(chi.URLParam(r, "kid"))
	if kid == "" {
		resp.Error(w, http.StatusBadRequest, "kid is required")
		return
	}
	if err := h.keySvc.Retire(r.Context(), kid); err != nil {
		// The service refuses to retire the last active key; that is a caller
		// error, not a server fault, so it must not be reported as a 500.
		resp.Error(w, http.StatusConflict, err.Error())
		return
	}
	resp.Success(w, nil, "Signing key retired")
}

// MarkCompromised handles POST /oauth/signing-keys/{kid}/compromise.
func (h *OAuthSigningKeyHandler) MarkCompromised(w http.ResponseWriter, r *http.Request) {
	kid := strings.TrimSpace(chi.URLParam(r, "kid"))
	if kid == "" {
		resp.Error(w, http.StatusBadRequest, "kid is required")
		return
	}
	if err := h.keySvc.MarkCompromised(r.Context(), kid); err != nil {
		resp.Error(w, http.StatusInternalServerError, "Failed to mark signing key compromised")
		return
	}
	resp.Success(w, nil, "Signing key marked compromised")
}

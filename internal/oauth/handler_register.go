package oauth

import (
	"encoding/json"
	"net/http"

	resp "github.com/maintainerd/auth/internal/platform/response"
)

// OAuthRegisterHandler handles Dynamic Client Registration (RFC 7591).
type OAuthRegisterHandler struct {
	registerService OAuthRegisterService
}

// NewOAuthRegisterHandler creates a new OAuthRegisterHandler.
func NewOAuthRegisterHandler(registerService OAuthRegisterService) *OAuthRegisterHandler {
	return &OAuthRegisterHandler{registerService: registerService}
}

// Register handles POST /oauth/register.
func (h *OAuthRegisterHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req OAuthClientRegistrationRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	result, oerr := h.registerService.Register(r.Context(), req)
	if oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(result)
}

package oauth

import (
	"net/http"

	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// OAuthPARHandler handles Pushed Authorization Requests (RFC 9126).
type OAuthPARHandler struct {
	parService OAuthPARService
}

// NewOAuthPARHandler creates a new OAuthPARHandler.
func NewOAuthPARHandler(parService OAuthPARService) *OAuthPARHandler {
	return &OAuthPARHandler{parService: parService}
}

// Push handles POST /oauth/par.
func (h *OAuthPARHandler) Push(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		resp.Error(w, http.StatusBadRequest, "invalid form data")
		return
	}

	req := OAuthPARRequestDTO{
		ResponseType:        r.FormValue("response_type"),
		ClientID:            r.FormValue("client_id"),
		RedirectURI:         r.FormValue("redirect_uri"),
		Scope:               r.FormValue("scope"),
		State:               r.FormValue("state"),
		Nonce:               r.FormValue("nonce"),
		CodeChallenge:       r.FormValue("code_challenge"),
		CodeChallengeMethod: r.FormValue("code_challenge_method"),
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	creds := extractOAuthClientCredentials(r, req.ClientID, r.FormValue("client_secret"))

	result, oerr := h.parService.Push(r.Context(), req, creds)
	if oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	writeOAuthJSON(w, http.StatusOK, result)
}

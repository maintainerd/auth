package oauth

import (
	"net/http"

	resp "github.com/maintainerd/auth/internal/platform/response"
)

// OAuthTokenExchangeHandler handles the Token Exchange grant (RFC 8693).
type OAuthTokenExchangeHandler struct {
	tokenExchangeService OAuthTokenExchangeService
}

// NewOAuthTokenExchangeHandler creates a new OAuthTokenExchangeHandler.
func NewOAuthTokenExchangeHandler(tokenExchangeService OAuthTokenExchangeService) *OAuthTokenExchangeHandler {
	return &OAuthTokenExchangeHandler{tokenExchangeService: tokenExchangeService}
}

// Exchange handles POST /oauth/token with grant_type=urn:ietf:params:oauth:grant-type:token-exchange.
func (h *OAuthTokenExchangeHandler) Exchange(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		resp.Error(w, http.StatusBadRequest, "invalid form data")
		return
	}

	req := OAuthTokenExchangeRequestDTO{
		SubjectToken:       r.FormValue("subject_token"),
		SubjectTokenType:   r.FormValue("subject_token_type"),
		RequestedTokenType: r.FormValue("requested_token_type"),
		Scope:              r.FormValue("scope"),
		Audience:           r.FormValue("audience"),
		Resource:           r.FormValue("resource"),
		ActorToken:         r.FormValue("actor_token"),
		ActorTokenType:     r.FormValue("actor_token_type"),
		ClientID:           r.FormValue("client_id"),
		ClientSecret:       r.FormValue("client_secret"),
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	creds := extractOAuthClientCredentials(r, r.FormValue("client_id"), r.FormValue("client_secret"))

	result, oerr := h.tokenExchangeService.Exchange(r.Context(), req, creds)
	if oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	writeOAuthJSON(w, http.StatusOK, result)
}

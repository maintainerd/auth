package oauth

import (
	"encoding/json"
	"net/http"

	"github.com/maintainerd/auth/internal/dto"
	resp "github.com/maintainerd/auth/internal/platform/response"
	"github.com/maintainerd/auth/internal/service"
)

// OAuthTokenExchangeHandler handles the Token Exchange grant (RFC 8693).
type OAuthTokenExchangeHandler struct {
	tokenExchangeService service.OAuthTokenExchangeService
}

// NewOAuthTokenExchangeHandler creates a new OAuthTokenExchangeHandler.
func NewOAuthTokenExchangeHandler(tokenExchangeService service.OAuthTokenExchangeService) *OAuthTokenExchangeHandler {
	return &OAuthTokenExchangeHandler{tokenExchangeService: tokenExchangeService}
}

// Exchange handles POST /oauth/token with grant_type=urn:ietf:params:oauth:grant-type:token-exchange.
func (h *OAuthTokenExchangeHandler) Exchange(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		resp.Error(w, http.StatusBadRequest, "invalid form data")
		return
	}

	req := dto.OAuthTokenExchangeRequestDTO{
		SubjectToken:       r.FormValue("subject_token"),
		SubjectTokenType:   r.FormValue("subject_token_type"),
		RequestedTokenType: r.FormValue("requested_token_type"),
		Scope:              r.FormValue("scope"),
		Audience:           r.FormValue("audience"),
		Resource:           r.FormValue("resource"),
		ActorToken:         r.FormValue("actor_token"),
		ActorTokenType:     r.FormValue("actor_token_type"),
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

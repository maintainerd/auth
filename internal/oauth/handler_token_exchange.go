package oauth

import (
	"net/http"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
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
	// Q6: r.PostFormValue calls ParseForm internally — no explicit call needed.
	// B3: Resolve credentials first so Basic-auth clients populate ClientID before validation.
	creds := extractOAuthClientCredentials(r, r.PostFormValue("client_id"), r.PostFormValue("client_secret"))

	req := OAuthTokenExchangeRequestDTO{
		SubjectToken:       r.PostFormValue("subject_token"),        // B5
		SubjectTokenType:   r.PostFormValue("subject_token_type"),   // B5
		RequestedTokenType: r.PostFormValue("requested_token_type"), // B5
		Scope:              r.PostFormValue("scope"),                // B5
		Audience:           r.PostFormValue("audience"),             // B5
		Resource:           r.PostFormValue("resource"),             // B5
		ActorToken:         r.PostFormValue("actor_token"),          // B5
		ActorTokenType:     r.PostFormValue("actor_token_type"),     // B5
		ClientID:           creds.ClientID,                          // B3: from resolved creds
		ClientSecret:       creds.ClientSecret,
	}

	if err := req.Validate(); err != nil {
		apperror.NewOAuthInvalidRequest(err.Error()).WriteJSON(w) // B4
		return
	}

	result, oerr := h.tokenExchangeService.Exchange(r.Context(), req, creds)
	if oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	writeOAuthJSON(w, http.StatusOK, result)
}

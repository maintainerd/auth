package oauth

import (
	"net/http"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
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

	// Workload identity federation (section 3.21): an external OIDC token is
	// presented with subject_token_type=jwt and no client credentials. Attempt
	// the WIF path first; it returns (nil, nil) when no federation trusts the
	// token's issuer, in which case we fall through to the standard RFC 8693
	// exchange (which requires client authentication).
	subjectTokenType := r.PostFormValue("subject_token_type")
	subjectToken := r.PostFormValue("subject_token")
	if workloadIdentityExchanger != nil && subjectTokenType == subjectTokenTypeJWT && subjectToken != "" {
		// This path is keyless — no client credentials are presented — so the
		// audience the caller names is the only thing standing between it and a
		// token addressed at some other resource server. Collapse `audience` and
		// `resource` into the one target they must agree on here, before anything
		// is minted: the exchanger only ever read `audience`, so a caller that
		// named `resource` alone had its target silently dropped, and a caller
		// that named two different targets had one of them silently dropped.
		wifTarget, oerr := normalizeRequestedTarget(r.PostFormValue("audience"), r.PostFormValue("resource"))
		if oerr != nil {
			oerr.WriteJSON(w)
			return
		}

		wifResult, oerr := workloadIdentityExchanger.ExchangeWorkloadToken(r.Context(), WorkloadTokenExchangeInput{
			SubjectToken: subjectToken,
			Scope:        r.PostFormValue("scope"),
			Audience:     wifTarget,
			Resource:     r.PostFormValue("resource"),
			IPAddress:    middleware.ClientIPFromContext(r.Context()),
		})
		if oerr != nil {
			oerr.WriteJSON(w)
			return
		}
		if wifResult != nil {
			writeOAuthJSON(w, http.StatusOK, OAuthTokenExchangeResponseDTO{
				AccessToken:     wifResult.AccessToken,
				IssuedTokenType: wifResult.IssuedTokenType,
				TokenType:       wifResult.TokenType,
				ExpiresIn:       int64(wifResult.ExpiresIn),
				Scope:           wifResult.Scope,
			})
			return
		}
		// No federation matched the issuer — fall through to standard exchange.
	}

	// B3: Resolve credentials first so Basic-auth clients populate ClientID before validation.
	creds := extractOAuthClientCredentials(r, r.PostFormValue("client_id"), r.PostFormValue("client_secret"))

	req := OAuthTokenExchangeRequestDTO{
		SubjectToken:       subjectToken,                            // B5
		SubjectTokenType:   subjectTokenType,                        // B5
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

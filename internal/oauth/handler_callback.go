package oauth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cookie"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// HandleBrokerCallback handles GET /oauth/callback/{idp_identifier} — the
// upstream provider redirects back to this endpoint after the user
// authenticates. It exchanges the upstream code, provisions the user, issues a
// maintainerd authorization code for the original downstream app, and sets an
// SSO cookie so the user has a maintainerd session.
func (h *OAuthAuthorizeHandler) HandleBrokerCallback(w http.ResponseWriter, r *http.Request) {
	idpIdentifier := chi.URLParam(r, "idp_identifier")
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		resp.Error(w, http.StatusBadRequest, "code and state are required")
		return
	}

	redirectURL, accessToken, oerr := h.authorizeService.HandleCallback(r.Context(), idpIdentifier, code, state)
	if oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	if accessToken != "" {
		cookie.SetAuthCookies(w, map[string]interface{}{
			"access_token": accessToken,
			"token_type":   "Bearer",
			"expires_in":   1800,
		})
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

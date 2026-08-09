package oauth

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cookie"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// HandleBrokerCallback handles GET /oauth/callback/{idp_identifier} — the
// upstream provider redirects back to this endpoint after the user
// authenticates. It exchanges the upstream code, provisions the user, issues a
// maintainerd authorization code for the original downstream app, and sets an
// SSO cookie so the user has a maintainerd session.
func (h *OAuthAuthorizeHandler) HandleBrokerCallback(w http.ResponseWriter, r *http.Request) {
	idpIdentifier := chi.URLParam(r, "idp_identifier")
	q := r.URL.Query()
	code := q.Get("code")
	state := q.Get("state")

	// This endpoint is only ever reached by a browser redirect from the upstream
	// IdP, so on ANY failure send the user back to the identity login UI with the
	// error — never a bare JSON/API dead-end. Three failure cases:
	//   1. the IdP redirected back with an OAuth error (e.g. invalid_scope),
	//   2. no authorization code arrived, or
	//   3. the server-side code exchange failed.
	if upErr := q.Get("error"); upErr != "" {
		http.Redirect(w, r, h.authorizeService.ResolveBrokerErrorRedirect(r.Context(), idpIdentifier, state, upErr, q.Get("error_description")), http.StatusFound)
		return
	}
	if code == "" || state == "" {
		http.Redirect(w, r, h.authorizeService.ResolveBrokerErrorRedirect(r.Context(), idpIdentifier, state, "invalid_request", "the identity provider did not return an authorization code"), http.StatusFound)
		return
	}

	redirectURL, accessToken, oerr := h.authorizeService.HandleCallback(r.Context(), idpIdentifier, code, state)
	if oerr != nil {
		http.Redirect(w, r, h.authorizeService.ResolveBrokerErrorRedirect(r.Context(), idpIdentifier, state, "access_denied", "sign-in with the identity provider could not be completed"), http.StatusFound)
		return
	}

	if accessToken != "" {
		cookie.SetAuthCookies(w, map[string]interface{}{
			"access_token": accessToken,
			"token_type":   "Bearer",
			"expires_in":   1800,
		})
	}

	http.Redirect(w, r, redirectURL, http.StatusFound) // nosemgrep
}

// BrokerResumeResponseDTO is the response body for POST /oauth/broker/resume.
type BrokerResumeResponseDTO struct {
	RedirectURL string `json:"redirect_url"`
	AccessToken string `json:"access_token,omitempty"`
}

// BrokerResume handles POST /oauth/broker/resume — called after the user
// confirms an account link that was triggered by a social login email collision.
// It redeems the confirmed link token + pending broker session, issues an
// authorization code for the linked user, and returns the downstream redirect URL.
func (h *OAuthAuthorizeHandler) BrokerResume(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth.User == nil || auth.Tenant == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var body BrokerResumeRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if body.BrokerSessionUUID == "" || body.AccountLinkToken == "" {
		resp.Error(w, http.StatusBadRequest, "broker_session_uuid and account_link_token are required")
		return
	}

	result, oerr := h.authorizeService.BrokerResume(r.Context(), body, auth.User.UserID)
	if oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	if result.AccessToken != "" {
		cookie.SetAuthCookies(w, map[string]interface{}{
			"access_token": result.AccessToken,
			"token_type":   "Bearer",
			"expires_in":   1800,
		})
	}

	resp.Success(w, BrokerResumeResponseDTO{
		RedirectURL: result.RedirectURL,
		AccessToken: result.AccessToken,
	}, "")
}

package oauth

import (
	"net/http"

	resp "github.com/maintainerd/auth/internal/platform/response"
)

// OAuthSessionHandler handles RP-Initiated Logout and Back-Channel Logout.
type OAuthSessionHandler struct {
	sessionService OAuthSessionService
}

// NewOAuthSessionHandler creates a new OAuthSessionHandler.
func NewOAuthSessionHandler(sessionService OAuthSessionService) *OAuthSessionHandler {
	return &OAuthSessionHandler{sessionService: sessionService}
}

// EndSession handles GET /oauth/end_session and POST /oauth/end_session
// (OIDC Session Management 1.0 §5, RP-Initiated Logout).
func (h *OAuthSessionHandler) EndSession(w http.ResponseWriter, r *http.Request) {
	var idTokenHint, clientID, postLogoutRedirectURI, state string

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			resp.Error(w, http.StatusBadRequest, "invalid form data")
			return
		}
		idTokenHint = r.FormValue("id_token_hint")
		clientID = r.FormValue("client_id")
		postLogoutRedirectURI = r.FormValue("post_logout_redirect_uri")
		state = r.FormValue("state")
	} else {
		q := r.URL.Query()
		idTokenHint = q.Get("id_token_hint")
		clientID = q.Get("client_id")
		postLogoutRedirectURI = q.Get("post_logout_redirect_uri")
		state = q.Get("state")
	}

	req := OAuthEndSessionRequestDTO{
		IDTokenHint:           idTokenHint,
		ClientID:              clientID,
		PostLogoutRedirectURI: postLogoutRedirectURI,
		State:                 state,
	}

	redirectURI, oerr := h.sessionService.EndSession(r.Context(), req)
	if oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	if redirectURI != "" {
		http.Redirect(w, r, redirectURI, http.StatusFound)
		return
	}

	resp.Success(w, nil, "session ended")
}

// BackchannelLogout handles POST /oauth/logout/backchannel
// (OIDC Back-Channel Logout 1.0 §2.5).
func (h *OAuthSessionHandler) BackchannelLogout(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		resp.Error(w, http.StatusBadRequest, "invalid form data")
		return
	}

	req := OAuthBackchannelLogoutRequestDTO{
		LogoutToken: r.FormValue("logout_token"),
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	if oerr := h.sessionService.BackchannelLogout(r.Context(), req); oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	w.WriteHeader(http.StatusOK)
}

package handler

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/service"
)

// OAuthTokenHandler handles the OAuth 2.0 token, revocation, and
// introspection endpoints.
type OAuthTokenHandler struct {
	tokenService service.OAuthTokenService
}

// NewOAuthTokenHandler creates a new OAuthTokenHandler.
func NewOAuthTokenHandler(tokenService service.OAuthTokenService) *OAuthTokenHandler {
	return &OAuthTokenHandler{tokenService: tokenService}
}

// Token handles POST /oauth/token (RFC 6749 §4.1.3, §6, §4.4). The
// request body is application/x-www-form-urlencoded.
func (h *OAuthTokenHandler) Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		oerr := apperror.NewOAuthInvalidRequest("malformed request body")
		oerr.WriteJSON(w)
		return
	}

	req := dto.OAuthTokenRequestDTO{
		GrantType:    r.PostFormValue("grant_type"),
		Code:         r.PostFormValue("code"),
		RedirectURI:  r.PostFormValue("redirect_uri"),
		CodeVerifier: r.PostFormValue("code_verifier"),
		RefreshToken: r.PostFormValue("refresh_token"),
		Scope:        r.PostFormValue("scope"),
		ClientID:     r.PostFormValue("client_id"),
		ClientSecret: r.PostFormValue("client_secret"),
	}

	if err := req.Validate(); err != nil {
		oerr := apperror.NewOAuthInvalidRequest(err.Error())
		oerr.WriteJSON(w)
		return
	}

	// Resolve client credentials from either HTTP Basic auth or the form body.
	creds := extractClientCredentials(r, req)

	result, oerr := h.tokenService.Exchange(r.Context(), req, creds)
	if oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	writeTokenResponse(w, result)
}

// Revoke handles POST /oauth/revoke (RFC 7009). The request body is
// application/x-www-form-urlencoded.
func (h *OAuthTokenHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		oerr := apperror.NewOAuthInvalidRequest("malformed request body")
		oerr.WriteJSON(w)
		return
	}

	req := dto.OAuthRevokeRequestDTO{
		Token:         r.PostFormValue("token"),
		TokenTypeHint: r.PostFormValue("token_type_hint"),
		ClientID:      r.PostFormValue("client_id"),
		ClientSecret:  r.PostFormValue("client_secret"),
	}

	if err := req.Validate(); err != nil {
		oerr := apperror.NewOAuthInvalidRequest(err.Error())
		oerr.WriteJSON(w)
		return
	}

	creds := extractClientCredentials(r, dto.OAuthTokenRequestDTO{
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
	})

	oerr := h.tokenService.Revoke(r.Context(), req, creds)
	if oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	// RFC 7009 §2.2: respond with 200 OK and empty body.
	w.WriteHeader(http.StatusOK)
}

// Introspect handles POST /oauth/introspect (RFC 7662). The request body is
// application/x-www-form-urlencoded.
func (h *OAuthTokenHandler) Introspect(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		oerr := apperror.NewOAuthInvalidRequest("malformed request body")
		oerr.WriteJSON(w)
		return
	}

	req := dto.OAuthIntrospectRequestDTO{
		Token:         r.PostFormValue("token"),
		TokenTypeHint: r.PostFormValue("token_type_hint"),
	}

	if err := req.Validate(); err != nil {
		oerr := apperror.NewOAuthInvalidRequest(err.Error())
		oerr.WriteJSON(w)
		return
	}

	result, oerr := h.tokenService.Introspect(r.Context(), req)
	if oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	writeOAuthJSON(w, http.StatusOK, result)
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

// extractClientCredentials resolves client_id and client_secret from either
// the HTTP Basic Authorization header (RFC 6749 §2.3.1) or the form body
// (§2.3.1 alternative). HTTP Basic takes precedence.
func extractClientCredentials(r *http.Request, req dto.OAuthTokenRequestDTO) dto.OAuthClientCredentials {
	if username, password, ok := parseBasicAuth(r); ok {
		return dto.OAuthClientCredentials{
			ClientID:     username,
			ClientSecret: password,
		}
	}
	return dto.OAuthClientCredentials{
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
	}
}

// parseBasicAuth extracts the username and password from an HTTP Basic
// Authorization header. Returns ("", "", false) when no valid header is present.
func parseBasicAuth(r *http.Request) (string, string, bool) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", "", false
	}

	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return "", "", false
	}

	decoded, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return "", "", false
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	return parts[0], parts[1], true
}

// writeTokenResponse writes the token response with the required OAuth 2.0
// cache headers.
func writeTokenResponse(w http.ResponseWriter, result *dto.OAuthTokenResult) {
	resp := dto.OAuthTokenResponseDTO{
		AccessToken:  result.AccessToken,
		TokenType:    result.TokenType,
		ExpiresIn:    result.ExpiresIn,
		RefreshToken: result.RefreshToken,
		IDToken:      result.IDToken,
		Scope:        result.Scope,
	}
	writeOAuthJSON(w, http.StatusOK, resp)
}

// writeOAuthJSON writes a JSON response with OAuth-required cache headers.
func writeOAuthJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

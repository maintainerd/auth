package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/stretchr/testify/assert"
)

func TestOAuthAuthorizeHandler_HandleBrokerCallback(t *testing.T) {
	// This endpoint is only ever reached by a browser redirect from the upstream
	// IdP, so EVERY failure redirects back to the identity login UI with the
	// error — never a bare JSON/API response the user would be stranded on.
	t.Run("missing code and state redirects to login with error", func(t *testing.T) {
		h := NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{})
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/oauth/callback/google", nil), "idp_identifier", "google")
		w := httptest.NewRecorder()
		h.HandleBrokerCallback(w, r)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "https://identity.example.com/login?error=invalid_request", w.Header().Get("Location"))
	})
	t.Run("service error redirects to login carrying the service's error code", func(t *testing.T) {
		svc := &mockOAuthAuthorizeService{handleCallbackFn: func(context.Context, string, string, string) (string, string, *apperror.OAuthError) {
			return "", "", apperror.NewOAuthInvalidRequest("invalid broker session")
		}}
		h := NewOAuthAuthorizeHandler(svc)
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/oauth/callback/google?code=c&state=s", nil), "idp_identifier", "google")
		w := httptest.NewRecorder()
		h.HandleBrokerCallback(w, r)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "https://identity.example.com/login?error=invalid_request", w.Header().Get("Location"))
	})

	t.Run("opaque server_error redirects to login as generic access_denied", func(t *testing.T) {
		svc := &mockOAuthAuthorizeService{handleCallbackFn: func(context.Context, string, string, string) (string, string, *apperror.OAuthError) {
			return "", "", apperror.NewOAuthServerError("db exploded")
		}}
		h := NewOAuthAuthorizeHandler(svc)
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/oauth/callback/google?code=c&state=s", nil), "idp_identifier", "google")
		w := httptest.NewRecorder()
		h.HandleBrokerCallback(w, r)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "https://identity.example.com/login?error=access_denied", w.Header().Get("Location"))
	})
	t.Run("upstream provider error redirects to login with the provider's error", func(t *testing.T) {
		h := NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{})
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/oauth/callback/google?error=invalid_scope&state=s", nil), "idp_identifier", "google")
		w := httptest.NewRecorder()
		h.HandleBrokerCallback(w, r)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "https://identity.example.com/login?error=invalid_scope", w.Header().Get("Location"))
	})
	t.Run("success returns redirect URL", func(t *testing.T) {
		h := NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{})
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/oauth/callback/google?code=c&state=s", nil), "idp_identifier", "google")
		w := httptest.NewRecorder()
		h.HandleBrokerCallback(w, r)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "https://app.example.com/cb?code=maintainerd-code&state=abc", w.Header().Get("Location"))
	})
	t.Run("success redirects and sets NO cookie on the issuer host", func(t *testing.T) {
		// The broker callback runs on the issuer host, which the identity app
		// never reads — so it must not set a session cookie here even when the
		// service returns a token. The identity session is established same-origin
		// by the identity /callback page instead.
		h := NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{
			handleCallbackFn: func(context.Context, string, string, string) (string, string, *apperror.OAuthError) {
				return "https://app.example.com/cb?code=maintainerd-code", "access-token", nil
			},
		})
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/oauth/callback/google?code=c&state=s", nil), "idp_identifier", "google")
		w := httptest.NewRecorder()
		h.HandleBrokerCallback(w, r)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "https://app.example.com/cb?code=maintainerd-code", w.Header().Get("Location"))
		assert.Empty(t, w.Header().Values("Set-Cookie"))
	})
}

func TestNormalizeBrokerErrorCode(t *testing.T) {
	// Known OAuth error codes pass through.
	for _, code := range []string{"invalid_request", "access_denied", "invalid_scope", "server_error", "temporarily_unavailable"} {
		assert.Equal(t, code, normalizeBrokerErrorCode(code))
	}
	// Anything else — including attacker-crafted content — collapses to access_denied,
	// so nothing arbitrary is reflected into the error slot on the trusted origin.
	assert.Equal(t, "access_denied", normalizeBrokerErrorCode("<script>alert(1)</script>"))
	assert.Equal(t, "access_denied", normalizeBrokerErrorCode("call 1-800-scam"))
	assert.Equal(t, "access_denied", normalizeBrokerErrorCode(""))
}

func TestCanonicalUpstreamErrorMessage(t *testing.T) {
	// Every returned message is a fixed, curated string — never the upstream's
	// free-form error_description.
	assert.Contains(t, CanonicalUpstreamErrorMessage("access_denied"), "declined")
	assert.Contains(t, CanonicalUpstreamErrorMessage("invalid_scope"), "permissions")
	// server_error hits the generic default message.
	assert.Equal(t, "Sign-in with the identity provider could not be completed.", CanonicalUpstreamErrorMessage("server_error"))
	// An unknown / attacker code never reflects its own text: it collapses to
	// access_denied's fixed message, so the injected string never appears.
	msg := CanonicalUpstreamErrorMessage("call 1-800-scam")
	assert.NotContains(t, msg, "1-800")
	assert.Contains(t, msg, "declined")
}

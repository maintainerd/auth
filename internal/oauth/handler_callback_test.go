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
	t.Run("missing code and state returns 400", func(t *testing.T) {
		h := NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{})
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/oauth/callback/google", nil), "idp_identifier", "google")
		w := httptest.NewRecorder()
		h.HandleBrokerCallback(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("service error returns oauth error", func(t *testing.T) {
		svc := &mockOAuthAuthorizeService{handleCallbackFn: func(context.Context, string, string, string) (string, string, *apperror.OAuthError) {
			return "", "", apperror.NewOAuthInvalidRequest("invalid broker session")
		}}
		h := NewOAuthAuthorizeHandler(svc)
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/oauth/callback/google?code=c&state=s", nil), "idp_identifier", "google")
		w := httptest.NewRecorder()
		h.HandleBrokerCallback(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("success returns redirect URL", func(t *testing.T) {
		h := NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{})
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/oauth/callback/google?code=c&state=s", nil), "idp_identifier", "google")
		w := httptest.NewRecorder()
		h.HandleBrokerCallback(w, r)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "https://app.example.com/cb?code=maintainerd-code&state=abc", w.Header().Get("Location"))
	})
	t.Run("success sets SSO cookie when service returns token", func(t *testing.T) {
		h := NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{
			handleCallbackFn: func(context.Context, string, string, string) (string, string, *apperror.OAuthError) {
				return "https://app.example.com/cb?code=maintainerd-code", "access-token", nil
			},
		})
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/oauth/callback/google?code=c&state=s", nil), "idp_identifier", "google")
		w := httptest.NewRecorder()
		h.HandleBrokerCallback(w, r)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.NotEmpty(t, w.Header().Values("Set-Cookie"))
	})
}

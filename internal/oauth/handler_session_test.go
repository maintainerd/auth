package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/stretchr/testify/assert"
)

type mockOAuthSessionService struct {
	endSessionFn        func(context.Context, OAuthEndSessionRequestDTO) (string, *apperror.OAuthError)
	backchannelLogoutFn func(context.Context, OAuthBackchannelLogoutRequestDTO) *apperror.OAuthError
}

func (m *mockOAuthSessionService) EndSession(ctx context.Context, req OAuthEndSessionRequestDTO) (string, *apperror.OAuthError) {
	if m.endSessionFn != nil {
		return m.endSessionFn(ctx, req)
	}
	return "", nil
}
func (m *mockOAuthSessionService) BackchannelLogout(ctx context.Context, req OAuthBackchannelLogoutRequestDTO) *apperror.OAuthError {
	if m.backchannelLogoutFn != nil {
		return m.backchannelLogoutFn(ctx, req)
	}
	return nil
}

func TestOAuthSessionHandler_EndSession_GET(t *testing.T) {
	t.Run("success returns 200", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/oauth/end_session", nil)
		w := httptest.NewRecorder()
		NewOAuthSessionHandler(&mockOAuthSessionService{}).EndSession(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("redirect when redirect_uri is returned", func(t *testing.T) {
		svc := &mockOAuthSessionService{
			endSessionFn: func(context.Context, OAuthEndSessionRequestDTO) (string, *apperror.OAuthError) {
				return "https://app.example.com/logout-cb", nil
			},
		}
		r := httptest.NewRequest(http.MethodGet, "/oauth/end_session", nil)
		w := httptest.NewRecorder()
		NewOAuthSessionHandler(svc).EndSession(w, r)
		assert.Equal(t, http.StatusFound, w.Code)
	})

	t.Run("service error returns oauth error", func(t *testing.T) {
		svc := &mockOAuthSessionService{
			endSessionFn: func(context.Context, OAuthEndSessionRequestDTO) (string, *apperror.OAuthError) {
				return "", apperror.NewOAuthInvalidRequest("bad")
			},
		}
		r := httptest.NewRequest(http.MethodGet, "/oauth/end_session", nil)
		w := httptest.NewRecorder()
		NewOAuthSessionHandler(svc).EndSession(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestOAuthSessionHandler_EndSession_POST(t *testing.T) {
	t.Run("body parse error returns 400", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/oauth/end_session", errReader{})
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		NewOAuthSessionHandler(&mockOAuthSessionService{}).EndSession(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := formPost(t, "/oauth/end_session", url.Values{})
		w := httptest.NewRecorder()
		NewOAuthSessionHandler(&mockOAuthSessionService{}).EndSession(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestOAuthSessionHandler_BackchannelLogout(t *testing.T) {
	t.Run("body parse error returns 400", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/oauth/logout/backchannel", errReader{})
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		NewOAuthSessionHandler(&mockOAuthSessionService{}).BackchannelLogout(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := formPost(t, "/oauth/logout/backchannel", url.Values{})
		w := httptest.NewRecorder()
		NewOAuthSessionHandler(&mockOAuthSessionService{}).BackchannelLogout(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := formPost(t, "/oauth/logout/backchannel", url.Values{"logout_token": {"eyJ..."}})
		w := httptest.NewRecorder()
		NewOAuthSessionHandler(&mockOAuthSessionService{}).BackchannelLogout(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service error returns oauth error", func(t *testing.T) {
		svc := &mockOAuthSessionService{
			backchannelLogoutFn: func(context.Context, OAuthBackchannelLogoutRequestDTO) *apperror.OAuthError {
				return apperror.NewOAuthInvalidRequest("bad token")
			},
		}
		r := formPost(t, "/oauth/logout/backchannel", url.Values{"logout_token": {"bad"}})
		w := httptest.NewRecorder()
		NewOAuthSessionHandler(svc).BackchannelLogout(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

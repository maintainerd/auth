package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/stretchr/testify/assert"
)

type mockOAuthCIBAService struct {
	initiateFn      func(context.Context, OAuthCIBARequestDTO, OAuthClientCredentials) (*OAuthCIBAResponseDTO, *apperror.OAuthError)
	exchangeTokenFn func(context.Context, OAuthCIBATokenRequestDTO, OAuthClientCredentials) (*OAuthTokenResponseDTO, *apperror.OAuthError)
	approveFn       func(context.Context, string, int64) *apperror.OAuthError
	denyFn          func(context.Context, string, int64) *apperror.OAuthError
}

func (m *mockOAuthCIBAService) Initiate(ctx context.Context, req OAuthCIBARequestDTO, creds OAuthClientCredentials) (*OAuthCIBAResponseDTO, *apperror.OAuthError) {
	if m.initiateFn != nil {
		return m.initiateFn(ctx, req, creds)
	}
	return nil, nil
}
func (m *mockOAuthCIBAService) ExchangeToken(ctx context.Context, req OAuthCIBATokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResponseDTO, *apperror.OAuthError) {
	if m.exchangeTokenFn != nil {
		return m.exchangeTokenFn(ctx, req, creds)
	}
	return nil, nil
}
func (m *mockOAuthCIBAService) ApproveRequest(ctx context.Context, authReqID string, userID int64) *apperror.OAuthError {
	if m.approveFn != nil {
		return m.approveFn(ctx, authReqID, userID)
	}
	return nil
}
func (m *mockOAuthCIBAService) DenyRequest(ctx context.Context, authReqID string, userID int64) *apperror.OAuthError {
	if m.denyFn != nil {
		return m.denyFn(ctx, authReqID, userID)
	}
	return nil
}

func formPost(t *testing.T, target string, values url.Values) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, target, strings.NewReader(values.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func TestOAuthCIBAHandler_ApproveRequest(t *testing.T) {
	t.Run("body parse error returns 400", func(t *testing.T) {
		r := withUser(httptest.NewRequest(http.MethodPost, "/oauth/ciba/approve", errReader{}))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		NewOAuthCIBAHandler(&mockOAuthCIBAService{}).ApproveRequest(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("no user returns 401", func(t *testing.T) {
		r := formPost(t, "/oauth/ciba/approve", url.Values{"auth_req_id": {"req123"}})
		w := httptest.NewRecorder()
		NewOAuthCIBAHandler(&mockOAuthCIBAService{}).ApproveRequest(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("missing auth_req_id returns 400", func(t *testing.T) {
		r := withUser(formPost(t, "/oauth/ciba/approve", url.Values{}))
		w := httptest.NewRecorder()
		NewOAuthCIBAHandler(&mockOAuthCIBAService{}).ApproveRequest(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns oauth error", func(t *testing.T) {
		svc := &mockOAuthCIBAService{
			approveFn: func(context.Context, string, int64) *apperror.OAuthError {
				return apperror.NewOAuthInvalidGrant("expired")
			},
		}
		r := withUser(formPost(t, "/oauth/ciba/approve", url.Values{"auth_req_id": {"req123"}}))
		w := httptest.NewRecorder()
		NewOAuthCIBAHandler(svc).ApproveRequest(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := withUser(formPost(t, "/oauth/ciba/approve", url.Values{"auth_req_id": {"req123"}}))
		w := httptest.NewRecorder()
		NewOAuthCIBAHandler(&mockOAuthCIBAService{}).ApproveRequest(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestOAuthCIBAHandler_DenyRequest(t *testing.T) {
	t.Run("body parse error returns 400", func(t *testing.T) {
		r := withUser(httptest.NewRequest(http.MethodPost, "/oauth/ciba/deny", errReader{}))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		NewOAuthCIBAHandler(&mockOAuthCIBAService{}).DenyRequest(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("no user returns 401", func(t *testing.T) {
		r := formPost(t, "/oauth/ciba/deny", url.Values{"auth_req_id": {"req123"}})
		w := httptest.NewRecorder()
		NewOAuthCIBAHandler(&mockOAuthCIBAService{}).DenyRequest(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("missing auth_req_id returns 400", func(t *testing.T) {
		r := withUser(formPost(t, "/oauth/ciba/deny", url.Values{}))
		w := httptest.NewRecorder()
		NewOAuthCIBAHandler(&mockOAuthCIBAService{}).DenyRequest(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns oauth error", func(t *testing.T) {
		svc := &mockOAuthCIBAService{
			denyFn: func(context.Context, string, int64) *apperror.OAuthError {
				return apperror.NewOAuthInvalidGrant("expired")
			},
		}
		r := withUser(formPost(t, "/oauth/ciba/deny", url.Values{"auth_req_id": {"req123"}}))
		w := httptest.NewRecorder()
		NewOAuthCIBAHandler(svc).DenyRequest(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := withUser(formPost(t, "/oauth/ciba/deny", url.Values{"auth_req_id": {"req123"}}))
		w := httptest.NewRecorder()
		NewOAuthCIBAHandler(&mockOAuthCIBAService{}).DenyRequest(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestOAuthCIBAHandler_Initiate(t *testing.T) {
	t.Run("body parse error returns 400", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/oauth/ciba", errReader{})
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		NewOAuthCIBAHandler(&mockOAuthCIBAService{}).Initiate(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := formPost(t, "/oauth/ciba", url.Values{})
		w := httptest.NewRecorder()
		NewOAuthCIBAHandler(&mockOAuthCIBAService{}).Initiate(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns oauth error", func(t *testing.T) {
		svc := &mockOAuthCIBAService{
			initiateFn: func(context.Context, OAuthCIBARequestDTO, OAuthClientCredentials) (*OAuthCIBAResponseDTO, *apperror.OAuthError) {
				return nil, apperror.NewOAuthInvalidClient("bad client")
			},
		}
		r := formPost(t, "/oauth/ciba", url.Values{"client_id": {"app"}, "scope": {"openid"}, "login_hint": {"jane@example.com"}})
		w := httptest.NewRecorder()
		NewOAuthCIBAHandler(svc).Initiate(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockOAuthCIBAService{
			initiateFn: func(_ context.Context, req OAuthCIBARequestDTO, creds OAuthClientCredentials) (*OAuthCIBAResponseDTO, *apperror.OAuthError) {
				assert.Equal(t, "app", req.ClientID)
				assert.Equal(t, "app", creds.ClientID)
				return &OAuthCIBAResponseDTO{AuthReqID: "auth-req", ExpiresIn: 60, Interval: 5}, nil
			},
		}
		r := formPost(t, "/oauth/ciba", url.Values{"client_id": {"app"}, "scope": {"openid"}, "login_hint": {"jane@example.com"}})
		w := httptest.NewRecorder()
		NewOAuthCIBAHandler(svc).Initiate(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestOAuthCIBAHandler_ExchangeToken(t *testing.T) {
	t.Run("validation error returns 400", func(t *testing.T) {
		r := formPost(t, "/oauth/token", url.Values{})
		w := httptest.NewRecorder()
		NewOAuthCIBAHandler(&mockOAuthCIBAService{}).ExchangeToken(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns oauth error", func(t *testing.T) {
		svc := &mockOAuthCIBAService{
			exchangeTokenFn: func(context.Context, OAuthCIBATokenRequestDTO, OAuthClientCredentials) (*OAuthTokenResponseDTO, *apperror.OAuthError) {
				return nil, apperror.NewOAuthInvalidGrant("authorization pending")
			},
		}
		r := formPost(t, "/oauth/token", url.Values{"client_id": {"app"}, "auth_req_id": {"auth-req"}})
		w := httptest.NewRecorder()
		NewOAuthCIBAHandler(svc).ExchangeToken(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockOAuthCIBAService{
			exchangeTokenFn: func(_ context.Context, req OAuthCIBATokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResponseDTO, *apperror.OAuthError) {
				assert.Equal(t, "auth-req", req.AuthReqID)
				assert.Equal(t, "app", req.ClientID)
				assert.Equal(t, "app", creds.ClientID)
				return &OAuthTokenResponseDTO{AccessToken: "token", TokenType: "Bearer", ExpiresIn: 60}, nil
			},
		}
		r := formPost(t, "/oauth/token", url.Values{"client_id": {"app"}, "auth_req_id": {"auth-req"}})
		w := httptest.NewRecorder()
		NewOAuthCIBAHandler(svc).ExchangeToken(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

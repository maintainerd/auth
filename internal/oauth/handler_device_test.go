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

type mockOAuthDeviceService struct {
	authorizeFn      func(context.Context, OAuthDeviceAuthorizationRequestDTO, OAuthClientCredentials) (*OAuthDeviceAuthorizationResponseDTO, *apperror.OAuthError)
	verifyUserCodeFn func(context.Context, OAuthDeviceVerifyRequestDTO, int64) *apperror.OAuthError
	exchangeTokenFn  func(context.Context, OAuthDeviceTokenRequestDTO, OAuthClientCredentials) (*OAuthTokenResponseDTO, *apperror.OAuthError)
	denyUserCodeFn   func(context.Context, OAuthDeviceVerifyRequestDTO, int64) *apperror.OAuthError
}

func (m *mockOAuthDeviceService) Authorize(ctx context.Context, req OAuthDeviceAuthorizationRequestDTO, creds OAuthClientCredentials) (*OAuthDeviceAuthorizationResponseDTO, *apperror.OAuthError) {
	if m.authorizeFn != nil {
		return m.authorizeFn(ctx, req, creds)
	}
	return nil, nil
}
func (m *mockOAuthDeviceService) VerifyUserCode(ctx context.Context, req OAuthDeviceVerifyRequestDTO, userID int64) *apperror.OAuthError {
	if m.verifyUserCodeFn != nil {
		return m.verifyUserCodeFn(ctx, req, userID)
	}
	return nil
}
func (m *mockOAuthDeviceService) ExchangeToken(ctx context.Context, req OAuthDeviceTokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResponseDTO, *apperror.OAuthError) {
	if m.exchangeTokenFn != nil {
		return m.exchangeTokenFn(ctx, req, creds)
	}
	return nil, nil
}
func (m *mockOAuthDeviceService) DenyUserCode(ctx context.Context, req OAuthDeviceVerifyRequestDTO, userID int64) *apperror.OAuthError {
	if m.denyUserCodeFn != nil {
		return m.denyUserCodeFn(ctx, req, userID)
	}
	return nil
}

func TestOAuthDeviceHandler_Authorize(t *testing.T) {
	t.Run("body parse error returns 400", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/oauth/device_authorization", errReader{})
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		NewOAuthDeviceHandler(&mockOAuthDeviceService{}).Authorize(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := formPost(t, "/oauth/device_authorization", url.Values{})
		w := httptest.NewRecorder()
		NewOAuthDeviceHandler(&mockOAuthDeviceService{}).Authorize(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns oauth error", func(t *testing.T) {
		svc := &mockOAuthDeviceService{
			authorizeFn: func(context.Context, OAuthDeviceAuthorizationRequestDTO, OAuthClientCredentials) (*OAuthDeviceAuthorizationResponseDTO, *apperror.OAuthError) {
				return nil, apperror.NewOAuthInvalidRequest("bad")
			},
		}
		r := formPost(t, "/oauth/device_authorization", url.Values{"client_id": {"app"}})
		w := httptest.NewRecorder()
		NewOAuthDeviceHandler(svc).Authorize(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockOAuthDeviceService{
			authorizeFn: func(context.Context, OAuthDeviceAuthorizationRequestDTO, OAuthClientCredentials) (*OAuthDeviceAuthorizationResponseDTO, *apperror.OAuthError) {
				return &OAuthDeviceAuthorizationResponseDTO{DeviceCode: "dc"}, nil
			},
		}
		r := formPost(t, "/oauth/device_authorization", url.Values{"client_id": {"app"}})
		w := httptest.NewRecorder()
		NewOAuthDeviceHandler(svc).Authorize(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestOAuthDeviceHandler_VerifyUserCode(t *testing.T) {
	t.Run("body parse error returns 400", func(t *testing.T) {
		r := withUser(httptest.NewRequest(http.MethodPost, "/oauth/device", errReader{}))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		NewOAuthDeviceHandler(&mockOAuthDeviceService{}).VerifyUserCode(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("no user returns 401", func(t *testing.T) {
		r := formPost(t, "/oauth/device", url.Values{"user_code": {"ABCD-123"}})
		w := httptest.NewRecorder()
		NewOAuthDeviceHandler(&mockOAuthDeviceService{}).VerifyUserCode(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withUser(formPost(t, "/oauth/device", url.Values{}))
		w := httptest.NewRecorder()
		NewOAuthDeviceHandler(&mockOAuthDeviceService{}).VerifyUserCode(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := withUser(formPost(t, "/oauth/device", url.Values{"user_code": {"ABCD-123"}}))
		w := httptest.NewRecorder()
		NewOAuthDeviceHandler(&mockOAuthDeviceService{}).VerifyUserCode(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service error returns oauth error", func(t *testing.T) {
		svc := &mockOAuthDeviceService{
			verifyUserCodeFn: func(context.Context, OAuthDeviceVerifyRequestDTO, int64) *apperror.OAuthError {
				return apperror.NewOAuthInvalidGrant("expired")
			},
		}
		r := withUser(formPost(t, "/oauth/device", url.Values{"user_code": {"ABCD-123"}}))
		w := httptest.NewRecorder()
		NewOAuthDeviceHandler(svc).VerifyUserCode(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestOAuthDeviceHandler_ExchangeDeviceToken(t *testing.T) {
	t.Run("validation error returns 400", func(t *testing.T) {
		r := formPost(t, "/oauth/token", url.Values{})
		w := httptest.NewRecorder()
		NewOAuthDeviceHandler(&mockOAuthDeviceService{}).ExchangeDeviceToken(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns oauth error", func(t *testing.T) {
		svc := &mockOAuthDeviceService{
			exchangeTokenFn: func(context.Context, OAuthDeviceTokenRequestDTO, OAuthClientCredentials) (*OAuthTokenResponseDTO, *apperror.OAuthError) {
				return nil, apperror.NewOAuthInvalidGrant("authorization pending")
			},
		}
		r := formPost(t, "/oauth/token", url.Values{"client_id": {"app"}, "device_code": {"device-code"}})
		w := httptest.NewRecorder()
		NewOAuthDeviceHandler(svc).ExchangeDeviceToken(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockOAuthDeviceService{
			exchangeTokenFn: func(_ context.Context, req OAuthDeviceTokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResponseDTO, *apperror.OAuthError) {
				assert.Equal(t, "device-code", req.DeviceCode)
				assert.Equal(t, "app", req.ClientID)
				assert.Equal(t, "app", creds.ClientID)
				return &OAuthTokenResponseDTO{AccessToken: "token", TokenType: "Bearer", ExpiresIn: 60}, nil
			},
		}
		r := formPost(t, "/oauth/token", url.Values{"client_id": {"app"}, "device_code": {"device-code"}})
		w := httptest.NewRecorder()
		NewOAuthDeviceHandler(svc).ExchangeDeviceToken(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestOAuthDeviceHandler_DenyUserCode(t *testing.T) {
	t.Run("body parse error returns 400", func(t *testing.T) {
		r := withUser(httptest.NewRequest(http.MethodPost, "/oauth/device/deny", errReader{}))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		NewOAuthDeviceHandler(&mockOAuthDeviceService{}).DenyUserCode(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("no user returns 401", func(t *testing.T) {
		r := formPost(t, "/oauth/device/deny", url.Values{"user_code": {"ABCD-123"}})
		w := httptest.NewRecorder()
		NewOAuthDeviceHandler(&mockOAuthDeviceService{}).DenyUserCode(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := withUser(formPost(t, "/oauth/device/deny", url.Values{"user_code": {"ABCD-123"}}))
		w := httptest.NewRecorder()
		NewOAuthDeviceHandler(&mockOAuthDeviceService{}).DenyUserCode(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withUser(formPost(t, "/oauth/device/deny", url.Values{}))
		w := httptest.NewRecorder()
		NewOAuthDeviceHandler(&mockOAuthDeviceService{}).DenyUserCode(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns oauth error", func(t *testing.T) {
		svc := &mockOAuthDeviceService{
			denyUserCodeFn: func(context.Context, OAuthDeviceVerifyRequestDTO, int64) *apperror.OAuthError {
				return apperror.NewOAuthInvalidGrant("expired")
			},
		}
		r := withUser(formPost(t, "/oauth/device/deny", url.Values{"user_code": {"ABCD-123"}}))
		w := httptest.NewRecorder()
		NewOAuthDeviceHandler(svc).DenyUserCode(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

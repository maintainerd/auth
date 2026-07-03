package authn

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginHandler_RefreshToken_MissingToken(t *testing.T) {
	h := NewLoginHandler(&mockLoginService{})
	r := withSecurityCtx(newLoginRequest(t, http.MethodPost, "/refresh-token", nil))
	w := httptest.NewRecorder()
	h.RefreshToken(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoginHandler_RefreshToken_FromBody(t *testing.T) {
	var gotToken, gotSession string
	svc := &mockLoginService{
		refreshTokenFn: func(refreshToken, sessionID string) (*LoginResponseDTO, error) {
			gotToken, gotSession = refreshToken, sessionID
			return &LoginResponseDTO{AccessToken: "new-access", RefreshToken: "new-refresh"}, nil
		},
	}
	h := NewLoginHandler(svc)
	r := withSecurityCtx(newLoginRequest(t, http.MethodPost, "/refresh-token", RefreshTokenRequestDTO{RefreshToken: "rt-123"}))
	r.Header.Set("X-Session-ID", "sess-abc")
	w := httptest.NewRecorder()
	h.RefreshToken(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "rt-123", gotToken)
	assert.Equal(t, "sess-abc", gotSession) // X-Session-ID forwarded
}

func TestLoginHandler_RefreshToken_FromCookie(t *testing.T) {
	var gotToken string
	svc := &mockLoginService{
		refreshTokenFn: func(refreshToken, sessionID string) (*LoginResponseDTO, error) {
			gotToken = refreshToken
			return &LoginResponseDTO{AccessToken: "new-access"}, nil
		},
	}
	h := NewLoginHandler(svc)
	r := withSecurityCtx(newLoginRequest(t, http.MethodPost, "/refresh-token", nil))
	r.AddCookie(&http.Cookie{Name: "refresh_token", Value: "rt-cookie"})
	w := httptest.NewRecorder()
	h.RefreshToken(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "rt-cookie", gotToken) // cookie used when body is empty
	// Rotation: the new tokens MUST be re-issued as cookies so the browser
	// replaces the now-revoked refresh cookie.
	assert.NotEmpty(t, w.Result().Cookies(), "rotated tokens must be set as cookies for cookie-mode refresh")
}

func TestLoginHandler_RefreshToken_BearerDoesNotSetCookies(t *testing.T) {
	svc := &mockLoginService{
		refreshTokenFn: func(_, _ string) (*LoginResponseDTO, error) {
			return &LoginResponseDTO{AccessToken: "a", RefreshToken: "r"}, nil
		},
	}
	h := NewLoginHandler(svc)
	// Bearer mode: token in body, no cookie, no X-Token-Delivery header.
	r := withSecurityCtx(newLoginRequest(t, http.MethodPost, "/refresh-token", RefreshTokenRequestDTO{RefreshToken: "rt"}))
	w := httptest.NewRecorder()
	h.RefreshToken(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Result().Cookies(), "bearer-mode refresh must not set cookies")
}

func TestLoginHandler_RefreshToken_ExplicitCookieDelivery(t *testing.T) {
	svc := &mockLoginService{
		refreshTokenFn: func(_, _ string) (*LoginResponseDTO, error) {
			return &LoginResponseDTO{AccessToken: "a", RefreshToken: "r"}, nil
		},
	}
	h := NewLoginHandler(svc)
	// Body token but client explicitly opts into cookie delivery.
	r := withSecurityCtx(newLoginRequest(t, http.MethodPost, "/refresh-token", RefreshTokenRequestDTO{RefreshToken: "rt"}))
	r.Header.Set("X-Token-Delivery", "cookie")
	w := httptest.NewRecorder()
	h.RefreshToken(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Result().Cookies(), "X-Token-Delivery: cookie must set cookies")
}

func TestLoginHandler_RefreshToken_BodyTakesPrecedenceOverCookie(t *testing.T) {
	var gotToken string
	svc := &mockLoginService{
		refreshTokenFn: func(refreshToken, _ string) (*LoginResponseDTO, error) {
			gotToken = refreshToken
			return &LoginResponseDTO{AccessToken: "x"}, nil
		},
	}
	h := NewLoginHandler(svc)
	r := withSecurityCtx(newLoginRequest(t, http.MethodPost, "/refresh-token", RefreshTokenRequestDTO{RefreshToken: "from-body"}))
	r.AddCookie(&http.Cookie{Name: "refresh_token", Value: "from-cookie"})
	w := httptest.NewRecorder()
	h.RefreshToken(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "from-body", gotToken)
}

func TestLoginHandler_RefreshToken_ServiceError(t *testing.T) {
	svc := &mockLoginService{
		refreshTokenFn: func(_, _ string) (*LoginResponseDTO, error) {
			return nil, apperror.NewUnauthorized("invalid or expired refresh token")
		},
	}
	h := NewLoginHandler(svc)
	r := withSecurityCtx(newLoginRequest(t, http.MethodPost, "/refresh-token", RefreshTokenRequestDTO{RefreshToken: "rt"}))
	w := httptest.NewRecorder()
	h.RefreshToken(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoginHandler_RefreshToken_Success(t *testing.T) {
	svc := &mockLoginService{
		refreshTokenFn: func(_, _ string) (*LoginResponseDTO, error) {
			return &LoginResponseDTO{AccessToken: "a", IDToken: "i", RefreshToken: "r"}, nil
		},
	}
	h := NewLoginHandler(svc)
	r := withSecurityCtx(newLoginRequest(t, http.MethodPost, "/refresh-token", RefreshTokenRequestDTO{RefreshToken: "rt"}))
	w := httptest.NewRecorder()
	h.RefreshToken(w, r)
	require.Equal(t, http.StatusOK, w.Code)
}

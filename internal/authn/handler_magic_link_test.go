package authn

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/platform/signedurl"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockMagicLinkService struct {
	sendMagicLinkFn      func(email string, clientID, providerID *string, isInternal bool) (*SendMagicLinkResponseDTO, error)
	loginWithMagicLinkFn func(token, clientID, providerID string) (*LoginResponseDTO, error)
}

func (m *mockMagicLinkService) SendMagicLink(_ context.Context, email string, clientID, providerID *string, isInternal bool) (*SendMagicLinkResponseDTO, error) {
	if m.sendMagicLinkFn != nil {
		return m.sendMagicLinkFn(email, clientID, providerID, isInternal)
	}
	return nil, nil
}

func (m *mockMagicLinkService) LoginWithMagicLink(_ context.Context, token string, clientID, tenantID *string) (*LoginResponseDTO, error) {
	if m.loginWithMagicLinkFn != nil {
		cid := ""
		if clientID != nil {
			cid = *clientID
		}
		pid := ""
		if tenantID != nil {
			pid = *tenantID
		}
		return m.loginWithMagicLinkFn(token, cid, pid)
	}
	return nil, nil
}

func (m *mockMagicLinkService) SetLoginCoordinator(MagicLinkLoginCoordinator) {}

func magicLinkJSONReq(t *testing.T, method, url string, body any) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	r := httptest.NewRequest(method, url, &buf)
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestMagicLinkHandler_SendMagicLinkPublic(t *testing.T) {
	t.Run("missing client_id returns 400", func(t *testing.T) {
		r := withSecurityCtx(httptest.NewRequest(http.MethodPost, "/magic-link/send?provider_id=idp", nil))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(&mockMagicLinkService{}).SendMagicLinkPublic(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing provider_id returns 400", func(t *testing.T) {
		r := withSecurityCtx(httptest.NewRequest(http.MethodPost, "/magic-link/send?client_id=app", nil))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(&mockMagicLinkService{}).SendMagicLinkPublic(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		r := withSecurityCtx(httptest.NewRequest(http.MethodPost, "/magic-link/send?client_id=app&provider_id=idp",
			bytes.NewBufferString(`{bad json}`)))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		NewMagicLinkHandler(&mockMagicLinkService{}).SendMagicLinkPublic(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withSecurityCtx(magicLinkJSONReq(t, http.MethodPost, "/magic-link/send?client_id=app&provider_id=idp",
			map[string]string{"email": ""}))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(&mockMagicLinkService{}).SendMagicLinkPublic(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockMagicLinkService{
			sendMagicLinkFn: func(email string, clientID, providerID *string, isInternal bool) (*SendMagicLinkResponseDTO, error) {
				return nil, errors.New("db error")
			},
		}
		r := withSecurityCtx(magicLinkJSONReq(t, http.MethodPost, "/magic-link/send?client_id=app&provider_id=idp",
			map[string]string{"email": "user@example.com"}))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(svc).SendMagicLinkPublic(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockMagicLinkService{
			sendMagicLinkFn: func(email string, clientID, providerID *string, isInternal bool) (*SendMagicLinkResponseDTO, error) {
				return &SendMagicLinkResponseDTO{Message: "sent", Success: true}, nil
			},
		}
		r := withSecurityCtx(magicLinkJSONReq(t, http.MethodPost, "/magic-link/send?client_id=app&provider_id=idp",
			map[string]string{"email": "user@example.com"}))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(svc).SendMagicLinkPublic(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service returns unauthorized error", func(t *testing.T) {
		svc := &mockMagicLinkService{
			sendMagicLinkFn: func(email string, clientID, providerID *string, isInternal bool) (*SendMagicLinkResponseDTO, error) {
				return nil, apperror.NewUnauthorized("invalid")
			},
		}
		r := withSecurityCtx(magicLinkJSONReq(t, http.MethodPost, "/magic-link/send?client_id=app&provider_id=idp",
			map[string]string{"email": "user@example.com"}))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(svc).SendMagicLinkPublic(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("unknown email returns 404", func(t *testing.T) {
		svc := &mockMagicLinkService{
			sendMagicLinkFn: func(email string, clientID, providerID *string, isInternal bool) (*SendMagicLinkResponseDTO, error) {
				return nil, apperror.NewNotFound("no account found with that email address")
			},
		}
		r := withSecurityCtx(magicLinkJSONReq(t, http.MethodPost, "/magic-link/send?client_id=app&provider_id=idp",
			map[string]string{"email": "missing@example.com"}))
		w := httptest.NewRecorder()

		NewMagicLinkHandler(svc).SendMagicLinkPublic(w, r)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "no account found with that email address")
	})
}

func magicLinkSignedQuery(t *testing.T, params map[string]string) string {
	t.Helper()
	signed, err := signedurl.GenerateSignedURL("http://x", params, 10*time.Minute)
	require.NoError(t, err)
	parsed, err := url.Parse(signed)
	require.NoError(t, err)
	return parsed.RawQuery
}

func TestMagicLinkHandler_VerifyMagicLink(t *testing.T) {
	t.Run("missing signature returns 400", func(t *testing.T) {
		r := withSecurityCtx(httptest.NewRequest(http.MethodPost, "/magic-link/verify?token=abc&client_id=app", nil))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(&mockMagicLinkService{}).VerifyMagicLink(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing token returns 400", func(t *testing.T) {
		q := magicLinkSignedQuery(t, map[string]string{"client_id": "app"})
		r := withSecurityCtx(httptest.NewRequest(http.MethodPost, "/magic-link/verify?"+q, nil))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(&mockMagicLinkService{}).VerifyMagicLink(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("MFA challenge returns without session cookies", func(t *testing.T) {
		challenge := "challenge-token"
		svc := &mockMagicLinkService{
			loginWithMagicLinkFn: func(token, clientID, tenantID string) (*LoginResponseDTO, error) {
				return &LoginResponseDTO{
					MFARequired:       true,
					MFAChallengeToken: &challenge,
					MFAAllowedMethods: []string{"totp"},
				}, nil
			},
		}
		q := magicLinkSignedQuery(t, map[string]string{"token": "abc123", "client_id": "app"})
		r := withSecurityCtx(httptest.NewRequest(http.MethodPost, "/magic-link/verify?"+q, nil))
		w := httptest.NewRecorder()

		NewMagicLinkHandler(svc).VerifyMagicLink(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, w.Header().Values("Set-Cookie"))
		assert.Contains(t, w.Body.String(), "mfa_required")
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockMagicLinkService{
			loginWithMagicLinkFn: func(token, clientID, tenantID string) (*LoginResponseDTO, error) {
				return nil, errors.New("expired")
			},
		}
		q := magicLinkSignedQuery(t, map[string]string{"token": "abc123", "client_id": "app"})
		r := withSecurityCtx(httptest.NewRequest(http.MethodPost, "/magic-link/verify?"+q, nil))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(svc).VerifyMagicLink(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockMagicLinkService{
			loginWithMagicLinkFn: func(token, clientID, tenantID string) (*LoginResponseDTO, error) {
				return &LoginResponseDTO{AccessToken: "at"}, nil
			},
		}
		q := magicLinkSignedQuery(t, map[string]string{"token": "abc123", "client_id": "app"})
		r := withSecurityCtx(httptest.NewRequest(http.MethodPost, "/magic-link/verify?"+q, nil))
		w := httptest.NewRecorder()
		NewMagicLinkHandler(svc).VerifyMagicLink(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func lockedRateLimiterML(t *testing.T, identifier string) func() {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	security.InitRateLimiter(rdb)
	require.NoError(t, mr.Set("rl:lock:"+identifier, "1"))
	return func() {
		security.InitRateLimiter(nil)
		_ = rdb.Close()
		mr.Close()
	}
}

func TestMagicLinkHandler_HandleSendMagicLink_RateLimited(t *testing.T) {
	email := "ratelimited-ml@example.com"
	cleanup := lockedRateLimiterML(t, email)
	defer cleanup()

	r := withSecurityCtx(magicLinkJSONReq(t, http.MethodPost, "/magic-link/send?client_id=app&provider_id=idp",
		map[string]string{"email": email}))
	w := httptest.NewRecorder()
	NewMagicLinkHandler(&mockMagicLinkService{}).SendMagicLinkPublic(w, r)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestMagicLinkHandler_VerifyMagicLink_RateLimited(t *testing.T) {
	cleanup := lockedRateLimiterML(t, "127.0.0.1")
	defer cleanup()

	q := magicLinkSignedQuery(t, map[string]string{"token": "abc123", "client_id": "app"})
	r := withSecurityCtx(httptest.NewRequest(http.MethodPost, "/magic-link/verify?"+q, nil))
	w := httptest.NewRecorder()
	NewMagicLinkHandler(&mockMagicLinkService{}).VerifyMagicLink(w, r)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

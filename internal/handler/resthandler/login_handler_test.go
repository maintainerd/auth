package resthandler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newLoginRequest(t *testing.T, method, url string, body any) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	r := httptest.NewRequest(method, url, &buf)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("User-Agent", "Mozilla/5.0 (test)")
	return r
}

func withSecurityCtx(r *http.Request) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, middleware.ClientIPKey, "127.0.0.1")
	ctx = context.WithValue(ctx, middleware.UserAgentKey, "Mozilla/5.0 (test)")
	ctx = context.WithValue(ctx, middleware.RequestIDKey, "req-001")
	return r.WithContext(ctx)
}

// withNonStringSecurityCtx injects a non-string value at ClientIPKey so that
// the v.(string) type assertion inside extractSecurityContext returns ok=false,
// covering the `return ""` fallback branch (line 41).
func withNonStringSecurityCtx(r *http.Request) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.ClientIPKey, 42) // int, not string
	ctx = context.WithValue(ctx, middleware.UserAgentKey, "Mozilla/5.0 (test)")
	ctx = context.WithValue(ctx, middleware.RequestIDKey, "req-999")
	return r.WithContext(ctx)
}

// ---------------------------------------------------------------------------
// extractSecurityContext
// ---------------------------------------------------------------------------

func TestExtractSecurityContext_NonStringValue(t *testing.T) {
	// Calling LoginPublic (which uses extractSecurityContext) with a non-string
	// value at ClientIPKey triggers the ok==false branch → strVal returns "".
	// The handler continues normally; we just care that no panic occurs and the
	// request is processed (validation still fails because client_id is missing).
	h := NewLoginHandler(&mockLoginService{})
	r := withNonStringSecurityCtx(newLoginRequest(t, http.MethodPost, "/public/login",
		map[string]string{"username": "u", "password": "p"}))
	w := httptest.NewRecorder()
	h.LoginPublic(w, r)
	// client_id is missing → LoginQueryDto.Validate fails → 400
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// LoginPublic
// ---------------------------------------------------------------------------

func TestLoginHandler_LoginPublic_BodyValidationError(t *testing.T) {
	// Valid JSON that fails LoginRequestDto.Validate() (empty username → Required fails)
	// covers lines 113-128.
	h := NewLoginHandler(&mockLoginService{})
	r := withSecurityCtx(newLoginRequest(t, http.MethodPost, "/public/login?client_id=c1&provider_id=p1",
		map[string]string{"username": "", "password": "pass1"}))
	w := httptest.NewRecorder()
	h.LoginPublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginHandler_LoginPublic_MissingClientID(t *testing.T) {
	h := NewLoginHandler(&mockLoginService{})
	r := withSecurityCtx(newLoginRequest(t, http.MethodPost, "/public/login", map[string]string{
		"username": "user1", "password": "pass1",
	}))
	w := httptest.NewRecorder()
	h.LoginPublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginHandler_LoginPublic_EmptyUserAgent(t *testing.T) {
	h := NewLoginHandler(&mockLoginService{})
	r := httptest.NewRequest(http.MethodPost, "/public/login?client_id=c1&provider_id=p1",
		bytes.NewBufferString(`{"username":"u","password":"p"}`))
	r.Header.Set("Content-Type", "application/json")
	// no User-Agent — ValidateUserAgent returns false
	ctx := context.WithValue(r.Context(), middleware.ClientIPKey, "127.0.0.1")
	ctx = context.WithValue(ctx, middleware.UserAgentKey, "")
	ctx = context.WithValue(ctx, middleware.RequestIDKey, "req-002")
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	h.LoginPublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginHandler_LoginPublic_InvalidBody(t *testing.T) {
	h := NewLoginHandler(&mockLoginService{})
	r := httptest.NewRequest(http.MethodPost, "/public/login?client_id=c1&provider_id=p1",
		bytes.NewBufferString(`not-json`))
	r.Header.Set("Content-Type", "application/json")
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.LoginPublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginHandler_LoginPublic_ServiceError(t *testing.T) {
	svc := &mockLoginService{
		loginPublicFn: func(u, p, c, pr string) (*dto.LoginResponseDto, error) {
			return nil, assert.AnError
		},
	}
	h := NewLoginHandler(svc)
	r := withSecurityCtx(newLoginRequest(t, http.MethodPost, "/public/login?client_id=c1&provider_id=p1",
		map[string]string{"username": "user1", "password": "pass1"}))
	w := httptest.NewRecorder()
	h.LoginPublic(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoginHandler_LoginPublic_Success(t *testing.T) {
	svc := &mockLoginService{
		loginPublicFn: func(u, p, c, pr string) (*dto.LoginResponseDto, error) {
			return &dto.LoginResponseDto{AccessToken: "tok"}, nil
		},
	}
	h := NewLoginHandler(svc)
	r := withSecurityCtx(newLoginRequest(t, http.MethodPost, "/public/login?client_id=c1&provider_id=p1",
		map[string]string{"username": "user1", "password": "pass1"}))
	w := httptest.NewRecorder()
	h.LoginPublic(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Logout
// ---------------------------------------------------------------------------

func TestLoginHandler_Logout_Success(t *testing.T) {
	h := NewLoginHandler(&mockLoginService{})
	r := withSecurityCtx(httptest.NewRequest(http.MethodPost, "/logout", nil))
	w := httptest.NewRecorder()
	h.Logout(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Login (internal)
// ---------------------------------------------------------------------------

func TestLoginHandler_Login_WithOptionalParams(t *testing.T) {
	// Passes client_id and provider_id query params → covers the two optional
	// pointer branches (lines 203-205 and 206-208).
	svc := &mockLoginService{
		loginFn: func(u, p string, c, pr *string) (*dto.LoginResponseDto, error) {
			return &dto.LoginResponseDto{AccessToken: "tok"}, nil
		},
	}
	h := NewLoginHandler(svc)
	r := withSecurityCtx(newLoginRequest(t, http.MethodPost, "/login?client_id=c1&provider_id=p1",
		map[string]string{"username": "user1", "password": "pass1"}))
	w := httptest.NewRecorder()
	h.Login(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoginHandler_Login_BodyValidationError(t *testing.T) {
	// Valid JSON that fails LoginRequestDto.Validate() → covers lines 218-233.
	h := NewLoginHandler(&mockLoginService{})
	r := withSecurityCtx(newLoginRequest(t, http.MethodPost, "/login",
		map[string]string{"username": "", "password": "pass1"}))
	w := httptest.NewRecorder()
	h.Login(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginHandler_Login_InvalidBody(t *testing.T) {
	h := NewLoginHandler(&mockLoginService{})
	r := withSecurityCtx(httptest.NewRequest(http.MethodPost, "/login",
		bytes.NewBufferString(`{bad json}`)))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginHandler_Login_ServiceError(t *testing.T) {
	svc := &mockLoginService{
		loginFn: func(u, p string, c, pr *string) (*dto.LoginResponseDto, error) {
			return nil, assert.AnError
		},
	}
	h := NewLoginHandler(svc)
	r := withSecurityCtx(newLoginRequest(t, http.MethodPost, "/login",
		map[string]string{"username": "user1", "password": "pass1"}))
	w := httptest.NewRecorder()
	h.Login(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoginHandler_Login_Success(t *testing.T) {
	svc := &mockLoginService{
		loginFn: func(u, p string, c, pr *string) (*dto.LoginResponseDto, error) {
			return &dto.LoginResponseDto{AccessToken: "tok"}, nil
		},
	}
	h := NewLoginHandler(svc)
	r := withSecurityCtx(newLoginRequest(t, http.MethodPost, "/login",
		map[string]string{"username": "user1", "password": "pass1"}))
	w := httptest.NewRecorder()
	h.Login(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/security"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fpRequest(t *testing.T, url string, body any) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	r := httptest.NewRequest(http.MethodPost, url, &buf)
	r.Header.Set("Content-Type", "application/json")
	return withSecurityCtx(r)
}

// ---------------------------------------------------------------------------
// ForgotPasswordPublic
// ---------------------------------------------------------------------------

func TestForgotPasswordHandler_ForgotPasswordPublic_MissingParams(t *testing.T) {
	h := NewForgotPasswordHandler(&mockForgotPasswordService{})
	r := fpRequest(t, "/public/forgot-password", map[string]string{"email": "a@b.com"})
	w := httptest.NewRecorder()
	h.ForgotPasswordPublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestForgotPasswordHandler_ForgotPasswordPublic_InvalidBody(t *testing.T) {
	h := NewForgotPasswordHandler(&mockForgotPasswordService{})
	r := httptest.NewRequest(http.MethodPost, "/public/forgot-password?client_id=c1&provider_id=p1",
		bytes.NewBufferString(`bad`))
	r.Header.Set("Content-Type", "application/json")
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.ForgotPasswordPublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestForgotPasswordHandler_ForgotPasswordPublic_ValidationError(t *testing.T) {
	h := NewForgotPasswordHandler(&mockForgotPasswordService{})
	r := fpRequest(t, "/public/forgot-password?client_id=c1&provider_id=p1",
		map[string]string{"email": "not-an-email"})
	w := httptest.NewRecorder()
	h.ForgotPasswordPublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestForgotPasswordHandler_ForgotPasswordPublic_ServiceError(t *testing.T) {
	svc := &mockForgotPasswordService{
		sendPasswordResetEmailFn: func(email string, c, p *string, internal bool) (*dto.ForgotPasswordResponseDto, error) {
			return nil, assert.AnError
		},
	}
	h := NewForgotPasswordHandler(svc)
	r := fpRequest(t, "/public/forgot-password?client_id=c1&provider_id=p1",
		map[string]string{"email": "user@example.com"})
	w := httptest.NewRecorder()
	h.ForgotPasswordPublic(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestForgotPasswordHandler_ForgotPasswordPublic_Success(t *testing.T) {
	svc := &mockForgotPasswordService{
		sendPasswordResetEmailFn: func(email string, c, p *string, internal bool) (*dto.ForgotPasswordResponseDto, error) {
			return &dto.ForgotPasswordResponseDto{}, nil
		},
	}
	h := NewForgotPasswordHandler(svc)
	r := fpRequest(t, "/public/forgot-password?client_id=c1&provider_id=p1",
		map[string]string{"email": "user@example.com"})
	w := httptest.NewRecorder()
	h.ForgotPasswordPublic(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// ForgotPassword (internal)
// ---------------------------------------------------------------------------

func TestForgotPasswordHandler_ForgotPassword_InvalidBody(t *testing.T) {
	h := NewForgotPasswordHandler(&mockForgotPasswordService{})
	r := httptest.NewRequest(http.MethodPost, "/forgot-password",
		bytes.NewBufferString(`bad`))
	r.Header.Set("Content-Type", "application/json")
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.ForgotPassword(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestForgotPasswordHandler_ForgotPassword_ValidationError(t *testing.T) {
	h := NewForgotPasswordHandler(&mockForgotPasswordService{})
	r := fpRequest(t, "/forgot-password", map[string]string{"email": "bad"})
	w := httptest.NewRecorder()
	h.ForgotPassword(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestForgotPasswordHandler_ForgotPassword_ServiceError(t *testing.T) {
	svc := &mockForgotPasswordService{
		sendPasswordResetEmailFn: func(email string, c, p *string, internal bool) (*dto.ForgotPasswordResponseDto, error) {
			return nil, assert.AnError
		},
	}
	h := NewForgotPasswordHandler(svc)
	r := fpRequest(t, "/forgot-password", map[string]string{"email": "user@example.com"})
	w := httptest.NewRecorder()
	h.ForgotPassword(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestForgotPasswordHandler_ForgotPassword_Success(t *testing.T) {
	svc := &mockForgotPasswordService{
		sendPasswordResetEmailFn: func(email string, c, p *string, internal bool) (*dto.ForgotPasswordResponseDto, error) {
			return &dto.ForgotPasswordResponseDto{}, nil
		},
	}
	h := NewForgotPasswordHandler(svc)
	r := fpRequest(t, "/forgot-password", map[string]string{"email": "user@example.com"})
	w := httptest.NewRecorder()
	h.ForgotPassword(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// lockedRateLimiter starts a miniredis instance, pre-sets the lock key for the
// given identifier, wires it into util.CheckRateLimit, and returns a cleanup
// function that resets the rate limiter to nil after the test.
func lockedRateLimiter(t *testing.T, identifier string) func() {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	security.InitRateLimiter(rdb)

	// Pre-set the lock key so CheckRateLimit returns an error immediately.
	require.NoError(t, mr.Set("rl:lock:"+identifier, "1"))

	return func() {
		security.InitRateLimiter(nil)
		rdb.Close()
		mr.Close()
	}
}

func TestForgotPasswordHandler_ForgotPasswordPublic_RateLimited(t *testing.T) {
	email := "ratelimited-public@example.com"
	cleanup := lockedRateLimiter(t, email)
	defer cleanup()

	h := NewForgotPasswordHandler(&mockForgotPasswordService{})
	r := fpRequest(t, "/public/forgot-password?client_id=c1&provider_id=p1",
		map[string]string{"email": email})
	w := httptest.NewRecorder()
	h.ForgotPasswordPublic(w, r)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestForgotPasswordHandler_ForgotPassword_RateLimited(t *testing.T) {
	email := "ratelimited-internal@example.com"
	cleanup := lockedRateLimiter(t, email)
	defer cleanup()

	h := NewForgotPasswordHandler(&mockForgotPasswordService{})
	r := fpRequest(t, "/forgot-password", map[string]string{"email": email})
	w := httptest.NewRecorder()
	h.ForgotPassword(w, r)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestForgotPasswordHandler_ForgotPassword_WithOptionalParams(t *testing.T) {
	// Covers the client_id != "" and provider_id != "" optional query-param branches (lines 145-150).
	svc := &mockForgotPasswordService{
		sendPasswordResetEmailFn: func(email string, c, p *string, internal bool) (*dto.ForgotPasswordResponseDto, error) {
			return &dto.ForgotPasswordResponseDto{}, nil
		},
	}
	h := NewForgotPasswordHandler(svc)
	r := fpRequest(t, "/forgot-password?client_id=c1&provider_id=p1", map[string]string{"email": "user@example.com"})
	w := httptest.NewRecorder()
	h.ForgotPassword(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

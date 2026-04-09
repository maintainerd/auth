package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/signedurl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	if os.Getenv("HMAC_SECRET_KEY") == "" {
		os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-unit-tests")
	}
}

func validSignedQuery(t *testing.T, params map[string]string) string {
	t.Helper()
	signed, err := signedurl.GenerateSignedURL("http://x", params, 10*time.Minute)
	require.NoError(t, err)
	parsed, err := url.Parse(signed)
	require.NoError(t, err)
	return parsed.RawQuery
}

// ---------------------------------------------------------------------------
// ResetPasswordPublic
// ---------------------------------------------------------------------------

func TestResetPasswordHandler_ResetPasswordPublic_MissingSignature(t *testing.T) {
	h := NewResetPasswordHandler(&mockResetPasswordService{})
	r := httptest.NewRequest(http.MethodPost, "/public/reset-password", nil)
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.ResetPasswordPublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResetPasswordHandler_ResetPasswordPublic_MissingRequiredSignedParams(t *testing.T) {
	// sig+expires valid but client_id/provider_id/token missing
	q := validSignedQuery(t, map[string]string{})
	h := NewResetPasswordHandler(&mockResetPasswordService{})
	r := httptest.NewRequest(http.MethodPost, "/public/reset-password?"+q, nil)
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.ResetPasswordPublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResetPasswordHandler_ResetPasswordPublic_InvalidBody(t *testing.T) {
	q := validSignedQuery(t, map[string]string{
		"client_id": "c1", "provider_id": "p1", "token": "tok123",
	})
	h := NewResetPasswordHandler(&mockResetPasswordService{})
	r := httptest.NewRequest(http.MethodPost, "/public/reset-password?"+q,
		bytes.NewBufferString(`bad`))
	r.Header.Set("Content-Type", "application/json")
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.ResetPasswordPublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResetPasswordHandler_ResetPasswordPublic_ServiceError(t *testing.T) {
	q := validSignedQuery(t, map[string]string{
		"client_id": "c1", "provider_id": "p1", "token": "tok123",
	})
	body, _ := json.Marshal(map[string]string{
		"new_password": "NewPass@1234", "confirm_password": "NewPass@1234",
	})
	svc := &mockResetPasswordService{
		resetPasswordFn: func(token, pw string, c, p *string) (*dto.ResetPasswordResponseDTO, error) {
			return nil, errValidation
		},
	}
	h := NewResetPasswordHandler(svc)
	r := httptest.NewRequest(http.MethodPost, "/public/reset-password?"+q, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.ResetPasswordPublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResetPasswordHandler_ResetPasswordPublic_Success(t *testing.T) {
	q := validSignedQuery(t, map[string]string{
		"client_id": "c1", "provider_id": "p1", "token": "tok123",
	})
	body, _ := json.Marshal(map[string]string{
		"new_password": "NewPass@1234", "confirm_password": "NewPass@1234",
	})
	svc := &mockResetPasswordService{
		resetPasswordFn: func(token, pw string, c, p *string) (*dto.ResetPasswordResponseDTO, error) {
			return &dto.ResetPasswordResponseDTO{}, nil
		},
	}
	h := NewResetPasswordHandler(svc)
	r := httptest.NewRequest(http.MethodPost, "/public/reset-password?"+q, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.ResetPasswordPublic(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// ResetPassword (internal) — also requires a signed URL
// ---------------------------------------------------------------------------

func TestResetPasswordHandler_ResetPassword_MissingSignature(t *testing.T) {
	h := NewResetPasswordHandler(&mockResetPasswordService{})
	r := httptest.NewRequest(http.MethodPost, "/reset-password", nil)
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.ResetPassword(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResetPasswordHandler_ResetPassword_InvalidBody(t *testing.T) {
	q := validSignedQuery(t, map[string]string{"token": "tok123"})
	h := NewResetPasswordHandler(&mockResetPasswordService{})
	r := httptest.NewRequest(http.MethodPost, "/reset-password?"+q,
		bytes.NewBufferString(`bad`))
	r.Header.Set("Content-Type", "application/json")
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.ResetPassword(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResetPasswordHandler_ResetPassword_ServiceError(t *testing.T) {
	q := validSignedQuery(t, map[string]string{"token": "tok123"})
	svc := &mockResetPasswordService{
		resetPasswordFn: func(token, pw string, c, p *string) (*dto.ResetPasswordResponseDTO, error) {
			return nil, errValidation
		},
	}
	h := NewResetPasswordHandler(svc)
	body, _ := json.Marshal(map[string]string{
		"new_password": "NewPass@1234", "confirm_password": "NewPass@1234",
	})
	r := httptest.NewRequest(http.MethodPost, "/reset-password?"+q, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.ResetPassword(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResetPasswordHandler_ResetPassword_Success(t *testing.T) {
	q := validSignedQuery(t, map[string]string{"token": "tok123"})
	svc := &mockResetPasswordService{
		resetPasswordFn: func(token, pw string, c, p *string) (*dto.ResetPasswordResponseDTO, error) {
			return &dto.ResetPasswordResponseDTO{}, nil
		},
	}
	h := NewResetPasswordHandler(svc)
	body, _ := json.Marshal(map[string]string{
		"new_password": "NewPass@1234", "confirm_password": "NewPass@1234",
	})
	r := httptest.NewRequest(http.MethodPost, "/reset-password?"+q, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.ResetPassword(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── ResetPasswordPublic: missing branches ────────────────────────────────────

// ValidationError: valid signed URL + body that passes decode but fails Validate()
// (new_password is required) → covers lines 105-120.
func TestResetPasswordHandler_ResetPasswordPublic_ValidationError(t *testing.T) {
	q := validSignedQuery(t, map[string]string{
		"client_id": "c1", "provider_id": "p1", "token": "tok123",
	})
	body, _ := json.Marshal(map[string]string{}) // missing new_password
	h := NewResetPasswordHandler(&mockResetPasswordService{})
	r := httptest.NewRequest(http.MethodPost, "/public/reset-password?"+q, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.ResetPasswordPublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// RateLimit: pre-locks the token key in miniredis → covers lines 126-141 → 429.
func TestResetPasswordHandler_ResetPasswordPublic_RateLimit(t *testing.T) {
	token := "tok-rate-pub"
	cleanup := lockedRateLimiter(t, token)
	defer cleanup()

	q := validSignedQuery(t, map[string]string{
		"client_id": "c1", "provider_id": "p1", "token": token,
	})
	body, _ := json.Marshal(map[string]string{"new_password": "NewPass@1234"})
	h := NewResetPasswordHandler(&mockResetPasswordService{})
	r := httptest.NewRequest(http.MethodPost, "/public/reset-password?"+q, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.ResetPasswordPublic(w, r)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

// ── ResetPassword (internal): missing branches ───────────────────────────────

// MissingToken: signed URL valid but no token param → covers lines 230-244 → 400.
func TestResetPasswordHandler_ResetPassword_MissingToken(t *testing.T) {
	q := validSignedQuery(t, map[string]string{}) // no token
	h := NewResetPasswordHandler(&mockResetPasswordService{})
	r := httptest.NewRequest(http.MethodPost, "/reset-password?"+q, nil)
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.ResetPassword(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// WithClientAndProvider: signed URL includes client_id + provider_id → covers
// lines 222-224 and 225-227 (pointer-assign branches) → 200.
func TestResetPasswordHandler_ResetPassword_WithClientAndProvider(t *testing.T) {
	q := validSignedQuery(t, map[string]string{
		"token": "tok-with-cp", "client_id": "c1", "provider_id": "p1",
	})
	svc := &mockResetPasswordService{
		resetPasswordFn: func(token, pw string, c, p *string) (*dto.ResetPasswordResponseDTO, error) {
			return &dto.ResetPasswordResponseDTO{}, nil
		},
	}
	body, _ := json.Marshal(map[string]string{"new_password": "NewPass@1234"})
	h := NewResetPasswordHandler(svc)
	r := httptest.NewRequest(http.MethodPost, "/reset-password?"+q, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.ResetPassword(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ValidationError: valid signed URL + empty body → covers lines 265-280 → 400.
func TestResetPasswordHandler_ResetPassword_ValidationError(t *testing.T) {
	q := validSignedQuery(t, map[string]string{"token": "tok123"})
	body, _ := json.Marshal(map[string]string{}) // missing new_password
	h := NewResetPasswordHandler(&mockResetPasswordService{})
	r := httptest.NewRequest(http.MethodPost, "/reset-password?"+q, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.ResetPassword(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// RateLimit: pre-locks the token key in miniredis → covers lines 285-300 → 429.
func TestResetPasswordHandler_ResetPassword_RateLimit(t *testing.T) {
	token := "tok-rate-internal"
	cleanup := lockedRateLimiter(t, token)
	defer cleanup()

	q := validSignedQuery(t, map[string]string{"token": token})
	body, _ := json.Marshal(map[string]string{"new_password": "NewPass@1234"})
	h := NewResetPasswordHandler(&mockResetPasswordService{})
	r := httptest.NewRequest(http.MethodPost, "/reset-password?"+q, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.ResetPassword(w, r)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

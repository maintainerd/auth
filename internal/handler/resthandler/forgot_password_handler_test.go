package resthandler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maintainerd/auth/internal/dto"
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

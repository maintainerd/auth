package resthandler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/util"
	"github.com/stretchr/testify/assert"
)

func init() {
	if os.Getenv("HMAC_SECRET_KEY") == "" {
		os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-unit-tests")
	}
}

func validSignedQuery(t *testing.T, params map[string]string) string {
	t.Helper()
	v := url.Values{}
	for k, val := range params {
		v.Set(k, val)
	}
	expires := time.Now().Add(10 * time.Minute).Unix()
	v.Set("expires", fmt.Sprintf("%d", expires))
	sig := util.ComputeSignature(v)
	v.Set("sig", sig)
	return v.Encode()
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
		resetPasswordFn: func(token, pw string, c, p *string) (*dto.ResetPasswordResponseDto, error) {
			return nil, assert.AnError
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
		resetPasswordFn: func(token, pw string, c, p *string) (*dto.ResetPasswordResponseDto, error) {
			return &dto.ResetPasswordResponseDto{}, nil
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
		resetPasswordFn: func(token, pw string, c, p *string) (*dto.ResetPasswordResponseDto, error) {
			return nil, assert.AnError
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
		resetPasswordFn: func(token, pw string, c, p *string) (*dto.ResetPasswordResponseDto, error) {
			return &dto.ResetPasswordResponseDto{}, nil
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

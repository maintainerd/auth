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

func regRequest(t *testing.T, url string, body any) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	r := httptest.NewRequest(http.MethodPost, url, &buf)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("User-Agent", "Mozilla/5.0 (test)")
	return withSecurityCtx(r)
}

func withEmptyUACtx(r *http.Request) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.ClientIPKey, "127.0.0.1")
	ctx = context.WithValue(ctx, middleware.UserAgentKey, "")
	ctx = context.WithValue(ctx, middleware.RequestIDKey, "req-r")
	return r.WithContext(ctx)
}

// ---------------------------------------------------------------------------
// RegisterPublic
// ---------------------------------------------------------------------------

func TestRegisterHandler_RegisterPublic_MissingClientID(t *testing.T) {
	h := NewRegisterHandler(&mockRegisterService{})
	r := regRequest(t, "/public/register", map[string]string{
		"username": "user1", "password": "Pass@1234", "fullname": "User One",
	})
	w := httptest.NewRecorder()
	h.RegisterPublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterHandler_RegisterPublic_EmptyUserAgent(t *testing.T) {
	h := NewRegisterHandler(&mockRegisterService{})
	body, _ := json.Marshal(map[string]string{
		"username": "user1", "password": "Pass@1234", "fullname": "User One",
	})
	r := httptest.NewRequest(http.MethodPost, "/public/register?client_id=c1&provider_id=p1",
		bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withEmptyUACtx(r)
	w := httptest.NewRecorder()
	h.RegisterPublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterHandler_RegisterPublic_InvalidBody(t *testing.T) {
	h := NewRegisterHandler(&mockRegisterService{})
	r := httptest.NewRequest(http.MethodPost, "/public/register?client_id=c1&provider_id=p1",
		bytes.NewBufferString(`bad json`))
	r.Header.Set("Content-Type", "application/json")
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.RegisterPublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterHandler_RegisterPublic_ServiceError(t *testing.T) {
	svc := &mockRegisterService{
		registerPublicFn: func(u, f, p string, e, ph *string, c, pr string) (*dto.RegisterResponseDto, error) {
			return nil, assert.AnError
		},
	}
	h := NewRegisterHandler(svc)
	r := regRequest(t, "/public/register?client_id=c1&provider_id=p1", map[string]string{
		"username": "user1", "password": "Pass@1234!", "fullname": "User One",
	})
	w := httptest.NewRecorder()
	h.RegisterPublic(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRegisterHandler_RegisterPublic_Success(t *testing.T) {
	svc := &mockRegisterService{
		registerPublicFn: func(u, f, p string, e, ph *string, c, pr string) (*dto.RegisterResponseDto, error) {
			return &dto.RegisterResponseDto{}, nil
		},
	}
	h := NewRegisterHandler(svc)
	r := regRequest(t, "/public/register?client_id=c1&provider_id=p1", map[string]string{
		"username": "user1", "password": "Pass@1234!", "fullname": "User One",
	})
	w := httptest.NewRecorder()
	h.RegisterPublic(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ---------------------------------------------------------------------------
// Register (internal)
// ---------------------------------------------------------------------------

func TestRegisterHandler_Register_InvalidBody(t *testing.T) {
	h := NewRegisterHandler(&mockRegisterService{})
	r := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{bad}`))
	r.Header.Set("Content-Type", "application/json")
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.Register(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterHandler_Register_ServiceError(t *testing.T) {
	svc := &mockRegisterService{
		registerFn: func(u, f, p string, e, ph, c, pr *string) (*dto.RegisterResponseDto, error) {
			return nil, assert.AnError
		},
	}
	h := NewRegisterHandler(svc)
	r := regRequest(t, "/register", map[string]string{
		"username": "user1", "password": "Pass@1234!", "fullname": "User One",
	})
	w := httptest.NewRecorder()
	h.Register(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRegisterHandler_Register_Success(t *testing.T) {
	svc := &mockRegisterService{
		registerFn: func(u, f, p string, e, ph, c, pr *string) (*dto.RegisterResponseDto, error) {
			return &dto.RegisterResponseDto{}, nil
		},
	}
	h := NewRegisterHandler(svc)
	r := regRequest(t, "/register", map[string]string{
		"username": "user1", "password": "Pass@1234!", "fullname": "User One",
	})
	w := httptest.NewRecorder()
	h.Register(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ---------------------------------------------------------------------------
// RegisterInvite (internal)
// ---------------------------------------------------------------------------

func TestRegisterHandler_RegisterInvite_MissingToken(t *testing.T) {
	h := NewRegisterHandler(&mockRegisterService{})
	r := regRequest(t, "/register/invite", map[string]string{"username": "u", "password": "p"})
	w := httptest.NewRecorder()
	h.RegisterInvite(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterHandler_RegisterInvite_ServiceError(t *testing.T) {
	svc := &mockRegisterService{
		registerInviteFn: func(u, p, t string, c, pr *string) (*dto.RegisterResponseDto, error) {
			return nil, assert.AnError
		},
	}
	h := NewRegisterHandler(svc)
	r := regRequest(t, "/register/invite?invite_token=tok", map[string]string{
		"username": "user1", "password": "Pass@1234",
	})
	w := httptest.NewRecorder()
	h.RegisterInvite(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRegisterHandler_RegisterInvite_Success(t *testing.T) {
	svc := &mockRegisterService{
		registerInviteFn: func(u, p, t string, c, pr *string) (*dto.RegisterResponseDto, error) {
			return &dto.RegisterResponseDto{}, nil
		},
	}
	h := NewRegisterHandler(svc)
	r := regRequest(t, "/register/invite?invite_token=tok", map[string]string{
		"username": "user1", "password": "Pass@1234",
	})
	w := httptest.NewRecorder()
	h.RegisterInvite(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ---------------------------------------------------------------------------
// RegisterInvitePublic
// ---------------------------------------------------------------------------

func TestRegisterHandler_RegisterInvitePublic_MissingParams(t *testing.T) {
	h := NewRegisterHandler(&mockRegisterService{})
	// Missing required client_id/provider_id/invite_token
	r := regRequest(t, "/public/register/invite", map[string]string{"username": "u", "password": "p"})
	w := httptest.NewRecorder()
	h.RegisterInvitePublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterHandler_RegisterInvitePublic_ServiceError(t *testing.T) {
	svc := &mockRegisterService{
		registerInvitePublicFn: func(u, p, c, pr, t string) (*dto.RegisterResponseDto, error) {
			return nil, assert.AnError
		},
	}
	h := NewRegisterHandler(svc)
	r := regRequest(t, "/public/register/invite?client_id=c1&provider_id=p1&invite_token=tok&expires=9999999999&sig=fake",
		map[string]string{"username": "user1", "password": "pass1"})
	w := httptest.NewRecorder()
	h.RegisterInvitePublic(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

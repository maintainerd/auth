package handler

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
		registerPublicFn: func(u, f, p string, e, ph *string, c, pr string) (*dto.RegisterResponseDTO, error) {
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
		registerPublicFn: func(u, f, p string, e, ph *string, c, pr string) (*dto.RegisterResponseDTO, error) {
			return &dto.RegisterResponseDTO{}, nil
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
		registerFn: func(u, f, p string, e, ph, c, pr *string) (*dto.RegisterResponseDTO, error) {
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
		registerFn: func(u, f, p string, e, ph, c, pr *string) (*dto.RegisterResponseDTO, error) {
			return &dto.RegisterResponseDTO{}, nil
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
		registerInviteFn: func(u, p, t string, c, pr *string) (*dto.RegisterResponseDTO, error) {
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
		registerInviteFn: func(u, p, t string, c, pr *string) (*dto.RegisterResponseDTO, error) {
			return &dto.RegisterResponseDTO{}, nil
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
		registerInvitePublicFn: func(u, p, c, pr, t string) (*dto.RegisterResponseDTO, error) {
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

// ── RegisterPublic ────────────────────────────────────────────────────────────

// ValidationError via weak password: passes basic Validate() (8+ chars, username,
// fullname present) but fails ValidatePasswordStrength() → covers the
// ValidateForRegistration() error path including "registration_weak_password" branch.
func TestRegisterHandler_RegisterPublic_ValidationError(t *testing.T) {
	h := NewRegisterHandler(&mockRegisterService{})
	r := regRequest(t, "/public/register?client_id=c1&provider_id=p1", map[string]string{
		"username": "user1",
		"fullname": "User One",
		"password": "Password1234", // no special char → weak password
	})
	w := httptest.NewRecorder()
	h.RegisterPublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── Register ──────────────────────────────────────────────────────────────────

// ValidationError: covers the ValidateForRegistration() error path + weak-password branch.
func TestRegisterHandler_Register_ValidationError(t *testing.T) {
	h := NewRegisterHandler(&mockRegisterService{})
	r := regRequest(t, "/register", map[string]string{
		"username": "user1",
		"fullname": "User One",
		"password": "Password1234", // no special char → weak password
	})
	w := httptest.NewRecorder()
	h.Register(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// WithOptionalParams: passes ?client_id and ?provider_id → covers the two
// pointer-assign branches (lines 163-165, 166-168).
func TestRegisterHandler_Register_WithOptionalParams(t *testing.T) {
	svc := &mockRegisterService{
		registerFn: func(u, f, p string, e, ph, c, pr *string) (*dto.RegisterResponseDTO, error) {
			return &dto.RegisterResponseDTO{}, nil
		},
	}
	h := NewRegisterHandler(svc)
	r := regRequest(t, "/register?client_id=c1&provider_id=p1", map[string]string{
		"username": "user1", "password": "Pass@1234!", "fullname": "User One",
	})
	w := httptest.NewRecorder()
	h.Register(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ── RegisterInvite ────────────────────────────────────────────────────────────

// BadJSON: invite_token present, body is malformed → covers decode error path.
func TestRegisterHandler_RegisterInvite_BadJSON(t *testing.T) {
	h := NewRegisterHandler(&mockRegisterService{})
	r := httptest.NewRequest(http.MethodPost, "/register/invite?invite_token=tok",
		bytes.NewBufferString("{bad}"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.RegisterInvite(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ValidationError: invite_token present, body decodes but fails LoginRequestDTO.Validate().
func TestRegisterHandler_RegisterInvite_ValidationError(t *testing.T) {
	h := NewRegisterHandler(&mockRegisterService{})
	r := regRequest(t, "/register/invite?invite_token=tok", map[string]string{})
	w := httptest.NewRecorder()
	h.RegisterInvite(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// WithOptionalParams: ?client_id and ?provider_id present → covers pointer-assign branches.
func TestRegisterHandler_RegisterInvite_WithOptionalParams(t *testing.T) {
	svc := &mockRegisterService{
		registerInviteFn: func(u, p, tok string, c, pr *string) (*dto.RegisterResponseDTO, error) {
			return &dto.RegisterResponseDTO{}, nil
		},
	}
	h := NewRegisterHandler(svc)
	r := regRequest(t, "/register/invite?invite_token=tok&client_id=c1&provider_id=p1",
		map[string]string{"username": "user1", "password": "Pass@1234"})
	w := httptest.NewRecorder()
	h.RegisterInvite(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ── RegisterInvitePublic ──────────────────────────────────────────────────────

const invitePublicURL = "/public/register/invite?client_id=c1&provider_id=p1&invite_token=tok&expires=9999999999&sig=fake"

// BadJSON: query params valid, body malformed → covers decode error path.
func TestRegisterHandler_RegisterInvitePublic_BadJSON(t *testing.T) {
	h := NewRegisterHandler(&mockRegisterService{})
	r := httptest.NewRequest(http.MethodPost, invitePublicURL, bytes.NewBufferString("{bad}"))
	r.Header.Set("Content-Type", "application/json")
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.RegisterInvitePublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ValidationError: query params valid, body decodes but fails LoginRequestDTO.Validate().
func TestRegisterHandler_RegisterInvitePublic_ValidationError(t *testing.T) {
	h := NewRegisterHandler(&mockRegisterService{})
	r := regRequest(t, invitePublicURL, map[string]string{})
	w := httptest.NewRecorder()
	h.RegisterInvitePublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Success: covers util.CreatedWithCookies response path (line 332).
func TestRegisterHandler_RegisterInvitePublic_Success(t *testing.T) {
	svc := &mockRegisterService{
		registerInvitePublicFn: func(u, p, c, pr, tok string) (*dto.RegisterResponseDTO, error) {
			return &dto.RegisterResponseDTO{}, nil
		},
	}
	h := NewRegisterHandler(svc)
	r := regRequest(t, invitePublicURL, map[string]string{"username": "user1", "password": "pass1"})
	w := httptest.NewRecorder()
	h.RegisterInvitePublic(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

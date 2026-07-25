package authn

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/signedurl"
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

func TestRegisterHandler_RegisterPublic_RequiresClientID(t *testing.T) {
	svc := &mockRegisterService{
		registerPublicFn: func(u, f, p string, e, ph, c, pr *string, _ string) (*RegisterResponseDTO, error) {
			return &RegisterResponseDTO{}, nil
		},
	}
	h := NewRegisterHandler(svc)
	r := regRequest(t, "/public/register", map[string]string{
		"username": "user1", "password": "Pass@1234!", "fullname": "User One",
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
	r := httptest.NewRequest(http.MethodPost, "/public/register?client_id=c1",
		bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withEmptyUACtx(r)
	w := httptest.NewRecorder()
	h.RegisterPublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterHandler_RegisterPublic_InvalidBody(t *testing.T) {
	h := NewRegisterHandler(&mockRegisterService{})
	r := httptest.NewRequest(http.MethodPost, "/public/register?client_id=c1",
		bytes.NewBufferString(`bad json`))
	r.Header.Set("Content-Type", "application/json")
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.RegisterPublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterHandler_RegisterPublic_ServiceError(t *testing.T) {
	svc := &mockRegisterService{
		registerPublicFn: func(u, f, p string, e, ph *string, c, pr *string, _ string) (*RegisterResponseDTO, error) {
			return nil, assert.AnError
		},
	}
	h := NewRegisterHandler(svc)
	r := regRequest(t, "/public/register?client_id=c1", map[string]string{
		"username": "user1", "password": "Pass@1234!", "fullname": "User One",
	})
	w := httptest.NewRecorder()
	h.RegisterPublic(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRegisterHandler_RegisterPublic_Success(t *testing.T) {
	svc := &mockRegisterService{
		registerPublicFn: func(u, f, p string, e, ph *string, c, pr *string, _ string) (*RegisterResponseDTO, error) {
			return &RegisterResponseDTO{}, nil
		},
	}
	h := NewRegisterHandler(svc)
	r := regRequest(t, "/public/register?client_id=c1", map[string]string{
		"username": "user1", "password": "Pass@1234!", "fullname": "User One",
	})
	w := httptest.NewRecorder()
	h.RegisterPublic(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// Note: the verification email is sent by RegisterService.RegisterPublic (policy-gated,
// inside its transaction), not by the handler — so its behavior is covered by the
// registration service tests, not here.

// ---------------------------------------------------------------------------
// Register (internal)
// ---------------------------------------------------------------------------

func TestRegisterHandler_Register_InvalidBody(t *testing.T) {
	h := NewRegisterHandler(&mockRegisterService{})
	r := httptest.NewRequest(http.MethodPost, "/register?tenant_id=system", bytes.NewBufferString(`{bad}`))
	r.Header.Set("Content-Type", "application/json")
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.Register(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterHandler_Register_ServiceError(t *testing.T) {
	svc := &mockRegisterService{
		registerFn: func(u, f, p string, e, ph, c, pr *string, _ string) (*RegisterResponseDTO, error) {
			return nil, assert.AnError
		},
	}
	h := NewRegisterHandler(svc)
	r := regRequest(t, "/register?tenant_id=system", map[string]string{
		"username": "user1", "password": "Pass@1234!", "fullname": "User One",
	})
	w := httptest.NewRecorder()
	h.Register(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRegisterHandler_Register_Success(t *testing.T) {
	svc := &mockRegisterService{
		registerFn: func(u, f, p string, e, ph, c, pr *string, _ string) (*RegisterResponseDTO, error) {
			return &RegisterResponseDTO{}, nil
		},
	}
	h := NewRegisterHandler(svc)
	r := regRequest(t, "/register?tenant_id=system", map[string]string{
		"username": "user1", "password": "Pass@1234!", "fullname": "User One",
	})
	w := httptest.NewRecorder()
	h.Register(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func testInvitePublicURL(clientID, inviteToken string) string {
	params := map[string]string{"client_id": clientID, "invite_token": inviteToken}
	u, _ := signedurl.GenerateSignedURL("/public/register/invite", params, time.Hour)
	parsed, _ := url.Parse(u)
	return "/public/register/invite?" + parsed.RawQuery
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
		registerInvitePublicFn: func(u, p, c, pr, t string) (*RegisterResponseDTO, error) {
			return nil, assert.AnError
		},
	}
	h := NewRegisterHandler(svc)
	u := testInvitePublicURL("c1", "tok")
	r := regRequest(t, u,
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
	r := regRequest(t, "/public/register?client_id=c1", map[string]string{
		"username": "user1",
		"fullname": "User One",
		// Too short for the DTO's absolute bound. Password STRENGTH is the tenant's
		// policy and is applied in the service layer, so the handler can only reject
		// a malformed request — it must not re-impose a hardcoded composition rule.
		"password": "short",
	})
	w := httptest.NewRecorder()
	h.RegisterPublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── Register ──────────────────────────────────────────────────────────────────

// ValidationError: covers the ValidateForRegistration() error path + weak-password branch.
func TestRegisterHandler_Register_ValidationError(t *testing.T) {
	h := NewRegisterHandler(&mockRegisterService{})
	r := regRequest(t, "/register?tenant_id=system", map[string]string{
		"username": "user1",
		"fullname": "User One",
		// Too short for the DTO's absolute bound. Password STRENGTH is the tenant's
		// policy and is applied in the service layer, so the handler can only reject
		// a malformed request — it must not re-impose a hardcoded composition rule.
		"password": "short",
	})
	w := httptest.NewRecorder()
	h.Register(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// WithOptionalParams: passes ?client_id and ?provider_id → covers the two
// pointer-assign branches (lines 163-165, 166-168).
func TestRegisterHandler_Register_RejectsClientContext(t *testing.T) {
	svc := &mockRegisterService{
		registerFn: func(u, f, p string, e, ph, c, pr *string, _ string) (*RegisterResponseDTO, error) {
			return &RegisterResponseDTO{}, nil
		},
	}
	h := NewRegisterHandler(svc)
	r := regRequest(t, "/register?client_id=c1", map[string]string{
		"username": "user1", "password": "Pass@1234!", "fullname": "User One",
	})
	w := httptest.NewRecorder()
	h.Register(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── RegisterInvitePublic ──────────────────────────────────────────────────────

// BadJSON: query params valid, body malformed -> covers decode error path.
func TestRegisterHandler_RegisterInvitePublic_BadJSON(t *testing.T) {
	h := NewRegisterHandler(&mockRegisterService{})
	u := testInvitePublicURL("c1", "tok")
	r := httptest.NewRequest(http.MethodPost, u, bytes.NewBufferString("{bad}"))
	r.Header.Set("Content-Type", "application/json")
	r = withSecurityCtx(r)
	w := httptest.NewRecorder()
	h.RegisterInvitePublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ValidationError: query params valid, body decodes but fails LoginRequestDTO.Validate().
func TestRegisterHandler_RegisterInvitePublic_ValidationError(t *testing.T) {
	h := NewRegisterHandler(&mockRegisterService{})
	u := testInvitePublicURL("c1", "tok")
	r := regRequest(t, u, map[string]string{})
	w := httptest.NewRecorder()
	h.RegisterInvitePublic(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Success: covers util.CreatedWithCookies response path (line 332).
func TestRegisterHandler_RegisterInvitePublic_Success(t *testing.T) {
	svc := &mockRegisterService{
		registerInvitePublicFn: func(u, p, c, pr, tok string) (*RegisterResponseDTO, error) {
			return &RegisterResponseDTO{}, nil
		},
	}
	h := NewRegisterHandler(svc)
	u := testInvitePublicURL("c1", "tok")
	r := regRequest(t, u, map[string]string{"username": "user1", "password": "pass1"})
	w := httptest.NewRecorder()
	h.RegisterInvitePublic(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ---------------------------------------------------------------------------
// registration_flow query parameter
//
// RegisterPublic used to drop ?registration_flow entirely, which made the whole
// feature inert: the flow's status, required_fields and role grants were all
// silently skipped on the self-service path. These tests pin the wiring.
// ---------------------------------------------------------------------------

func TestRegisterHandler_RegisterPublic_ForwardsRegistrationFlow(t *testing.T) {
	var got string
	called := false
	svc := &mockRegisterService{
		registerPublicFn: func(u, f, p string, e, ph, c, pr *string, registrationFlow string) (*RegisterResponseDTO, error) {
			called = true
			got = registrationFlow
			return &RegisterResponseDTO{}, nil
		},
	}
	h := NewRegisterHandler(svc)
	r := regRequest(t, "/public/register?client_id=c1&registration_flow=partner-signup-abcd1234", map[string]string{
		"username": "user1", "password": "Pass@1234!", "fullname": "User One",
	})
	w := httptest.NewRecorder()
	h.RegisterPublic(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
	require.True(t, called)
	assert.Equal(t, "partner-signup-abcd1234", got)
}

func TestRegisterHandler_RegisterPublic_OmittedRegistrationFlowIsEmpty(t *testing.T) {
	got := "sentinel"
	svc := &mockRegisterService{
		registerPublicFn: func(u, f, p string, e, ph, c, pr *string, registrationFlow string) (*RegisterResponseDTO, error) {
			got = registrationFlow
			return &RegisterResponseDTO{}, nil
		},
	}
	h := NewRegisterHandler(svc)
	r := regRequest(t, "/public/register?client_id=c1", map[string]string{
		"username": "user1", "password": "Pass@1234!", "fullname": "User One",
	})
	w := httptest.NewRecorder()
	h.RegisterPublic(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Empty(t, got, "an absent selector must reach the service as the empty string")
}

func TestRegisterHandler_Register_ForwardsRegistrationFlow(t *testing.T) {
	var got string
	svc := &mockRegisterService{
		registerFn: func(u, f, p string, e, ph, c, pr *string, registrationFlow string) (*RegisterResponseDTO, error) {
			got = registrationFlow
			return &RegisterResponseDTO{}, nil
		},
	}
	h := NewRegisterHandler(svc)
	r := regRequest(t, "/register?tenant_id=system&registration_flow=internal-flow-abcd1234", map[string]string{
		"username": "user1", "password": "Pass@1234!", "fullname": "User One",
	})
	w := httptest.NewRecorder()
	h.Register(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "internal-flow-abcd1234", got)
}

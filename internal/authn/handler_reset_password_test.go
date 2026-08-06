package authn

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/signedurl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		resetPasswordFn: func(token, pw string, c, p *string) (*ResetPasswordResponseDTO, error) {
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
		resetPasswordFn: func(token, pw string, c, p *string) (*ResetPasswordResponseDTO, error) {
			return &ResetPasswordResponseDTO{}, nil
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

// A reset token is a bearer credential. It was logged verbatim in four places,
// including on failure paths that leave the token STILL VALID — handing anyone with
// log or SIEM access a working account-takeover token for the rest of its TTL.
func TestResetTokenLogRef_NeverExposesTheToken(t *testing.T) {
	const token = "5f4dcc3b5aa765d61d8327deb882cf99deadbeefcafebabe"

	ref := resetTokenLogRef(token)
	assert.NotContains(t, ref, token, "the raw token must never appear in a log field")
	assert.NotEqual(t, token, ref)
	assert.True(t, strings.HasPrefix(ref, "reset:"), "refs are namespaced so they read as non-credentials")

	// Stable, so the lines of one attempt still correlate.
	assert.Equal(t, ref, resetTokenLogRef(token))
	// Distinct per token, so two attempts do not merge.
	assert.NotEqual(t, ref, resetTokenLogRef(token+"x"))
	// Short enough to be non-reversible by lookup, long enough not to collide.
	assert.Len(t, ref, len("reset:")+8)

	assert.Equal(t, "none", resetTokenLogRef(""))
}

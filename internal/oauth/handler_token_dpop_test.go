package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/dpop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeDPoPNonceStore satisfies dpop.NonceStore for the gate tests.
type fakeDPoPNonceStore struct {
	consumeOK bool
}

func (f *fakeDPoPNonceStore) SaveNonce(_, _ int64, _ string, _ time.Time) error { return nil }
func (f *fakeDPoPNonceStore) ConsumeNonce(_ string) (bool, error)               { return f.consumeOK, nil }

// mockDPoPResolver satisfies oauth.DPoPRequirementResolver.
type mockDPoPResolver struct {
	requirement DPoPRequirement
	ok          bool
}

func (m *mockDPoPResolver) ResolveDPoPRequirement(_ context.Context, _ string) (DPoPRequirement, bool) {
	return m.requirement, m.ok
}

func dpopProofWithNonce(t *testing.T, nonce string) string {
	t.Helper()
	claims := jwtlib.MapClaims{"htm": "POST", "jti": "x"}
	if nonce != "" {
		claims["nonce"] = nonce
	}
	tok := jwtlib.NewWithClaims(jwtlib.SigningMethodNone, claims)
	s, err := tok.SignedString(jwtlib.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)
	return s
}

func newGateHandler(consumeOK bool, resolver *mockDPoPResolver) *OAuthTokenHandler {
	h := NewOAuthTokenHandler(&mockOAuthTokenService{}, nil, nil)
	h.SetDPoPNonceGate(dpop.NewStoreNonceManager(&fakeDPoPNonceStore{consumeOK: consumeOK}), resolver)
	return h
}

func TestEnforceDPoPNonce_GateDisabled(t *testing.T) {
	// No gate configured → never handled.
	h := NewOAuthTokenHandler(&mockOAuthTokenService{}, nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
	assert.False(t, h.enforceDPoPNonce(w, r, OAuthClientCredentials{ClientID: "c"}))
}

func TestEnforceDPoPNonce_ClientNotResolved(t *testing.T) {
	h := newGateHandler(true, &mockDPoPResolver{ok: false})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
	assert.False(t, h.enforceDPoPNonce(w, r, OAuthClientCredentials{ClientID: "c"}))
}

func TestEnforceDPoPNonce_NotRequired(t *testing.T) {
	h := newGateHandler(true, &mockDPoPResolver{ok: true, requirement: DPoPRequirement{Required: false}})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
	assert.False(t, h.enforceDPoPNonce(w, r, OAuthClientCredentials{ClientID: "c"}))
}

func TestEnforceDPoPNonce_RequiredNoNonce_Issues400(t *testing.T) {
	h := newGateHandler(true, &mockDPoPResolver{ok: true, requirement: DPoPRequirement{Required: true, TenantID: 1, InternalClientID: 2}})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/oauth/token", nil) // no DPoP header
	handled := h.enforceDPoPNonce(w, r, OAuthClientCredentials{ClientID: "c"})
	assert.True(t, handled)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.NotEmpty(t, w.Header().Get("DPoP-Nonce"))
	assert.Contains(t, w.Body.String(), "use_dpop_nonce")
}

func TestEnforceDPoPNonce_RequiredInvalidNonce_Issues400(t *testing.T) {
	// Store rejects the nonce (unknown/used/expired) → gate re-issues + 400.
	h := newGateHandler(false, &mockDPoPResolver{ok: true, requirement: DPoPRequirement{Required: true, TenantID: 1, InternalClientID: 2}})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
	r.Header.Set("DPoP", dpopProofWithNonce(t, "stale-nonce"))
	handled := h.enforceDPoPNonce(w, r, OAuthClientCredentials{ClientID: "c"})
	assert.True(t, handled)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.NotEmpty(t, w.Header().Get("DPoP-Nonce"))
}

func TestEnforceDPoPNonce_RequiredValidNonce_Proceeds(t *testing.T) {
	// Store accepts the nonce → gate passes (not handled), request proceeds.
	h := newGateHandler(true, &mockDPoPResolver{ok: true, requirement: DPoPRequirement{Required: true, TenantID: 1, InternalClientID: 2}})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
	r.Header.Set("DPoP", dpopProofWithNonce(t, "good-nonce"))
	handled := h.enforceDPoPNonce(w, r, OAuthClientCredentials{ClientID: "c"})
	assert.False(t, handled)
}

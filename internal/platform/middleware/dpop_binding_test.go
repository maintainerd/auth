package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boundClaims() map[string]any {
	return map[string]any{"sub": "user-1", "cnf": map[string]any{"jkt": "thumb-abc"}}
}

func resetDPoPBinding(t *testing.T) {
	t.Helper()
	origValidator, origURL := dpopBindingValidator, dpopRequestURL
	t.Cleanup(func() { dpopBindingValidator, dpopRequestURL = origValidator, origURL })
	dpopBindingValidator, dpopRequestURL = nil, nil
}

func TestTokenConfirmationThumbprint(t *testing.T) {
	assert.Equal(t, "thumb-abc", tokenConfirmationThumbprint(boundClaims()))
	assert.Empty(t, tokenConfirmationThumbprint(map[string]any{"sub": "user-1"}))
	// A malformed cnf must read as "unbound-looking" only in the sense that there is
	// no thumbprint to compare; the token then follows plain bearer rules.
	assert.Empty(t, tokenConfirmationThumbprint(map[string]any{"cnf": "not-an-object"}))
	assert.Empty(t, tokenConfirmationThumbprint(map[string]any{"cnf": map[string]any{"jkt": "  "}}))
}

// The headline gap: a token bound to a key was accepted as a plain Bearer token, so
// stealing it was enough and the binding protected nothing (RFC 9449 §7.1).
func TestEnforceDPoPBinding_RejectsBoundTokenPresentedAsBearer(t *testing.T) {
	resetDPoPBinding(t)
	called := false
	ConfigureDPoPBinding(func(context.Context, string, string, string, string, string) error {
		called = true
		return nil
	}, nil)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	r.Header.Set("DPoP", "a-proof")

	err := enforceDPoPBinding(r, "bearer", "token", boundClaims())
	require.Error(t, err)
	assert.ErrorIs(t, err, errDPoPRequired)
	assert.False(t, called, "the proof must not even be consulted under the wrong scheme")
}

// A cookie carries no scheme, so a bound token must not be usable from one either.
func TestEnforceDPoPBinding_RejectsBoundTokenFromCookie(t *testing.T) {
	resetDPoPBinding(t)
	ConfigureDPoPBinding(func(context.Context, string, string, string, string, string) error {
		return nil
	}, nil)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	r.Header.Set("DPoP", "a-proof")

	require.ErrorIs(t, enforceDPoPBinding(r, "cookie", "token", boundClaims()), errDPoPRequired)
}

func TestEnforceDPoPBinding_RejectsBoundTokenWithNoProof(t *testing.T) {
	resetDPoPBinding(t)
	ConfigureDPoPBinding(func(context.Context, string, string, string, string, string) error {
		return nil
	}, nil)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	require.ErrorIs(t, enforceDPoPBinding(r, "dpop", "token", boundClaims()), errDPoPRequired)

	r.Header.Set("DPoP", "   ")
	require.ErrorIs(t, enforceDPoPBinding(r, "dpop", "token", boundClaims()), errDPoPRequired)
}

// With no validator wired there is no way to check the binding. Accepting the token
// anyway would silently downgrade it to a bearer token.
func TestEnforceDPoPBinding_FailsClosedWithoutAValidator(t *testing.T) {
	resetDPoPBinding(t)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	r.Header.Set("DPoP", "a-proof")

	require.ErrorIs(t, enforceDPoPBinding(r, "dpop", "token", boundClaims()), errDPoPRequired)
}

func TestEnforceDPoPBinding_PassesTheRequestDetailsToTheValidator(t *testing.T) {
	resetDPoPBinding(t)
	var gotProof, gotMethod, gotURL, gotToken, gotJKT string
	ConfigureDPoPBinding(
		func(_ context.Context, proof, method, requestURL, accessToken, cnfJKT string) error {
			gotProof, gotMethod, gotURL, gotToken, gotJKT = proof, method, requestURL, accessToken, cnfJKT
			return nil
		},
		// htu is bound to the PUBLIC URL, not whatever internal URL the request
		// arrived on, or every proof would fail behind a proxy.
		func(r *http.Request) string { return "https://auth.example.com" + r.URL.Path },
	)

	r := httptest.NewRequest(http.MethodPost, "/api/v1/users?page=2", nil)
	r.Header.Set("DPoP", "a-proof")

	require.NoError(t, enforceDPoPBinding(r, "DPoP", "the-token", boundClaims()))
	assert.Equal(t, "a-proof", gotProof)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "https://auth.example.com/api/v1/users", gotURL)
	assert.Equal(t, "the-token", gotToken)
	assert.Equal(t, "thumb-abc", gotJKT)
}

// A mismatched key is the theft case: the thief has the token but not the key.
func TestEnforceDPoPBinding_PropagatesAValidationFailure(t *testing.T) {
	resetDPoPBinding(t)
	ConfigureDPoPBinding(func(context.Context, string, string, string, string, string) error {
		return errors.New("DPoP proof key thumbprint does not match token binding")
	}, nil)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	r.Header.Set("DPoP", "a-proof")

	err := enforceDPoPBinding(r, "dpop", "token", boundClaims())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not match token binding")
}

// An unconstrained token keeps plain bearer semantics — this must not become a
// blanket DPoP requirement for every caller.
func TestEnforceDPoPBinding_UnboundTokenIsUnaffected(t *testing.T) {
	resetDPoPBinding(t)
	ConfigureDPoPBinding(func(context.Context, string, string, string, string, string) error {
		return errors.New("must not be called")
	}, nil)

	unbound := map[string]any{"sub": "user-1"}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)

	assert.NoError(t, enforceDPoPBinding(r, "bearer", "token", unbound))
	assert.NoError(t, enforceDPoPBinding(r, "cookie", "token", unbound))
}

// Guards against the failure mode this fix actually had: the binding was enforced on
// ONE of five token-validation paths, so four others still accepted a bound token as
// a plain bearer token. Any new path that validates an access token must either
// enforce the binding or explicitly refuse sender-constrained tokens.
func TestEveryTokenEntryPointEnforcesTheBinding(t *testing.T) {
	paths := map[string]string{
		"jwt_middleware.go":                 "../middleware/jwt_middleware.go",
		"optional_auth_middleware.go":       "../middleware/optional_auth_middleware.go",
		"management_client_middleware.go":   "../middleware/management_client_middleware.go",
		"multi_issuer_middleware.go":        "../middleware/multi_issuer_middleware.go",
		"../../server/grpc_interceptors.go": "../../server/grpc_interceptors.go",
	}

	for name, path := range paths {
		body, err := os.ReadFile(path)
		require.NoError(t, err, name)
		source := string(body)

		if !strings.Contains(source, "ValidateTokenWithContext") {
			continue // not a token entry point (any more)
		}
		assert.True(t,
			strings.Contains(source, "enforceDPoPBinding") ||
				strings.Contains(source, "IsSenderConstrainedToken"),
			"%s validates access tokens but never checks the DPoP binding: a bound token "+
				"would be accepted as a plain bearer token here", name)
	}
}

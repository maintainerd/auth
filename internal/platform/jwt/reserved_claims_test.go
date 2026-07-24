package jwt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Custom claim mappers are merged into tokens last, so without this denylist a
// mapper is a token-forgery primitive: it could impersonate any subject in any
// tenant with any permission, and never expire.
func TestIsReservedClaim(t *testing.T) {
	reserved := []string{
		// RFC 7519 registered claims.
		"iss", "sub", "aud", "exp", "nbf", "iat", "jti",
		// OIDC authentication context.
		"azp", "nonce", "acr", "amr", "auth_time", "at_hash", "c_hash", "s_hash",
		// What this server authorizes on.
		"scope", "scp", "client_id", "sub_type", "tenant_id", "permissions", "roles", "sid",
		// Sender-constrained binding.
		"cnf",
	}
	for _, name := range reserved {
		assert.True(t, IsReservedClaim(name), name)
	}

	// Case and padding must not be an escape hatch.
	assert.True(t, IsReservedClaim("SUB"))
	assert.True(t, IsReservedClaim("  exp  "))
	assert.True(t, IsReservedClaim("Permissions"))

	// Organisation-specific claims are the legitimate use case.
	for _, name := range []string{"org_id", "department", "custom:tier", "https://example.com/claim", ""} {
		assert.False(t, IsReservedClaim(name), name)
	}
}

func TestSanitizeClientClaimMappers(t *testing.T) {
	t.Run("keeps non-reserved claims", func(t *testing.T) {
		safe := SanitizeClientClaimMappers(map[string]any{"org_id": "acme", "tier": "gold"})
		assert.Equal(t, map[string]any{"org_id": "acme", "tier": "gold"}, safe)
	})

	// The impersonation attempt: every reserved name must be dropped, while the
	// legitimate claim in the same payload survives.
	t.Run("drops identity, authorization and expiry claims", func(t *testing.T) {
		safe := SanitizeClientClaimMappers(map[string]any{
			"sub": "victim", "aud": "other-client", "exp": 9999999999,
			"tenant_id": int64(999), "permissions": []string{"*"}, "acr": "2",
			"iss": "evil", "cnf": "x", "sid": "y",
			"org_id": "acme",
		})
		assert.Equal(t, map[string]any{"org_id": "acme"}, safe)
	})

	// Issuance is on the login path: a bad mapper must degrade to a correct token,
	// not deny service.
	t.Run("a fully reserved mapper yields nil rather than an error", func(t *testing.T) {
		assert.Nil(t, SanitizeClientClaimMappers(map[string]any{"sub": "victim", "iss": "evil"}))
	})

	t.Run("nil and empty are safe", func(t *testing.T) {
		assert.Nil(t, SanitizeClientClaimMappers(nil))
		assert.Nil(t, SanitizeClientClaimMappers(map[string]any{}))
	})
}

package client

import (
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func redirects(uris ...string) []RedirectURIMatch {
	out := make([]RedirectURIMatch, 0, len(uris))
	for _, u := range uris {
		out = append(out, RedirectURIMatch{URI: u, Type: shared.ClientURITypeRedirect})
	}
	return out
}

func TestMatchClientRedirectURI(t *testing.T) {
	t.Run("exact match is required for ordinary redirects", func(t *testing.T) {
		reg := redirects("https://app.example.com/callback")
		require.NoError(t, MatchClientRedirectURI(reg, "https://app.example.com/callback"))

		// No prefix, subdomain, path or query flexibility — BCP §4.1.3.
		for _, candidate := range []string{
			"https://app.example.com/callback/extra",
			"https://app.example.com/callback?x=1",
			"https://evil.app.example.com/callback",
			"https://app.example.com.evil.com/callback",
			"http://app.example.com/callback",
		} {
			assert.Error(t, MatchClientRedirectURI(reg, candidate), candidate)
		}
	})

	t.Run("fails closed when nothing is registered", func(t *testing.T) {
		err := MatchClientRedirectURI(nil, "https://app.example.com/callback")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no redirect URIs registered")
	})

	t.Run("ignores URIs of other types", func(t *testing.T) {
		uris := []RedirectURIMatch{{URI: "https://app.example.com/cb", Type: shared.ClientURITypeLogout}}
		assert.Error(t, MatchClientRedirectURI(uris, "https://app.example.com/cb"))
	})

	t.Run("rejects code-executing schemes outright", func(t *testing.T) {
		reg := redirects("https://app.example.com/callback")
		for _, candidate := range []string{
			"javascript:alert(1)", "data:text/html,x", "vbscript:x", "file:///etc/passwd",
		} {
			err := MatchClientRedirectURI(reg, candidate)
			require.Error(t, err, candidate)
			assert.Contains(t, err.Error(), "forbidden scheme")
		}
	})

	// RFC 8252 §7.3: a native app binds an ephemeral loopback port, so the port
	// must not participate in the comparison. Without this, a native client using a
	// loopback redirect could never authenticate.
	t.Run("loopback redirects match on any port", func(t *testing.T) {
		reg := redirects("http://127.0.0.1:8080/callback")
		for _, candidate := range []string{
			"http://127.0.0.1:49152/callback",
			"http://127.0.0.1:1/callback",
			"http://127.0.0.1:8080/callback",
		} {
			assert.NoError(t, MatchClientRedirectURI(reg, candidate), candidate)
		}
	})

	t.Run("loopback port flexibility does not extend to path, query or host", func(t *testing.T) {
		reg := redirects("http://127.0.0.1:8080/callback")
		for _, candidate := range []string{
			"http://127.0.0.1:9000/other",        // different path
			"http://127.0.0.1:9000/callback?x=1", // different query
			"http://localhost:9000/callback",     // not an IP literal
			"http://10.0.0.1:9000/callback",      // routable address
			"https://127.0.0.1:9000/callback",    // scheme must still match
		} {
			assert.Error(t, MatchClientRedirectURI(reg, candidate), candidate)
		}
	})

	t.Run("IPv6 loopback matches on any port", func(t *testing.T) {
		reg := redirects("http://[::1]:8080/callback")
		assert.NoError(t, MatchClientRedirectURI(reg, "http://[::1]:49152/callback"))
		assert.Error(t, MatchClientRedirectURI(reg, "http://[::1]:49152/other"))
	})

	t.Run("private-use scheme still requires an exact match", func(t *testing.T) {
		reg := redirects("com.example.app:/oauth")
		assert.NoError(t, MatchClientRedirectURI(reg, "com.example.app:/oauth"))
		assert.Error(t, MatchClientRedirectURI(reg, "com.example.app:/other"))
	})
}

// Registration-time rules are stricter than the runtime denylist because a
// developer is choosing the value, and the choice differs per client type.
func TestValidateRegisteredRedirectURI(t *testing.T) {
	browserTypes := []string{shared.ClientTypeTraditional, shared.ClientTypeSPA, shared.ClientTypeM2M}

	t.Run("accepts https for every client type", func(t *testing.T) {
		for _, ct := range append(browserTypes, shared.ClientTypeMobile) {
			assert.NoError(t, ValidateRegisteredRedirectURI(ct, "https://app.example.com/cb"), ct)
		}
	})

	t.Run("accepts http only on loopback", func(t *testing.T) {
		assert.NoError(t, ValidateRegisteredRedirectURI(shared.ClientTypeSPA, "http://127.0.0.1:3000/cb"))
		assert.NoError(t, ValidateRegisteredRedirectURI(shared.ClientTypeSPA, "http://[::1]:3000/cb"))

		err := ValidateRegisteredRedirectURI(shared.ClientTypeSPA, "http://app.example.com/cb")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must use https")
	})

	// OIDC Core §3.1.2.1 — the authorization response appends its own fragment.
	t.Run("rejects a fragment", func(t *testing.T) {
		err := ValidateRegisteredRedirectURI(shared.ClientTypeSPA, "https://app.example.com/cb#part")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fragment")
	})

	t.Run("rejects embedded credentials", func(t *testing.T) {
		err := ValidateRegisteredRedirectURI(shared.ClientTypeSPA, "https://user:pw@app.example.com/cb")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "credentials")
	})

	t.Run("rejects a relative URI", func(t *testing.T) {
		err := ValidateRegisteredRedirectURI(shared.ClientTypeSPA, "/callback")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "absolute")
	})

	t.Run("rejects code-executing schemes", func(t *testing.T) {
		for _, uri := range []string{"javascript:alert(1)", "data:text/html,x"} {
			assert.Error(t, ValidateRegisteredRedirectURI(shared.ClientTypeSPA, uri), uri)
		}
	})

	// A private-use scheme is how a native app receives the response, and is
	// meaningless (and dangerous as an open redirect) for a browser client.
	t.Run("private-use scheme is mobile-only", func(t *testing.T) {
		assert.NoError(t, ValidateRegisteredRedirectURI(shared.ClientTypeMobile, "com.example.app:/oauth"))

		for _, ct := range browserTypes {
			err := ValidateRegisteredRedirectURI(ct, "com.example.app:/oauth")
			require.Error(t, err, ct)
			assert.Contains(t, err.Error(), "only allowed for mobile")
		}
	})

	// RFC 8252 §7.1 recommends a reverse-domain scheme so it cannot collide with
	// another app's.
	t.Run("mobile custom scheme must be reverse-domain", func(t *testing.T) {
		err := ValidateRegisteredRedirectURI(shared.ClientTypeMobile, "myapp:/oauth")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reverse-domain")
	})
}

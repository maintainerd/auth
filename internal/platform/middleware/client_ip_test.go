package middleware

import (
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExtractClientIP_TrustBoundary covers provenance: extractClientIP is the key
// every per-IP control uses — rate limits,
// registration abuse counters, tenant IP restrictions. If a caller can choose
// its own key, all of them become advisory, so provenance is asserted here
// rather than just syntax.
// resetTrustedProxies clears the once-cached trusted-proxy config so a test can
// install its own. The config is package-level and resolved once at first use,
// so any test that changes TRUSTED_PROXY_CIDRS / TRUST_ALL_PROXIES must call
// this first or it will observe (and leak) another test's setup.
func resetTrustedProxies() {
	trustedProxyNets = nil
	trustAllProxies = false
	trustedProxyOnce = sync.Once{}
}

func TestExtractClientIP_TrustBoundary(t *testing.T) {
	reset := resetTrustedProxies
	t.Cleanup(resetTrustedProxies)

	newReq := func(remoteAddr string, headers map[string]string) *http.Request {
		r, _ := http.NewRequest(http.MethodPost, "/register", nil)
		r.RemoteAddr = remoteAddr
		for k, v := range headers {
			r.Header.Set(k, v)
		}
		return r
	}

	t.Run("untrusted peer: forwarding headers are ignored", func(t *testing.T) {
		reset()
		t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
		r := newReq("203.0.113.9:44321", map[string]string{
			"X-Forwarded-For":  "1.2.3.4",
			"X-Real-IP":        "5.6.7.8",
			"CF-Connecting-IP": "9.9.9.9",
		})
		// The spoofed headers must not shift the key off the real peer.
		assert.Equal(t, "203.0.113.9", extractClientIP(r))
	})

	t.Run("untrusted peer cannot rotate the rate-limit key", func(t *testing.T) {
		reset()
		t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
		first := extractClientIP(newReq("203.0.113.9:1111", map[string]string{"X-Forwarded-For": "1.1.1.1"}))
		second := extractClientIP(newReq("203.0.113.9:2222", map[string]string{"X-Forwarded-For": "2.2.2.2"}))
		assert.Equal(t, first, second, "a rotating header must not produce a new limiter key")
	})

	t.Run("trusted proxy: rightmost non-proxy entry wins", func(t *testing.T) {
		reset()
		t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
		// Client-supplied prefix, then the real client, then our own hop.
		r := newReq("10.0.0.5:44321", map[string]string{
			"X-Forwarded-For": "1.1.1.1, 203.0.113.7, 10.0.0.4",
		})
		assert.Equal(t, "203.0.113.7", extractClientIP(r),
			"walking from the right skips our hops and ignores a client-supplied prefix")
	})

	t.Run("trusted proxy: single entry is taken", func(t *testing.T) {
		reset()
		t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
		r := newReq("10.0.0.5:44321", map[string]string{"X-Forwarded-For": "203.0.113.7"})
		assert.Equal(t, "203.0.113.7", extractClientIP(r))
	})

	t.Run("trusted proxy: falls back to X-Real-IP then peer", func(t *testing.T) {
		reset()
		t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
		r := newReq("10.0.0.5:44321", map[string]string{"X-Real-IP": "203.0.113.7"})
		assert.Equal(t, "203.0.113.7", extractClientIP(r))

		reset()
		t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
		bare := newReq("10.0.0.5:44321", nil)
		assert.Equal(t, "10.0.0.5", extractClientIP(bare))
	})

	t.Run("garbage header falls through instead of being trusted", func(t *testing.T) {
		reset()
		t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
		r := newReq("10.0.0.5:44321", map[string]string{"X-Forwarded-For": "not-an-ip, also-bad"})
		assert.Equal(t, "10.0.0.5", extractClientIP(r))
	})

	t.Run("loopback is trusted by default", func(t *testing.T) {
		reset()
		r := newReq("127.0.0.1:44321", map[string]string{"X-Forwarded-For": "203.0.113.7"})
		assert.Equal(t, "203.0.113.7", extractClientIP(r))
	})

	t.Run("public peer is not trusted by default", func(t *testing.T) {
		reset()
		r := newReq("203.0.113.9:44321", map[string]string{"X-Forwarded-For": "1.2.3.4"})
		assert.Equal(t, "203.0.113.9", extractClientIP(r))
	})

	t.Run("TRUST_ALL_PROXIES restores header trust", func(t *testing.T) {
		reset()
		t.Setenv("TRUST_ALL_PROXIES", "true")
		r := newReq("203.0.113.9:44321", map[string]string{"X-Forwarded-For": "1.2.3.4, 5.6.7.8"})
		assert.Equal(t, "5.6.7.8", extractClientIP(r))
	})

	t.Run("IPv6 peer and header", func(t *testing.T) {
		reset()
		t.Setenv("TRUSTED_PROXY_CIDRS", "fc00::/7")
		r := newReq("[fc00::1]:44321", map[string]string{"X-Forwarded-For": "2001:db8::5"})
		assert.Equal(t, "2001:db8::5", extractClientIP(r))
	})
}

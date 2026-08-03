package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeOrigin(t *testing.T) {
	// A browser's Origin header is always scheme://host[:port] with no path and
	// a lowercase host. Registrations are hand-entered, so they must be reduced
	// to the same shape or a perfectly valid entry silently never matches.
	t.Run("equivalent forms normalise to the same origin", func(t *testing.T) {
		for _, in := range []string{
			"https://app.thirdparty.example",
			"https://app.thirdparty.example/",
			"https://APP.ThirdParty.Example",
			"  https://app.thirdparty.example/callback  ",
		} {
			assert.Equal(t, "https://app.thirdparty.example", normalizeOrigin(in), "input %q", in)
		}
	})

	t.Run("port and scheme are part of the origin", func(t *testing.T) {
		assert.Equal(t, "https://app.example:8443", normalizeOrigin("https://app.example:8443"))
		// Different scheme and different port are different origins — collapsing
		// them would let an http site borrow an https registration.
		assert.NotEqual(t, normalizeOrigin("http://app.example"), normalizeOrigin("https://app.example"))
		assert.NotEqual(t, normalizeOrigin("https://app.example"), normalizeOrigin("https://app.example:8443"))
	})

	// Everything below must be rejected outright: returning "" means "deny".
	t.Run("non-origins are rejected", func(t *testing.T) {
		for _, in := range []string{
			"",
			"   ",
			// Sandboxed iframes and file:// contexts send the literal "null".
			// Reflecting it back grants any opaque origin access.
			"null",
			"NULL",
			"javascript:alert(1)",
			"data:text/html,<script>",
			"file:///etc/passwd",
			"ftp://app.example",
			// A bare host is not an origin and must not be coerced into one.
			"app.thirdparty.example",
			"/callback",
		} {
			assert.Empty(t, normalizeOrigin(in), "input %q must be rejected", in)
		}
	})
}

func TestCORSOriginResolver_FailsClosed(t *testing.T) {
	// A nil resolver or an unconfigured DB must deny rather than panic — the
	// middleware calls this on the token endpoint's hot path.
	var nilResolver *CORSOriginResolver
	assert.False(t, nilResolver.IsAllowedCORSOrigin(t.Context(), "https://app.example"))

	empty := &CORSOriginResolver{}
	assert.False(t, empty.IsAllowedCORSOrigin(t.Context(), "https://app.example"))
	assert.False(t, empty.IsAllowedCORSOrigin(t.Context(), ""))
}

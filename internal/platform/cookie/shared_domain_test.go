package cookie

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withCookieDomain(t *testing.T, domain string) {
	t.Helper()
	prev := config.CookieDomain
	config.CookieDomain = domain
	t.Cleanup(func() { config.CookieDomain = prev })
}

func setCookieHeaders(w *httptest.ResponseRecorder) []string {
	return w.Result().Header["Set-Cookie"]
}

// Without a shared domain the strongest prefix applies: host-only, no Domain.
func TestAuthCookies_HostOnlyByDefault(t *testing.T) {
	withCookieDomain(t, "")
	w := httptest.NewRecorder()

	SetAuthCookies(w, map[string]interface{}{"access_token": "tok"})

	joined := strings.Join(setCookieHeaders(w), "\n")
	assert.Contains(t, joined, "__Host-access_token=tok")
	assert.NotContains(t, joined, "Domain=")
}

// With a shared domain every first-party surface under it must receive the same
// cookie — that is what makes sign-in and sign-out mutual across the console and
// identity apps. __Host- forbids Domain, so the prefix must step down.
func TestAuthCookies_SharedDomainSpansSubdomains(t *testing.T) {
	withCookieDomain(t, "auth.example.com")
	w := httptest.NewRecorder()

	SetAuthCookies(w, map[string]interface{}{"access_token": "tok"})

	joined := strings.Join(setCookieHeaders(w), "\n")
	assert.Contains(t, joined, "__Secure-access_token=tok")
	assert.Contains(t, joined, "Domain=auth.example.com")
	// __Host- with a Domain would be silently rejected by the browser.
	assert.NotContains(t, joined, "__Host-access_token=tok")
}

func TestAuthCookies_SharedDomainNormalizesLeadingDot(t *testing.T) {
	withCookieDomain(t, ".auth.example.com")
	w := httptest.NewRecorder()

	SetAuthCookies(w, map[string]interface{}{"access_token": "tok"})

	assert.Contains(t, strings.Join(setCookieHeaders(w), "\n"), "Domain=auth.example.com")
}

// Logout must clear BOTH prefixes. Flipping COOKIE_DOMAIN changes which one is
// issued; a cookie stranded under the other name would keep a "logged out" user
// signed in.
func TestClearAuthCookies_ClearsBothPrefixes(t *testing.T) {
	for _, domain := range []string{"", "auth.example.com"} {
		withCookieDomain(t, domain)
		w := httptest.NewRecorder()

		ClearAuthCookies(w)

		joined := strings.Join(setCookieHeaders(w), "\n")
		for _, name := range []string{
			"__Host-access_token=", "__Secure-access_token=", "access_token=",
			"__Host-id_token=", "__Secure-id_token=",
		} {
			assert.Contains(t, joined, name, "domain=%q must clear %s", domain, name)
		}
		assert.Contains(t, joined, "Max-Age=0")
	}
}

// Readers must accept whichever prefix the deployment issues, or a shared-domain
// deployment cannot see its own session.
func TestAccessTokenReadersAcceptBothPrefixes(t *testing.T) {
	for _, name := range []string{"__Host-access_token", "__Secure-access_token", "access_token"} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: name, Value: "tok"})

		c, err := r.Cookie(name)
		require.NoError(t, err)
		assert.Equal(t, "tok", c.Value)
	}
}

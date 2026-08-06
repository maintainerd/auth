package client

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	// middleware calls this on the token endpoint's hot path. A resolvable tenant
	// is supplied so the denial is attributable to the resolver, not to the
	// tenant-scope check.
	onSurface := middleware.WithRequestTenant(t.Context(), middleware.RequestTenant{Slug: "acme", OK: true})

	var nilResolver *CORSOriginResolver
	assert.False(t, nilResolver.IsAllowedCORSOrigin(onSurface, "https://app.example"))

	empty := &CORSOriginResolver{}
	assert.False(t, empty.IsAllowedCORSOrigin(onSurface, "https://app.example"))
	assert.False(t, empty.IsAllowedCORSOrigin(onSurface, ""))
}

// ===========================================================================
// End-to-end: the real resolver behind the real middleware stack
// ===========================================================================

// The unit tests above hand the resolver a context that already carries a
// RequestTenant, and the middleware-side tests stub the resolver out entirely,
// so between them nobody exercised the seam: the tenant-scoped resolver is only
// reachable if middleware.RequestTenantMiddleware has already run on the same
// chain. It had not been mounted where CORSMiddleware runs, so the scope lookup
// always failed closed and NO operator-registered origin could ever be allowed —
// the exact third-party-SPA breakage the feature exists to prevent, reintroduced
// silently. These tests drive the real CORSMiddleware over the real
// CORSOriginResolver and assert on the response headers, so the seam cannot come
// apart again unnoticed.

// corsStack wires the real resolver into the real CORS middleware. When
// mountRequestTenant is false the stack reproduces the broken chain.
func corsStack(t *testing.T, mountRequestTenant bool, rows ...[3]any) (http.Handler, sqlmock.Sqlmock) {
	t.Helper()

	// Only the client registry may decide this; the env allow-list and the
	// tenant-host rule must not be able to mask the result.
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	origIdentity, origConsole := config.AppFrontendIdentityHostname, config.AppFrontendConsoleHostname
	config.AppFrontendIdentityHostname = "auth.example.com"
	config.AppFrontendConsoleHostname = "console.auth.example.com"
	t.Cleanup(func() {
		config.AppFrontendIdentityHostname = origIdentity
		config.AppFrontendConsoleHostname = origConsole
	})

	gormDB, mock := newMockGormDBRegex(t)
	mock.ExpectQuery(`client_uris`).WillReturnRows(corsOriginRows(rows...))

	middleware.SetCORSOriginResolver(NewCORSOriginResolver(gormDB))
	t.Cleanup(func() { middleware.SetCORSOriginResolver(nil) })

	var h http.Handler = middleware.CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if mountRequestTenant {
		h = middleware.RequestTenantMiddleware(h)
	}
	return h, mock
}

func corsProbe(h http.Handler, host, origin string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/oauth/token", nil)
	r.Host = host
	r.Header.Set("Origin", origin)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestCORSMiddleware_AllowsARegisteredOriginThroughTheRealResolver(t *testing.T) {
	const registered = "https://app.thirdparty.example"

	t.Run("registered origin on its own tenant's host is allowed with credentials", func(t *testing.T) {
		h, mock := corsStack(t, true, [3]any{"acme", false, registered})

		w := corsProbe(h, "acme.auth.example.com", registered)

		assert.Equal(t, registered, w.Header().Get("Access-Control-Allow-Origin"),
			"an origin an operator registered in the console must actually be allowed")
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "Origin", w.Header().Get("Vary"))
		require.NoError(t, mock.ExpectationsWereMet(), "the registry must actually have been consulted")
	})

	// The whole point of the tenant scope: same registration, different tenant's
	// surface, no credentialed access.
	t.Run("the same origin is denied on another tenant's host", func(t *testing.T) {
		h, _ := corsStack(t, true, [3]any{"acme", false, registered})

		w := corsProbe(h, "other.auth.example.com", registered)

		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))
	})

	// This is the regression guard. Deleting RequestTenantMiddleware from the
	// chain that CORSMiddleware runs on takes the feature out entirely, and
	// because it fails closed it does so without a single error anywhere.
	t.Run("without RequestTenantMiddleware the registry can never allow anything", func(t *testing.T) {
		h, mock := corsStack(t, false, [3]any{"acme", false, registered})

		w := corsProbe(h, "acme.auth.example.com", registered)

		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"),
			"documents the dependency: CORSMiddleware must run downstream of RequestTenantMiddleware")
		// It never even gets as far as a lookup, which is why no log line and no
		// error ever pointed at the missing mount.
		assert.Error(t, mock.ExpectationsWereMet(), "the registry is not even consulted without a request tenant")
	})
}

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withTenantBases(t *testing.T) {
	t.Helper()
	origI := config.AppFrontendIdentityHostname
	origC := config.AppFrontendConsoleHostname
	config.AppFrontendIdentityHostname = "auth.example.com"
	config.AppFrontendConsoleHostname = "console.auth.example.com"
	t.Cleanup(func() {
		config.AppFrontendIdentityHostname = origI
		config.AppFrontendConsoleHostname = origC
	})
}

func TestResolveRequestTenant_SignalPriority(t *testing.T) {
	withTenantBases(t)

	// Origin wins over X-Forwarded-Host and Host (cross-origin prod).
	t.Run("origin is preferred", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Host = "acme.auth.example.com"
		r.Header.Set("X-Forwarded-Host", "beta.auth.example.com")
		r.Header.Set("Origin", "https://gamma.auth.example.com")

		rt := ResolveRequestTenant(r)
		assert.True(t, rt.OK)
		assert.Equal(t, "gamma", rt.Slug)
		assert.False(t, rt.IsSystem)
	})

	// No Origin (same-origin via per-app nginx): X-Forwarded-Host is used next.
	t.Run("x-forwarded-host used when origin absent", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Host = "acme.auth.example.com"
		r.Header.Set("X-Forwarded-Host", "beta.auth.example.com")

		rt := ResolveRequestTenant(r)
		assert.True(t, rt.OK)
		assert.Equal(t, "beta", rt.Slug)
	})

	// Neither Origin nor X-Forwarded-Host: fall back to Host (proxy_set_header
	// Host $host in nginx).
	t.Run("host used as last resort", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Host = "acme.auth.example.com"

		rt := ResolveRequestTenant(r)
		assert.True(t, rt.OK)
		assert.Equal(t, "acme", rt.Slug)
	})

	// Bare system-tenant base → OK with empty slug and IsSystem true.
	t.Run("bare base is the system tenant", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Host = "auth.example.com"

		rt := ResolveRequestTenant(r)
		assert.True(t, rt.OK)
		assert.Empty(t, rt.Slug)
		assert.True(t, rt.IsSystem)
	})

	// Unrecognized host → OK false so callers fall back to existing behavior.
	t.Run("unrecognized host is not ok", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Host = "evil.com"
		r.Header.Set("Origin", "https://evil.com")

		rt := ResolveRequestTenant(r)
		assert.False(t, rt.OK)
	})

	// A recognized Host still resolves even when Origin is an unrecognized value:
	// the resolver picks the first signal that ResolveTenantHost recognizes.
	t.Run("unrecognized origin falls through to recognized host", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Host = "acme.auth.example.com"
		r.Header.Set("Origin", "https://evil.com")

		rt := ResolveRequestTenant(r)
		assert.True(t, rt.OK)
		assert.Equal(t, "acme", rt.Slug)
	})
}

func TestRequestTenantMiddleware_StoresInContext(t *testing.T) {
	withTenantBases(t)

	var got RequestTenant
	var ok bool
	handler := RequestTenantMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got, ok = RequestTenantFromContext(r.Context())
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Host = "acme.auth.example.com"
	handler.ServeHTTP(httptest.NewRecorder(), r)

	require.True(t, ok)
	assert.True(t, got.OK)
	assert.Equal(t, "acme", got.Slug)
}

func TestRequestTenantFromContext_Absent(t *testing.T) {
	_, ok := RequestTenantFromContext(httptest.NewRequest(http.MethodGet, "/", nil).Context())
	assert.False(t, ok)
}

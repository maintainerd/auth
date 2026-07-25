package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubMaintenanceReader struct {
	cfgByTenant map[int64]map[string]any
}

func (s *stubMaintenanceReader) GetMaintenanceConfig(_ context.Context, tenantID int64) (map[string]any, error) {
	return s.cfgByTenant[tenantID], nil
}

func runAuthMaintenance(reader TenantMaintenanceReader, resolver TenantSlugResolver, host string) (*httptest.ResponseRecorder, bool) {
	nextCalled := false
	h := AuthEndpointMaintenanceMiddleware(reader, resolver)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	}))
	r := httptest.NewRequest(http.MethodPost, "/login", nil)
	r.Host = host
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec, nextCalled
}

func TestAuthEndpointMaintenance(t *testing.T) {
	withTenantBases(t) // identity=auth.example.com, console=console.auth.example.com
	reader := &stubMaintenanceReader{cfgByTenant: map[int64]map[string]any{
		1: {"enabled": true, "message": "back soon"},
	}}
	resolver := &stubSlugResolver{idBySlug: map[string]int64{"acme": 1}}

	t.Run("identity-surface login is blocked during maintenance", func(t *testing.T) {
		rec, next := runAuthMaintenance(reader, resolver, "acme.auth.example.com")
		assert.False(t, next)
		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	})

	// Console/admin login must NOT be blocked, so an operator can always sign in
	// to lift maintenance.
	t.Run("console-surface login is never blocked", func(t *testing.T) {
		_, next := runAuthMaintenance(reader, resolver, "acme.console.auth.example.com")
		assert.True(t, next)
	})

	t.Run("no maintenance window: allowed", func(t *testing.T) {
		reader2 := &stubMaintenanceReader{cfgByTenant: map[int64]map[string]any{1: {"enabled": false}}}
		_, next := runAuthMaintenance(reader2, resolver, "acme.auth.example.com")
		assert.True(t, next)
	})

	t.Run("unresolved host: allowed (no tenant)", func(t *testing.T) {
		_, next := runAuthMaintenance(reader, resolver, "random.other.com")
		assert.True(t, next)
	})
}

// Fail-safe: a rate-limit config that is enabled but omits requests_per_window
// must NOT become "enabled but unlimited" — it falls back to the default.
func TestMapToRequestRateLimitConfig_FailSafeDefault(t *testing.T) {
	rc := mapToRequestRateLimitConfig(map[string]any{"enabled": true})
	assert.True(t, rc.Enabled)
	assert.Equal(t, defaultRequestsPerWindow, rc.RequestsPerWindow, "missing limit must not mean unlimited")

	// A disabled config is left as-is (no enforcement, so no floor needed).
	rcOff := mapToRequestRateLimitConfig(map[string]any{"enabled": false})
	assert.False(t, rcOff.Enabled)
	assert.Equal(t, 0, rcOff.RequestsPerWindow)

	// An explicit positive limit is preserved.
	rcSet := mapToRequestRateLimitConfig(map[string]any{"enabled": true, "requests_per_window": 25})
	assert.Equal(t, 25, rcSet.RequestsPerWindow)
}

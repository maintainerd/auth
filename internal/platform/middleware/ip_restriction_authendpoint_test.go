package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubIPReader struct {
	rulesByTenant map[int64][]IPRestriction
	errByTenant   map[int64]error
	calls         []int64
}

func (s *stubIPReader) GetActiveIPRestrictions(_ context.Context, tenantID int64) ([]IPRestriction, error) {
	s.calls = append(s.calls, tenantID)
	if s.errByTenant != nil {
		if err, ok := s.errByTenant[tenantID]; ok {
			return nil, err
		}
	}
	return s.rulesByTenant[tenantID], nil
}

type stubSlugResolver struct {
	idBySlug map[string]int64
	err      error
}

func (s *stubSlugResolver) ResolveTenantIDBySlug(_ context.Context, slug string) (int64, bool, error) {
	if s.err != nil {
		return 0, false, s.err
	}
	id, ok := s.idBySlug[slug]
	return id, ok, nil
}

func authEndpointRequest(host string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/login", nil)
	r.Host = host
	// httptest default RemoteAddr is 192.0.2.1:1234, and without a trusted proxy
	// extractClientIP uses it, so the client IP under test is 192.0.2.1.
	return r
}

func runAuthEndpoint(reader TenantIPRestrictionReader, resolver TenantSlugResolver, r *http.Request) (*httptest.ResponseRecorder, bool) {
	nextCalled := false
	h := AuthEndpointIPRestrictionMiddleware(reader, resolver)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec, nextCalled
}

func TestAuthEndpointIPRestriction(t *testing.T) {
	withTenantBases(t)

	t.Run("unrecognized host: nothing to enforce, allowed through", func(t *testing.T) {
		reader := &stubIPReader{}
		resolver := &stubSlugResolver{idBySlug: map[string]int64{"acme": 1}}
		_, next := runAuthEndpoint(reader, resolver, authEndpointRequest("random.other.com"))
		assert.True(t, next)
		assert.Empty(t, reader.calls, "no tenant → rules never loaded")
	})

	t.Run("tenant with no rules is allowed", func(t *testing.T) {
		reader := &stubIPReader{rulesByTenant: map[int64][]IPRestriction{}}
		resolver := &stubSlugResolver{idBySlug: map[string]int64{"acme": 1}}
		_, next := runAuthEndpoint(reader, resolver, authEndpointRequest("acme.auth.example.com"))
		assert.True(t, next)
		assert.Equal(t, []int64{1}, reader.calls, "rules loaded for the request's tenant only")
	})

	t.Run("allowlist: matching client IP passes", func(t *testing.T) {
		reader := &stubIPReader{rulesByTenant: map[int64][]IPRestriction{
			1: {{Type: "allow", IPAddress: "192.0.2.0/24"}},
		}}
		resolver := &stubSlugResolver{idBySlug: map[string]int64{"acme": 1}}
		_, next := runAuthEndpoint(reader, resolver, authEndpointRequest("acme.auth.example.com"))
		assert.True(t, next)
	})

	t.Run("allowlist: non-matching client IP is blocked with 403", func(t *testing.T) {
		reader := &stubIPReader{rulesByTenant: map[int64][]IPRestriction{
			1: {{Type: "allow", IPAddress: "10.0.0.0/8"}},
		}}
		resolver := &stubSlugResolver{idBySlug: map[string]int64{"acme": 1}}
		rec, next := runAuthEndpoint(reader, resolver, authEndpointRequest("acme.auth.example.com"))
		assert.False(t, next)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("deny rule blocks a matching client IP", func(t *testing.T) {
		reader := &stubIPReader{rulesByTenant: map[int64][]IPRestriction{
			1: {{Type: "deny", IPAddress: "192.0.2.1"}},
		}}
		resolver := &stubSlugResolver{idBySlug: map[string]int64{"acme": 1}}
		rec, next := runAuthEndpoint(reader, resolver, authEndpointRequest("acme.auth.example.com"))
		assert.False(t, next)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	// Fail CLOSED: a resolved tenant whose rules cannot be loaded (cold error, no
	// cache) must not be reachable — it may have an allowlist we cannot verify.
	t.Run("cold rule-load error fails closed with 503", func(t *testing.T) {
		reader := &stubIPReader{errByTenant: map[int64]error{1: errors.New("db down")}}
		resolver := &stubSlugResolver{idBySlug: map[string]int64{"acme": 1}}
		rec, next := runAuthEndpoint(reader, resolver, authEndpointRequest("acme.auth.example.com"))
		assert.False(t, next, "must not proceed to the credential handler")
		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	})

	// Per-tenant isolation: tenant A's allowlist must never gate tenant B. A
	// request on tenant B's subdomain loads only B's (empty) rules and passes,
	// even though tenant A has a restrictive allowlist that the same client fails.
	t.Run("one tenant's allowlist does not affect another tenant", func(t *testing.T) {
		reader := &stubIPReader{rulesByTenant: map[int64][]IPRestriction{
			1: {{Type: "allow", IPAddress: "10.0.0.0/8"}}, // tenant A: would block 192.0.2.1
			2: {},                                         // tenant B: no rules
		}}
		resolver := &stubSlugResolver{idBySlug: map[string]int64{"acme": 1, "beta": 2}}

		recA, nextA := runAuthEndpoint(reader, resolver, authEndpointRequest("acme.auth.example.com"))
		assert.False(t, nextA)
		assert.Equal(t, http.StatusForbidden, recA.Code)

		_, nextB := runAuthEndpoint(reader, resolver, authEndpointRequest("beta.auth.example.com"))
		assert.True(t, nextB, "tenant B has no rules; tenant A's allowlist must not apply")
	})

	// A slug that resolves to no tenant is not a load failure — allow through.
	t.Run("unknown slug is allowed (no tenant to enforce)", func(t *testing.T) {
		reader := &stubIPReader{}
		resolver := &stubSlugResolver{idBySlug: map[string]int64{}} // nothing resolves
		_, next := runAuthEndpoint(reader, resolver, authEndpointRequest("ghost.auth.example.com"))
		assert.True(t, next)
		assert.Empty(t, reader.calls)
	})
}

// The cache serves the last known ruleset through a transient error (fail to
// last known state), so a blip neither over-blocks nor drops enforcement.
func TestIPRestrictionCache_ServesStaleOnTransientError(t *testing.T) {
	cache := newIPRestrictionCache()
	reader := &stubIPReader{rulesByTenant: map[int64][]IPRestriction{1: {{Type: "allow", IPAddress: "10.0.0.0/8"}}}}

	// Prime the cache.
	rules, ok := cache.get(context.Background(), reader, 1)
	require.True(t, ok)
	require.Len(t, rules, 1)

	// Expire it, then make the reader error: stale rules are still served.
	cache.entries[1] = ipRestrictionCacheEntry{rules: rules} // expiresAt zero → expired
	reader.errByTenant = map[int64]error{1: errors.New("blip")}
	rules, ok = cache.get(context.Background(), reader, 1)
	assert.True(t, ok, "transient error with a cached entry serves stale")
	assert.Len(t, rules, 1)
}

func TestIPRestrictionCache_ColdErrorReportsNotLoaded(t *testing.T) {
	cache := newIPRestrictionCache()
	reader := &stubIPReader{errByTenant: map[int64]error{1: errors.New("db down")}}
	rules, ok := cache.get(context.Background(), reader, 1)
	assert.False(t, ok, "cold error with no cache → not loaded (caller fails closed)")
	assert.Nil(t, rules)
}

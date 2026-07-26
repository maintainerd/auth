package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func TestIPRateLimitMiddleware_NilRedis(t *testing.T) {
	handler := IPRateLimitMiddleware(nil, 10, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIPRateLimitMiddleware_UnderLimit(t *testing.T) {
	rdb := newTestRedisClient(t)
	handler := IPRateLimitMiddleware(rdb, 10, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIPRateLimitMiddleware_ExceedsLimit(t *testing.T) {
	rdb := newTestRedisClient(t)
	limit := 2
	handler := IPRateLimitMiddleware(rdb, limit, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ip := "10.0.0.55:9999"
	// Send requests up to limit (pass).
	for i := 0; i < limit; i++ {
		r := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
		r.RemoteAddr = ip
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		require.Equal(t, http.StatusOK, w.Code, "request %d should pass", i+1)
	}

	// Next request must be rate limited.
	r := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	r.RemoteAddr = ip
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}

func TestIPRateLimitMiddleware_DifferentPathsSeparateKeys(t *testing.T) {
	rdb := newTestRedisClient(t)
	handler := IPRateLimitMiddleware(rdb, 1, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ip := "172.16.0.1:8080"

	// First path: hit limit.
	r1 := httptest.NewRequest(http.MethodGet, "/path-a", nil)
	r1.RemoteAddr = ip
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, r1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Second request to same path: blocked.
	r1b := httptest.NewRequest(http.MethodGet, "/path-a", nil)
	r1b.RemoteAddr = ip
	w1b := httptest.NewRecorder()
	handler.ServeHTTP(w1b, r1b)
	assert.Equal(t, http.StatusTooManyRequests, w1b.Code)

	// Different path: still passes (separate rate-limit key).
	r2 := httptest.NewRequest(http.MethodGet, "/path-b", nil)
	r2.RemoteAddr = ip
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, r2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestIPRateLimitMiddleware_DifferentIPsSeparateKeys(t *testing.T) {
	rdb := newTestRedisClient(t)
	handler := IPRateLimitMiddleware(rdb, 1, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// IP A: first request passes.
	rA := httptest.NewRequest(http.MethodGet, "/api", nil)
	rA.RemoteAddr = "10.0.0.1:12345"
	wA := httptest.NewRecorder()
	handler.ServeHTTP(wA, rA)
	assert.Equal(t, http.StatusOK, wA.Code)

	// IP A: second request blocked.
	rA2 := httptest.NewRequest(http.MethodGet, "/api", nil)
	rA2.RemoteAddr = "10.0.0.1:12345"
	wA2 := httptest.NewRecorder()
	handler.ServeHTTP(wA2, rA2)
	assert.Equal(t, http.StatusTooManyRequests, wA2.Code)

	// IP B: still passes (different IP = different key).
	rB := httptest.NewRequest(http.MethodGet, "/api", nil)
	rB.RemoteAddr = "10.0.0.2:12345"
	wB := httptest.NewRecorder()
	handler.ServeHTTP(wB, rB)
	assert.Equal(t, http.StatusOK, wB.Code)
}

func TestIPRateLimitMiddleware_DoesNotRefreshTTLOnEveryHit(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	handler := IPRateLimitMiddleware(rdb, 10, 10*time.Second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.RemoteAddr = "10.0.0.10:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	mr.FastForward(2 * time.Second)

	req = httptest.NewRequest(http.MethodGet, "/api", nil)
	req.RemoteAddr = "10.0.0.10:12345"
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	ttl, err := rdb.TTL(context.Background(), "rl:ip:10.0.0.10:/api").Result()
	require.NoError(t, err)
	assert.Less(t, ttl, 9*time.Second)
}

type tenantRateLimitReaderStub struct {
	config map[string]any
	err    error
	calls  int
}

func (s *tenantRateLimitReaderStub) GetRateLimitConfig(context.Context, int64) (map[string]any, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.config, nil
}

func tenantRateLimitedHandler(t *testing.T, rdb *redis.Client, cfg map[string]any) http.Handler {
	t.Helper()
	return TenantRequestRateLimitMiddleware(rdb, &tenantRateLimitReaderStub{config: cfg})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
}

func tenantRateLimitRequest(method, path, remoteAddr string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.RemoteAddr = remoteAddr
	return WithAuthContext(req, &authctx.AuthContext{
		Tenant: &authctx.AuthTenant{TenantID: 42},
	})
}

func TestTenantRequestRateLimitMiddleware_DisabledConfigPasses(t *testing.T) {
	rdb := newTestRedisClient(t)
	handler := tenantRateLimitedHandler(t, rdb, map[string]any{
		"enabled":             false,
		"requests_per_window": 1,
		"per_ip":              true,
	})

	for range 3 {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, tenantRateLimitRequest(http.MethodGet, "/tenant-settings/rate-limit", "10.0.0.1:1234"))
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestTenantRateLimitConfigCache_CachesDecodedConfig(t *testing.T) {
	reader := &tenantRateLimitReaderStub{config: map[string]any{
		"enabled":             true,
		"requests_per_window": 5,
	}}
	cache := newTenantRateLimitConfigCache(time.Minute)

	first := cache.get(context.Background(), reader, 42)
	second := cache.get(context.Background(), reader, 42)

	require.NotNil(t, first)
	require.NotNil(t, second)
	assert.Equal(t, 1, reader.calls)
	assert.Equal(t, 5, second.RequestsPerWindow)
}

func TestTenantRequestRateLimitMiddleware_PerIPLimit(t *testing.T) {
	rdb := newTestRedisClient(t)
	handler := tenantRateLimitedHandler(t, rdb, map[string]any{
		"enabled":                 true,
		"requests_per_window":     1,
		"window_duration_seconds": 60,
		"per_ip":                  true,
	})

	req := tenantRateLimitRequest(http.MethodGet, "/tenant-settings/rate-limit", "10.0.0.2:1234")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = tenantRateLimitRequest(http.MethodGet, "/tenant-settings/rate-limit", "10.0.0.2:1234")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}

func TestTenantRequestRateLimitMiddleware_EndpointOverride(t *testing.T) {
	rdb := newTestRedisClient(t)
	handler := tenantRateLimitedHandler(t, rdb, map[string]any{
		"enabled":             true,
		"requests_per_window": 10,
		"per_ip":              true,
		"endpoint_overrides": map[string]any{
			"/tenant-settings/rate-limit": 1,
		},
	})

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, tenantRateLimitRequest(http.MethodGet, "/tenant-settings/rate-limit", "10.0.0.3:1234"))
		if i == 0 {
			assert.Equal(t, http.StatusOK, w.Code)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, w.Code)
		}
	}
}

func TestTenantRequestRateLimitMiddleware_ExemptIPPasses(t *testing.T) {
	rdb := newTestRedisClient(t)
	handler := tenantRateLimitedHandler(t, rdb, map[string]any{
		"enabled":             true,
		"requests_per_window": 1,
		"per_ip":              true,
		"exempt_ips":          []any{"10.0.0.4"},
	})

	for range 3 {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, tenantRateLimitRequest(http.MethodGet, "/tenant-settings/rate-limit", "10.0.0.4:1234"))
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestTenantRequestRateLimitMiddleware_TenantWideLimitWhenNotPerIP(t *testing.T) {
	rdb := newTestRedisClient(t)
	handler := tenantRateLimitedHandler(t, rdb, map[string]any{
		"enabled":             true,
		"requests_per_window": 1,
	})

	// per_ip defaults to false → the limiter keys per tenant+path, so the counter
	// is shared across clients regardless of source IP.
	req := tenantRateLimitRequest(http.MethodGet, "/tenant-settings/rate-limit", "10.0.0.5:1234")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = tenantRateLimitRequest(http.MethodGet, "/tenant-settings/rate-limit", "10.0.0.99:1234")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

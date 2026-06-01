package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
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

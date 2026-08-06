package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/platform/telemetry"
	"github.com/redis/go-redis/v9"
)

// Middleware is the common HTTP middleware shape used across the project.
type Middleware func(http.Handler) http.Handler

// RequestRateLimitConfig is the decoded tenant_settings.rate_limit_config policy.
type RequestRateLimitConfig struct {
	Enabled           bool
	RequestsPerWindow int
	WindowDuration    time.Duration
	PerIP             bool
	ExemptIPs         []string
	EndpointOverrides map[string]int
}

// TenantRateLimitReader reads a tenant's stored rate_limit_config.
type TenantRateLimitReader interface {
	GetRateLimitConfig(ctx context.Context, tenantID int64) (map[string]any, error)
}

const tenantRateLimitConfigCacheTTL = 5 * time.Second

var rateLimitHitScript = redis.NewScript(`
local ttl = tonumber(ARGV[1])
local counts = {}
for i, key in ipairs(KEYS) do
	local count = redis.call("INCR", key)
	if count == 1 then
		redis.call("PEXPIRE", key, ttl)
	end
	counts[i] = count
end
return counts
`)

// OptionalMiddleware chains all supplied middleware, or returns a pass-through
// middleware when none is supplied. It lets route packages accept optional
// cross-cutting middleware without repeating nil checks in every route group.
func OptionalMiddleware(mws ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		handler := next
		for i := len(mws) - 1; i >= 0; i-- {
			if mws[i] != nil {
				handler = mws[i](handler)
			}
		}
		return handler
	}
}

// IPRateLimitMiddleware limits requests per unique client IP within a sliding window.
// Responses exceeding the limit receive 429 Too Many Requests with a Retry-After header.
// A nil rdb disables rate limiting (useful in tests and local dev without Redis).
func IPRateLimitMiddleware(rdb *redis.Client, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rdb == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := extractClientIP(r)
			key := fmt.Sprintf("rl:ip:%s:%s", ip, r.URL.Path)
			count, ok := recordRateLimitHit(context.Background(), rdb, key, window)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			if count > limit {
				retryAfter := int(window.Seconds())
				security.LogSecurityEvent(security.SecurityEvent{
					EventType: "ip_rate_limited",
					ClientIP:  ip,
					Endpoint:  r.URL.Path,
					Method:    r.Method,
					Timestamp: time.Now(),
					Details:   fmt.Sprintf("IP exceeded %d req/%v limit", limit, window),
					Severity:  "HIGH",
				})
				telemetry.RecordSecurityDenial(r.Context(), telemetry.DenialRateLimit)
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				resp.Error(w, http.StatusTooManyRequests, "rate limit exceeded — try again later")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// TenantRequestRateLimitMiddleware applies the tenant_settings.rate_limit_config
// policy after auth middleware has populated the request AuthContext.
func TenantRequestRateLimitMiddleware(rdb *redis.Client, reader TenantRateLimitReader) func(http.Handler) http.Handler {
	configCache := newTenantRateLimitConfigCache(tenantRateLimitConfigCacheTTL)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := AuthFromRequest(r)
			if rdb == nil || reader == nil || auth.Tenant == nil {
				next.ServeHTTP(w, r)
				return
			}

			cfg := configCache.get(r.Context(), reader, auth.Tenant.TenantID)
			if cfg == nil || !cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			ip := extractClientIP(r)
			if isExemptIP(ip, cfg.ExemptIPs) {
				next.ServeHTTP(w, r)
				return
			}

			limit := cfg.RequestsPerWindow
			if override, ok := cfg.EndpointOverrides[r.URL.Path]; ok {
				limit = override
			}
			if limit <= 0 {
				next.ServeHTTP(w, r)
				return
			}

			window := cfg.WindowDuration
			if window <= 0 {
				window = time.Minute
			}

			keys := tenantRateLimitKeys(auth, ip, r, cfg)
			counts, ok := recordRateLimitHits(context.Background(), rdb, keys, window)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			for _, count := range counts {
				if count > limit {
					retryAfter := int(window.Seconds())
					security.LogSecurityEvent(security.SecurityEvent{
						EventType: "tenant_rate_limited",
						ClientIP:  ip,
						Endpoint:  r.URL.Path,
						Method:    r.Method,
						Timestamp: time.Now(),
						Details: fmt.Sprintf(
							"tenant %d exceeded %d req/%v limit",
							auth.Tenant.TenantID,
							limit,
							window,
						),
						Severity: "HIGH",
					})
					telemetry.RecordSecurityDenial(r.Context(), telemetry.DenialRateLimit)
					w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
					resp.Error(w, http.StatusTooManyRequests, "rate limit exceeded — try again later")
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func recordRateLimitHit(ctx context.Context, rdb *redis.Client, key string, window time.Duration) (int, bool) {
	counts, ok := recordRateLimitHits(ctx, rdb, []string{key}, window)
	if !ok || len(counts) == 0 {
		return 0, ok
	}
	return counts[0], true
}

func recordRateLimitHits(ctx context.Context, rdb *redis.Client, keys []string, window time.Duration) ([]int, bool) {
	if len(keys) == 0 {
		return nil, true
	}
	ttlMillis := window.Milliseconds()
	if ttlMillis <= 0 {
		ttlMillis = int64(time.Minute / time.Millisecond)
	}
	rawCounts, err := rateLimitHitScript.Run(ctx, rdb, keys, ttlMillis).Result()
	if err != nil {
		// Redis failure → allow the request through (fail OPEN), deliberately, and
		// deliberately the opposite of the credential path.
		//
		// This limiter meters ordinary authenticated API traffic, where the cost of
		// a Redis outage failing closed is a total outage of the product for every
		// tenant — a self-inflicted denial of service far worse than the burst of
		// unmetered requests that failing open permits.
		//
		// The credential surface does NOT share this posture: login, MFA and the
		// other brute-forceable paths go through security.CheckRateLimit*, whose
		// limiterOutage fails CLOSED, because there the unmetered burst IS the
		// attack. If you are reconciling these two, reconcile them by keeping them
		// different — the asymmetry is the point, not an oversight.
		return nil, false
	}
	values, ok := rawCounts.([]interface{})
	if !ok {
		return nil, false
	}
	counts := make([]int, 0, len(values))
	for _, value := range values {
		count, ok := value.(int64)
		if !ok {
			return nil, false
		}
		counts = append(counts, int(count))
	}
	return counts, true
}

// defaultRequestsPerWindow mirrors tenant_settings rate_limit defaults. It is a
// fail-SAFE floor: a config that enables rate limiting but omits (or zeroes)
// requests_per_window must not degrade to "no limit". Tenant setting updates do
// whole-object replacement, so a partial PATCH like {"enabled":true} would
// otherwise parse to requests_per_window=0 and the middleware would treat
// limit<=0 as pass-through — an enabled limiter that limits nothing.
const defaultRequestsPerWindow = 100

func mapToRequestRateLimitConfig(cfg map[string]any) *RequestRateLimitConfig {
	rc := &RequestRateLimitConfig{}
	rc.Enabled = boolFromAny(cfg["enabled"])
	rc.RequestsPerWindow = intFromAny(cfg["requests_per_window"])
	rc.WindowDuration = time.Duration(intFromAny(cfg["window_duration_seconds"])) * time.Second
	rc.PerIP = boolFromAny(cfg["per_ip"])
	rc.ExemptIPs = stringSliceFromAny(cfg["exempt_ips"])
	rc.EndpointOverrides = intMapFromAny(cfg["endpoint_overrides"])
	// Fail safe: an enabled limiter with no positive window budget falls back to
	// the default rather than silently allowing unlimited requests.
	if rc.Enabled && rc.RequestsPerWindow <= 0 {
		rc.RequestsPerWindow = defaultRequestsPerWindow
	}
	return rc
}

func tenantRateLimitKeys(auth *authctx.AuthContext, ip string, r *http.Request, cfg *RequestRateLimitConfig) []string {
	keys := make([]string, 0, 2)
	base := fmt.Sprintf("rl:tenant:%d", auth.Tenant.TenantID)
	if cfg.PerIP {
		keys = append(keys, fmt.Sprintf("%s:ip:%s:%s", base, ip, r.URL.Path))
	}
	if len(keys) == 0 {
		keys = append(keys, fmt.Sprintf("%s:path:%s", base, r.URL.Path))
	}
	return keys
}

func boolFromAny(v any) bool {
	b, _ := v.(bool)
	return b
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func stringSliceFromAny(v any) []string {
	switch values := v.(type) {
	case []string:
		return values
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if s, ok := value.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func intMapFromAny(v any) map[string]int {
	values, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]int, len(values))
	for key, value := range values {
		out[key] = intFromAny(value)
	}
	return out
}

func isExemptIP(ip string, exemptIPs []string) bool {
	for _, exemptIP := range exemptIPs {
		if exemptIP == ip {
			return true
		}
	}
	return false
}

type tenantRateLimitConfigCache struct {
	ttl     time.Duration
	mu      sync.Mutex
	entries map[int64]tenantRateLimitConfigCacheEntry
}

type tenantRateLimitConfigCacheEntry struct {
	config    *RequestRateLimitConfig
	expiresAt time.Time
}

func newTenantRateLimitConfigCache(ttl time.Duration) *tenantRateLimitConfigCache {
	return &tenantRateLimitConfigCache{
		ttl:     ttl,
		entries: map[int64]tenantRateLimitConfigCacheEntry{},
	}
}

func (c *tenantRateLimitConfigCache) get(ctx context.Context, reader TenantRateLimitReader, tenantID int64) *RequestRateLimitConfig {
	now := time.Now()
	c.mu.Lock()
	entry, ok := c.entries[tenantID]
	if ok && now.Before(entry.expiresAt) {
		c.mu.Unlock()
		return entry.config
	}
	c.mu.Unlock()

	raw, err := reader.GetRateLimitConfig(ctx, tenantID)
	if err != nil {
		return nil
	}
	config := mapToRequestRateLimitConfig(raw)

	c.mu.Lock()
	c.entries[tenantID] = tenantRateLimitConfigCacheEntry{
		config:    config,
		expiresAt: now.Add(c.ttl),
	}
	c.mu.Unlock()

	return config
}

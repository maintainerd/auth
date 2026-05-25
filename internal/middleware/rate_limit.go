package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/maintainerd/auth/internal/security"
	"github.com/redis/go-redis/v9"
	resp "github.com/maintainerd/auth/internal/rest/response"
)

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
			ctx := context.Background()

			pipe := rdb.Pipeline()
			incr := pipe.Incr(ctx, key)
			pipe.Expire(ctx, key, window)
			if _, err := pipe.Exec(ctx); err != nil {
				// Redis failure → allow request through (fail open)
				next.ServeHTTP(w, r)
				return
			}

			count := int(incr.Val())
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
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				resp.Error(w, http.StatusTooManyRequests, "rate limit exceeded — try again later")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

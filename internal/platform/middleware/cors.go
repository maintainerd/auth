package middleware

import (
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

// CORSMiddleware enforces a per-environment origin allow-list.
// Origins are read from the CORS_ALLOWED_ORIGINS env var (comma-separated).
// A wildcard "*" is never combined with credentials; if no origins are
// configured the header is simply not set.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		allowed := allowedOrigins()

		if len(allowed) == 1 && allowed[0] == "*" {
			// Wildcard is only safe without credentials.
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if slices.Contains(allowed, origin) || originMatchesTenantHost(origin) {
			// Allow either a statically configured origin or a tenant-surface
			// origin that shared.ResolveTenantHost recognizes (any {tenant}.<base>
			// or a bare configured base). This is required for cross-origin prod,
			// where the browser is on {tenant}.auth.<domain> and the API is on its
			// own host; it is harmless in dev/same-origin. Arbitrary origins are
			// never allowed — only ones ResolveTenantHost accepts.
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-CSRF-Token, X-Request-ID, X-Session-ID, X-Token-Delivery")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// originMatchesTenantHost reports whether an Origin header value points at a
// tenant surface that shared.ResolveTenantHost recognizes. Only origins whose
// host is a configured base or a {tenant}.<base> subdomain are accepted, so this
// never opens CORS to arbitrary origins.
func originMatchesTenantHost(origin string) bool {
	if origin == "" {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	_, _, _, ok := shared.ResolveTenantHost(u.Host)
	return ok
}

func allowedOrigins() []string {
	raw := config.GetEnvOrDefault("CORS_ALLOWED_ORIGINS", "")
	if raw == "" {
		return nil
	}
	var out []string
	for part := range strings.SplitSeq(raw, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

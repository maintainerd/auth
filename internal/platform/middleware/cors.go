package middleware

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

// CORSOriginResolver reports whether an Origin was registered for cross-origin
// access by an active OAuth client. client.CORSOriginResolver satisfies it.
type CORSOriginResolver interface {
	IsAllowedCORSOrigin(ctx context.Context, origin string) bool
}

// registeredCORSOrigins is set once at startup. Nil is valid and simply means
// only the env var and tenant-surface hosts are consulted.
var registeredCORSOrigins CORSOriginResolver

// SetCORSOriginResolver wires the client registry into CORS decisions. Without
// it, a third-party SPA cannot call the token endpoint from its own domain no
// matter what the operator registers in the admin console — see
// client.CORSOriginResolver for why that mattered.
func SetCORSOriginResolver(resolver CORSOriginResolver) {
	registeredCORSOrigins = resolver
}

// CORSMiddleware enforces an origin allow-list drawn from three sources: the
// CORS_ALLOWED_ORIGINS env var, maintainerd's own tenant surface hosts, and the
// `cors_origin_uri` entries operators register against their OAuth clients.
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
		} else if slices.Contains(allowed, origin) || originMatchesTenantHost(origin) || originIsRegisteredClient(r, origin) {
			// Allow either a statically configured origin or a tenant-surface
			// origin that shared.ResolveTenantHost recognizes (any {tenant}.<base>
			// or a bare configured base). This is required for cross-origin prod,
			// where the browser is on {tenant}.auth.<domain> and the API is on its
			// own host; it is harmless in dev/same-origin. Third-party origins are
			// allowed only when an operator has registered them as a
			// `cors_origin_uri` on an ACTIVE client. Arbitrary origins are never
			// allowed.
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

// originIsRegisteredClient consults the client registry. Fails closed when no
// resolver is wired.
func originIsRegisteredClient(r *http.Request, origin string) bool {
	if registeredCORSOrigins == nil {
		return false
	}
	return registeredCORSOrigins.IsAllowedCORSOrigin(r.Context(), origin)
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

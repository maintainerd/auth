package middleware

import (
	"net/http"
	"slices"
	"strings"

	"github.com/maintainerd/auth/internal/config"
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
		} else if slices.Contains(allowed, origin) {
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

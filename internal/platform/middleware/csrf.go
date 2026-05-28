package middleware

import (
	"net/http"
	"time"

	resp "github.com/maintainerd/auth/internal/platform/response"
	"github.com/maintainerd/auth/internal/platform/security"
)

const (
	csrfCookieName = "__Host-csrf"
	csrfHeaderName = "X-CSRF-Token"
	csrfMaxAge     = 86400 // 24 hours
)

// safeMethods are HTTP methods that cannot trigger state changes and therefore
// do not require CSRF token validation.
var safeMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodOptions: true,
	http.MethodTrace:   true,
}

// CSRFMiddleware implements the double-submit cookie pattern for CSRF protection.
//
// For safe methods (GET/HEAD/OPTIONS/TRACE): issues a new CSRF token cookie if
// one is not already present.
// For state-changing methods (POST/PUT/PATCH/DELETE): validates that the
// X-CSRF-Token request header matches the __Host-csrf cookie value.
//
// Using SameSite=Strict cookies (which we do) already blocks most CSRF vectors;
// this middleware adds defence-in-depth for environments where the SameSite
// attribute may not be respected (older browsers, mixed same-site deployments).
func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if safeMethods[r.Method] {
			// Ensure a CSRF token cookie exists for future non-safe requests.
			ensureCSRFCookie(w, r)
			next.ServeHTTP(w, r)
			return
		}

		// Non-safe method: validate the token.
		cookie, err := r.Cookie(csrfCookieName)
		if err != nil || cookie.Value == "" {
			security.LogSecurityEvent(security.SecurityEvent{
				EventType: "csrf_missing_cookie",
				ClientIP:  extractClientIP(r),
				Endpoint:  r.URL.Path,
				Method:    r.Method,
				Timestamp: time.Now(),
				Severity:  "HIGH",
				Details:   "CSRF cookie absent",
			})
			resp.Error(w, http.StatusForbidden, "CSRF validation failed")
			return
		}

		headerToken := r.Header.Get(csrfHeaderName)
		if headerToken == "" || headerToken != cookie.Value {
			security.LogSecurityEvent(security.SecurityEvent{
				EventType: "csrf_token_mismatch",
				ClientIP:  extractClientIP(r),
				Endpoint:  r.URL.Path,
				Method:    r.Method,
				Timestamp: time.Now(),
				Severity:  "HIGH",
				Details:   "CSRF token header does not match cookie",
			})
			resp.Error(w, http.StatusForbidden, "CSRF validation failed")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func ensureCSRFCookie(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(csrfCookieName); err == nil && c.Value != "" {
		return
	}
	token, err := security.GenerateCSRFToken()
	if err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   csrfMaxAge,
		Secure:   true,
		HttpOnly: false, // Must be readable by JS so it can set the request header.
		SameSite: http.SameSiteStrictMode,
	})
}

package middleware

import (
	"net/http"
	"strings"
	"time"

	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
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

		// A request authenticated by an Authorization header is not vulnerable to
		// CSRF and must not be gated on a CSRF cookie.
		//
		// CSRF exists because a browser attaches cookies to cross-site requests
		// automatically; a Bearer/DPoP token is never ambient — an attacker's page
		// cannot make the victim's browser add it. Requiring the double-submit
		// cookie here made every state-changing call impossible for any non-browser
		// client (CLI, service, or a third-party OAuth2 app holding an access
		// token), which is the entire point of issuing them access tokens.
		//
		// Cookie-authenticated requests still go through the full check below.
		if isBearerAuthenticated(r) {
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
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- CSRF cookie intentionally readable by JS nosemgrep
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   csrfMaxAge,
		Secure:   true,
		HttpOnly: false, // Must be readable by JS so it can set the request header.
		SameSite: http.SameSiteStrictMode,
	})
}

// isBearerAuthenticated reports whether the request carries its credential in
// the Authorization header rather than a cookie.
//
// Only the header form is exempt from CSRF: a token supplied explicitly by the
// caller cannot be replayed by a cross-site page, whereas a cookie is attached
// by the browser automatically. The scheme must be one this server actually
// authenticates, so an unrelated Authorization value cannot be used to skip the
// check while the request is really authenticated by a cookie.
func isBearerAuthenticated(r *http.Request) bool {
	parts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return false
	}
	return strings.EqualFold(parts[0], "bearer") || strings.EqualFold(parts[0], "dpop")
}

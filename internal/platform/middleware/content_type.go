package middleware

import (
	"net/http"
	"strings"

	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// EnforceJSONContentType rejects state-changing requests (POST/PUT/PATCH) whose
// Content-Type is not application/json. This blocks accidental form submissions
// and provides defence against content-type confusion attacks.
//
// Excluded paths (uses application/x-www-form-urlencoded per RFC):
//   - /oauth/token, /oauth/revoke, /oauth/introspect, /oauth/par
//
// GET/HEAD/DELETE/OPTIONS are not checked since they carry no body.
func EnforceJSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			// OAuth form-encoded endpoints are exempt (RFC 6749).
			if isFormEncodedPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			ct := r.Header.Get("Content-Type")
			if ct == "" || !strings.HasPrefix(ct, "application/json") {
				resp.Error(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// isFormEncodedPath lists the endpoints whose handlers read r.ParseForm().
// Anything here MUST be exempt from the JSON content-type gate, or the request
// is rejected with 415 before the handler ever runs. The device and CIBA
// approve/deny endpoints were missing, which made both flows unusable.
func isFormEncodedPath(path string) bool {
	formPaths := []string{
		"/oauth/token",
		"/oauth/revoke",
		"/oauth/introspect",
		"/oauth/par",
		"/oauth/end_session",
		"/oauth/device",
		"/oauth/device/deny",
		"/oauth/device_authorization",
		"/oauth/ciba",
		"/oauth/ciba/approve",
		"/oauth/ciba/deny",
		"/oauth/logout/backchannel",
	}
	for _, p := range formPaths {
		if strings.HasSuffix(path, p) {
			return true
		}
	}
	return false
}

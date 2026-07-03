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

func isFormEncodedPath(path string) bool {
	formPaths := []string{
		"/oauth/token",
		"/oauth/revoke",
		"/oauth/introspect",
		"/oauth/par",
		"/oauth/end_session",
	}
	for _, p := range formPaths {
		if strings.HasSuffix(path, p) {
			return true
		}
	}
	return false
}

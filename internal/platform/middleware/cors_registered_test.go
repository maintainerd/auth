package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubCORSResolver struct {
	allow map[string]bool
	calls int
}

func (s *stubCORSResolver) IsAllowedCORSOrigin(_ context.Context, origin string) bool {
	s.calls++
	return s.allow[origin]
}

// A third-party SPA completes the redirect leg of authorization-code + PKCE via
// a top-level navigation (which CORS does not police), then exchanges the code
// with a cross-origin fetch to /oauth/token. Without an
// Access-Control-Allow-Origin for its domain the browser blocks that fetch, so
// login dies at the last step even though the code is valid. Registering the
// origin as a `cors_origin_uri` in the admin console had no effect because
// nothing read those rows.
func TestCORSMiddleware_RegisteredClientOrigins(t *testing.T) {
	t.Cleanup(func() { SetCORSOriginResolver(nil) })

	call := func(origin string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/oauth/token", nil)
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		CORSMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(w, r)
		return w
	}

	t.Run("a registered third-party origin is allowed with credentials", func(t *testing.T) {
		SetCORSOriginResolver(&stubCORSResolver{allow: map[string]bool{"https://app.thirdparty.example": true}})
		w := call("https://app.thirdparty.example")
		assert.Equal(t, "https://app.thirdparty.example", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "Origin", w.Header().Get("Vary"), "responses vary by Origin and must not be cached across them")
	})

	t.Run("an unregistered origin is denied", func(t *testing.T) {
		SetCORSOriginResolver(&stubCORSResolver{allow: map[string]bool{"https://app.thirdparty.example": true}})
		w := call("https://evil.example")
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})

	// Fail closed rather than falling back to permissive behaviour.
	t.Run("no resolver wired denies unknown origins", func(t *testing.T) {
		SetCORSOriginResolver(nil)
		w := call("https://app.thirdparty.example")
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})

	// The browser sends the preflight before the real request; if OPTIONS does
	// not carry the header the POST never happens.
	t.Run("the preflight also carries the header", func(t *testing.T) {
		SetCORSOriginResolver(&stubCORSResolver{allow: map[string]bool{"https://app.thirdparty.example": true}})
		r := httptest.NewRequest(http.MethodOptions, "/api/v1/oauth/token", nil)
		r.Header.Set("Origin", "https://app.thirdparty.example")
		w := httptest.NewRecorder()
		reached := false
		CORSMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true })).ServeHTTP(w, r)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "https://app.thirdparty.example", w.Header().Get("Access-Control-Allow-Origin"))
		assert.False(t, reached, "a preflight must short-circuit, not hit the handler")
	})

	t.Run("a request with no Origin skips the resolver entirely", func(t *testing.T) {
		stub := &stubCORSResolver{}
		SetCORSOriginResolver(stub)
		r := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		w := httptest.NewRecorder()
		CORSMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(w, r)
		assert.Zero(t, stub.calls, "same-origin traffic must not pay for a registry lookup")
	})
}

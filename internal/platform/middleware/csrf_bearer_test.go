package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// CSRF defends against a browser attaching an AMBIENT credential to a
// cross-site request. A Bearer/DPoP token is supplied explicitly by the caller
// and is never ambient, so requiring the double-submit cookie for it protected
// nothing while making every state-changing call impossible for CLI, service,
// and third-party OAuth2 clients.
func TestCSRFMiddlewareBearerExemption(t *testing.T) {
	handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	call := func(t *testing.T, mutate func(*http.Request)) int {
		t.Helper()
		r := httptest.NewRequest(http.MethodDelete, "/api/v1/account/sessions/others", nil)
		mutate(r)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w.Code
	}

	t.Run("bearer-authenticated mutation is allowed without a CSRF cookie", func(t *testing.T) {
		if code := call(t, func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer some.access.token")
		}); code != http.StatusOK {
			t.Fatalf("expected 200 for a header-authenticated request, got %d", code)
		}
	})

	t.Run("DPoP is exempt too", func(t *testing.T) {
		if code := call(t, func(r *http.Request) {
			r.Header.Set("Authorization", "DPoP some.access.token")
		}); code != http.StatusOK {
			t.Fatalf("expected 200 for a DPoP-authenticated request, got %d", code)
		}
	})

	// The exemption must key on a scheme this server actually authenticates,
	// otherwise any Authorization value becomes a way to skip CSRF on a request
	// that is really authenticated by a cookie.
	t.Run("an unrelated Authorization scheme does not skip the check", func(t *testing.T) {
		if code := call(t, func(r *http.Request) {
			r.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
		}); code != http.StatusForbidden {
			t.Fatalf("expected 403 for a non-token scheme, got %d", code)
		}
	})

	t.Run("an empty bearer value does not skip the check", func(t *testing.T) {
		if code := call(t, func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer ")
		}); code != http.StatusForbidden {
			t.Fatalf("expected 403 for an empty token, got %d", code)
		}
	})

	t.Run("a cookie-authenticated mutation is still gated", func(t *testing.T) {
		if code := call(t, func(*http.Request) {}); code != http.StatusForbidden {
			t.Fatalf("expected 403 without a CSRF cookie, got %d", code)
		}
	})
}

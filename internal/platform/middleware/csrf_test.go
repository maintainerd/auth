package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCSRFMiddleware_SafeMethodNoCookie(t *testing.T) {
	handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	// Should have set a CSRF cookie.
	cookies := w.Result().Cookies()
	assert.NotEmpty(t, cookies)
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "__Host-csrf" {
			csrfCookie = c
			break
		}
	}
	assert.NotNil(t, csrfCookie)
	assert.NotEmpty(t, csrfCookie.Value)
	assert.True(t, csrfCookie.Secure)
	assert.False(t, csrfCookie.HttpOnly)
	assert.Equal(t, http.SameSiteStrictMode, csrfCookie.SameSite)
}

func TestCSRFMiddleware_SafeMethodExistingCookie(t *testing.T) {
	handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "__Host-csrf", Value: "existing-token"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	// Must not overwrite the existing cookie — no Set-Cookie header.
	assert.Empty(t, w.Header().Get("Set-Cookie"))
}

func TestCSRFMiddleware_NonSafeNoCookie(t *testing.T) {
	handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called")
	}))

	r := httptest.NewRequest(http.MethodPost, "/submit", nil)
	r.Header.Set("X-Real-Ip", "10.0.0.1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCSRFMiddleware_NonSafeEmptyCookie(t *testing.T) {
	handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called")
	}))

	r := httptest.NewRequest(http.MethodPost, "/submit", nil)
	r.Header.Set("X-Real-Ip", "10.0.0.1")
	r.AddCookie(&http.Cookie{Name: "__Host-csrf", Value: ""})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCSRFMiddleware_NonSafeMissingHeader(t *testing.T) {
	handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called")
	}))

	r := httptest.NewRequest(http.MethodPost, "/submit", nil)
	r.Header.Set("X-Real-Ip", "10.0.0.1")
	r.AddCookie(&http.Cookie{Name: "__Host-csrf", Value: "token-123"})
	// No X-CSRF-Token header.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCSRFMiddleware_NonSafeMismatch(t *testing.T) {
	handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called")
	}))

	r := httptest.NewRequest(http.MethodPost, "/submit", nil)
	r.Header.Set("X-Real-Ip", "10.0.0.1")
	r.Header.Set("X-CSRF-Token", "wrong-token")
	r.AddCookie(&http.Cookie{Name: "__Host-csrf", Value: "token-123"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCSRFMiddleware_NonSafeSuccess(t *testing.T) {
	handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPost, "/submit", nil)
	r.Header.Set("X-CSRF-Token", "token-123")
	r.AddCookie(&http.Cookie{Name: "__Host-csrf", Value: "token-123"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRFMiddleware_AllSafeMethods(t *testing.T) {
	safe := []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace}
	for _, method := range safe {
		t.Run(method, func(t *testing.T) {
			handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			r := httptest.NewRequest(method, "/", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestCSRFMiddleware_AllNonSafeMethods(t *testing.T) {
	nonSafe := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	for _, method := range nonSafe {
		t.Run(method, func(t *testing.T) {
			handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("handler must not be called for %s", method)
			}))
			r := httptest.NewRequest(method, "/", nil)
			r.Header.Set("X-Real-Ip", "10.0.0.1")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			assert.Equal(t, http.StatusForbidden, w.Code)
		})
	}
}

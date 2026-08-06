package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// HSTS is the control that stops an attacker downgrading a user to plaintext.
// The gate used to read "ENV", which nothing in the project sets — every other
// environment check uses APP_ENV — so a correctly-configured production deploy
// silently shipped without the header.
func TestSecurityHeaders_HSTS(t *testing.T) {
	header := func(t *testing.T) string {
		t.Helper()
		w := httptest.NewRecorder()
		SecurityHeadersMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).
			ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		return w.Header().Get("Strict-Transport-Security")
	}

	t.Run("APP_ENV=production sends HSTS", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		assert.Contains(t, header(t), "max-age=31536000")
	})

	t.Run("legacy ENV=production still sends HSTS", func(t *testing.T) {
		t.Setenv("APP_ENV", "development")
		t.Setenv("ENV", "production")
		assert.Contains(t, header(t), "max-age=31536000")
	})

	t.Run("development sends no HSTS", func(t *testing.T) {
		t.Setenv("APP_ENV", "development")
		t.Setenv("ENV", "")
		assert.Empty(t, header(t))
	})
}

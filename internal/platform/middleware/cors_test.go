package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/stretchr/testify/assert"
)

func TestCORSMiddleware_NoOriginHeader(t *testing.T) {
	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Do not set t.Setenv here — let allowedOrigins read the default (empty).
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Methods"))
}

func TestCORSMiddleware_WildcardOrigin(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")

	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	// Wildcard must NOT set credentials header.
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "GET, POST, PUT, PATCH, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
}

func TestCORSMiddleware_ExplicitOriginMatch(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com,https://admin.example.com")

	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "https://app.example.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Origin", w.Header().Get("Vary"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORSMiddleware_OriginNotAllowed(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")

	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	// Origin-specific headers must NOT be set.
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	// Methods/headers still set (see cors.go lines 34-36).
	assert.Equal(t, "GET, POST, PUT, PATCH, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
}

func TestCORSMiddleware_OptionsPreflight(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")

	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodOptions, "/", nil)
	r.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "https://app.example.com", w.Header().Get("Access-Control-Allow-Origin"))
}

// setTenantBases points the configured surface bases at test hostnames so
// shared.ResolveTenantHost recognizes tenant-surface origins.
func setTenantBases(t *testing.T) {
	t.Helper()
	origI := config.AppFrontendIdentityHostname
	origC := config.AppFrontendConsoleHostname
	config.AppFrontendIdentityHostname = "auth.example.com"
	config.AppFrontendConsoleHostname = "console.auth.example.com"
	t.Cleanup(func() {
		config.AppFrontendIdentityHostname = origI
		config.AppFrontendConsoleHostname = origC
	})
}

// (e) A tenant-surface origin that shared.ResolveTenantHost recognizes is allowed
// with credentials even when it is NOT in the static CORS_ALLOWED_ORIGINS list.
func TestCORSMiddleware_TenantSubdomainOriginAllowed(t *testing.T) {
	setTenantBases(t)
	// No static origins configured → only the tenant-host rule can allow it.
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, origin := range []string{
		"https://acme.auth.example.com",         // subdomain tenant on identity surface
		"https://auth.example.com",              // bare system-tenant base
		"https://acme.console.auth.example.com", // subdomain tenant on console surface
	} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		assert.Equal(t, origin, w.Header().Get("Access-Control-Allow-Origin"), origin)
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"), origin)
		assert.Equal(t, "Origin", w.Header().Get("Vary"), origin)
	}
}

// (e) A random origin that is neither in the static list nor a recognized tenant
// host is rejected.
func TestCORSMiddleware_RandomOriginRejected(t *testing.T) {
	setTenantBases(t)
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")

	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, origin := range []string{
		"https://evil.com",             // unrelated host
		"https://evil.auth.example.io", // wrong base
		"https://a.b.auth.example.com", // multi-label prefix (not a single tenant slug)
	} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"), origin)
	}
}

func TestCORSMiddleware_AllowedOriginsEmpty(t *testing.T) {
	origins := allowedOrigins()
	assert.Nil(t, origins)
}

func TestCORSMiddleware_AllowedOriginsParsing(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://a.com, https://b.com")
	origins := allowedOrigins()
	assert.Equal(t, []string{"https://a.com", "https://b.com"}, origins)
}

func TestCORSMiddleware_AllowedOriginsWhitespace(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", " https://a.com ,  https://b.com  ")
	origins := allowedOrigins()
	assert.Equal(t, []string{"https://a.com", "https://b.com"}, origins)
}

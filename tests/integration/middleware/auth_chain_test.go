//go:build integration

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/response"
	"github.com/stretchr/testify/assert"
)

func TestIntegration_AuthChain_FullFlow(t *testing.T) {
	r := chi.NewRouter()

	r.Route("/api/v1/test", func(r chi.Router) {
		r.Use(middleware.SecurityHeadersMiddleware)
		r.Use(middleware.SecurityContextMiddleware)
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.LoggingMiddleware)
		r.Get("/protected", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response.Success(w, map[string]string{"status": "ok"}, "")
		}))
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("no token returns 401", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/test/protected", nil)
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assert.NotEmpty(t, resp.Header.Get("X-Content-Type-Options"))
		assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))
	})

	t.Run("malformed token returns 401", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/test/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.here")
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestIntegration_PermissionChain(t *testing.T) {
	allowed := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, map[string]string{"access": "granted"}, "")
	})

	r := chi.NewRouter()
	r.Route("/api/v1/admin", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(middleware.PermissionMiddleware([]string{"admin:access"}))
		r.Get("/resource", allowed)
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("without token returns 401", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/admin/resource", nil)
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestIntegration_CSRFProtection(t *testing.T) {
	r := chi.NewRouter()
	r.Use(middleware.CSRFMiddleware)
	r.Get("/api/v1/submit", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, nil, "page")
	}))
	r.Post("/api/v1/submit", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, nil, "submitted")
	}))

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("GET request issues CSRF cookie", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/submit", nil)
		req.Header.Set("X-Real-Ip", "10.0.0.1")
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		// Should have Set-Cookie with __Host-csrf
		cookies := resp.Header.Values("Set-Cookie")
		hasCSRF := false
		for _, c := range cookies {
			if len(c) > 11 && c[:11] == "__Host-csrf" {
				hasCSRF = true
			}
		}
		assert.True(t, hasCSRF, "CSRF cookie should be set")
	})

	t.Run("POST without CSRF token returns 403", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/submit", nil)
		req.Header.Set("X-Real-Ip", "10.0.0.1")
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

func TestIntegration_ContentTypeEnforcement(t *testing.T) {
	r := chi.NewRouter()
	r.Use(middleware.EnforceJSONContentType)
	r.Get("/api/v1/resource", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, nil, "ok")
	}))
	r.Post("/api/v1/resource", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, nil, "ok")
	}))

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("POST with JSON passes", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/resource", nil)
		req.Header.Set("Content-Type", "application/json")
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("POST without Content-Type returns 415", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/resource", nil)
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
	})

	t.Run("GET skips content type check", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/resource", nil)
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestIntegration_SessionValidationMiddleware(t *testing.T) {
	r := chi.NewRouter()

	r.Route("/api/v1/session-test", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Use(func(next http.Handler) http.Handler {
			return middleware.SessionValidationMiddleware(nil)(next)
		})
		r.Get("/resource", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response.Success(w, nil, "ok")
		}))
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("without X-Session-ID passes through", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/session-test/resource", nil)
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestIntegration_CORSMiddleware(t *testing.T) {
	r := chi.NewRouter()
	r.Use(middleware.CORSMiddleware)
	r.Get("/api/v1/cors-test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("OPTIONS preflight returns 204", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, ts.URL+"/api/v1/cors-test", nil)
		req.Header.Set("Origin", "https://example.com")
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("GET without Origin passes through", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/cors-test", nil)
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

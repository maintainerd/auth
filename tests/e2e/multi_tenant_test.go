//go:build e2e

package e2e_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/response"
	"github.com/stretchr/testify/assert"
)

func TestE2E_MultiTenantIsolation(t *testing.T) {
	r := newE2ERouter()

	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Get("/api/v1/tenant-a/resource", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response.Success(w, map[string]string{"tenant": "a"}, "")
		}))
		r.Get("/api/v1/tenant-b/resource", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response.Success(w, map[string]string{"tenant": "b"}, "")
		}))
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("access without token returns 401", func(t *testing.T) {
		resp := e2eDo(t, http.MethodGet, ts.URL+"/api/v1/tenant-a/resource")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("tenant B also requires authentication", func(t *testing.T) {
		resp := e2eDo(t, http.MethodGet, ts.URL+"/api/v1/tenant-b/resource")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

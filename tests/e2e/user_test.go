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

func TestE2E_UserEndpoints(t *testing.T) {
	r := newE2ERouter()

	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Get("/api/v1/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response.Success(w, []map[string]string{}, "")
		}))
		r.Get("/api/v1/profiles", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response.Success(w, map[string]string{"name": "Test User"}, "")
		}))
		r.Get("/api/v1/settings", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response.Success(w, map[string]string{"theme": "dark"}, "")
		}))
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("users requires auth", func(t *testing.T) {
		resp := e2eDo(t, http.MethodGet, ts.URL+"/api/v1/users")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("profiles requires auth", func(t *testing.T) {
		resp := e2eDo(t, http.MethodGet, ts.URL+"/api/v1/profiles")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("settings requires auth", func(t *testing.T) {
		resp := e2eDo(t, http.MethodGet, ts.URL+"/api/v1/settings")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

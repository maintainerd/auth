//go:build e2e

package e2e_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_TenantPublicDiscovery(t *testing.T) {
	defaultTenant := map[string]any{
		"name":         "system",
		"display_name": "System Tenant",
		"identifier":   "system",
		"status":       "active",
		"is_public":    true,
	}
	tenantsByIdentifier := map[string]map[string]any{
		"acme": {
			"name":         "acme",
			"display_name": "Acme",
			"identifier":   "acme",
			"status":       "active",
			"is_public":    true,
		},
	}

	r := newE2ERouter()
	r.Get("/api/v1/tenant", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, defaultTenant, "Default tenant retrieved successfully")
	}))
	r.Get("/api/v1/tenant/{identifier}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant, ok := tenantsByIdentifier[chi.URLParam(r, "identifier")]
		if !ok {
			response.Error(w, http.StatusNotFound, "tenant not found")
			return
		}
		response.Success(w, tenant, "Tenant retrieved successfully")
	}))

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("get default tenant does not require auth", func(t *testing.T) {
		resp := e2eDo(t, http.MethodGet, ts.URL+"/api/v1/tenant")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var body struct {
			Data map[string]any `json:"data"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "system", body.Data["identifier"])
	})

	t.Run("get public tenant by identifier does not require auth", func(t *testing.T) {
		resp := e2eDo(t, http.MethodGet, ts.URL+"/api/v1/tenant/acme")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var body struct {
			Data map[string]any `json:"data"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "acme", body.Data["identifier"])
	})

	t.Run("unknown public tenant returns 404", func(t *testing.T) {
		resp := e2eDo(t, http.MethodGet, ts.URL+"/api/v1/tenant/missing")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

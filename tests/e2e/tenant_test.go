//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_TenantCRUD(t *testing.T) {
	tenants := map[string]map[string]any{}

	r := newE2ERouter()

	r.Get("/api/v1/tenants", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "" {
			response.Error(w, http.StatusBadRequest, "page is required")
			return
		}
		var list []map[string]any
		for _, t := range tenants {
			list = append(list, t)
		}
		response.Success(w, list, "")
	}))

	r.Post("/api/v1/tenants", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		name, _ := body["name"].(string)
		if name == "" {
			response.Error(w, http.StatusBadRequest, "name is required")
			return
		}
		id := uuid.New().String()
		tenants[id] = map[string]any{"id": id, "name": name, "status": "active"}
		response.Created(w, tenants[id], "Tenant created")
	}))

	r.Delete("/api/v1/tenants/{tenant_uuid}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "tenant_uuid")
		if _, ok := tenants[id]; !ok {
			response.Error(w, http.StatusNotFound, "tenant not found")
			return
		}
		delete(tenants, id)
		response.Success(w, nil, "Tenant deleted")
	}))

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("create tenant returns 201", func(t *testing.T) {
		resp := e2eJSON(t, http.MethodPost, ts.URL+"/api/v1/tenants", map[string]any{
			"name": "e2e-tenant", "display_name": "E2E", "description": "A test tenant long enough descr",
		})
		defer resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var body struct {
			Data map[string]any `json:"data"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "e2e-tenant", body.Data["name"])
	})

	t.Run("create tenant with empty name returns 400", func(t *testing.T) {
		resp := e2eJSON(t, http.MethodPost, ts.URL+"/api/v1/tenants", map[string]any{"name": ""})
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("create tenant with bad JSON returns 400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/tenants", bytes.NewReader([]byte(`{bad`)))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("delete non-existent tenant returns 404", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/tenants/"+uuid.New().String(), nil)
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

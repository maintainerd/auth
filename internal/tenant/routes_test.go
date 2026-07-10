package tenant

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestTenantPublicRoute(t *testing.T) {
	handler := newTenantHandler(&mockTenantService{
		getSystemFn: func() (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{Name: "system"}, nil
		},
		getByNameFn: func(name string) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{Name: name}, nil
		},
	}, nil)
	router := chi.NewRouter()
	TenantPublicRoute(router, handler)

	t.Run("default tenant route", func(t *testing.T) {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/tenant/", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("identifier route", func(t *testing.T) {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/tenant/acme", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTenantRoute_PublicAndProtectedRoutes(t *testing.T) {
	handler := newTenantHandler(&mockTenantService{
		getSystemFn: func() (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{Name: "system"}, nil
		},
	}, nil)
	router := chi.NewRouter()
	TenantRoute(router, handler, nil, nil)

	t.Run("public default tenant route is reachable", func(t *testing.T) {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/tenant/", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("protected tenants route requires auth", func(t *testing.T) {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/tenants/", nil))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestTenantSettingRoute_ProtectedRoutesRequireAuth(t *testing.T) {
	router := chi.NewRouter()
	TenantSettingRoute(router, NewTenantSettingHandler(&mockTenantSettingService{}), nil, nil, nil)

	for _, path := range []string{
		"/tenant-settings/rate-limit",
		"/tenant-settings/audit",
		"/tenant-settings/maintenance",
	} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

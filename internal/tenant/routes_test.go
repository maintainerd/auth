package tenant

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// routeHasStepUp reports whether the middleware chain chi assembled for
// method+route contains middleware.RequireStepUp. Comparing function pointers is
// the only way to assert the gate from outside: the chain is opaque once built,
// and driving the route end to end would need a signed acr=2 token.
func routeHasStepUp(t *testing.T, router chi.Routes, method, route string) bool {
	t.Helper()
	want := reflect.ValueOf(middleware.RequireStepUp).Pointer()
	found := false
	matched := false
	require.NoError(t, chi.Walk(router, func(m, r string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		if m != method || r != route {
			return nil
		}
		matched = true
		for _, mw := range mws {
			if reflect.ValueOf(mw).Pointer() == want {
				found = true
			}
		}
		return nil
	}))
	require.True(t, matched, "route %s %s is not registered", method, route)
	return found
}

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

// Tenant member MUTATIONS used to carry only PermissionMiddleware("tenant:update")
// while the plain tenant update/delete on the same router were step-up gated —
// so the cheapest route to tenant-wide super-admin (add/promote an owner, which
// implicitly grants shared.RoleSuperAdmin) was the one demanding the least proof
// of presence. Every mutation is now gated; the read is deliberately not.
func TestTenantRoute_MemberMutationsRequireStepUp(t *testing.T) {
	handler := newTenantHandler(&mockTenantService{}, nil)
	router := chi.NewRouter()
	TenantRoute(router, handler, nil, nil)

	gated := []struct {
		method string
		route  string
	}{
		{http.MethodPost, "/tenants/{tenant_uuid}/members/"},
		{http.MethodPatch, "/tenants/{tenant_uuid}/members/{tenant_member_uuid}/role"},
		{http.MethodDelete, "/tenants/{tenant_uuid}/members/{tenant_member_uuid}"},
		// Regression guard for the routes that already had the gate.
		{http.MethodPut, "/tenants/{tenant_uuid}"},
		{http.MethodPut, "/tenants/{tenant_uuid}/status"},
		{http.MethodDelete, "/tenants/{tenant_uuid}"},
	}
	for _, tc := range gated {
		t.Run(tc.method+" "+tc.route, func(t *testing.T) {
			assert.True(t, routeHasStepUp(t, router, tc.method, tc.route),
				"expected RequireStepUp on %s %s", tc.method, tc.route)
		})
	}

	ungated := []struct {
		method string
		route  string
	}{
		{http.MethodGet, "/tenants/{tenant_uuid}/members/"},
		{http.MethodGet, "/tenants/"},
		{http.MethodGet, "/tenants/{tenant_uuid}"},
	}
	for _, tc := range ungated {
		t.Run("read "+tc.method+" "+tc.route, func(t *testing.T) {
			assert.False(t, routeHasStepUp(t, router, tc.method, tc.route),
				"read-only %s %s must not require step-up", tc.method, tc.route)
		})
	}
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

package iam

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestIAMRoutesRegister(t *testing.T) {
	tests := []struct {
		name     string
		register func(chi.Router)
		routes   []struct {
			method string
			path   string
		}
	}{
		{
			name: "apis",
			register: func(r chi.Router) {
				APIRoute(r, NewAPIHandler(&mockAPIService{}), nil, nil)
			},
			routes: []struct {
				method string
				path   string
			}{
				{http.MethodGet, "/apis/"},
				{http.MethodGet, "/apis/{api_uuid}"},
				{http.MethodPost, "/apis/"},
				{http.MethodPut, "/apis/{api_uuid}"},
				{http.MethodPut, "/apis/{api_uuid}/status"},
				{http.MethodDelete, "/apis/{api_uuid}"},
			},
		},
		{
			name: "permissions",
			register: func(r chi.Router) {
				PermissionRoute(r, NewPermissionHandler(&mockPermissionService{}), nil, nil)
			},
			routes: []struct {
				method string
				path   string
			}{
				{http.MethodGet, "/permissions/"},
				{http.MethodGet, "/permissions/{permission_uuid}"},
				{http.MethodPost, "/permissions/"},
				{http.MethodPut, "/permissions/{permission_uuid}"},
				{http.MethodPut, "/permissions/{permission_uuid}/status"},
				{http.MethodDelete, "/permissions/{permission_uuid}"},
			},
		},
		{
			name: "policies",
			register: func(r chi.Router) {
				PolicyRoute(r, NewPolicyHandler(&mockPolicyService{}), nil, nil)
			},
			routes: []struct {
				method string
				path   string
			}{
				{http.MethodGet, "/policies/"},
				{http.MethodGet, "/policies/{policy_uuid}"},
				{http.MethodGet, "/policies/{policy_uuid}/services"},
				{http.MethodPost, "/policies/"},
				{http.MethodPut, "/policies/{policy_uuid}"},
				{http.MethodPut, "/policies/{policy_uuid}/status"},
				{http.MethodDelete, "/policies/{policy_uuid}"},
			},
		},
		{
			name: "roles",
			register: func(r chi.Router) {
				RoleRoute(r, NewRoleHandler(&mockRoleService{}), nil, nil)
			},
			routes: []struct {
				method string
				path   string
			}{
				{http.MethodGet, "/roles/"},
				{http.MethodGet, "/roles/{role_uuid}"},
				{http.MethodPost, "/roles/"},
				{http.MethodPut, "/roles/{role_uuid}"},
				{http.MethodPut, "/roles/{role_uuid}/status"},
				{http.MethodDelete, "/roles/{role_uuid}"},
				{http.MethodGet, "/roles/{role_uuid}/permissions"},
				{http.MethodPost, "/roles/{role_uuid}/permissions"},
				{http.MethodDelete, "/roles/{role_uuid}/permissions/{permission_uuid}"},
			},
		},
		{
			name: "services",
			register: func(r chi.Router) {
				ServiceRoute(r, NewServiceHandler(&mockServiceService{}), nil, nil)
			},
			routes: []struct {
				method string
				path   string
			}{
				{http.MethodGet, "/services/"},
				{http.MethodGet, "/services/{service_uuid}"},
				{http.MethodPost, "/services/"},
				{http.MethodPut, "/services/{service_uuid}"},
				{http.MethodPut, "/services/{service_uuid}/status"},
				{http.MethodDelete, "/services/{service_uuid}"},
				{http.MethodPost, "/services/{service_uuid}/policies/{policy_uuid}"},
				{http.MethodDelete, "/services/{service_uuid}/policies/{policy_uuid}"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := chi.NewRouter()
			tc.register(r)

			for _, route := range tc.routes {
				t.Run(route.method+" "+route.path, func(t *testing.T) {
					match := chi.NewRouteContext()
					ok := r.Match(match, route.method, route.path)
					assert.True(t, ok)
				})
			}
		})
	}
}

package idp

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestFederationPublicRoute(t *testing.T) {
	r := chi.NewRouter()
	FederationPublicRoute(r, NewFederationHandler(&mockFederationService{}))

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/federation/token"},
		{http.MethodPost, "/federation/oauth2/callback"},
		{http.MethodGet, "/federation/hrd"},
	}
	for _, tc := range paths {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			match := chi.NewRouteContext()
			assert.True(t, r.Match(match, tc.method, tc.path))
		})
	}
}

func TestFederationIdentityRoute(t *testing.T) {
	r := chi.NewRouter()
	FederationIdentityRoute(r, NewFederationHandler(&mockFederationService{}), nil, nil)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/account/identities/"},
		{http.MethodPost, "/account/identities/link"},
		{http.MethodDelete, "/account/identities/abc"},
	}
	for _, tc := range paths {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			match := chi.NewRouteContext()
			assert.True(t, r.Match(match, tc.method, tc.path))
		})
	}
}

func TestIdentityProviderRoute(t *testing.T) {
	r := chi.NewRouter()
	IdentityProviderRoute(r, NewIdentityProviderHandler(&mockIdentityProviderService{}), nil, nil, nil)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/identity_providers/"},
		{http.MethodGet, "/identity_providers/abc"},
		{http.MethodPost, "/identity_providers/"},
		{http.MethodPut, "/identity_providers/abc"},
		{http.MethodPut, "/identity_providers/abc/status"},
		{http.MethodDelete, "/identity_providers/abc"},
	}
	for _, tc := range paths {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			match := chi.NewRouteContext()
			assert.True(t, r.Match(match, tc.method, tc.path))
		})
	}
}

func TestRegistrationFlowRouteRegistration(t *testing.T) {
	r := chi.NewRouter()
	RegistrationFlowRoute(r, NewRegistrationFlowHandler(&mockRegistrationFlowService{}), nil, nil)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/registration_flows/"},
		{http.MethodGet, "/registration_flows/abc"},
		{http.MethodPost, "/registration_flows/"},
		{http.MethodPut, "/registration_flows/abc"},
		{http.MethodPatch, "/registration_flows/abc/status"},
		{http.MethodDelete, "/registration_flows/abc"},
		{http.MethodPost, "/registration_flows/abc/roles/"},
		{http.MethodGet, "/registration_flows/abc/roles/"},
		{http.MethodDelete, "/registration_flows/abc/roles/role"},
	}
	for _, tc := range paths {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			match := chi.NewRouteContext()
			assert.True(t, r.Match(match, tc.method, tc.path))
		})
	}
}

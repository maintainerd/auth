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
	IdentityProviderRoute(r, NewIdentityProviderHandler(&mockIdentityProviderService{}), nil, nil)

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

func TestSignupFlowRouteRegistration(t *testing.T) {
	r := chi.NewRouter()
	SignupFlowRoute(r, NewSignupFlowHandler(&mockSignupFlowService{}), nil, nil)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/signup_flows/"},
		{http.MethodGet, "/signup_flows/abc"},
		{http.MethodPost, "/signup_flows/"},
		{http.MethodPut, "/signup_flows/abc"},
		{http.MethodPatch, "/signup_flows/abc/status"},
		{http.MethodDelete, "/signup_flows/abc"},
		{http.MethodPost, "/signup_flows/abc/roles/"},
		{http.MethodGet, "/signup_flows/abc/roles/"},
		{http.MethodDelete, "/signup_flows/abc/roles/role"},
	}
	for _, tc := range paths {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			match := chi.NewRouteContext()
			assert.True(t, r.Match(match, tc.method, tc.path))
		})
	}
}

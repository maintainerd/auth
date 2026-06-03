package oauth

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/dpop"
	"github.com/stretchr/testify/assert"
)

func TestOAuthRoutesMountEndpoints(t *testing.T) {
	public := chi.NewRouter()
	OAuthPublicRoute(
		public,
		NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{}),
		NewOAuthTokenHandler(&mockOAuthTokenService{}, &dpop.NonceManager{}, nil),
		NewOAuthTokenExchangeHandler(&mockOAuthTokenExchangeService{}),
		NewOAuthConsentHandler(&mockOAuthConsentService{}),
		NewOAuthUserInfoHandler(),
		NewOAuthPARHandler(&mockOAuthPARService{}),
		NewOAuthDeviceHandler(&mockOAuthDeviceService{}),
		NewOAuthSessionHandler(&mockOAuthSessionService{}),
		NewOAuthCIBAHandler(&mockOAuthCIBAService{}),
		NewOAuthRegisterHandler(&mockOAuthRegisterService{}),
		nil,
		nil,
		nil,
	)

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/oauth/authorize"},
		{http.MethodGet, "/oauth/consent/" + testResourceUUID.String()},
		{http.MethodPost, "/oauth/consent"},
		{http.MethodPost, "/oauth/token"},
		{http.MethodPost, "/oauth/revoke"},
		{http.MethodGet, "/oauth/userinfo"},
		{http.MethodGet, "/oauth/consent/grants"},
		{http.MethodDelete, "/oauth/consent/grants/" + testResourceUUID.String()},
		{http.MethodPost, "/oauth/par"},
		{http.MethodPost, "/oauth/device_authorization"},
		{http.MethodPost, "/oauth/device"},
		{http.MethodPost, "/oauth/device/deny"},
		{http.MethodPost, "/oauth/ciba"},
		{http.MethodPost, "/oauth/ciba/approve"},
		{http.MethodPost, "/oauth/ciba/deny"},
		{http.MethodPost, "/oauth/register"},
		{http.MethodGet, "/oauth/end_session"},
		{http.MethodPost, "/oauth/end_session"},
		{http.MethodPost, "/oauth/logout/backchannel"},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			match := chi.NewRouteContext()
			assert.True(t, public.Match(match, tt.method, tt.path))
		})
	}

	discovery := chi.NewRouter()
	OAuthDiscoveryRoute(discovery, NewOAuthDiscoveryHandler())
	for _, tt := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/.well-known/openid-configuration"},
		{http.MethodGet, "/.well-known/oauth-authorization-server"},
		{http.MethodGet, "/.well-known/jwks.json"},
	} {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			match := chi.NewRouteContext()
			assert.True(t, discovery.Match(match, tt.method, tt.path))
		})
	}

	internal := chi.NewRouter()
	OAuthInternalRoute(internal, NewOAuthTokenHandler(&mockOAuthTokenService{}, nil, nil), nil, nil)
	match := chi.NewRouteContext()
	assert.True(t, internal.Match(match, http.MethodPost, "/oauth/introspect"))
}

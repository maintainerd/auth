package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/dpop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthRoutesMountEndpoints(t *testing.T) {
	public := chi.NewRouter()
	OAuthPublicRoute(
		public,
		NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{}),
		NewOAuthConnectionsHandler(nil),
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
		{http.MethodGet, "/oauth/connections"},
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
		// POST /oauth/register (Dynamic Client Registration) is deliberately NOT
		// mounted — see the rationale in routes.go. It is asserted absent below.
		{http.MethodGet, "/oauth/callback/" + testResourceUUID.String()},
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

	// Dynamic Client Registration must stay off the PUBLIC plane. RFC 7591 §3's
	// initial access token would here be any access token from any third-party
	// client, so client creation would ride on whatever authority that token
	// happens to carry. It lives on the control plane instead — see
	// TestOAuthInternalRouteWithRegistration_MountsKeyAndRegistrationSurfaces.
	t.Run("POST /oauth/register is not mounted on the public plane", func(t *testing.T) {
		match := chi.NewRouteContext()
		assert.False(t, public.Match(match, http.MethodPost, "/oauth/register"),
			"DCR is control-plane only; a public mount makes client creation reachable with any third-party token")
	})

	discovery := chi.NewRouter()
	OAuthDiscoveryRoute(discovery, NewOAuthDiscoveryHandler(nil))
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
	OAuthInternalRouteWithRegistration(
		internal,
		NewOAuthTokenHandler(&mockOAuthTokenService{}, nil, nil),
		NewOAuthSigningKeyHandler(NewKeyRotationService(&fakeSigningKeyRepo{})),
		NewOAuthRegisterHandler(&mockOAuthRegisterService{}),
		nil,
		nil,
	)
	match := chi.NewRouteContext()
	assert.True(t, internal.Match(match, http.MethodPost, "/oauth/introspect"))
}

// The signing-key lifecycle and RFC 7591/7592 registration shipped as handler +
// service + permission + unit tests while no router mounted either of them, so
// "the code exists" read as "the endpoint is reachable". These assert the mount.
func TestOAuthInternalRouteWithRegistration_MountsKeyAndRegistrationSurfaces(t *testing.T) {
	mounted := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/oauth/introspect"},
		{http.MethodGet, "/oauth/signing-keys"},
		{http.MethodPost, "/oauth/signing-keys/rotate"},
		{http.MethodPost, "/oauth/signing-keys/kid-1/retire"},
		{http.MethodPost, "/oauth/signing-keys/kid-1/compromise"},
		{http.MethodPost, "/oauth/register"},
		{http.MethodGet, "/oauth/register/client-abc"},
	}

	internal := chi.NewRouter()
	OAuthInternalRouteWithRegistration(
		internal,
		NewOAuthTokenHandler(&mockOAuthTokenService{}, nil, nil),
		NewOAuthSigningKeyHandler(NewKeyRotationService(&fakeSigningKeyRepo{})),
		NewOAuthRegisterHandler(&mockOAuthRegisterService{}),
		nil,
		nil,
	)

	for _, tt := range mounted {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			match := chi.NewRouteContext()
			assert.True(t, internal.Match(match, tt.method, tt.path))

			// Reachable is not the same as open: every one of them sits behind JWT
			// auth, so an unauthenticated caller is refused before a handler runs.
			r := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			internal.ServeHTTP(w, r)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

// INVERTED. This test used to be TestOAuthInternalRoute_MountsNeitherKeyNor
// RegistrationSurface and asserted the OPPOSITE: that OAuthInternalRoute — the
// wrapper that passed nil for both the signing-key and the registration handler
// — mounted neither surface, "pinning the gap so it is visible in the suite".
//
// Pinning a gap is not a guard. It made the broken wiring the tested behaviour:
// the composition root could keep calling (or revert to) the nil-handler wrapper
// and the suite stayed green, which is precisely how the key lifecycle and RFC
// 7591/7592 shipped reachable on no port at all. The wrappers are deleted, so
// that revert is now a compile error, and a nil handler is a boot panic instead
// of a silently absent control-plane surface.
func TestOAuthInternalRouteWithRegistration_NilHandlerRefusesToMount(t *testing.T) {
	tokenHandler := NewOAuthTokenHandler(&mockOAuthTokenService{}, nil, nil)
	keyHandler := NewOAuthSigningKeyHandler(NewKeyRotationService(&fakeSigningKeyRepo{}))
	registerHandler := NewOAuthRegisterHandler(&mockOAuthRegisterService{})

	for _, tt := range []struct {
		name     string
		token    *OAuthTokenHandler
		key      *OAuthSigningKeyHandler
		register *OAuthRegisterHandler
		wantIn   string
	}{
		{"nil signing-key handler", tokenHandler, nil, registerHandler, "signing-key handler"},
		{"nil registration handler", tokenHandler, keyHandler, nil, "client-registration handler"},
		{"nil token handler", nil, keyHandler, registerHandler, "token handler"},
		{"all nil", nil, nil, nil, "signing-key handler"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.PanicsWithValue(t,
				"oauth: internal OAuth plane mounted without "+
					strings.Join(missingInternalOAuthHandlers(tt.token, tt.key, tt.register), ", ")+
					"; the composition root must build and pass every internal OAuth handler",
				func() {
					OAuthInternalRouteWithRegistration(chi.NewRouter(), tt.token, tt.key, tt.register, nil, nil)
				},
				"a half-wired control plane must refuse to boot, not mount a subset of itself silently")
			assert.Contains(t, strings.Join(missingInternalOAuthHandlers(tt.token, tt.key, tt.register), ", "), tt.wantIn)
		})
	}
}

// The public discovery documents must not advertise an endpoint that only
// exists on the VPN-only control plane: a conformant RP that reads
// registration_endpoint out of the public metadata would POST to a URL that
// 404s, and publishing it at all frames client creation as a public,
// token-gated operation instead of an operator one.
func TestPublicDiscoveryOmitsControlPlaneOnlyEndpoints(t *testing.T) {
	h := NewOAuthDiscoveryHandler(nil)

	for _, tt := range []struct {
		name    string
		serve   func(http.ResponseWriter, *http.Request)
		path    string
		omitted []string
	}{
		{
			"openid-configuration",
			h.Discovery,
			"/.well-known/openid-configuration",
			[]string{"registration_endpoint", "introspection_endpoint"},
		},
		{
			"oauth-authorization-server",
			h.AuthorizationServerMetadata,
			"/.well-known/oauth-authorization-server",
			[]string{"registration_endpoint", "introspection_endpoint"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.serve(w, httptest.NewRequest(http.MethodGet, tt.path, nil))
			require.Equal(t, http.StatusOK, w.Code)

			var doc map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &doc))
			for _, key := range tt.omitted {
				assert.NotContains(t, doc, key,
					"%s is mounted on the control plane only; advertising it on the public host points RPs at a 404", key)
			}
			// Sanity: the document is a real one, so NotContains above is not
			// passing on an empty body.
			assert.NotEmpty(t, doc["token_endpoint"])
		})
	}
}

func TestOAuthRoutesMountEndpointsWithRateLimit(t *testing.T) {
	public := chi.NewRouter()
	OAuthPublicRoute(
		public,
		NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{}),
		NewOAuthConnectionsHandler(nil),
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
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		},
	)

	match := chi.NewRouteContext()
	assert.True(t, public.Match(match, http.MethodPost, "/oauth/token"))
}

func TestOAuthPublicRoute_TokenGrantTypeDispatch(t *testing.T) {
	t.Run("token-exchange", func(t *testing.T) {
		var called bool
		tokenExSvc := &mockOAuthTokenExchangeService{}
		tokenExSvc.exchangeFn = func(_ context.Context, _ OAuthTokenExchangeRequestDTO, _ OAuthClientCredentials) (*OAuthTokenExchangeResponseDTO, *apperror.OAuthError) {
			called = true
			return &OAuthTokenExchangeResponseDTO{}, nil
		}

		router := chi.NewRouter()
		OAuthPublicRoute(
			router,
			NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{}),
			NewOAuthConnectionsHandler(nil),
			NewOAuthTokenHandler(&mockOAuthTokenService{}, nil, nil),
			NewOAuthTokenExchangeHandler(tokenExSvc),
			NewOAuthConsentHandler(&mockOAuthConsentService{}),
			NewOAuthUserInfoHandler(),
			NewOAuthPARHandler(&mockOAuthPARService{}),
			NewOAuthDeviceHandler(&mockOAuthDeviceService{}),
			NewOAuthSessionHandler(&mockOAuthSessionService{}),
			NewOAuthCIBAHandler(&mockOAuthCIBAService{}),
			NewOAuthRegisterHandler(&mockOAuthRegisterService{}),
			nil, nil, nil,
		)

		body := url.Values{
			"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
			"subject_token":      {"token-abc"},
			"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
			"client_id":          {"test"},
		}
		r := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		assert.True(t, called)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("device_code", func(t *testing.T) {
		var called bool
		deviceSvc := &mockOAuthDeviceService{}
		deviceSvc.exchangeTokenFn = func(_ context.Context, _ OAuthDeviceTokenRequestDTO, _ OAuthClientCredentials) (*OAuthTokenResponseDTO, *apperror.OAuthError) {
			called = true
			return &OAuthTokenResponseDTO{}, nil
		}

		router := chi.NewRouter()
		OAuthPublicRoute(
			router,
			NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{}),
			NewOAuthConnectionsHandler(nil),
			NewOAuthTokenHandler(&mockOAuthTokenService{}, nil, nil),
			NewOAuthTokenExchangeHandler(&mockOAuthTokenExchangeService{}),
			NewOAuthConsentHandler(&mockOAuthConsentService{}),
			NewOAuthUserInfoHandler(),
			NewOAuthPARHandler(&mockOAuthPARService{}),
			NewOAuthDeviceHandler(deviceSvc),
			NewOAuthSessionHandler(&mockOAuthSessionService{}),
			NewOAuthCIBAHandler(&mockOAuthCIBAService{}),
			NewOAuthRegisterHandler(&mockOAuthRegisterService{}),
			nil, nil, nil,
		)

		body := url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code": {"dc-abc"},
			"client_id":   {"test"},
		}
		r := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		assert.True(t, called)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("CIBA", func(t *testing.T) {
		var called bool
		cibaSvc := &mockOAuthCIBAService{}
		cibaSvc.exchangeTokenFn = func(_ context.Context, _ OAuthCIBATokenRequestDTO, _ OAuthClientCredentials) (*OAuthTokenResponseDTO, *apperror.OAuthError) {
			called = true
			return &OAuthTokenResponseDTO{}, nil
		}

		router := chi.NewRouter()
		OAuthPublicRoute(
			router,
			NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{}),
			NewOAuthConnectionsHandler(nil),
			NewOAuthTokenHandler(&mockOAuthTokenService{}, nil, nil),
			NewOAuthTokenExchangeHandler(&mockOAuthTokenExchangeService{}),
			NewOAuthConsentHandler(&mockOAuthConsentService{}),
			NewOAuthUserInfoHandler(),
			NewOAuthPARHandler(&mockOAuthPARService{}),
			NewOAuthDeviceHandler(&mockOAuthDeviceService{}),
			NewOAuthSessionHandler(&mockOAuthSessionService{}),
			NewOAuthCIBAHandler(cibaSvc),
			NewOAuthRegisterHandler(&mockOAuthRegisterService{}),
			nil, nil, nil,
		)

		body := url.Values{
			"grant_type":  {"urn:openid:params:grant-type:ciba"},
			"auth_req_id": {"arid-abc"},
			"client_id":   {"test"},
		}
		r := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		assert.True(t, called)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("authorization_code default", func(t *testing.T) {
		var called bool
		tokenSvc := &mockOAuthTokenService{}
		tokenSvc.exchangeFn = func(_ context.Context, _ OAuthTokenRequestDTO, _ OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
			called = true
			return &OAuthTokenResult{AccessToken: "at", TokenType: "Bearer", ExpiresIn: 900}, nil
		}

		router := chi.NewRouter()
		OAuthPublicRoute(
			router,
			NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{}),
			NewOAuthConnectionsHandler(nil),
			NewOAuthTokenHandler(tokenSvc, nil, nil),
			NewOAuthTokenExchangeHandler(&mockOAuthTokenExchangeService{}),
			NewOAuthConsentHandler(&mockOAuthConsentService{}),
			NewOAuthUserInfoHandler(),
			NewOAuthPARHandler(&mockOAuthPARService{}),
			NewOAuthDeviceHandler(&mockOAuthDeviceService{}),
			NewOAuthSessionHandler(&mockOAuthSessionService{}),
			NewOAuthCIBAHandler(&mockOAuthCIBAService{}),
			NewOAuthRegisterHandler(&mockOAuthRegisterService{}),
			nil, nil, nil,
		)

		body := url.Values{
			"grant_type": {"authorization_code"},
			"code":       {"some-code"},
		}
		r := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		assert.True(t, called)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestOAuthPublicRoute_TokenRateLimitWrapsEndpoint(t *testing.T) {
	var rateLimiterCalled bool
	rateLimitMW := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rateLimiterCalled = true
			w.Header().Set("X-RateLimit-Wrap", "true")
			next.ServeHTTP(w, r)
		})
	}

	var tokenCalled bool
	tokenSvc := &mockOAuthTokenService{}
	tokenSvc.exchangeFn = func(_ context.Context, _ OAuthTokenRequestDTO, _ OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
		tokenCalled = true
		return &OAuthTokenResult{AccessToken: "at", TokenType: "Bearer", ExpiresIn: 900}, nil
	}

	router := chi.NewRouter()
	OAuthPublicRoute(
		router,
		NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{}),
		NewOAuthConnectionsHandler(nil),
		NewOAuthTokenHandler(tokenSvc, nil, nil),
		NewOAuthTokenExchangeHandler(&mockOAuthTokenExchangeService{}),
		NewOAuthConsentHandler(&mockOAuthConsentService{}),
		NewOAuthUserInfoHandler(),
		NewOAuthPARHandler(&mockOAuthPARService{}),
		NewOAuthDeviceHandler(&mockOAuthDeviceService{}),
		NewOAuthSessionHandler(&mockOAuthSessionService{}),
		NewOAuthCIBAHandler(&mockOAuthCIBAService{}),
		NewOAuthRegisterHandler(&mockOAuthRegisterService{}),
		nil, nil,
		rateLimitMW,
	)

	body := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {"some-code"},
	}
	r := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	assert.True(t, rateLimiterCalled, "rate limiter should wrap the token endpoint")
	assert.True(t, tokenCalled, "token handler should be called through the rate limiter")
	assert.Equal(t, "true", w.Header().Get("X-RateLimit-Wrap"))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOAuthPublicRoute_TokenParseFormError(t *testing.T) {
	router := chi.NewRouter()
	OAuthPublicRoute(
		router,
		NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{}),
		NewOAuthConnectionsHandler(nil),
		NewOAuthTokenHandler(&mockOAuthTokenService{}, nil, nil),
		NewOAuthTokenExchangeHandler(&mockOAuthTokenExchangeService{}),
		NewOAuthConsentHandler(&mockOAuthConsentService{}),
		NewOAuthUserInfoHandler(),
		NewOAuthPARHandler(&mockOAuthPARService{}),
		NewOAuthDeviceHandler(&mockOAuthDeviceService{}),
		NewOAuthSessionHandler(&mockOAuthSessionService{}),
		NewOAuthCIBAHandler(&mockOAuthCIBAService{}),
		NewOAuthRegisterHandler(&mockOAuthRegisterService{}),
		nil, nil, nil,
	)

	r := httptest.NewRequest(http.MethodPost, "/oauth/token", errReader{})
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

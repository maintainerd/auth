package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/dpop"
	"github.com/stretchr/testify/assert"
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
		{http.MethodPost, "/oauth/register"},
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

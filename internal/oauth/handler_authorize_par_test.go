package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubPARService struct {
	consumed  string
	consumeFn func(context.Context, string) (*OAuthAuthorizeRequestDTO, *apperror.OAuthError)
}

func (s *stubPARService) Push(context.Context, OAuthPARRequestDTO, OAuthClientCredentials) (*OAuthPARResponseDTO, *apperror.OAuthError) {
	return nil, nil
}

func (s *stubPARService) ConsumeRequestURI(ctx context.Context, requestURI string) (*OAuthAuthorizeRequestDTO, *apperror.OAuthError) {
	s.consumed = requestURI
	if s.consumeFn != nil {
		return s.consumeFn(ctx, requestURI)
	}
	return &OAuthAuthorizeRequestDTO{
		ResponseType: "code",
		ClientID:     "my-client",
		RedirectURI:  "https://app.example.com/cb",
		Scope:        "openid",
		State:        "pushed-state",
	}, nil
}

// POST /oauth/par minted a request_uri and RFC 8414 metadata advertised the
// endpoint, but /oauth/authorize never read the parameter and
// ConsumeRequestURI had no caller anywhere — PAR was a dead end.
func TestOAuthAuthorizeHandler_RequestURI(t *testing.T) {
	t.Run("the pushed request is consumed and replaces the query string", func(t *testing.T) {
		par := &stubPARService{}
		authorizeSvc := &mockOAuthAuthorizeService{
			prepareAuthorizeFn: func(_ context.Context, req OAuthAuthorizeRequestDTO) *apperror.OAuthError {
				// The pushed copy wins: nothing the caller put in the query string
				// survives, or an attacker holding the request_uri could append
				// parameters the client never pushed.
				assert.Equal(t, "pushed-state", req.State)
				assert.Equal(t, "https://app.example.com/cb", req.RedirectURI)
				return nil
			},
		}
		h := NewOAuthAuthorizeHandler(authorizeSvc)
		h.AttachPARService(par)

		r := httptest.NewRequest(http.MethodGet,
			"/oauth/authorize?client_id=my-client&request_uri=urn:ietf:params:oauth:request-uri:abc&state=attacker-state", nil)
		w := httptest.NewRecorder()
		h.Authorize(w, r)

		assert.Equal(t, "urn:ietf:params:oauth:request-uri:abc", par.consumed)
	})

	t.Run("a client_id that disagrees with the pushed request is refused", func(t *testing.T) {
		h := NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{})
		h.AttachPARService(&stubPARService{})

		r := httptest.NewRequest(http.MethodGet,
			"/oauth/authorize?client_id=other-client&request_uri=urn:ietf:params:oauth:request-uri:abc", nil)
		w := httptest.NewRecorder()
		h.Authorize(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// Fails CLOSED: honouring the query parameters while dropping the pushed
	// request would authorize something the client never pushed.
	t.Run("request_uri with no PAR service is refused, not ignored", func(t *testing.T) {
		h := NewOAuthAuthorizeHandler(&mockOAuthAuthorizeService{})

		r := httptest.NewRequest(http.MethodGet,
			"/oauth/authorize?response_type=code&client_id=my-client&redirect_uri=https://app.example.com/cb&state=s&request_uri=urn:ietf:params:oauth:request-uri:abc", nil)
		w := httptest.NewRecorder()
		h.Authorize(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// OIDC Core §6: silently ignoring a signed request object means honouring the
// unsigned query parameters the RP intended it to override.
func TestOAuthAuthorizeRequestDTO_RejectsJAR(t *testing.T) {
	req := OAuthAuthorizeRequestDTO{
		ResponseType: "code",
		ClientID:     "my-client",
		RedirectURI:  "https://app.example.com/cb",
		Request:      "eyJhbGciOiJSUzI1NiJ9.e30.sig",
	}
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request_not_supported")
}

func TestOAuthAuthorizeRequestDTO_OIDCParameters(t *testing.T) {
	base := func() OAuthAuthorizeRequestDTO {
		return OAuthAuthorizeRequestDTO{
			ResponseType: "code",
			ClientID:     "my-client",
			RedirectURI:  "https://app.example.com/cb",
		}
	}

	t.Run("max_age is parsed, not merely length-checked", func(t *testing.T) {
		req := base()
		req.MaxAge = "300"
		require.NoError(t, req.Validate())
		assert.True(t, req.MaxAgeSet)
		assert.Equal(t, int64(300), req.MaxAgeSeconds)
	})

	t.Run("max_age=0 is distinguishable from absent", func(t *testing.T) {
		req := base()
		req.MaxAge = "0"
		require.NoError(t, req.Validate())
		assert.True(t, req.MaxAgeSet)
		assert.Equal(t, int64(0), req.MaxAgeSeconds)

		absent := base()
		require.NoError(t, absent.Validate())
		assert.False(t, absent.MaxAgeSet)
	})

	// An unparseable max_age must not silently become "no limit".
	t.Run("a non-numeric or negative max_age is rejected", func(t *testing.T) {
		for _, bad := range []string{"soon", "-1"} {
			req := base()
			req.MaxAge = bad
			require.Error(t, req.Validate(), bad)
		}
	})

	t.Run("only response_mode=query is accepted", func(t *testing.T) {
		req := base()
		req.ResponseMode = "query"
		require.NoError(t, req.Validate())

		req = base()
		req.ResponseMode = "fragment"
		require.Error(t, req.Validate())
	})

	t.Run("acr_values, login_hint and ui_locales are accepted", func(t *testing.T) {
		req := base()
		req.ACRValues = "2"
		req.LoginHint = "a@b.com"
		req.UILocales = "en-GB en"
		require.NoError(t, req.Validate())
		assert.Equal(t, "2", req.ACRValues)
	})
}

// The OIDC discovery document omitted end_session_endpoint entirely, so an RP
// doing conformant discovery could not find RP-initiated logout at all; and it
// advertised introspection on the public host, where the route 404s.
func TestOAuthDiscoveryHandler_OIDCDocumentCompleteness(t *testing.T) {
	h := NewOAuthDiscoveryHandler(nil)
	r := httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil)
	w := httptest.NewRecorder()
	h.Discovery(w, r)

	var doc map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&doc))

	for _, key := range []string{
		"end_session_endpoint",
		"pushed_authorization_request_endpoint",
		"device_authorization_endpoint",
		"backchannel_authentication_endpoint",
		"acr_values_supported",
		"claims_supported",
		"token_endpoint_auth_signing_alg_values_supported",
	} {
		assert.Contains(t, doc, key, key)
	}

	_, advertised := doc["introspection_endpoint"]
	assert.False(t, advertised, "introspection is control-plane only; advertising it on the public host 404s")
	assert.Equal(t, false, doc["request_parameter_supported"])
	assert.Equal(t, true, doc["request_uri_parameter_supported"])
}

func TestOAuthDiscoveryHandler_AuthorizationServerMetadata_NoIntrospection(t *testing.T) {
	h := NewOAuthDiscoveryHandler(nil)
	r := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	w := httptest.NewRecorder()
	h.AuthorizationServerMetadata(w, r)

	var doc map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&doc))

	_, advertised := doc["introspection_endpoint"]
	assert.False(t, advertised)
	assert.Contains(t, doc, "token_endpoint_auth_signing_alg_values_supported")
}

package oauth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/dpop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// formReq creates a POST request with application/x-www-form-urlencoded body.
func formReq(t *testing.T, target string, values url.Values) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, target, strings.NewReader(values.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

// basicAuth returns the HTTP Basic Authorization header value.
func basicAuth(user, pass string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}

// ---------------------------------------------------------------------------
// NewOAuthTokenHandler
// ---------------------------------------------------------------------------

func TestNewOAuthTokenHandler(t *testing.T) {
	h := NewOAuthTokenHandler(&mockOAuthTokenService{}, nil, nil)
	assert.NotNil(t, h)
}

// ---------------------------------------------------------------------------
// Token
// ---------------------------------------------------------------------------

func TestOAuthTokenHandler_Token_MalformedBody(t *testing.T) {
	h := NewOAuthTokenHandler(&mockOAuthTokenService{}, nil, nil)
	// Send a body that can't be parsed as form data — use a too-large body
	// that triggers ParseForm error by exceeding limit. In practice, a nil
	// body with wrong content type suffices with Go < 1.28.
	// Simplest way: pass a reader that errors on Read.
	r := httptest.NewRequest(http.MethodPost, "/oauth/token", errReader{})
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "invalid_request", body["error"])
}

func TestOAuthTokenHandler_Token_ValidationError(t *testing.T) {
	h := NewOAuthTokenHandler(&mockOAuthTokenService{}, nil, nil)
	// Missing grant_type.
	r := formReq(t, "/oauth/token", url.Values{"code": {"abc"}})
	w := httptest.NewRecorder()

	h.Token(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "invalid_request", body["error"])
}

func TestOAuthTokenHandler_Token_InvalidGrantType(t *testing.T) {
	h := NewOAuthTokenHandler(&mockOAuthTokenService{}, nil, nil)
	r := formReq(t, "/oauth/token", url.Values{"grant_type": {"password"}})
	w := httptest.NewRecorder()

	h.Token(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOAuthTokenHandler_Token_ServiceOAuthError(t *testing.T) {
	svc := &mockOAuthTokenService{
		exchangeFn: func(_ context.Context, _ OAuthTokenRequestDTO, _ OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
			return nil, apperror.NewOAuthInvalidGrant("expired code")
		},
	}
	h := NewOAuthTokenHandler(svc, nil, nil)
	r := formReq(t, "/oauth/token", url.Values{
		"grant_type": {"authorization_code"},
		"code":       {"somecode"},
	})
	w := httptest.NewRecorder()

	h.Token(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "invalid_grant", body["error"])
}

func TestOAuthTokenHandler_Token_Success_AuthorizationCode(t *testing.T) {
	svc := &mockOAuthTokenService{
		exchangeFn: func(_ context.Context, _ OAuthTokenRequestDTO, _ OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
			return &OAuthTokenResult{
				AccessToken:  "at-123",
				TokenType:    "Bearer",
				ExpiresIn:    900,
				RefreshToken: "rt-456",
				IDToken:      "id-789",
				Scope:        "openid profile",
			}, nil
		},
	}
	h := NewOAuthTokenHandler(svc, nil, nil)
	r := formReq(t, "/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"valid-code"},
		"redirect_uri":  {"https://app.example.com/cb"},
		"code_verifier": {"dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"},
		"client_id":     {"myapp"},
	})
	w := httptest.NewRecorder()

	h.Token(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	assert.Equal(t, "no-cache", w.Header().Get("Pragma"))

	var resp OAuthTokenResponseDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "at-123", resp.AccessToken)
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.Equal(t, int64(900), resp.ExpiresIn)
	assert.Equal(t, "rt-456", resp.RefreshToken)
	assert.Equal(t, "id-789", resp.IDToken)
	assert.Equal(t, "openid profile", resp.Scope)
}

func TestOAuthTokenHandler_Token_Success_RefreshToken(t *testing.T) {
	svc := &mockOAuthTokenService{
		exchangeFn: func(_ context.Context, _ OAuthTokenRequestDTO, _ OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
			return &OAuthTokenResult{
				AccessToken: "at-new",
				TokenType:   "Bearer",
				ExpiresIn:   900,
			}, nil
		},
	}
	h := NewOAuthTokenHandler(svc, nil, nil)
	r := formReq(t, "/oauth/token", url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {"rt-old"},
	})
	w := httptest.NewRecorder()

	h.Token(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOAuthTokenHandler_Token_Success_ClientCredentials(t *testing.T) {
	svc := &mockOAuthTokenService{
		exchangeFn: func(_ context.Context, _ OAuthTokenRequestDTO, _ OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
			return &OAuthTokenResult{
				AccessToken: "at-cc",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
			}, nil
		},
	}
	h := NewOAuthTokenHandler(svc, nil, nil)
	r := formReq(t, "/oauth/token", url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {"myapp"},
		"client_secret": {"secret"},
		"scope":         {"openid"},
	})
	w := httptest.NewRecorder()

	h.Token(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOAuthTokenHandler_Token_BasicAuthCredentials(t *testing.T) {
	var capturedCreds OAuthClientCredentials
	svc := &mockOAuthTokenService{
		exchangeFn: func(_ context.Context, _ OAuthTokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
			capturedCreds = creds
			return &OAuthTokenResult{
				AccessToken: "at",
				TokenType:   "Bearer",
				ExpiresIn:   900,
			}, nil
		},
	}
	h := NewOAuthTokenHandler(svc, nil, nil)
	r := formReq(t, "/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"c"},
		"client_id":     {"body-id"},
		"client_secret": {"body-secret"},
	})
	// HTTP Basic should take precedence over form body.
	r.Header.Set("Authorization", basicAuth("basic-id", "basic-secret"))
	w := httptest.NewRecorder()

	h.Token(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "basic-id", capturedCreds.ClientID)
	assert.Equal(t, "basic-secret", capturedCreds.ClientSecret)
}

func TestOAuthTokenHandler_Token_FormBodyCredentials(t *testing.T) {
	var capturedCreds OAuthClientCredentials
	svc := &mockOAuthTokenService{
		exchangeFn: func(_ context.Context, _ OAuthTokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
			capturedCreds = creds
			return &OAuthTokenResult{
				AccessToken: "at",
				TokenType:   "Bearer",
				ExpiresIn:   900,
			}, nil
		},
	}
	h := NewOAuthTokenHandler(svc, nil, nil)
	r := formReq(t, "/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"c"},
		"client_id":     {"form-id"},
		"client_secret": {"form-secret"},
	})
	// No Authorization header — should use form body.
	w := httptest.NewRecorder()

	h.Token(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "form-id", capturedCreds.ClientID)
	assert.Equal(t, "form-secret", capturedCreds.ClientSecret)
}

// ---------------------------------------------------------------------------
// Revoke
// ---------------------------------------------------------------------------

func TestOAuthTokenHandler_Revoke_MalformedBody(t *testing.T) {
	h := NewOAuthTokenHandler(&mockOAuthTokenService{}, nil, nil)
	r := httptest.NewRequest(http.MethodPost, "/oauth/revoke", errReader{})
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Revoke(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOAuthTokenHandler_Revoke_ValidationError(t *testing.T) {
	h := NewOAuthTokenHandler(&mockOAuthTokenService{}, nil, nil)
	// Missing token.
	r := formReq(t, "/oauth/revoke", url.Values{})
	w := httptest.NewRecorder()

	h.Revoke(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOAuthTokenHandler_Revoke_InvalidTokenTypeHint(t *testing.T) {
	h := NewOAuthTokenHandler(&mockOAuthTokenService{}, nil, nil)
	r := formReq(t, "/oauth/revoke", url.Values{
		"token":           {"some-token"},
		"token_type_hint": {"invalid_hint"},
	})
	w := httptest.NewRecorder()

	h.Revoke(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOAuthTokenHandler_Revoke_ServiceOAuthError(t *testing.T) {
	svc := &mockOAuthTokenService{
		revokeFn: func(_ context.Context, _ OAuthRevokeRequestDTO, _ OAuthClientCredentials) *apperror.OAuthError {
			return apperror.NewOAuthInvalidClient("unknown client")
		},
	}
	h := NewOAuthTokenHandler(svc, nil, nil)
	r := formReq(t, "/oauth/revoke", url.Values{
		"token":     {"some-token"},
		"client_id": {"myapp"},
	})
	w := httptest.NewRecorder()

	h.Revoke(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "invalid_client", body["error"])
}

func TestOAuthTokenHandler_Revoke_Success(t *testing.T) {
	svc := &mockOAuthTokenService{
		revokeFn: func(_ context.Context, _ OAuthRevokeRequestDTO, _ OAuthClientCredentials) *apperror.OAuthError {
			return nil
		},
	}
	h := NewOAuthTokenHandler(svc, nil, nil)
	r := formReq(t, "/oauth/revoke", url.Values{
		"token": {"some-token"},
	})
	w := httptest.NewRecorder()

	h.Revoke(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Body.String())
}

func TestOAuthTokenHandler_Revoke_BasicAuthCredentials(t *testing.T) {
	var capturedCreds OAuthClientCredentials
	svc := &mockOAuthTokenService{
		revokeFn: func(_ context.Context, _ OAuthRevokeRequestDTO, creds OAuthClientCredentials) *apperror.OAuthError {
			capturedCreds = creds
			return nil
		},
	}
	h := NewOAuthTokenHandler(svc, nil, nil)
	r := formReq(t, "/oauth/revoke", url.Values{
		"token":     {"some-token"},
		"client_id": {"body-id"},
	})
	r.Header.Set("Authorization", basicAuth("basic-id", "basic-secret"))
	w := httptest.NewRecorder()

	h.Revoke(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "basic-id", capturedCreds.ClientID)
	assert.Equal(t, "basic-secret", capturedCreds.ClientSecret)
}

// ---------------------------------------------------------------------------
// Introspect
// ---------------------------------------------------------------------------

func TestOAuthTokenHandler_Introspect_MalformedBody(t *testing.T) {
	h := NewOAuthTokenHandler(&mockOAuthTokenService{}, nil, nil)
	r := httptest.NewRequest(http.MethodPost, "/oauth/introspect", errReader{})
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Introspect(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOAuthTokenHandler_Introspect_ValidationError(t *testing.T) {
	h := NewOAuthTokenHandler(&mockOAuthTokenService{}, nil, nil)
	// Missing token.
	r := formReq(t, "/oauth/introspect", url.Values{})
	w := httptest.NewRecorder()

	h.Introspect(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOAuthTokenHandler_Introspect_InvalidTokenTypeHint(t *testing.T) {
	h := NewOAuthTokenHandler(&mockOAuthTokenService{}, nil, nil)
	r := formReq(t, "/oauth/introspect", url.Values{
		"token":           {"some-token"},
		"token_type_hint": {"bad_hint"},
	})
	w := httptest.NewRecorder()

	h.Introspect(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOAuthTokenHandler_Introspect_ServiceOAuthError(t *testing.T) {
	svc := &mockOAuthTokenService{
		introspectFn: func(_ context.Context, _ OAuthIntrospectRequestDTO, _ OAuthClientCredentials) (*OAuthIntrospectResponseDTO, *apperror.OAuthError) {
			return nil, apperror.NewOAuthInvalidRequest("missing token")
		},
	}
	h := NewOAuthTokenHandler(svc, nil, nil)
	r := formReq(t, "/oauth/introspect", url.Values{
		"token": {"some-token"},
	})
	w := httptest.NewRecorder()

	h.Introspect(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOAuthTokenHandler_Introspect_ActiveToken(t *testing.T) {
	svc := &mockOAuthTokenService{
		introspectFn: func(_ context.Context, _ OAuthIntrospectRequestDTO, _ OAuthClientCredentials) (*OAuthIntrospectResponseDTO, *apperror.OAuthError) {
			return &OAuthIntrospectResponseDTO{
				Active:    true,
				Scope:     "openid profile",
				ClientID:  "myapp",
				Username:  "alice",
				TokenType: "Bearer",
				Exp:       1700000000,
				Sub:       "user-uuid",
			}, nil
		},
	}
	h := NewOAuthTokenHandler(svc, nil, nil)
	r := formReq(t, "/oauth/introspect", url.Values{
		"token": {"valid-token"},
	})
	w := httptest.NewRecorder()

	h.Introspect(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	assert.Equal(t, "no-cache", w.Header().Get("Pragma"))

	var resp OAuthIntrospectResponseDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp.Active)
	assert.Equal(t, "openid profile", resp.Scope)
	assert.Equal(t, "myapp", resp.ClientID)
	assert.Equal(t, "alice", resp.Username)
	assert.Equal(t, "user-uuid", resp.Sub)
}

func TestOAuthTokenHandler_Introspect_InactiveToken(t *testing.T) {
	svc := &mockOAuthTokenService{
		introspectFn: func(_ context.Context, _ OAuthIntrospectRequestDTO, _ OAuthClientCredentials) (*OAuthIntrospectResponseDTO, *apperror.OAuthError) {
			return &OAuthIntrospectResponseDTO{Active: false}, nil
		},
	}
	h := NewOAuthTokenHandler(svc, nil, nil)
	r := formReq(t, "/oauth/introspect", url.Values{
		"token": {"expired-token"},
	})
	w := httptest.NewRecorder()

	h.Introspect(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp OAuthIntrospectResponseDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.False(t, resp.Active)
}

// ---------------------------------------------------------------------------
// parseBasicAuth (exercised indirectly through extractClientCredentials)
// ---------------------------------------------------------------------------

func TestOAuthTokenHandler_Token_InvalidBasicAuth_NotBase64(t *testing.T) {
	var capturedCreds OAuthClientCredentials
	svc := &mockOAuthTokenService{
		exchangeFn: func(_ context.Context, _ OAuthTokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
			capturedCreds = creds
			return &OAuthTokenResult{AccessToken: "at", TokenType: "Bearer", ExpiresIn: 900}, nil
		},
	}
	h := NewOAuthTokenHandler(svc, nil, nil)
	r := formReq(t, "/oauth/token", url.Values{
		"grant_type": {"authorization_code"},
		"code":       {"c"},
		"client_id":  {"fallback-id"},
	})
	// Invalid base64 after "Basic " — should fall back to form body.
	r.Header.Set("Authorization", "Basic !!!invalid!!!")
	w := httptest.NewRecorder()

	h.Token(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "fallback-id", capturedCreds.ClientID)
}

func TestOAuthTokenHandler_Token_InvalidBasicAuth_NoColon(t *testing.T) {
	var capturedCreds OAuthClientCredentials
	svc := &mockOAuthTokenService{
		exchangeFn: func(_ context.Context, _ OAuthTokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
			capturedCreds = creds
			return &OAuthTokenResult{AccessToken: "at", TokenType: "Bearer", ExpiresIn: 900}, nil
		},
	}
	h := NewOAuthTokenHandler(svc, nil, nil)
	r := formReq(t, "/oauth/token", url.Values{
		"grant_type": {"authorization_code"},
		"code":       {"c"},
		"client_id":  {"fallback-id2"},
	})
	// "nocolon" base64 = "bm9jb2xvbg==" — no ":" separator.
	r.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("nocolon")))
	w := httptest.NewRecorder()

	h.Token(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "fallback-id2", capturedCreds.ClientID)
}

func TestOAuthTokenHandler_Token_BearerAuth_FallsBackToForm(t *testing.T) {
	var capturedCreds OAuthClientCredentials
	svc := &mockOAuthTokenService{
		exchangeFn: func(_ context.Context, _ OAuthTokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
			capturedCreds = creds
			return &OAuthTokenResult{AccessToken: "at", TokenType: "Bearer", ExpiresIn: 900}, nil
		},
	}
	h := NewOAuthTokenHandler(svc, nil, nil)
	r := formReq(t, "/oauth/token", url.Values{
		"grant_type": {"authorization_code"},
		"code":       {"c"},
		"client_id":  {"form-id"},
	})
	// Non-Basic scheme — should fall back to form body.
	r.Header.Set("Authorization", "Bearer some-token")
	w := httptest.NewRecorder()

	h.Token(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "form-id", capturedCreds.ClientID)
}

func TestOAuthTokenHandler_Token_NoAuthHeader_FallsBackToForm(t *testing.T) {
	var capturedCreds OAuthClientCredentials
	svc := &mockOAuthTokenService{
		exchangeFn: func(_ context.Context, _ OAuthTokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
			capturedCreds = creds
			return &OAuthTokenResult{AccessToken: "at", TokenType: "Bearer", ExpiresIn: 900}, nil
		},
	}
	h := NewOAuthTokenHandler(svc, nil, nil)
	r := formReq(t, "/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"c"},
		"client_id":     {"no-auth-id"},
		"client_secret": {"no-auth-secret"},
	})
	w := httptest.NewRecorder()

	h.Token(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "no-auth-id", capturedCreds.ClientID)
	assert.Equal(t, "no-auth-secret", capturedCreds.ClientSecret)
}

// errReader is an io.Reader that always returns an error.
type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, assert.AnError
}

// mockDpopStore is a minimal mock of dpop.JTIStore for handler tests.
type mockDpopStore struct{}

func (mockDpopStore) DenyJTI(_ context.Context, _ string, _ time.Duration) error { return nil }
func (mockDpopStore) IsJTIDenied(_ context.Context, _ string) (bool, error)      { return false, nil }

func TestOAuthTokenHandler_Token_InvalidDPoPProof(t *testing.T) {
	svc := &mockOAuthTokenService{
		exchangeFn: func(_ context.Context, _ OAuthTokenRequestDTO, _ OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
			return &OAuthTokenResult{AccessToken: "at"}, nil
		},
	}

	// Use a real NonceManager and mock DPoP store so the DPoP path is exercised.
	nm := dpop.NewNonceManager()
	store := &mockDpopStore{}

	h := NewOAuthTokenHandler(svc, nm, store)
	r := formReq(t, "/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"valid-code"},
		"redirect_uri":  {"https://app.example.com/cb"},
		"code_verifier": {"dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"},
		"client_id":     {"myapp"},
	})
	r.Header.Set("DPoP", "invalid-dpop-proof-jwt")
	w := httptest.NewRecorder()

	h.Token(w, r)

	// DPoP validation should fail with an invalid proof → 400.
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOAuthTokenHandler_Token_DPoPHeaderIgnoredWithoutStore(t *testing.T) {
	svc := &mockOAuthTokenService{
		exchangeFn: func(_ context.Context, _ OAuthTokenRequestDTO, _ OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
			return &OAuthTokenResult{AccessToken: "at"}, nil
		},
	}

	// No DPoP store — DPoP header should be ignored, handler proceeds normally.
	h := NewOAuthTokenHandler(svc, nil, nil)
	r := formReq(t, "/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"valid-code"},
		"redirect_uri":  {"https://app.example.com/cb"},
		"code_verifier": {"dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"},
		"client_id":     {"myapp"},
	})
	r.Header.Set("DPoP", "ignored-proof")
	w := httptest.NewRecorder()

	h.Token(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOAuthTokenHandler_Token_SuccessWithDPoPProof(t *testing.T) {
	origHost := config.AppPublicHostname
	config.AppPublicHostname = "https://auth.example.com"
	defer func() { config.AppPublicHostname = origHost }()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	jwk := ecJWKForTest(t, &key.PublicKey)

	claims := jwtlib.MapClaims{
		"jti": "test-dpop-jti-2",
		"htm": "POST",
		"htu": "https://auth.example.com/api/v1/oauth/token",
		"iat": time.Now().Unix(),
	}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodES256, claims)
	token.Header["typ"] = "dpop+jwt"
	token.Header["jwk"] = jwk
	proof, err := token.SignedString(key)
	require.NoError(t, err)

	var capturedThumbprint string
	svc := &mockOAuthTokenService{
		exchangeFn: func(_ context.Context, req OAuthTokenRequestDTO, _ OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
			capturedThumbprint = req.DPoPThumbprint
			return &OAuthTokenResult{AccessToken: "at", TokenType: "Bearer", ExpiresIn: 900}, nil
		},
	}

	nm := dpop.NewNonceManager()
	store := &mockDpopStore{}

	h := NewOAuthTokenHandler(svc, nm, store)
	r := formReq(t, "/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"valid-code"},
		"redirect_uri":  {"https://app.example.com/cb"},
		"code_verifier": {"dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"},
		"client_id":     {"myapp"},
	})
	r.Header.Set("DPoP", proof)
	w := httptest.NewRecorder()

	h.Token(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, capturedThumbprint)
	assert.NotEmpty(t, w.Header().Get("DPoP-Nonce"))
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
}

func ecJWKForTest(t *testing.T, pub *ecdsa.PublicKey) map[string]any {
	t.Helper()
	ecdhKey, err := pub.ECDH()
	require.NoError(t, err)
	point := ecdhKey.Bytes()
	size := (len(point) - 1) / 2
	return map[string]any{
		"kty": "EC",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(point[1 : 1+size]),
		"y":   base64.RawURLEncoding.EncodeToString(point[1+size:]),
	}
}



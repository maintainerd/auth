package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/dto"
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
	h := NewOAuthTokenHandler(&mockOAuthTokenService{})
	assert.NotNil(t, h)
}

// ---------------------------------------------------------------------------
// Token
// ---------------------------------------------------------------------------

func TestOAuthTokenHandler_Token_MalformedBody(t *testing.T) {
	h := NewOAuthTokenHandler(&mockOAuthTokenService{})
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
	h := NewOAuthTokenHandler(&mockOAuthTokenService{})
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
	h := NewOAuthTokenHandler(&mockOAuthTokenService{})
	r := formReq(t, "/oauth/token", url.Values{"grant_type": {"password"}})
	w := httptest.NewRecorder()

	h.Token(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOAuthTokenHandler_Token_ServiceOAuthError(t *testing.T) {
	svc := &mockOAuthTokenService{
		exchangeFn: func(_ context.Context, _ dto.OAuthTokenRequestDTO, _ dto.OAuthClientCredentials) (*dto.OAuthTokenResult, *apperror.OAuthError) {
			return nil, apperror.NewOAuthInvalidGrant("expired code")
		},
	}
	h := NewOAuthTokenHandler(svc)
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
		exchangeFn: func(_ context.Context, _ dto.OAuthTokenRequestDTO, _ dto.OAuthClientCredentials) (*dto.OAuthTokenResult, *apperror.OAuthError) {
			return &dto.OAuthTokenResult{
				AccessToken:  "at-123",
				TokenType:    "Bearer",
				ExpiresIn:    900,
				RefreshToken: "rt-456",
				IDToken:      "id-789",
				Scope:        "openid profile",
			}, nil
		},
	}
	h := NewOAuthTokenHandler(svc)
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

	var resp dto.OAuthTokenResponseDTO
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
		exchangeFn: func(_ context.Context, _ dto.OAuthTokenRequestDTO, _ dto.OAuthClientCredentials) (*dto.OAuthTokenResult, *apperror.OAuthError) {
			return &dto.OAuthTokenResult{
				AccessToken: "at-new",
				TokenType:   "Bearer",
				ExpiresIn:   900,
			}, nil
		},
	}
	h := NewOAuthTokenHandler(svc)
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
		exchangeFn: func(_ context.Context, _ dto.OAuthTokenRequestDTO, _ dto.OAuthClientCredentials) (*dto.OAuthTokenResult, *apperror.OAuthError) {
			return &dto.OAuthTokenResult{
				AccessToken: "at-cc",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
			}, nil
		},
	}
	h := NewOAuthTokenHandler(svc)
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
	var capturedCreds dto.OAuthClientCredentials
	svc := &mockOAuthTokenService{
		exchangeFn: func(_ context.Context, _ dto.OAuthTokenRequestDTO, creds dto.OAuthClientCredentials) (*dto.OAuthTokenResult, *apperror.OAuthError) {
			capturedCreds = creds
			return &dto.OAuthTokenResult{
				AccessToken: "at",
				TokenType:   "Bearer",
				ExpiresIn:   900,
			}, nil
		},
	}
	h := NewOAuthTokenHandler(svc)
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
	var capturedCreds dto.OAuthClientCredentials
	svc := &mockOAuthTokenService{
		exchangeFn: func(_ context.Context, _ dto.OAuthTokenRequestDTO, creds dto.OAuthClientCredentials) (*dto.OAuthTokenResult, *apperror.OAuthError) {
			capturedCreds = creds
			return &dto.OAuthTokenResult{
				AccessToken: "at",
				TokenType:   "Bearer",
				ExpiresIn:   900,
			}, nil
		},
	}
	h := NewOAuthTokenHandler(svc)
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
	h := NewOAuthTokenHandler(&mockOAuthTokenService{})
	r := httptest.NewRequest(http.MethodPost, "/oauth/revoke", errReader{})
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Revoke(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOAuthTokenHandler_Revoke_ValidationError(t *testing.T) {
	h := NewOAuthTokenHandler(&mockOAuthTokenService{})
	// Missing token.
	r := formReq(t, "/oauth/revoke", url.Values{})
	w := httptest.NewRecorder()

	h.Revoke(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOAuthTokenHandler_Revoke_InvalidTokenTypeHint(t *testing.T) {
	h := NewOAuthTokenHandler(&mockOAuthTokenService{})
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
		revokeFn: func(_ context.Context, _ dto.OAuthRevokeRequestDTO, _ dto.OAuthClientCredentials) *apperror.OAuthError {
			return apperror.NewOAuthInvalidClient("unknown client")
		},
	}
	h := NewOAuthTokenHandler(svc)
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
		revokeFn: func(_ context.Context, _ dto.OAuthRevokeRequestDTO, _ dto.OAuthClientCredentials) *apperror.OAuthError {
			return nil
		},
	}
	h := NewOAuthTokenHandler(svc)
	r := formReq(t, "/oauth/revoke", url.Values{
		"token": {"some-token"},
	})
	w := httptest.NewRecorder()

	h.Revoke(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Body.String())
}

func TestOAuthTokenHandler_Revoke_BasicAuthCredentials(t *testing.T) {
	var capturedCreds dto.OAuthClientCredentials
	svc := &mockOAuthTokenService{
		revokeFn: func(_ context.Context, _ dto.OAuthRevokeRequestDTO, creds dto.OAuthClientCredentials) *apperror.OAuthError {
			capturedCreds = creds
			return nil
		},
	}
	h := NewOAuthTokenHandler(svc)
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
	h := NewOAuthTokenHandler(&mockOAuthTokenService{})
	r := httptest.NewRequest(http.MethodPost, "/oauth/introspect", errReader{})
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Introspect(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOAuthTokenHandler_Introspect_ValidationError(t *testing.T) {
	h := NewOAuthTokenHandler(&mockOAuthTokenService{})
	// Missing token.
	r := formReq(t, "/oauth/introspect", url.Values{})
	w := httptest.NewRecorder()

	h.Introspect(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOAuthTokenHandler_Introspect_InvalidTokenTypeHint(t *testing.T) {
	h := NewOAuthTokenHandler(&mockOAuthTokenService{})
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
		introspectFn: func(_ context.Context, _ dto.OAuthIntrospectRequestDTO) (*dto.OAuthIntrospectResponseDTO, *apperror.OAuthError) {
			return nil, apperror.NewOAuthInvalidRequest("missing token")
		},
	}
	h := NewOAuthTokenHandler(svc)
	r := formReq(t, "/oauth/introspect", url.Values{
		"token": {"some-token"},
	})
	w := httptest.NewRecorder()

	h.Introspect(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOAuthTokenHandler_Introspect_ActiveToken(t *testing.T) {
	svc := &mockOAuthTokenService{
		introspectFn: func(_ context.Context, _ dto.OAuthIntrospectRequestDTO) (*dto.OAuthIntrospectResponseDTO, *apperror.OAuthError) {
			return &dto.OAuthIntrospectResponseDTO{
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
	h := NewOAuthTokenHandler(svc)
	r := formReq(t, "/oauth/introspect", url.Values{
		"token": {"valid-token"},
	})
	w := httptest.NewRecorder()

	h.Introspect(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	assert.Equal(t, "no-cache", w.Header().Get("Pragma"))

	var resp dto.OAuthIntrospectResponseDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp.Active)
	assert.Equal(t, "openid profile", resp.Scope)
	assert.Equal(t, "myapp", resp.ClientID)
	assert.Equal(t, "alice", resp.Username)
	assert.Equal(t, "user-uuid", resp.Sub)
}

func TestOAuthTokenHandler_Introspect_InactiveToken(t *testing.T) {
	svc := &mockOAuthTokenService{
		introspectFn: func(_ context.Context, _ dto.OAuthIntrospectRequestDTO) (*dto.OAuthIntrospectResponseDTO, *apperror.OAuthError) {
			return &dto.OAuthIntrospectResponseDTO{Active: false}, nil
		},
	}
	h := NewOAuthTokenHandler(svc)
	r := formReq(t, "/oauth/introspect", url.Values{
		"token": {"expired-token"},
	})
	w := httptest.NewRecorder()

	h.Introspect(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.OAuthIntrospectResponseDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.False(t, resp.Active)
}

// ---------------------------------------------------------------------------
// parseBasicAuth (exercised indirectly through extractClientCredentials)
// ---------------------------------------------------------------------------

func TestOAuthTokenHandler_Token_InvalidBasicAuth_NotBase64(t *testing.T) {
	var capturedCreds dto.OAuthClientCredentials
	svc := &mockOAuthTokenService{
		exchangeFn: func(_ context.Context, _ dto.OAuthTokenRequestDTO, creds dto.OAuthClientCredentials) (*dto.OAuthTokenResult, *apperror.OAuthError) {
			capturedCreds = creds
			return &dto.OAuthTokenResult{AccessToken: "at", TokenType: "Bearer", ExpiresIn: 900}, nil
		},
	}
	h := NewOAuthTokenHandler(svc)
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
	var capturedCreds dto.OAuthClientCredentials
	svc := &mockOAuthTokenService{
		exchangeFn: func(_ context.Context, _ dto.OAuthTokenRequestDTO, creds dto.OAuthClientCredentials) (*dto.OAuthTokenResult, *apperror.OAuthError) {
			capturedCreds = creds
			return &dto.OAuthTokenResult{AccessToken: "at", TokenType: "Bearer", ExpiresIn: 900}, nil
		},
	}
	h := NewOAuthTokenHandler(svc)
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
	var capturedCreds dto.OAuthClientCredentials
	svc := &mockOAuthTokenService{
		exchangeFn: func(_ context.Context, _ dto.OAuthTokenRequestDTO, creds dto.OAuthClientCredentials) (*dto.OAuthTokenResult, *apperror.OAuthError) {
			capturedCreds = creds
			return &dto.OAuthTokenResult{AccessToken: "at", TokenType: "Bearer", ExpiresIn: 900}, nil
		},
	}
	h := NewOAuthTokenHandler(svc)
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
	var capturedCreds dto.OAuthClientCredentials
	svc := &mockOAuthTokenService{
		exchangeFn: func(_ context.Context, _ dto.OAuthTokenRequestDTO, creds dto.OAuthClientCredentials) (*dto.OAuthTokenResult, *apperror.OAuthError) {
			capturedCreds = creds
			return &dto.OAuthTokenResult{AccessToken: "at", TokenType: "Bearer", ExpiresIn: 900}, nil
		},
	}
	h := NewOAuthTokenHandler(svc)
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

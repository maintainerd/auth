package handler

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maintainerd/auth/internal/config"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NewOAuthDiscoveryHandler
// ---------------------------------------------------------------------------

func TestNewOAuthDiscoveryHandler(t *testing.T) {
	h := NewOAuthDiscoveryHandler()
	assert.NotNil(t, h)
}

// ---------------------------------------------------------------------------
// Discovery
// ---------------------------------------------------------------------------

func TestOAuthDiscoveryHandler_Discovery(t *testing.T) {
	origHost := config.AppPublicHostname
	t.Cleanup(func() { config.AppPublicHostname = origHost })
	config.AppPublicHostname = "https://auth.example.com"

	h := NewOAuthDiscoveryHandler()
	r := httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil)
	w := httptest.NewRecorder()

	h.Discovery(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "public, max-age=3600", w.Header().Get("Cache-Control"))

	var doc dto.OAuthDiscoveryResponseDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&doc))

	assert.Equal(t, "https://auth.example.com", doc.Issuer)
	assert.Equal(t, "https://auth.example.com/api/v1/oauth/authorize", doc.AuthorizationEndpoint)
	assert.Equal(t, "https://auth.example.com/api/v1/oauth/token", doc.TokenEndpoint)
	assert.Equal(t, "https://auth.example.com/api/v1/oauth/userinfo", doc.UserinfoEndpoint)
	assert.Equal(t, "https://auth.example.com/.well-known/jwks.json", doc.JwksURI)
	assert.Equal(t, "https://auth.example.com/api/v1/oauth/revoke", doc.RevocationEndpoint)
	assert.Equal(t, "https://auth.example.com/api/v1/oauth/introspect", doc.IntrospectionEndpoint)
	assert.Equal(t, []string{"openid", "profile", "email", "offline_access"}, doc.ScopesSupported)
	assert.Equal(t, []string{"code"}, doc.ResponseTypesSupp)
	assert.Equal(t, []string{"authorization_code", "refresh_token", "client_credentials"}, doc.GrantTypesSupported)
	assert.Equal(t, []string{"public"}, doc.SubjectTypesSupported)
	assert.Equal(t, []string{"RS256"}, doc.IDTokenSignAlgValues)
	assert.Equal(t, []string{"client_secret_basic", "client_secret_post", "none"}, doc.TokenEndpointAuth)
	assert.Equal(t, []string{"S256"}, doc.CodeChallengeMethods)
}

// ---------------------------------------------------------------------------
// JWKS
// ---------------------------------------------------------------------------

func TestOAuthDiscoveryHandler_JWKS_KeysNotInitialised(t *testing.T) {
	jwt.ResetJWTKeys()
	t.Cleanup(jwt.ResetJWTKeys)

	h := NewOAuthDiscoveryHandler()
	r := httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil)
	w := httptest.NewRecorder()

	h.JWKS(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "keys not initialised", body["error"])
}

func TestOAuthDiscoveryHandler_JWKS_Success(t *testing.T) {
	// Generate an RSA key pair and set it via config + init.
	initTestJWTKeysForHandler(t)

	t.Setenv("JWT_KEY_ID", "test-kid-1")

	h := NewOAuthDiscoveryHandler()
	r := httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil)
	w := httptest.NewRecorder()

	h.JWKS(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "public, max-age=3600", w.Header().Get("Cache-Control"))

	var jwks dto.JWKSResponseDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&jwks))
	require.Len(t, jwks.Keys, 1)

	key := jwks.Keys[0]
	assert.Equal(t, "RSA", key.Kty)
	assert.Equal(t, "sig", key.Use)
	assert.Equal(t, "test-kid-1", key.Kid)
	assert.Equal(t, "RS256", key.Alg)
	assert.NotEmpty(t, key.N)
	assert.NotEmpty(t, key.E)
}

func TestOAuthDiscoveryHandler_JWKS_DefaultKeyID(t *testing.T) {
	initTestJWTKeysForHandler(t)

	// No JWT_KEY_ID env set — should fall back to default.
	h := NewOAuthDiscoveryHandler()
	r := httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil)
	w := httptest.NewRecorder()

	h.JWKS(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var jwks dto.JWKSResponseDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&jwks))
	require.Len(t, jwks.Keys, 1)
	assert.Equal(t, "maintainerd-auth-key-1", jwks.Keys[0].Kid)
}

// initTestJWTKeysForHandler generates an RSA key pair, sets config vars,
// and calls jwt.InitJWTKeys. It cleans up after the test.
func initTestJWTKeysForHandler(t *testing.T) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privPEM := pemEncodeRSAPrivateKey(key)
	pubPEM := pemEncodeRSAPublicKey(&key.PublicKey)

	origPriv := config.JWTPrivateKey
	origPub := config.JWTPublicKey
	t.Cleanup(func() {
		config.JWTPrivateKey = origPriv
		config.JWTPublicKey = origPub
		jwt.ResetJWTKeys()
	})

	config.JWTPrivateKey = privPEM
	config.JWTPublicKey = pubPEM
	require.NoError(t, jwt.InitJWTKeys())
}

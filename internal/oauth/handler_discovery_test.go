package oauth

import (
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthDiscoveryHandler_AuthorizationServerMetadata(t *testing.T) {
	origHost := config.AppPublicHostname
	config.AppPublicHostname = "https://auth.example.com"
	defer func() { config.AppPublicHostname = origHost }()

	h := NewOAuthDiscoveryHandler()
	r := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	w := httptest.NewRecorder()

	h.AuthorizationServerMetadata(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "public, max-age=3600", w.Header().Get("Cache-Control"))

	var doc OAuthAuthorizationServerMetadataDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&doc))

	assert.Equal(t, "https://auth.example.com", doc.Issuer)
	assert.NotEmpty(t, doc.AuthorizationEndpoint)
	assert.NotEmpty(t, doc.TokenEndpoint)
	assert.NotEmpty(t, doc.JwksURI)
	assert.Contains(t, doc.GrantTypesSupported, "authorization_code")
	assert.Contains(t, doc.ResponseTypesSupported, "code")
}

func TestOAuthDiscoveryHandler_Discovery(t *testing.T) {
	origHost := config.AppPublicHostname
	config.AppPublicHostname = "https://auth.example.com"
	defer func() { config.AppPublicHostname = origHost }()

	w := httptest.NewRecorder()
	NewOAuthDiscoveryHandler().Discovery(w, httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "public, max-age=3600", w.Header().Get("Cache-Control"))
	var doc OAuthDiscoveryResponseDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&doc))
	assert.Equal(t, "https://auth.example.com", doc.Issuer)
	assert.Contains(t, doc.ScopesSupported, "openid")
}

func TestOAuthDiscoveryHandler_JWKS(t *testing.T) {
	initTestJWTKeysService(t)

	w := httptest.NewRecorder()
	NewOAuthDiscoveryHandler().JWKS(w, httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	var body JWKSResponseDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.NotEmpty(t, body.Keys)
	assert.Equal(t, "RSA", body.Keys[0].Kty)
	assert.NotEmpty(t, body.Keys[0].N)
	assert.NotEmpty(t, body.Keys[0].E)
}

func TestBase64URLEncodeUint(t *testing.T) {
	assert.Equal(t, base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}), base64URLEncodeUint(big.NewInt(65537)))
}

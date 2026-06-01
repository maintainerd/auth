package oauth

import (
	"encoding/json"
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

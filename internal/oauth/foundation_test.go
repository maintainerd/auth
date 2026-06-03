package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/maintainerd/auth/internal/platform/ptr"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientSecretMatches(t *testing.T) {
	currentHash, err := security.HashClientSecret(context.Background(), "current-secret")
	require.NoError(t, err)
	previousHash, err := security.HashClientSecret(context.Background(), "previous-secret")
	require.NoError(t, err)

	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)

	assert.True(t, clientSecretMatches(&Client{SecretHash: &currentHash}, "current-secret"))
	assert.True(t, clientSecretMatches(&Client{SecretHash: ptr.Ptr(currentHash), PreviousSecretHash: &previousHash, PreviousSecretExpiresAt: &future}, "previous-secret"))
	assert.False(t, clientSecretMatches(&Client{SecretHash: ptr.Ptr(currentHash), PreviousSecretHash: &previousHash, PreviousSecretExpiresAt: &past}, "previous-secret"))
	assert.False(t, clientSecretMatches(&Client{SecretHash: ptr.Ptr(currentHash)}, "wrong-secret"))
	assert.False(t, clientSecretMatches(&Client{SecretHash: ptr.Ptr(currentHash)}, ""))
}

func TestValidateClientAllowedScopes(t *testing.T) {
	client := &Client{AllowedScopes: []string{"openid", "email"}}

	assert.Nil(t, validateClientAllowedScopes(client, "openid email"))

	oerr := validateClientAllowedScopes(client, "openid profile")
	require.NotNil(t, oerr)
	assert.Equal(t, "invalid_scope", oerr.Code)

	assert.Nil(t, validateClientAllowedScopes(&Client{}, "openid profile"))
}

func TestValidateRequestedScopesSubset(t *testing.T) {
	assert.Nil(t, validateRequestedScopesSubset("", "openid email"))
	assert.Nil(t, validateRequestedScopesSubset("   ", "openid email"))
	assert.Nil(t, validateRequestedScopesSubset("openid", "openid email"))

	oerr := validateRequestedScopesSubset("profile", "openid email")
	require.NotNil(t, oerr)
	assert.Equal(t, "invalid_scope", oerr.Code)
}

func TestExtractOAuthClientCredentials(t *testing.T) {
	t.Run("uses basic auth when present", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
		req.SetBasicAuth("basic-client", "basic-secret")

		creds := extractOAuthClientCredentials(req, "form-client", "form-secret")

		assert.Equal(t, "basic-client", creds.ClientID)
		assert.Equal(t, "basic-secret", creds.ClientSecret)
	})

	t.Run("uses form credentials without basic auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)

		creds := extractOAuthClientCredentials(req, "form-client", "form-secret")

		assert.Equal(t, "form-client", creds.ClientID)
		assert.Equal(t, "form-secret", creds.ClientSecret)
	})
}

func TestScopeHelpers(t *testing.T) {
	assert.Equal(t, map[string]struct{}{"openid": {}, "email": {}}, scopeSet([]string{" openid ", "", "email"}))
	assert.Equal(t, []string{"openid", "email"}, []string(parseScopeFields(" openid   email ")))
}

package client

import (
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These combinations are individually legal — every value passes its own
// allowlist and the DB CHECK constraints — and jointly unsafe. The matrix is the
// only thing standing between an operator and a client that either cannot
// authenticate at all or does not have to.
func TestValidateClientOAuthMatrix(t *testing.T) {
	authzCode := []string{GrantTypeAuthorizationCode}
	clientCreds := []string{GrantTypeClientCredentials}
	scopes := []string{"api:read"}

	t.Run("rejects none on a confidential client", func(t *testing.T) {
		for _, clientType := range []string{shared.ClientTypeTraditional, shared.ClientTypeM2M} {
			err := ValidateClientOAuthMatrix(clientType, TokenAuthMethodNone, nil, nil, false, false)
			require.Error(t, err, clientType)
			assert.Contains(t, err.Error(), "only valid for public clients")
		}
	})

	t.Run("allows none on a public client", func(t *testing.T) {
		for _, clientType := range []string{shared.ClientTypeSPA, shared.ClientTypeMobile} {
			assert.NoError(t, ValidateClientOAuthMatrix(clientType, TokenAuthMethodNone, authzCode, nil, false, false), clientType)
		}
	})

	// The exploit: client_id is public, so "no credential" plus this grant means
	// anyone can mint the client's tokens.
	t.Run("rejects client_credentials with no client authentication", func(t *testing.T) {
		err := ValidateClientOAuthMatrix(shared.ClientTypeM2M, TokenAuthMethodNone, clientCreds, scopes, false, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "only valid for public clients")
	})

	t.Run("rejects client_credentials on a public client", func(t *testing.T) {
		err := ValidateClientOAuthMatrix(shared.ClientTypeSPA, TokenAuthMethodNone, clientCreds, scopes, false, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "requires client authentication")
	})

	t.Run("allows client_credentials on an authenticated confidential client", func(t *testing.T) {
		assert.NoError(t, ValidateClientOAuthMatrix(
			shared.ClientTypeM2M, TokenAuthMethodSecretBasic, clientCreds, scopes, true, false))
	})

	// An empty allowlist means "all scopes"; for a machine credential with no user
	// in the loop that is unbounded.
	t.Run("client_credentials requires declared scopes", func(t *testing.T) {
		err := ValidateClientOAuthMatrix(shared.ClientTypeM2M, TokenAuthMethodSecretBasic, clientCreds, nil, true, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must declare allowed_scopes")

		assert.NoError(t, ValidateClientOAuthMatrix(
			shared.ClientTypeM2M, TokenAuthMethodSecretBasic, clientCreds, scopes, true, false))
	})

	t.Run("rejects authorization_code on m2m", func(t *testing.T) {
		err := ValidateClientOAuthMatrix(shared.ClientTypeM2M, TokenAuthMethodSecretBasic, authzCode, nil, true, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no user to authorize")
	})

	// A secret shipped inside browser or mobile code is readable, so a
	// secret-based method there is a false sense of security.
	t.Run("rejects a secret-based method on a public client", func(t *testing.T) {
		for _, method := range []string{TokenAuthMethodSecretBasic, TokenAuthMethodSecretPost, TokenAuthMethodClientSecretJWT} {
			err := ValidateClientOAuthMatrix(shared.ClientTypeSPA, method, authzCode, nil, true, false)
			require.Error(t, err, method)
			assert.Contains(t, err.Error(), "cannot keep a secret")
		}
	})

	t.Run("rejects a secret-based method with no secret", func(t *testing.T) {
		err := ValidateClientOAuthMatrix(shared.ClientTypeTraditional, TokenAuthMethodSecretBasic, authzCode, nil, false, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "requires a client secret")
	})

	// Accepted by the registry allowlist and the CHECK constraint, but the token
	// endpoint has no certificate-binding path — such a client could never
	// authenticate, so it must be refused at write time rather than at first login.
	t.Run("rejects mTLS methods until implemented", func(t *testing.T) {
		for _, method := range []string{TokenAuthMethodTLSClientAuth, TokenAuthMethodSelfSignedTLSClientAuth} {
			err := ValidateClientOAuthMatrix(shared.ClientTypeTraditional, method, authzCode, nil, true, false)
			require.Error(t, err, method)
			assert.Contains(t, err.Error(), "does not implement")
		}
	})

	// private_key_jwt authenticates with the client's own key pair, so no shared
	// secret is needed — but the public keys must be registered or the token
	// endpoint rejects every assertion.
	t.Run("allows private_key_jwt with registered keys and no shared secret", func(t *testing.T) {
		assert.NoError(t, ValidateClientOAuthMatrix(
			shared.ClientTypeTraditional, TokenAuthMethodPrivateKeyJWT, authzCode, nil, false, true))
	})

	t.Run("rejects private_key_jwt without registered keys", func(t *testing.T) {
		err := ValidateClientOAuthMatrix(
			shared.ClientTypeTraditional, TokenAuthMethodPrivateKeyJWT, authzCode, nil, false, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "requires the client's public keys")
	})

	t.Run("checks every grant, not just the first", func(t *testing.T) {
		err := ValidateClientOAuthMatrix(shared.ClientTypeM2M, TokenAuthMethodSecretBasic,
			[]string{GrantTypeClientCredentials, GrantTypeAuthorizationCode}, scopes, true, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no user to authorize")
	})
}

func TestIsPublicClientType(t *testing.T) {
	assert.True(t, IsPublicClientType(shared.ClientTypeSPA))
	assert.True(t, IsPublicClientType(shared.ClientTypeMobile))
	assert.False(t, IsPublicClientType(shared.ClientTypeTraditional))
	assert.False(t, IsPublicClientType(shared.ClientTypeM2M))
	// An unknown or empty type must not be treated as public — that would be the
	// permissive direction on the credential-requirement decision.
	assert.False(t, IsPublicClientType(""))
	assert.False(t, IsPublicClientType("confidential"))
}

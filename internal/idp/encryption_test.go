package idp

import (
	"errors"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setTestEncryptionKey(t *testing.T) {
	t.Helper()
	orig := config.AppEncryptionKey
	config.AppEncryptionKey = []byte("0123456789abcdef0123456789abcdef")
	t.Cleanup(func() { config.AppEncryptionKey = orig })
}

func TestEncryptIdpClientSecret(t *testing.T) {
	setTestEncryptionKey(t)

	t.Run("blank yields nil", func(t *testing.T) {
		result, err := encryptProviderClientSecret("")
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("whitespace yields nil", func(t *testing.T) {
		result, err := encryptProviderClientSecret("   ")
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("non-blank encrypts and round-trips", func(t *testing.T) {
		result, err := encryptProviderClientSecret("my-secret")
		require.NoError(t, err)
		require.NotNil(t, result)
		// Ciphertext is not the plaintext.
		assert.NotEqual(t, "my-secret", *result)
		// And it decrypts back to the original plaintext.
		assert.Equal(t, "my-secret", crypto.SafeDecryptAtRest(*result))
	})

	t.Run("encrypt error propagates", func(t *testing.T) {
		orig := crypto.EncryptAtRest
		defer func() { crypto.EncryptAtRest = orig }()
		crypto.EncryptAtRest = func(string) (string, error) { return "", errors.New("encrypt failure") }

		result, err := encryptProviderClientSecret("secret")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "encrypt failure")
		assert.Nil(t, result)
	})
}

func TestPreserveIdpClientSecret(t *testing.T) {
	setTestEncryptionKey(t)

	existingEnc, err := crypto.EncryptAtRest("OLD")
	require.NoError(t, err)
	existing := &existingEnc

	t.Run("blank preserves existing", func(t *testing.T) {
		result, err := preserveProviderClientSecret("", existing)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, existingEnc, *result)
	})

	t.Run("whitespace preserves existing", func(t *testing.T) {
		result, err := preserveProviderClientSecret("   ", existing)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, existingEnc, *result)
	})

	t.Run("redaction sentinel preserves existing", func(t *testing.T) {
		result, err := preserveProviderClientSecret(providerClientSecretRedacted, existing)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, existingEnc, *result)
	})

	t.Run("new plaintext replaces existing", func(t *testing.T) {
		result, err := preserveProviderClientSecret("NEW", existing)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEqual(t, existingEnc, *result)
		assert.Equal(t, "NEW", crypto.SafeDecryptAtRest(*result))
	})

	t.Run("blank with no existing yields nil", func(t *testing.T) {
		result, err := preserveProviderClientSecret("", nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("encrypt error propagates for new secret", func(t *testing.T) {
		orig := crypto.EncryptAtRest
		defer func() { crypto.EncryptAtRest = orig }()
		crypto.EncryptAtRest = func(string) (string, error) { return "", errors.New("encrypt failure") }

		result, err := preserveProviderClientSecret("NEW", existing)
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestIdentityProvider_DecryptedProviderClientSecret(t *testing.T) {
	setTestEncryptionKey(t)

	t.Run("nil column yields empty", func(t *testing.T) {
		idp := &IdentityProvider{}
		assert.Equal(t, "", idp.DecryptedProviderClientSecret())
	})

	t.Run("empty column yields empty", func(t *testing.T) {
		empty := ""
		idp := &IdentityProvider{ProviderClientSecretEncrypted: &empty}
		assert.Equal(t, "", idp.DecryptedProviderClientSecret())
	})

	t.Run("set column decrypts", func(t *testing.T) {
		enc, err := crypto.EncryptAtRest("topsecret")
		require.NoError(t, err)
		idp := &IdentityProvider{ProviderClientSecretEncrypted: &enc}
		assert.Equal(t, "topsecret", idp.DecryptedProviderClientSecret())
	})
}

func TestIdentityProvider_IssuerAndProviderClientIDOrEmpty(t *testing.T) {
	t.Run("nil pointers yield empty", func(t *testing.T) {
		idp := &IdentityProvider{}
		assert.Equal(t, "", idp.IssuerOrEmpty())
		assert.Equal(t, "", idp.ProviderClientIDOrEmpty())
	})

	t.Run("set pointers yield value", func(t *testing.T) {
		issuer := "https://idp.example.com"
		clientID := "app-1"
		idp := &IdentityProvider{Issuer: &issuer, ProviderClientID: &clientID}
		assert.Equal(t, issuer, idp.IssuerOrEmpty())
		assert.Equal(t, clientID, idp.ProviderClientIDOrEmpty())
	})
}

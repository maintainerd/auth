package idp

import (
	"encoding/json"
	"testing"

	"github.com/maintainerd/auth/internal/platform/config"
	"gorm.io/datatypes"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setTestEncryptionKey(t *testing.T) {
	t.Helper()
	orig := config.AppEncryptionKey
	config.AppEncryptionKey = []byte("0123456789abcdef0123456789abcdef")
	t.Cleanup(func() { config.AppEncryptionKey = orig })
}

func TestEncryptIdpConfig(t *testing.T) {
	setTestEncryptionKey(t)
	t.Run("empty config", func(t *testing.T) {
		result, err := encryptIdpConfig(datatypes.JSON{})
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("config with client_secret", func(t *testing.T) {
		config := datatypes.JSON(json.RawMessage(`{"client_secret":"my-secret","issuer":"https://idp.example.com"}`))
		result, err := encryptIdpConfig(config)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		// The client_secret should be encrypted (not plaintext)
		assert.NotContains(t, string(result), "my-secret")
	})

	t.Run("config without client_secret", func(t *testing.T) {
		config := datatypes.JSON(json.RawMessage(`{"client_id":"app-1","issuer":"https://idp.example.com"}`))
		result, err := encryptIdpConfig(config)
		require.NoError(t, err)
		// Should pass through unchanged
		assert.Equal(t, config, result)
	})

	t.Run("invalid JSON returns original", func(t *testing.T) {
		config := datatypes.JSON(json.RawMessage(`not-json`))
		result, err := encryptIdpConfig(config)
		require.Error(t, err)
		assert.Equal(t, config, result)
	})

	t.Run("nil config", func(t *testing.T) {
		result, err := encryptIdpConfig(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestDecryptIdpConfig(t *testing.T) {
	setTestEncryptionKey(t)
	t.Run("empty config", func(t *testing.T) {
		result := decryptIdpConfig(datatypes.JSON{})
		assert.Empty(t, result)
	})

	t.Run("config without client_secret", func(t *testing.T) {
		config := datatypes.JSON(json.RawMessage(`{"client_id":"app-1"}`))
		result := decryptIdpConfig(config)
		assert.Equal(t, config, result)
	})

	t.Run("invalid JSON returns original", func(t *testing.T) {
		config := datatypes.JSON(json.RawMessage(`not-json`))
		result := decryptIdpConfig(config)
		assert.Equal(t, config, result)
	})

	t.Run("nil config returns nil", func(t *testing.T) {
		result := decryptIdpConfig(nil)
		assert.Nil(t, result)
	})

	t.Run("config with client_secret decrypts", func(t *testing.T) {
		// First encrypt, then decrypt
		original := datatypes.JSON(json.RawMessage(`{"client_secret":"my-secret","issuer":"https://idp.example.com"}`))
		encrypted, err := encryptIdpConfig(original)
		require.NoError(t, err)

		result := decryptIdpConfig(encrypted)
		// After encrypt+decrypt, we should get back the original value
		assert.Equal(t, string(original), string(result))
	})
}

func TestRedactIdpConfig(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		result := redactIdpConfig(datatypes.JSON{})
		require.NotNil(t, result)
		assert.Empty(t, *result)
	})

	t.Run("config with client_secret", func(t *testing.T) {
		config := datatypes.JSON(json.RawMessage(`{"client_secret":"secret123","issuer":"https://idp.example.com"}`))
		result := redactIdpConfig(config)
		require.NotNil(t, result)
		assert.Contains(t, string(*result), "***REDACTED***")
		assert.NotContains(t, string(*result), "secret123")
	})

	t.Run("config without client_secret", func(t *testing.T) {
		config := datatypes.JSON(json.RawMessage(`{"client_id":"app-1"}`))
		result := redactIdpConfig(config)
		require.NotNil(t, result)
		assert.Equal(t, config, *result)
	})

	t.Run("invalid JSON returns original", func(t *testing.T) {
		config := datatypes.JSON(json.RawMessage(`not-json`))
		result := redactIdpConfig(config)
		require.NotNil(t, result)
		assert.Equal(t, config, *result)
	})
}

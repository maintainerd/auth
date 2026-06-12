package idp

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/crypto"
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

	t.Run("nil config", func(t *testing.T) {
		result := redactIdpConfig(nil)
		require.NotNil(t, result)
		assert.Nil(t, *result)
	})
}

func TestEncryptIdpConfig_NonStringClientSecret(t *testing.T) {
	setTestEncryptionKey(t)

	t.Run("null client_secret passes through", func(t *testing.T) {
		config := datatypes.JSON(json.RawMessage(`{"client_secret":null,"issuer":"https://idp.example.com"}`))
		result, err := encryptIdpConfig(config)
		require.NoError(t, err)
		assert.Equal(t, config, result)
	})

	t.Run("numeric client_secret passes through", func(t *testing.T) {
		config := datatypes.JSON(json.RawMessage(`{"client_secret":12345,"issuer":"https://idp.example.com"}`))
		result, err := encryptIdpConfig(config)
		require.NoError(t, err)
		assert.Equal(t, config, result)
	})

	t.Run("empty client_secret passes through", func(t *testing.T) {
		setTestEncryptionKey(t)
		config := datatypes.JSON(json.RawMessage(`{"client_secret":"","issuer":"https://idp.example.com"}`))
		result, err := encryptIdpConfig(config)
		require.NoError(t, err)
		assert.Equal(t, config, result)
	})

	t.Run("encrypt error propagates", func(t *testing.T) {
		orig := crypto.EncryptAtRest
		defer func() { crypto.EncryptAtRest = orig }()
		crypto.EncryptAtRest = func(string) (string, error) { return "", errors.New("encrypt failure") }

		config := datatypes.JSON(json.RawMessage(`{"client_secret":"secret","issuer":"https://idp.example.com"}`))
		result, err := encryptIdpConfig(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "encrypt failure")
		assert.Nil(t, result)
	})
}

func TestDecryptIdpConfig_NonStringClientSecret(t *testing.T) {
	setTestEncryptionKey(t)

	t.Run("null client_secret passes through", func(t *testing.T) {
		config := datatypes.JSON(json.RawMessage(`{"client_secret":null,"issuer":"https://idp.example.com"}`))
		result := decryptIdpConfig(config)
		assert.Equal(t, config, result)
	})

	t.Run("numeric client_secret passes through", func(t *testing.T) {
		config := datatypes.JSON(json.RawMessage(`{"client_secret":12345,"issuer":"https://idp.example.com"}`))
		result := decryptIdpConfig(config)
		assert.Equal(t, config, result)
	})

	t.Run("empty client_secret passes through", func(t *testing.T) {
		config := datatypes.JSON(json.RawMessage(`{"client_secret":"","issuer":"https://idp.example.com"}`))
		result := decryptIdpConfig(config)
		assert.Equal(t, config, result)
	})
}

func TestRedactIdpConfig_NonStringClientSecret(t *testing.T) {
	t.Run("null client_secret redacts", func(t *testing.T) {
		config := datatypes.JSON(json.RawMessage(`{"client_secret":null,"issuer":"https://idp.example.com"}`))
		result := redactIdpConfig(config)
		require.NotNil(t, result)
		assert.Contains(t, string(*result), "***REDACTED***")
	})

	t.Run("numeric client_secret redacts", func(t *testing.T) {
		config := datatypes.JSON(json.RawMessage(`{"client_secret":12345,"issuer":"https://idp.example.com"}`))
		result := redactIdpConfig(config)
		require.NotNil(t, result)
		assert.Contains(t, string(*result), "***REDACTED***")
	})
}

func TestPreserveIdpClientSecret(t *testing.T) {
	existing := datatypes.JSON(json.RawMessage(`{"client_id":"app","client_secret":"ENC_OLD","issuer":"https://idp.example.com"}`))

	t.Run("blank secret preserves existing", func(t *testing.T) {
		incoming := datatypes.JSON(json.RawMessage(`{"client_id":"app","client_secret":"","issuer":"https://idp.example.com"}`))
		result := preserveIdpClientSecret(incoming, existing)
		assert.Contains(t, string(result), `"client_secret":"ENC_OLD"`)
	})

	t.Run("redacted sentinel preserves existing", func(t *testing.T) {
		incoming := datatypes.JSON(json.RawMessage(`{"client_id":"app","client_secret":"` + idpClientSecretRedacted + `"}`))
		result := preserveIdpClientSecret(incoming, existing)
		assert.Contains(t, string(result), `"client_secret":"ENC_OLD"`)
	})

	t.Run("missing key preserves existing", func(t *testing.T) {
		incoming := datatypes.JSON(json.RawMessage(`{"client_id":"app"}`))
		result := preserveIdpClientSecret(incoming, existing)
		assert.Contains(t, string(result), `"client_secret":"ENC_OLD"`)
	})

	t.Run("new secret is kept", func(t *testing.T) {
		incoming := datatypes.JSON(json.RawMessage(`{"client_id":"app","client_secret":"ENC_NEW"}`))
		result := preserveIdpClientSecret(incoming, existing)
		assert.Contains(t, string(result), `"client_secret":"ENC_NEW"`)
		assert.NotContains(t, string(result), "ENC_OLD")
	})

	t.Run("blank with no existing drops the key", func(t *testing.T) {
		incoming := datatypes.JSON(json.RawMessage(`{"client_id":"app","client_secret":""}`))
		result := preserveIdpClientSecret(incoming, datatypes.JSON(json.RawMessage(`{"client_id":"app"}`)))
		assert.NotContains(t, string(result), "client_secret")
	})
}

func TestRedactIdpConfig_EmptyStringNotRedacted(t *testing.T) {
	config := datatypes.JSON(json.RawMessage(`{"client_secret":"","issuer":"https://idp.example.com"}`))
	result := redactIdpConfig(config)
	require.NotNil(t, result)
	assert.NotContains(t, string(*result), "***REDACTED***")
}

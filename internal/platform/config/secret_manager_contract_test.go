package config

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Swapping SECRET_PROVIDER is meant to be transparent to the application, so
// value handling must not vary by provider. Two behaviours used to:
// `base64:` decoding existed only in the env provider, and trailing whitespace
// was stripped by GetSecretString but not by GetSecret — the one that returns
// the raw bytes used as key material.
func TestNormalizeSecret(t *testing.T) {
	t.Run("decodes a base64 payload regardless of provider", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString([]byte("hello world"))
		out, err := normalizeSecret("K", []byte("base64:"+encoded))
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(out))
	})

	t.Run("rejects a malformed base64 payload", func(t *testing.T) {
		_, err := normalizeSecret("K", []byte("base64:!!!invalid!!!"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode base64")
	})

	// `echo value > secretfile` and several secret stores append a newline. The
	// raw path used to keep it, so the encryption key the app used was a byte
	// longer than the one the operator stored — silently, with no error.
	t.Run("strips trailing newlines that secret files carry", func(t *testing.T) {
		for _, raw := range []string{"s3cr3t\n", "s3cr3t\r\n", "  s3cr3t  ", "s3cr3t"} {
			out, err := normalizeSecret("K", []byte(raw))
			require.NoError(t, err)
			assert.Equal(t, "s3cr3t", string(out), "input %q", raw)
		}
	})

	t.Run("a base64 payload with a trailing newline still decodes", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString([]byte("hello"))
		out, err := normalizeSecret("K", []byte("base64:"+encoded+"\n"))
		require.NoError(t, err)
		assert.Equal(t, "hello", string(out))
	})

	t.Run("binary key material is preserved byte for byte", func(t *testing.T) {
		key := []byte{0x00, 0x01, 0xFE, 0xFF, 0x10}
		encoded := base64.StdEncoding.EncodeToString(key)
		out, err := normalizeSecret("K", []byte("base64:"+encoded))
		require.NoError(t, err)
		assert.Equal(t, key, out, "base64 is how binary keys survive a text-only store")
	})
}

// Every secret in the system crosses this connection, including the JWT signing
// key. The Vault default address is plaintext for local dev, so an operator who
// sets SECRET_PROVIDER=vault and forgets VAULT_ADDR would ship secrets in the
// clear with no warning.
func TestRequireSecureSecretTransport(t *testing.T) {
	t.Run("https is always accepted", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		assert.NoError(t, requireSecureSecretTransport("VAULT_ADDR", "https://vault.internal:8200"))
	})

	t.Run("plaintext to a remote host is refused in production", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		err := requireSecureSecretTransport("VAULT_ADDR", "http://vault.internal:8200")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must use https in production")
	})

	t.Run("plaintext to a remote host only warns outside production", func(t *testing.T) {
		t.Setenv("APP_ENV", "development")
		assert.NoError(t, requireSecureSecretTransport("VAULT_ADDR", "http://vault.internal:8200"))
	})

	// A local dev Vault must keep working.
	t.Run("loopback is permitted even in production", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		for _, addr := range []string{"http://localhost:8200", "http://127.0.0.1:8200", "http://[::1]:8200"} {
			assert.NoError(t, requireSecureSecretTransport("VAULT_ADDR", addr), "addr %s", addr)
		}
	})

	t.Run("an unparseable address is rejected", func(t *testing.T) {
		require.Error(t, requireSecureSecretTransport("VAULT_ADDR", "://nonsense"))
	})
}

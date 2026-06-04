package crypto

import (
	"errors"
	"testing"

	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptBytes_Roundtrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	t.Run("roundtrip short", func(t *testing.T) {
		plaintext := []byte("hello world")
		ct, err := EncryptBytes(plaintext, key)
		require.NoError(t, err)
		pt, err := DecryptBytes(ct, key)
		require.NoError(t, err)
		assert.Equal(t, plaintext, pt)
	})

	t.Run("roundtrip empty", func(t *testing.T) {
		ct, err := EncryptBytes([]byte{}, key)
		require.NoError(t, err)
		pt, err := DecryptBytes(ct, key)
		require.NoError(t, err)
		assert.Empty(t, pt)
	})

	t.Run("roundtrip 4KB", func(t *testing.T) {
		plaintext := make([]byte, 4096)
		for i := range plaintext {
			plaintext[i] = byte(i % 256)
		}
		ct, err := EncryptBytes(plaintext, key)
		require.NoError(t, err)
		pt, err := DecryptBytes(ct, key)
		require.NoError(t, err)
		assert.Equal(t, plaintext, pt)
	})

	t.Run("different ciphertext for same plaintext", func(t *testing.T) {
		ct1, err := EncryptBytes([]byte("test"), key)
		require.NoError(t, err)
		ct2, err := EncryptBytes([]byte("test"), key)
		require.NoError(t, err)
		assert.NotEqual(t, ct1, ct2)
	})

	t.Run("wrong key fails", func(t *testing.T) {
		ct, err := EncryptBytes([]byte("secret"), key)
		require.NoError(t, err)
		wrongKey := make([]byte, 32)
		wrongKey[0] = 1
		_, err = DecryptBytes(ct, wrongKey)
		require.Error(t, err)
	})

	t.Run("tampered ciphertext fails", func(t *testing.T) {
		ct, err := EncryptBytes([]byte("secret"), key)
		require.NoError(t, err)
		ct[GCMNonceSize+1] ^= 0x01
		_, err = DecryptBytes(ct, key)
		require.Error(t, err)
	})

	t.Run("key too short", func(t *testing.T) {
		_, err := EncryptBytes([]byte("x"), make([]byte, 16))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "32 bytes")
	})

	t.Run("ciphertext too short", func(t *testing.T) {
		_, err := DecryptBytes([]byte{1, 2, 3}, key)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too short")
	})
}

func TestEncryptDecryptString_Roundtrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	t.Run("roundtrip", func(t *testing.T) {
		ct, err := EncryptString("hello world", key)
		require.NoError(t, err)
		pt, err := DecryptString(ct, key)
		require.NoError(t, err)
		assert.Equal(t, "hello world", pt)
	})

	t.Run("roundtrip empty", func(t *testing.T) {
		ct, err := EncryptString("", key)
		require.NoError(t, err)
		assert.Empty(t, ct)
		pt, err := DecryptString(ct, key)
		require.NoError(t, err)
		assert.Empty(t, pt)
	})

	t.Run("invalid base64", func(t *testing.T) {
		_, err := DecryptString("not!valid!base64", key)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid base64")
	})
}

func TestEncryptAtRest_DecryptAtRest(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	origEncrypt := EncryptAtRest
	origDecrypt := DecryptAtRest
	defer func() {
		EncryptAtRest = origEncrypt
		DecryptAtRest = origDecrypt
	}()

	EncryptAtRest = func(plaintext string) (string, error) { return EncryptString(plaintext, key) }
	DecryptAtRest = func(ciphertext string) (string, error) { return DecryptString(ciphertext, key) }

	t.Run("roundtrip", func(t *testing.T) {
		ct, err := EncryptAtRest("my secret value")
		require.NoError(t, err)
		pt, err := DecryptAtRest(ct)
		require.NoError(t, err)
		assert.Equal(t, "my secret value", pt)
	})

	t.Run("empty in = empty out", func(t *testing.T) {
		ct, err := EncryptAtRest("")
		require.NoError(t, err)
		assert.Empty(t, ct)
		pt, err := DecryptAtRest("")
		require.NoError(t, err)
		assert.Empty(t, pt)
	})
}

func TestSafeDecryptAtRest(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	origEncrypt := EncryptAtRest
	origDecrypt := DecryptAtRest
	defer func() {
		EncryptAtRest = origEncrypt
		DecryptAtRest = origDecrypt
	}()

	EncryptAtRest = func(plaintext string) (string, error) { return EncryptString(plaintext, key) }

	t.Run("decrypts valid ciphertext", func(t *testing.T) {
		ct, err := EncryptAtRest("sensitive data")
		require.NoError(t, err)

		DecryptAtRest = func(ciphertext string) (string, error) { return DecryptString(ciphertext, key) }
		result := SafeDecryptAtRest(ct)
		assert.Equal(t, "sensitive data", result)
	})

	t.Run("returns empty for empty input", func(t *testing.T) {
		assert.Equal(t, "", SafeDecryptAtRest(""))
	})

	t.Run("returns original value on decrypt failure", func(t *testing.T) {
		DecryptAtRest = func(ciphertext string) (string, error) {
			return "", errors.New("decrypt error")
		}
		result := SafeDecryptAtRest("corrupted-data")
		assert.Equal(t, "corrupted-data", result)
	})
}

func TestAppEncryptionKey(t *testing.T) {
	origKey := config.AppEncryptionKey
	t.Cleanup(func() { config.AppEncryptionKey = origKey })

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	config.AppEncryptionKey = key

	assert.Equal(t, key, appEncryptionKey())
}

func TestEncryptBytes_ErrorPaths(t *testing.T) {
	t.Run("key too short on encrypt", func(t *testing.T) {
		_, err := encryptBytes([]byte("test"), make([]byte, 16))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "32 bytes")
	})
}

func TestDecryptBytes_ErrorPaths(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	t.Run("key too short on decrypt", func(t *testing.T) {
		ct, err := encryptBytes([]byte("test"), key)
		require.NoError(t, err)
		_, err = decryptBytes(ct, make([]byte, 16))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "32 bytes")
	})
}

func TestEncryptString_ErrorPropagation(t *testing.T) {
	origEncrypt := EncryptBytes
	t.Cleanup(func() { EncryptBytes = origEncrypt })

	EncryptBytes = func(plaintext, key []byte) ([]byte, error) {
		return nil, errors.New("encrypt failure")
	}

	_, err := EncryptString("test", make([]byte, 32))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "encrypt failure")
}

func TestDecryptString_ErrorPropagation(t *testing.T) {
	origDecrypt := DecryptBytes
	t.Cleanup(func() { DecryptBytes = origDecrypt })

	DecryptBytes = func(ciphertext, key []byte) ([]byte, error) {
		return nil, errors.New("decrypt failure")
	}

	_, err := DecryptString("dGVzdA", make([]byte, 32))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decrypt failure")
}

func TestEncryptBytes_FailingRand(t *testing.T) {
	withFailingRand(t)
	_, err := encryptBytes([]byte("test"), make([]byte, 32))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonce")
}

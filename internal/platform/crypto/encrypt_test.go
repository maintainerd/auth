package crypto

import (
	"testing"

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

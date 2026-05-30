package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/maintainerd/auth/internal/platform/config"
)

const GCMNonceSize = 12

var EncryptBytes = encryptBytes

func encryptBytes(plaintext, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes (AES-256)")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	nonce := make([]byte, GCMNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	result := gcm.Seal(nonce, nonce, plaintext, nil)
	return result, nil
}

var DecryptBytes = decryptBytes

func decryptBytes(ciphertext, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes (AES-256)")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	if len(ciphertext) < GCMNonceSize {
		return nil, fmt.Errorf("ciphertext too short (%d bytes)", len(ciphertext))
	}
	nonce, ciphertextOnly := ciphertext[:GCMNonceSize], ciphertext[GCMNonceSize:]
	result, err := gcm.Open(nil, nonce, ciphertextOnly, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return result, nil
}

func EncryptString(plaintext string, key []byte) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	ct, err := EncryptBytes([]byte(plaintext), key)
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(ct), nil
}

func DecryptString(ciphertext string, key []byte) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	raw, err := base64.RawStdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decrypt: invalid base64: %w", err)
	}
	pt, err := DecryptBytes(raw, key)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

func appEncryptionKey() []byte {
	return config.AppEncryptionKey
}

var EncryptAtRest = func(plaintext string) (string, error) {
	return EncryptString(plaintext, appEncryptionKey())
}

var DecryptAtRest = func(ciphertext string) (string, error) {
	return DecryptString(ciphertext, appEncryptionKey())
}

func SafeDecryptAtRest(value string) string {
	if value == "" {
		return ""
	}
	dec, err := DecryptAtRest(value)
	if err != nil {
		return value
	}
	return dec
}

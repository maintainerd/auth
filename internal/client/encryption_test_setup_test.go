package client

import (
	"os"
	"strings"
	"testing"

	"github.com/maintainerd/auth/internal/platform/crypto"
)

func TestMain(m *testing.M) {
	origEncrypt := crypto.EncryptAtRest
	origDecrypt := crypto.DecryptAtRest
	origEncryptClientSecret := encryptClientSecret
	crypto.EncryptAtRest = func(plaintext string) (string, error) {
		if plaintext == "" {
			return "", nil
		}
		return "test-enc:" + plaintext, nil
	}
	encryptClientSecret = crypto.EncryptAtRest
	crypto.DecryptAtRest = func(ciphertext string) (string, error) {
		return strings.TrimPrefix(ciphertext, "test-enc:"), nil
	}
	code := m.Run()
	crypto.EncryptAtRest = origEncrypt
	crypto.DecryptAtRest = origDecrypt
	encryptClientSecret = origEncryptClientSecret
	os.Exit(code)
}

package oauth

import (
	"os"
	"strings"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
)

func TestMain(m *testing.M) {
	// APP_PUBLIC_HOSTNAME is a required deployment variable and is now the issuer
	// on every token this server mints, the fallback the issuer allowlist accepts
	// when it is unconfigured, and the audience a client assertion must name. With
	// it empty, jwt validation rejects every self-issued token, so the package's
	// tests need it set exactly as a real process would have it.
	origHost := config.AppPublicHostname
	config.AppPublicHostname = "https://auth.example.com"

	origEncrypt := crypto.EncryptAtRest
	origDecrypt := crypto.DecryptAtRest
	crypto.EncryptAtRest = func(plaintext string) (string, error) {
		if plaintext == "" {
			return "", nil
		}
		return "test-enc:" + plaintext, nil
	}
	crypto.DecryptAtRest = func(ciphertext string) (string, error) {
		return strings.TrimPrefix(ciphertext, "test-enc:"), nil
	}
	code := m.Run()
	crypto.EncryptAtRest = origEncrypt
	crypto.DecryptAtRest = origDecrypt
	config.AppPublicHostname = origHost
	os.Exit(code)
}

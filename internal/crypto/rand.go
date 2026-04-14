package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
)

// GenerateIdentifier returns a cryptographically random alphanumeric identifier
// of length n. Returns an error if the system's random source fails.
// Assigned to a var so that tests can swap the implementation.
var GenerateIdentifier = generateIdentifier

func generateIdentifier(n int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("crypto/rand failure: %w", err)
		}
		b[i] = charset[num.Int64()]
	}
	return string(b), nil
}

// GenerateRandomString returns a URL-safe base64-encoded string derived from
// n random bytes. The resulting string length is approximately 4*n/3.
func GenerateRandomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto/rand failure: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

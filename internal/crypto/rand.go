package crypto

import (
	"crypto/rand"
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

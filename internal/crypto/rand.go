package crypto

import (
	"crypto/rand"
	"math/big"
)

// GenerateIdentifier returns a random alphanumeric identifier string (no special characters)
func GenerateIdentifier(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			panic(err) // or return empty string + error
		}
		b[i] = charset[num.Int64()]
	}
	return string(b)
}

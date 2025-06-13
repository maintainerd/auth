package util

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
)

// GenerateIdentifier returns a random URL-safe identifier string
func GenerateIdentifier(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err) // or return empty string + error
	}
	return strings.TrimRight(base64.URLEncoding.EncodeToString(b), "=")
}

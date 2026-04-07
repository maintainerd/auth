package crypto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const identifierCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func TestGenerateIdentifier(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"length 1", 1},
		{"length 8", 8},
		{"length 16", 16},
		{"length 32", 32},
		{"length 64", 64},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GenerateIdentifier(tc.length)
			assert.Len(t, got, tc.length)
			for _, ch := range got {
				assert.True(t, strings.ContainsRune(identifierCharset, ch),
					"character %q is not in allowed charset", ch)
			}
		})
	}
}

func TestGenerateIdentifierUniqueness(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		id := GenerateIdentifier(16)
		_, dup := seen[id]
		assert.False(t, dup, "duplicate identifier generated: %s", id)
		seen[id] = struct{}{}
	}
}

func TestGenerateIdentifierZeroLength(t *testing.T) {
	// Zero length should return empty string, not panic
	got := GenerateIdentifier(0)
	assert.Equal(t, "", got)
}

func TestGenerateIdentifier_CryptoRandError(t *testing.T) {
	withFailingRand(t)
	assert.Panics(t, func() { GenerateIdentifier(8) })
}

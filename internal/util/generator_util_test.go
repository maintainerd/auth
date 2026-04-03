package util

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestGenerateOTP(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{"length 4", 4, false},
		{"length 6 (standard OTP)", 6, false},
		{"length 8", 8, false},
		{"zero length returns error", 0, true},
		{"negative length returns error", -1, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := GenerateOTP(tc.length)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Empty(t, got)
				return
			}
			require.NoError(t, err)
			assert.Len(t, got, tc.length)
			for _, ch := range got {
				assert.True(t, ch >= '0' && ch <= '9',
					"character %q is not a digit", ch)
			}
		})
	}
}

func TestGenerateOTPUniqueness(t *testing.T) {
	seen := make(map[string]struct{}, 50)
	collisions := 0
	for i := 0; i < 50; i++ {
		otp, err := GenerateOTP(6)
		require.NoError(t, err)
		if _, dup := seen[otp]; dup {
			collisions++
		}
		seen[otp] = struct{}{}
	}
	// With 10^6 space and 50 samples, collisions are extremely rare
	assert.Less(t, collisions, 3, "too many OTP collisions - RNG may be broken")
}

func TestGenerateOTPOnlyDigits(t *testing.T) {
	for i := 0; i < 20; i++ {
		otp, err := GenerateOTP(6)
		require.NoError(t, err)
		for _, ch := range otp {
			assert.True(t, ch >= '0' && ch <= '9')
		}
	}
}


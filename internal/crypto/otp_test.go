package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestGenerateOTP_CryptoRandError(t *testing.T) {
	withFailingRand(t)
	_, err := GenerateOTP(6)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forced crypto/rand failure")
}

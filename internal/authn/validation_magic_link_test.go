package authn

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendMagicLinkRequestDTO_Validate(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr string
	}{
		{"valid", "user@example.com", ""},
		{"missing email", "", "Email is required"},
		{"invalid email format", "not-an-email", "Email must be a valid email address"},
		{"email too long", strings.Repeat("a", 250) + "@example.com", "email"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dto := &SendMagicLinkRequestDTO{Email: tc.email}
			err := dto.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}

func TestVerifyMagicLinkRequestDTO_Validate(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr string
	}{
		{"valid", "abcdef1234567890", ""},
		{"missing token", "", "Token is required"},
		{"token too short", "abc123", "Token has an invalid length"},
		{"token too long", strings.Repeat("x", 257), "Token has an invalid length"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dto := &VerifyMagicLinkRequestDTO{Token: tc.token}
			err := dto.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}

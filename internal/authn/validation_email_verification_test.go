package authn

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendEmailVerificationRequestDTO_Validate(t *testing.T) {
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
			dto := &SendEmailVerificationRequestDTO{Email: tc.email}
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

func TestVerifyEmailRequestDTO_Validate(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		otp     string
		wantErr string
	}{
		{"valid", "user@example.com", "123456", ""},
		{"missing email", "", "123456", "Email is required"},
		{"invalid email", "bad", "123456", "Email must be a valid email address"},
		{"missing OTP", "user@example.com", "", "Verification code is required"},
		{"OTP too short", "user@example.com", "123", "Verification code must be between 4 and 12 characters"},
		{"OTP too long", "user@example.com", "1234567890123", "Verification code must be between 4 and 12 characters"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dto := &VerifyEmailRequestDTO{Email: tc.email, OTP: tc.otp}
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

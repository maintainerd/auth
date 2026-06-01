package user

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangeEmailRequestDTO_Validate(t *testing.T) {
	tests := []struct {
		name            string
		newEmail        string
		currentPassword string
		wantErr         string
	}{
		{"valid", "new@example.com", "password123", ""},
		{"missing email", "", "pass", "new_email"},
		{"invalid email", "not-an-email", "pass", "new_email"},
		{"missing password", "new@example.com", "", "current_password"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dto := &ChangeEmailRequestDTO{NewEmail: tc.newEmail, CurrentPassword: tc.currentPassword}
			err := dto.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, strings.ToLower(err.Error()), tc.wantErr)
			}
		})
	}
}

func TestVerifyEmailChangeDTO_Validate(t *testing.T) {
	tests := []struct {
		name    string
		otp     string
		wantErr string
	}{
		{"valid", "123456", ""},
		{"missing otp", "", "otp"},
		{"otp too short", "12345", "otp"},
		{"otp too long", "1234567", "otp"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dto := &VerifyEmailChangeDTO{OTP: tc.otp}
			err := dto.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, strings.ToLower(err.Error()), tc.wantErr)
			}
		})
	}
}

func TestChangeUsernameDTO_Validate(t *testing.T) {
	tests := []struct {
		name            string
		newUsername     string
		currentPassword string
		wantErr         string
	}{
		{"valid", "newuser", "pass", ""},
		{"missing username", "", "pass", "new_username"},
		{"username too short", "ab", "pass", "new_username"},
		{"username too long", strings.Repeat("x", 51), "pass", "new_username"},
		{"missing password", "newuser", "", "current_password"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dto := &ChangeUsernameDTO{NewUsername: tc.newUsername, CurrentPassword: tc.currentPassword}
			err := dto.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, strings.ToLower(err.Error()), tc.wantErr)
			}
		})
	}
}

func TestAccountDeleteDTO_Validate(t *testing.T) {
	tests := []struct {
		name            string
		currentPassword string
		wantErr         string
	}{
		{"valid", "pass", ""},
		{"missing password", "", "current_password"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dto := &AccountDeleteDTO{CurrentPassword: tc.currentPassword}
			err := dto.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, strings.ToLower(err.Error()), tc.wantErr)
			}
		})
	}
}

func TestVerifyBackupCodeDTO_Validate(t *testing.T) {
	tests := []struct {
		name       string
		email      string
		code       string
		clientID   string
		providerID string
		wantErr    string
	}{
		{"valid", "user@example.com", "code123", "app", "idp", ""},
		{"missing email", "", "c", "app", "idp", "email"},
		{"invalid email", "bad", "c", "app", "idp", "email"},
		{"missing code", "user@example.com", "", "app", "idp", "code"},
		{"missing client_id", "user@example.com", "c", "", "idp", "client_id"},
		{"missing provider_id", "user@example.com", "c", "app", "", "provider_id"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dto := &VerifyBackupCodeDTO{Email: tc.email, Code: tc.code, ClientID: tc.clientID, ProviderID: tc.providerID}
			err := dto.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, strings.ToLower(err.Error()), tc.wantErr)
			}
		})
	}
}

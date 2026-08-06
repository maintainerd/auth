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

func TestSendPhoneVerificationDTO_Validate(t *testing.T) {
	tests := []struct {
		name    string
		phone   string
		wantErr string
	}{
		{"valid", "+15550001111", ""},
		{"missing phone", "", "phone"},
		{"invalid phone format", "abc", "phone"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dto := &SendPhoneVerificationDTO{Phone: tc.phone}
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

func TestVerifyPhoneDTO_Validate(t *testing.T) {
	tests := []struct {
		name    string
		phone   string
		code    string
		wantErr string
	}{
		{"valid", "+15550001111", "123456", ""},
		{"missing phone", "", "123456", "phone"},
		{"invalid phone format", "abc", "123456", "phone"},
		{"missing code", "+15550001111", "", "code"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dto := &VerifyPhoneDTO{Phone: tc.phone, Code: tc.code}
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
		password   string
		code       string
		clientID   string
		providerID string
		wantErr    string
	}{
		{"valid", "user@example.com", "pw", "code123", "app", "idp", ""},
		{"missing email", "", "pw", "c", "app", "idp", "email"},
		{"invalid email", "bad", "pw", "c", "app", "idp", "email"},
		{
			// A backup code is a recovery SECOND factor. Without a first factor
			// this endpoint mints a full token set from an email address and one
			// short code, straight past the tenant's enforced-MFA policy.
			name: "missing password", email: "user@example.com", password: "",
			code: "c", clientID: "app", providerID: "idp", wantErr: "password",
		},
		{"missing code", "user@example.com", "pw", "", "app", "idp", "code"},
		{"missing client_id", "user@example.com", "pw", "c", "", "idp", "client_id"},
		{"missing provider_id", "user@example.com", "pw", "c", "app", "", "provider_id"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dto := &VerifyBackupCodeDTO{Email: tc.email, Password: tc.password, Code: tc.code, ClientID: tc.clientID, ProviderID: tc.providerID}
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

func TestChangeUsernameDTO_Validate_RejectsEmailShapedUsername(t *testing.T) {
	// authn.findLoginUser resolves the username column FIRST, so renaming to
	// another user's email address hijacks their email login — and the
	// uniqueness pre-check queries usernames alone, so nothing collides.
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{"email-shaped username rejected", "alice@corp.com", true},
		{"bare @ rejected", "al@ice", true},
		{"space rejected", "alice smith", true},
		{"plain username allowed", "alicesmith", false},
		{"dot underscore hyphen allowed", "alice.smith_1-x", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dto := &ChangeUsernameDTO{NewUsername: tc.username, CurrentPassword: "pw"}
			err := dto.Validate()
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, strings.ToLower(err.Error()), "username")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

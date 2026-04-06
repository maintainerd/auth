package dto

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maintainerd/auth/internal/signedurl"
)

func TestRegisterRequestDto_Validate(t *testing.T) {
	tests := []struct {
		name    string
		dto     RegisterRequestDto
		wantErr bool
	}{
		{
			name:    "valid minimal",
			dto:     RegisterRequestDto{Username: "johndoe", Fullname: "John Doe", Password: "SecurePass1!"},
			wantErr: false,
		},
		{
			name:    "valid with email",
			dto:     RegisterRequestDto{Username: "johndoe", Fullname: "John Doe", Password: "SecurePass1!", Email: strPtr("john@example.com")},
			wantErr: false,
		},
		{
			name:    "missing username",
			dto:     RegisterRequestDto{Fullname: "John Doe", Password: "SecurePass1!"},
			wantErr: true,
		},
		{
			name:    "missing fullname",
			dto:     RegisterRequestDto{Username: "johndoe", Password: "SecurePass1!"},
			wantErr: true,
		},
		{
			name:    "missing password",
			dto:     RegisterRequestDto{Username: "johndoe", Fullname: "John Doe"},
			wantErr: true,
		},
		{
			name:    "password too short",
			dto:     RegisterRequestDto{Username: "johndoe", Fullname: "John Doe", Password: "short"},
			wantErr: true,
		},
		{
			name:    "invalid email",
			dto:     RegisterRequestDto{Username: "johndoe", Fullname: "John Doe", Password: "SecurePass1!", Email: strPtr("not-an-email")},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := tc.dto
			err := d.Validate()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRegisterQueryDto_Validate(t *testing.T) {
	tests := []struct {
		name    string
		dto     RegisterQueryDto
		wantErr bool
	}{
		{name: "valid", dto: RegisterQueryDto{ClientID: "c1", ProviderID: "p1"}, wantErr: false},
		{name: "missing client_id", dto: RegisterQueryDto{ProviderID: "p1"}, wantErr: true},
		{name: "missing provider_id", dto: RegisterQueryDto{ClientID: "c1"}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := tc.dto
			err := d.Validate()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRegisterInviteQueryDto_Validate(t *testing.T) {
	valid := RegisterInviteQueryDto{
		ClientID:    "c1",
		ProviderID:  "p1",
		InviteToken: "token123",
		Expires:     "1700000000",
		Sig:         "abc123sig",
	}

	t.Run("valid", func(t *testing.T) {
		d := valid
		assert.NoError(t, d.Validate())
	})

	t.Run("missing client_id", func(t *testing.T) {
		d := valid
		d.ClientID = ""
		require.Error(t, d.Validate())
	})

	t.Run("missing invite_token", func(t *testing.T) {
		d := valid
		d.InviteToken = ""
		require.Error(t, d.Validate())
	})

	t.Run("missing sig", func(t *testing.T) {
		d := valid
		d.Sig = ""
		require.Error(t, d.Validate())
	})
}

func TestRegisterRequestDto_Validate_WithPhone(t *testing.T) {
	t.Run("valid phone", func(t *testing.T) {
		d := RegisterRequestDto{Username: "johndoe", Fullname: "John Doe", Password: "SecurePass1!", Phone: strPtr("+12125551234")}
		assert.NoError(t, d.Validate())
	})

	t.Run("invalid phone", func(t *testing.T) {
		d := RegisterRequestDto{Username: "johndoe", Fullname: "John Doe", Password: "SecurePass1!", Phone: strPtr("not-a-phone")}
		require.Error(t, d.Validate())
	})
}

func TestRegisterRequestDto_ValidateForRegistration(t *testing.T) {
	t.Run("valid passes strength check", func(t *testing.T) {
		d := RegisterRequestDto{Username: "johndoe", Fullname: "John Doe", Password: "SecurePass1!"}
		assert.NoError(t, d.ValidateForRegistration())
	})

	t.Run("base validation error propagates", func(t *testing.T) {
		d := RegisterRequestDto{Username: "", Fullname: "John Doe", Password: "SecurePass1!"}
		require.Error(t, d.ValidateForRegistration())
	})

	t.Run("weak password fails strength check", func(t *testing.T) {
		d := RegisterRequestDto{Username: "johndoe", Fullname: "John Doe", Password: "password1"}
		require.Error(t, d.ValidateForRegistration())
	})
}

func TestRegisterInviteQueryDto_ValidateSignedURL(t *testing.T) {
	q := &RegisterInviteQueryDto{}

	t.Run("empty values returns error", func(t *testing.T) {
		err := q.ValidateSignedURL(url.Values{})
		require.Error(t, err)
	})

	t.Run("valid signed url returns nil", func(t *testing.T) {
		t.Setenv("HMAC_SECRET_KEY", "test-secret-key")
		raw, _ := signedurl.GenerateSignedURL("https://example.com", map[string]string{"invite_token": "tok"}, time.Minute)
		parsed, _ := url.Parse(raw)
		err := q.ValidateSignedURL(parsed.Query())
		assert.NoError(t, err)
	})
}

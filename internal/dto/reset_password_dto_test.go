package dto

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maintainerd/auth/internal/signedurl"
)

func TestResetPasswordRequestDto_Validate(t *testing.T) {
	tests := []struct {
		name    string
		dto     ResetPasswordRequestDTO
		wantErr bool
	}{
		{
			name:    "valid new password",
			dto:     ResetPasswordRequestDTO{NewPassword: "NewSecurePass123!"},
			wantErr: false,
		},
		{
			name:    "missing new password",
			dto:     ResetPasswordRequestDTO{NewPassword: ""},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.dto.Validate()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResetPasswordQueryDto_Validate(t *testing.T) {
	valid := ResetPasswordQueryDTO{
		Token:      "reset-token-abc",
		ClientID:   "client-1",
		ProviderID: "provider-1",
		Expires:    "1700000000",
		Sig:        "sig-abc123",
	}

	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, valid.Validate())
	})

	t.Run("missing token", func(t *testing.T) {
		d := valid
		d.Token = ""
		require.Error(t, d.Validate())
	})

	t.Run("missing client_id", func(t *testing.T) {
		d := valid
		d.ClientID = ""
		require.Error(t, d.Validate())
	})

	t.Run("missing provider_id", func(t *testing.T) {
		d := valid
		d.ProviderID = ""
		require.Error(t, d.Validate())
	})

	t.Run("missing expires", func(t *testing.T) {
		d := valid
		d.Expires = ""
		require.Error(t, d.Validate())
	})

	t.Run("missing sig", func(t *testing.T) {
		d := valid
		d.Sig = ""
		require.Error(t, d.Validate())
	})
}

func TestResetPasswordQueryDto_ValidateSignedURL(t *testing.T) {
	q := &ResetPasswordQueryDTO{}

	t.Run("empty values returns error", func(t *testing.T) {
		err := q.ValidateSignedURL(url.Values{})
		require.Error(t, err)
	})

	t.Run("valid signed url returns nil", func(t *testing.T) {
		t.Setenv("HMAC_SECRET_KEY", "test-secret-key")
		raw, _ := signedurl.GenerateSignedURL("https://example.com", map[string]string{"token": "reset-tok"}, time.Minute)
		parsed, _ := url.Parse(raw)
		err := q.ValidateSignedURL(parsed.Query())
		assert.NoError(t, err)
	})
}

package dto

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResetPasswordRequestDto_Validate(t *testing.T) {
	tests := []struct {
		name    string
		dto     ResetPasswordRequestDto
		wantErr bool
	}{
		{
			name:    "valid new password",
			dto:     ResetPasswordRequestDto{NewPassword: "NewSecurePass123!"},
			wantErr: false,
		},
		{
			name:    "missing new password",
			dto:     ResetPasswordRequestDto{NewPassword: ""},
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
	valid := ResetPasswordQueryDto{
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
	q := &ResetPasswordQueryDto{}
	t.Run("empty values returns error", func(t *testing.T) {
		err := q.ValidateSignedURL(url.Values{})
		require.Error(t, err)
	})
}


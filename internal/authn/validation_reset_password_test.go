package authn

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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



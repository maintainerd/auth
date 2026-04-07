package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForgotPasswordRequestDto_Validate(t *testing.T) {
	tests := []struct {
		name    string
		dto     ForgotPasswordRequestDTO
		wantErr bool
	}{
		{
			name:    "valid email",
			dto:     ForgotPasswordRequestDTO{Email: "user@example.com"},
			wantErr: false,
		},
		{
			name:    "missing email",
			dto:     ForgotPasswordRequestDTO{Email: ""},
			wantErr: true,
		},
		{
			name:    "invalid email format",
			dto:     ForgotPasswordRequestDTO{Email: "not-an-email"},
			wantErr: true,
		},
		{
			name:    "email too long",
			dto:     ForgotPasswordRequestDTO{Email: string(make([]byte, 256)) + "@x.com"},
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

func TestForgotPasswordResponseDto_Fields(t *testing.T) {
	resp := ForgotPasswordResponseDTO{Message: "Check your email", Success: true}
	assert.Equal(t, "Check your email", resp.Message)
	assert.True(t, resp.Success)
}


package invite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendInviteRequestDTO_Validate(t *testing.T) {
	registrationFlowStr := "00000000-0000-0000-0000-000000000001"

	tests := []struct {
		name    string
		dto     SendInviteRequestDTO
		wantErr bool
	}{
		{
			name:    "valid with email only",
			dto:     SendInviteRequestDTO{Email: "user@example.com"},
			wantErr: false,
		},
		{
			name:    "valid with email and registration_flow",
			dto:     SendInviteRequestDTO{Email: "user@example.com", RegistrationFlowUUID: &registrationFlowStr},
			wantErr: false,
		},
		{
			name:    "missing email",
			dto:     SendInviteRequestDTO{Email: ""},
			wantErr: true,
		},
		{
			name:    "invalid email format",
			dto:     SendInviteRequestDTO{Email: "not-an-email"},
			wantErr: true,
		},
		{
			name:    "email too short",
			dto:     SendInviteRequestDTO{Email: "a@b"},
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

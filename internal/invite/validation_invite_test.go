package invite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendInviteRequest_Validate(t *testing.T) {
	authFlowStr := "00000000-0000-0000-0000-000000000001"

	tests := []struct {
		name    string
		dto     SendInviteRequest
		wantErr bool
	}{
		{
			name:    "valid with email only",
			dto:     SendInviteRequest{Email: "user@example.com"},
			wantErr: false,
		},
		{
			name:    "valid with email and auth_flow",
			dto:     SendInviteRequest{Email: "user@example.com", AuthFlowUUID: &authFlowStr},
			wantErr: false,
		},
		{
			name:    "missing email",
			dto:     SendInviteRequest{Email: ""},
			wantErr: true,
		},
		{
			name:    "invalid email format",
			dto:     SendInviteRequest{Email: "not-an-email"},
			wantErr: true,
		},
		{
			name:    "email too short",
			dto:     SendInviteRequest{Email: "a@b"},
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

package dto

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendInviteRequest_Validate(t *testing.T) {
	validRoles := []uuid.UUID{uuid.New()}

	tests := []struct {
		name    string
		dto     SendInviteRequest
		wantErr bool
	}{
		{
			name:    "valid",
			dto:     SendInviteRequest{Email: "user@example.com", Roles: validRoles},
			wantErr: false,
		},
		{
			name:    "missing email",
			dto:     SendInviteRequest{Email: "", Roles: validRoles},
			wantErr: true,
		},
		{
			name:    "invalid email format",
			dto:     SendInviteRequest{Email: "not-an-email", Roles: validRoles},
			wantErr: true,
		},
		{
			name:    "email too short",
			dto:     SendInviteRequest{Email: "a@b", Roles: validRoles},
			wantErr: true,
		},
		{
			name:    "missing roles",
			dto:     SendInviteRequest{Email: "user@example.com", Roles: nil},
			wantErr: true,
		},
		{
			name:    "empty roles slice",
			dto:     SendInviteRequest{Email: "user@example.com", Roles: []uuid.UUID{}},
			wantErr: true,
		},
		{
			name: "too many roles",
			dto: SendInviteRequest{
				Email: "user@example.com",
				Roles: []uuid.UUID{
					uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(),
					uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(),
				},
			},
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


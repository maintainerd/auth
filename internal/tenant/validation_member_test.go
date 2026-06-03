package tenant

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantMemberAddMemberRequestDTO_Validate(t *testing.T) {
	cases := []struct {
		name    string
		dto     TenantMemberAddMemberRequestDTO
		wantErr string
	}{
		{name: "valid", dto: TenantMemberAddMemberRequestDTO{UserUUID: uuid.New(), Role: "member"}},
		{name: "user required", dto: TenantMemberAddMemberRequestDTO{Role: "member"}, wantErr: "user_id"},
		{name: "role required", dto: TenantMemberAddMemberRequestDTO{UserUUID: uuid.New()}, wantErr: "role"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.dto.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestTenantMemberUpdateRoleRequestDTO_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		require.NoError(t, TenantMemberUpdateRoleRequestDTO{Role: "owner"}.Validate())
	})

	t.Run("role required", func(t *testing.T) {
		err := TenantMemberUpdateRoleRequestDTO{}.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role")
	})
}

func TestTenantMemberFilterDTO_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		err := TenantMemberFilterDTO{PaginationRequestDTO: PaginationRequestDTO{Page: 1, Limit: 10}}.Validate()
		require.NoError(t, err)
	})

	t.Run("invalid pagination", func(t *testing.T) {
		err := TenantMemberFilterDTO{PaginationRequestDTO: PaginationRequestDTO{Page: -1, Limit: 10}}.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "page")
	})
}

package tenant

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantMember_TableName(t *testing.T) {
	assert.Equal(t, "tenant_members", TenantMember{}.TableName())
}

func TestTenantMember_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid when missing", func(t *testing.T) {
		model := &TenantMember{}

		err := model.BeforeCreate(nil)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, model.TenantMemberUUID)
	})

	t.Run("preserves existing uuid", func(t *testing.T) {
		existing := uuid.New()
		model := &TenantMember{TenantMemberUUID: existing}

		err := model.BeforeCreate(nil)

		require.NoError(t, err)
		assert.Equal(t, existing, model.TenantMemberUUID)
	})
}

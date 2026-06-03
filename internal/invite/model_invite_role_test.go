package invite

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInviteRole_TableName(t *testing.T) {
	assert.Equal(t, "invite_roles", InviteRole{}.TableName())
}

func TestInviteRole_BeforeCreate(t *testing.T) {
	t.Run("generates UUID when nil", func(t *testing.T) {
		ir := &InviteRole{}
		err := ir.BeforeCreate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, ir.InviteRoleUUID)
	})

	t.Run("keeps existing UUID", func(t *testing.T) {
		existing := uuid.New()
		ir := &InviteRole{InviteRoleUUID: existing}
		err := ir.BeforeCreate(nil)
		require.NoError(t, err)
		assert.Equal(t, existing, ir.InviteRoleUUID)
	})
}

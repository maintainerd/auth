package invite

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvite_TableName(t *testing.T) {
	assert.Equal(t, "invites", Invite{}.TableName())
}

func TestInvite_BeforeCreate(t *testing.T) {
	t.Run("generates UUID when nil", func(t *testing.T) {
		invite := &Invite{}
		err := invite.BeforeCreate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, invite.InviteUUID)
	})

	t.Run("keeps existing UUID", func(t *testing.T) {
		existing := uuid.New()
		invite := &Invite{InviteUUID: existing}
		err := invite.BeforeCreate(nil)
		require.NoError(t, err)
		assert.Equal(t, existing, invite.InviteUUID)
	})
}

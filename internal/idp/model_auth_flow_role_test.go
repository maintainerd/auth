package idp

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthFlowRole_TableName(t *testing.T) {
	assert.Equal(t, "auth_flow_roles", AuthFlowRole{}.TableName())
}

func TestAuthFlowRole_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid when empty", func(t *testing.T) {
		role := &AuthFlowRole{}
		require.NoError(t, role.BeforeCreate(nil))
		assert.NotEqual(t, uuid.Nil, role.AuthFlowRoleUUID)
	})

	t.Run("keeps existing uuid", func(t *testing.T) {
		existing := uuid.New()
		role := &AuthFlowRole{AuthFlowRoleUUID: existing}
		require.NoError(t, role.BeforeCreate(nil))
		assert.Equal(t, existing, role.AuthFlowRoleUUID)
	})
}

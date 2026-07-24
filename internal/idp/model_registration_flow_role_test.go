package idp

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistrationFlowRole_TableName(t *testing.T) {
	assert.Equal(t, "registration_flow_roles", RegistrationFlowRole{}.TableName())
}

func TestRegistrationFlowRole_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid when empty", func(t *testing.T) {
		role := &RegistrationFlowRole{}
		require.NoError(t, role.BeforeCreate(nil))
		assert.NotEqual(t, uuid.Nil, role.RegistrationFlowRoleUUID)
	})

	t.Run("keeps existing uuid", func(t *testing.T) {
		existing := uuid.New()
		role := &RegistrationFlowRole{RegistrationFlowRoleUUID: existing}
		require.NoError(t, role.BeforeCreate(nil))
		assert.Equal(t, existing, role.RegistrationFlowRoleUUID)
	})
}

func TestRegistrationFlowRole_BeforeCreate_LeavesTheRestAlone(t *testing.T) {
	role := &RegistrationFlowRole{RegistrationFlowID: 3, RoleID: 10}
	require.NoError(t, role.BeforeCreate(nil))
	assert.NotEqual(t, uuid.Nil, role.RegistrationFlowRoleUUID)
	assert.Equal(t, int64(3), role.RegistrationFlowID)
	assert.Equal(t, int64(10), role.RoleID)
	assert.Nil(t, role.Role)
	assert.Nil(t, role.RegistrationFlow)
}

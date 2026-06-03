package idp

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignupFlowRole_TableName(t *testing.T) {
	assert.Equal(t, "signup_flow_roles", SignupFlowRole{}.TableName())
}

func TestSignupFlowRole_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid when empty", func(t *testing.T) {
		role := &SignupFlowRole{}
		require.NoError(t, role.BeforeCreate(nil))
		assert.NotEqual(t, uuid.Nil, role.SignupFlowRoleUUID)
	})

	t.Run("keeps existing uuid", func(t *testing.T) {
		existing := uuid.New()
		role := &SignupFlowRole{SignupFlowRoleUUID: existing}
		require.NoError(t, role.BeforeCreate(nil))
		assert.Equal(t, existing, role.SignupFlowRoleUUID)
	})
}

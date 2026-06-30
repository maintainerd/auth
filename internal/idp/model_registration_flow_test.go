package idp

import (
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistrationFlow_TableName(t *testing.T) {
	assert.Equal(t, "registration_flows", RegistrationFlow{}.TableName())
}

func TestRegistrationFlow_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid and default status when empty", func(t *testing.T) {
		flow := &RegistrationFlow{}
		require.NoError(t, flow.BeforeCreate(nil))
		assert.NotEqual(t, uuid.Nil, flow.RegistrationFlowUUID)
		assert.Equal(t, shared.StatusActive, flow.Status)
	})

	t.Run("keeps existing uuid and status", func(t *testing.T) {
		existing := uuid.New()
		flow := &RegistrationFlow{
			RegistrationFlowUUID: existing,
			Status:               shared.StatusInactive,
		}
		require.NoError(t, flow.BeforeCreate(nil))
		assert.Equal(t, existing, flow.RegistrationFlowUUID)
		assert.Equal(t, shared.StatusInactive, flow.Status)
	})
}

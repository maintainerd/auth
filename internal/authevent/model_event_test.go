package authevent

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthEvent_TableName(t *testing.T) {
	assert.Equal(t, "auth_events", AuthEvent{}.TableName())
}

func TestAuthEvent_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid when missing", func(t *testing.T) {
		event := &AuthEvent{}

		err := event.BeforeCreate(nil)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, event.AuthEventUUID)
	})

	t.Run("preserves existing uuid", func(t *testing.T) {
		existing := uuid.New()
		event := &AuthEvent{AuthEventUUID: existing}

		err := event.BeforeCreate(nil)

		require.NoError(t, err)
		assert.Equal(t, existing, event.AuthEventUUID)
	})
}

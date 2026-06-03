package client

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIKey_TableName(t *testing.T) {
	assert.Equal(t, "api_keys", APIKey{}.TableName())
}

func TestAPIKey_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid when missing", func(t *testing.T) {
		a := &APIKey{}
		err := a.BeforeCreate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, a.APIKeyUUID)
	})
	t.Run("preserves existing uuid", func(t *testing.T) {
		existing := uuid.New()
		a := &APIKey{APIKeyUUID: existing}
		err := a.BeforeCreate(nil)
		require.NoError(t, err)
		assert.Equal(t, existing, a.APIKeyUUID)
	})
}

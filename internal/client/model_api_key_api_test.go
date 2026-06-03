package client

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyAPI_TableName(t *testing.T) {
	assert.Equal(t, "api_key_apis", APIKeyAPI{}.TableName())
}

func TestAPIKeyAPI_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid when missing", func(t *testing.T) {
		a := &APIKeyAPI{}
		err := a.BeforeCreate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, a.APIKeyAPIUUID)
	})
	t.Run("preserves existing uuid", func(t *testing.T) {
		existing := uuid.New()
		a := &APIKeyAPI{APIKeyAPIUUID: existing}
		err := a.BeforeCreate(nil)
		require.NoError(t, err)
		assert.Equal(t, existing, a.APIKeyAPIUUID)
	})
}

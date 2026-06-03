package client

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_TableName(t *testing.T) {
	assert.Equal(t, "clients", Client{}.TableName())
}

func TestClient_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid when missing", func(t *testing.T) {
		a := &Client{}
		err := a.BeforeCreate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, a.ClientUUID)
	})
	t.Run("preserves existing uuid", func(t *testing.T) {
		existing := uuid.New()
		a := &Client{ClientUUID: existing}
		err := a.BeforeCreate(nil)
		require.NoError(t, err)
		assert.Equal(t, existing, a.ClientUUID)
	})
}

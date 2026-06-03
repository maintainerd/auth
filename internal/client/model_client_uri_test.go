package client

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientURI_TableName(t *testing.T) {
	assert.Equal(t, "client_uris", ClientURI{}.TableName())
}

func TestClientURI_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid when missing", func(t *testing.T) {
		a := &ClientURI{}
		err := a.BeforeCreate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, a.ClientURIUUID)
	})
	t.Run("preserves existing uuid", func(t *testing.T) {
		existing := uuid.New()
		a := &ClientURI{ClientURIUUID: existing}
		err := a.BeforeCreate(nil)
		require.NoError(t, err)
		assert.Equal(t, existing, a.ClientURIUUID)
	})
}

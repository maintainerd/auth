package iam

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIModel(t *testing.T) {
	t.Run("table name", func(t *testing.T) {
		assert.Equal(t, "apis", API{}.TableName())
	})

	t.Run("before create assigns uuid when nil", func(t *testing.T) {
		api := &API{}

		require.NoError(t, api.BeforeCreate(nil))

		assert.NotEqual(t, uuid.Nil, api.APIUUID)
	})

	t.Run("before create preserves existing uuid", func(t *testing.T) {
		id := uuid.New()
		api := &API{APIUUID: id}

		require.NoError(t, api.BeforeCreate(nil))

		assert.Equal(t, id, api.APIUUID)
	})
}

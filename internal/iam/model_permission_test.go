package iam

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermissionModel(t *testing.T) {
	t.Run("table name", func(t *testing.T) {
		assert.Equal(t, "permissions", Permission{}.TableName())
	})

	t.Run("before create assigns uuid when nil", func(t *testing.T) {
		permission := &Permission{}

		require.NoError(t, permission.BeforeCreate(nil))

		assert.NotEqual(t, uuid.Nil, permission.PermissionUUID)
	})

	t.Run("before create preserves existing uuid", func(t *testing.T) {
		id := uuid.New()
		permission := &Permission{PermissionUUID: id}

		require.NoError(t, permission.BeforeCreate(nil))

		assert.Equal(t, id, permission.PermissionUUID)
	})
}

package iam

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRolePermissionModel(t *testing.T) {
	t.Run("table name", func(t *testing.T) {
		assert.Equal(t, "role_permissions", RolePermission{}.TableName())
	})

	t.Run("before create assigns uuid when nil", func(t *testing.T) {
		rolePermission := &RolePermission{}

		require.NoError(t, rolePermission.BeforeCreate(nil))

		assert.NotEqual(t, uuid.Nil, rolePermission.RolePermissionUUID)
	})

	t.Run("before create preserves existing uuid", func(t *testing.T) {
		id := uuid.New()
		rolePermission := &RolePermission{RolePermissionUUID: id}

		require.NoError(t, rolePermission.BeforeCreate(nil))

		assert.Equal(t, id, rolePermission.RolePermissionUUID)
	})
}

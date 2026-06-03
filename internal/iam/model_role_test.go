package iam

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoleModel(t *testing.T) {
	t.Run("table name", func(t *testing.T) {
		assert.Equal(t, "roles", Role{}.TableName())
	})

	t.Run("before create assigns uuid when nil", func(t *testing.T) {
		role := &Role{}

		require.NoError(t, role.BeforeCreate(nil))

		assert.NotEqual(t, uuid.Nil, role.RoleUUID)
	})

	t.Run("before create preserves existing uuid", func(t *testing.T) {
		id := uuid.New()
		role := &Role{RoleUUID: id}

		require.NoError(t, role.BeforeCreate(nil))

		assert.Equal(t, id, role.RoleUUID)
	})
}

package client

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyPermission_TableName(t *testing.T) {
	assert.Equal(t, "api_key_permissions", APIKeyPermission{}.TableName())
}

func TestAPIKeyPermission_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid when missing", func(t *testing.T) {
		a := &APIKeyPermission{}
		err := a.BeforeCreate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, a.APIKeyPermissionUUID)
	})
	t.Run("preserves existing uuid", func(t *testing.T) {
		existing := uuid.New()
		a := &APIKeyPermission{APIKeyPermissionUUID: existing}
		err := a.BeforeCreate(nil)
		require.NoError(t, err)
		assert.Equal(t, existing, a.APIKeyPermissionUUID)
	})
}

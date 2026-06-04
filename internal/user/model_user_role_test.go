package user

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserRoleTableName(t *testing.T) {
	assert.Equal(t, "user_roles", UserRole{}.TableName())
}

func TestUserRoleBeforeCreate(t *testing.T) {
	t.Run("sets UUID when nil", func(t *testing.T) {
		ur := &UserRole{}
		err := ur.BeforeCreate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, ur.UserRoleUUID)
	})

	t.Run("preserves existing UUID", func(t *testing.T) {
		existing := uuid.New()
		ur := &UserRole{UserRoleUUID: existing}
		err := ur.BeforeCreate(&gorm.DB{})
		require.NoError(t, err)
		assert.Equal(t, existing, ur.UserRoleUUID)
	})
}

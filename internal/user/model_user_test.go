package user

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserTableName(t *testing.T) {
	assert.Equal(t, "users", User{}.TableName())
}

func TestUserBeforeCreate(t *testing.T) {
	t.Run("sets UUID when nil", func(t *testing.T) {
		u := &User{}
		err := u.BeforeCreate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, u.UserUUID)
	})

	t.Run("preserves existing UUID", func(t *testing.T) {
		existing := uuid.New()
		u := &User{UserUUID: existing}
		err := u.BeforeCreate(&gorm.DB{})
		require.NoError(t, err)
		assert.Equal(t, existing, u.UserUUID)
	})
}

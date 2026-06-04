package user

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserPoolTableName(t *testing.T) {
	assert.Equal(t, "user_pools", UserPool{}.TableName())
}

func TestUserPoolBeforeCreate(t *testing.T) {
	t.Run("sets UUID when nil", func(t *testing.T) {
		up := &UserPool{}
		err := up.BeforeCreate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, up.UserPoolUUID)
	})

	t.Run("preserves existing UUID", func(t *testing.T) {
		existing := uuid.New()
		up := &UserPool{UserPoolUUID: existing}
		err := up.BeforeCreate(&gorm.DB{})
		require.NoError(t, err)
		assert.Equal(t, existing, up.UserPoolUUID)
	})
}

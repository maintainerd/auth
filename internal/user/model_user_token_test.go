package user

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserTokenTableName(t *testing.T) {
	assert.Equal(t, "user_tokens", UserToken{}.TableName())
}

func TestUserTokenBeforeCreate(t *testing.T) {
	t.Run("sets UUID when nil", func(t *testing.T) {
		ut := &UserToken{}
		err := ut.BeforeCreate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, ut.UserTokenUUID)
	})

	t.Run("preserves existing UUID", func(t *testing.T) {
		existing := uuid.New()
		ut := &UserToken{UserTokenUUID: existing}
		err := ut.BeforeCreate(&gorm.DB{})
		require.NoError(t, err)
		assert.Equal(t, existing, ut.UserTokenUUID)
	})
}

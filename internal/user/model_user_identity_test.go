package user

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserIdentityTableName(t *testing.T) {
	assert.Equal(t, "user_identities", UserIdentity{}.TableName())
}

func TestUserIdentityBeforeCreate(t *testing.T) {
	t.Run("sets UUID when nil", func(t *testing.T) {
		ui := &UserIdentity{}
		err := ui.BeforeCreate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, ui.UserIdentityUUID)
	})

	t.Run("preserves existing UUID", func(t *testing.T) {
		existing := uuid.New()
		ui := &UserIdentity{UserIdentityUUID: existing}
		err := ui.BeforeCreate(&gorm.DB{})
		require.NoError(t, err)
		assert.Equal(t, existing, ui.UserIdentityUUID)
	})
}

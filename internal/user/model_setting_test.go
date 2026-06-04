package user

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserSettingTableName(t *testing.T) {
	assert.Equal(t, "user_settings", UserSetting{}.TableName())
}

func TestUserSettingBeforeCreate(t *testing.T) {
	t.Run("sets UUID when nil", func(t *testing.T) {
		us := &UserSetting{}
		err := us.BeforeCreate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, us.UserSettingUUID)
	})

	t.Run("preserves existing UUID", func(t *testing.T) {
		existing := uuid.New()
		us := &UserSetting{UserSettingUUID: existing}
		err := us.BeforeCreate(&gorm.DB{})
		require.NoError(t, err)
		assert.Equal(t, existing, us.UserSettingUUID)
	})
}

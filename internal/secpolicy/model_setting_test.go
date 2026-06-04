package secpolicy

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecuritySetting_TableName(t *testing.T) {
	assert.Equal(t, "security_settings", SecuritySetting{}.TableName())
}

func TestSecuritySetting_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid when missing", func(t *testing.T) {
		ss := &SecuritySetting{}
		err := ss.BeforeCreate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, ss.SecuritySettingUUID)
	})

	t.Run("preserves existing uuid", func(t *testing.T) {
		existing := uuid.New()
		ss := &SecuritySetting{SecuritySettingUUID: existing}
		err := ss.BeforeCreate(nil)
		require.NoError(t, err)
		assert.Equal(t, existing, ss.SecuritySettingUUID)
	})
}

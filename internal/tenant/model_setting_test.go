package tenant

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantSetting_TableName(t *testing.T) {
	assert.Equal(t, "tenant_settings", TenantSetting{}.TableName())
}

func TestTenantSetting_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid when missing", func(t *testing.T) {
		model := &TenantSetting{}

		err := model.BeforeCreate(nil)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, model.TenantSettingUUID)
	})

	t.Run("preserves existing uuid", func(t *testing.T) {
		existing := uuid.New()
		model := &TenantSetting{TenantSettingUUID: existing}

		err := model.BeforeCreate(nil)

		require.NoError(t, err)
		assert.Equal(t, existing, model.TenantSettingUUID)
	})
}

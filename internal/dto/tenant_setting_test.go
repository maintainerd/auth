package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantSettingUpdateConfigRequestDTO_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		d := TenantSettingUpdateConfigRequestDTO{"max_rps": 100}
		assert.NoError(t, d.Validate())
	})

	t.Run("empty map rejected", func(t *testing.T) {
		d := TenantSettingUpdateConfigRequestDTO{}
		require.Error(t, d.Validate())
	})

	t.Run("nil map rejected", func(t *testing.T) {
		var d TenantSettingUpdateConfigRequestDTO
		require.Error(t, d.Validate())
	})
}

package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecuritySettingUpdateConfigRequestDto_Validate(t *testing.T) {
	t.Run("valid with data", func(t *testing.T) {
		d := SecuritySettingUpdateConfigRequestDTO{
			"max_login_attempts": 5,
			"lockout_duration":   300,
		}
		assert.NoError(t, d.Validate())
	})

	t.Run("valid single key", func(t *testing.T) {
		d := SecuritySettingUpdateConfigRequestDTO{"key": "value"}
		assert.NoError(t, d.Validate())
	})

	t.Run("empty config is invalid", func(t *testing.T) {
		d := SecuritySettingUpdateConfigRequestDTO{}
		require.Error(t, d.Validate())
	})

	t.Run("nil config is invalid", func(t *testing.T) {
		var d SecuritySettingUpdateConfigRequestDTO
		require.Error(t, d.Validate())
	})
}


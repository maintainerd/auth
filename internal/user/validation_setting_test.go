package user

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserSettingRequestDto_Validate(t *testing.T) {
	t.Run("valid empty (all optional)", func(t *testing.T) {
		d := UserSettingRequestDTO{}
		assert.NoError(t, d.Validate())
	})

	t.Run("valid full", func(t *testing.T) {
		d := UserSettingRequestDTO{
			Timezone:          strPtr("America/New_York"),
			PreferredLanguage: strPtr("en"),
			Locale:            strPtr("en-US"),
		}
		assert.NoError(t, d.Validate())
	})

	t.Run("timezone too long", func(t *testing.T) {
		d := UserSettingRequestDTO{Timezone: strPtr(string(make([]byte, 51)))}
		require.Error(t, d.Validate())
	})

	t.Run("preferred_language too short", func(t *testing.T) {
		d := UserSettingRequestDTO{PreferredLanguage: strPtr("e")}
		require.Error(t, d.Validate())
	})

	t.Run("preferred_language too long", func(t *testing.T) {
		d := UserSettingRequestDTO{PreferredLanguage: strPtr("en-US-extra1")}
		require.Error(t, d.Validate())
	})

	t.Run("locale too short", func(t *testing.T) {
		d := UserSettingRequestDTO{Locale: strPtr("e")}
		require.Error(t, d.Validate())
	})

	t.Run("locale too long", func(t *testing.T) {
		d := UserSettingRequestDTO{Locale: strPtr("en-US-extra1")}
		require.Error(t, d.Validate())
	})
}

func TestNewUserSettingResponseDTO(t *testing.T) {
	t.Run("nil input returns empty DTO", func(t *testing.T) {
		us := &UserSetting{UserSettingUUID: uuid.New()}
		dto := NewUserSettingResponseDTO(us)
		assert.NotNil(t, dto)
	})

	t.Run("timezone is mapped", func(t *testing.T) {
		tz := "UTC"
		us := &UserSetting{UserSettingUUID: uuid.New(), Timezone: &tz}
		dto := NewUserSettingResponseDTO(us)
		assert.Equal(t, &tz, dto.Timezone)
	})
}

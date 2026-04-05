package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maintainerd/auth/internal/model"
)

func TestUserSettingRequestDto_Validate(t *testing.T) {
	t.Run("valid empty (all optional)", func(t *testing.T) {
		d := UserSettingRequestDto{}
		assert.NoError(t, d.Validate())
	})

	t.Run("valid full", func(t *testing.T) {
		d := UserSettingRequestDto{
			Timezone:               strPtr("America/New_York"),
			PreferredLanguage:      strPtr("en"),
			Locale:                 strPtr("en-US"),
			PreferredContactMethod: strPtr(model.ContactMethodEmail),
			ProfileVisibility:      strPtr(model.VisibilityPublic),
			EmergencyContactName:   strPtr("Jane Doe"),
			EmergencyContactPhone:  strPtr("+1234567890"),
			EmergencyContactEmail:  strPtr("jane@example.com"),
		}
		assert.NoError(t, d.Validate())
	})

	t.Run("timezone too long", func(t *testing.T) {
		d := UserSettingRequestDto{Timezone: strPtr(string(make([]byte, 51)))}
		require.Error(t, d.Validate())
	})

	t.Run("preferred_language too short", func(t *testing.T) {
		d := UserSettingRequestDto{PreferredLanguage: strPtr("e")}
		require.Error(t, d.Validate())
	})

	t.Run("preferred_language too long", func(t *testing.T) {
		d := UserSettingRequestDto{PreferredLanguage: strPtr("en-US-extra1")}
		require.Error(t, d.Validate())
	})

	t.Run("invalid preferred_contact_method", func(t *testing.T) {
		d := UserSettingRequestDto{PreferredContactMethod: strPtr("telegram")}
		require.Error(t, d.Validate())
	})

	t.Run("valid sms contact method", func(t *testing.T) {
		d := UserSettingRequestDto{PreferredContactMethod: strPtr(model.ContactMethodSMS)}
		assert.NoError(t, d.Validate())
	})

	t.Run("valid phone contact method", func(t *testing.T) {
		d := UserSettingRequestDto{PreferredContactMethod: strPtr(model.ContactMethodPhone)}
		assert.NoError(t, d.Validate())
	})

	t.Run("invalid profile_visibility", func(t *testing.T) {
		d := UserSettingRequestDto{ProfileVisibility: strPtr("hidden")}
		require.Error(t, d.Validate())
	})

	t.Run("valid private visibility", func(t *testing.T) {
		d := UserSettingRequestDto{ProfileVisibility: strPtr(model.VisibilityPrivate)}
		assert.NoError(t, d.Validate())
	})

	t.Run("valid friends visibility", func(t *testing.T) {
		d := UserSettingRequestDto{ProfileVisibility: strPtr(model.VisibilityFriends)}
		assert.NoError(t, d.Validate())
	})

	t.Run("invalid emergency contact email", func(t *testing.T) {
		d := UserSettingRequestDto{EmergencyContactEmail: strPtr("not-an-email")}
		require.Error(t, d.Validate())
	})

	t.Run("emergency contact name too long", func(t *testing.T) {
		d := UserSettingRequestDto{EmergencyContactName: strPtr(string(make([]byte, 201)))}
		require.Error(t, d.Validate())
	})
}


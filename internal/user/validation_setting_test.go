package user

import (
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func TestUserSettingRequestDto_Validate(t *testing.T) {
	t.Run("valid empty (all optional)", func(t *testing.T) {
		d := UserSettingRequestDTO{}
		assert.NoError(t, d.Validate())
	})

	t.Run("valid full", func(t *testing.T) {
		d := UserSettingRequestDTO{
			Timezone:               strPtr("America/New_York"),
			PreferredLanguage:      strPtr("en"),
			Locale:                 strPtr("en-US"),
			PreferredContactMethod: strPtr(shared.ContactMethodEmail),
			ProfileVisibility:      strPtr(shared.VisibilityPublic),
			EmergencyContactName:   strPtr("Jane Doe"),
			EmergencyContactPhone:  strPtr("+1234567890"),
			EmergencyContactEmail:  strPtr("jane@example.com"),
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

	t.Run("invalid preferred_contact_method", func(t *testing.T) {
		d := UserSettingRequestDTO{PreferredContactMethod: strPtr("telegram")}
		require.Error(t, d.Validate())
	})

	t.Run("valid sms contact method", func(t *testing.T) {
		d := UserSettingRequestDTO{PreferredContactMethod: strPtr(shared.ContactMethodSMS)}
		assert.NoError(t, d.Validate())
	})

	t.Run("valid phone contact method", func(t *testing.T) {
		d := UserSettingRequestDTO{PreferredContactMethod: strPtr(shared.ContactMethodPhone)}
		assert.NoError(t, d.Validate())
	})

	t.Run("invalid profile_visibility", func(t *testing.T) {
		d := UserSettingRequestDTO{ProfileVisibility: strPtr("hidden")}
		require.Error(t, d.Validate())
	})

	t.Run("valid private visibility", func(t *testing.T) {
		d := UserSettingRequestDTO{ProfileVisibility: strPtr(shared.VisibilityPrivate)}
		assert.NoError(t, d.Validate())
	})

	t.Run("valid friends visibility", func(t *testing.T) {
		d := UserSettingRequestDTO{ProfileVisibility: strPtr(shared.VisibilityFriends)}
		assert.NoError(t, d.Validate())
	})

	t.Run("invalid emergency contact email", func(t *testing.T) {
		d := UserSettingRequestDTO{EmergencyContactEmail: strPtr("not-an-email")}
		require.Error(t, d.Validate())
	})

	t.Run("emergency contact name too long", func(t *testing.T) {
		d := UserSettingRequestDTO{EmergencyContactName: strPtr(string(make([]byte, 201)))}
		require.Error(t, d.Validate())
	})
}

func TestNewUserSettingResponseDTO(t *testing.T) {
	t.Run("empty social links", func(t *testing.T) {
		us := &UserSetting{UserSettingUUID: uuid.New()}
		dto := NewUserSettingResponseDTO(us)
		assert.NotNil(t, dto)
		assert.Nil(t, dto.SocialLinks)
	})

	t.Run("valid social links JSON", func(t *testing.T) {
		us := &UserSetting{UserSettingUUID: uuid.New(), SocialLinks: datatypes.JSON(`{"twitter":"@user"}`)}
		dto := NewUserSettingResponseDTO(us)
		assert.Equal(t, "@user", dto.SocialLinks["twitter"])
	})

	t.Run("invalid social links JSON sets nil", func(t *testing.T) {
		us := &UserSetting{UserSettingUUID: uuid.New(), SocialLinks: datatypes.JSON([]byte("not-json"))}
		dto := NewUserSettingResponseDTO(us)
		assert.Nil(t, dto.SocialLinks)
	})
}

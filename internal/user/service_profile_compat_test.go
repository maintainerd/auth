package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputeFullname(t *testing.T) {
	t.Run("nil user returns empty", func(t *testing.T) {
		assert.Equal(t, "", computeFullname(nil))
	})

	t.Run("nil profile returns empty", func(t *testing.T) {
		assert.Equal(t, "", computeFullname(&User{}))
	})

	t.Run("display name used when present", func(t *testing.T) {
		dn := "Johnny"
		user := &User{Profile: &Profile{DisplayName: &dn, FirstName: "John"}}
		assert.Equal(t, "Johnny", computeFullname(user))
	})

	t.Run("display name empty strings fall back", func(t *testing.T) {
		dn := "   "
		ln := strPtr("Doe")
		user := &User{Profile: &Profile{DisplayName: &dn, FirstName: "John", LastName: ln}}
		assert.Equal(t, "John Doe", computeFullname(user))
	})

	t.Run("first name only", func(t *testing.T) {
		user := &User{Profile: &Profile{FirstName: "Alice"}}
		assert.Equal(t, "Alice", computeFullname(user))
	})

	t.Run("first and last name", func(t *testing.T) {
		ln := strPtr("Smith")
		user := &User{Profile: &Profile{FirstName: "Alice", LastName: ln}}
		assert.Equal(t, "Alice Smith", computeFullname(user))
	})

	t.Run("empty first name with last name", func(t *testing.T) {
		ln := strPtr("Doe")
		user := &User{Profile: &Profile{FirstName: "", LastName: ln}}
		assert.Equal(t, "Doe", computeFullname(user))
	})
}

func TestHydrateProfileTransients(t *testing.T) {
	t.Run("nil profile does nothing", func(t *testing.T) {
		hydrateProfileTransients(nil, &User{Email: "a@b.com"}, &UserSetting{})
	})

	t.Run("nil user does not set email", func(t *testing.T) {
		p := &Profile{}
		hydrateProfileTransients(p, nil, nil)
		assert.Nil(t, p.Email)
	})

	t.Run("user with email sets email on profile", func(t *testing.T) {
		p := &Profile{}
		hydrateProfileTransients(p, &User{Email: "test@example.com"}, nil)
		assert.NotNil(t, p.Email)
		assert.Equal(t, "test@example.com", *p.Email)
	})

	t.Run("user with empty email does not set email", func(t *testing.T) {
		p := &Profile{}
		hydrateProfileTransients(p, &User{Email: ""}, nil)
		assert.Nil(t, p.Email)
	})

	t.Run("nil settings does not set transient fields", func(t *testing.T) {
		p := &Profile{}
		hydrateProfileTransients(p, nil, nil)
		assert.Nil(t, p.Timezone)
		assert.Nil(t, p.Language)
	})

	t.Run("settings sets timezone and language", func(t *testing.T) {
		p := &Profile{}
		tz := strPtr("UTC")
		loc := strPtr("en")
		hydrateProfileTransients(p, nil, &UserSetting{Timezone: tz, Locale: loc})
		assert.Equal(t, "UTC", *p.Timezone)
		assert.Equal(t, "en", *p.Language)
	})

	t.Run("both user and settings hydrate all fields", func(t *testing.T) {
		p := &Profile{}
		tz := strPtr("America/New_York")
		loc := strPtr("fr")
		hydrateProfileTransients(p, &User{Email: "user@test.com"}, &UserSetting{Timezone: tz, Locale: loc})
		assert.Equal(t, "user@test.com", *p.Email)
		assert.Equal(t, "America/New_York", *p.Timezone)
		assert.Equal(t, "fr", *p.Language)
	})
}

func TestHydrateUserSettingTransients(t *testing.T) {
	t.Run("nil setting does nothing", func(t *testing.T) {
		hydrateUserSettingTransients(nil)
	})

	t.Run("setting copies locale to preferred language", func(t *testing.T) {
		loc := strPtr("en-US")
		s := &UserSetting{Locale: loc}
		hydrateUserSettingTransients(s)
		assert.Equal(t, "en-US", *s.PreferredLanguage)
	})

	t.Run("nil locale copies nil", func(t *testing.T) {
		s := &UserSetting{}
		hydrateUserSettingTransients(s)
		assert.Nil(t, s.PreferredLanguage)
	})
}

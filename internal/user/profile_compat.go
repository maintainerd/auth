package user

import (
	"strings"

	"github.com/maintainerd/auth/internal/model"
)

// computeFullname derives a display string for a user from their Profile.
// Order of preference: Profile.DisplayName → FirstName + LastName → FirstName.
// Returns empty string when Profile is nil or has no usable name fields.
func computeFullname(user *model.User) string {
	if user == nil || user.Profile == nil {
		return ""
	}
	p := user.Profile
	if p.DisplayName != nil && strings.TrimSpace(*p.DisplayName) != "" {
		return strings.TrimSpace(*p.DisplayName)
	}
	parts := []string{}
	if strings.TrimSpace(p.FirstName) != "" {
		parts = append(parts, strings.TrimSpace(p.FirstName))
	}
	if p.LastName != nil && strings.TrimSpace(*p.LastName) != "" {
		parts = append(parts, strings.TrimSpace(*p.LastName))
	}
	return strings.Join(parts, " ")
}

// hydrateProfileTransients copies values from related models into the profile's
// non-persisted (transient) fields so existing API/handler code paths keep
// returning the data they used to. Should be called after loading a Profile
// with its User and (optionally) UserSetting preloaded.
func hydrateProfileTransients(p *model.Profile, user *model.User, settings *model.UserSetting) {
	if p == nil {
		return
	}
	if user != nil && user.Email != "" {
		email := user.Email
		p.Email = &email
	}
	if settings != nil {
		p.Timezone = settings.Timezone
		p.Language = settings.Locale
	}
}

// hydrateUserSettingTransients mirrors Locale → PreferredLanguage on a UserSetting
// for API compatibility. PreferredLanguage was removed from the schema; Locale
// (BCP-47) is the single source of truth.
func hydrateUserSettingTransients(s *model.UserSetting) {
	if s == nil {
		return
	}
	s.PreferredLanguage = s.Locale
}

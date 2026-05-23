package service

import (
	"strings"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
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

// splitFullname splits a free-form name string into FirstName / LastName.
// The first whitespace-delimited token is FirstName; everything after is LastName.
// Returns ("", nil) when input is empty/whitespace.
func splitFullname(fullname string) (firstName string, lastName *string) {
	trimmed := strings.TrimSpace(fullname)
	if trimmed == "" {
		return "", nil
	}
	parts := strings.SplitN(trimmed, " ", 2)
	firstName = parts[0]
	if len(parts) > 1 {
		last := strings.TrimSpace(parts[1])
		if last != "" {
			lastName = &last
		}
	}
	return firstName, lastName
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

// ensureDefaultProfile creates a default Profile for the given user when one
// doesn't already exist, populated from a free-form fullname. No-ops on empty
// fullname or when a default profile is already present. Callers must invoke
// this inside a transaction (pass the *gorm.DB tx).
func ensureDefaultProfile(tx *gorm.DB, userID int64, fullname string) error {
	if strings.TrimSpace(fullname) == "" {
		return nil
	}
	var existing model.Profile
	err := tx.Where("user_id = ? AND is_default = ?", userID, true).First(&existing).Error
	if err == nil {
		return nil // already has a default profile
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	firstName, lastName := splitFullname(fullname)
	profile := &model.Profile{
		ProfileUUID: uuid.New(),
		UserID:      userID,
		FirstName:   firstName,
		LastName:    lastName,
		IsDefault:   true,
	}
	return tx.Create(profile).Error
}

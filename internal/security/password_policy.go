package security

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PasswordPolicy defines the complexity and lifecycle rules applied to a user's password.
// All fields are optional overrides — unset fields fall back to DefaultPasswordPolicy().
type PasswordPolicy struct {
	MinLength        int  `json:"min_length"`
	MaxLength        int  `json:"max_length"`
	RequireUpper     bool `json:"require_upper"`
	RequireLower     bool `json:"require_lower"`
	RequireDigit     bool `json:"require_digit"`
	RequireSpecial   bool `json:"require_special"`
	BlocklistEnabled bool `json:"blocklist_enabled"`
	// HistoryCount is the number of previous hashes to check for reuse. 0 disables history.
	HistoryCount int `json:"history_count"`
	// ExpiryDays forces password change after N days. 0 disables expiry.
	ExpiryDays int `json:"expiry_days"`
}

// DefaultPasswordPolicy returns the system-wide baseline that applies when no
// tenant-specific policy is configured.
func DefaultPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:        8,
		MaxLength:        128,
		RequireUpper:     true,
		RequireLower:     true,
		RequireDigit:     true,
		RequireSpecial:   true,
		BlocklistEnabled: true,
	}
}

// MergePasswordPolicy parses raw JSON (e.g. SecuritySetting.PasswordConfig) and
// merges it over the defaults. Fields absent in the JSON keep their default values.
func MergePasswordPolicy(raw []byte) PasswordPolicy {
	policy := DefaultPasswordPolicy()
	if len(raw) == 0 {
		return policy
	}
	s := strings.TrimSpace(string(raw))
	if s == "{}" || s == "null" || s == "" {
		return policy
	}
	var override struct {
		MinLength        *int  `json:"min_length"`
		MaxLength        *int  `json:"max_length"`
		RequireUpper     *bool `json:"require_upper"`
		RequireLower     *bool `json:"require_lower"`
		RequireDigit     *bool `json:"require_digit"`
		RequireSpecial   *bool `json:"require_special"`
		BlocklistEnabled *bool `json:"blocklist_enabled"`
		HistoryCount     *int  `json:"history_count"`
		ExpiryDays       *int  `json:"expiry_days"`
	}
	if err := json.Unmarshal(raw, &override); err != nil {
		return policy
	}
	if override.MinLength != nil {
		policy.MinLength = *override.MinLength
	}
	if override.MaxLength != nil {
		policy.MaxLength = *override.MaxLength
	}
	if override.RequireUpper != nil {
		policy.RequireUpper = *override.RequireUpper
	}
	if override.RequireLower != nil {
		policy.RequireLower = *override.RequireLower
	}
	if override.RequireDigit != nil {
		policy.RequireDigit = *override.RequireDigit
	}
	if override.RequireSpecial != nil {
		policy.RequireSpecial = *override.RequireSpecial
	}
	if override.BlocklistEnabled != nil {
		policy.BlocklistEnabled = *override.BlocklistEnabled
	}
	if override.HistoryCount != nil {
		policy.HistoryCount = *override.HistoryCount
	}
	if override.ExpiryDays != nil {
		policy.ExpiryDays = *override.ExpiryDays
	}
	return policy
}

// weakPasswords is the common-pattern blocklist used by ValidatePasswordPolicy.
var weakPasswords = []string{
	"password", "123456", "password123", "admin", "qwerty",
	"letmein", "welcome", "monkey", "dragon", "master",
}

// ValidatePasswordPolicy validates a password against the given policy.
func ValidatePasswordPolicy(password string, policy PasswordPolicy) error {
	if len(password) < policy.MinLength {
		return fmt.Errorf("password must be at least %d characters long", policy.MinLength)
	}
	if policy.MaxLength > 0 && len(password) > policy.MaxLength {
		return fmt.Errorf("password must not exceed %d characters", policy.MaxLength)
	}
	if policy.RequireUpper && !reUpper.MatchString(password) {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if policy.RequireLower && !reLower.MatchString(password) {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if policy.RequireDigit && !reDigit.MatchString(password) {
		return fmt.Errorf("password must contain at least one digit")
	}
	if policy.RequireSpecial && !reSpecial.MatchString(password) {
		return fmt.Errorf("password must contain at least one special character")
	}
	if policy.BlocklistEnabled {
		lower := strings.ToLower(password)
		for _, weak := range weakPasswords {
			if strings.Contains(lower, weak) {
				return fmt.Errorf("password contains common weak patterns")
			}
		}
	}
	return nil
}

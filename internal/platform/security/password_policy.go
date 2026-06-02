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

// commonPasswordBlocklist contains a curated embedded top-N common password list.
// Matching is exact after lowercasing and trimming to avoid substring false positives.
var commonPasswordBlocklist = map[string]struct{}{
	"123456":        {},
	"123456789":     {},
	"12345":         {},
	"qwerty":        {},
	"password":      {},
	"12345678":      {},
	"111111":        {},
	"123123":        {},
	"1234567890":    {},
	"1234567":       {},
	"qwerty123":     {},
	"000000":        {},
	"1q2w3e":        {},
	"aa12345678":    {},
	"abc123":        {},
	"password1":     {},
	"1234":          {},
	"qwertyuiop":    {},
	"123321":        {},
	"password123":   {},
	"1q2w3e4r5t":    {},
	"iloveyou":      {},
	"654321":        {},
	"666666":        {},
	"987654321":     {},
	"123":           {},
	"monkey":        {},
	"dragon":        {},
	"letmein":       {},
	"football":      {},
	"baseball":      {},
	"sunshine":      {},
	"princess":      {},
	"admin":         {},
	"welcome":       {},
	"login":         {},
	"solo":          {},
	"starwars":      {},
	"master":        {},
	"hello":         {},
	"freedom":       {},
	"whatever":      {},
	"qazwsx":        {},
	"trustno1":      {},
	"jordan":        {},
	"harley":        {},
	"buster":        {},
	"thomas":        {},
	"tigger":        {},
	"robert":        {},
	"soccer":        {},
	"hockey":        {},
	"killer":        {},
	"george":        {},
	"charlie":       {},
	"andrew":        {},
	"michelle":      {},
	"love":          {},
	"jessica":       {},
	"pepper":        {},
	"daniel":        {},
	"access":        {},
	"shadow":        {},
	"maggie":        {},
	"computer":      {},
	"ashley":        {},
	"bailey":        {},
	"passw0rd":      {},
	"superman":      {},
	"michael":       {},
	"football1":     {},
	"q1w2e3r4":      {},
	"zaq12wsx":      {},
	"password!":     {},
	"password1!":    {},
	"password123!":  {},
	"password1234!": {},
	"welcome1":      {},
	"welcome1!":     {},
	"admin123":      {},
	"admin123!":     {},
	"changeme":      {},
	"changeme1":     {},
	"changeme1!":    {},
	"letmein1":      {},
	"letmein1!":     {},
	"default":       {},
	"default1":      {},
	"default1!":     {},
	"maintainerd":   {},
	"maintainerd1":  {},
	"maintainerd1!": {},
	"company123":    {},
	"company123!":   {},
	"summer2024!":   {},
	"summer2025!":   {},
	"summer2026!":   {},
	"winter2024!":   {},
	"winter2025!":   {},
	"winter2026!":   {},
	"spring2024!":   {},
	"spring2025!":   {},
	"spring2026!":   {},
	"autumn2024!":   {},
	"autumn2025!":   {},
	"autumn2026!":   {},
	"fall2024!":     {},
	"fall2025!":     {},
	"fall2026!":     {},
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
		normalized := strings.ToLower(strings.TrimSpace(password))
		if _, found := commonPasswordBlocklist[normalized]; found {
			return fmt.Errorf("password is a common weak password")
		}
	}
	return nil
}

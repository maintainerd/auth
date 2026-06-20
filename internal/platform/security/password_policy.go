package security

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
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
	// CheckHIBP checks k-anonymity against HaveIBeenPwned on password change.
	CheckHIBP bool `json:"check_hibp"`
	// MinStrengthScore is the minimum zxcvbn 0-4 score. 0 disables the check.
	MinStrengthScore int `json:"min_strength_score"`
	// HashAlgorithm selects the password hashing function.
	HashAlgorithm string `json:"hash_algorithm"`
	// TempPasswordValidityHours is the lifetime of admin-generated temp passwords.
	TempPasswordValidityHours int `json:"temporary_password_validity_hours"`
}

// DefaultPasswordPolicy returns the system-wide baseline that applies when no
// tenant-specific policy is configured. MinStrengthScore and CheckHIBP default
// to 0/false here; they are enabled by tenant-level security_settings.
func DefaultPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:        8,
		MaxLength:        128,
		RequireUpper:     true,
		RequireLower:     true,
		RequireDigit:     true,
		RequireSpecial:   true,
		BlocklistEnabled: true,
		HashAlgorithm:    "argon2id",
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
		MinLength                      *int    `json:"min_length"`
		MaxLength                      *int    `json:"max_length"`
		RequireUpper                   *bool   `json:"require_upper"`
		RequireUppercase               *bool   `json:"require_uppercase"`
		RequireLower                   *bool   `json:"require_lower"`
		RequireLowercase               *bool   `json:"require_lowercase"`
		RequireDigit                   *bool   `json:"require_digit"`
		RequireNumber                  *bool   `json:"require_number"`
		RequireSpecial                 *bool   `json:"require_special"`
		RequireSymbol                  *bool   `json:"require_symbol"`
		BlocklistEnabled               *bool   `json:"blocklist_enabled"`
		RejectCommonPasswords          *bool   `json:"reject_common_passwords"`
		HistoryCount                   *int    `json:"history_count"`
		PasswordHistoryCount           *int    `json:"password_history_count"`
		ExpiryDays                     *int    `json:"expiry_days"`
		MaxAgeDays                     *int    `json:"max_age_days"`
		CheckHIBP                      *bool   `json:"check_hibp"`
		MinStrengthScore               *int    `json:"min_strength_score"`
		HashAlgorithm                  *string `json:"hash_algorithm"`
		TemporaryPasswordValidityHours *int    `json:"temporary_password_validity_hours"`
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
	if override.RequireUppercase != nil {
		policy.RequireUpper = *override.RequireUppercase
	}
	if override.RequireLower != nil {
		policy.RequireLower = *override.RequireLower
	}
	if override.RequireLowercase != nil {
		policy.RequireLower = *override.RequireLowercase
	}
	if override.RequireDigit != nil {
		policy.RequireDigit = *override.RequireDigit
	}
	if override.RequireNumber != nil {
		policy.RequireDigit = *override.RequireNumber
	}
	if override.RequireSpecial != nil {
		policy.RequireSpecial = *override.RequireSpecial
	}
	if override.RequireSymbol != nil {
		policy.RequireSpecial = *override.RequireSymbol
	}
	if override.BlocklistEnabled != nil {
		policy.BlocklistEnabled = *override.BlocklistEnabled
	}
	if override.RejectCommonPasswords != nil {
		policy.BlocklistEnabled = *override.RejectCommonPasswords
	}
	if override.HistoryCount != nil {
		policy.HistoryCount = *override.HistoryCount
	}
	if override.PasswordHistoryCount != nil {
		policy.HistoryCount = *override.PasswordHistoryCount
	}
	if override.ExpiryDays != nil {
		policy.ExpiryDays = *override.ExpiryDays
	}
	if override.MaxAgeDays != nil {
		policy.ExpiryDays = *override.MaxAgeDays
	}
	if override.CheckHIBP != nil {
		policy.CheckHIBP = *override.CheckHIBP
	}
	if override.MinStrengthScore != nil {
		policy.MinStrengthScore = *override.MinStrengthScore
	}
	if override.HashAlgorithm != nil {
		policy.HashAlgorithm = *override.HashAlgorithm
	}
	if override.TemporaryPasswordValidityHours != nil {
		policy.TempPasswordValidityHours = *override.TemporaryPasswordValidityHours
	}
	return policy
}

// commonPasswordBlocklist contains a curated embedded top-N common password list.
// Matching is exact after lowercasing and trimming to avoid substring false positives.
// Populated at init from common_passwords.txt; replace that file with the SecLists
// "10-million-password-list-top-10000.txt" for OWASP V2.1.7 compliance.
//
//go:embed common_passwords.txt
var commonPasswordsFile string

var commonPasswordBlocklist = func() map[string]struct{} {
	m := map[string]struct{}{
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
	for _, line := range strings.Split(commonPasswordsFile, "\n") {
		if p := strings.TrimSpace(strings.ToLower(line)); p != "" {
			m[p] = struct{}{}
		}
	}
	return m
}()

// ValidatePasswordPolicy validates a password against the given policy.
func ValidatePasswordPolicy(password string, policy PasswordPolicy) error {
	return ValidatePasswordPolicyWithContext(context.Background(), password, policy)
}

// ValidatePasswordPolicyWithContext validates a password against the given policy,
// including HIBP k-anonymity checks which require a context for tracing.
func ValidatePasswordPolicyWithContext(ctx context.Context, password string, policy PasswordPolicy) error {
	passwordLength := utf8.RuneCountInString(password)
	if passwordLength < policy.MinLength {
		return fmt.Errorf("password must be at least %d characters long", policy.MinLength)
	}
	if policy.MaxLength > 0 && passwordLength > policy.MaxLength {
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
	if policy.MinStrengthScore > 0 {
		score := PasswordStrengthScore(password)
		if score < policy.MinStrengthScore {
			return fmt.Errorf("password strength score %d is below the required minimum of %d", score, policy.MinStrengthScore)
		}
	}
	if policy.CheckHIBP {
		if CheckHIBPPassword(ctx, []byte(password)) {
			return fmt.Errorf("password was found in known data breaches")
		}
	}
	return nil
}

// Package valid provides input validation helpers.
package valid

import "regexp"

// Package-level compiled regexes — compiled once at program start.
var (
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	nonDigitRegex = regexp.MustCompile(`\D`)
	phoneRegex    = regexp.MustCompile(`^[\+]?[1-9][\d\s\-\(\)\.]{6,20}$`)
)

// IsValidEmail validates email format with security considerations.
// Complies with SOC2 CC6.1 and ISO27001 A.14.2.1.
func IsValidEmail(email string) bool {
	// Check length limit for security (RFC 5321 limit)
	if len(email) > 254 {
		return false
	}
	return emailRegex.MatchString(email)
}

// IsValidPhoneNumber validates phone number format.
//
// The first character after an optional leading '+' must be 1-9, so a number
// written with a leading parenthesis — "(123) 456-7890" — is REJECTED. Separators
// (space, hyphen, parenthesis, dot) are permitted only after that first digit.
// Accepted: +1234567890, 1234567890, 123-456-7890, 123 (456) 7890.
// Complies with SOC2 CC6.1 and ISO27001 A.9.4.2.
func IsValidPhoneNumber(phone string) bool {
	// Remove all non-digit characters for digit-count validation
	digitsOnly := nonDigitRegex.ReplaceAllString(phone, "")

	// Phone number should have 7-15 digits (international standard)
	if len(digitsOnly) < 7 || len(digitsOnly) > 15 {
		return false
	}

	return phoneRegex.MatchString(phone)
}

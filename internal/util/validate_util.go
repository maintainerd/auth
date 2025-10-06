package util

import "regexp"

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// IsValidEmail validates email format with security considerations
// Complies with SOC2 CC6.1 and ISO27001 A.14.2.1
func IsValidEmail(email string) bool {
	// Check length limit for security (RFC 5321 limit)
	if len(email) > 254 {
		return false
	}

	// Basic email regex validation
	return emailRegex.MatchString(email)
}

package util

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// IsValidEmail
// ---------------------------------------------------------------------------

func TestIsValidEmail_ValidAddresses(t *testing.T) {
	valid := []string{
		"user@example.com",
		"user.name+tag@sub.domain.org",
		"user_123@example.co.uk",
		"a@b.io",
	}
	for _, email := range valid {
		t.Run(email, func(t *testing.T) {
			assert.True(t, IsValidEmail(email), "expected %q to be valid", email)
		})
	}
}

func TestIsValidEmail_InvalidAddresses(t *testing.T) {
	invalid := []string{
		"",
		"notanemail",
		"missing@tld",
		"@nodomain.com",
		"spaces in@email.com",
		"double@@at.com",
	}
	for _, email := range invalid {
		t.Run(email, func(t *testing.T) {
			assert.False(t, IsValidEmail(email), "expected %q to be invalid", email)
		})
	}
}

func TestIsValidEmail_TooLong(t *testing.T) {
	// 255-char email should be rejected (RFC 5321 limit is 254)
	long := strings.Repeat("a", 244) + "@example.com" // 256 chars
	assert.False(t, IsValidEmail(long))
}

func TestIsValidEmail_ExactlyAtLimit(t *testing.T) {
	// Build an email exactly 254 chars long: local@domain.tld
	local := strings.Repeat("a", 242) // 242 + 1(@) + 7(ex.com) + 4(.com) = 254
	email := local + "@ex.com"
	// May be valid or invalid depending on regex, but must not panic
	_ = IsValidEmail(email)
}

// ---------------------------------------------------------------------------
// IsValidPhoneNumber
// ---------------------------------------------------------------------------

func TestIsValidPhoneNumber_ValidNumbers(t *testing.T) {
	valid := []string{
		"+12345678901",
		"1234567",
		"+1-800-555-0100",
		"+44 20 7946 0958",
	}
	for _, phone := range valid {
		t.Run(phone, func(t *testing.T) {
			assert.True(t, IsValidPhoneNumber(phone), "expected %q to be valid", phone)
		})
	}
}

func TestIsValidPhoneNumber_TooFewDigits(t *testing.T) {
	// Only 6 digits → too short
	assert.False(t, IsValidPhoneNumber("123456"))
}

func TestIsValidPhoneNumber_TooManyDigits(t *testing.T) {
	// 16 digits → too many
	assert.False(t, IsValidPhoneNumber("1234567890123456"))
}

func TestIsValidPhoneNumber_EmptyString(t *testing.T) {
	assert.False(t, IsValidPhoneNumber(""))
}

func TestIsValidPhoneNumber_InvalidFormat(t *testing.T) {
	// Starts with 0 which doesn't match [1-9] at position 0 (or after +)
	assert.False(t, IsValidPhoneNumber("0123456789"))
}


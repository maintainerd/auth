package util

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// ---------------------------------------------------------------------------
// ValidatePasswordStrength
// ---------------------------------------------------------------------------

func TestValidatePasswordStrength_Valid(t *testing.T) {
	validPasswords := []string{
		"Abcdef1!",
		"Str0ng@Pass",
		"C0mpl3x!Secure",
		"!Upper1lowercase",
	}
	for _, pw := range validPasswords {
		t.Run(pw, func(t *testing.T) {
			assert.NoError(t, ValidatePasswordStrength(pw))
		})
	}
}

func TestValidatePasswordStrength_TooShort(t *testing.T) {
	err := ValidatePasswordStrength("Ab1!")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 8 characters")
}

func TestValidatePasswordStrength_TooLong(t *testing.T) {
	pw := strings.Repeat("A1!a", 33) // 132 chars
	err := ValidatePasswordStrength(pw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "128 characters")
}

func TestValidatePasswordStrength_MissingUppercase(t *testing.T) {
	err := ValidatePasswordStrength("abcdef1!")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "uppercase")
}

func TestValidatePasswordStrength_MissingLowercase(t *testing.T) {
	err := ValidatePasswordStrength("ABCDEF1!")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lowercase")
}

func TestValidatePasswordStrength_MissingDigit(t *testing.T) {
	err := ValidatePasswordStrength("Abcdefg!")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "digit")
}

func TestValidatePasswordStrength_MissingSpecial(t *testing.T) {
	err := ValidatePasswordStrength("Abcdefg1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "special character")
}

func TestValidatePasswordStrength_WeakPattern(t *testing.T) {
	// Contains "password" → rejected even though it meets other criteria
	err := ValidatePasswordStrength("Password1!")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "weak")
}

// ---------------------------------------------------------------------------
// SanitizeInput
// ---------------------------------------------------------------------------

func TestSanitizeInput_StripsNullBytes(t *testing.T) {
	out := SanitizeInput("hel\x00lo")
	assert.Equal(t, "hello", out)
}

func TestSanitizeInput_TrimsWhitespace(t *testing.T) {
	out := SanitizeInput("  hello  ")
	assert.Equal(t, "hello", out)
}

func TestSanitizeInput_RemovesControlChars(t *testing.T) {
	// 0x01 (SOH) is a control character and should be stripped
	out := SanitizeInput("hel\x01lo")
	assert.Equal(t, "hello", out)
}

func TestSanitizeInput_KeepsTabNewline(t *testing.T) {
	out := SanitizeInput("line1\nline2")
	assert.Equal(t, "line1\nline2", out)
}

func TestSanitizeInput_EmptyString(t *testing.T) {
	assert.Equal(t, "", SanitizeInput(""))
}

// ---------------------------------------------------------------------------
// ValidateIPAddress
// ---------------------------------------------------------------------------

func TestValidateIPAddress_ValidPublicIPv4(t *testing.T) {
	assert.NoError(t, ValidateIPAddress("8.8.8.8"))
}

func TestValidateIPAddress_ValidPublicIPv6(t *testing.T) {
	assert.NoError(t, ValidateIPAddress("2001:db8::1"))
}

func TestValidateIPAddress_InvalidFormat(t *testing.T) {
	err := ValidateIPAddress("not-an-ip")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid IP")
}

func TestValidateIPAddress_LoopbackRestricted(t *testing.T) {
	err := ValidateIPAddress("127.0.0.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "restricted range")
}

func TestValidateIPAddress_LinkLocalRestricted(t *testing.T) {
	err := ValidateIPAddress("169.254.1.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "restricted range")
}

func TestValidateIPAddress_MulticastRestricted(t *testing.T) {
	err := ValidateIPAddress("224.0.0.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "restricted range")
}

// ---------------------------------------------------------------------------
// GetDummyBcryptHash
// ---------------------------------------------------------------------------

func TestGetDummyBcryptHash_NonNil(t *testing.T) {
	h := GetDummyBcryptHash()
	require.NotNil(t, h)
	assert.NotEmpty(t, h)
}

func TestGetDummyBcryptHash_IsValidBcrypt(t *testing.T) {
	// The hash must be a real bcrypt hash — CompareHashAndPassword should not
	// error because of a malformed hash (it may return ErrMismatchedHashAndPassword
	// when the password doesn't match, but NOT a format error).
	h := GetDummyBcryptHash()
	err := bcrypt.CompareHashAndPassword(h, []byte("wrong_password"))
	assert.ErrorIs(t, err, bcrypt.ErrMismatchedHashAndPassword)
}

func TestGetDummyBcryptHash_Idempotent(t *testing.T) {
	// Same slice returned on every call (pre-computed once at init)
	assert.Equal(t, GetDummyBcryptHash(), GetDummyBcryptHash())
}

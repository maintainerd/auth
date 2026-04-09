package security

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
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

// ---------------------------------------------------------------------------
// LogSecurityEvent
// ---------------------------------------------------------------------------

func TestLogSecurityEvent_ExplicitSeverity(t *testing.T) {
	// Should not panic and should keep the provided severity
	event := SecurityEvent{
		EventType: "login_success",
		UserID:    "user-123",
		Severity:  "INFO",
		Timestamp: time.Now(),
	}
	assert.NotPanics(t, func() { LogSecurityEvent(event) })
}

func TestLogSecurityEvent_AutoDeterminedSeverity(t *testing.T) {
	// Empty severity → determineSeverity fills it in; must not panic
	event := SecurityEvent{
		EventType: "account_locked",
		UserID:    "user-456",
		Timestamp: time.Now(),
	}
	assert.NotPanics(t, func() { LogSecurityEvent(event) })
}

// ---------------------------------------------------------------------------
// determineSeverity (unexported — accessible within package)
// ---------------------------------------------------------------------------

func TestDetermineSeverity(t *testing.T) {
	tests := []struct {
		eventType string
		want      string
	}{
		{"account_locked", "HIGH"},
		{"login_rate_limited", "HIGH"},
		{"suspicious_login", "HIGH"},
		{"ip_blocked", "HIGH"},
		{"token_validation_failure", "HIGH"},
		{"login_failure", "MEDIUM"},
		{"registration_failure", "MEDIUM"},
		{"validation_failure", "MEDIUM"},
		{"login_success", "LOW"},
		{"unknown_event", "LOW"},
	}
	for _, tc := range tests {
		t.Run(tc.eventType, func(t *testing.T) {
			assert.Equal(t, tc.want, determineSeverity(tc.eventType))
		})
	}
}

// ---------------------------------------------------------------------------
// GenerateCSRFToken
// ---------------------------------------------------------------------------

func TestGenerateCSRFToken_Format(t *testing.T) {
	tok, err := GenerateCSRFToken()
	require.NoError(t, err)
	assert.Len(t, tok, 64, "32 random bytes hex-encoded = 64 chars")
	for _, c := range tok {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'), "must be lowercase hex")
	}
}

func TestGenerateCSRFToken_Unique(t *testing.T) {
	a, err := GenerateCSRFToken()
	require.NoError(t, err)
	b, err := GenerateCSRFToken()
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
}

// ---------------------------------------------------------------------------
// ValidateUserAgent
// ---------------------------------------------------------------------------

func TestValidateUserAgent(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		want      bool
	}{
		{"empty", "", false},
		{"normal browser", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36", true},
		{"curl", "curl/7.68.0", true},
		{"sqlmap", "sqlmap/1.4", false},
		{"nikto scanner", "Nikto/2.1.6", false},
		{"script injection", "<script>alert(1)</script>", false},
		{"javascript protocol", "javascript:void(0)", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ValidateUserAgent(tc.userAgent))
		})
	}
}

// ---------------------------------------------------------------------------
// RateLimitKey / rateLimitCountKey / rateLimitLockKey
// ---------------------------------------------------------------------------

func TestRateLimitKey(t *testing.T) {
	key := RateLimitKey("user@example.com", "login")
	assert.Equal(t, "rate_limit:login:user@example.com", key)
}

func TestRateLimitCountKey(t *testing.T) {
	key := rateLimitCountKey("user@example.com")
	assert.Equal(t, "rl:count:user@example.com", key)
}

func TestRateLimitLockKey(t *testing.T) {
	key := rateLimitLockKey("user@example.com")
	assert.Equal(t, "rl:lock:user@example.com", key)
}

// ---------------------------------------------------------------------------
// Rate limiter — nil client (graceful degradation path)
// ---------------------------------------------------------------------------

func TestInitRateLimiter_AcceptsNil(t *testing.T) {
	// Should not panic
	assert.NotPanics(t, func() { InitRateLimiter(nil) })
}

func TestCheckRateLimit_NilClient(t *testing.T) {
	InitRateLimiter(nil)
	err := CheckRateLimit("user@example.com")
	assert.NoError(t, err, "nil client must degrade gracefully")
}

func TestRecordFailedAttempt_NilClient(t *testing.T) {
	InitRateLimiter(nil)
	assert.NotPanics(t, func() { RecordFailedAttempt("user@example.com") })
}

func TestResetFailedAttempts_NilClient(t *testing.T) {
	InitRateLimiter(nil)
	assert.NotPanics(t, func() { ResetFailedAttempts("user@example.com") })
}

// ---------------------------------------------------------------------------
// ValidateSessionLimit
// ---------------------------------------------------------------------------

func TestValidateSessionLimit(t *testing.T) {
	tests := []struct {
		name    string
		count   int
		wantErr bool
	}{
		{"below limit", MaxConcurrentSessions - 1, false},
		{"at limit", MaxConcurrentSessions, true},
		{"above limit", MaxConcurrentSessions + 1, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSessionLimit("user-123", tc.count)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "maximum concurrent sessions")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// newMiniredisClient — helper for rate-limiting tests
// ---------------------------------------------------------------------------

func newMiniredisClient(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	cli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mr, cli
}

func saveAndRestoreRateLimiter(t *testing.T) {
	t.Helper()
	saved := rateLimiterClient
	t.Cleanup(func() { rateLimiterClient = saved })
}

// ---------------------------------------------------------------------------
// ValidateIPAddress — reserved range (240.0.0.0/4)
// ---------------------------------------------------------------------------

func TestValidateIPAddress_ReservedRestricted(t *testing.T) {
	err := ValidateIPAddress("240.0.0.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "restricted range")
}

// ---------------------------------------------------------------------------
// CheckRateLimit — with Redis (locked account path)
// ---------------------------------------------------------------------------

func TestCheckRateLimit_LockedAccount(t *testing.T) {
	saveAndRestoreRateLimiter(t)
	mr, cli := newMiniredisClient(t)
	InitRateLimiter(cli)

	identifier := "locked-user@example.com"
	// Pre-set the lock key
	require.NoError(t, mr.Set(rateLimitLockKey(identifier), "1"))
	mr.SetTTL(rateLimitLockKey(identifier), AccountLockoutTime)

	err := CheckRateLimit(identifier)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account is locked")
}

// ---------------------------------------------------------------------------
// CheckRateLimit — count below threshold (no lockout)
// ---------------------------------------------------------------------------

func TestCheckRateLimit_BelowThreshold(t *testing.T) {
	saveAndRestoreRateLimiter(t)
	_, cli := newMiniredisClient(t)
	InitRateLimiter(cli)

	identifier := "user@example.com"
	// No keys set at all → should pass
	err := CheckRateLimit(identifier)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// CheckRateLimit — count at threshold → promotes to lockout
// ---------------------------------------------------------------------------

func TestCheckRateLimit_ExceedsMaxAttempts(t *testing.T) {
	saveAndRestoreRateLimiter(t)
	mr, cli := newMiniredisClient(t)
	InitRateLimiter(cli)

	identifier := "bad-actor@example.com"
	// Pre-set the count at the max
	require.NoError(t, mr.Set(rateLimitCountKey(identifier), fmt.Sprintf("%d", MaxLoginAttempts)))

	err := CheckRateLimit(identifier)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account locked")

	// Verify the lock key was set
	assert.True(t, mr.Exists(rateLimitLockKey(identifier)))
	// Verify the count key was removed
	assert.False(t, mr.Exists(rateLimitCountKey(identifier)))
}

// ---------------------------------------------------------------------------
// CheckRateLimit — count below max (no lockout promotion)
// ---------------------------------------------------------------------------

func TestCheckRateLimit_BelowMaxAttempts(t *testing.T) {
	saveAndRestoreRateLimiter(t)
	mr, cli := newMiniredisClient(t)
	InitRateLimiter(cli)

	identifier := "some-user@example.com"
	require.NoError(t, mr.Set(rateLimitCountKey(identifier), "2"))

	err := CheckRateLimit(identifier)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// RecordFailedAttempt — with Redis
// ---------------------------------------------------------------------------

func TestRecordFailedAttempt_WithRedis(t *testing.T) {
	saveAndRestoreRateLimiter(t)
	mr, cli := newMiniredisClient(t)
	InitRateLimiter(cli)

	identifier := "fail-user@example.com"
	RecordFailedAttempt(identifier)

	val, err := mr.Get(rateLimitCountKey(identifier))
	require.NoError(t, err)
	assert.Equal(t, "1", val)

	// Second attempt
	RecordFailedAttempt(identifier)
	val, err = mr.Get(rateLimitCountKey(identifier))
	require.NoError(t, err)
	assert.Equal(t, "2", val)
}

// ---------------------------------------------------------------------------
// ResetFailedAttempts — with Redis
// ---------------------------------------------------------------------------

func TestResetFailedAttempts_WithRedis(t *testing.T) {
	saveAndRestoreRateLimiter(t)
	mr, cli := newMiniredisClient(t)
	InitRateLimiter(cli)

	identifier := "reset-user@example.com"
	require.NoError(t, mr.Set(rateLimitCountKey(identifier), "3"))
	require.NoError(t, mr.Set(rateLimitLockKey(identifier), "1"))

	ResetFailedAttempts(identifier)

	assert.False(t, mr.Exists(rateLimitCountKey(identifier)))
	assert.False(t, mr.Exists(rateLimitLockKey(identifier)))
}

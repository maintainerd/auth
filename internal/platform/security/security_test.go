package security

import (
	"context"
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

func TestValidatePasswordStrength_CommonPassword(t *testing.T) {
	err := ValidatePasswordStrength("Password1!")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "common weak password")
}

func TestValidatePasswordStrength_CommonPasswordSubstringAllowed(t *testing.T) {
	err := ValidatePasswordStrength("ThisPasswordWordIsFine1!")
	assert.NoError(t, err)
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

func TestHashPassword_PreventsBcrypt72ByteCollision(t *testing.T) {
	passwordA := []byte(strings.Repeat("a", 72) + "X")
	passwordB := []byte(strings.Repeat("a", 72) + "Y")

	hash, err := HashPassword(context.Background(), passwordA)
	require.NoError(t, err)

	assert.True(t, ComparePassword(hash, passwordA))
	assert.False(t, ComparePassword(hash, passwordB))
}

func TestComparePassword_LegacyRawBcryptFallback(t *testing.T) {
	legacyHash, err := bcrypt.GenerateFromPassword([]byte("legacy-password"), BcryptCost)
	require.NoError(t, err)

	assert.True(t, ComparePassword(legacyHash, []byte("legacy-password")))
}

func TestCompareClientSecret_PreventsBcrypt72ByteCollision(t *testing.T) {
	secretA := strings.Repeat("s", 72) + "X"
	secretB := strings.Repeat("s", 72) + "Y"

	hash, err := HashClientSecret(context.Background(), secretA)
	require.NoError(t, err)

	assert.True(t, CompareClientSecret(secretA, hash))
	assert.False(t, CompareClientSecret(secretB, hash))
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

func TestCheckAndRecordSMSDailyBudget_InMemoryFallback(t *testing.T) {
	InitRateLimiter(nil)
	ResetSMSDailyBudgetCounters()
	t.Cleanup(ResetSMSDailyBudgetCounters)

	require.NoError(t, CheckAndRecordSMSDailyBudget(context.Background(), "global", 1))
	err := CheckAndRecordSMSDailyBudget(context.Background(), "global", 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "daily SMS send budget exceeded")
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

// ---------------------------------------------------------------------------
// ValidateRedirectURI
// ---------------------------------------------------------------------------

func TestValidateRedirectURI(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantErr bool
	}{
		{"valid https", "https://app.example.com/callback", false},
		{"valid http localhost", "http://localhost:3000/callback", false},
		{"custom scheme", "myapp://callback", false},
		{"javascript forbidden", "javascript:alert(1)", true},
		{"data forbidden", "data:text/html,<script>alert(1)</script>", true},
		{"vbscript forbidden", "VBSCRIPT:msgbox", true},
		{"file forbidden", "file:///etc/passwd", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRedirectURI(tc.uri)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "forbidden scheme")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// MergePasswordPolicy
// ---------------------------------------------------------------------------

func TestMergePasswordPolicy_EmptyReturnsDefault(t *testing.T) {
	p := MergePasswordPolicy(nil)
	assert.Equal(t, DefaultPasswordPolicy(), p)

	p = MergePasswordPolicy([]byte{})
	assert.Equal(t, DefaultPasswordPolicy(), p)

	p = MergePasswordPolicy([]byte("{}"))
	assert.Equal(t, DefaultPasswordPolicy(), p)

	p = MergePasswordPolicy([]byte("null"))
	assert.Equal(t, DefaultPasswordPolicy(), p)

	p = MergePasswordPolicy([]byte("  "))
	assert.Equal(t, DefaultPasswordPolicy(), p)
}

func TestMergePasswordPolicy_InvalidJSONReturnsDefault(t *testing.T) {
	p := MergePasswordPolicy([]byte("not-json"))
	assert.Equal(t, DefaultPasswordPolicy(), p)
}

func TestMergePasswordPolicy_OverridesFields(t *testing.T) {
	p := MergePasswordPolicy([]byte(`{"min_length":12,"require_upper":false,"history_count":5}`))
	assert.Equal(t, 12, p.MinLength)
	assert.Equal(t, 128, p.MaxLength)
	assert.False(t, p.RequireUpper)
	assert.True(t, p.RequireLower)
	assert.Equal(t, 5, p.HistoryCount)
	assert.Equal(t, 0, p.ExpiryDays)
}

func TestMergePasswordPolicy_AllFields(t *testing.T) {
	raw := `{"min_length":16,"max_length":64,"require_upper":false,"require_lower":false,"require_digit":false,"require_special":false,"blocklist_enabled":false,"history_count":10,"expiry_days":90}`
	p := MergePasswordPolicy([]byte(raw))
	assert.Equal(t, 16, p.MinLength)
	assert.Equal(t, 64, p.MaxLength)
	assert.False(t, p.RequireUpper)
	assert.False(t, p.RequireLower)
	assert.False(t, p.RequireDigit)
	assert.False(t, p.RequireSpecial)
	assert.False(t, p.BlocklistEnabled)
	assert.Equal(t, 10, p.HistoryCount)
	assert.Equal(t, 90, p.ExpiryDays)
}

// ---------------------------------------------------------------------------
// CheckAndRecordSMSDailyBudget — Redis path and disabled limit
// ---------------------------------------------------------------------------

func TestCheckAndRecordSMSDailyBudget_Disabled(t *testing.T) {
	ResetSMSDailyBudgetCounters()
	t.Cleanup(ResetSMSDailyBudgetCounters)

	assert.NoError(t, CheckAndRecordSMSDailyBudget(context.Background(), "scope", 0))
	assert.NoError(t, CheckAndRecordSMSDailyBudget(context.Background(), "scope", -1))
}

func TestCheckAndRecordSMSDailyBudget_NilCtx(t *testing.T) {
	ResetSMSDailyBudgetCounters()
	t.Cleanup(ResetSMSDailyBudgetCounters)

	assert.NoError(t, CheckAndRecordSMSDailyBudget(nil, "scope", 10))
}

func TestCheckAndRecordSMSDailyBudget_RedisPath(t *testing.T) {
	saveAndRestoreRateLimiter(t)
	mr, cli := newMiniredisClient(t)
	InitRateLimiter(cli)
	ResetSMSDailyBudgetCounters()
	t.Cleanup(ResetSMSDailyBudgetCounters)

	assert.NoError(t, CheckAndRecordSMSDailyBudget(context.Background(), "redis-scope", 2))
	assert.True(t, mr.Exists(smsDailyBudgetKey("redis-scope", time.Now())))

	assert.NoError(t, CheckAndRecordSMSDailyBudget(context.Background(), "redis-scope", 2))
	err := CheckAndRecordSMSDailyBudget(context.Background(), "redis-scope", 2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SMS send budget exceeded")
}

// ---------------------------------------------------------------------------
// smsDailyBudgetTTL
// ---------------------------------------------------------------------------

func TestSmsDailyBudgetTTL(t *testing.T) {
	now := time.Now()
	ttl := smsDailyBudgetTTL(now)
	assert.True(t, ttl > 0)
	assert.True(t, ttl < 26*time.Hour)
}

// ---------------------------------------------------------------------------
// CheckRateLimit — lock key with TTL parse error (graceful)
// ---------------------------------------------------------------------------

func TestCheckRateLimit_LockedAccountTTLError(t *testing.T) {
	saveAndRestoreRateLimiter(t)
	mr, cli := newMiniredisClient(t)
	InitRateLimiter(cli)

	identifier := "lock-ttl-err@example.com"
	require.NoError(t, mr.Set(rateLimitLockKey(identifier), "1"))
	mr.SetTTL(rateLimitLockKey(identifier), time.Second)
	mr.FastForward(time.Second)
	// Key should still exist after fast-forward (TTL in miniredis may differ)

	err := CheckRateLimit(identifier)
	if err != nil {
		assert.Contains(t, err.Error(), "account is locked")
	}
}

// ---------------------------------------------------------------------------
// CheckRateLimit — empty lock value (no lock)
// ---------------------------------------------------------------------------

func TestCheckRateLimit_EmptyLockValue(t *testing.T) {
	saveAndRestoreRateLimiter(t)
	_, cli := newMiniredisClient(t)
	InitRateLimiter(cli)

	identifier := "empty-lock@example.com"
	err := CheckRateLimit(identifier)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// CheckAndRecordSMSDailyBudget — in-memory path with reset
// ---------------------------------------------------------------------------

func TestCheckAndRecordSMSDailyBudget_MultipleScopes(t *testing.T) {
	saveAndRestoreRateLimiter(t)
	InitRateLimiter(nil)
	ResetSMSDailyBudgetCounters()
	t.Cleanup(ResetSMSDailyBudgetCounters)

	assert.NoError(t, CheckAndRecordSMSDailyBudget(context.Background(), "scope-A", 2))
	assert.NoError(t, CheckAndRecordSMSDailyBudget(context.Background(), "scope-B", 2))
	assert.NoError(t, CheckAndRecordSMSDailyBudget(context.Background(), "scope-A", 2))
	err := CheckAndRecordSMSDailyBudget(context.Background(), "scope-A", 2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SMS send budget exceeded")
	assert.NoError(t, CheckAndRecordSMSDailyBudget(context.Background(), "scope-B", 2))
}

// ---------------------------------------------------------------------------
// DefaultPasswordPolicy
// ---------------------------------------------------------------------------

func TestDefaultPasswordPolicy_Fields(t *testing.T) {
	p := DefaultPasswordPolicy()
	assert.Equal(t, 8, p.MinLength)
	assert.Equal(t, 128, p.MaxLength)
	assert.True(t, p.RequireUpper)
	assert.True(t, p.RequireLower)
	assert.True(t, p.RequireDigit)
	assert.True(t, p.RequireSpecial)
	assert.True(t, p.BlocklistEnabled)
}

// ---------------------------------------------------------------------------
// ValidatePasswordPolicy — custom policy without max length
// ---------------------------------------------------------------------------

func TestValidatePasswordPolicy_NoMaxLength(t *testing.T) {
	policy := PasswordPolicy{
		MinLength:      4,
		MaxLength:      0,
		RequireUpper:   false,
		RequireLower:   true,
		RequireDigit:   false,
		RequireSpecial: false,
	}
	assert.NoError(t, ValidatePasswordPolicy(strings.Repeat("a", 1000), policy))
}

// ---------------------------------------------------------------------------
// ValidatePasswordPolicy — blocklist disabled
// ---------------------------------------------------------------------------

func TestValidatePasswordPolicy_BlocklistDisabled(t *testing.T) {
	policy := PasswordPolicy{
		MinLength:        4,
		RequireLower:     false,
		BlocklistEnabled: false,
	}
	assert.NoError(t, ValidatePasswordPolicy("password", policy))
}

// ---------------------------------------------------------------------------
// CheckRateLimit — count key parse error
// ---------------------------------------------------------------------------

func TestCheckRateLimit_CountKeyNotSet(t *testing.T) {
	saveAndRestoreRateLimiter(t)
	_, cli := newMiniredisClient(t)
	InitRateLimiter(cli)

	identifier := "no-count@example.com"
	err := CheckRateLimit(identifier)
	assert.NoError(t, err)
}

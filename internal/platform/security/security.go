/*
Package security provides comprehensive security utilities for authentication and authorization.

This module implements security controls required for SOC2 and ISO27001 compliance, including:

SECURITY FEATURES:
- Rate limiting and account lockout protection
- Password strength validation and complexity requirements
- Input validation and sanitization
- Cryptographic utilities for secure token generation
- Security event logging and audit trails
- Session management and concurrent session limits
- Timing-safe operations to prevent timing attacks

COMPLIANCE STANDARDS:
- SOC2 Type II (CC6.1, CC6.3, CC7.2)
- ISO27001 (A.9.4.2, A.9.4.3, A.12.4.1, A.13.1.1, A.14.2.1)

USAGE:
This utility module is designed to be used across all services in the authentication
system to ensure consistent security policies and compliance requirements.

Example:

	// Rate limiting
	if err := util.CheckRateLimit(username); err != nil {
		return err
	}

	// Password validation
	if err := util.ValidatePasswordStrength(password); err != nil {
		return err
	}

	// Security logging
	util.LogSecurityEvent(util.SecurityEvent{
		EventType: "login_success",
		UserID:    userID,
		Timestamp: time.Now(),
	})
*/
package security

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================================
// SECURITY CONSTANTS
// ============================================================================
// These constants define security policies and limits across the application

// Rate Limiting Constants (SOC2 CC6.1 - Logical Access Controls)
// These are the fallback defaults used when no tenant-specific lockout config is
// available. Per-tenant values are read from security_settings.lockout_config.
const (
	MaxLoginAttempts   = 5                // Maximum failed attempts before lockout
	LoginAttemptWindow = 15 * time.Minute // Time window for counting attempts
	AccountLockoutTime = 30 * time.Minute // Account lockout duration
)

// Session Security Constants (SOC2 CC6.3 - Logical Access Controls)
const (
	MaxConcurrentSessions = 5 // Maximum concurrent sessions per user
)

// RateLimitConfig carries tenant-specific lockout thresholds read from
// security_settings.lockout_config. A nil config means "use the hardcoded
// constants above".
type RateLimitConfig struct {
	Enabled             bool
	MaxFailedAttempts   int
	LockoutDuration     time.Duration
	ObservationWindow   time.Duration
	AutoUnlock          bool
	ResetCountOnSuccess bool
	NotifyUserOnLockout bool
	MaxLockoutDuration  time.Duration
	ProgressiveLockout  bool
	ProgressionReset    time.Duration
}

// ============================================================================
// DATA STRUCTURES
// ============================================================================
// Types used for security operations and audit logging

// SecurityEvent represents a security-related event for audit logging
// Used for SOC2 CC7.2 and ISO27001 A.12.4.1 compliance
// Complies with SOC2 CC7.2 and ISO27001 A.12.4.1
type SecurityEvent struct {
	EventType string    `json:"event_type"`
	UserID    string    `json:"user_id,omitempty"`
	ClientID  string    `json:"client_id,omitempty"`
	ClientIP  string    `json:"client_ip,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	RequestID string    `json:"request_id,omitempty"`
	Endpoint  string    `json:"endpoint,omitempty"`
	Method    string    `json:"method,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Details   string    `json:"details,omitempty"`
	Severity  string    `json:"severity,omitempty"`
}

// LoginAttempt tracks failed login attempts for rate limiting
// Used by rate limiting functions to maintain attempt history
type LoginAttempt struct {
	Identifier  string     // Username, email, or IP
	Attempts    int        // Number of failed attempts
	LastAttempt time.Time  // Time of last attempt
	LockedUntil *time.Time // Account locked until this time
}

// ============================================================================
// SECURITY LOGGING
// ============================================================================
// Functions for audit logging and security event monitoring

// LogSecurityEvent logs security events for compliance monitoring
// Complies with SOC2 CC7.2 (System Monitoring) and ISO27001 A.12.4.1
func LogSecurityEvent(event SecurityEvent) {
	if event.Severity == "" {
		event.Severity = determineSeverity(event.EventType)
	}

	slog.Info("security_event",
		"event_type", event.EventType,
		"severity", event.Severity,
		"client_ip", event.ClientIP,
		"user_id", event.UserID,
		"request_id", event.RequestID,
		"endpoint", event.Endpoint,
		"method", event.Method,
		"details", event.Details,
		"timestamp", event.Timestamp.Format(time.RFC3339),
	)
}

// determineSeverity assigns severity levels to security events
func determineSeverity(eventType string) string {
	highSeverityEvents := []string{
		"account_locked",
		"login_rate_limited",
		"suspicious_login",
		"ip_blocked",
		"token_validation_failure",
	}

	mediumSeverityEvents := []string{
		"login_failure",
		"registration_failure",
		"validation_failure",
	}

	if slices.Contains(highSeverityEvents, eventType) {
		return "HIGH"
	}
	if slices.Contains(mediumSeverityEvents, eventType) {
		return "MEDIUM"
	}
	return "LOW"
}

// ============================================================================
// INPUT VALIDATION & SANITIZATION
// ============================================================================
// Functions for validating and sanitizing user input

// Package-level compiled regexes — compiled once, reused on every call.
var (
	reUpper   = regexp.MustCompile(`[A-Z]`)
	reLower   = regexp.MustCompile(`[a-z]`)
	reDigit   = regexp.MustCompile(`[0-9]`)
	reSpecial = regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`)
)

// ValidatePasswordStrength enforces the system-default password complexity requirements.
// For per-tenant policy enforcement, use ValidatePasswordPolicy with a loaded PasswordPolicy.
// Complies with SOC2 CC6.1 and ISO27001 A.9.4.3
func ValidatePasswordStrength(password string) error {
	return ValidatePasswordPolicy(password, DefaultPasswordPolicy())
}

// SanitizeInput sanitizes user input to prevent injection attacks
// Complies with SOC2 CC6.1 and ISO27001 A.14.2.1
func SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Trim whitespace
	input = strings.TrimSpace(input)

	// Remove control characters except tab, newline, carriage return
	var result strings.Builder
	for _, r := range input {
		if r >= 32 || r == 9 || r == 10 || r == 13 {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// dangerousSchemes contains URI schemes that must never appear in redirect URIs
// because they execute code in the browser (XSS via open redirect).
var dangerousSchemes = []string{"javascript:", "data:", "vbscript:", "file:"}

// ValidateRedirectURI rejects URIs that use dangerous schemes (javascript:, data:,
// vbscript:, file:) which could turn an open-redirect into code execution.
func ValidateRedirectURI(uri string) error {
	lower := strings.ToLower(strings.TrimSpace(uri))
	for _, scheme := range dangerousSchemes {
		if strings.HasPrefix(lower, scheme) {
			return fmt.Errorf("redirect_uri uses a forbidden scheme: %s", scheme)
		}
	}
	return nil
}

// ValidateIPAddress validates if an IP address is valid and not from restricted ranges
// Complies with SOC2 CC6.1 and ISO27001 A.13.1.1
func ValidateIPAddress(ipStr string) error {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Errorf("invalid IP address format")
	}

	// Check for private/internal IP ranges that shouldn't be allowed
	restrictedRanges := []string{
		"127.0.0.0/8",    // Loopback
		"169.254.0.0/16", // Link-local
		"224.0.0.0/4",    // Multicast
		"240.0.0.0/4",    // Reserved
	}

	for _, cidr := range restrictedRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil && network.Contains(ip) {
			return fmt.Errorf("IP address from restricted range: %s", cidr)
		}
	}

	return nil
}

// ============================================================================
// CRYPTOGRAPHIC UTILITIES
// ============================================================================
// Functions for secure random generation and cryptographic operations

// GenerateCSRFToken generates a cryptographically secure CSRF token.
// Returns an error if the system's random source fails.
// Complies with SOC2 CC6.1 and ISO27001 A.13.2.1
func GenerateCSRFToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("crypto/rand failure: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// ValidateUserAgent checks for suspicious or malicious user agents
// Complies with SOC2 CC7.2 and ISO27001 A.12.4.1
func ValidateUserAgent(userAgent string) bool {
	if userAgent == "" {
		return false
	}

	// Check for suspicious patterns
	suspiciousPatterns := []string{
		"sqlmap", "nikto", "nmap", "masscan", "zap",
		"burp", "w3af", "acunetix", "nessus", "openvas",
		"<script", "javascript:", "vbscript:", "onload=",
	}

	lowerUA := strings.ToLower(userAgent)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerUA, pattern) {
			return false
		}
	}

	return true
}

// RateLimitKey generates a consistent key for rate limiting
// Complies with SOC2 CC6.1 and ISO27001 A.9.4.2
func RateLimitKey(identifier, action string) string {
	return fmt.Sprintf("rate_limit:%s:%s", action, identifier)
}

// dummyBcryptHash is computed lazily for timing-safe operations.
// Complies with SOC2 CC6.1 and ISO27001 A.9.4.2
var (
	dummyBcryptHash []byte
	dummyBcryptOnce sync.Once
)

// GetDummyBcryptHash returns the dummy bcrypt hash for timing-safe operations.
func GetDummyBcryptHash() []byte {
	dummyBcryptOnce.Do(func() {
		// bcrypt.GenerateFromPassword with DefaultCost cannot fail with a valid cost.
		dummyBcryptHash, _ = bcrypt.GenerateFromPassword([]byte("dummy_password_for_timing_safety"), BcryptCost)
	})
	return dummyBcryptHash
}

// ============================================================================
// RATE LIMITING (Redis-backed — works across multiple instances)
// ============================================================================
// SOC2 CC6.1, ISO27001 A.9.4.2

// rateLimiterClient holds the Redis client used for rate limiting.
// Initialised once at startup via InitRateLimiter.
var rateLimiterClient *redis.Client

// OnAccountLockout is an optional hook fired when a user's account is locked
// due to too many failed login attempts and NotifyUserOnLockout is true.
// The identifier is the tenant-scoped key ("tenantID:username").
// Wired during application startup in internal/app.
var OnAccountLockout func(ctx context.Context, identifier string)

// InitRateLimiter wires the Redis client for rate limiting.
// Must be called before CheckRateLimit / RecordFailedAttempt / ResetFailedAttempts.
func InitRateLimiter(rdb *redis.Client) {
	rateLimiterClient = rdb
}

func rateLimitCountKey(identifier string) string {
	return "rl:count:" + identifier
}

func rateLimitLockKey(identifier string) string {
	return "rl:lock:" + identifier
}

// CheckRateLimit returns an error if the identifier is currently locked out.
// Complies with SOC2 CC6.1 and ISO27001 A.9.4.2
func CheckRateLimit(identifier string) error {
	_, span := otel.Tracer("security").Start(context.Background(), "security.check_rate_limit")
	defer span.End()
	span.SetAttributes(attribute.String("identifier", identifier))

	if rateLimiterClient == nil {
		return nil // graceful degradation during unit tests that skip InitRateLimiter
	}

	ctx := context.Background()

	// Check lock key first
	lockVal, err := rateLimiterClient.Get(ctx, rateLimitLockKey(identifier)).Result()
	if err == nil && lockVal != "" {
		// Parse remaining TTL for a useful error message
		ttl, _ := rateLimiterClient.TTL(ctx, rateLimitLockKey(identifier)).Result()
		span.SetStatus(codes.Error, "account locked")
		return fmt.Errorf("account is locked for %v due to too many failed login attempts", ttl.Round(time.Minute))
	}

	// Count check
	countStr, err := rateLimiterClient.Get(ctx, rateLimitCountKey(identifier)).Result()
	if err != nil {
		return nil // key absent ⇒ no attempts yet
	}
	count, _ := strconv.Atoi(countStr)
	if count >= MaxLoginAttempts {
		// Promote to lockout
		_ = rateLimiterClient.Set(ctx, rateLimitLockKey(identifier), "1", AccountLockoutTime).Err()
		_ = rateLimiterClient.Del(ctx, rateLimitCountKey(identifier)).Err()

		LogSecurityEvent(SecurityEvent{
			EventType: "account_locked",
			UserID:    identifier,
			Timestamp: time.Now(),
			Details:   fmt.Sprintf("Account locked after %d failed login attempts", count),
		})

		span.SetStatus(codes.Error, "account locked")
		return fmt.Errorf("account locked for %v due to too many failed login attempts", AccountLockoutTime)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// RecordFailedAttempt increments the failure counter in Redis with a sliding window TTL.
func RecordFailedAttempt(identifier string) {
	_, span := otel.Tracer("security").Start(context.Background(), "security.record_failed_attempt")
	defer span.End()
	span.SetAttributes(attribute.String("identifier", identifier))

	if rateLimiterClient == nil {
		return
	}
	ctx := context.Background()
	key := rateLimitCountKey(identifier)
	pipe := rateLimiterClient.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, LoginAttemptWindow)
	_, _ = pipe.Exec(ctx)
}

// CheckRateLimitWithConfig checks whether identifier is locked out using
// tenant-specific thresholds. Falls back to hardcoded constants when cfg is nil.
func CheckRateLimitWithConfig(identifier string, cfg *RateLimitConfig) error {
	maxAttempts := MaxLoginAttempts
	lockoutDuration := AccountLockoutTime
	observationWindow := LoginAttemptWindow
	if cfg != nil {
		if !cfg.Enabled {
			return nil
		}
		if cfg.MaxFailedAttempts > 0 {
			maxAttempts = cfg.MaxFailedAttempts
		}
		if cfg.LockoutDuration > 0 {
			lockoutDuration = cfg.LockoutDuration
		}
		if cfg.ObservationWindow > 0 {
			observationWindow = cfg.ObservationWindow
		}
	}
	return checkRateLimitWithThresholds(identifier, maxAttempts, lockoutDuration, observationWindow, cfg)
}

func checkRateLimitWithThresholds(identifier string, maxAttempts int, lockoutDuration, observationWindow time.Duration, cfg *RateLimitConfig) error {
	_, span := otel.Tracer("security").Start(context.Background(), "security.check_rate_limit")
	defer span.End()
	span.SetAttributes(attribute.String("identifier", identifier))

	if rateLimiterClient == nil {
		return nil
	}
	ctx := context.Background()

	lockVal, err := rateLimiterClient.Get(ctx, rateLimitLockKey(identifier)).Result()
	if err == nil && lockVal != "" {
		ttl, _ := rateLimiterClient.TTL(ctx, rateLimitLockKey(identifier)).Result()
		span.SetStatus(codes.Error, "account locked")
		if ttl <= 0 {
			return fmt.Errorf("account is locked due to too many failed login attempts")
		}
		return fmt.Errorf("account is locked for %v due to too many failed login attempts", ttl.Round(time.Minute))
	}

	countStr, err := rateLimiterClient.Get(ctx, rateLimitCountKey(identifier)).Result()
	if err != nil {
		return nil
	}
	count, _ := strconv.Atoi(countStr)
	if count >= maxAttempts {
		effectiveDuration := effectiveLockoutDuration(ctx, identifier, lockoutDuration, cfg)
		_ = rateLimiterClient.Set(ctx, rateLimitLockKey(identifier), "1", effectiveDuration).Err()
		_ = rateLimiterClient.Del(ctx, rateLimitCountKey(identifier)).Err()
		details := fmt.Sprintf("Account locked after %d failed login attempts", count)
		if cfg != nil && cfg.NotifyUserOnLockout {
			if OnAccountLockout != nil {
				OnAccountLockout(context.Background(), identifier)
			}
		}
		LogSecurityEvent(SecurityEvent{
			EventType: "account_locked",
			UserID:    identifier,
			Timestamp: time.Now(),
			Details:   details,
		})
		span.SetStatus(codes.Error, "account locked")
		if effectiveDuration <= 0 {
			return fmt.Errorf("account locked due to too many failed login attempts")
		}
		return fmt.Errorf("account locked for %v due to too many failed login attempts", effectiveDuration)
	}
	span.SetStatus(codes.Ok, "")
	return nil
}

func effectiveLockoutDuration(ctx context.Context, identifier string, base time.Duration, cfg *RateLimitConfig) time.Duration {
	if cfg == nil {
		return base
	}
	if !cfg.AutoUnlock {
		return 0
	}
	if !cfg.ProgressiveLockout {
		return base
	}
	tierKey := rateLimitTierKey(identifier)
	tier, err := rateLimiterClient.Incr(ctx, tierKey).Result()
	if err != nil || tier < 1 {
		tier = 1
	}
	resetAfter := cfg.ProgressionReset
	if resetAfter <= 0 {
		resetAfter = cfg.ObservationWindow
	}
	if resetAfter > 0 {
		_ = rateLimiterClient.Expire(ctx, tierKey, resetAfter).Err()
	}
	duration := time.Duration(tier) * base
	if cfg.MaxLockoutDuration > 0 && duration > cfg.MaxLockoutDuration {
		return cfg.MaxLockoutDuration
	}
	return duration
}

// ResetFailedAttemptsWithConfig clears rate-limit state when reset-on-success is enabled.
func ResetFailedAttemptsWithConfig(identifier string, cfg *RateLimitConfig) {
	if cfg != nil && !cfg.ResetCountOnSuccess {
		return
	}
	ResetFailedAttempts(identifier)
}

// ResetFailedAttempts clears all rate-limit state after a successful login.
func ResetFailedAttempts(identifier string) {
	_, span := otel.Tracer("security").Start(context.Background(), "security.reset_failed_attempts")
	defer span.End()
	span.SetAttributes(attribute.String("identifier", identifier))

	if rateLimiterClient == nil {
		return
	}
	ctx := context.Background()
	_ = rateLimiterClient.Del(ctx, rateLimitCountKey(identifier), rateLimitLockKey(identifier)).Err()
	span.SetStatus(codes.Ok, "")
}

func rateLimitTierKey(identifier string) string {
	return "rl:tier:" + identifier
}

var (
	smsBudgetMu       sync.Mutex
	smsBudgetCounters = map[string]int{}
)

func smsDailyBudgetKey(scope string, now time.Time) string {
	return fmt.Sprintf("sms:budget:daily:%s:%s", now.UTC().Format("2006-01-02"), scope)
}

func smsDailyBudgetTTL(now time.Time) time.Duration {
	nextDay := now.UTC().Truncate(24 * time.Hour).Add(25 * time.Hour)
	return time.Until(nextDay)
}

// CheckAndRecordSMSDailyBudget enforces a hard daily SMS send ceiling. The
// Redis path works across pods; the in-memory fallback keeps local/test mode
// from silently disabling the cost guard.
func CheckAndRecordSMSDailyBudget(ctx context.Context, scope string, limit int) error {
	if limit <= 0 {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now()
	key := smsDailyBudgetKey(scope, now)
	if rateLimiterClient != nil {
		count, err := rateLimiterClient.Incr(ctx, key).Result()
		if err == nil {
			if count == 1 {
				_ = rateLimiterClient.Expire(ctx, key, smsDailyBudgetTTL(now)).Err()
			}
			if count > int64(limit) {
				return fmt.Errorf("daily SMS send budget exceeded for %s", scope)
			}
			return nil
		}
	}

	smsBudgetMu.Lock()
	defer smsBudgetMu.Unlock()
	smsBudgetCounters[key]++
	if smsBudgetCounters[key] > limit {
		return fmt.Errorf("daily SMS send budget exceeded for %s", scope)
	}
	return nil
}

// ResetSMSDailyBudgetCounters clears process-local SMS budget counters. It is
// intended for tests and does not affect Redis-backed counters.
func ResetSMSDailyBudgetCounters() {
	smsBudgetMu.Lock()
	defer smsBudgetMu.Unlock()
	smsBudgetCounters = map[string]int{}
}

// ============================================================================
// SESSION MANAGEMENT
// ============================================================================
// Functions for managing user sessions and concurrent session limits

// ValidateSessionLimit checks if user has exceeded maximum concurrent sessions
// This can be used by services that need to enforce session limits
// Complies with SOC2 CC6.3 and ISO27001 A.9.4.2
func ValidateSessionLimit(userID string, currentSessionCount int) error {
	_, span := otel.Tracer("security").Start(context.Background(), "security.validate_session_limit")
	defer span.End()
	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.Int("session_count", currentSessionCount),
	)

	if currentSessionCount >= MaxConcurrentSessions {
		span.SetStatus(codes.Error, "session limit exceeded")
		return fmt.Errorf("maximum concurrent sessions (%d) exceeded for user", MaxConcurrentSessions)
	}
	span.SetStatus(codes.Ok, "")
	return nil
}

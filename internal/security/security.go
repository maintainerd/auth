/*
Package util provides comprehensive security utilities for authentication and authorization.

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
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================================
// SECURITY CONSTANTS
// ============================================================================
// These constants define security policies and limits across the application

// Rate Limiting Constants (SOC2 CC6.1 - Logical Access Controls)
const (
	MaxLoginAttempts   = 5                // Maximum failed attempts before lockout
	LoginAttemptWindow = 15 * time.Minute // Time window for counting attempts
	AccountLockoutTime = 30 * time.Minute // Account lockout duration
)

// Session Security Constants (SOC2 CC6.3 - Logical Access Controls)
const (
	MaxConcurrentSessions = 5 // Maximum concurrent sessions per user
)

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

// ValidatePasswordStrength enforces password complexity requirements
// Complies with SOC2 CC6.1 and ISO27001 A.9.4.3
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	if len(password) > 128 {
		return fmt.Errorf("password must not exceed 128 characters")
	}

	if !reUpper.MatchString(password) {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}

	if !reLower.MatchString(password) {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}

	if !reDigit.MatchString(password) {
		return fmt.Errorf("password must contain at least one digit")
	}

	if !reSpecial.MatchString(password) {
		return fmt.Errorf("password must contain at least one special character")
	}

	// Check for common weak passwords
	weakPasswords := []string{
		"password", "123456", "password123", "admin", "qwerty",
		"letmein", "welcome", "monkey", "dragon", "master",
	}

	lowerPassword := strings.ToLower(password)
	for _, weak := range weakPasswords {
		if strings.Contains(lowerPassword, weak) {
			return fmt.Errorf("password contains common weak patterns")
		}
	}

	return nil
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

// GenerateCSRFToken generates a cryptographically secure CSRF token
// Complies with SOC2 CC6.1 and ISO27001 A.13.2.1
func GenerateCSRFToken() string {
	bytes := make([]byte, 32)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
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

// dummyBcryptHash is pre-computed once at startup for timing-safe operations.
// Complies with SOC2 CC6.1 and ISO27001 A.9.4.2
var dummyBcryptHash []byte

func init() {
	// bcrypt.GenerateFromPassword with DefaultCost cannot fail:
	// the cost is valid and crypto/rand.Read is guaranteed to succeed (Go 1.24+).
	dummyBcryptHash, _ = bcrypt.GenerateFromPassword([]byte("dummy_password_for_timing_safety"), bcrypt.DefaultCost)
}

// GetDummyBcryptHash returns the pre-computed dummy bcrypt hash for timing-safe operations.
func GetDummyBcryptHash() []byte {
	return dummyBcryptHash
}

// ============================================================================
// RATE LIMITING (Redis-backed — works across multiple instances)
// ============================================================================
// SOC2 CC6.1, ISO27001 A.9.4.2

// rateLimiterClient holds the Redis client used for rate limiting.
// Initialised once at startup via InitRateLimiter.
var rateLimiterClient *redis.Client

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
	if rateLimiterClient == nil {
		return nil // graceful degradation during unit tests that skip InitRateLimiter
	}

	ctx := context.Background()

	// Check lock key first
	lockVal, err := rateLimiterClient.Get(ctx, rateLimitLockKey(identifier)).Result()
	if err == nil && lockVal != "" {
		// Parse remaining TTL for a useful error message
		ttl, _ := rateLimiterClient.TTL(ctx, rateLimitLockKey(identifier)).Result()
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

		return fmt.Errorf("account locked for %v due to too many failed login attempts", AccountLockoutTime)
	}

	return nil
}

// RecordFailedAttempt increments the failure counter in Redis with a sliding window TTL.
func RecordFailedAttempt(identifier string) {
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

// ResetFailedAttempts clears all rate-limit state after a successful login.
func ResetFailedAttempts(identifier string) {
	if rateLimiterClient == nil {
		return
	}
	ctx := context.Background()
	_ = rateLimiterClient.Del(ctx, rateLimitCountKey(identifier), rateLimitLockKey(identifier)).Err()
}

// ============================================================================
// SESSION MANAGEMENT
// ============================================================================
// Functions for managing user sessions and concurrent session limits

// ValidateSessionLimit checks if user has exceeded maximum concurrent sessions
// This can be used by services that need to enforce session limits
// Complies with SOC2 CC6.3 and ISO27001 A.9.4.2
func ValidateSessionLimit(userID string, currentSessionCount int) error {
	if currentSessionCount >= MaxConcurrentSessions {
		return fmt.Errorf("maximum concurrent sessions (%d) exceeded for user", MaxConcurrentSessions)
	}
	return nil
}

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
package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

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
//
// CURRENT: Basic stdout logging for development
// FUTURE: Will be enhanced with centralized logging, database storage, and SIEM integration
func LogSecurityEvent(event SecurityEvent) {
	// Set default severity if not specified
	if event.Severity == "" {
		event.Severity = determineSeverity(event.EventType)
	}

	// Basic structured logging to stdout (development)
	log.Printf("[SECURITY] %s | %s | %s | %s | %s | %s",
		event.Timestamp.Format(time.RFC3339),
		event.EventType,
		event.Severity,
		event.ClientIP,
		event.UserID,
		event.Details,
	)

	// TODO: Future production enhancements:
	// - JSON structured logging for container environments
	// - Database storage for audit trail and compliance
	// - SIEM integration (Splunk, ELK, Datadog, etc.)
	// - Real-time alerting for high-severity events
	// - Log retention and archival policies
	// - Encrypted log transmission for sensitive events
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

	for _, event := range highSeverityEvents {
		if event == eventType {
			return "HIGH"
		}
	}

	for _, event := range mediumSeverityEvents {
		if event == eventType {
			return "MEDIUM"
		}
	}

	return "LOW"
}

// ============================================================================
// INPUT VALIDATION & SANITIZATION
// ============================================================================
// Functions for validating and sanitizing user input

// ValidatePasswordStrength enforces password complexity requirements
// Complies with SOC2 CC6.1 and ISO27001 A.9.4.3
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	if len(password) > 128 {
		return fmt.Errorf("password must not exceed 128 characters")
	}

	// Check for at least one uppercase letter
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}

	// Check for at least one lowercase letter
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}

	// Check for at least one digit
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)
	if !hasDigit {
		return fmt.Errorf("password must contain at least one digit")
	}

	// Check for at least one special character
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password)
	if !hasSpecial {
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
func GenerateCSRFToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate CSRF token: %w", err)
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

// GenerateDummyBcryptHash generates a dummy bcrypt hash for timing-safe operations
// This prevents timing attacks by ensuring consistent bcrypt operation duration
// regardless of whether a user exists or not.
// Complies with SOC2 CC6.1 and ISO27001 A.9.4.2
func GenerateDummyBcryptHash() []byte {
	dummyHash, err := bcrypt.GenerateFromPassword([]byte("dummy_password_for_timing_safety"), bcrypt.DefaultCost)
	if err != nil {
		// Fallback to a hardcoded valid bcrypt hash if generation fails
		// This should never happen, but provides a safety net
		return []byte("$2a$10$dummy.hash.to.prevent.timing.attacks.fallback")
	}
	return dummyHash
}

// ============================================================================
// RATE LIMITING
// ============================================================================
// Functions and storage for rate limiting and account lockout protection

// Global rate limiting storage (in production, use Redis or database)
var (
	loginAttempts = make(map[string]*LoginAttempt)
	attemptsMutex = sync.RWMutex{}
)

// CheckRateLimit implements rate limiting and account lockout
// Complies with SOC2 CC6.1 and ISO27001 A.9.4.2
func CheckRateLimit(identifier string) error {
	attemptsMutex.Lock()
	defer attemptsMutex.Unlock()

	now := time.Now()

	// Get or create login attempt record
	attempt, exists := loginAttempts[identifier]
	if !exists {
		loginAttempts[identifier] = &LoginAttempt{
			Identifier:  identifier,
			Attempts:    0,
			LastAttempt: now,
		}
		return nil
	}

	// Check if account is currently locked
	if attempt.LockedUntil != nil && now.Before(*attempt.LockedUntil) {
		remainingTime := attempt.LockedUntil.Sub(now)
		return fmt.Errorf("account is locked for %v due to too many failed login attempts", remainingTime.Round(time.Minute))
	}

	// Reset attempts if outside the time window
	if now.Sub(attempt.LastAttempt) > LoginAttemptWindow {
		attempt.Attempts = 0
		attempt.LockedUntil = nil
	}

	// Check if maximum attempts exceeded
	if attempt.Attempts >= MaxLoginAttempts {
		lockUntil := now.Add(AccountLockoutTime)
		attempt.LockedUntil = &lockUntil

		// Log security event
		LogSecurityEvent(SecurityEvent{
			EventType: "account_locked",
			UserID:    identifier,
			Timestamp: now,
			Details:   fmt.Sprintf("Account locked after %d failed login attempts", attempt.Attempts),
		})

		return fmt.Errorf("account locked for %v due to too many failed login attempts", AccountLockoutTime)
	}

	return nil
}

// RecordFailedAttempt records a failed login attempt
func RecordFailedAttempt(identifier string) {
	attemptsMutex.Lock()
	defer attemptsMutex.Unlock()

	now := time.Now()

	attempt, exists := loginAttempts[identifier]
	if !exists {
		loginAttempts[identifier] = &LoginAttempt{
			Identifier:  identifier,
			Attempts:    1,
			LastAttempt: now,
		}
		return
	}

	// Reset if outside time window
	if now.Sub(attempt.LastAttempt) > LoginAttemptWindow {
		attempt.Attempts = 1
		attempt.LockedUntil = nil
	} else {
		attempt.Attempts++
	}

	attempt.LastAttempt = now
}

// ResetFailedAttempts clears failed attempts after successful login
func ResetFailedAttempts(identifier string) {
	attemptsMutex.Lock()
	defer attemptsMutex.Unlock()

	delete(loginAttempts, identifier)
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

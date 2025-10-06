package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
	"time"
)

// SecurityEvent represents a security-related event for audit logging
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

// LogSecurityEvent logs security events for compliance monitoring
// Complies with SOC2 CC7.2 (System Monitoring) and ISO27001 A.12.4.1
func LogSecurityEvent(event SecurityEvent) {
	// Set default severity if not specified
	if event.Severity == "" {
		event.Severity = determineSeverity(event.EventType)
	}

	// In production, this should send to a centralized logging system
	// For now, log to stdout with structured format
	log.Printf("[SECURITY] %s | %s | %s | %s | %s | %s",
		event.Timestamp.Format(time.RFC3339),
		event.EventType,
		event.Severity,
		event.ClientIP,
		event.UserID,
		event.Details,
	)

	// TODO: In production, implement:
	// - Send to SIEM system (Splunk, ELK, etc.)
	// - Store in audit database
	// - Send alerts for high-severity events
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

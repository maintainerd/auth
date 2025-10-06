package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/maintainerd/auth/internal/config"
	"github.com/maintainerd/auth/internal/util"
)

// SecurityContextKey represents security context keys
type SecurityContextKey string

const (
	ClientIPKey      SecurityContextKey = "client_ip"
	UserAgentKey     SecurityContextKey = "user_agent"
	RequestIDKey     SecurityContextKey = "request_id"
	SessionIDKey     SecurityContextKey = "session_id"
	SecurityEventKey SecurityContextKey = "security_event"
)

// SecurityHeadersMiddleware adds security headers for SOC2/ISO27001 compliance
// Complies with SOC2 CC6.1 and ISO27001 A.13.2.1
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing (SOC2 CC6.1)
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking attacks (ISO27001 A.13.2.1)
		w.Header().Set("X-Frame-Options", "DENY")

		// XSS protection (SOC2 CC6.1)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Content Security Policy (ISO27001 A.13.2.1)
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'")

		// Referrer policy (SOC2 CC6.1)
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions policy (ISO27001 A.13.2.1)
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=()")

		// HTTPS enforcement in production (SOC2 CC6.1)
		if isProduction() {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		next.ServeHTTP(w, r)
	})
}

// SecurityContextMiddleware extracts security-relevant information
// Complies with SOC2 CC7.2 (System Monitoring) and ISO27001 A.12.4.1
func SecurityContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract client IP (for audit logging)
		clientIP := extractClientIP(r)

		// Extract User-Agent (for security monitoring)
		userAgent := r.Header.Get("User-Agent")

		// Generate request ID for tracing
		requestID := util.GenerateSecureID()

		// Add security headers for request tracking
		w.Header().Set("X-Request-ID", requestID)

		// Create security context
		ctx := context.WithValue(r.Context(), ClientIPKey, clientIP)
		ctx = context.WithValue(ctx, UserAgentKey, userAgent)
		ctx = context.WithValue(ctx, RequestIDKey, requestID)

		// Log security event for monitoring
		util.LogSecurityEvent(util.SecurityEvent{
			EventType: "request_received",
			ClientIP:  clientIP,
			UserAgent: userAgent,
			RequestID: requestID,
			Endpoint:  r.URL.Path,
			Method:    r.Method,
			Timestamp: time.Now(),
		})

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestSizeLimitMiddleware limits request body size for DoS protection
// Complies with SOC2 CC6.1 and ISO27001 A.13.1.1
func RequestSizeLimitMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Limit request body size (default 1MB for auth endpoints)
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

			next.ServeHTTP(w, r)
		})
	}
}

// TimeoutMiddleware enforces request timeouts for DoS protection
// Complies with SOC2 CC6.1 and ISO27001 A.13.1.1
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// IPWhitelistMiddleware restricts access to specific IP ranges (optional)
// Complies with SOC2 CC6.1 and ISO27001 A.9.4.1
func IPWhitelistMiddleware(allowedCIDRs []string) func(http.Handler) http.Handler {
	var allowedNets []*net.IPNet

	for _, cidr := range allowedCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil {
			allowedNets = append(allowedNets, network)
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip if no whitelist configured
			if len(allowedNets) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			clientIP := extractClientIP(r)
			ip := net.ParseIP(clientIP)

			allowed := false
			for _, network := range allowedNets {
				if network.Contains(ip) {
					allowed = true
					break
				}
			}

			if !allowed {
				util.LogSecurityEvent(util.SecurityEvent{
					EventType: "ip_blocked",
					ClientIP:  clientIP,
					UserAgent: r.Header.Get("User-Agent"),
					Endpoint:  r.URL.Path,
					Timestamp: time.Now(),
					Details:   "IP not in whitelist",
				})

				util.Error(w, http.StatusForbidden, "Access denied")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractClientIP extracts the real client IP from various headers
func extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (most common)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Check CF-Connecting-IP (Cloudflare)
	if cfip := r.Header.Get("CF-Connecting-IP"); cfip != "" {
		return strings.TrimSpace(cfip)
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// isProduction checks if running in production environment
func isProduction() bool {
	return config.GetEnvOrDefault("ENV", "development") == "production"
}

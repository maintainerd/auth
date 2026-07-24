package middleware

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
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
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'self'; upgrade-insecure-requests")

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
		requestID := jwt.GenerateSecureID()

		// Add security headers for request tracking
		w.Header().Set("X-Request-ID", requestID)

		// Create security context
		ctx := context.WithValue(r.Context(), ClientIPKey, clientIP)
		ctx = context.WithValue(ctx, UserAgentKey, userAgent)
		ctx = context.WithValue(ctx, RequestIDKey, requestID)

		// Log security event for monitoring
		security.LogSecurityEvent(security.SecurityEvent{
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
				security.LogSecurityEvent(security.SecurityEvent{
					EventType: "ip_blocked",
					ClientIP:  clientIP,
					UserAgent: r.Header.Get("User-Agent"),
					Endpoint:  r.URL.Path,
					Timestamp: time.Now(),
					Details:   "IP not in whitelist",
				})

				resp.Error(w, http.StatusForbidden, "Access denied")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Trusted-proxy configuration for client-IP resolution.
//
// TRUSTED_PROXY_CIDRS is a comma-separated list of CIDRs (or bare IPs) that are
// permitted to speak for the client via forwarding headers — normally just the
// reverse proxy in front of the app. Defaults to loopback plus the RFC1918 and
// RFC4193 ranges, which covers the nginx-in-docker topology this project ships
// with (see CLAUDE.md) while still refusing headers from the public internet.
//
// Set TRUST_ALL_PROXIES=true ONLY when the platform guarantees the header (e.g.
// a managed load balancer that overwrites X-Forwarded-For). It restores the old
// trust-anything behaviour and is logged as a warning at startup.
var (
	trustedProxyNets []*net.IPNet
	trustAllProxies  bool
	trustedProxyOnce sync.Once
)

const defaultTrustedProxyCIDRs = "127.0.0.0/8,::1/128,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,fc00::/7"

func initTrustedProxies() {
	trustAllProxies = strings.EqualFold(strings.TrimSpace(os.Getenv("TRUST_ALL_PROXIES")), "true")
	if trustAllProxies {
		slog.Warn("TRUST_ALL_PROXIES is enabled: forwarding headers are accepted from any peer, " +
			"so every per-IP rate limit and IP restriction can be bypassed by a spoofed header")
		return
	}

	raw := strings.TrimSpace(os.Getenv("TRUSTED_PROXY_CIDRS"))
	if raw == "" {
		raw = defaultTrustedProxyCIDRs
	}
	for entry := range strings.SplitSeq(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if !strings.Contains(entry, "/") {
			// Bare IP — treat as a single-host network.
			if ip := net.ParseIP(entry); ip != nil {
				bits := 32
				if ip.To4() == nil {
					bits = 128
				}
				trustedProxyNets = append(trustedProxyNets, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
				continue
			}
			slog.Error("ignoring invalid TRUSTED_PROXY_CIDRS entry", "entry", entry)
			continue
		}
		_, network, err := net.ParseCIDR(entry)
		if err != nil {
			slog.Error("ignoring invalid TRUSTED_PROXY_CIDRS entry", "entry", entry, "error", err)
			continue
		}
		trustedProxyNets = append(trustedProxyNets, network)
	}
}

func isTrustedProxy(ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, network := range trustedProxyNets {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// extractClientIP resolves the client IP that every per-IP control keys on —
// rate limiting, registration abuse counters, tenant IP restrictions and audit
// records.
//
// Forwarding headers are attacker-controlled unless the immediate peer is a
// trusted proxy, so they are consulted ONLY when r.RemoteAddr is in
// TRUSTED_PROXY_CIDRS. Otherwise RemoteAddr wins. Validating a header's *syntax*
// (net.ParseIP) proves nothing about its provenance: without this check any
// caller can rotate `X-Forwarded-For` per request and reset every limiter.
//
// Within X-Forwarded-For the list is appended left-to-right by each hop, so the
// rightmost entry is the one our trusted proxy observed. We walk from the right
// and take the first entry that is not itself a trusted proxy, which yields the
// real client even behind several trusted hops and cannot be shifted by a
// client-supplied prefix.
func extractClientIP(r *http.Request) string {
	trustedProxyOnce.Do(initTrustedProxies)

	remoteIP := remoteAddrIP(r)

	if !trustAllProxies && !isTrustedProxy(remoteIP) {
		if remoteIP != nil {
			return remoteIP.String()
		}
		return r.RemoteAddr
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			candidate := net.ParseIP(strings.TrimSpace(parts[i]))
			if candidate == nil {
				continue
			}
			if isTrustedProxy(candidate) && !trustAllProxies {
				continue // another hop in our own infrastructure
			}
			return candidate.String()
		}
	}

	if xri := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); xri != nil {
		return xri.String()
	}

	if cfip := net.ParseIP(strings.TrimSpace(r.Header.Get("CF-Connecting-IP"))); cfip != nil {
		return cfip.String()
	}

	if remoteIP != nil {
		return remoteIP.String()
	}
	return r.RemoteAddr
}

func remoteAddrIP(r *http.Request) net.IP {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return net.ParseIP(strings.TrimSpace(r.RemoteAddr))
	}
	return net.ParseIP(host)
}

// ClientIPFromContext returns the client IP address stored in ctx by
// SecurityContextMiddleware. Returns an empty string when the value is absent.
func ClientIPFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ClientIPKey).(string); ok {
		return v
	}
	return ""
}

// UserAgentFromContext returns the User-Agent string stored in ctx by
// SecurityContextMiddleware. Returns an empty string when the value is absent.
func UserAgentFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(UserAgentKey).(string); ok {
		return v
	}
	return ""
}

// isProduction checks if running in production environment
func isProduction() bool {
	return config.GetEnvOrDefault("ENV", "development") == "production"
}

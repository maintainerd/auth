package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// OnNewDeviceLogin / OnImpossibleTravel are optional notification hooks fired
// when the corresponding threat signal is observed and its config toggle is on.
//
// LogSecurityEvent alone is a local slog line that notifies nobody — yet the
// field is literally named new_device_NOTIFICATION_enabled. These hooks let the
// application route the signal into the auth-event system (which fans out to
// webhooks and can drive a user notification), so the "notification" is real.
// Wired at startup in internal/app, mirroring security.OnAccountLockout. Nil is
// safe (no-op): the slog line still records the signal.
var (
	OnNewDeviceLogin   func(ctx context.Context, tenantID, userID int64, ip, userAgent string)
	OnImpossibleTravel func(ctx context.Context, tenantID, userID int64, ip, userAgent string)
)

// ThreatConfig is the effective tenant threat-detection runtime policy.
type ThreatConfig struct {
	BruteForceDetectionEnabled             bool
	ImpossibleTravelDetectionEnabled       bool
	NewDeviceNotificationEnabled           bool
	VelocityCheckEnabled                   bool
	RiskBasedStepUpEnabled                 bool
	CompromisedCredentialMonitoringEnabled bool
	IPReputationCheckEnabled               bool
	BlockTorExitNodes                      bool
	RiskStepUpThreshold                    int
	RiskBlockThreshold                     int
	VelocityFailuresPerIPPerHour           int
}

// ThreatDecision is returned by AssessLoginThreat before password validation.
type ThreatDecision struct {
	RiskScore      int
	RequiresStepUp bool
	Blocked        bool
	Reason         string
}

func registrationRateLimitKey(tenantID int64, ip string) string {
	return fmt.Sprintf("registration:rate:%d:%s", tenantID, strings.TrimSpace(ip))
}

// CheckRegistrationRateLimit records a registration attempt and enforces a
// per-IP hourly cap. Missing Redis is treated as fail-open, matching the older
// login limiter behavior in tests/local dev.
func CheckRegistrationRateLimit(ctx context.Context, tenantID int64, ip string, perHour int) error {
	if perHour <= 0 || rateLimiterClient == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ip = strings.TrimSpace(ip)
	if ip == "" {
		ip = "unknown"
	}
	key := registrationRateLimitKey(tenantID, ip)
	count, err := rateLimiterClient.Incr(ctx, key).Result()
	if err != nil {
		return nil
	}
	if count == 1 {
		_ = rateLimiterClient.Expire(ctx, key, time.Hour).Err()
	}
	if count > int64(perHour) {
		return fmt.Errorf("registration rate limit exceeded")
	}
	return nil
}

func threatFailureKey(tenantID int64, ip string) string {
	return fmt.Sprintf("threat:failures:%d:%s", tenantID, strings.TrimSpace(ip))
}

func threatDeviceKey(tenantID, userID int64, fingerprint string) string {
	return fmt.Sprintf("threat:device:%d:%d:%s", tenantID, userID, fingerprint)
}

func threatLastLoginKey(tenantID, userID int64) string {
	return fmt.Sprintf("threat:last_login:%d:%d", tenantID, userID)
}

func threatDeviceFingerprint(ip, userAgent string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(ip) + "\x00" + strings.TrimSpace(userAgent)))
	return hex.EncodeToString(sum[:])
}

// AssessLoginThreat evaluates pre-auth signals that are available before the
// user is known. The current implementation covers per-IP velocity/brute-force
// thresholds and feeds a risk score for future step-up decisions.
func AssessLoginThreat(ctx context.Context, tenantID int64, ip, userAgent string, cfg *ThreatConfig) ThreatDecision {
	_ = userAgent
	if cfg == nil || rateLimiterClient == nil {
		return ThreatDecision{}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	limit := cfg.VelocityFailuresPerIPPerHour
	if limit <= 0 {
		limit = 20
	}
	score := 0
	reason := ""
	if cfg.BruteForceDetectionEnabled || cfg.VelocityCheckEnabled {
		count, err := rateLimiterClient.Get(ctx, threatFailureKey(tenantID, ip)).Int()
		if err == nil && count > 0 {
			switch {
			case count >= limit:
				score = max(score, 100)
				reason = "velocity threshold exceeded"
			case count >= limit/2:
				score = max(score, 50)
				reason = "elevated failure velocity"
			}
		}
	}
	blockThreshold := cfg.RiskBlockThreshold
	if blockThreshold <= 0 {
		blockThreshold = 90
	}
	stepUpThreshold := cfg.RiskStepUpThreshold
	if stepUpThreshold <= 0 {
		stepUpThreshold = 50
	}
	return ThreatDecision{
		RiskScore:      score,
		RequiresStepUp: cfg.RiskBasedStepUpEnabled && score >= stepUpThreshold,
		Blocked:        score >= blockThreshold,
		Reason:         reason,
	}
}

// RecordLoginThreatFailure increments the per-IP failure velocity counter.
func RecordLoginThreatFailure(ctx context.Context, tenantID int64, ip string, cfg *ThreatConfig) {
	if cfg == nil || rateLimiterClient == nil || (!cfg.BruteForceDetectionEnabled && !cfg.VelocityCheckEnabled) {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	key := threatFailureKey(tenantID, ip)
	pipe := rateLimiterClient.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, time.Hour)
	_, _ = pipe.Exec(ctx)
}

// RecordLoginThreatSuccess clears velocity failures and records device/last
// login signals used by new-device and impossible-travel detections.
func RecordLoginThreatSuccess(ctx context.Context, tenantID, userID int64, ip, userAgent string, cfg *ThreatConfig) {
	if cfg == nil || rateLimiterClient == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg.BruteForceDetectionEnabled || cfg.VelocityCheckEnabled {
		_ = rateLimiterClient.Del(ctx, threatFailureKey(tenantID, ip)).Err()
	}
	if cfg.NewDeviceNotificationEnabled {
		fingerprint := threatDeviceFingerprint(ip, userAgent)
		key := threatDeviceKey(tenantID, userID, fingerprint)
		_, err := rateLimiterClient.SetArgs(ctx, key, time.Now().UTC().Format(time.RFC3339), redis.SetArgs{Mode: "NX", TTL: 365 * 24 * time.Hour}).Result()
		isNew := err == nil
		if err == redis.Nil { // key already existed; not new, not an error
			isNew = false
			err = nil
		}
		if err == nil && isNew {
			LogSecurityEvent(SecurityEvent{
				EventType: "new_device_login",
				UserID:    fmt.Sprintf("%d", userID),
				ClientIP:  ip,
				UserAgent: userAgent,
				Timestamp: time.Now(),
				Details:   "New device fingerprint observed for user",
				Severity:  "MEDIUM",
			})
			if OnNewDeviceLogin != nil {
				OnNewDeviceLogin(ctx, tenantID, userID, ip, userAgent)
			}
		}
	}
	if cfg.ImpossibleTravelDetectionEnabled {
		lastKey := threatLastLoginKey(tenantID, userID)
		if last, err := rateLimiterClient.Get(ctx, lastKey).Result(); err == nil && last != "" {
			parts := strings.SplitN(last, "|", 2)
			if len(parts) == 2 && parts[1] != ip {
				if lastAt, parseErr := time.Parse(time.RFC3339, parts[0]); parseErr == nil && time.Since(lastAt) < 15*time.Minute {
					LogSecurityEvent(SecurityEvent{
						EventType: "impossible_travel_signal",
						UserID:    fmt.Sprintf("%d", userID),
						ClientIP:  ip,
						UserAgent: userAgent,
						Timestamp: time.Now(),
						Details:   "Rapid login from a different IP address",
						Severity:  "HIGH",
					})
					if OnImpossibleTravel != nil {
						OnImpossibleTravel(ctx, tenantID, userID, ip, userAgent)
					}
				}
			}
		}
		_ = rateLimiterClient.Set(ctx, lastKey, time.Now().UTC().Format(time.RFC3339)+"|"+ip, 30*24*time.Hour).Err()
	}
}

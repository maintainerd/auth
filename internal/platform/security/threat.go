package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
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
	DistinctAccountsPerIPPerHour           int
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

// threatDistinctAccountsKey is a per-IP HyperLogLog of the DISTINCT accounts
// that saw a failed login from this IP within the window. Cardinality — not raw
// volume — is the credential-stuffing fingerprint (one host spraying many
// usernames), and HLL keeps it to ~12KB regardless of how many usernames an
// attacker sprays.
func threatDistinctAccountsKey(tenantID int64, ip string) string {
	return fmt.Sprintf("threat:distinct_accts:%d:%s", tenantID, strings.TrimSpace(ip))
}

// threatIdentifierHash normalizes and hashes a login identifier before it is
// used as an HLL element, so raw emails/usernames never land in Redis.
func threatIdentifierHash(identifier string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(identifier))))
	return hex.EncodeToString(sum[:])
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
		limit = 50
	}
	distinctLimit := cfg.DistinctAccountsPerIPPerHour
	if distinctLimit <= 0 {
		distinctLimit = 10
	}

	// The risk model reads two independent per-IP signals and takes the higher
	// score. They deliberately do different jobs, and neither overlaps the
	// per-account lockout (which owns "one account hammered many times"):
	//
	//   VOLUME (raw per-IP failure count) is an AMBIGUOUS signal — a busy shared
	//   NAT/CGNAT/VPN egress produces volume just like an attacker does — so on
	//   its own it only ever elevates to the step-up band, never a hard block. A
	//   legitimate user behind a hammered IP is merely asked to prove a second
	//   factor; a bot cannot.
	//
	//   DISTINCT-ACCOUNT FAN-OUT (how many distinct accounts one IP has failed
	//   against) is the credential-stuffing fingerprint that lockout structurally
	//   cannot see — each account gets too few tries to lock. High fan-out is what
	//   justifies escalating to a hard block.
	volumeScore, volumeReason := 0, ""
	distinctScore, distinctReason := 0, ""
	if cfg.BruteForceDetectionEnabled || cfg.VelocityCheckEnabled {
		if count, err := rateLimiterClient.Get(ctx, threatFailureKey(tenantID, ip)).Int(); err == nil && count > 0 {
			switch {
			case count >= limit:
				volumeScore, volumeReason = 60, "elevated failure velocity from this IP"
			case count >= limit/2:
				volumeScore, volumeReason = 40, "elevated failure velocity from this IP"
			}
		} else if err != nil && err != redis.Nil {
			logVelocityDegraded(ctx, tenantID, err)
		}

		if distinct, err := rateLimiterClient.PFCount(ctx, threatDistinctAccountsKey(tenantID, ip)).Result(); err == nil && distinct > 0 {
			half := int64(distinctLimit) / 2
			switch {
			case distinct >= int64(distinctLimit)*2:
				distinctScore, distinctReason = 100, "credential stuffing: many distinct accounts targeted from one IP"
			case distinct >= int64(distinctLimit):
				distinctScore, distinctReason = 75, "many distinct accounts targeted from one IP"
			case half > 0 && distinct >= half:
				distinctScore, distinctReason = 50, "elevated distinct-account failures from one IP"
			}
		} else if err != nil && err != redis.Nil {
			logVelocityDegraded(ctx, tenantID, err)
		}
	}
	// Take the higher of the two signals; on a tie prefer the distinct-account
	// reason — it is the more specific and actionable of the two.
	score, reason := volumeScore, volumeReason
	if distinctScore >= score {
		score, reason = distinctScore, distinctReason
	}

	// Fallbacks mirror the seeded config defaults (see secpolicy threat defaults)
	// so an unconfigured policy behaves identically to the shipped one.
	blockThreshold := cfg.RiskBlockThreshold
	if blockThreshold <= 0 {
		blockThreshold = 81
	}
	stepUpThreshold := cfg.RiskStepUpThreshold
	if stepUpThreshold <= 0 {
		stepUpThreshold = 21
	}
	return ThreatDecision{
		RiskScore:      score,
		RequiresStepUp: cfg.RiskBasedStepUpEnabled && score >= stepUpThreshold,
		Blocked:        score >= blockThreshold,
		Reason:         reason,
	}
}

// logVelocityDegraded records that a per-IP velocity signal could not be read
// from the store, so this login is being assessed as low-risk purely because
// the counter is invisible — a fail-open worth surfacing to operators.
func logVelocityDegraded(ctx context.Context, tenantID int64, err error) {
	slog.WarnContext(ctx, "login threat velocity check degraded: failure-velocity store unavailable, failing open",
		"tenant_id", tenantID, "error", err)
}

// RecordLoginThreatFailure records a failed login against both per-IP signals:
// the raw failure-VOLUME counter and the DISTINCT-ACCOUNT fan-out set (see
// AssessLoginThreat for how each is scored). identifier is the account the login
// was attempted against; it is hashed before use so no raw email/username is
// stored. Both keys carry a rolling one-hour TTL and are never reset by a
// success (an aggregate abuse signal must decay on its own, not be cleared by a
// single valid login — see RecordLoginThreatSuccess).
func RecordLoginThreatFailure(ctx context.Context, tenantID int64, ip, identifier string, cfg *ThreatConfig) {
	if cfg == nil || rateLimiterClient == nil || (!cfg.BruteForceDetectionEnabled && !cfg.VelocityCheckEnabled) {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	pipe := rateLimiterClient.Pipeline()
	volumeKey := threatFailureKey(tenantID, ip)
	pipe.Incr(ctx, volumeKey)
	pipe.Expire(ctx, volumeKey, time.Hour)
	if id := strings.TrimSpace(identifier); id != "" {
		distinctKey := threatDistinctAccountsKey(tenantID, ip)
		pipe.PFAdd(ctx, distinctKey, threatIdentifierHash(id))
		pipe.Expire(ctx, distinctKey, time.Hour)
	}
	_, _ = pipe.Exec(ctx)
}

// RecordLoginThreatSuccess records device/last-login signals used by
// new-device and impossible-travel detections after a successful login.
//
// It deliberately does NOT clear the per-IP failure-velocity counter. That
// counter is an aggregate credential-stuffing signal (many accounts, one IP),
// so a single successful login must not reset it: an attacker only needs ONE
// valid credential of their own to zero the shared counter and un-flag every
// other account they are stuffing from the same host. The counter instead
// decays naturally via its rolling one-hour TTL (see RecordLoginThreatFailure).
// Rewarding a legitimate user's success is the per-ACCOUNT lockout's job (it
// clears that user's own failure state), which is a separate, account-scoped
// signal from this IP-scoped one.
func RecordLoginThreatSuccess(ctx context.Context, tenantID, userID int64, ip, userAgent string, cfg *ThreatConfig) {
	if cfg == nil || rateLimiterClient == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
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

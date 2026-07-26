package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// authEventCounter is the auth-domain event counter (auth_events_total). It is
// initialized by InitMetrics against the Prometheus-backed MeterProvider and
// consumed by RecordAuthEvent. It stays nil when metrics were never
// initialized (e.g. in unit tests), in which case RecordAuthEvent is a no-op.
var authEventCounter metric.Int64Counter

// RecordAuthEvent increments the auth-domain event counter, labeled by
// category, event type, and result. This is the single metering hook for
// authentication/authorization activity (logins, token issuance/revocation,
// lockouts, OAuth authorize/consent, etc.); it is called from the central
// auth-event Log path so every recorded auth event is also metered.
//
// It is safe to call before InitMetrics and safe under concurrency; when the
// counter is not initialized it does nothing.
func RecordAuthEvent(ctx context.Context, category, eventType, result string) {
	if authEventCounter == nil {
		return
	}
	authEventCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("category", category),
		attribute.String("event_type", eventType),
		attribute.String("result", result),
	))
}

// securityDenialCounter counts access denials at the middleware boundary
// (security_denials_total), labeled by denial type. It is the signal a
// monitoring consumer alerts on for probing / brute-force / policy-violation
// activity. Nil until InitMetrics runs (no-op in tests).
var securityDenialCounter metric.Int64Counter

// Denial-type labels for RecordSecurityDenial (kept low-cardinality on purpose —
// no per-tenant/per-IP labels, which would explode Prometheus cardinality).
const (
	DenialPermission = "permission_denied"
	DenialRateLimit  = "rate_limited"
	DenialIPBlocked  = "ip_blocked"
)

// RecordSecurityDenial increments the access-denial counter for the given type.
// Safe before InitMetrics and under concurrency.
func RecordSecurityDenial(ctx context.Context, denialType string) {
	if securityDenialCounter == nil {
		return
	}
	securityDenialCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("type", denialType),
	))
}

// auditWriteFailureCounter counts management-audit writes that failed
// (audit_write_failures_total). Audit writes are best-effort and do not block
// the business action, so this counter is the ONLY reliable signal that the
// audit trail has gaps — a monitoring consumer must alert on any non-zero rate.
var auditWriteFailureCounter metric.Int64Counter

// RecordAuditWriteFailure increments the audit-write-failure counter. Safe before
// InitMetrics and under concurrency.
func RecordAuditWriteFailure(ctx context.Context) {
	if auditWriteFailureCounter == nil {
		return
	}
	auditWriteFailureCounter.Add(ctx, 1)
}

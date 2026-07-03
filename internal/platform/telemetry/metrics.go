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

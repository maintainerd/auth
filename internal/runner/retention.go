package runner

import (
	"context"
	"log/slog"
	"time"
)

const (
	// DefaultRetentionPeriod is the default maximum age for auth events.
	// PCI DSS 10.7.1 requires at least 12 months of audit trail history.
	DefaultRetentionPeriod = 365 * 24 * time.Hour

	// DefaultRetentionInterval is how often the retention job runs.
	DefaultRetentionInterval = 24 * time.Hour
)

// RetentionDeleter is the subset of AuthEventService that the retention runner
// needs. Defined here to avoid an import cycle (service ↔ runner).
type RetentionDeleter interface {
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}

// StartRetentionRunner starts a background goroutine that periodically deletes
// auth events older than the configured retention period. It respects context
// cancellation for graceful shutdown.
func StartRetentionRunner(ctx context.Context, deleter RetentionDeleter, retention, interval time.Duration) {
	if retention <= 0 {
		retention = DefaultRetentionPeriod
	}
	if interval <= 0 {
		interval = DefaultRetentionInterval
	}

	slog.Info("retention: starting auth event retention runner",
		"retention_days", int(retention.Hours()/24),
		"interval_hours", int(interval.Hours()),
	)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("retention: shutting down")
			return
		case <-ticker.C:
			cutoff := time.Now().UTC().Add(-retention)
			count, err := deleter.DeleteOlderThan(ctx, cutoff)
			if err != nil {
				slog.Error("retention: failed to delete old auth events", "error", err)
				continue
			}
			if count > 0 {
				slog.Info("retention: deleted old auth events",
					"count", count,
					"cutoff", cutoff.Format(time.RFC3339),
				)
			}
		}
	}
}

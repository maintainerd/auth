// Package retry provides a simple exponential-backoff helper for startup
// dependency probes (database, Redis, AMQP, etc.).
package retry

import (
	"context"
	"log/slog"
	"time"
)

const maxDelay = 16 * time.Second

// WithBackoff calls op up to maxAttempts times. On each failure it waits for
// an exponentially increasing delay (starting at baseDelay, capped at 16s)
// before the next attempt. It returns nil on the first success, ctx.Err() if
// the context is cancelled during a wait, or the last op error when all
// attempts are exhausted.
func WithBackoff(ctx context.Context, label string, maxAttempts int, baseDelay time.Duration, op func() error) error {
	delay := baseDelay
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := op(); err == nil {
			return nil
		} else if attempt == maxAttempts {
			return err
		} else {
			slog.Warn("dependency not ready, retrying",
				"dependency", label,
				"attempt", attempt,
				"max_attempts", maxAttempts,
				"retry_in", delay,
				"error", err,
			)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}
	return nil
}

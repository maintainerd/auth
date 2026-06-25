package oauth

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// SweepExpiredBrokerSessions periodically deletes broker sessions past their
// ExpiresAt, freeing storage and preventing stale sessions from accumulating.
// Call once during startup with a long-lived context; the goroutine exits when
// the context is cancelled.
func SweepExpiredBrokerSessions(ctx context.Context, db *gorm.DB, interval time.Duration) {
	repo := NewOAuthBrokerSessionRepository(db)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n, err := repo.DeleteExpired(time.Now())
				if err != nil {
					slog.Warn("oauth_broker_sessions sweeper: delete failed", "err", err)
				} else if n > 0 {
					slog.Info("oauth_broker_sessions sweeper: cleaned expired rows", "count", n)
				}
			}
		}
	}()
}

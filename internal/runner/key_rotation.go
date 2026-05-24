package runner

import (
	"context"
	"log/slog"
	"time"

	"github.com/maintainerd/auth/internal/jwt"
)

// StartKeyRotationRunner rotates the active JWT signing key every period.
// It performs an immediate rotation on startup so the service never runs with
// only the boot-time key. The runner exits when ctx is cancelled.
func StartKeyRotationRunner(ctx context.Context, period time.Duration) {
	if err := jwt.RotateKeys(); err != nil {
		slog.Error("key_rotation: initial rotation failed", "err", err)
	} else {
		slog.Info("key_rotation: initial rotation complete")
	}

	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("key_rotation: runner stopped")
			return
		case <-ticker.C:
			if err := jwt.RotateKeys(); err != nil {
				slog.Error("key_rotation: rotation failed", "err", err)
			} else {
				slog.Info("key_rotation: keys rotated")
			}
		}
	}
}

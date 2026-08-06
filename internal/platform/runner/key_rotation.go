package runner

import (
	"context"
	"log/slog"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
)

// rotateKeys is a seam so the runner's timing can be tested without mutating the
// real process key store.
var rotateKeys = jwt.RotateKeys

// StartKeyRotationRunner rotates the PROCESS-LOCAL signing key every period.
//
// This is only correct when the key came from the environment (JWT_PRIVATE_KEY).
// When the signing_keys table owns the key, rotating in memory would publish a
// JWKS key that exists in no row — the database, the key-management API and the
// served JWKS would silently disagree about what is signing tokens. That case
// belongs to oauth.StartSigningKeyRotationRunner, which persists; see
// cmd/server/workers.go for the choice between them.
//
// The runner exits when ctx is cancelled.
func StartKeyRotationRunner(ctx context.Context, period time.Duration) {
	// Deliberately NO rotation at startup.
	//
	// This used to rotate unconditionally on boot "so the service never runs with
	// only the boot-time key", which is backwards: the boot key was just loaded
	// and is seconds old. Rotating immediately discarded it, and because a
	// container restart re-runs this, every deploy or crash loop burned another
	// key and forced every relying party to refetch JWKS. A key that is genuinely
	// overdue is picked up by the first tick, bounded by one rotation period —
	// which is the guarantee the period expresses anyway.
	slog.Info("key_rotation: in-memory runner started", "period", period)

	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("key_rotation: runner stopped")
			return
		case <-ticker.C:
			if err := rotateKeys(); err != nil {
				slog.Error("key_rotation: rotation failed", "err", err)
			} else {
				slog.Info("key_rotation: keys rotated")
			}
		}
	}
}

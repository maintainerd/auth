package oauth

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// DefaultSigningKeyRotationPeriod is the fallback interval when the operator
// configured none. It matches the interval the in-memory runner used, so
// switching a deployment onto the persisted path does not silently change how
// often keys turn over.
const DefaultSigningKeyRotationPeriod = 24 * time.Hour

// StartSigningKeyRotationRunner rotates the global signing key through the
// signing_keys TABLE every period, and blocks until ctx is cancelled.
//
// It exists because runner.StartKeyRotationRunner calls jwt.RotateKeys(), which
// only swaps the process's in-memory key: no row is written. On the DB-backed
// path — the self-host default whenever JWT_PRIVATE_KEY is unset — that has two
// consequences an IAM server cannot carry. Every token minted after the first
// tick is signed by a kid that exists in no row, so a restart cannot reload the
// key and every one of those tokens becomes permanently unverifiable; and in a
// multi-replica deployment each replica rotates to its own private key, so a
// token minted by replica A cannot be verified against the key set replica B
// publishes. Rotation has to go through the store the key set is published
// from, which is what RotateGlobalSigningKey does.
//
// Deliberately NOT rotating on startup, unlike the in-memory runner. That
// runner rotated immediately because its boot key was ephemeral and unpublished;
// here EnsureGlobalSigningKeyFromDB has already persisted and published the boot
// key, so rotating on every start would burn a key row per restart and make a
// rolling deploy rotate once per replica.
func StartSigningKeyRotationRunner(ctx context.Context, db *gorm.DB, period time.Duration) {
	if db == nil {
		// No store means no persisted rotation. Falling back to jwt.RotateKeys()
		// here would reintroduce exactly the unpersisted-key failure this runner
		// exists to remove, so the runner declines to run instead.
		slog.ErrorContext(ctx, "signing key rotation: no database handle; periodic key rotation is disabled")
		return
	}
	runSigningKeyRotation(ctx, NewSigningKeyRepository(db), period)
}

func runSigningKeyRotation(ctx context.Context, repo SigningKeyRepository, period time.Duration) {
	if period <= 0 {
		// time.NewTicker panics on a non-positive period, and a panic in a
		// background goroutine takes the whole process down.
		period = DefaultSigningKeyRotationPeriod
	}

	ticker := time.NewTicker(period)
	defer ticker.Stop()

	rotateOnTick(ctx, repo, ticker.C)
}

// rotateOnTick takes the tick source as a parameter so a test can drive exactly
// one rotation. Driving it with a real short-period ticker instead makes the
// number of rotations a race with the scheduler, which is how "one tick
// persisted one key" turns into a flaky assertion.
func rotateOnTick(ctx context.Context, repo SigningKeyRepository, ticks <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			slog.Info("signing key rotation: runner stopped")
			return
		case <-ticks:
			if err := rotateGlobalSigningKey(ctx, repo); err != nil {
				// A failed rotation leaves the previous PERSISTED key active,
				// signing and published — degraded, but every token it minted stays
				// verifiable across restarts and replicas. The unsafe outcome is
				// installing a key the store does not have, which
				// rotateGlobalSigningKey refuses to do.
				slog.ErrorContext(ctx, "signing key rotation failed; the current persisted key stays active", "error", err)
				continue
			}
			slog.InfoContext(ctx, "signing key rotation: rotated and persisted a new global signing key")
		}
	}
}

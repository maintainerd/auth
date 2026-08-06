package runner

import (
	"context"
	"testing"
	"time"
)

// The runner used to rotate unconditionally at startup "so the service never
// runs with only the boot-time key". That was backwards: the boot key is seconds
// old, so rotating discarded it — and because a container restart re-runs this,
// every deploy or crash loop burned another key and forced every relying party
// to refetch JWKS.
func TestStartKeyRotationRunnerDoesNotRotateAtStartup(t *testing.T) {
	rotations := make(chan struct{}, 8)
	restore := swapRotateForTest(func() error {
		rotations <- struct{}{}
		return nil
	})
	defer restore()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		// A long period guarantees no tick fires, so any rotation is a startup one.
		StartKeyRotationRunner(ctx, time.Hour)
		close(done)
	}()

	select {
	case <-rotations:
		t.Fatal("startup must not rotate — a restart would burn a signing key each time")
	case <-time.After(80 * time.Millisecond):
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runner did not stop on context cancellation")
	}
}

// It must still rotate on each tick, and a failing rotation must not kill the
// runner — one transient error should not stop rotation for the process lifetime.
func TestStartKeyRotationRunnerRotatesOnTick(t *testing.T) {
	rotations := make(chan struct{}, 16)
	restore := swapRotateForTest(func() error {
		rotations <- struct{}{}
		return errAlwaysFails
	})
	defer restore()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go StartKeyRotationRunner(ctx, 5*time.Millisecond)

	for i := 0; i < 3; i++ {
		select {
		case <-rotations:
		case <-time.After(2 * time.Second):
			t.Fatalf("expected rotation %d; a failing rotation must not stop the runner", i+1)
		}
	}
}

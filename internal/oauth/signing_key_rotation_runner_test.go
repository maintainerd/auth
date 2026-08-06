package oauth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failFirstCreateRepo fails the first N persists. The failure count lives in
// the repo rather than being flipped by the test mid-run so that every write to
// it happens on the runner's goroutine — a test that reaches in and mutates the
// repo while the runner is reading it is a data race, not a fixture.
type failFirstCreateRepo struct {
	fakeSigningKeyRepo
	remainingFailures int
}

func (r *failFirstCreateRepo) Create(k *SigningKey) error {
	if r.remainingFailures > 0 {
		r.remainingFailures--
		return errors.New("db down")
	}
	return r.fakeSigningKeyRepo.Create(k)
}

// tick drives exactly one rotation and waits for it to finish, so every
// assertion below is about a known number of rotations rather than about how
// many times a real ticker happened to fire.
func tick(t *testing.T, ticks chan<- time.Time, rotated <-chan string) string {
	t.Helper()
	select {
	case ticks <- time.Now():
	case <-time.After(5 * time.Second):
		t.Fatal("the rotation runner never read the tick")
	}
	select {
	case kid := <-rotated:
		return kid
	case <-time.After(5 * time.Second):
		t.Fatal("the rotation runner never finished the tick")
		return ""
	}
}

// startRotationRunner runs rotateOnTick against repo and returns the tick
// channel, a channel that reports the kid installed by each completed rotation
// (empty string when the rotation failed), and a stop func.
func startRotationRunner(t *testing.T, repo SigningKeyRepository) (chan time.Time, <-chan string, func()) {
	t.Helper()

	rotated := make(chan string, 8)
	orig := installSigningKey
	installSigningKey = func(privPEM []byte, kid string) error {
		assert.NotEmpty(t, privPEM)
		rotated <- kid
		return nil
	}

	ticks := make(chan time.Time)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		rotateOnTick(ctx, repo, ticks)
	}()

	return ticks, rotated, func() {
		cancel()
		<-done
		installSigningKey = orig
	}
}

// The periodic runner used to call jwt.RotateKeys(), which writes no row: after
// the first tick JWKS advertised a kid that existed in signing_keys nowhere, so
// a restart could not reload the signing key and every token minted since the
// last restart became permanently unverifiable. Each replica also rotated to its
// own private key, so replica A's tokens failed against replica B's key set.
func TestSigningKeyRotationRunner(t *testing.T) {
	t.Run("one tick persists the new active kid and installs that same kid", func(t *testing.T) {
		repo := &fakeSigningKeyRepo{}
		ticks, rotated, stop := startRotationRunner(t, repo)

		installedKID := tick(t, ticks, rotated)
		stop()

		require.Len(t, repo.created, 1, "one tick must persist exactly one key")
		persisted := repo.created[0]
		assert.Equal(t, persisted.KID, installedKID,
			"the process must sign with the kid it persisted, not an in-memory-only one")

		// The kid GetActiveSigningKey resolves is what JWKS publishes and what a
		// restart reloads; it has to be the kid now signing tokens.
		active, err := NewKeyRotationService(repo).GetActiveSigningKey(context.Background(), 0)
		require.NoError(t, err)
		assert.Equal(t, installedKID, active.KID,
			"JWKS would otherwise publish a key set that does not contain the signing key")
		assert.Equal(t, "active", active.Status)
		assert.NotEmpty(t, active.PrivateKeyEncrypted,
			"a row with no stored private key cannot be reloaded after a restart")
		assert.NotEmpty(t, active.PublicKeyPEM,
			"JWKS is built from the public PEM; an empty one publishes nothing to verify with")
	})

	t.Run("a failed rotation installs nothing and the runner keeps ticking", func(t *testing.T) {
		repo := &failFirstCreateRepo{remainingFailures: 1}
		ticks, rotated, stop := startRotationRunner(t, repo)

		// The tick channel is unbuffered and only read at the top of the loop, so
		// this second send cannot complete until the first rotation has finished.
		// That ordering is what lets the assertions below be about a known number
		// of rotations without a sleep.
		for i := 0; i < 2; i++ {
			select {
			case ticks <- time.Now():
			case <-time.After(5 * time.Second):
				t.Fatalf("the rotation runner never read tick %d", i+1)
			}
		}

		var installedKID string
		select {
		case installedKID = <-rotated:
		case <-time.After(5 * time.Second):
			t.Fatal("the runner stopped rotating after a failed rotation")
		}
		stop()

		// A failed persist must not install: signing with a key the store does not
		// have is the exact failure the persisted path exists to remove.
		assert.Empty(t, rotated,
			"the rotation whose row never committed must not have installed a key")
		require.Len(t, repo.created, 1)
		assert.Equal(t, repo.created[0].KID, installedKID,
			"the runner must recover on the next tick rather than die on a transient DB error")
	})

	t.Run("successive ticks each persist a distinct key", func(t *testing.T) {
		repo := &fakeSigningKeyRepo{}
		ticks, rotated, stop := startRotationRunner(t, repo)

		first := tick(t, ticks, rotated)
		second := tick(t, ticks, rotated)
		stop()

		require.Len(t, repo.created, 2)
		assert.NotEqual(t, first, second)

		active, err := NewKeyRotationService(repo).GetActiveSigningKey(context.Background(), 0)
		require.NoError(t, err)
		assert.Equal(t, second, active.KID)
	})

	// A tick must not take the key it just superseded out of the published set:
	// tokens signed with the boot key are still inside their lifetime, and JWKS
	// is the only thing a relying party can verify them against.
	t.Run("a tick keeps the superseded boot key published", func(t *testing.T) {
		repo := &fakeSigningKeyRepo{keys: []SigningKey{{
			KID:          "boot",
			Status:       "active",
			PublicKeyPEM: testPublicKeyPEM(t),
			CreatedAt:    time.Now(),
		}}}
		ticks, rotated, stop := startRotationRunner(t, repo)

		rotatedKID := tick(t, ticks, rotated)
		stop()

		assert.Empty(t, repo.retired,
			"a freshly superseded key must keep being published until its tokens have expired")

		jwks, err := NewKeyRotationService(repo).ListJWKS(context.Background(), 0)
		require.NoError(t, err)
		published := make([]string, 0, len(jwks))
		for _, k := range jwks {
			published = append(published, k.Kid)
		}
		assert.Contains(t, published, rotatedKID, "the new signer must be verifiable")
		assert.Contains(t, published, "boot", "tokens signed before the tick must stay verifiable")
	})

	t.Run("a non-positive period falls back to the default instead of panicking", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			defer close(done)
			runSigningKeyRotation(ctx, &fakeSigningKeyRepo{}, 0)
		}()
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("the runner did not stop on context cancellation")
		}
	})

	// The one behaviour that separates this runner from runner.StartKeyRotationRunner,
	// which rotates immediately at startup (internal/platform/runner/key_rotation.go).
	// Startup rotation is safe only for an ephemeral, unpublished in-memory boot
	// key. On the DB path EnsureGlobalSigningKeyFromDB has already persisted and
	// published the boot key, so rotating at startup writes a second signing_keys
	// row on every process start: JWKS grows a key per restart, a rolling deploy
	// rotates once per replica, and superseded rows stay active for the whole
	// refresh-token retention window. If this assertion ever has to be relaxed,
	// the DB path is being driven by the wrong runner.
	t.Run("starting the runner rotates nothing until the first tick", func(t *testing.T) {
		repo := &fakeSigningKeyRepo{}

		var installed bool
		orig := installSigningKey
		defer func() { installSigningKey = orig }()
		installSigningKey = func([]byte, string) error { installed = true; return nil }

		// A period long enough that the ticker cannot fire during the test, so the
		// only rotation this could observe is a startup one. Cancelling and waiting
		// for the goroutine to return is what makes it deterministic: a rotation
		// placed before the select loop would have completed by then.
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			defer close(done)
			runSigningKeyRotation(ctx, repo, time.Hour)
		}()
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("the runner did not stop on context cancellation")
		}

		assert.Empty(t, repo.created,
			"a process start must not mint a signing key; only a tick may")
		assert.False(t, installed,
			"a process start must not swap the boot key the store already published")
	})

	// Falling back to the in-memory jwt.RotateKeys() when there is no store would
	// reintroduce the unpersisted-key failure this runner exists to remove.
	t.Run("a nil db disables rotation rather than rotating in memory", func(t *testing.T) {
		var installed bool
		orig := installSigningKey
		defer func() { installSigningKey = orig }()
		installSigningKey = func([]byte, string) error { installed = true; return nil }

		StartSigningKeyRotationRunner(context.Background(), nil, time.Millisecond)

		assert.False(t, installed, "no store means no rotation, not an unpersisted rotation")
	})
}

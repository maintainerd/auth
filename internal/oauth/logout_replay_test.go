package oauth

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeJTIStore struct {
	denied  map[string]bool
	failOn  string
	denyErr error
}

func (f *fakeJTIStore) IsJTIDenied(_ context.Context, jti string) (bool, error) {
	if f.failOn == "is" {
		return false, errors.New("redis unavailable")
	}
	return f.denied[jti], nil
}

func (f *fakeJTIStore) DenyJTI(_ context.Context, jti string, _ time.Duration) error {
	if f.denyErr != nil {
		return f.denyErr
	}
	if f.denied == nil {
		f.denied = map[string]bool{}
	}
	f.denied[jti] = true
	return nil
}

// OIDC Back-Channel Logout 1.0 §2.6 requires a logout token's jti to be
// single-use. The in-process guard could not deliver that across replicas: the
// same token replayed against another instance saw an empty map and was
// accepted, so the property vanished the moment the service was scaled.
func TestLogoutTokenReplayUsesTheSharedStore(t *testing.T) {
	restore := func() { logoutTokenReplayStore = nil }
	t.Cleanup(restore)

	t.Run("first use is accepted, replay is refused", func(t *testing.T) {
		logoutTokenReplayStore = &fakeJTIStore{}
		if !rememberLogoutTokenJTI(context.Background(), "jti-1") {
			t.Fatal("the first use of a jti must be accepted")
		}
		if rememberLogoutTokenJTI(context.Background(), "jti-1") {
			t.Fatal("a replayed jti must be refused")
		}
	})

	// A second replica shares the store, so the replay is caught there too —
	// which is the whole point of moving off the in-process map.
	t.Run("a replay against another replica is refused", func(t *testing.T) {
		shared := &fakeJTIStore{}
		logoutTokenReplayStore = shared
		if !rememberLogoutTokenJTI(context.Background(), "jti-2") {
			t.Fatal("first use must be accepted")
		}
		// Same shared store, as a different process would see it.
		logoutTokenReplayStore = shared
		if rememberLogoutTokenJTI(context.Background(), "jti-2") {
			t.Fatal("the replay must be refused on every replica, not just the first")
		}
	})

	// Fail CLOSED: an unreachable store must not turn single-use into
	// unlimited-use for a token that ends sessions.
	t.Run("a read error refuses the token", func(t *testing.T) {
		logoutTokenReplayStore = &fakeJTIStore{failOn: "is"}
		if rememberLogoutTokenJTI(context.Background(), "jti-3") {
			t.Fatal("an unreadable replay store must refuse, not accept")
		}
	})

	t.Run("a write error refuses the token", func(t *testing.T) {
		logoutTokenReplayStore = &fakeJTIStore{denyErr: errors.New("redis unavailable")}
		if rememberLogoutTokenJTI(context.Background(), "jti-4") {
			t.Fatal("a jti that could not be recorded must refuse, not accept")
		}
	})

	// With no shared store wired, the in-process guard still applies.
	t.Run("falls back to the in-process guard", func(t *testing.T) {
		logoutTokenReplayStore = nil
		if !rememberLogoutTokenJTI(context.Background(), "jti-5") {
			t.Fatal("first use must be accepted by the fallback guard")
		}
		if rememberLogoutTokenJTI(context.Background(), "jti-5") {
			t.Fatal("the fallback guard must still refuse a replay")
		}
	})
}

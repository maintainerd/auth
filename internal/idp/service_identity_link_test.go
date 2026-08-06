package idp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeLinkStore records what the service does with in-flight link requests and
// enforces the same single-use contract the real store does.
type fakeLinkStore struct {
	byState  map[string]*IdentityLinkRequest
	consumed map[int64]bool
	created  []*IdentityLinkRequest
}

func newFakeLinkStore() *fakeLinkStore {
	return &fakeLinkStore{byState: map[string]*IdentityLinkRequest{}, consumed: map[int64]bool{}}
}

func (f *fakeLinkStore) Create(_ context.Context, req *IdentityLinkRequest) error {
	req.ID = int64(len(f.created) + 1)
	f.created = append(f.created, req)
	f.byState[req.State] = req
	return nil
}

func (f *fakeLinkStore) FindByState(_ context.Context, state string) (*IdentityLinkRequest, error) {
	req, ok := f.byState[state]
	if !ok {
		return nil, nil
	}
	clone := *req
	clone.Consumed = f.consumed[req.ID]
	return &clone, nil
}

func (f *fakeLinkStore) Consume(_ context.Context, id int64) error {
	if f.consumed[id] {
		return errors.New("already consumed")
	}
	f.consumed[id] = true
	return nil
}

// Account linking is a credential-granting operation: whoever controls a linked
// provider identity can sign into the account. Every rejection below is what
// stops that being handed to the wrong person.
func TestCompleteIdentityLink_Rejections(t *testing.T) {
	const victim int64 = 1
	const attacker int64 = 2

	newSvc := func(store IdentityLinkRequestStore) *federationService {
		return &federationService{linkStore: store}
	}

	valid := func(store *fakeLinkStore, userID int64, state string, expiry time.Time) {
		store.byState[state] = &IdentityLinkRequest{
			ID: 10, UserID: userID, State: state, ExpiresAt: expiry,
			ProviderIdentifier: "google", IdentityProviderID: 5,
		}
	}

	// THE account-linking CSRF case. An attacker starts a link with their own
	// provider account, then induces the victim's browser to hit the callback.
	// If the request were not bound to the user who started it, the attacker's
	// provider identity would end up attached to the victim's account — a
	// permanent, silent way in.
	t.Run("a state belonging to another user is refused", func(t *testing.T) {
		store := newFakeLinkStore()
		valid(store, attacker, "st-attacker", time.Now().Add(time.Minute))

		_, err := newSvc(store).CompleteIdentityLink(context.Background(), victim, "st-attacker", "code", "https://app/cb")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no longer valid")
		assert.False(t, store.consumed[10], "a request that is not ours must not even be consumed")
	})

	t.Run("an unknown state is refused", func(t *testing.T) {
		_, err := newSvc(newFakeLinkStore()).CompleteIdentityLink(context.Background(), victim, "nope", "code", "https://app/cb")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no longer valid")
	})

	t.Run("an expired state is refused", func(t *testing.T) {
		store := newFakeLinkStore()
		valid(store, victim, "st-old", time.Now().Add(-time.Minute))

		_, err := newSvc(store).CompleteIdentityLink(context.Background(), victim, "st-old", "code", "https://app/cb")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no longer valid")
	})

	// Replay: the same callback delivered twice must not perform two exchanges.
	t.Run("an already-consumed state is refused", func(t *testing.T) {
		store := newFakeLinkStore()
		valid(store, victim, "st-used", time.Now().Add(time.Minute))
		store.consumed[10] = true

		_, err := newSvc(store).CompleteIdentityLink(context.Background(), victim, "st-used", "code", "https://app/cb")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no longer valid")
	})

	t.Run("state and code are both required", func(t *testing.T) {
		svc := newSvc(newFakeLinkStore())
		_, err := svc.CompleteIdentityLink(context.Background(), victim, "", "code", "https://app/cb")
		require.Error(t, err)
		_, err = svc.CompleteIdentityLink(context.Background(), victim, "st", "", "https://app/cb")
		require.Error(t, err)
	})

	// Rejections must be indistinguishable, so probing cannot map out which
	// states exist or who they belong to.
	t.Run("every rejection reports the same message", func(t *testing.T) {
		store := newFakeLinkStore()
		valid(store, attacker, "st-other", time.Now().Add(time.Minute))
		store.byState["st-expired"] = &IdentityLinkRequest{ID: 11, UserID: victim, State: "st-expired", ExpiresAt: time.Now().Add(-time.Minute)}

		var msgs []string
		for _, st := range []string{"st-other", "st-expired", "st-missing"} {
			_, err := newSvc(store).CompleteIdentityLink(context.Background(), victim, st, "code", "https://app/cb")
			require.Error(t, err)
			msgs = append(msgs, err.Error())
		}
		assert.Equal(t, msgs[0], msgs[1])
		assert.Equal(t, msgs[1], msgs[2])
	})

	t.Run("linking is disabled when no store is configured", func(t *testing.T) {
		_, err := (&federationService{}).CompleteIdentityLink(context.Background(), victim, "st", "code", "https://app/cb")
		require.Error(t, err)
	})
}

// The provider redirect must carry PKCE and a nonce, or the authorization code
// can be intercepted and the id_token replayed.
func TestBuildProviderAuthorizeURL(t *testing.T) {
	info := &BrokerProviderInfo{
		AuthorizationEndpoint: "https://provider.test/authorize",
		ClientID:              "client-abc",
		Scopes:                []string{"openid", "email"},
	}

	raw, err := buildProviderAuthorizeURL(info, "https://app.test/cb", "state-1", "nonce-1", "verifier-1")
	require.NoError(t, err)

	assert.Contains(t, raw, "response_type=code")
	assert.Contains(t, raw, "client_id=client-abc")
	assert.Contains(t, raw, "state=state-1")
	assert.Contains(t, raw, "nonce=nonce-1")
	assert.Contains(t, raw, "code_challenge_method=S256")
	// The verifier itself must never appear in a URL the browser can read.
	assert.NotContains(t, raw, "verifier-1")

	t.Run("falls back to OIDC scopes when the provider configures none", func(t *testing.T) {
		bare := &BrokerProviderInfo{AuthorizationEndpoint: "https://provider.test/authorize", ClientID: "c"}
		raw, err := buildProviderAuthorizeURL(bare, "https://app.test/cb", "s", "n", "v")
		require.NoError(t, err)
		assert.Contains(t, raw, "openid")
	})

	t.Run("an invalid endpoint is rejected", func(t *testing.T) {
		_, err := buildProviderAuthorizeURL(&BrokerProviderInfo{AuthorizationEndpoint: "://bad"}, "https://app/cb", "s", "n", "v")
		require.Error(t, err)
	})
}

func TestRandomURLToken(t *testing.T) {
	a, err := randomURLToken(32)
	require.NoError(t, err)
	b, err := randomURLToken(32)
	require.NoError(t, err)
	assert.NotEqual(t, a, b, "state and nonce must be unpredictable")
	assert.NotContains(t, a, "=", "must be URL-safe and unpadded")
}

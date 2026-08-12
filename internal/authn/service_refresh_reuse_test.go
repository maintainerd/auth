package authn

import (
	"context"

	"github.com/google/uuid"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubDenylist is a real in-memory denylist, unlike recordingLogoutJTIDenylister
// which always answers "not denied" and so cannot exercise reuse detection at
// all. That gap is why the family-revocation path shipped unreachable.
type stubDenylist struct {
	mu      sync.Mutex
	denied  map[string]time.Time
	replays map[string][]byte
}

func newStubDenylist() *stubDenylist {
	return &stubDenylist{denied: map[string]time.Time{}, replays: map[string][]byte{}}
}

func (d *stubDenylist) DenyJTI(_ context.Context, jti string, ttl time.Duration) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.denied[jti] = time.Now().Add(ttl)
	return nil
}

func (d *stubDenylist) IsJTIDenied(_ context.Context, jti string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	exp, ok := d.denied[jti]
	return ok && time.Now().Before(exp), nil
}

func (d *stubDenylist) has(prefix string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	for k := range d.denied {
		if strings.HasPrefix(k, prefix) {
			return true
		}
	}
	return false
}

// stubDenylist also implements cache.RefreshReplayStore so the idempotent-rotation
// path is exercised, exactly as *cache.Cache does in production.
func (d *stubDenylist) StoreRefreshReplay(_ context.Context, jti string, payload []byte, _ time.Duration) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	cp := make([]byte, len(payload))
	copy(cp, payload)
	d.replays[jti] = cp
	return nil
}

func (d *stubDenylist) GetRefreshReplay(_ context.Context, jti string) ([]byte, bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	p, ok := d.replays[jti]
	return p, ok, nil
}

// expireReplays simulates the rotation-overlap window elapsing (the cached sets are
// gone) so a later replay is treated as genuine out-of-window reuse.
func (d *stubDenylist) expireReplays() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.replays = map[string][]byte{}
}

// Inside the rotation-overlap window (refresh_token_reuse_interval_seconds), a
// duplicate of a just-consumed refresh token is the benign concurrency case (two
// tabs, a retry, parallel requests racing on the shared cookie). It MUST be answered
// idempotently with the SAME token set the rotation minted, and MUST NOT revoke the
// family — otherwise every in-window race logs the user out (RFC 9700 §4.14.2).
func TestRefreshToken_InWindowDuplicateIsIdempotent(t *testing.T) {
	initTestJWTKeysService(t)

	const sub = "user-sub-1"
	const clientID = "test-client"

	sessionID := uuid.New().String()
	parent, err := jwt.GenerateRefreshTokenWithOptionsContext(context.Background(), sub,
		"https://auth.example.com", clientID, "realm", &jwt.RefreshTokenOptions{SessionID: sessionID})
	require.NoError(t, err)

	denylist := newStubDenylist()
	svc := &loginService{
		userRepo:       &mockUserRepo{findBySubAndClientIDFn: func(_, _ string) (*User, error) { return buildActiveUser(t, "pw"), nil }},
		clientRepo:     &mockClientRepo{findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return buildActiveClient(), nil }},
		sessionService: &mockSessionService{},
		jtiDenylist:    denylist,
	}

	first, err := svc.RefreshToken(context.Background(), parent, "")
	require.NoError(t, err)
	require.NotEmpty(t, first.RefreshToken)

	// Immediate duplicate of the consumed parent → same set, success, no revocation.
	dup, err := svc.RefreshToken(context.Background(), parent, "")
	require.NoError(t, err, "an in-window duplicate must succeed idempotently, not fail")
	assert.Equal(t, first.RefreshToken, dup.RefreshToken,
		"must return the SAME rotated token, not a new independent one")
	assert.Equal(t, first.AccessToken, dup.AccessToken)
	assert.False(t, denylist.has("rtfam:"), "an in-window duplicate must NOT revoke the family")

	// The child issued by the first rotation must still be usable.
	_, err = svc.RefreshToken(context.Background(), first.RefreshToken, "")
	require.NoError(t, err, "the legitimately-rotated child must survive an in-window duplicate")
}

// Outside the overlap window (the cached set has expired), replay of a consumed
// token is genuine reuse and MUST revoke the whole family (RFC 6819 §5.2.1.1,
// OAuth 2.1 §6.1) — a stolen sibling token must not stay live.
//
// This previously could not be detected at all: the consumed jti was written to the
// generic access-token denylist, so the shared validator rejected the replay as
// merely "invalid" before reuse detection ran.
func TestRefreshToken_OutOfWindowReuseRevokesFamily(t *testing.T) {
	initTestJWTKeysService(t)

	const sub = "user-sub-1"
	const clientID = "test-client"

	sessionID := uuid.New().String()
	parent, err := jwt.GenerateRefreshTokenWithOptionsContext(context.Background(), sub,
		"https://auth.example.com", clientID, "realm", &jwt.RefreshTokenOptions{SessionID: sessionID})
	require.NoError(t, err)

	denylist := newStubDenylist()
	svc := &loginService{
		userRepo:       &mockUserRepo{findBySubAndClientIDFn: func(_, _ string) (*User, error) { return buildActiveUser(t, "pw"), nil }},
		clientRepo:     &mockClientRepo{findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return buildActiveClient(), nil }},
		sessionService: &mockSessionService{},
		jtiDenylist:    denylist,
	}

	first, err := svc.RefreshToken(context.Background(), parent, "")
	require.NoError(t, err)
	require.NotEmpty(t, first.RefreshToken)
	assert.True(t, denylist.has("rtused:"), "the consumed token must be marked used")
	assert.False(t, denylist.has("jti:"),
		"the refresh jti must NOT enter the generic access-token denylist — that is what made reuse detection unreachable")

	// The overlap window elapses: the cached replay is gone.
	denylist.expireReplays()

	// Replay of the consumed parent is now genuine reuse → revoke the family.
	_, err = svc.RefreshToken(context.Background(), parent, "")
	require.Error(t, err)

	// …and the legitimately-issued child must now be dead too.
	_, err = svc.RefreshToken(context.Background(), first.RefreshToken, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "family has been revoked",
		"a stolen sibling must not survive a detected out-of-window replay")
}

// The realm claim must be identical across rotations. It previously flipped from
// the IdP identifier to the tenant name, and because the refresh path resolved
// the client by joining on identity_providers.identifier, the SECOND refresh
// failed with "client not found" — refresh worked exactly once.
func TestRefreshToken_RealmClaimIsStableAcrossRotations(t *testing.T) {
	initTestJWTKeysService(t)

	const sub = "user-sub-1"
	const clientID = "test-client"

	sessionID := uuid.New().String()
	token, err := jwt.GenerateRefreshTokenWithOptionsContext(context.Background(), sub,
		"https://auth.example.com", clientID, "realm", &jwt.RefreshTokenOptions{SessionID: sessionID})
	require.NoError(t, err)

	svc := &loginService{
		userRepo:       &mockUserRepo{findBySubAndClientIDFn: func(_, _ string) (*User, error) { return buildActiveUser(t, "pw"), nil }},
		clientRepo:     &mockClientRepo{findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return buildActiveClient(), nil }},
		sessionService: &mockSessionService{},
		jtiDenylist:    newStubDenylist(),
	}

	var realms []string
	for i := 0; i < 4; i++ {
		resp, err := svc.RefreshToken(context.Background(), token, "")
		require.NoErrorf(t, err, "rotation %d failed — refresh must work more than once", i+1)
		claims, err := jwt.ValidateTokenWithContext(context.Background(), resp.RefreshToken)
		require.NoError(t, err)
		realm, _ := claims["provider_id"].(string)
		realms = append(realms, realm)
		token = resp.RefreshToken
	}

	for i := 1; i < len(realms); i++ {
		assert.Equalf(t, realms[0], realms[i],
			"provider_id drifted between rotations (%q -> %q); the next refresh would fail to resolve its client",
			realms[0], realms[i])
	}
}

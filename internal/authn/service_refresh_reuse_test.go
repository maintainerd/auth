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
	mu     sync.Mutex
	denied map[string]time.Time
}

func newStubDenylist() *stubDenylist {
	return &stubDenylist{denied: map[string]time.Time{}}
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

// Replaying a rotated refresh token must be detected AS REUSE and must revoke
// the whole family (RFC 6819 §5.2.1.1, OAuth 2.1 §6.1).
//
// This previously could not happen: the consumed jti was written to the generic
// access-token denylist, so the shared validator rejected the replay as merely
// "invalid" before reuse detection ran — and a stolen sibling token stayed live.
func TestRefreshToken_ReuseRevokesFamily(t *testing.T) {
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

	// First use succeeds and rotates.
	first, err := svc.RefreshToken(context.Background(), parent, "")
	require.NoError(t, err)
	require.NotEmpty(t, first.RefreshToken)
	assert.True(t, denylist.has("rtused:"), "the consumed token must be marked used")
	assert.False(t, denylist.has("jti:"),
		"the refresh jti must NOT enter the generic access-token denylist — that is what made reuse detection unreachable")

	// Replay of the consumed parent is reuse, not a generic validation failure.
	// Immediately, i.e. INSIDE the grace window — the worst case, and the one an
	// attacker actually produces. The message may say "already consumed", but the
	// family must be revoked regardless.
	_, err = svc.RefreshToken(context.Background(), parent, "")
	require.Error(t, err)

	// …and the legitimately-issued child must now be dead too.
	_, err = svc.RefreshToken(context.Background(), first.RefreshToken, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "family has been revoked",
		"a stolen sibling must not survive a detected replay")
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

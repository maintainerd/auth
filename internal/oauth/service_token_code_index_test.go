package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTokenStore satisfies both cache.JTIDenylister and the wider
// issuedTokenIndex, the same way the shared *cache.Cache does in production.
type fakeTokenStore struct {
	mu      sync.Mutex
	denied  map[string]time.Duration
	entries map[string][]byte
	denyErr error
}

func newFakeTokenStore() *fakeTokenStore {
	return &fakeTokenStore{
		denied:  map[string]time.Duration{},
		entries: map[string][]byte{},
	}
}

func (f *fakeTokenStore) DenyJTI(_ context.Context, jti string, ttl time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.denyErr != nil {
		return f.denyErr
	}
	f.denied[jti] = ttl
	return nil
}

func (f *fakeTokenStore) IsJTIDenied(_ context.Context, jti string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.denied[jti]
	return ok, nil
}

func (f *fakeTokenStore) SetSession(_ context.Context, key string, value any, _ time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries[key] = b
	return nil
}

func (f *fakeTokenStore) GetSession(_ context.Context, key string, dest any) error {
	f.mu.Lock()
	raw, ok := f.entries[key]
	f.mu.Unlock()
	if !ok {
		return errors.New("not found")
	}
	return json.Unmarshal(raw, dest)
}

// RFC 6749 §4.1.2 requires revoking the tokens issued FROM a replayed code. The
// reuse branch only called RevokeByUserAndClient, which destroyed the user's
// unrelated refresh tokens on that client while leaving live the one credential
// the replay is racing for: the access token the first redemption minted.
func TestRevokeTokensIssuedFromCode(t *testing.T) {
	initTestJWTKeysService(t)
	ctx := context.Background()

	mintToken := func(t *testing.T) string {
		t.Helper()
		tok, err := jwt.GenerateAccessTokenWithContext(ctx,
			"user-1", "openid", "https://auth.example.com", "client-1", "client-1", "provider-1")
		require.NoError(t, err)
		return tok
	}

	authCode := &OAuthAuthorizationCode{TenantID: 1, UserID: 42}

	t.Run("the access token the code minted is denylisted on replay", func(t *testing.T) {
		store := newFakeTokenStore()
		svc := &oauthTokenService{jtiDenylist: store, authEventService: &mockAuthEventService{}}

		token := mintToken(t)
		svc.rememberTokenIssuedFromCode(ctx, "code-hash", token)

		wantJTI, _ := accessTokenJTIAndExpiry(token)
		require.NotEmpty(t, wantJTI)

		denied, err := store.IsJTIDenied(ctx, wantJTI)
		require.NoError(t, err)
		assert.False(t, denied, "issuing a token must not revoke it")

		svc.revokeTokensIssuedFromCode(ctx, "code-hash", authCode)

		denied, err = store.IsJTIDenied(ctx, wantJTI)
		require.NoError(t, err)
		assert.True(t, denied, "the replayed code's access token must be revoked")
	})

	t.Run("the denylist TTL does not outlive the token", func(t *testing.T) {
		store := newFakeTokenStore()
		svc := &oauthTokenService{jtiDenylist: store, authEventService: &mockAuthEventService{}}

		token := mintToken(t)
		svc.rememberTokenIssuedFromCode(ctx, "code-hash", token)
		svc.revokeTokensIssuedFromCode(ctx, "code-hash", authCode)

		wantJTI, _ := accessTokenJTIAndExpiry(token)
		store.mu.Lock()
		ttl := store.denied[wantJTI]
		store.mu.Unlock()
		assert.Greater(t, ttl, time.Duration(0))
		assert.LessOrEqual(t, ttl, jwt.AccessTokenTTL)
	})

	t.Run("an unknown code revokes nothing", func(t *testing.T) {
		store := newFakeTokenStore()
		svc := &oauthTokenService{jtiDenylist: store, authEventService: &mockAuthEventService{}}

		svc.revokeTokensIssuedFromCode(ctx, "never-seen", authCode)

		store.mu.Lock()
		defer store.mu.Unlock()
		assert.Empty(t, store.denied)
	})

	t.Run("a store without the wider capability degrades instead of failing", func(t *testing.T) {
		// The narrow no-op denylist used by most unit tests satisfies only
		// JTIDenylister; issuance must not break because of that.
		svc := &oauthTokenService{jtiDenylist: nopDenylister{}, authEventService: &mockAuthEventService{}}
		assert.NotPanics(t, func() {
			svc.rememberTokenIssuedFromCode(ctx, "code-hash", mintToken(t))
			svc.revokeTokensIssuedFromCode(ctx, "code-hash", authCode)
		})
	})
}

type nopDenylister struct{}

func (nopDenylister) DenyJTI(context.Context, string, time.Duration) error { return nil }
func (nopDenylister) IsJTIDenied(context.Context, string) (bool, error)    { return false, nil }

// A hardcoded amr/acr threw away what the session row already recorded: a user
// who had just completed TOTP got acr=1 and was re-challenged by every
// RequireStepUp route, and magic-link/SMS/passkey logins were asserted as
// password logins (RFC 8176, OIDC Core §2).
func TestResolveSessionAuthContext(t *testing.T) {
	ctx := context.Background()
	sessionUUID := uuid.New()

	t.Run("the session's real acr and amr are carried onto the token", func(t *testing.T) {
		authTime := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
		svc := &oauthTokenService{sessionAuthResolver: fakeSessionAuthResolver{
			result: &SessionAuthContext{
				ACR:      jwt.ACRLevel2,
				AMR:      []string{jwt.AMRWebAuthn},
				AuthTime: authTime,
			},
		}}

		facts := svc.resolveSessionAuthContext(ctx, &sessionUUID)
		assert.Equal(t, jwt.ACRLevel2, facts.ACR)
		assert.Equal(t, []string{jwt.AMRWebAuthn}, facts.AMR)
		assert.Equal(t, authTime, facts.AuthTime)
	})

	t.Run("no session falls back to single-factor password", func(t *testing.T) {
		svc := &oauthTokenService{sessionAuthResolver: fakeSessionAuthResolver{}}
		facts := svc.resolveSessionAuthContext(ctx, nil)
		assert.Equal(t, jwt.ACRLevel1, facts.ACR)
		assert.Equal(t, []string{jwt.AMRPassword}, facts.AMR)
	})

	t.Run("no resolver wired falls back to single-factor password", func(t *testing.T) {
		svc := &oauthTokenService{}
		facts := svc.resolveSessionAuthContext(ctx, &sessionUUID)
		assert.Equal(t, jwt.ACRLevel1, facts.ACR)
		assert.Equal(t, []string{jwt.AMRPassword}, facts.AMR)
	})

	t.Run("a resolver error falls back rather than failing issuance", func(t *testing.T) {
		svc := &oauthTokenService{sessionAuthResolver: fakeSessionAuthResolver{err: errors.New("db down")}}
		facts := svc.resolveSessionAuthContext(ctx, &sessionUUID)
		assert.Equal(t, jwt.ACRLevel1, facts.ACR)
		assert.Equal(t, []string{jwt.AMRPassword}, facts.AMR)
	})

	t.Run("a session missing acr keeps the fallback for that field only", func(t *testing.T) {
		svc := &oauthTokenService{sessionAuthResolver: fakeSessionAuthResolver{
			result: &SessionAuthContext{AMR: []string{jwt.AMRMagicLink}},
		}}
		facts := svc.resolveSessionAuthContext(ctx, &sessionUUID)
		assert.Equal(t, jwt.ACRLevel1, facts.ACR)
		assert.Equal(t, []string{jwt.AMRMagicLink}, facts.AMR)
	})
}

type fakeSessionAuthResolver struct {
	result *SessionAuthContext
	err    error
}

func (f fakeSessionAuthResolver) ResolveSessionAuthContext(context.Context, uuid.UUID) (*SessionAuthContext, error) {
	return f.result, f.err
}

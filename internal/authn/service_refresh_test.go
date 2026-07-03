package authn

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshToken(t *testing.T) {
	initTestJWTKeysService(t)

	const sub = "user-sub-1"
	const clientID = "test-client"
	const providerID = "test-provider"

	newRefreshToken := func(t *testing.T) string {
		t.Helper()
		tok, err := jwt.GenerateRefreshTokenWithContext(context.Background(), sub, "https://auth.example.com", clientID, providerID)
		require.NoError(t, err)
		return tok
	}

	userFound := func(t *testing.T) *mockUserRepo {
		return &mockUserRepo{findBySubAndClientIDFn: func(_, _ string) (*User, error) { return buildActiveUser(t, "pw"), nil }}
	}
	clientFound := &mockClientRepo{findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return buildActiveClient(), nil }}

	t.Run("empty token is rejected", func(t *testing.T) {
		svc := &loginService{}
		_, err := svc.RefreshToken(context.Background(), "   ", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "refresh token is required")
	})

	t.Run("malformed token is rejected", func(t *testing.T) {
		svc := &loginService{}
		_, err := svc.RefreshToken(context.Background(), "not-a-jwt", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid or expired refresh token")
	})

	t.Run("access token is rejected at refresh endpoint", func(t *testing.T) {
		access, _, _, err := generateTokenSetWithAuthContext(context.Background(), sub, buildActiveUser(t, "pw"), buildActiveClient(), tokenAuthContext{ACR: jwt.ACRLevel1})
		require.NoError(t, err)
		svc := &loginService{}
		_, err = svc.RefreshToken(context.Background(), access, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a refresh token")
	})

	t.Run("user not found", func(t *testing.T) {
		svc := &loginService{
			userRepo:       &mockUserRepo{findBySubAndClientIDFn: func(_, _ string) (*User, error) { return nil, nil }},
			clientRepo:     clientFound,
			sessionService: &mockSessionService{},
		}
		_, err := svc.RefreshToken(context.Background(), newRefreshToken(t), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("client not found", func(t *testing.T) {
		svc := &loginService{
			userRepo:       userFound(t),
			clientRepo:     &mockClientRepo{findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return nil, nil }},
			sessionService: &mockSessionService{},
		}
		_, err := svc.RefreshToken(context.Background(), newRefreshToken(t), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "client not found")
	})

	t.Run("happy path creates a new session and rotates the refresh token", func(t *testing.T) {
		denylist := &recordingLogoutJTIDenylister{}
		newSessUUID := uuid.New()
		svc := &loginService{
			userRepo:   userFound(t),
			clientRepo: clientFound,
			sessionService: &mockSessionService{
				createSessionFn: func(_ context.Context, _ int64, _, _ string) (*UserToken, error) {
					return &UserToken{UserTokenUUID: newSessUUID}, nil
				},
			},
			jtiDenylist: denylist,
		}
		resp, err := svc.RefreshToken(context.Background(), newRefreshToken(t), "")
		require.NoError(t, err)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.IDToken)
		assert.NotEmpty(t, resp.RefreshToken)
		require.NotNil(t, resp.SessionID)
		assert.Equal(t, newSessUUID.String(), *resp.SessionID)
		// The consumed refresh token must be denylisted (single-use rotation).
		assert.NotEmpty(t, denylist.jti, "consumed refresh token should be denylisted")
	})

	t.Run("reuses a valid supplied session", func(t *testing.T) {
		existing := uuid.New().String()
		var touched bool
		svc := &loginService{
			userRepo:   userFound(t),
			clientRepo: clientFound,
			sessionService: &mockSessionService{
				validateAndTouchFn: func(_ context.Context, sid uuid.UUID, _ int64) error {
					touched = true
					assert.Equal(t, existing, sid.String())
					return nil
				},
				createSessionFn: func(_ context.Context, _ int64, _, _ string) (*UserToken, error) {
					t.Fatal("must not create a new session when reusing a valid one")
					return nil, nil
				},
			},
		}
		resp, err := svc.RefreshToken(context.Background(), newRefreshToken(t), existing)
		require.NoError(t, err)
		assert.True(t, touched)
		require.NotNil(t, resp.SessionID)
		assert.Equal(t, existing, *resp.SessionID)
	})

	t.Run("invalid supplied session forces re-login", func(t *testing.T) {
		svc := &loginService{
			userRepo:   userFound(t),
			clientRepo: clientFound,
			sessionService: &mockSessionService{
				validateAndTouchFn: func(_ context.Context, _ uuid.UUID, _ int64) error { return assert.AnError },
			},
		}
		_, err := svc.RefreshToken(context.Background(), newRefreshToken(t), uuid.New().String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "session is no longer valid")
	})

	t.Run("malformed supplied session id is rejected", func(t *testing.T) {
		svc := &loginService{
			userRepo:       userFound(t),
			clientRepo:     clientFound,
			sessionService: &mockSessionService{},
		}
		_, err := svc.RefreshToken(context.Background(), newRefreshToken(t), "not-a-uuid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid session id")
	})
}

func TestSessionIDFromAccessToken(t *testing.T) {
	initTestJWTKeysService(t)

	t.Run("extracts sid from a token set", func(t *testing.T) {
		sid := uuid.New().String()
		access, _, _, err := generateTokenSetWithAuthContext(context.Background(), "user-sub-1", buildActiveUser(t, "pw"), buildActiveClient(), tokenAuthContext{ACR: jwt.ACRLevel1, SessionID: sid})
		require.NoError(t, err)
		assert.Equal(t, sid, sessionIDFromAccessToken(access))
	})

	t.Run("empty input returns empty", func(t *testing.T) {
		assert.Equal(t, "", sessionIDFromAccessToken("  "))
	})

	t.Run("garbage input returns empty", func(t *testing.T) {
		assert.Equal(t, "", sessionIDFromAccessToken("not-a-jwt"))
	})
}

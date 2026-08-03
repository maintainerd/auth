package authn

import (
	"context"
	"testing"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func refreshClaims(t *testing.T, token string) jwtlib.MapClaims {
	t.Helper()
	parsed, _, err := jwtlib.NewParser().ParseUnverified(token, jwtlib.MapClaims{})
	require.NoError(t, err)
	return parsed.Claims.(jwtlib.MapClaims)
}

// A refresh token must survive REPEATED rotation.
//
// This is the failure mode that only shows up in production: the first refresh
// works, so it looks fine in a smoke test, and users get signed out roughly one
// access-token lifetime later. Refresh tokens are now required to carry a `sid`,
// so if rotation dropped that claim the SECOND refresh would fail with "not
// bound to a session" and the identity app — the public-facing surface, where
// refresh actually matters — would log everyone out on their second hour.
func TestRefreshToken_SessionBindingSurvivesRepeatedRotation(t *testing.T) {
	initTestJWTKeysService(t)

	const sub = "user-sub-1"
	const clientID = "test-client"
	const providerID = "test-provider"

	boundSession := uuid.New()
	token, err := jwt.GenerateRefreshTokenWithOptionsContext(context.Background(), sub,
		"https://auth.example.com", clientID, providerID,
		&jwt.RefreshTokenOptions{SessionID: boundSession.String()})
	require.NoError(t, err)

	var validated []uuid.UUID
	svc := &loginService{
		userRepo:   &mockUserRepo{findBySubAndClientIDFn: func(_, _ string) (*User, error) { return buildActiveUser(t, "pw"), nil }},
		clientRepo: &mockClientRepo{findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return buildActiveClient(), nil }},
		sessionService: &mockSessionService{
			validateAndTouchFn: func(_ context.Context, id uuid.UUID, _ int64) error {
				validated = append(validated, id)
				return nil
			},
		},
		jtiDenylist: &recordingLogoutJTIDenylister{},
	}

	// Five hops: enough that a claim dropped on any single rotation shows up.
	for hop := 1; hop <= 5; hop++ {
		resp, err := svc.RefreshToken(context.Background(), token, "")
		require.NoErrorf(t, err, "refresh hop %d failed — the session binding was lost during rotation", hop)
		require.NotEmpty(t, resp.RefreshToken)

		claims := refreshClaims(t, resp.RefreshToken)
		assert.Equalf(t, boundSession.String(), claims["sid"],
			"hop %d: the rotated refresh token must carry the same sid, or the next refresh is rejected", hop)

		require.NotNil(t, resp.SessionID)
		assert.Equal(t, boundSession.String(), *resp.SessionID)

		token = resp.RefreshToken
	}

	// Every hop re-checked the session is still alive, so revoking it stops the
	// chain immediately rather than at the next full token expiry.
	assert.Len(t, validated, 5)
	for _, id := range validated {
		assert.Equal(t, boundSession, id)
	}
}

// The rotated token must also keep the family id, or reuse detection silently
// stops working after the first hop: each rotation would start a fresh family
// and replaying an old token would no longer implicate its siblings.
func TestRefreshToken_FamilyIDSurvivesRotation(t *testing.T) {
	initTestJWTKeysService(t)

	boundSession := uuid.New()
	token, err := jwt.GenerateRefreshTokenWithOptionsContext(context.Background(), "user-sub-1",
		"https://auth.example.com", "test-client", "test-provider",
		&jwt.RefreshTokenOptions{SessionID: boundSession.String()})
	require.NoError(t, err)

	svc := &loginService{
		userRepo:       &mockUserRepo{findBySubAndClientIDFn: func(_, _ string) (*User, error) { return buildActiveUser(t, "pw"), nil }},
		clientRepo:     &mockClientRepo{findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return buildActiveClient(), nil }},
		sessionService: &mockSessionService{},
		jtiDenylist:    &recordingLogoutJTIDenylister{},
	}

	first, err := svc.RefreshToken(context.Background(), token, "")
	require.NoError(t, err)
	familyOne := refreshClaims(t, first.RefreshToken)["rfid"]
	require.NotEmpty(t, familyOne)

	second, err := svc.RefreshToken(context.Background(), first.RefreshToken, "")
	require.NoError(t, err)
	assert.Equal(t, familyOne, refreshClaims(t, second.RefreshToken)["rfid"],
		"rotation must stay within one family, otherwise reuse detection cannot link siblings")
}

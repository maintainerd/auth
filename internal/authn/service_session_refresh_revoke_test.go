package authn

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingRefreshRevoker records the SCOPE of each revocation so a test can
// assert not just that refresh tokens were revoked, but that the right blast
// radius was used.
type recordingRefreshRevoker struct {
	byUser    []int64
	bySession []uuid.UUID
	err       error
}

func (r *recordingRefreshRevoker) RevokeByUserID(userID int64) (int64, error) {
	r.byUser = append(r.byUser, userID)
	return 1, r.err
}

func (r *recordingRefreshRevoker) RevokeBySession(sessionUUID uuid.UUID) (int64, error) {
	r.bySession = append(r.bySession, sessionUUID)
	return 1, r.err
}

// Revoking a session must revoke its refresh tokens too, and NOTHING wider.
//
// Two separate holes met here. Revoking a session did not touch refresh tokens
// at all, so the "ended" session could mint a fresh access token on its next
// refresh — logout, password reset and "sign out everywhere" all revoked
// sessions, and all of them were defeated by a token that outlived them. And
// the fix must not overcorrect: ending one browser has to leave the same user's
// other browsers and their phone signed in, or people get mysteriously kicked
// out of a device they never touched.
func TestSessionService_RevokeScopesRefreshTokens(t *testing.T) {
	ctx := context.Background()
	const userID int64 = 42

	t.Run("revoking one session revokes only that session's refresh tokens", func(t *testing.T) {
		sessionUUID := uuid.New()
		revoker := &recordingRefreshRevoker{}
		repo := &mockUserSessionRepo{
			findActiveByUUIDFn: func(_ int64, u uuid.UUID) (*UserSession, error) {
				return &UserSession{UserSessionUUID: u}, nil
			},
			revokeByUUIDFn: func(int64, uuid.UUID, string) error { return nil },
		}
		svc := NewSessionService(repo, revoker)

		require.NoError(t, svc.RevokeSession(ctx, userID, sessionUUID))

		assert.Equal(t, []uuid.UUID{sessionUUID}, revoker.bySession)
		assert.Empty(t, revoker.byUser,
			"a single logout must not revoke every token the user has — that signs them out of other browsers and mobile")
	})

	t.Run("sign out everywhere revokes every refresh token", func(t *testing.T) {
		revoker := &recordingRefreshRevoker{}
		repo := &mockUserSessionRepo{
			revokeAllByUserIDFn: func(int64, string) error { return nil },
		}
		svc := NewSessionService(repo, revoker)

		require.NoError(t, svc.RevokeAllSessions(ctx, userID, "user_revoke"))

		// This is the one place a global revoke is correct: an explicit
		// "everywhere" act, or a credential change routed through it.
		assert.Equal(t, []int64{userID}, revoker.byUser)
		assert.Empty(t, revoker.bySession)
	})

	t.Run("a refresh-revocation failure does not fail the logout", func(t *testing.T) {
		revoker := &recordingRefreshRevoker{err: assert.AnError}
		repo := &mockUserSessionRepo{
			findActiveByUUIDFn: func(_ int64, u uuid.UUID) (*UserSession, error) {
				return &UserSession{UserSessionUUID: u}, nil
			},
			revokeByUUIDFn: func(int64, uuid.UUID, string) error { return nil },
		}
		svc := NewSessionService(repo, revoker)

		// The session row is already gone, which is what actually ends the login.
		// Reporting a failure here would tell the user logout did not work when it
		// substantially did.
		assert.NoError(t, svc.RevokeSession(ctx, userID, uuid.New()))
	})

	t.Run("no revoker configured is not a crash", func(t *testing.T) {
		repo := &mockUserSessionRepo{
			revokeAllByUserIDFn: func(int64, string) error { return nil },
		}
		assert.NoError(t, NewSessionService(repo).RevokeAllSessions(ctx, userID, "logout"))
	})
}

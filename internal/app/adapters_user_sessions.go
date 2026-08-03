package app

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/maintainerd/maintainerd-auth/internal/authn"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/maintainerd/maintainerd-auth/internal/user"
)

// sessionRevokerAdapter satisfies user.SessionRevoker over the authn session
// repository. It exists for WithTx: the repository returns its own concrete
// type, which cannot satisfy an interface method declared to return
// user.SessionRevoker, so the conversion happens here rather than widening
// either domain's contract.
type sessionRevokerAdapter struct {
	sessions authn.UserSessionRepository
}

func (a sessionRevokerAdapter) WithTx(tx *gorm.DB) user.SessionRevoker {
	return sessionRevokerAdapter{sessions: a.sessions.WithTx(tx)}
}

func (a sessionRevokerAdapter) RevokeAllByUserID(userID int64, reason string) error {
	return a.sessions.RevokeAllByUserID(userID, reason)
}

func (a sessionRevokerAdapter) RevokeAllExceptUUID(userID int64, keepSessionUUID uuid.UUID, reason string) error {
	return a.sessions.RevokeAllExceptUUID(userID, keepSessionUUID, reason)
}

// userSessionBackedTokenRepo makes the user service's session-admin operations
// (list/revoke a user's active sessions) read and write the canonical
// user_sessions store — owned by authn — instead of the legacy user_tokens
// session rows, which are no longer written by the login flow (so the admin
// "Sessions" view was always empty). All non-session token methods delegate to
// the real user_tokens repository via the embedded interface.
type userSessionBackedTokenRepo struct {
	user.UserTokenRepository // token methods + WithTx delegate here
	sessions                 authn.UserSessionRepository
}

func newUserSessionBackedTokenRepo(
	tokens user.UserTokenRepository,
	sessions authn.UserSessionRepository,
) user.UserTokenRepository {
	return &userSessionBackedTokenRepo{UserTokenRepository: tokens, sessions: sessions}
}

// sessionToUserToken projects a canonical user_sessions row onto the UserToken
// shape the user service consumes for its admin session views.
func sessionToUserToken(s authn.UserSession) user.UserToken {
	expires := s.ExpiresAt
	lastActive := s.LastActiveAt
	idle := s.IdleTimeoutSeconds
	return user.UserToken{
		UserTokenUUID:      s.UserSessionUUID,
		UserID:             s.UserID,
		TokenType:          "session",
		UserAgent:          s.UserAgent,
		IPAddress:          s.IPAddress,
		IsRevoked:          s.RevokedAt != nil,
		ExpiresAt:          &expires,
		LastUsedAt:         &lastActive,
		IdleTimeoutSeconds: &idle,
		AbsoluteExpiresAt:  &expires,
		CreatedAt:          s.CreatedAt,
	}
}

func (r *userSessionBackedTokenRepo) FindActiveSessions(userID int64) ([]user.UserToken, error) {
	ss, err := r.sessions.FindActiveByUserID(userID)
	if err != nil {
		return nil, err
	}
	out := make([]user.UserToken, len(ss))
	for i := range ss {
		out[i] = sessionToUserToken(ss[i])
	}
	return out, nil
}

func (r *userSessionBackedTokenRepo) FindActiveSessionByUUID(userID int64, sessionUUID uuid.UUID) (*user.UserToken, error) {
	s, err := r.sessions.FindActiveByUUID(userID, sessionUUID)
	if err != nil || s == nil {
		return nil, err
	}
	t := sessionToUserToken(*s)
	return &t, nil
}

func (r *userSessionBackedTokenRepo) RevokeSessionByUUID(userID int64, sessionUUID uuid.UUID) error {
	return r.sessions.RevokeByUUID(userID, sessionUUID, shared.SessionRevokeAdmin)
}

func (r *userSessionBackedTokenRepo) RevokeAllSessionsByUserID(userID int64) error {
	return r.sessions.RevokeAllByUserID(userID, shared.SessionRevokeRoleChange)
}

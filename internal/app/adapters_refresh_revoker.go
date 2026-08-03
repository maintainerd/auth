package app

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/maintainerd/maintainerd-auth/internal/oauth"
	"github.com/maintainerd/maintainerd-auth/internal/user"
)

// refreshRevokerAdapter binds the OAuth refresh-token store to the authn and
// user packages.
//
// Those packages cannot import internal/oauth — oauth already imports both, so
// the edge would be a cycle. Each declares its own minimal RefreshTokenRevoker
// interface and this adapter, in internal/app (which imports everything),
// supplies the real repository.
//
// It exists at all because revoking sessions was never enough: a refresh token
// is a long-lived credential that survives its session, so logout, password
// change, password reset and "sign out everywhere" all dropped session rows
// while the refresh token kept minting fresh access tokens. Which tokens go is
// deliberately scoped by the caller — RevokeBySession for an ordinary logout
// (one browser), RevokeByUserID only for an explicit "everywhere" act or a
// credential change.
type refreshRevokerAdapter struct {
	tokens oauth.OAuthRefreshTokenRepository
}

func (a refreshRevokerAdapter) RevokeByUserID(userID int64) (int64, error) {
	return a.tokens.RevokeByUserID(userID)
}

func (a refreshRevokerAdapter) RevokeBySession(sessionUUID uuid.UUID) (int64, error) {
	return a.tokens.RevokeBySession(sessionUUID)
}

// WithTx lets a password change and its revocation commit or roll back
// together. The repository returns its own concrete type, which cannot satisfy
// an interface method declared to return user.RefreshTokenRevoker, so the
// conversion happens here rather than widening the oauth contract.
func (a refreshRevokerAdapter) WithTx(tx *gorm.DB) user.RefreshTokenRevoker {
	return refreshRevokerAdapter{tokens: a.tokens.WithTx(tx)}
}

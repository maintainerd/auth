package oauth

import (
	"context"
	"log/slog"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
)

// issuedTokenIndex is the small key/value capability needed to remember which
// access token an authorization code minted.
//
// It is discovered by type assertion on the injected JTI denylist rather than
// being a separate constructor parameter: the application already hands this
// service the shared *cache.Cache, which satisfies both, so the reuse defence
// works with no additional wiring. A store that only satisfies the narrow
// JTIDenylister (the test no-op) degrades to the previous behaviour instead of
// failing token issuance.
type issuedTokenIndex interface {
	SetSession(ctx context.Context, key string, value any, ttl time.Duration) error
	GetSession(ctx context.Context, key string, dest any) error
}

// codeIssuedTokenRecord is what gets stored per authorization code.
type codeIssuedTokenRecord struct {
	JTI       string    `json:"jti"`
	ExpiresAt time.Time `json:"expires_at"`
}

func codeIssuedTokenKey(codeHash string) string {
	return "oauth:code_issued_token:" + codeHash
}

// codeIndex returns the issued-token index when the injected store supports it.
func (s *oauthTokenService) codeIndex() issuedTokenIndex {
	idx, _ := s.jtiDenylist.(issuedTokenIndex)
	return idx
}

// accessTokenJTIAndExpiry reads the jti and exp out of a token this server just
// minted. The signature is not re-verified — the token was produced two lines
// earlier by this process, and the only alternative is threading the jti back
// out through every issuance helper.
func accessTokenJTIAndExpiry(accessToken string) (string, time.Time) {
	claims, _, err := jwt.ParseTokenUnverified(accessToken)
	if err != nil {
		return "", time.Time{}
	}
	jti, _ := claims["jti"].(string)
	if jti == "" {
		return "", time.Time{}
	}
	return jti, time.Now().Add(tokenRemainingTTL(claims["exp"]))
}

// rememberTokenIssuedFromCode records the access token an authorization code
// produced, keyed by the code hash, for the token's remaining lifetime.
//
// The record is only useful while the token it names is still valid, so it
// expires with it. Failures are logged and swallowed: a token has already been
// handed to the client at this point, so failing the exchange here would report
// an error for a grant that in fact succeeded.
func (s *oauthTokenService) rememberTokenIssuedFromCode(ctx context.Context, codeHash, accessToken string) {
	idx := s.codeIndex()
	if idx == nil || accessToken == "" {
		return
	}
	jti, expiresAt := accessTokenJTIAndExpiry(accessToken)
	if jti == "" {
		return
	}
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return
	}
	if err := idx.SetSession(ctx, codeIssuedTokenKey(codeHash), codeIssuedTokenRecord{
		JTI:       jti,
		ExpiresAt: expiresAt,
	}, ttl); err != nil {
		slog.Warn("could not index the access token issued for this authorization code; a later code replay will not be able to revoke it",
			"error", err)
	}
}

// revokeTokensIssuedFromCode denylists the access token the FIRST redemption of
// this code produced, which is what RFC 6749 §4.1.2 means by revoking the tokens
// issued from a replayed code.
func (s *oauthTokenService) revokeTokensIssuedFromCode(ctx context.Context, codeHash string, authCode *OAuthAuthorizationCode) {
	idx := s.codeIndex()
	if idx == nil || s.jtiDenylist == nil {
		return
	}
	var record codeIssuedTokenRecord
	if err := idx.GetSession(ctx, codeIssuedTokenKey(codeHash), &record); err != nil || record.JTI == "" {
		// Absent is the common case once the token has expired on its own.
		return
	}
	ttl := time.Until(record.ExpiresAt)
	if ttl <= 0 {
		return
	}
	if err := s.jtiDenylist.DenyJTI(ctx, record.JTI, ttl); err != nil {
		slog.Error("authorization code replay detected but the access token it issued could NOT be revoked",
			"error", err)
		return
	}

	var tenantID int64
	var actorUserID *int64
	if authCode != nil {
		tenantID = authCode.TenantID
		actorUserID = &authCode.UserID
	}
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
		ActorUserID: actorUserID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeOAuthTokenRevoke,
		Severity:    authevent.AuthEventSeverityCritical,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr("Access token issued from a replayed authorization code was revoked"),
	})
}

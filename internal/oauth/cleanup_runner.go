package oauth

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/maintainerd/maintainerd-auth/internal/authn"
	"github.com/maintainerd/maintainerd-auth/internal/invite"
	"github.com/maintainerd/maintainerd-auth/internal/mfa"
	"github.com/maintainerd/maintainerd-auth/internal/notifier"
	"github.com/maintainerd/maintainerd-auth/internal/user"
)

func StartCleanupRunner(ctx context.Context, db *gorm.DB, interval time.Duration) {
	authCodeRepo := NewOAuthAuthorizationCodeRepository(db)
	refreshTokenRepo := NewOAuthRefreshTokenRepository(db)
	consentChallengeRepo := NewOAuthConsentChallengeRepository(db)
	parRequestRepo := NewOAuthPARRequestRepository(db)
	cibaRequestRepo := NewOAuthCIBARequestRepository(db)
	deviceCodeRepo := NewOAuthDeviceCodeRepository(db)
	authorizeRequestRepo := NewOAuthAuthorizeRequestRepository(db)
	brokerSessionRepo := NewOAuthBrokerSessionRepository(db)
	userTokenRepo := user.NewUserTokenRepository(db)
	userOTPRepo := notifier.NewUserOTPRepository(db)
	inviteRepo := invite.NewInviteRepository(db)
	revocationRepo := NewOAuthTokenRevocationRepository(db)
	webAuthnChallengeRepo := mfa.NewWebAuthnChallengeRepository(db)
	accountLinkRepo := authn.NewAccountLinkRequestRepository(db)
	dpopNonceRepo := NewOAuthDPoPNonceRepository(db)
	userSessionRepo := authn.NewUserSessionRepository(db)
	trustedDeviceRepo := user.NewUserTrustedDeviceRepository(db)
	lockoutRepo := authn.NewUserLockoutRepository(db)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := time.Now()
				inviteCutoff := now.Add(-30 * 24 * time.Hour)

				n, err := authCodeRepo.DeleteExpired(now)
				logCleanup("oauth_authorization_codes", n, err)

				n, err = refreshTokenRepo.DeleteExpired(now)
				logCleanup("oauth_refresh_tokens", n, err)

				n, err = consentChallengeRepo.DeleteExpired(now)
				logCleanup("oauth_consent_challenges", n, err)

				n, err = parRequestRepo.DeleteExpired(now)
				logCleanup("oauth_par_requests", n, err)

				n, err = cibaRequestRepo.DeleteExpired(now)
				logCleanup("oauth_ciba_requests", n, err)

				n, err = deviceCodeRepo.DeleteExpired(now)
				logCleanup("oauth_device_codes", n, err)

				n, err = authorizeRequestRepo.DeleteExpired(now)
				logCleanup("oauth_authorize_requests", n, err)

				if err := userTokenRepo.DeleteExpiredTokens(now); err != nil {
					slog.Warn("cleanup runner: delete failed", "table", "user_tokens", "err", err)
				}

				n, err = userOTPRepo.DeleteExpired(now)
				logCleanup("user_otps", n, err)

				n, err = inviteRepo.DeleteExpired(inviteCutoff)
				logCleanup("invites", n, err)

				n, err = revocationRepo.DeleteExpired()
				logCleanup("oauth_token_revocations", n, err)

				n, err = webAuthnChallengeRepo.DeleteExpired()
				logCleanup("webauthn_challenges", n, err)

				n, err = accountLinkRepo.ExpireStale(now)
				logCleanup("account_link_requests", n, err)

				n, err = dpopNonceRepo.DeleteExpired()
				logCleanup("oauth_dpop_nonces", n, err)

				n, err = brokerSessionRepo.DeleteExpired(now)
				logCleanup("oauth_broker_sessions", n, err)

				n, err = userSessionRepo.DeleteExpired()
				logCleanup("user_sessions", n, err)

				n, err = trustedDeviceRepo.DeleteExpired()
				logCleanup("user_trusted_devices", n, err)

				n, err = lockoutRepo.ResetExpiredLockouts()
				logCleanup("user_lockouts (reset)", n, err)
			}
		}
	}()
}

func logCleanup(table string, deleted int64, err error) {
	if err != nil {
		slog.Warn("cleanup runner: delete failed", "table", table, "err", err)
	} else if deleted > 0 {
		slog.Info("cleanup runner: cleaned expired rows", "table", table, "count", deleted)
	}
}

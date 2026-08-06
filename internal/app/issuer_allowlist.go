package app

import (
	"log/slog"
	"strings"

	"gorm.io/gorm"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
)

// seedAcceptedIssuers loads the issuer allowlist that validateTokenClaims
// enforces.
//
// The `iss` claim on an issued token is the CLIENT's domain (see
// authn.generateTokenSetWithAuthContext and oauth.oauthAccessTokenOptions), not
// one server-wide hostname — so the allowlist is the set of registered client
// domains and can only come from the database.
//
// Without this the allowlist stays empty, and the jwt package then falls back
// to accepting ONLY the authorization server's own issuer — correct but strict:
// tokens minted under a client domain (deployments predating jwt.TokenIssuer,
// or any deployment with APP_PUBLIC_HOSTNAME unset) would stop validating.
// Seeding it restores those without ever widening the check to "any issuer".
//
// New and updated clients register their domain through registerClientIssuer,
// so the set stays current without a restart.
func seedAcceptedIssuers(db *gorm.DB) {
	if db == nil {
		// The wiring test constructs services without a database.
		return
	}
	var domains []string
	err := db.
		Table("clients").
		Where("domain IS NOT NULL AND domain <> '' AND deleted_at IS NULL").
		Distinct().
		Pluck("domain", &domains).Error
	if err != nil {
		// Deliberately non-fatal: refusing to boot because the allowlist could
		// not be read would turn a transient DB hiccup into an outage, and the
		// jwt package fails closed on an empty set — it accepts only this
		// server's own issuer, never an arbitrary one.
		slog.Error("issuer allowlist: could not load client domains; only the server's own issuer will be accepted", "error", err)
		return
	}
	// The authorization server's own issuer is what tokens now carry
	// (jwt.TokenIssuer); the client domains stay accepted so tokens minted before
	// that change, and any deployment with APP_PUBLIC_HOSTNAME unset, still
	// validate.
	if issuer := strings.TrimSpace(config.AppPublicHostname); issuer != "" {
		domains = append(domains, issuer)
	}
	if len(domains) == 0 {
		slog.Warn("issuer allowlist: no issuers resolved; only the server's own issuer will be accepted")
		return
	}
	jwt.SetAcceptedIssuers(domains)
	slog.Info("issuer allowlist loaded", "count", len(domains))
}

package jwt

import (
	"strings"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
)

// TokenIssuer returns the `iss` value every token this server mints must carry.
//
// The issuer identifies the AUTHORIZATION SERVER, not the client (OIDC Core §2,
// RFC 8414 §2). Tokens used to be stamped with the requesting client's own
// domain, which broke discovery outright: a relying party fetches
// {issuer}/.well-known/openid-configuration, reads the `issuer` there, and per
// OIDC Core §3.1.3.7 step 2 MUST reject any token whose `iss` does not match it
// exactly. Since discovery advertises APP_PUBLIC_HOSTNAME while tokens carried
// the client domain, every spec-compliant third-party client rejected every
// token — and two clients in one tenant produced two different issuers for the
// same authorization server, which is not a thing an issuer is allowed to be.
//
// clientDomain is the legacy fallback, used only when APP_PUBLIC_HOSTNAME is
// unset, so a misconfigured deployment still mints a non-empty issuer instead
// of an unusable one.
func TokenIssuer(clientDomain string) string {
	if issuer := strings.TrimSpace(config.AppPublicHostname); issuer != "" {
		return issuer
	}
	return clientDomain
}

// TokenIssuerPtr is TokenIssuer for the many call sites holding a *string
// client domain.
func TokenIssuerPtr(clientDomain *string) string {
	if clientDomain == nil {
		return TokenIssuer("")
	}
	return TokenIssuer(*clientDomain)
}

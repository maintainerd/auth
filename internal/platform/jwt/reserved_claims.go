package jwt

import "strings"

// reservedClaims are claims a client-configured mapper must never set.
//
// Custom claim mappers (clients.claim_mappers) are merged into tokens LAST so an
// operator can add organisation-specific claims. Without this denylist that merge
// is a token-forgery primitive: a mapper of {"sub":"<victim>","permissions":["*"]}
// would produce a token signed by the real authorization-server key that
// impersonates any subject, in any tenant, with any permission — and
// {"exp":9999999999} would make it never expire.
//
// The list covers three groups:
//   - JWT registered claims (RFC 7519 §4.1)
//   - OIDC claims that carry authentication meaning (OIDC Core §2, §3.1.3.6)
//   - claims THIS system authorizes on: tenant, permissions, roles, session
var reservedClaims = map[string]struct{}{
	// RFC 7519 registered claims.
	"iss": {}, "sub": {}, "aud": {}, "exp": {}, "nbf": {}, "iat": {}, "jti": {},
	// OIDC authentication context — forging these defeats step-up and nonce binding.
	"azp": {}, "nonce": {}, "acr": {}, "amr": {}, "auth_time": {},
	"at_hash": {}, "c_hash": {}, "s_hash": {},
	// Authorization inputs this server trusts when validating a request.
	"scope": {}, "scp": {}, "client_id": {}, "sub_type": {},
	"tenant_id": {}, "permissions": {}, "roles": {}, "sid": {},
	// Sender-constrained token binding (RFC 9449 / RFC 8705).
	"cnf": {},
	// Service-principal identity. The gRPC interceptor admits a token when
	// sub_type=="service" and then authorizes on `svc` as the principal name,
	// resolving its policies WITHOUT tenant scoping — so a forged `svc` is an
	// arbitrary service principal in any tenant. `provider_id` and `token_type`
	// are stamped by the issuer and read back by middleware; `act` is the RFC 8693
	// delegation chain, which must record who actually delegated.
	"svc": {}, "provider_id": {}, "token_type": {}, "act": {}, "token_use": {},
}

// IsReservedClaim reports whether a claim name is protected from client-supplied
// claim mappers. Comparison is case-insensitive: JSON keys are case-sensitive, so
// "Sub" would be a different claim, but accepting it invites a consumer that
// lowercases keys to treat it as the real one.
func IsReservedClaim(name string) bool {
	_, ok := reservedClaims[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

// SanitizeClientClaimMappers strips reserved names from a CLIENT-CONFIGURED claim
// mapper set (clients.claim_mappers).
//
// Apply it where the mappers are read, not where claims are merged: the merge path
// is also how this server stamps its own claims (tenant_id, permissions), so
// filtering there would block legitimate internal values. This function guards the
// untrusted boundary only.
//
// Dropping a reserved claim is deliberate rather than returning an error: token
// issuance sits on the login path, so a misconfigured mapper must degrade to a
// correct token rather than deny service. Returns nil when nothing survives, so
// callers can leave ExtraClaims unset.
func SanitizeClientClaimMappers(mappers map[string]any) map[string]any {
	if len(mappers) == 0 {
		return nil
	}
	safe := make(map[string]any, len(mappers))
	for k, v := range mappers {
		if IsReservedClaim(k) {
			continue
		}
		safe[k] = v
	}
	if len(safe) == 0 {
		return nil
	}
	return safe
}

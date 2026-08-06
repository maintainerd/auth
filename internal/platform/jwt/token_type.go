package jwt

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

// Values of the `token_type` claim this server stamps at issuance. Every token
// minted here carries exactly one of them, which is what makes it possible to
// refuse a token at an endpoint it was never meant for.
const (
	TokenTypeAccess  = "access_token"
	TokenTypeDPoP    = "DPoP" // an access token additionally bound to a DPoP key (RFC 9449 §6.1)
	TokenTypeID      = "id_token"
	TokenTypeRefresh = "refresh_token"
)

// ErrWrongTokenType is returned when a token is structurally valid and correctly
// signed but is not of the type the calling endpoint accepts.
var ErrWrongTokenType = errors.New("token is not an access token")

// IsAccessTokenType reports whether a `token_type` claim value denotes an access
// token. DPoP-bound access tokens carry "DPoP" instead of "access_token"
// (RFC 9449 §6.1 replaces the type when a cnf.jkt is present), so both are
// access tokens for authorization purposes.
func IsAccessTokenType(tokenType string) bool {
	switch tokenType {
	case TokenTypeAccess, TokenTypeDPoP:
		return true
	}
	return false
}

// ValidateAccessToken validates a token AND requires it to be an access token.
func ValidateAccessToken(tokenString string) (jwtlib.MapClaims, error) {
	return ValidateAccessTokenWithContext(context.Background(), tokenString)
}

// ValidateAccessTokenWithContext is the entry point every request-authentication
// path must use. ValidateTokenWithContext deliberately does not check what KIND
// of token it was handed — the refresh, id_token_hint, logout-token and
// introspection paths all need to validate their own type — so on the
// authorization path the type check has to happen here.
//
// Without it an ID token authenticates as its subject. Every token this server
// issues is signed with the same key and an ID token carries the same `sub` and
// `client_id` that the user-context middleware resolves on, so signature +
// sub/aud/iss/iat/exp/jti alone cannot tell the two apart. ID tokens are handed
// to relying parties by design (that is their entire purpose), and an ID token
// carries no `sid`, so one replayed as a Bearer token would also outlive logout,
// session revocation and password change for its full lifetime.
//
// RFC 9068 §4 (JWT access tokens) and OIDC Core §2 draw exactly this line: an
// ID token is an authentication receipt for the client, never an authorization
// credential for a resource server.
func ValidateAccessTokenWithContext(ctx context.Context, tokenString string) (jwtlib.MapClaims, error) {
	claims, err := ValidateTokenWithContext(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	tokenType, _ := claims["token_type"].(string)
	if !IsAccessTokenType(tokenType) {
		// The value is issuer-stamped and non-secret, so naming it aids debugging
		// without leaking anything. A token with no token_type at all (step-up
		// challenge handles, or anything minted outside the standard helpers) is
		// rejected by the same branch — an unlabelled token is not an access token.
		if tokenType == "" {
			return nil, ErrWrongTokenType
		}
		return nil, fmt.Errorf("%w (token_type=%q)", ErrWrongTokenType, tokenType)
	}
	return claims, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Issuer allowlist
// ──────────────────────────────────────────────────────────────────────────────

// acceptedIssuers is the set of `iss` values this server will accept on its own
// tokens.
//
// RFC 7519 §4.1.1 requires a recipient to reject a token whose issuer it does
// not recognize; validateTokenClaims previously only checked that `iss` was
// non-empty. That is a real gap here because every tenant's tokens are signed
// with ONE process-wide key (see keyStore), so the signature alone proves
// nothing about which tenant an assertion came from.
//
// The set is empty until an operator populates it. An empty set does NOT mean
// "accept anything": see validateIssuerClaim, which falls back to the single
// issuer the process can always vouch for on its own — the authorization
// server's own issuer — and rejects everything else. The unconfigured state is
// logged once so it is visible rather than silent.
var acceptedIssuers struct {
	mu     sync.RWMutex
	values map[string]struct{}
	warned bool
}

// SetAcceptedIssuers replaces the issuer allowlist. Call once at startup with
// every issuer this deployment mints tokens under (the server's public
// hostname plus each tenant's configured client domain).
func SetAcceptedIssuers(issuers []string) {
	acceptedIssuers.mu.Lock()
	defer acceptedIssuers.mu.Unlock()
	if len(issuers) == 0 {
		acceptedIssuers.values = nil
		return
	}
	set := make(map[string]struct{}, len(issuers))
	for _, iss := range issuers {
		if iss = strings.TrimSpace(iss); iss != "" {
			set[iss] = struct{}{}
		}
	}
	acceptedIssuers.values = set
}

// AddAcceptedIssuer adds one issuer to the allowlist. Safe to call as tenants
// are provisioned at runtime.
func AddAcceptedIssuer(issuer string) {
	issuer = strings.TrimSpace(issuer)
	if issuer == "" {
		return
	}
	acceptedIssuers.mu.Lock()
	defer acceptedIssuers.mu.Unlock()
	if acceptedIssuers.values == nil {
		acceptedIssuers.values = map[string]struct{}{}
	}
	acceptedIssuers.values[issuer] = struct{}{}
}

// IsSelfIssued reports whether `iss` is an issuer THIS deployment mints tokens
// under.
//
// Tokens carry the CLIENT's domain as `iss`, so a deployment has as many
// issuers as it has clients — there is no single hostname to compare against.
// The multi-issuer middleware needs this to tell "our own token" from "a
// federated partner's token"; getting it wrong sends first-party tokens down
// the federation path, where the issuer resolves to no registered IdP and the
// request is rejected.
//
// Returns false when the allowlist is unconfigured, so the caller keeps
// whatever fallback it had rather than silently classifying everything as
// first-party.
func IsSelfIssued(iss string) bool {
	iss = strings.TrimRight(strings.TrimSpace(iss), "/")
	if iss == "" {
		return false
	}
	acceptedIssuers.mu.RLock()
	defer acceptedIssuers.mu.RUnlock()
	for known := range acceptedIssuers.values {
		if strings.TrimRight(known, "/") == iss {
			return true
		}
	}
	return false
}

// ResetAcceptedIssuers clears the allowlist. Intended for tests.
func ResetAcceptedIssuers() {
	acceptedIssuers.mu.Lock()
	defer acceptedIssuers.mu.Unlock()
	acceptedIssuers.values = nil
	acceptedIssuers.warned = false
}

// validateIssuerClaim matches `iss` against the allowlist and fails CLOSED when
// the allowlist is empty.
//
// It used to return nil for an empty allowlist ("unconfigured" was read as
// "accept anything"). That made the issuer check a no-op for every token the
// process validated whenever seeding failed — a transient DB error in
// seedAcceptedIssuers, or a startup path that never calls SetAcceptedIssuers,
// is enough. With one process-wide signing key the `iss` match IS the tenant
// boundary (see validateTokenClaims), so a config failure silently removing it
// is precisely the outcome a security check must not have.
//
// The fallback when the allowlist is empty is the authorization server's own
// issuer — APP_PUBLIC_HOSTNAME, a required env var, and the `iss` every token
// this server mints now carries (see TokenIssuer). That keeps a deployment
// whose seeding failed serving its own tokens while still refusing a token
// minted under any other issuer. If APP_PUBLIC_HOSTNAME is unset too there is
// no issuer this process can vouch for, so every token is rejected.
func validateIssuerClaim(iss string) error {
	acceptedIssuers.mu.RLock()
	values := acceptedIssuers.values
	acceptedIssuers.mu.RUnlock()

	if len(values) == 0 {
		warnIssuerAllowlistUnconfigured()
		self := strings.TrimRight(strings.TrimSpace(TokenIssuer("")), "/")
		if self != "" && strings.TrimRight(iss, "/") == self {
			return nil
		}
		return fmt.Errorf("issuer (iss) claim %q is not a recognized issuer (issuer allowlist is not configured)", iss)
	}
	if _, ok := values[iss]; !ok {
		return fmt.Errorf("issuer (iss) claim %q is not a recognized issuer", iss)
	}
	return nil
}

// warnIssuerAllowlistUnconfigured logs the unconfigured allowlist exactly once.
// It is once-per-process on purpose: the condition is per-deployment, not
// per-request, and logging it per token would bury the rest of the log under a
// message that never changes.
func warnIssuerAllowlistUnconfigured() {
	acceptedIssuers.mu.RLock()
	warned := acceptedIssuers.warned
	acceptedIssuers.mu.RUnlock()
	if warned {
		return
	}
	acceptedIssuers.mu.Lock()
	defer acceptedIssuers.mu.Unlock()
	if acceptedIssuers.warned {
		return
	}
	acceptedIssuers.warned = true
	slog.Warn("JWT issuer allowlist is not configured; only the authorization server's own issuer (APP_PUBLIC_HOSTNAME) will be accepted. Call jwt.SetAcceptedIssuers at startup.")
}

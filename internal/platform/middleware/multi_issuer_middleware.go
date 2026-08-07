package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	jwtlib "github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

type FederatedIDPRecord struct {
	IdentityProviderID   int64
	TenantID             int64
	Provider             string
	AllowTokenFederation bool
	AllowJITProvisioning bool
	Status               string
}

type FederatedAudienceRecord struct {
	Audience string
}

type FederatedPrincipal struct {
	UserID   int64
	UserUUID string
	TenantID int64
}

type FederatedPrincipalResolver func(ctx context.Context, rawToken string, idpID int64, audAllowed func(aud string) bool) (*FederatedPrincipal, error)
type FederatedUserContextLoader func(ctx context.Context, userID int64, tenantID int64) (*authctx.AuthContext, error)
type FederatedIDPLookup func(issuer string) (*FederatedIDPRecord, error)
type FederatedAudienceLookup func(idpID int64) ([]FederatedAudienceRecord, error)

type multiIssuerTokenCache struct {
	mu       sync.RWMutex
	idpByIss map[string]cachedFederatedIDP
	ttl      time.Duration
}

type cachedFederatedIDP struct {
	idp    *FederatedIDPRecord
	expiry time.Time
}

func newMultiIssuerTokenCache(ttl time.Duration) *multiIssuerTokenCache {
	return &multiIssuerTokenCache{
		idpByIss: make(map[string]cachedFederatedIDP),
		ttl:      ttl,
	}
}

func (c *multiIssuerTokenCache) get(iss string) *FederatedIDPRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.idpByIss[iss]
	if !ok || time.Now().After(entry.expiry) {
		return nil
	}
	return entry.idp
}

func (c *multiIssuerTokenCache) set(iss string, idp *FederatedIDPRecord) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.idpByIss[iss] = cachedFederatedIDP{idp: idp, expiry: time.Now().Add(c.ttl)}
}

func MultiIssuerAuthMiddleware(
	idpLookup FederatedIDPLookup,
	audienceLookup FederatedAudienceLookup,
	resolvePrincipal FederatedPrincipalResolver,
	loadUserContext FederatedUserContextLoader,
) func(http.Handler) http.Handler {
	cache := newMultiIssuerTokenCache(5 * time.Minute)
	maintainerdIssuer := getMaintainerdIssuer()
	maintainerdPrefix := strings.TrimRight(maintainerdIssuer, "/")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			// "DPoP" is recognized alongside "Bearer" so a sender-constrained token
			// reaches the binding check below instead of falling through unvalidated.
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") && !strings.EqualFold(parts[0], "dpop") {
				next.ServeHTTP(w, r)
				return
			}
			scheme := strings.ToLower(parts[0])
			rawToken := strings.TrimSpace(parts[1])
			if rawToken == "" {
				next.ServeHTTP(w, r)
				return
			}

			if iss := peekIss(rawToken); iss != "" && isFirstPartyIssuer(iss, maintainerdIssuer, maintainerdPrefix) {
				claims, err := jwtlib.ValidateTokenWithContext(r.Context(), rawToken)
				if err != nil {
					w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
					resp.Error(w, http.StatusUnauthorized, "Invalid or expired token")
					return
				}
				// A token bound to a DPoP key needs its proof here too, or this path
				// becomes the way around the binding.
				if bindErr := enforceDPoPBinding(r, scheme, rawToken, claims); bindErr != nil {
					w.Header().Set("WWW-Authenticate", `DPoP error="invalid_token"`)
					resp.Error(w, http.StatusUnauthorized, "Invalid token", bindErr.Error())
					return
				}
				next.ServeHTTP(w, r.WithContext(ContextWithJWTClaims(r.Context(), buildJWTClaims(r.Context(), claims))))
				return
			}

			var iss string
			if rawClaims, err := peekIssAndClaims(rawToken); err == nil {
				iss, _ = rawClaims["iss"].(string)
			}
			if isFirstPartyIssuer(iss, maintainerdIssuer, maintainerdPrefix) {
				next.ServeHTTP(w, r)
				return
			}

			idp := cache.get(iss)
			if idp == nil {
				found, err := idpLookup(iss)
				if err != nil || found == nil {
					respError401(w, "Invalid token")
					return
				}
				if found.Status != "active" || !found.AllowTokenFederation {
					respError401(w, "Invalid token")
					return
				}
				idp = found
				cache.set(iss, idp)
			}

			audiences, err := audienceLookup(idp.IdentityProviderID)
			if err != nil {
				respError401(w, "Invalid token")
				return
			}
			allowed := make(map[string]bool, len(audiences))
			for _, a := range audiences {
				allowed[a.Audience] = true
			}

			principal, err := resolvePrincipal(r.Context(), rawToken, idp.IdentityProviderID, func(aud string) bool {
				return allowed[aud]
			})
			if err != nil {
				respError401(w, "Invalid token")
				return
			}

			auth, err := loadUserContext(r.Context(), principal.UserID, principal.TenantID)
			if err != nil || auth == nil {
				respError401(w, "Invalid token")
				return
			}
			auth.Provider = &authctx.AuthProvider{
				IdentityProviderID: idp.IdentityProviderID,
			}
			r = WithAuthContext(r, auth)
			next.ServeHTTP(w, r)
		})
	}
}

func respError401(w http.ResponseWriter, msg string) {
	w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
	resp.Error(w, http.StatusUnauthorized, msg)
}

func peekIssAndClaims(rawToken string) (map[string]interface{}, error) {
	claims, _, err := jwtlib.ParseTokenUnverified(rawToken)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func getMaintainerdIssuer() string {
	issuer := config.AppPublicHostname
	if issuer == "" {
		issuer = "maintainerd-auth"
	}
	return issuer
}

func peekIss(rawToken string) string {
	claims, _, err := jwtlib.ParseTokenUnverified(rawToken)
	if err != nil {
		return ""
	}
	iss, _ := claims["iss"].(string)
	return iss
}

// isFirstPartyIssuer reports whether a token was minted by THIS deployment.
//
// Tokens carry the CLIENT's domain as `iss`, so a deployment has one issuer per
// client — comparing against a single APP_PUBLIC_HOSTNAME cannot recognise them.
// When that variable is unset the fallback is the literal "maintainerd-auth",
// which matches no real issuer at all, so every first-party Bearer token was
// classified as federated, looked up as an external IdP, and rejected 401.
// Cookie-authenticated requests skip this middleware entirely, which is why the
// SPAs kept working and only API clients using Authorization: Bearer broke.
//
// The issuer allowlist is the authoritative set of our own issuers (seeded from
// the registered clients at startup and kept current as clients are created),
// so it is consulted first; the hostname comparison stays as a fallback for a
// deployment that has not populated the allowlist yet.
func isFirstPartyIssuer(iss, maintainerdIssuer, maintainerdPrefix string) bool {
	// An unparseable or issuer-less token is NOT ours. peekIss returns "" for
	// anything it cannot parse, so treating that as first-party classified junk
	// as trusted and passed it downstream unvalidated. It survived only because
	// the per-route JWT middleware re-validates — a fail-closed default costs
	// nothing and removes the dependency on that second line of defence.
	if iss == "" {
		return false
	}
	if jwtlib.IsSelfIssued(iss) {
		return true
	}
	// Exact match, or a path UNDER our issuer — never a bare prefix.
	//
	// A bare HasPrefix made "https://<our-host>.evil.com" first-party, because it
	// starts with our hostname. Signature verification stops that being
	// exploitable today, but it would silently mis-route a legitimate federated
	// IdP whose issuer happens to share our prefix, and it is the wrong rule to
	// leave sitting in front of an authorization decision. Requiring a "/"
	// boundary means only a genuine sub-path matches.
	trimmed := strings.TrimRight(iss, "/")
	if trimmed == maintainerdIssuer || trimmed == maintainerdPrefix {
		return true
	}
	return strings.HasPrefix(trimmed, maintainerdPrefix+"/")
}

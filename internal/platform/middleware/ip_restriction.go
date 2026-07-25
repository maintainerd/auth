package middleware

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

type IPRestriction struct {
	Type      string
	IPAddress string
}

type TenantIPRestrictionReader interface {
	GetActiveIPRestrictions(ctx context.Context, tenantID int64) ([]IPRestriction, error)
}

// TenantSlugResolver maps a subdomain tenant slug to its tenant ID. The slug is
// the authoritative, server-derived tenant identity for the public auth surface
// (see request_tenant.go). ok is false when the slug names no tenant.
type TenantSlugResolver interface {
	ResolveTenantIDBySlug(ctx context.Context, slug string) (tenantID int64, ok bool, err error)
}

const ipRestrictionCacheTTL = 5 * time.Second

// TenantIPRestrictionMiddleware enforces a tenant's IP allow/deny rules on
// AUTHENTICATED routes, where auth.Tenant is already set from the session. It
// fails OPEN on a cold rule-load error (kept as-is: an authenticated user
// mid-session should not be ejected by a transient store blip; the login surface
// — see AuthEndpointIPRestrictionMiddleware — is the one that fails closed).
func TenantIPRestrictionMiddleware(reader TenantIPRestrictionReader) func(http.Handler) http.Handler {
	cache := newIPRestrictionCache()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if reader == nil {
				next.ServeHTTP(w, r)
				return
			}
			auth := AuthFromRequest(r)
			if auth == nil || auth.Tenant == nil {
				next.ServeHTTP(w, r)
				return
			}
			// ok ignored → cold error yields nil rules → allow (fail-open).
			rules, _ := cache.get(r.Context(), reader, auth.Tenant.TenantID)
			if !enforceIPRules(w, r, rules) {
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// AuthEndpointIPRestrictionMiddleware enforces a tenant's IP allow/deny rules on
// the PRE-AUTH credential surface (login, register, password reset, SMS login,
// magic link). This is the primary purpose of an IP allowlist for an IAM —
// restricting WHERE users may authenticate from — which the post-auth middleware
// above cannot cover because no session (and thus no auth.Tenant) exists yet.
//
// The enforcement tenant is derived STRICTLY from the request's own subdomain
// (ResolveRequestTenantTrusted → slug → tenant ID), never from the request body
// or a caller-supplied client_id, so an attacker cannot select an unrestricted
// tenant to bypass a restricted one. Rules are loaded and evaluated per that one
// tenant only, so one tenant's allowlist can never affect another tenant.
//
// Fail posture (product decision): a tenant whose rules cannot be loaded must
// NOT be reachable — it may have an allowlist we cannot verify. The cache serves
// the last known ruleset through transient blips (per-tenant), and only a cold
// load error with no cached state fails closed (503). A request whose host names
// no tenant is allowed through (there is no tenant to scope rules to; legitimate
// tenant logins always arrive on the tenant subdomain) but is logged.
func AuthEndpointIPRestrictionMiddleware(reader TenantIPRestrictionReader, resolver TenantSlugResolver) func(http.Handler) http.Handler {
	cache := newIPRestrictionCache()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if reader == nil || resolver == nil {
				next.ServeHTTP(w, r)
				return
			}

			rt := ResolveRequestTenantTrusted(r)
			if !rt.OK {
				// No tenant subdomain on the request → nothing to scope rules to.
				next.ServeHTTP(w, r)
				return
			}

			tenantID, ok, err := resolver.ResolveTenantIDBySlug(r.Context(), rt.Slug)
			if err != nil || !ok {
				// The slug did not resolve to a tenant. Not a rule-load failure —
				// there is simply no tenant to enforce against.
				next.ServeHTTP(w, r)
				return
			}

			rules, loaded := cache.get(r.Context(), reader, tenantID)
			if !loaded {
				// Cold load error for an identified tenant: fail CLOSED. The tenant
				// may restrict logins to specific IPs and we cannot verify it, so we
				// must not admit the request. Scoped to this tenant only.
				slog.Error("ip restriction: failing closed, could not load rules",
					"tenant_id", tenantID, "path", r.URL.Path)
				resp.Error(w, http.StatusServiceUnavailable, "Access temporarily unavailable")
				return
			}
			if !enforceIPRules(w, r, rules) {
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// enforceIPRules evaluates the client IP against rules, writing a 403 and
// returning false when denied. len(rules)==0 means the tenant has no policy and
// the request proceeds.
func enforceIPRules(w http.ResponseWriter, r *http.Request, rules []IPRestriction) bool {
	if len(rules) == 0 {
		return true
	}
	if !ipAllowed(extractClientIP(r), rules) {
		resp.Error(w, http.StatusForbidden, "Access denied by IP restriction policy")
		return false
	}
	return true
}

func ipAllowed(clientIP string, rules []IPRestriction) bool {
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return true
	}

	var hasDeny, hasAllow bool
	for _, rule := range rules {
		match := matchIP(ip, rule.IPAddress)
		switch rule.Type {
		case "deny":
			hasDeny = true
			if match {
				return false
			}
		case "allow":
			hasAllow = true
		}
	}

	if hasDeny && !hasAllow {
		return true
	}

	if hasAllow {
		for _, rule := range rules {
			if rule.Type == "allow" && matchIP(ip, rule.IPAddress) {
				return true
			}
		}
		return false
	}

	return true
}

func matchIP(clientIP net.IP, ruleAddr string) bool {
	_, cidr, err := net.ParseCIDR(ruleAddr)
	if err == nil {
		return cidr.Contains(clientIP)
	}
	ruleIP := net.ParseIP(ruleAddr)
	if ruleIP == nil {
		return false
	}
	return clientIP.Equal(ruleIP)
}

type ipRestrictionCacheEntry struct {
	rules     []IPRestriction
	expiresAt time.Time
}

type ipRestrictionCache struct {
	ttl     time.Duration
	mu      sync.Mutex
	entries map[int64]ipRestrictionCacheEntry
}

func newIPRestrictionCache() *ipRestrictionCache {
	return &ipRestrictionCache{ttl: ipRestrictionCacheTTL, entries: map[int64]ipRestrictionCacheEntry{}}
}

// get returns the tenant's active rules. loaded is false ONLY on a cold load
// error — i.e., the store errored and there is no cached entry to fall back on.
// A transient error with a cached entry serves the stale rules (fail to last
// known state) so a blip neither over-blocks a no-rules tenant nor drops a
// restricted tenant's allowlist.
func (c *ipRestrictionCache) get(ctx context.Context, reader TenantIPRestrictionReader, tenantID int64) (rules []IPRestriction, loaded bool) {
	now := time.Now()
	c.mu.Lock()
	entry, has := c.entries[tenantID]
	c.mu.Unlock()
	if has && now.Before(entry.expiresAt) {
		return entry.rules, true
	}

	fresh, err := reader.GetActiveIPRestrictions(ctx, tenantID)
	if err != nil {
		if has {
			// Serve stale through the transient error.
			return entry.rules, true
		}
		return nil, false
	}

	c.mu.Lock()
	c.entries[tenantID] = ipRestrictionCacheEntry{rules: fresh, expiresAt: now.Add(c.ttl)}
	c.mu.Unlock()
	return fresh, true
}

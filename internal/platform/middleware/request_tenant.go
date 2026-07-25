package middleware

import (
	"context"
	"net/http"
	"net/url"

	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

// RequestTenant is the tenant a browser request is actually on, derived
// server-side from the request's own host — NEVER from a caller-supplied
// tenant_id. It closes the cross-tenant hole where an external app could drive a
// client_id from tenant A while the browser is on tenant B's subdomain.
type RequestTenant struct {
	// Surface is the resolved frontend surface (identity/console).
	Surface string
	// Slug is the tenant DNS label; empty for the system-tenant (bare) host.
	Slug string
	// IsSystem is true when the request host is the bare system-tenant host.
	IsSystem bool
	// OK is true when the request host matched a configured base (so Surface,
	// Slug and IsSystem are meaningful). When false the host is unrecognized and
	// callers must fall back to their existing tenant-resolution behavior.
	OK bool
}

type requestTenantKey struct{}

// ResolveRequestTenant derives the request's tenant from the request itself,
// topology-agnostically. The tenant subdomain reaches the backend differently
// per deployment:
//
//   - Cross-origin (API on its own host): the browser sends the tenant surface
//     in the Origin header.
//   - Same-origin via per-app nginx (dev and same-origin prod): there is no
//     Origin on same-origin requests, but nginx forwards the tenant host in the
//     Host header (proxy_set_header Host $host) and may also set
//     X-Forwarded-Host.
//
// It therefore tries, in order, the host from Origin, then X-Forwarded-Host,
// then Host, and returns the first that shared.ResolveTenantHost recognizes. The
// base host is org-configured; no domain label (".auth."/".console.") is ever
// hardcoded.
func ResolveRequestTenant(r *http.Request) RequestTenant {
	candidates := []string{
		hostFromOrigin(r.Header.Get("Origin")),
		r.Header.Get("X-Forwarded-Host"),
		r.Host,
	}
	for _, h := range candidates {
		if h == "" {
			continue
		}
		if surface, slug, isSystem, ok := shared.ResolveTenantHost(h); ok {
			return RequestTenant{Surface: surface, Slug: slug, IsSystem: isSystem, OK: true}
		}
	}
	return RequestTenant{}
}

// ResolveRequestTenantTrusted is ResolveRequestTenant hardened for use as a
// SECURITY decision (IP restriction on the login surface). It trusts the
// forwarded host headers (Origin, X-Forwarded-Host) ONLY when the peer is a
// trusted proxy — the same discipline extractClientIP applies to
// X-Forwarded-For. Direct-to-origin, only the real Host is honored, so an
// attacker who can reach the origin cannot spoof a different (unrestricted)
// tenant via a forged X-Forwarded-Host header. The plain ResolveRequestTenant
// stays for non-security tenant hints (e.g. branding) where a spoofed host is
// harmless.
func ResolveRequestTenantTrusted(r *http.Request) RequestTenant {
	var candidates []string
	if trustAllProxies || isTrustedProxy(remoteAddrIP(r)) {
		candidates = append(candidates, hostFromOrigin(r.Header.Get("Origin")), r.Header.Get("X-Forwarded-Host"))
	}
	candidates = append(candidates, r.Host)
	for _, h := range candidates {
		if h == "" {
			continue
		}
		if surface, slug, isSystem, ok := shared.ResolveTenantHost(h); ok {
			return RequestTenant{Surface: surface, Slug: slug, IsSystem: isSystem, OK: true}
		}
	}
	return RequestTenant{}
}

// hostFromOrigin extracts the host component from an Origin header value
// (scheme://host[:port]). Returns "" when the value is empty or unparseable.
func hostFromOrigin(origin string) string {
	if origin == "" {
		return ""
	}
	u, err := url.Parse(origin)
	if err != nil {
		return ""
	}
	return u.Host
}

// WithRequestTenant stores the resolved request tenant in ctx.
func WithRequestTenant(ctx context.Context, rt RequestTenant) context.Context {
	return context.WithValue(ctx, requestTenantKey{}, rt)
}

// RequestTenantFromContext returns the request tenant stored by
// RequestTenantMiddleware. ok is false when no request tenant was stored.
func RequestTenantFromContext(ctx context.Context) (RequestTenant, bool) {
	rt, ok := ctx.Value(requestTenantKey{}).(RequestTenant)
	return rt, ok
}

// RequestTenantMiddleware resolves the request's tenant from its own host and
// stores it in the context so downstream handlers/services can make the request
// host authoritative for tenant binding. It never rejects a request; an
// unrecognized host is stored with OK=false so callers fall back to their
// existing behavior.
func RequestTenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rt := ResolveRequestTenant(r)
		next.ServeHTTP(w, r.WithContext(WithRequestTenant(r.Context(), rt)))
	})
}

package shared

import (
	"sort"
	"strings"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
)

// Frontend surfaces. The identity surface is the public hosted login UI; the
// console surface is the internal admin dashboard.
const (
	FrontendSurfaceIdentity = "identity"
	FrontendSurfaceConsole  = "console"
)

// FrontendURL builds an absolute https URL to a tenant's frontend surface.
//
// Multi-tenancy is subdomain-based: the tenant name is the DNS slug. The system
// tenant uses the bare configured host (e.g. auth.maintainerd.local); every
// other tenant is served from {name}.{systemHost}
// (e.g. acme.auth.maintainerd.local, acme.console.auth.maintainerd.local).
//
// The configured host comes from APP_FRONTEND_IDENTITY_HOSTNAME /
// APP_FRONTEND_CONSOLE_HOSTNAME and holds the SYSTEM-tenant host. Any scheme on
// the configured value is stripped and normalized to https so callers never
// need to reason about the scheme.
func FrontendURL(surface, tenantName string, isSystem bool, path string) string {
	systemHost := config.AppFrontendIdentityHostname
	if surface == FrontendSurfaceConsole {
		systemHost = config.AppFrontendConsoleHostname
	}

	// Normalize: strip any scheme and trailing slash from the configured host.
	host := strings.TrimSpace(systemHost)
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimRight(host, "/")

	// Regular tenants are served from their subdomain; the system tenant uses
	// the bare host.
	if !isSystem && tenantName != "" {
		host = tenantName + "." + host
	}

	return "https://" + host + path
}

// normalizeHost reduces a configured base or an incoming Host header to a bare,
// comparable DNS name: lowercased, without scheme, path, port, or trailing dot.
func normalizeHost(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	if i := strings.Index(h, "://"); i >= 0 {
		h = h[i+3:]
	}
	// Drop any path (a configured base may be stored scheme-ful with a path).
	if i := strings.IndexByte(h, '/'); i >= 0 {
		h = h[:i]
	}
	// Drop an explicit port. Hostnames here are DNS names, never bracketed IPv6.
	if i := strings.LastIndexByte(h, ':'); i >= 0 {
		h = h[:i]
	}
	return strings.Trim(h, ".")
}

// ResolveTenantHost is the inverse of FrontendURL: given an incoming host it
// reports which frontend surface it belongs to, the tenant slug (empty for the
// system tenant), and whether the host is the system-tenant host.
//
// The base host per surface is configured (org-configurable, open source) via
// APP_FRONTEND_CONSOLE_HOSTNAME / APP_FRONTEND_IDENTITY_HOSTNAME and holds the
// SYSTEM-tenant host. The base is never assumed to contain any particular label
// (e.g. ".auth." / ".console."); resolution works purely by stripping the
// configured base:
//
//   - An exact match of a base → the system tenant on that surface.
//   - A single-DNS-label prefix followed by "."+base → a regular tenant whose
//     slug is that prefix.
//
// Bases are checked most-specific (longest) first so an overlapping shorter
// base cannot shadow a longer one (e.g. identity=auth.x.com,
// console=console.auth.x.com). ok is false when the host matches no base.
func ResolveTenantHost(host string) (surface string, slug string, isSystem bool, ok bool) {
	h := normalizeHost(host)
	if h == "" {
		return "", "", false, false
	}

	type baseHost struct {
		host    string
		surface string
	}
	bases := []baseHost{
		{normalizeHost(config.AppFrontendConsoleHostname), FrontendSurfaceConsole},
		{normalizeHost(config.AppFrontendIdentityHostname), FrontendSurfaceIdentity},
	}
	// Most-specific (longest) base first.
	sort.SliceStable(bases, func(i, j int) bool {
		return len(bases[i].host) > len(bases[j].host)
	})

	// Exact match → system tenant on that surface.
	for _, b := range bases {
		if b.host != "" && h == b.host {
			return b.surface, "", true, true
		}
	}

	// Suffix match → subdomain tenant. The stripped prefix must be a single DNS
	// label (no dot) so a deeper host cannot masquerade as a tenant slug.
	for _, b := range bases {
		if b.host == "" {
			continue
		}
		suffix := "." + b.host
		if strings.HasSuffix(h, suffix) {
			prefix := strings.TrimSuffix(h, suffix)
			if prefix != "" && !strings.Contains(prefix, ".") {
				return b.surface, prefix, false, true
			}
		}
	}

	return "", "", false, false
}

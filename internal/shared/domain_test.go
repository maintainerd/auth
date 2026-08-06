package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// SameRegistrableDomain decides whether a client reaches the account-management
// surface, so the cases that matter most are the ones an attacker picks: a host
// that merely CONTAINS the real domain, and a suffix that looks shared but is a
// public suffix two tenants happen to sit under.
func TestSameRegistrableDomain(t *testing.T) {
	tests := []struct {
		name  string
		a     string
		b     string
		match bool
	}{
		// The real deployment shape: the console and the auth API are different
		// hosts on one site, which is why the browser shares the cookie.
		{"subdomains of one site match", "https://console.auth.maintainerd.local", "identity-api.auth.maintainerd.local", true},
		{"identical hosts match", "example.com", "example.com", true},
		{"scheme, port and trailing dot are not part of the site", "https://Console.Example.com.:8443", "example.com", true},
		{"deep subdomain still matches", "a.b.c.example.com", "example.com", true},

		// Attacker-shaped inputs.
		{"suffix lookalike does not match", "example.com.evil.net", "example.com", false},
		{"prefix lookalike does not match", "notexample.com", "example.com", false},
		{"different site does not match", "evil.net", "example.com", false},
		{"embedded target in path is not a host", "evil.net/example.com", "example.com", false},
		{"userinfo does not smuggle a host", "example.com@evil.net", "example.com", false},

		// Multi-label public suffixes: a "last two labels" rule would call these
		// the same site and hand one tenant the other tenant's account surface.
		{"two tenants under co.uk are different sites", "foo.co.uk", "bar.co.uk", false},
		{"same site under co.uk matches", "app.foo.co.uk", "foo.co.uk", true},
		{"two users on github.io are different sites", "alice.github.io", "bob.github.io", false},

		// A bare ICANN public suffix has no registrable domain; if it matched,
		// every client on that suffix would become first-party.
		{"bare public suffix matches nothing", "com", "com", false},
		{"bare multi-label public suffix matches nothing", "co.uk", "co.uk", false},
		// ".local" is not ICANN-managed, so a bare "local" is treated as an
		// ordinary single-label host and matches only itself. That is degenerate
		// config (it would need APP_PUBLIC_HOSTNAME to be literally "local") and
		// grants nothing an exact string comparison would not.
		{"bare non-ICANN label matches only itself", "local", "local", true},
		{"bare non-ICANN label is not a real host under it", "local", "maintainerd.local", false},

		// Unknown or absent input must never be a match — this gates a privilege.
		{"empty is never a match", "", "", false},
		{"empty against a real host", "", "example.com", false},
		{"whitespace is never a match", "   ", "example.com", false},

		// IPs compare literally: sharing an octet prefix is not sharing a site.
		{"same IP matches", "10.0.0.1", "10.0.0.1", true},
		{"different IPs do not match", "10.0.0.1", "10.0.0.2", false},
		{"IPv6 with port matches itself", "[::1]:8080", "::1", true},

		// Single-label hosts are legitimate in development.
		{"localhost matches localhost", "http://localhost:3000", "localhost", true},
		{"localhost is not example.com", "localhost", "example.com", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.match, SameRegistrableDomain(tc.a, tc.b))
			// The relation is symmetric; an asymmetry would mean the answer
			// depended on argument order, which no caller could reason about.
			assert.Equal(t, tc.match, SameRegistrableDomain(tc.b, tc.a), "must be symmetric")
		})
	}
}

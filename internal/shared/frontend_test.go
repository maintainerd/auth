package shared

import (
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
)

// withBases sets the configured surface bases for the duration of a test and
// restores them afterwards.
func withBases(t *testing.T, identity, console string) {
	t.Helper()
	origIdentity := config.AppFrontendIdentityHostname
	origConsole := config.AppFrontendConsoleHostname
	config.AppFrontendIdentityHostname = identity
	config.AppFrontendConsoleHostname = console
	t.Cleanup(func() {
		config.AppFrontendIdentityHostname = origIdentity
		config.AppFrontendConsoleHostname = origConsole
	})
}

func TestResolveTenantHost(t *testing.T) {
	tests := []struct {
		name         string
		identityBase string
		consoleBase  string
		host         string
		wantSurface  string
		wantSlug     string
		wantSystem   bool
		wantOK       bool
	}{
		{
			name:         "identity system exact match",
			identityBase: "https://auth.maintainerd.local",
			consoleBase:  "https://console.auth.maintainerd.local",
			host:         "auth.maintainerd.local",
			wantSurface:  FrontendSurfaceIdentity,
			wantSystem:   true,
			wantOK:       true,
		},
		{
			name:         "console system exact match",
			identityBase: "https://auth.maintainerd.local",
			consoleBase:  "https://console.auth.maintainerd.local",
			host:         "console.auth.maintainerd.local",
			wantSurface:  FrontendSurfaceConsole,
			wantSystem:   true,
			wantOK:       true,
		},
		{
			name:         "identity tenant subdomain",
			identityBase: "https://auth.maintainerd.local",
			consoleBase:  "https://console.auth.maintainerd.local",
			host:         "acme.auth.maintainerd.local",
			wantSurface:  FrontendSurfaceIdentity,
			wantSlug:     "acme",
			wantOK:       true,
		},
		{
			name:         "console tenant subdomain",
			identityBase: "https://auth.maintainerd.local",
			consoleBase:  "https://console.auth.maintainerd.local",
			host:         "acme.console.auth.maintainerd.local",
			wantSurface:  FrontendSurfaceConsole,
			wantSlug:     "acme",
			wantOK:       true,
		},
		{
			name:         "overlap: longer console base wins over identity base",
			identityBase: "https://auth.x.com",
			consoleBase:  "https://console.auth.x.com",
			host:         "acme.console.auth.x.com",
			wantSurface:  FrontendSurfaceConsole,
			wantSlug:     "acme",
			wantOK:       true,
		},
		{
			name:         "overlap: identity subdomain still resolves to identity",
			identityBase: "https://auth.x.com",
			consoleBase:  "https://console.auth.x.com",
			host:         "acme.auth.x.com",
			wantSurface:  FrontendSurfaceIdentity,
			wantSlug:     "acme",
			wantOK:       true,
		},
		{
			name:         "overlap: identity system exact match not shadowed by console",
			identityBase: "https://auth.x.com",
			consoleBase:  "https://console.auth.x.com",
			host:         "auth.x.com",
			wantSurface:  FrontendSurfaceIdentity,
			wantSystem:   true,
			wantOK:       true,
		},
		{
			name:         "custom console base exact match",
			identityBase: "https://auth.ssi.com",
			consoleBase:  "https://console-auth.ssi.com",
			host:         "console-auth.ssi.com",
			wantSurface:  FrontendSurfaceConsole,
			wantSystem:   true,
			wantOK:       true,
		},
		{
			name:         "custom console base tenant subdomain",
			identityBase: "https://auth.ssi.com",
			consoleBase:  "https://console-auth.ssi.com",
			host:         "acme.console-auth.ssi.com",
			wantSurface:  FrontendSurfaceConsole,
			wantSlug:     "acme",
			wantOK:       true,
		},
		{
			name:         "host with scheme port and trailing dot is normalized",
			identityBase: "https://auth.maintainerd.local",
			consoleBase:  "https://console.auth.maintainerd.local",
			host:         "https://ACME.auth.maintainerd.local.:8081",
			wantSurface:  FrontendSurfaceIdentity,
			wantSlug:     "acme",
			wantOK:       true,
		},
		{
			name:         "multi-label prefix is rejected (not a single slug)",
			identityBase: "https://auth.maintainerd.local",
			consoleBase:  "https://console.auth.maintainerd.local",
			host:         "a.b.auth.maintainerd.local",
			wantOK:       false,
		},
		{
			name:         "unrelated host does not match",
			identityBase: "https://auth.maintainerd.local",
			consoleBase:  "https://console.auth.maintainerd.local",
			host:         "evil.example.com",
			wantOK:       false,
		},
		{
			name:         "empty host does not match",
			identityBase: "https://auth.maintainerd.local",
			consoleBase:  "https://console.auth.maintainerd.local",
			host:         "",
			wantOK:       false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withBases(t, tc.identityBase, tc.consoleBase)

			surface, slug, isSystem, ok := ResolveTenantHost(tc.host)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if surface != tc.wantSurface {
				t.Errorf("surface = %q, want %q", surface, tc.wantSurface)
			}
			if slug != tc.wantSlug {
				t.Errorf("slug = %q, want %q", slug, tc.wantSlug)
			}
			if isSystem != tc.wantSystem {
				t.Errorf("isSystem = %v, want %v", isSystem, tc.wantSystem)
			}
		})
	}
}

// TestResolveTenantHostRoundTrip verifies ResolveTenantHost inverts FrontendURL.
func TestResolveTenantHostRoundTrip(t *testing.T) {
	withBases(t, "https://auth.maintainerd.local", "https://console.auth.maintainerd.local")

	cases := []struct {
		surface  string
		slug     string
		isSystem bool
	}{
		{FrontendSurfaceIdentity, "", true},
		{FrontendSurfaceConsole, "", true},
		{FrontendSurfaceIdentity, "acme", false},
		{FrontendSurfaceConsole, "acme", false},
	}

	for _, c := range cases {
		url := FrontendURL(c.surface, c.slug, c.isSystem, "")
		surface, slug, isSystem, ok := ResolveTenantHost(url)
		if !ok {
			t.Fatalf("FrontendURL(%q,%q,%v)=%q did not resolve", c.surface, c.slug, c.isSystem, url)
		}
		if surface != c.surface || slug != c.slug || isSystem != c.isSystem {
			t.Errorf("round trip of %q: got (%q,%q,%v), want (%q,%q,%v)",
				url, surface, slug, isSystem, c.surface, c.slug, c.isSystem)
		}
	}
}

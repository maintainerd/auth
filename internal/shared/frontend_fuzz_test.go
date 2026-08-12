package shared

import "testing"

// FuzzResolveTenantHost exercises the request-host → tenant resolver, which parses
// attacker-controllable Host / Origin / X-Forwarded-Host values. It must never panic
// on arbitrary input, and must uphold its structural invariants: a recognized host
// always yields a non-empty surface, and the system tenant always has an empty slug.
func FuzzResolveTenantHost(f *testing.F) {
	for _, seed := range []string{
		"", "auth.example.com", "console.auth.example.com",
		"acme.console.auth.example.com", "acme.identity.auth.example.com",
		"://", "a.b.c.d.e", "HTTPS://Auth.Example.Com", "auth.example.com:443",
		"..auth.example.com", "\x00\n\r", "evil.com",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, host string) {
		surface, slug, isSystem, ok := ResolveTenantHost(host)
		if ok && surface == "" {
			t.Fatalf("ResolveTenantHost(%q): ok with empty surface (slug=%q system=%v)", host, slug, isSystem)
		}
		if ok && isSystem && slug != "" {
			t.Fatalf("ResolveTenantHost(%q): system tenant must have an empty slug, got %q", host, slug)
		}
	})
}

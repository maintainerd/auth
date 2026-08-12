package mfa

import (
	"strings"
	"testing"
)

// FuzzRPIDFromHostname exercises the WebAuthn RP-ID derivation, which parses a
// configured hostname. It must never panic, and it must strip the scheme and any
// port — so the derived RP ID never contains a ':'.
func FuzzRPIDFromHostname(f *testing.F) {
	for _, seed := range []string{
		"", "https://auth.example.com", "http://x.example.com:8080",
		"auth.example.com", "auth.example.com:443", "://",
		"a\nb", "HTTPS://Auth.Example.Com", "\x00", "[::1]:8443",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, hostname string) {
		host := rpIDFromHostname(hostname)
		if strings.ContainsRune(host, ':') {
			t.Fatalf("rpIDFromHostname(%q) leaked a colon/port: %q", hostname, host)
		}
	})
}

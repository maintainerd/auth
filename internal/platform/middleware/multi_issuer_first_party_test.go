package middleware

import (
	"testing"

	jwtlib "github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
)

// A first-party token must never be routed down the federation path.
//
// Tokens carry the CLIENT's domain as `iss`, so a deployment has one issuer per
// client. The check used to compare against a single APP_PUBLIC_HOSTNAME, whose
// fallback is the literal "maintainerd-auth" — matching no real issuer — so
// every first-party Bearer token was treated as a foreign federated token,
// resolved against no registered IdP, and rejected with 401. Cookie auth skips
// this middleware, so the SPAs kept working and only API clients broke.
func TestIsFirstPartyIssuer(t *testing.T) {
	t.Cleanup(jwtlib.ResetAcceptedIssuers)

	const fallbackIssuer = "maintainerd-auth"
	const clientIssuer = "https://identity.auth.maintainerd.local"

	t.Run("a registered client domain is first-party", func(t *testing.T) {
		jwtlib.ResetAcceptedIssuers()
		jwtlib.SetAcceptedIssuers([]string{clientIssuer})

		if !isFirstPartyIssuer(clientIssuer, fallbackIssuer, fallbackIssuer) {
			t.Fatal("a token issued under a registered client domain must be recognised as our own")
		}
	})

	t.Run("a trailing slash does not change the verdict", func(t *testing.T) {
		jwtlib.ResetAcceptedIssuers()
		jwtlib.SetAcceptedIssuers([]string{clientIssuer + "/"})

		if !isFirstPartyIssuer(clientIssuer, fallbackIssuer, fallbackIssuer) {
			t.Fatal("issuer comparison must ignore a trailing slash")
		}
	})

	t.Run("an unrelated issuer stays federated", func(t *testing.T) {
		jwtlib.ResetAcceptedIssuers()
		jwtlib.SetAcceptedIssuers([]string{clientIssuer})

		if isFirstPartyIssuer("https://accounts.google.com", fallbackIssuer, fallbackIssuer) {
			t.Fatal("a partner issuer must still go down the federation path")
		}
	})

	t.Run("falls back to the hostname when the allowlist is empty", func(t *testing.T) {
		jwtlib.ResetAcceptedIssuers()

		if !isFirstPartyIssuer("maintainerd-auth", fallbackIssuer, fallbackIssuer) {
			t.Fatal("an unconfigured allowlist must not break the hostname fallback")
		}
		if isFirstPartyIssuer("https://accounts.google.com", fallbackIssuer, fallbackIssuer) {
			t.Fatal("an empty allowlist must not classify everything as first-party")
		}
	})

	// INVERTED. This used to assert an absent issuer is first-party. peekIss
	// returns "" for any token it cannot parse, so that classified junk as ours
	// and passed it downstream unvalidated — safe only because the per-route JWT
	// middleware re-validates. Failing closed removes that dependency.
	t.Run("a missing or unparseable issuer is not first-party", func(t *testing.T) {
		jwtlib.ResetAcceptedIssuers()
		if isFirstPartyIssuer("", fallbackIssuer, fallbackIssuer) {
			t.Fatal("a token whose issuer cannot be read must not be treated as our own")
		}
	})
}

// A bare prefix match made "https://<our-host>.evil.com" first-party, because it
// literally starts with our hostname. Only an exact match or a genuine sub-path
// may qualify.
func TestIsFirstPartyIssuerRequiresAPathBoundary(t *testing.T) {
	jwtlib.ResetAcceptedIssuers()
	t.Cleanup(jwtlib.ResetAcceptedIssuers)

	const ours = "https://auth.example.com"

	if !isFirstPartyIssuer(ours, ours, ours) {
		t.Fatal("our own issuer must match exactly")
	}
	if !isFirstPartyIssuer(ours+"/realms/tenant-a", ours, ours) {
		t.Fatal("a genuine sub-path of our issuer must match")
	}
	for _, impostor := range []string{
		"https://auth.example.com.evil.com",
		"https://auth.example.com-evil.com",
		"https://auth.example.comX",
	} {
		if isFirstPartyIssuer(impostor, ours, ours) {
			t.Fatalf("%q must not be treated as first-party", impostor)
		}
	}
}

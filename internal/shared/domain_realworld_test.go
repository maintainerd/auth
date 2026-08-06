package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The classification this deployment actually produces, pinned against the real
// client rows and APP_PUBLIC_HOSTNAME. The unit tests above prove the rule; this
// proves the rule gives the ANSWER this product needs — that swapping is_system
// for a domain check did not quietly change who reaches the account surface.
func TestFirstPartyClassificationForThisDeployment(t *testing.T) {
	const publicHostname = "identity-api.auth.maintainerd.local"

	tests := []struct {
		name         string
		clientDomain string
		firstParty   bool
	}{
		// The two seeded system clients. They were first-party under the old
		// is_system rule and must stay first-party under the domain rule, or the
		// admin console loses the account surface.
		{"auth-console", "https://console.auth.maintainerd.local", true},
		{"auth-identity", "https://identity.auth.maintainerd.local", true},

		// An ecosystem component deployed on this domain is first-party for the
		// same reason the console is: the browser would share the cookie with it.
		{"core on the same site", "https://core.maintainerd.local", true},

		// A tenant's own application is third-party over the WEB boundary and
		// reaches the account surface through consented OAuth, not the cookie.
		{"a tenant's application", "https://app.acme.com", false},

		// The lookalike that a suffix comparison would wrongly admit.
		{"suffix lookalike", "https://maintainerd.local.evil.net", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.firstParty, SameRegistrableDomain(tc.clientDomain, publicHostname))
		})
	}
}

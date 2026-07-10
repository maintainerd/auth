package seeder

import (
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
)

// TestFrontendURLPerTenant verifies the per-tenant frontend host derivation used
// when seeding client URIs. The system tenant uses the bare configured host; a
// regular tenant is served from its {name}. subdomain, and any scheme on the
// configured value is normalized to https.
func TestFrontendURLPerTenant(t *testing.T) {
	config.AppFrontendConsoleHostname = "https://console.auth.maintainerd.local"
	config.AppFrontendIdentityHostname = "auth.maintainerd.local"

	assert.Equal(t,
		"https://console.auth.maintainerd.local/auth/callback",
		shared.FrontendURL(shared.FrontendSurfaceConsole, "maintainerd", true, "/auth/callback"),
	)
	assert.Equal(t,
		"https://acme.auth.maintainerd.local/callback",
		shared.FrontendURL(shared.FrontendSurfaceIdentity, "acme", false, "/callback"),
	)
}

package setup

import (
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
)

// Both predicates read package-level config, so a test that flips one has to put
// it back or it leaks into every test that runs after it.

func withControlPlaneEnabled(t *testing.T, enabled bool) {
	t.Helper()
	orig := config.ControlPlaneEnabled
	t.Cleanup(func() { config.ControlPlaneEnabled = orig })
	config.ControlPlaneEnabled = enabled
}

func withConfiguredCredential(t *testing.T, token string) {
	t.Helper()
	orig := config.SetupBootstrapToken
	t.Cleanup(func() { config.SetupBootstrapToken = orig })
	config.SetupBootstrapToken = token
}

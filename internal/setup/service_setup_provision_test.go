package setup

import (
	"context"
	"testing"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The setup window is the only period in this instance's life when tenants,
// clients and policies can be created with nothing but the bootstrap credential.
// These are the properties that keep that window bounded.
func TestSetupWindowTTL(t *testing.T) {
	withWindow := func(t *testing.T, ttl time.Duration, started time.Time) {
		t.Helper()
		origTTL, origStart, origCP := setupWindowTTL, setupProcessStart, bootstrapControlPlaneEnabled
		t.Cleanup(func() {
			setupWindowTTL, setupProcessStart, bootstrapControlPlaneEnabled = origTTL, origStart, origCP
		})
		setupWindowTTL = func() time.Duration { return ttl }
		setupProcessStart = started
		// The deadline is an orchestrated-mode control; these cases are about it.
		bootstrapControlPlaneEnabled = func() bool { return true }
	}

	t.Run("open inside the window", func(t *testing.T) {
		withWindow(t, time.Hour, time.Now().Add(-time.Minute))
		require.NoError(t, (&setupService{}).ensureSetupWindowOpen())
	})

	// An orchestrator that dies mid-provision must not leave tenant, client and
	// policy creation reachable indefinitely to whoever holds the credential.
	t.Run("an abandoned provision fails closed once the window expires", func(t *testing.T) {
		withWindow(t, time.Minute, time.Now().Add(-time.Hour))
		err := (&setupService{}).ensureSetupWindowOpen()
		require.Error(t, err)
		var forbidden *apperror.ForbiddenError
		assert.ErrorAs(t, err, &forbidden, "an expired window is a refusal, not a server fault")
	})

	// A standalone install bootstraps through the REST wizard at human pace. If
	// the deadline applied there, an operator interrupted mid-wizard would return
	// to an instance that had locked itself, fixable only by a container restart.
	t.Run("standalone is never bounded by the window", func(t *testing.T) {
		withWindow(t, time.Minute, time.Now().Add(-time.Hour))
		bootstrapControlPlaneEnabled = func() bool { return false }
		require.NoError(t, (&setupService{}).ensureSetupWindowOpen(),
			"the REST setup wizard must not expire under an operator")
	})

	// The clock starts at process start, not first contact. Anchoring it to the
	// first request would let whoever reaches the instance first restart it.
	t.Run("the window is anchored to process start", func(t *testing.T) {
		withWindow(t, time.Minute, time.Now().Add(-time.Hour))
		require.Error(t, (&setupService{}).ensureSetupWindowOpen())
		require.Error(t, (&setupService{}).ensureSetupWindowOpen(),
			"a second call must not reopen an expired window")
	})
}

// A redirect URI is where an authorization code is delivered, so a value that is
// not an exact https origin is an open redirect carrying a credential.
func TestValidateHTTPSURIs(t *testing.T) {
	t.Run("accepts https", func(t *testing.T) {
		got, err := validateHTTPSURIs([]string{"https://console.example.com/cb"}, "redirect_uris")
		require.NoError(t, err)
		assert.Equal(t, []string{"https://console.example.com/cb"}, got)
	})

	// A local dev client has no https option and the loopback interface is not on
	// the network.
	t.Run("accepts loopback over http", func(t *testing.T) {
		got, err := validateHTTPSURIs([]string{"http://localhost:3000/cb", "http://127.0.0.1:3000/cb"}, "redirect_uris")
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("rejects plain http on a real host", func(t *testing.T) {
		_, err := validateHTTPSURIs([]string{"http://console.example.com/cb"}, "redirect_uris")
		assert.Error(t, err)
	})

	// A wildcard matches origins the operator never enumerated — the classic way
	// authorization codes get delivered to an attacker.
	t.Run("rejects wildcards", func(t *testing.T) {
		_, err := validateHTTPSURIs([]string{"https://*.example.com/cb"}, "redirect_uris")
		assert.Error(t, err)
	})

	t.Run("rejects a relative path", func(t *testing.T) {
		_, err := validateHTTPSURIs([]string{"/callback"}, "redirect_uris")
		assert.Error(t, err)
	})

	t.Run("skips blanks rather than registering them", func(t *testing.T) {
		got, err := validateHTTPSURIs([]string{"", "   "}, "redirect_uris")
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

// The chain that decides whether the orchestrator can do anything at all:
//
//	client bound to service → token carries `svc` → interceptor resolves the
//	control policy attached to that service.
//
// Break any link and the orchestrator authenticates fine and is then denied
// every permission-gated method — a failure that looks like a policy problem and
// is actually a missing foreign key. These pin the first link.
func TestEnsureControlClientRequiresAServiceBinding(t *testing.T) {
	// A service with setup open and every provisioning dependency wired, so these
	// assertions land on request validation rather than on missing plumbing.
	newProvisioningService := func(t *testing.T) SetupService {
		t.Helper()
		db, _ := newMockGormDB(t)
		return NewSetupService(db, &mockUserRepo{},
			&mockTenantRepo{findSystemFn: func() (*Tenant, error) { return nil, nil }},
			&mockTenantMemberRepo{}, &mockClientRepo{}, &mockRoleRepo{}, &mockUserRoleRepo{},
			&mockUserIdentityRepo{}, &mockProfileRepo{},
			ControlRegistrationDeps{
				ServiceRepo:        &mockServiceRepo{},
				PolicyRepo:         &mockPolicyRepo{},
				ServicePolicyRepo:  &mockServicePolicyRepo{},
				APIRepo:            &mockAPIRepo{},
				PermissionRepo:     &mockPermissionRepo{},
				RolePermissionRepo: &mockRolePermissionRepo{},
				ClientURIRepo:      &mockClientURIRepo{},
			},
		)
	}
	t.Run("an unbound client is refused", func(t *testing.T) {
		_, err := newProvisioningService(t).EnsureControlClient(context.Background(), EnsureControlClientRequestDTO{
			Name: "core",
			JWKS: `{"keys":[]}`,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service_name is required")
	})

	t.Run("a credential is still required", func(t *testing.T) {
		_, err := newProvisioningService(t).EnsureControlClient(context.Background(), EnsureControlClientRequestDTO{
			Name:        "core",
			ServiceName: "core",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "jwks")
	})

	// Two sources of truth for the verification key can disagree, and which one
	// wins decides who may act as the orchestrator.
	t.Run("jwks and jwks_uri are mutually exclusive", func(t *testing.T) {
		_, err := newProvisioningService(t).EnsureControlClient(context.Background(), EnsureControlClientRequestDTO{
			Name:        "core",
			ServiceName: "core",
			JWKS:        `{"keys":[]}`,
			JWKSUri:     "https://core.example.com/jwks.json",
		})
		require.Error(t, err)
	})

	// The key set decides who may act as the orchestrator; over plain HTTP it is
	// whatever the network says it is.
	t.Run("a plaintext jwks_uri is refused", func(t *testing.T) {
		_, err := newProvisioningService(t).EnsureControlClient(context.Background(), EnsureControlClientRequestDTO{
			Name:        "core",
			ServiceName: "core",
			JWKSUri:     "http://core.example.com/jwks.json",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "https")
	})
}

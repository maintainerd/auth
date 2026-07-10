package authn

import (
	"context"
	"errors"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubTenantResolver is a function-field test double for TenantResolver.
type stubTenantResolver struct {
	fn func(ctx context.Context, name string) (int64, bool, error)
}

func (s stubTenantResolver) ResolveTenantIDByName(ctx context.Context, name string) (int64, bool, error) {
	if s.fn != nil {
		return s.fn(ctx, name)
	}
	return 0, false, nil
}

// withTenantResolver installs r as the package-level public tenant resolver for
// the duration of the test and restores the previous value afterwards. authn
// tests do not run in parallel, so the shared global is safe here.
func withTenantResolver(t *testing.T, r TenantResolver) {
	t.Helper()
	prev := publicTenantResolver
	publicTenantResolver = r
	t.Cleanup(func() { publicTenantResolver = prev })
}

func TestResolveClient_InternalTenantUsesAuthConsole(t *testing.T) {
	tenantIdentifier := "acme"
	repo := &mockClientRepo{
		findSystemByTenantIdentifierNameFn: func(gotTenant, gotName string) (*Client, error) {
			assert.Equal(t, tenantIdentifier, gotTenant)
			assert.Equal(t, shared.SystemClientNameAuthConsole, gotName)
			return &Client{Name: gotName}, nil
		},
	}

	client, err := resolveClient(repo, nil, &tenantIdentifier)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, shared.SystemClientNameAuthConsole, client.Name)
}

// When no subdomain tenant resolver is wired, a tenant_id on the public surface
// is ignored (legacy client-scoped behavior). This is the byte-for-byte
// backward-compatible path that keeps the subdomain change additive.
func TestResolvePublicClient_NoResolver_IgnoresTenantID(t *testing.T) {
	tenantIdentifier := "acme"
	repo := &mockClientRepo{}

	client, err := resolvePublicClient(context.Background(), repo, nil, &tenantIdentifier)

	require.NoError(t, err)
	assert.Nil(t, client)
}

func TestResolvePublicClient_ClientIDRejectsNonFirstPartySystemClient(t *testing.T) {
	clientID := "seeded-system-client"
	repo := &mockClientRepo{
		findByIdentifierFn: func(got string) (*Client, error) {
			assert.Equal(t, clientID, got)
			return &Client{Name: "auth-some-internal-system", IsSystem: true}, nil
		},
	}

	client, err := resolvePublicClient(context.Background(), repo, &clientID, nil)

	require.NoError(t, err)
	assert.Nil(t, client)
}

// The hosted identity UI runs on first-party SPA system clients (auth-console,
// auth-identity). Public auth (/login, /register, …) must accept them even
// though they are system clients, mirroring the OAuth authorize endpoint —
// otherwise the console's hosted-login flow can pass /oauth/authorize but always
// fails at /login.
func TestResolvePublicClient_ClientIDAllowsFirstPartySystemClients(t *testing.T) {
	for _, name := range []string{shared.SystemClientNameAuthConsole, shared.SystemClientNameAuthIdentity} {
		t.Run(name, func(t *testing.T) {
			clientID := "seeded-" + name
			repo := &mockClientRepo{
				findByIdentifierFn: func(got string) (*Client, error) {
					assert.Equal(t, clientID, got)
					return &Client{Name: name, IsSystem: true}, nil
				},
			}

			client, err := resolvePublicClient(context.Background(), repo, &clientID, nil)

			require.NoError(t, err)
			require.NotNil(t, client)
			assert.Equal(t, name, client.Name)
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Subdomain-authoritative public auth (tenant_id = slug + wired resolver)
// ─────────────────────────────────────────────────────────────────────────────

// tenant_id (slug) + matching client_id → accepted; the acting client belongs to
// the authoritative subdomain tenant.
func TestResolvePublicClient_Subdomain_MatchingClientAccepted(t *testing.T) {
	const subTenantID int64 = 42
	slug := "acme"
	clientID := "acme-web"

	withTenantResolver(t, stubTenantResolver{fn: func(_ context.Context, name string) (int64, bool, error) {
		assert.Equal(t, slug, name)
		return subTenantID, false, nil
	}})

	repo := &mockClientRepo{
		findByIdentifierFn: func(got string) (*Client, error) {
			assert.Equal(t, clientID, got)
			return &Client{Name: "acme-web", TenantID: subTenantID}, nil
		},
	}

	client, err := resolvePublicClient(context.Background(), repo, &clientID, &slug)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, subTenantID, client.TenantID)
}

// tenant_id (slug) + mismatched client_id → hard reject (ErrClientTenantMismatch),
// NO fallback to the system tenant.
func TestResolvePublicClient_Subdomain_MismatchedClientHardRejects(t *testing.T) {
	const subTenantID int64 = 42
	slug := "acme"
	clientID := "evil-web"

	withTenantResolver(t, stubTenantResolver{fn: func(_ context.Context, _ string) (int64, bool, error) {
		return subTenantID, false, nil
	}})

	repo := &mockClientRepo{
		findByIdentifierFn: func(string) (*Client, error) {
			// Client belongs to a DIFFERENT tenant.
			return &Client{Name: "evil-web", TenantID: 999}, nil
		},
	}

	client, err := resolvePublicClient(context.Background(), repo, &clientID, &slug)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrClientTenantMismatch))
	assert.Nil(t, client)
}

// tenant_id (slug) + no client_id (direct nav) → acting tenant is the subdomain
// tenant, driven by its seeded identity client.
func TestResolvePublicClient_Subdomain_DirectNavUsesIdentityClient(t *testing.T) {
	const subTenantID int64 = 42
	slug := "acme"

	withTenantResolver(t, stubTenantResolver{fn: func(_ context.Context, _ string) (int64, bool, error) {
		return subTenantID, false, nil
	}})

	repo := &mockClientRepo{
		findSystemByTenantIdentifierNameFn: func(gotSlug, gotName string) (*Client, error) {
			assert.Equal(t, slug, gotSlug)
			assert.Equal(t, shared.SystemClientNameAuthIdentity, gotName)
			return &Client{Name: shared.SystemClientNameAuthIdentity, TenantID: subTenantID, IsSystem: true}, nil
		},
	}

	client, err := resolvePublicClient(context.Background(), repo, nil, &slug)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, subTenantID, client.TenantID)
	assert.Equal(t, shared.SystemClientNameAuthIdentity, client.Name)
}

// A seeded (non hosted-login) system client is still rejected as a public
// client_id even under an authoritative subdomain tenant context. The
// system-client invariant is checked before the tenant binding.
func TestResolvePublicClient_Subdomain_SystemClientStillRejected(t *testing.T) {
	const subTenantID int64 = 42
	slug := "acme"
	clientID := "seeded-system"

	withTenantResolver(t, stubTenantResolver{fn: func(_ context.Context, _ string) (int64, bool, error) {
		return subTenantID, false, nil
	}})

	repo := &mockClientRepo{
		findByIdentifierFn: func(string) (*Client, error) {
			return &Client{Name: "auth-internal-system", IsSystem: true, TenantID: subTenantID}, nil
		},
	}

	client, err := resolvePublicClient(context.Background(), repo, &clientID, &slug)

	require.NoError(t, err)
	assert.Nil(t, client)
}

// An unknown subdomain slug is rejected: no client is resolved.
func TestResolvePublicClient_Subdomain_UnknownSlugRejected(t *testing.T) {
	slug := "does-not-exist"

	withTenantResolver(t, stubTenantResolver{fn: func(_ context.Context, _ string) (int64, bool, error) {
		return 0, false, errors.New("tenant not found")
	}})

	repo := &mockClientRepo{}

	client, err := resolvePublicClient(context.Background(), repo, nil, &slug)

	require.Error(t, err)
	assert.Nil(t, client)
}

// Regression guard: with NO tenant_id present, the client_id alone determines the
// tenant — existing behavior is unchanged even when a resolver is wired.
func TestResolvePublicClient_NoTenantID_ClientScopedUnchanged(t *testing.T) {
	clientID := "acme-web"

	// Resolver wired but must NOT be consulted when tenant_id is absent.
	resolverCalled := false
	withTenantResolver(t, stubTenantResolver{fn: func(_ context.Context, _ string) (int64, bool, error) {
		resolverCalled = true
		return 7, false, nil
	}})

	repo := &mockClientRepo{
		findByIdentifierFn: func(got string) (*Client, error) {
			assert.Equal(t, clientID, got)
			return &Client{Name: "acme-web", TenantID: 7}, nil
		},
	}

	client, err := resolvePublicClient(context.Background(), repo, &clientID, nil)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, int64(7), client.TenantID)
	assert.False(t, resolverCalled, "resolver must not be consulted without a tenant_id")
}

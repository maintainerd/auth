package authn

import (
	"testing"

	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestResolvePublicClient_RejectsTenantID(t *testing.T) {
	tenantIdentifier := "acme"
	repo := &mockClientRepo{}

	client, err := resolvePublicClient(repo, nil, &tenantIdentifier)

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

	client, err := resolvePublicClient(repo, &clientID, nil)

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

			client, err := resolvePublicClient(repo, &clientID, nil)

			require.NoError(t, err)
			require.NotNil(t, client)
			assert.Equal(t, name, client.Name)
		})
	}
}

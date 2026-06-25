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

func TestResolvePublicClient_ClientIDRejectsSystemClient(t *testing.T) {
	clientID := "seeded-system-client"
	repo := &mockClientRepo{
		findByIdentifierFn: func(got string) (*Client, error) {
			assert.Equal(t, clientID, got)
			return &Client{Name: shared.SystemClientNameAuthIdentity, IsSystem: true}, nil
		},
	}

	client, err := resolvePublicClient(repo, &clientID, nil)

	require.NoError(t, err)
	assert.Nil(t, client)
}

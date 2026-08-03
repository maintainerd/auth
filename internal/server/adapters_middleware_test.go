package server

import (
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strptr(s string) *string { return &s }

func TestToUserContext(t *testing.T) {
	tenantUUID := uuid.New()
	clientUUID := uuid.New()
	idpUUID := uuid.New()
	userUUID := uuid.New()

	// A user with two identities for two different clients; only "client-b"
	// matches the clientID the subject authenticated with.
	u := &user.User{
		UserID:   7,
		UserUUID: userUUID,
		UserIdentities: []user.UserIdentity{
			{
				TenantID: 1,
				Client:   &user.Client{ClientID: 100, Identifier: strptr("client-a")},
			},
			{
				TenantID: 42,
				Tenant:   &user.Tenant{TenantID: 42, TenantUUID: tenantUUID},
				Client: &user.Client{
					ClientID:   200,
					ClientUUID: clientUUID,
					Identifier: strptr("client-b"),
					IdentityProvider: &user.IdentityProvider{
						IdentityProviderID:   9,
						IdentityProviderUUID: idpUUID,
					},
				},
			},
		},
	}

	t.Run("extracts tenant/client/provider from matching identity", func(t *testing.T) {
		uc := toUserContext(u, "client-b")

		require.NotNil(t, uc.User)
		assert.Equal(t, userUUID, uc.User.UserUUID)

		require.NotNil(t, uc.Tenant)
		assert.Equal(t, int64(42), uc.Tenant.TenantID)
		assert.Equal(t, tenantUUID, uc.Tenant.TenantUUID)

		require.NotNil(t, uc.Client)
		assert.Equal(t, int64(200), uc.Client.ClientID)
		assert.Equal(t, clientUUID, uc.Client.ClientUUID)

		require.NotNil(t, uc.Provider)
		assert.Equal(t, int64(9), uc.Provider.IdentityProviderID)
		assert.Equal(t, idpUUID, uc.Provider.IdentityProviderUUID)
	})

	t.Run("falls back to identity TenantID when Tenant relation is nil", func(t *testing.T) {
		uc := toUserContext(u, "client-a")
		require.NotNil(t, uc.Tenant)
		assert.Equal(t, int64(1), uc.Tenant.TenantID)
		assert.Equal(t, uuid.Nil, uc.Tenant.TenantUUID)
	})

	t.Run("no matching identity leaves tenant nil", func(t *testing.T) {
		uc := toUserContext(u, "client-unknown")
		require.NotNil(t, uc.User)
		assert.Nil(t, uc.Tenant)
		assert.Nil(t, uc.Client)
		assert.Nil(t, uc.Provider)
	})
}

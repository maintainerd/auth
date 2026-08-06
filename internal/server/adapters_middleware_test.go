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

	// A user with two identities, one per identity provider. Identities carry no
	// client_id (migration 030) — the token's `sub` selects the identity, and the
	// client comes from the authenticated request, never from the identity row.
	u := &user.User{
		UserID:   7,
		UserUUID: userUUID,
		UserIdentities: []user.UserIdentity{
			{
				TenantID:           1,
				Sub:                "sub-from-provider-a",
				IdentityProviderID: 8,
			},
			{
				TenantID:           42,
				Sub:                "sub-from-provider-b",
				IdentityProviderID: 9,
				Tenant:             &user.Tenant{TenantID: 42, TenantUUID: tenantUUID},
				IdentityProvider: &user.IdentityProvider{
					IdentityProviderID:   9,
					IdentityProviderUUID: idpUUID,
				},
			},
		},
	}

	requestClient := &user.Client{
		ClientID:   200,
		ClientUUID: clientUUID,
		Identifier: strptr("client-b"),
	}

	t.Run("extracts tenant and provider from the identity matching sub", func(t *testing.T) {
		uc := toUserContext(u, "sub-from-provider-b", requestClient)

		require.NotNil(t, uc.User)
		assert.Equal(t, userUUID, uc.User.UserUUID)

		require.NotNil(t, uc.Tenant)
		assert.Equal(t, int64(42), uc.Tenant.TenantID)
		assert.Equal(t, tenantUUID, uc.Tenant.TenantUUID)

		require.NotNil(t, uc.Provider)
		assert.Equal(t, int64(9), uc.Provider.IdentityProviderID)
		assert.Equal(t, idpUUID, uc.Provider.IdentityProviderUUID)
	})

	t.Run("client comes from the request, not the identity", func(t *testing.T) {
		uc := toUserContext(u, "sub-from-provider-b", requestClient)
		require.NotNil(t, uc.Client)
		assert.Equal(t, int64(200), uc.Client.ClientID)
		assert.Equal(t, clientUUID, uc.Client.ClientUUID)

		// Same identity, a different caller — the context follows the caller.
		other := &user.Client{ClientID: 300, Identifier: strptr("client-c")}
		uc = toUserContext(u, "sub-from-provider-b", other)
		require.NotNil(t, uc.Client)
		assert.Equal(t, int64(300), uc.Client.ClientID)
	})

	t.Run("falls back to identity TenantID when Tenant relation is nil", func(t *testing.T) {
		uc := toUserContext(u, "sub-from-provider-a", requestClient)
		require.NotNil(t, uc.Tenant)
		assert.Equal(t, int64(1), uc.Tenant.TenantID)
		assert.Equal(t, uuid.Nil, uc.Tenant.TenantUUID)
		// That identity had no IdentityProvider relation preloaded.
		assert.Nil(t, uc.Provider)
	})

	t.Run("an unknown sub leaves tenant and provider nil", func(t *testing.T) {
		uc := toUserContext(u, "sub-that-does-not-exist", requestClient)
		require.NotNil(t, uc.User)
		assert.Nil(t, uc.Tenant)
		assert.Nil(t, uc.Provider)
	})

	t.Run("no request client leaves client nil", func(t *testing.T) {
		uc := toUserContext(u, "sub-from-provider-b", nil)
		require.NotNil(t, uc.Tenant)
		assert.Nil(t, uc.Client)
	})
}

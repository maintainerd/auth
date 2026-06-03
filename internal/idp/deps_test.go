package idp

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDepsTableNames(t *testing.T) {
	assert.Equal(t, "tenants", Tenant{}.TableName())
	assert.Equal(t, "user_identities", UserIdentity{}.TableName())
	assert.Equal(t, "users", User{}.TableName())
	assert.Equal(t, "clients", Client{}.TableName())
	assert.Equal(t, "roles", Role{}.TableName())
	assert.Equal(t, "user_roles", UserRole{}.TableName())
}

func TestValidateTenantAccess(t *testing.T) {
	t.Run("nil actor returns unauthorized", func(t *testing.T) {
		err := ValidateTenantAccess(nil, &Tenant{TenantID: tenantID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
	})

	t.Run("nil target returns not found", func(t *testing.T) {
		err := ValidateTenantAccess(&User{}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant not found")
	})

	t.Run("actor with no identities returns forbidden", func(t *testing.T) {
		err := ValidateTenantAccess(&User{}, &Tenant{TenantID: tenantID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no identities")
	})

	t.Run("matching tenant grants access", func(t *testing.T) {
		err := ValidateTenantAccess(&User{
			UserIdentities: []UserIdentity{{TenantID: tenantID}},
		}, &Tenant{TenantID: tenantID})
		require.NoError(t, err)
	})

	t.Run("system tenant identity grants access to any tenant", func(t *testing.T) {
		err := ValidateTenantAccess(&User{
			UserIdentities: []UserIdentity{{TenantID: 99, Tenant: &Tenant{IsSystem: true}}},
		}, &Tenant{TenantID: tenantID})
		require.NoError(t, err)
	})

	t.Run("non-matching tenant without system access returns forbidden", func(t *testing.T) {
		err := ValidateTenantAccess(&User{
			UserIdentities: []UserIdentity{{TenantID: 99}},
		}, &Tenant{TenantID: tenantID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant access denied")
	})
}

func TestToTenantServiceDataResult(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		assert.Nil(t, toTenantServiceDataResult(nil))
	})

	t.Run("full tenant mapping", func(t *testing.T) {
		now := time.Now()
		tnt := &Tenant{
			TenantUUID:  uuid.New(),
			Name:        "tenant",
			DisplayName: "Tenant",
			Description: "desc",
			Identifier:  "tenant",
			Status:      shared.StatusActive,
			IsPublic:    true,
			IsSystem:    true,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		dto := toTenantServiceDataResult(tnt)
		require.NotNil(t, dto)
		assert.Equal(t, tnt.TenantUUID, dto.TenantUUID)
		assert.Equal(t, tnt.Name, dto.Name)
		assert.Equal(t, tnt.DisplayName, dto.DisplayName)
		assert.Equal(t, tnt.Description, dto.Description)
		assert.Equal(t, tnt.Identifier, dto.Identifier)
		assert.Equal(t, tnt.Status, dto.Status)
		assert.True(t, dto.IsPublic)
		assert.True(t, dto.IsSystem)
		assert.Equal(t, now, dto.CreatedAt)
		assert.Equal(t, now, dto.UpdatedAt)
	})
}

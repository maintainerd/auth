package idp

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func int64Ptr(v int64) *int64 { return &v }

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

	t.Run("system tenant identity denied cross-tenant", func(t *testing.T) {
		// Lockdown: a system-tenant identity no longer grants cross-tenant
		// access here. The system override is confined to the tenant package.
		err := ValidateTenantAccess(&User{
			UserIdentities: []UserIdentity{{TenantID: 99, Tenant: &Tenant{IsSystem: true}}},
		}, &Tenant{TenantID: tenantID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant access denied")
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
			Status:      shared.StatusActive,
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
		// Tenant identifier was dropped; the mapper now surfaces the name as the slug.
		assert.Equal(t, tnt.Name, dto.Identifier)
		assert.Equal(t, tnt.Status, dto.Status)
		assert.True(t, dto.IsSystem)
		assert.Equal(t, now, dto.CreatedAt)
		assert.Equal(t, now, dto.UpdatedAt)
	})
}

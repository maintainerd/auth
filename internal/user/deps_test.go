package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTenantTableName(t *testing.T) {
	assert.Equal(t, "tenants", Tenant{}.TableName())
}

func TestRoleTableName(t *testing.T) {
	assert.Equal(t, "roles", Role{}.TableName())
}

func TestRolePermissionTableName(t *testing.T) {
	assert.Equal(t, "role_permissions", RolePermission{}.TableName())
}

func TestClientTableName(t *testing.T) {
	assert.Equal(t, "clients", Client{}.TableName())
}

func TestIdentityProviderTableName(t *testing.T) {
	assert.Equal(t, "identity_providers", IdentityProvider{}.TableName())
}

func TestUserBackupCodeTableName(t *testing.T) {
	assert.Equal(t, "user_mfa_backup_codes", UserBackupCode{}.TableName())
}

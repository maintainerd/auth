package invite

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTenantRecord_TableName(t *testing.T) {
	assert.Equal(t, "tenants", TenantRecord{}.TableName())
}

func TestIdentityProvider_TableName(t *testing.T) {
	assert.Equal(t, "identity_providers", IdentityProvider{}.TableName())
}

func TestClient_TableName(t *testing.T) {
	assert.Equal(t, "clients", Client{}.TableName())
}

func TestRole_TableName(t *testing.T) {
	assert.Equal(t, "roles", Role{}.TableName())
}

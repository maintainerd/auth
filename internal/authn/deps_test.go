package authn

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthnProjectionTableNames(t *testing.T) {
	assert.Equal(t, "tenants", Tenant{}.TableName())
	assert.Equal(t, "identity_providers", IdentityProvider{}.TableName())
	assert.Equal(t, "clients", Client{}.TableName())
	assert.Equal(t, "users", User{}.TableName())
	assert.Equal(t, "user_identities", UserIdentity{}.TableName())
	assert.Equal(t, "user_tokens", UserToken{}.TableName())
	assert.Equal(t, "user_roles", UserRole{}.TableName())
	assert.Equal(t, "roles", Role{}.TableName())
	assert.Equal(t, "invites", Invite{}.TableName())
}

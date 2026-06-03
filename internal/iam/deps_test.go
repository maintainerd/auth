package iam

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDepsTableNames(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "tenant", got: Tenant{}.TableName(), want: "tenants"},
		{name: "user identity", got: UserIdentity{}.TableName(), want: "user_identities"},
		{name: "user", got: User{}.TableName(), want: "users"},
		{name: "client", got: Client{}.TableName(), want: "clients"},
		{name: "tenant service", got: TenantService{}.TableName(), want: "tenant_services"},
		{name: "user role", got: UserRole{}.TableName(), want: "user_roles"},
		{name: "user token", got: UserToken{}.TableName(), want: "user_tokens"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.got)
		})
	}
}

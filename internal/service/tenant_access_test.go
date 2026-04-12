package service

import (
	"testing"

	"github.com/maintainerd/auth/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildUserWithIdentities creates a user with the provided identities.
func buildUserWithIdentities(identities []model.UserIdentity) *model.User {
	return &model.User{
		UserID:         1,
		UserIdentities: identities,
	}
}

// buildTenant creates a minimal tenant for tests.
func buildTenant(id int64, isSystem bool) *model.Tenant {
	return &model.Tenant{
		TenantID: id,
		IsSystem: isSystem,
	}
}

// buildIdentity creates a UserIdentity linked to the given tenant.
func buildIdentity(tenantID int64, isSystem bool) model.UserIdentity {
	return model.UserIdentity{
		TenantID: tenantID,
		Tenant:   &model.Tenant{TenantID: tenantID, IsSystem: isSystem},
	}
}

func TestValidateTenantAccess(t *testing.T) {
	cases := []struct {
		name        string
		user        *model.User
		target      *model.Tenant
		expectError bool
		errContains string
	}{
		{
			name:        "no identities → error",
			user:        buildUserWithIdentities(nil),
			target:      buildTenant(10, false),
			expectError: true,
			errContains: "no identities",
		},
		{
			name: "user from default tenant → allowed on any tenant",
			user: buildUserWithIdentities([]model.UserIdentity{
				buildIdentity(1, true),
			}),
			target:      buildTenant(99, false),
			expectError: false,
		},
		{
			name: "user from same tenant → allowed",
			user: buildUserWithIdentities([]model.UserIdentity{
				buildIdentity(10, false),
			}),
			target:      buildTenant(10, false),
			expectError: false,
		},
		{
			name: "user from different non-default tenant → denied",
			user: buildUserWithIdentities([]model.UserIdentity{
				buildIdentity(20, false),
			}),
			target:      buildTenant(10, false),
			expectError: true,
			errContains: "access denied",
		},
		{
			name: "multiple identities; one matches target → allowed",
			user: buildUserWithIdentities([]model.UserIdentity{
				buildIdentity(20, false),
				buildIdentity(10, false),
			}),
			target:      buildTenant(10, false),
			expectError: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTenantAccess(tc.user, tc.target)
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTenantAccessByID(t *testing.T) {
	cases := []struct {
		name           string
		user           *model.User
		targetTenantID int64
		expectError    bool
		errContains    string
	}{
		{
			name:           "no identities → error",
			user:           buildUserWithIdentities(nil),
			targetTenantID: 10,
			expectError:    true,
			errContains:    "no identities",
		},
		{
			name: "default tenant user → allowed on any tenant",
			user: buildUserWithIdentities([]model.UserIdentity{
				buildIdentity(1, true),
			}),
			targetTenantID: 99,
			expectError:    false,
		},
		{
			name: "matching tenant ID → allowed",
			user: buildUserWithIdentities([]model.UserIdentity{
				buildIdentity(10, false),
			}),
			targetTenantID: 10,
			expectError:    false,
		},
		{
			name: "non-matching non-default → denied",
			user: buildUserWithIdentities([]model.UserIdentity{
				buildIdentity(20, false),
			}),
			targetTenantID: 10,
			expectError:    true,
			errContains:    "access denied",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTenantAccessByID(tc.user, tc.targetTenantID)
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

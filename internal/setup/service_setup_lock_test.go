package setup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// First-run is complete once the system tenant exists and has an owner.
//
// The admin's PROFILE is deliberately excluded. It is collected on first
// sign-in through the identity app — first name, last name, gender, from which
// the display name is derived. Gating the lock on it was a deadlock: setup could
// not complete, so the tenant stayed `pending`, and
// AuthEndpointTenantStatusMiddleware then refused the very login that would
// have created the profile.
func TestCompleteSetup_LocksOnTenantAndAdminOnly(t *testing.T) {
	newSvc := func(hasAdmin bool, tenants []Tenant) SetupService {
		userRepo := &mockUserRepo{}
		if hasAdmin {
			userRepo.findSuperAdminFn = func() (*User, error) { return &User{UserID: 1}, nil }
		}
		return NewSetupService(nil,
			userRepo,
			&mockTenantRepo{
				findAllFn:        func(...string) ([]Tenant, error) { return tenants, nil },
				findSystemFn:     func() (*Tenant, error) { return &Tenant{TenantID: 1}, nil },
				createOrUpdateFn: func(tn *Tenant) (*Tenant, error) { return tn, nil },
			},
			&mockTenantMemberRepo{}, &mockClientRepo{},
			&mockRoleRepo{}, &mockUserRoleRepo{}, &mockUserIdentityRepo{},
			// No profile exists — the normal state right after admin creation.
			&mockProfileRepo{},
		)
	}

	t.Run("completes with tenant and admin, without any profile", func(t *testing.T) {
		res, err := newSvc(true, []Tenant{{TenantID: 1}}).CompleteSetup(context.Background())
		require.NoError(t, err, "a missing profile must not block the lock")
		assert.True(t, res.IsSetupComplete)
	})

	t.Run("refuses without an admin", func(t *testing.T) {
		_, err := newSvc(false, []Tenant{{TenantID: 1}}).CompleteSetup(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant and admin setup")
	})

	t.Run("refuses without a tenant", func(t *testing.T) {
		_, err := newSvc(true, nil).CompleteSetup(context.Background())
		require.Error(t, err)
	})
}

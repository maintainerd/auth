package tenant

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTenantIsolationInvariant verifies the core access control rule:
// a user whose identities are all in non-system tenants can never access
// a different tenant, regardless of how many identities they have.
func TestTenantIsolationInvariant(t *testing.T) {
	// Enumerate a matrix of (user tenants) × (target tenant) combinations
	// and assert the invariant holds in every case.
	type row struct {
		userTenantIDs  []int64
		targetTenantID int64
		systemTenantID int64 // which tenant ID is the system tenant (0 = none)
		wantAllowed    bool
		label          string
	}

	rows := []row{
		// ---- single identity, matching ----
		{userTenantIDs: []int64{10}, targetTenantID: 10, wantAllowed: true, label: "same tenant"},
		// ---- single identity, non-matching ----
		{userTenantIDs: []int64{10}, targetTenantID: 20, wantAllowed: false, label: "cross-tenant blocked"},
		// ---- zero identities ----
		{userTenantIDs: []int64{}, targetTenantID: 10, wantAllowed: false, label: "no identities blocked"},
		// ---- multiple identities, one matches ----
		{userTenantIDs: []int64{10, 20, 30}, targetTenantID: 20, wantAllowed: true, label: "multi-identity one match"},
		// ---- multiple identities, none match ----
		{userTenantIDs: []int64{10, 20, 30}, targetTenantID: 40, wantAllowed: false, label: "multi-identity no match"},
		// ---- system tenant: NO resource-access bypass (strict isolation) ----
		{userTenantIDs: []int64{1}, targetTenantID: 99, systemTenantID: 1, wantAllowed: false, label: "system identity does not bypass resource access"},
		{userTenantIDs: []int64{1}, targetTenantID: 1, systemTenantID: 1, wantAllowed: true, label: "system user own system tenant"},
		// ---- system tenant user alongside non-system identity ----
		{userTenantIDs: []int64{10, 1}, targetTenantID: 99, systemTenantID: 1, wantAllowed: false, label: "mixed identities: system does not grant cross-tenant access"},
		// ---- large tenant ID gap ----
		{userTenantIDs: []int64{1000}, targetTenantID: 9999, wantAllowed: false, label: "large id gap blocked"},
		// ---- target == 0 (invalid) ----
		{userTenantIDs: []int64{10}, targetTenantID: 0, wantAllowed: false, label: "target id 0 blocked"},
	}

	for _, r := range rows {
		r := r
		t.Run(r.label, func(t *testing.T) {
			identities := make([]AccessIdentity, len(r.userTenantIDs))
			for i, tid := range r.userTenantIDs {
				isSystem := r.systemTenantID != 0 && tid == r.systemTenantID
				identities[i] = buildIdentity(tid, isSystem)
			}
			user := buildUserWithIdentities(identities)
			err := ValidateTenantAccessByID(user, r.targetTenantID)
			if r.wantAllowed {
				assert.NoError(t, err, "expected access to be allowed")
			} else {
				assert.Error(t, err, "expected access to be denied")
			}
		})
	}
}

// TestTenantIsolation_CrossTenantNeverLeaks verifies the property exhaustively:
// for any pair of distinct non-system tenant IDs, a user from one cannot
// access the other. Tests a 5×5 matrix (skipping same-tenant diagonal).
func TestTenantIsolation_CrossTenantNeverLeaks(t *testing.T) {
	tenantIDs := []int64{2, 3, 4, 5, 6}

	for _, userTenant := range tenantIDs {
		for _, targetTenant := range tenantIDs {
			if userTenant == targetTenant {
				continue
			}
			name := fmt.Sprintf("user_in_%d_cannot_access_%d", userTenant, targetTenant)
			t.Run(name, func(t *testing.T) {
				user := buildUserWithIdentities([]AccessIdentity{
					buildIdentity(userTenant, false),
				})
				err := ValidateTenantAccessByID(user, targetTenant)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "access denied")
			})
		}
	}
}

// TestTenantIsolation_SystemIdentityDoesNotBypassResourceAccess verifies that a
// system-tenant identity does NOT grant cross-tenant access to another tenant's
// records. The system tenant's only cross-tenant power (tenant-management:
// create/update/members) is authorized separately by CanManageTenant, never
// through this resource-access gate (rules #9/#10). This guards against a future
// refactor silently re-enabling system god-mode over every tenant's users.
func TestTenantIsolation_SystemIdentityDoesNotBypassResourceAccess(t *testing.T) {
	// A non-system identity cannot reach another tenant.
	user := buildUserWithIdentities([]AccessIdentity{
		buildIdentity(2, false),
	})
	require.Error(t, ValidateTenantAccessByID(user, 99), "non-system tenant must not bypass access control")

	// A system-tenant identity ALSO cannot reach another tenant's records.
	userSystem := buildUserWithIdentities([]AccessIdentity{
		buildIdentity(1, true),
	})
	require.Error(t, ValidateTenantAccessByID(userSystem, 99),
		"system identity must not bypass resource-level tenant isolation")

	// But it can still reach its OWN tenant.
	require.NoError(t, ValidateTenantAccessByID(userSystem, 1))
}

// TestTenantIsolation_NilUser verifies that a nil user is handled safely.
func TestTenantIsolation_NilUser(t *testing.T) {
	err := ValidateTenantAccess(nil, buildTenant(10, false))
	require.Error(t, err)
}

// TestTenantIsolation_ValidateTenantAccess_MirrorsByID confirms that
// ValidateTenantAccess and ValidateTenantAccessByID produce the same decision
// for the same logical inputs.
func TestTenantIsolation_ValidateTenantAccess_MirrorsByID(t *testing.T) {
	cases := []struct {
		label     string
		userTenID int64
		targetID  int64
		isSystem  bool
	}{
		{"same non-system", 10, 10, false},
		{"different non-system", 10, 20, false},
		{"system user", 1, 50, true},
	}

	for _, c := range cases {
		c := c
		t.Run(c.label, func(t *testing.T) {
			user := buildUserWithIdentities([]AccessIdentity{
				buildIdentity(c.userTenID, c.isSystem),
			})
			target := buildTenant(c.targetID, false)

			errByObj := ValidateTenantAccess(user, target)
			errByID := ValidateTenantAccessByID(user, c.targetID)

			// Both must agree on whether access is allowed.
			assert.Equal(t, errByObj == nil, errByID == nil,
				"ValidateTenantAccess and ValidateTenantAccessByID must agree")
		})
	}
}

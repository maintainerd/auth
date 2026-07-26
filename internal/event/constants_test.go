package event

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllEventTypes(t *testing.T) {
	specs := AllEventTypes()
	assert.Len(t, specs, 42, "should have 42 event types in the v1.0.0 catalog")

	seen := make(map[string]bool)
	for _, s := range specs {
		assert.NotEmpty(t, s.Key, "every event type must have a key")
		assert.NotEmpty(t, s.Category, "every event type must have a category")
		assert.False(t, seen[s.Key], "duplicate event type key: %s", s.Key)
		assert.GreaterOrEqual(t, s.Version, 1, "version must be >= 1 for: %s", s.Key)
		seen[s.Key] = true
	}
}

func TestEventTypeCategories(t *testing.T) {
	specs := AllEventTypes()
	cats := map[string]int{}
	for _, s := range specs {
		cats[s.Category]++
	}

	assert.Greater(t, cats[CategoryUser], 0, "should have USER events")
	assert.Greater(t, cats[CategoryTenant], 0, "should have TENANT events")
	assert.Greater(t, cats[CategoryIAM], 0, "should have IAM events")
	assert.Greater(t, cats[CategoryClient], 0, "should have CLIENT events")
}

func TestEventTypeConstants_NotExposed(t *testing.T) {
	// Verify authentication ceremonies are NOT in the catalog
	specs := AllEventTypes()
	for _, s := range specs {
		assert.NotContains(t, s.Key, "authn_login", "auth events must not be integration events")
		assert.NotContains(t, s.Key, "authn_oauth", "OAuth auth events must not be integration events")
		assert.NotContains(t, s.Key, "authn_mfa", "MFA events must not be integration events")
	}
}

func TestEventTypeConstants_Group1Coverage(t *testing.T) {
	specs := AllEventTypes()
	find := func(key string) bool {
		for _, s := range specs {
			if s.Key == key {
				return true
			}
		}
		return false
	}
	assert.True(t, find(EventTypeUserCreated))
	assert.True(t, find(EventTypeUserUpdated))
	assert.True(t, find(EventTypeUserStatusChanged))
	assert.True(t, find(EventTypeUserDeleted))
	assert.True(t, find(EventTypeUserRoleAssigned))
	assert.True(t, find(EventTypeUserRoleRemoved))
}

func TestEventTypeConstants_Group2Coverage(t *testing.T) {
	specs := AllEventTypes()
	find := func(key string) bool {
		for _, s := range specs {
			if s.Key == key {
				return true
			}
		}
		return false
	}
	assert.True(t, find(EventTypeRoleCreated))
	assert.True(t, find(EventTypeRolePermissionsChanged))
	assert.True(t, find(EventTypePermissionCreated))
	assert.True(t, find(EventTypeIAMPolicyUpdated))
	assert.True(t, find(EventTypePolicyCreated))
}

func TestEventTypeConstants_Group3Coverage(t *testing.T) {
	specs := AllEventTypes()
	find := func(key string) bool {
		for _, s := range specs {
			if s.Key == key {
				return true
			}
		}
		return false
	}
	assert.True(t, find(EventTypeTenantCreated))
	assert.True(t, find(EventTypeTenantMemberAdded))
	assert.True(t, find(EventTypeTenantMemberRemoved))
}

func TestEventTypeConstants_Group4Coverage(t *testing.T) {
	specs := AllEventTypes()
	find := func(key string) bool {
		for _, s := range specs {
			if s.Key == key {
				return true
			}
		}
		return false
	}
	assert.True(t, find(EventTypeClientCreated))
	assert.True(t, find(EventTypeClientSecretRotated))
}

func TestEventTypeConstants_Group5Coverage(t *testing.T) {
	specs := AllEventTypes()
	find := func(key string) bool {
		for _, s := range specs {
			if s.Key == key {
				return true
			}
		}
		return false
	}
	assert.True(t, find(EventTypeAPICreated))
	assert.True(t, find(EventTypeServiceCreated))
	assert.True(t, find(EventTypeServiceDeleted))
}

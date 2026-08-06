package setup

import (
	"regexp"
	"strings"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/setup/seeder"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
)

// The policy row is written by setup and edited afterwards through the console,
// so its name has to satisfy the same validator PolicyUpdateRequestDTO applies —
// otherwise setup produces a policy that cannot be saved back.
func TestControlPolicySatisfiesTheValidator(t *testing.T) {
	assert.Regexp(t, regexp.MustCompile(`^[a-z0-9_:/\\-]+$`), DefaultControlPolicyName)
	assert.GreaterOrEqual(t, len(DefaultControlPolicyName), 3)
	assert.LessOrEqual(t, len(DefaultControlPolicyName), 150)
}

// The control policy is the most powerful grant this system issues. These are
// the properties that keep it reviewable.
func TestNormalizeControlActions(t *testing.T) {
	t.Run("empty falls back to the documented default set", func(t *testing.T) {
		assert.Equal(t, seeder.DefaultControlActions, normalizeControlActions(nil))
		assert.Equal(t, seeder.DefaultControlActions, normalizeControlActions([]string{"", "  "}))
	})

	// A bare "*" cannot be reviewed and silently absorbs every permission family
	// added later — the exact failure the seeded policy had.
	t.Run("a bare wildcard is refused, not honoured", func(t *testing.T) {
		assert.Equal(t, seeder.DefaultControlActions, normalizeControlActions([]string{"*"}))
		assert.Equal(t, seeder.DefaultControlActions, normalizeControlActions([]string{"*:*"}))
		assert.Equal(t, []string{"tenant:*"}, normalizeControlActions([]string{"tenant:*", "*"}))
	})

	t.Run("caller-supplied actions are honoured and de-duplicated", func(t *testing.T) {
		assert.Equal(t, []string{"tenant:*", "client:*"},
			normalizeControlActions([]string{"tenant:*", " client:* ", "tenant:*"}))
	})

	// An orchestrator provisions tenants; holding these would let one compromise
	// read and mutate every end user, or lower the defences and erase the trail.
	t.Run("the default set withholds end-user and defence-control families", func(t *testing.T) {
		for _, withheld := range []string{
			"user:*", "account:*:self", "security-setting:*",
			"ip-restriction-rule:*", "audit:read", "auth_event:*",
		} {
			assert.NotContains(t, seeder.DefaultControlActions, withheld)
		}
	})

	// Every default action must be management-plane; a public: or :self entry
	// would mean the control policy was granting something it has no business in.
	t.Run("every default action is an elevated family", func(t *testing.T) {
		for _, action := range seeder.DefaultControlActions {
			name := strings.TrimSuffix(action, ":*")
			assert.True(t, shared.IsElevatedPermission(name), action)
		}
	})
}

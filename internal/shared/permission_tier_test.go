package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The classification is the security boundary for what a public, guessable
// registration link may grant, so each tier is asserted explicitly.
func TestIsElevatedPermission(t *testing.T) {
	tests := []struct {
		name       string
		permission string
		elevated   bool
	}{
		// public:* — auto-assigned to every user.
		{"public register", "public:register", false},
		{"public login", "public:login", false},
		{"public nested", "public:oauth2:callback", false},

		// account:*:self — the holder's own data only.
		{"own profile read", "account:profile:read:self", false},
		{"own user delete", "account:user:delete:self", false},
		{"own mfa enroll", "account:mfa:enroll:self", false},

		// account:* WITHOUT :self reads/writes ANY user — management plane.
		{"account without self scope", "account:user:read", true},
		{"account update without self scope", "account:user:update", true},

		// Management plane.
		{"tenant delete", "tenant:delete", true},
		{"role update", "role:update", true},
		{"registration flow update", "registration-flow:update", true},
		{"user invite", "user:invite", true},

		// Unknown namespaces default to elevated — a permission nobody has
		// classified must not become self-service grantable by omission.
		{"unrecognized namespace", "billing:approve", true},
		{"bare word", "everything", true},

		// Normalization.
		{"mixed case public", "PUBLIC:Register", false},
		{"padded self scope", "  account:profile:read:self  ", false},

		// Empty is not a permission; treated as non-elevated so a blank row
		// cannot block an otherwise valid role.
		{"empty", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.elevated, IsElevatedPermission(tc.permission))
		})
	}
}

func TestFirstElevatedPermission(t *testing.T) {
	t.Run("returns the offending permission so the operator can be told which", func(t *testing.T) {
		got := FirstElevatedPermission([]string{
			"public:login",
			"account:profile:read:self",
			"tenant:delete",
			"role:update",
		})
		assert.Equal(t, "tenant:delete", got)
	})

	t.Run("empty when every permission is public or own-account", func(t *testing.T) {
		got := FirstElevatedPermission([]string{"public:register", "account:user:read:self"})
		assert.Empty(t, got)
	})

	t.Run("empty for a role with no permissions at all", func(t *testing.T) {
		assert.Empty(t, FirstElevatedPermission(nil))
		assert.Empty(t, FirstElevatedPermission([]string{}))
	})
}

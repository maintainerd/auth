package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// This is the one comparison behind every "you cannot grant what you do not
// hold" guard. It was three copies before, so the cases that matter are the
// ones a divergent copy would get wrong.
func TestFirstUnheldElevatedPermission(t *testing.T) {
	tests := []struct {
		name     string
		granting []string
		held     []string
		unheld   string
	}{
		{"actor holds the elevated permission", []string{"user:update"}, []string{"user:update"}, ""},
		{"actor does not hold it", []string{"user:update"}, []string{"role:read"}, "user:update"},
		{"reports the first unheld, not the last", []string{"user:update", "tenant:delete"}, []string{}, "user:update"},
		{"holds some but not all", []string{"user:update", "tenant:delete"}, []string{"user:update"}, "tenant:delete"},

		// Gating these would break ordinary self-service.
		{"public permissions are never gated", []string{"public:login"}, nil, ""},
		{"own-account permissions are never gated", []string{"account:profile:read:self"}, nil, ""},
		{"non-self account permission IS gated", []string{"account:user:read"}, nil, "account:user:read"},

		// The direction of failure: no held set means grant nothing.
		{"empty held refuses every elevated grant", []string{"user:update"}, nil, "user:update"},
		{"empty held still permits public", []string{"public:login"}, nil, ""},

		{"nothing to grant is allowed", nil, nil, ""},
		{"holding extra permissions is irrelevant", []string{"user:update"}, []string{"user:update", "tenant:delete"}, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.unheld, FirstUnheldElevatedPermission(tc.granting, tc.held))
		})
	}
}

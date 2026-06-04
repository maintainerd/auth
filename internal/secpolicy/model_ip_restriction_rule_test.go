package secpolicy

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIPRestrictionRule_TableName(t *testing.T) {
	assert.Equal(t, "ip_restriction_rules", IPRestrictionRule{}.TableName())
}

func TestIPRestrictionRule_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid when missing", func(t *testing.T) {
		rule := &IPRestrictionRule{}
		err := rule.BeforeCreate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, rule.IPRestrictionRuleUUID)
	})

	t.Run("preserves existing uuid", func(t *testing.T) {
		existing := uuid.New()
		rule := &IPRestrictionRule{IPRestrictionRuleUUID: existing}
		err := rule.BeforeCreate(nil)
		require.NoError(t, err)
		assert.Equal(t, existing, rule.IPRestrictionRuleUUID)
	})
}

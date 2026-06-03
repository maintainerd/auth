package iam

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicyModel(t *testing.T) {
	t.Run("table name", func(t *testing.T) {
		assert.Equal(t, "policies", Policy{}.TableName())
	})

	t.Run("before create assigns uuid when nil", func(t *testing.T) {
		policy := &Policy{}

		require.NoError(t, policy.BeforeCreate(nil))

		assert.NotEqual(t, uuid.Nil, policy.PolicyUUID)
	})

	t.Run("before create preserves existing uuid", func(t *testing.T) {
		id := uuid.New()
		policy := &Policy{PolicyUUID: id}

		require.NoError(t, policy.BeforeCreate(nil))

		assert.Equal(t, id, policy.PolicyUUID)
	})
}

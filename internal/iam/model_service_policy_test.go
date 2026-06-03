package iam

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServicePolicyModel(t *testing.T) {
	t.Run("table name", func(t *testing.T) {
		assert.Equal(t, "service_policies", ServicePolicy{}.TableName())
	})

	t.Run("before create assigns uuid when nil", func(t *testing.T) {
		servicePolicy := &ServicePolicy{}

		require.NoError(t, servicePolicy.BeforeCreate(nil))

		assert.NotEqual(t, uuid.Nil, servicePolicy.ServicePolicyUUID)
	})

	t.Run("before create preserves existing uuid", func(t *testing.T) {
		id := uuid.New()
		servicePolicy := &ServicePolicy{ServicePolicyUUID: id}

		require.NoError(t, servicePolicy.BeforeCreate(nil))

		assert.Equal(t, id, servicePolicy.ServicePolicyUUID)
	})
}

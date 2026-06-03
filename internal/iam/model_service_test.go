package iam

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceModel(t *testing.T) {
	t.Run("table name", func(t *testing.T) {
		assert.Equal(t, "services", Service{}.TableName())
	})

	t.Run("before create assigns uuid when nil", func(t *testing.T) {
		service := &Service{}

		require.NoError(t, service.BeforeCreate(nil))

		assert.NotEqual(t, uuid.Nil, service.ServiceUUID)
	})

	t.Run("before create preserves existing uuid", func(t *testing.T) {
		id := uuid.New()
		service := &Service{ServiceUUID: id}

		require.NoError(t, service.BeforeCreate(nil))

		assert.Equal(t, id, service.ServiceUUID)
	})
}

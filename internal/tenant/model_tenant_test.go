package tenant

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenant_TableName(t *testing.T) {
	assert.Equal(t, "tenants", Tenant{}.TableName())
}

func TestTenant_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid when missing", func(t *testing.T) {
		model := &Tenant{}

		err := model.BeforeCreate(nil)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, model.TenantUUID)
	})

	t.Run("preserves existing uuid", func(t *testing.T) {
		existing := uuid.New()
		model := &Tenant{TenantUUID: existing}

		err := model.BeforeCreate(nil)

		require.NoError(t, err)
		assert.Equal(t, existing, model.TenantUUID)
	})
}

package idp

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentityProvider_TableName(t *testing.T) {
	assert.Equal(t, "identity_providers", IdentityProvider{}.TableName())
}

func TestIdentityProvider_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid when empty", func(t *testing.T) {
		idp := &IdentityProvider{}
		require.NoError(t, idp.BeforeCreate(nil))
		assert.NotEqual(t, uuid.Nil, idp.IdentityProviderUUID)
	})

	t.Run("keeps existing uuid", func(t *testing.T) {
		existing := uuid.New()
		idp := &IdentityProvider{IdentityProviderUUID: existing}
		require.NoError(t, idp.BeforeCreate(nil))
		assert.Equal(t, existing, idp.IdentityProviderUUID)
	})
}

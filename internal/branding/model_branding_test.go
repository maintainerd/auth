package branding

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranding_TableName(t *testing.T) {
	assert.Equal(t, "branding", Branding{}.TableName())
}

func TestBranding_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid when missing", func(t *testing.T) {
		b := &Branding{}
		err := b.BeforeCreate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, b.BrandingUUID)
	})

	t.Run("preserves existing uuid", func(t *testing.T) {
		existing := uuid.New()
		b := &Branding{BrandingUUID: existing}
		err := b.BeforeCreate(nil)
		require.NoError(t, err)
		assert.Equal(t, existing, b.BrandingUUID)
	})
}

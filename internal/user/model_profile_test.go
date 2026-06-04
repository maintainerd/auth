package user

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestProfileTableName(t *testing.T) {
	assert.Equal(t, "profiles", Profile{}.TableName())
}

func TestProfileBeforeCreate(t *testing.T) {
	t.Run("sets UUID when nil", func(t *testing.T) {
		p := &Profile{}
		err := p.BeforeCreate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, p.ProfileUUID)
	})

	t.Run("preserves existing UUID", func(t *testing.T) {
		existing := uuid.New()
		p := &Profile{ProfileUUID: existing}
		err := p.BeforeCreate(&gorm.DB{})
		require.NoError(t, err)
		assert.Equal(t, existing, p.ProfileUUID)
	})
}

package branding

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailTemplate_TableName(t *testing.T) {
	assert.Equal(t, "email_templates", EmailTemplate{}.TableName())
}

func TestEmailTemplate_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid when missing", func(t *testing.T) {
		e := &EmailTemplate{}
		err := e.BeforeCreate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, e.EmailTemplateUUID)
	})

	t.Run("preserves existing uuid", func(t *testing.T) {
		existing := uuid.New()
		e := &EmailTemplate{EmailTemplateUUID: existing}
		err := e.BeforeCreate(nil)
		require.NoError(t, err)
		assert.Equal(t, existing, e.EmailTemplateUUID)
	})
}

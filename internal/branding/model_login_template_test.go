package branding

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginTemplate_TableName(t *testing.T) {
	assert.Equal(t, "login_templates", LoginTemplate{}.TableName())
}

func TestLoginTemplate_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid when missing", func(t *testing.T) {
		tmpl := &LoginTemplate{}
		err := tmpl.BeforeCreate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, tmpl.LoginTemplateUUID)
	})

	t.Run("preserves existing uuid", func(t *testing.T) {
		existing := uuid.New()
		tmpl := &LoginTemplate{LoginTemplateUUID: existing}
		err := tmpl.BeforeCreate(nil)
		require.NoError(t, err)
		assert.Equal(t, existing, tmpl.LoginTemplateUUID)
	})
}

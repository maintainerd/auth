package branding

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSMSTemplate_TableName(t *testing.T) {
	assert.Equal(t, "sms_templates", SMSTemplate{}.TableName())
}

func TestSMSTemplate_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid when missing", func(t *testing.T) {
		s := &SMSTemplate{}
		err := s.BeforeCreate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, s.SMSTemplateUUID)
	})

	t.Run("preserves existing uuid", func(t *testing.T) {
		existing := uuid.New()
		s := &SMSTemplate{SMSTemplateUUID: existing}
		err := s.BeforeCreate(nil)
		require.NoError(t, err)
		assert.Equal(t, existing, s.SMSTemplateUUID)
	})
}

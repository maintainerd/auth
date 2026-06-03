package notifier

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSMSConfigModel(t *testing.T) {
	t.Run("table name", func(t *testing.T) {
		assert.Equal(t, "sms_config", SMSConfig{}.TableName())
	})

	t.Run("BeforeCreate assigns UUID", func(t *testing.T) {
		model := &SMSConfig{}
		require.NoError(t, model.BeforeCreate(nil))
		assert.NotEqual(t, uuid.Nil, model.SMSConfigUUID)
	})

	t.Run("BeforeCreate keeps existing UUID", func(t *testing.T) {
		id := uuid.MustParse("00000000-0000-0000-0000-000000000102")
		model := &SMSConfig{SMSConfigUUID: id}
		require.NoError(t, model.BeforeCreate(nil))
		assert.Equal(t, id, model.SMSConfigUUID)
	})
}

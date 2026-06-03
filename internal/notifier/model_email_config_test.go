package notifier

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailConfigModel(t *testing.T) {
	t.Run("table name", func(t *testing.T) {
		assert.Equal(t, "email_config", EmailConfig{}.TableName())
	})

	t.Run("BeforeCreate assigns UUID", func(t *testing.T) {
		model := &EmailConfig{}
		require.NoError(t, model.BeforeCreate(nil))
		assert.NotEqual(t, uuid.Nil, model.EmailConfigUUID)
	})

	t.Run("BeforeCreate keeps existing UUID", func(t *testing.T) {
		id := uuid.MustParse("00000000-0000-0000-0000-000000000101")
		model := &EmailConfig{EmailConfigUUID: id}
		require.NoError(t, model.BeforeCreate(nil))
		assert.Equal(t, id, model.EmailConfigUUID)
	})
}

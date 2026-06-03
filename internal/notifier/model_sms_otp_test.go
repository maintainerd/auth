package notifier

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSMSOtpModel(t *testing.T) {
	t.Run("table name", func(t *testing.T) {
		assert.Equal(t, "sms_otps", SMSOtp{}.TableName())
	})

	t.Run("BeforeCreate assigns UUID", func(t *testing.T) {
		model := &SMSOtp{}
		require.NoError(t, model.BeforeCreate(nil))
		assert.NotEqual(t, uuid.Nil, model.SMSOtpUUID)
	})

	t.Run("BeforeCreate keeps existing UUID", func(t *testing.T) {
		id := uuid.MustParse("00000000-0000-0000-0000-000000000103")
		model := &SMSOtp{SMSOtpUUID: id}
		require.NoError(t, model.BeforeCreate(nil))
		assert.Equal(t, id, model.SMSOtpUUID)
	})
}

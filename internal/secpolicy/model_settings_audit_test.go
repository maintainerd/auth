package secpolicy

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecuritySettingsAudit_TableName(t *testing.T) {
	assert.Equal(t, "security_settings_audit", SecuritySettingsAudit{}.TableName())
}

func TestSecuritySettingsAudit_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid when missing", func(t *testing.T) {
		audit := &SecuritySettingsAudit{}
		err := audit.BeforeCreate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, audit.SecuritySettingsAuditUUID)
	})

	t.Run("preserves existing uuid", func(t *testing.T) {
		existing := uuid.New()
		audit := &SecuritySettingsAudit{SecuritySettingsAuditUUID: existing}
		err := audit.BeforeCreate(nil)
		require.NoError(t, err)
		assert.Equal(t, existing, audit.SecuritySettingsAuditUUID)
	})
}

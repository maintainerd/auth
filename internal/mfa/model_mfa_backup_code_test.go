package mfa

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserMFABackupCodeModel(t *testing.T) {
	t.Run("table name", func(t *testing.T) {
		assert.Equal(t, "user_mfa_backup_codes", UserMFABackupCode{}.TableName())
	})

	t.Run("BeforeCreate assigns UUID", func(t *testing.T) {
		code := &UserMFABackupCode{}
		require.NoError(t, code.BeforeCreate(nil))
		assert.NotEqual(t, uuid.Nil, code.BackupCodeUUID)
	})

	t.Run("BeforeCreate keeps existing UUID", func(t *testing.T) {
		id := uuid.MustParse("00000000-0000-0000-0000-000000000123")
		code := &UserMFABackupCode{BackupCodeUUID: id}
		require.NoError(t, code.BeforeCreate(nil))
		assert.Equal(t, id, code.BackupCodeUUID)
	})
}

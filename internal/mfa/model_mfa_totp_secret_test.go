package mfa

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserMFATOTPSecretModel(t *testing.T) {
	t.Run("table name", func(t *testing.T) {
		assert.Equal(t, "user_mfa_totp_secrets", UserMFATOTPSecret{}.TableName())
	})

	t.Run("BeforeCreate assigns UUID", func(t *testing.T) {
		secret := &UserMFATOTPSecret{}
		require.NoError(t, secret.BeforeCreate(nil))
		assert.NotEqual(t, uuid.Nil, secret.TOTPSecretUUID)
	})

	t.Run("BeforeCreate keeps existing UUID", func(t *testing.T) {
		id := uuid.MustParse("00000000-0000-0000-0000-000000000124")
		secret := &UserMFATOTPSecret{TOTPSecretUUID: id}
		require.NoError(t, secret.BeforeCreate(nil))
		assert.Equal(t, id, secret.TOTPSecretUUID)
	})
}

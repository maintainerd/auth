package mfa

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserWebAuthnCredentialModel(t *testing.T) {
	t.Run("table name", func(t *testing.T) {
		assert.Equal(t, "user_webauthn_credentials", UserWebAuthnCredential{}.TableName())
	})

	t.Run("BeforeCreate assigns UUID", func(t *testing.T) {
		credential := &UserWebAuthnCredential{}
		require.NoError(t, credential.BeforeCreate(nil))
		assert.NotEqual(t, uuid.Nil, credential.CredentialUUID)
	})

	t.Run("BeforeCreate keeps existing UUID", func(t *testing.T) {
		id := uuid.MustParse("00000000-0000-0000-0000-000000000125")
		credential := &UserWebAuthnCredential{CredentialUUID: id}
		require.NoError(t, credential.BeforeCreate(nil))
		assert.Equal(t, id, credential.CredentialUUID)
	})
}

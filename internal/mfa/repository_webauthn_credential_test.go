package mfa

import (
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserWebAuthnCredentialRepository(t *testing.T) {
	t.Run("constructor and WithTx", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		repo := NewUserWebAuthnCredentialRepository(db)
		require.NotNil(t, repo)
		assert.NotNil(t, repo.(*userWebAuthnCredentialRepository).WithTx(db))
	})

	t.Run("FindByUserID returns rows", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectMFASelect(mock, "user_webauthn_credentials").WillReturnRows(userWebAuthnCredentialRows())

		got, err := NewUserWebAuthnCredentialRepository(db).FindByUserID(mfaTestUserID)

		require.NoError(t, err)
		assert.Len(t, got, 1)
		assertExpectationsMet(t, mock)
	})

	t.Run("FindByCredentialKeyID success not found and error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectMFASelect(mock, "user_webauthn_credentials").WillReturnRows(userWebAuthnCredentialRows())
		expectMFASelect(mock, "user_webauthn_credentials").WillReturnError(gorm.ErrRecordNotFound)
		expectMFASelect(mock, "user_webauthn_credentials").WillReturnError(errors.New("db error"))
		repo := NewUserWebAuthnCredentialRepository(db)

		got, err := repo.FindByCredentialKeyID("cred-key")
		require.NoError(t, err)
		require.NotNil(t, got)
		got, err = repo.FindByCredentialKeyID("cred-key")
		require.NoError(t, err)
		assert.Nil(t, got)
		got, err = repo.FindByCredentialKeyID("cred-key")
		require.Error(t, err)
		assert.Nil(t, got)
		assertExpectationsMet(t, mock)
	})

	t.Run("create update and delete methods", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO "user_webauthn_credentials"`).WillReturnRows(sqlmock.NewRows([]string{"credential_id"}).AddRow(1))
		mock.ExpectCommit()
		for i := 0; i < 2; i++ {
			mock.ExpectBegin()
			expectMFAUpdate(mock, "user_webauthn_credentials").WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()
		}
		for i := 0; i < 2; i++ {
			mock.ExpectBegin()
			expectMFADelete(mock, "user_webauthn_credentials").WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()
		}
		repo := NewUserWebAuthnCredentialRepository(db)

		require.NoError(t, repo.CreateCredential(&UserWebAuthnCredential{UserID: mfaTestUserID, CredentialKeyID: "cred-key", PublicKey: []byte("public")}))
		require.NoError(t, repo.UpdateSignCount(1, 2))
		require.NoError(t, repo.UpdateLastUsed(1))
		require.NoError(t, repo.DeleteCredentialByID(1, mfaTestUserID))
		require.NoError(t, repo.DeleteAllByUserID(mfaTestUserID))
		assertExpectationsMet(t, mock)
	})
}

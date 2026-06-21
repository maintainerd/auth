package mfa

import (
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserMFATOTPSecretRepository(t *testing.T) {
	t.Run("constructor and WithTx", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		repo := NewUserMFATOTPSecretRepository(db)
		require.NotNil(t, repo)
		assert.NotNil(t, repo.(*userMFATOTPSecretRepository).WithTx(db))
	})

	t.Run("FindByUserID success not found and error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectMFASelect(mock, "user_mfa_totp_secrets").WillReturnRows(userTOTPSecretRows())
		expectMFASelect(mock, "user_mfa_totp_secrets").WillReturnError(gorm.ErrRecordNotFound)
		expectMFASelect(mock, "user_mfa_totp_secrets").WillReturnError(errors.New("db error"))
		repo := NewUserMFATOTPSecretRepository(db)

		got, err := repo.FindByUserID(mfaTestUserID)
		require.NoError(t, err)
		require.NotNil(t, got)
		got, err = repo.FindByUserID(mfaTestUserID)
		require.NoError(t, err)
		assert.Nil(t, got)
		got, err = repo.FindByUserID(mfaTestUserID)
		require.Error(t, err)
		assert.Nil(t, got)
		assertExpectationsMet(t, mock)
	})

	t.Run("Upsert creates missing secret updates existing and returns lookup error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectMFASelect(mock, "user_mfa_totp_secrets").WillReturnError(gorm.ErrRecordNotFound)
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO "user_mfa_totp_secrets"`).WillReturnRows(sqlmock.NewRows([]string{"totp_secret_id"}).AddRow(1))
		mock.ExpectCommit()
		expectMFASelect(mock, "user_mfa_totp_secrets").WillReturnRows(userTOTPSecretRows())
		mock.ExpectBegin()
		expectMFAUpdate(mock, "user_mfa_totp_secrets").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		expectMFASelect(mock, "user_mfa_totp_secrets").WillReturnError(errors.New("db error"))
		repo := NewUserMFATOTPSecretRepository(db)

		require.NoError(t, repo.Upsert(&UserMFATOTPSecret{UserID: mfaTestUserID, Secret: "secret"}))
		require.NoError(t, repo.Upsert(&UserMFATOTPSecret{UserID: mfaTestUserID, Secret: "secret"}))
		require.Error(t, repo.Upsert(&UserMFATOTPSecret{UserID: mfaTestUserID, Secret: "secret"}))
		assertExpectationsMet(t, mock)
	})

	t.Run("state mutations", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		for i := 0; i < 5; i++ {
			mock.ExpectBegin()
			if i == 4 {
				expectMFADelete(mock, "user_mfa_totp_secrets").WillReturnResult(sqlmock.NewResult(0, 1))
			} else {
				expectMFAUpdate(mock, "user_mfa_totp_secrets").WillReturnResult(sqlmock.NewResult(0, 1))
			}
			mock.ExpectCommit()
		}
		repo := NewUserMFATOTPSecretRepository(db)

		require.NoError(t, repo.Enable(mfaTestUserID))
		require.NoError(t, repo.Disable(mfaTestUserID))
		require.NoError(t, repo.UpdateLastUsed(mfaTestUserID))
		accepted, err := repo.MarkStepUsed(mfaTestUserID, 123)
		require.NoError(t, err)
		assert.True(t, accepted)
		require.NoError(t, repo.DeleteByUserID(mfaTestUserID))
		assertExpectationsMet(t, mock)
	})

	t.Run("MarkStepUsed false when no rows updated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		expectMFAUpdate(mock, "user_mfa_totp_secrets").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		accepted, err := NewUserMFATOTPSecretRepository(db).MarkStepUsed(mfaTestUserID, 123)

		require.NoError(t, err)
		assert.False(t, accepted)
		assertExpectationsMet(t, mock)
	})
}

package mfa

import (
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserMFABackupCodeRepository(t *testing.T) {
	t.Run("constructor and WithTx", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		repo := NewUserMFABackupCodeRepository(db)
		require.NotNil(t, repo)
		assert.NotNil(t, repo.(*userMFABackupCodeRepository).WithTx(db))
	})

	t.Run("CreateBulk stores codes", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO "user_mfa_backup_codes"`).WillReturnRows(sqlmock.NewRows([]string{"backup_code_id"}).AddRow(1))
		mock.ExpectCommit()

		err := NewUserMFABackupCodeRepository(db).CreateBulk([]*UserMFABackupCode{{UserID: mfaTestUserID, CodeHash: "hash"}})

		require.NoError(t, err)
		assertExpectationsMet(t, mock)
	})

	t.Run("FindUnusedByUserID returns rows", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectMFASelect(mock, "user_mfa_backup_codes").WillReturnRows(userBackupCodeRows())

		got, err := NewUserMFABackupCodeRepository(db).FindUnusedByUserID(mfaTestUserID)

		require.NoError(t, err)
		assert.Len(t, got, 1)
		assertExpectationsMet(t, mock)
	})

	t.Run("FindByUserIDAndCodeHash success not found and error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectMFASelect(mock, "user_mfa_backup_codes").WillReturnRows(userBackupCodeRows())
		expectMFASelect(mock, "user_mfa_backup_codes").WillReturnError(gorm.ErrRecordNotFound)
		expectMFASelect(mock, "user_mfa_backup_codes").WillReturnError(errors.New("db error"))
		repo := NewUserMFABackupCodeRepository(db)

		got, err := repo.FindByUserIDAndCodeHash(mfaTestUserID, "hash")
		require.NoError(t, err)
		require.NotNil(t, got)
		got, err = repo.FindByUserIDAndCodeHash(mfaTestUserID, "hash")
		require.NoError(t, err)
		assert.Nil(t, got)
		got, err = repo.FindByUserIDAndCodeHash(mfaTestUserID, "hash")
		require.Error(t, err)
		assert.Nil(t, got)
		assertExpectationsMet(t, mock)
	})

	t.Run("MarkUsed and DeleteAllByUserID mutate rows", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		expectMFAUpdate(mock, "user_mfa_backup_codes").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		mock.ExpectBegin()
		expectMFADelete(mock, "user_mfa_backup_codes").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		repo := NewUserMFABackupCodeRepository(db)

		require.NoError(t, repo.MarkUsed(1))
		require.NoError(t, repo.DeleteAllByUserID(mfaTestUserID))
		assertExpectationsMet(t, mock)
	})
}

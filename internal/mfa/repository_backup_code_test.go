package mfa

import (
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserBackupCodeRepository(t *testing.T) {
	t.Run("constructor and WithTx", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		repo := NewUserBackupCodeRepository(db)
		require.NotNil(t, repo)
		assert.NotNil(t, repo.(*userBackupCodeRepository).WithTx(db))
	})

	t.Run("CreateBulk stores codes", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO "user_backup_codes"`).WillReturnRows(sqlmock.NewRows([]string{"backup_code_id"}).AddRow(1))
		mock.ExpectCommit()

		err := NewUserBackupCodeRepository(db).CreateBulk([]*UserBackupCode{{UserID: mfaTestUserID, CodeHash: "hash"}})

		require.NoError(t, err)
		assertExpectationsMet(t, mock)
	})

	t.Run("FindUnusedByUserID returns rows", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectMFASelect(mock, "user_backup_codes").WillReturnRows(userBackupCodeRows())

		got, err := NewUserBackupCodeRepository(db).FindUnusedByUserID(mfaTestUserID)

		require.NoError(t, err)
		assert.Len(t, got, 1)
		assertExpectationsMet(t, mock)
	})

	t.Run("FindByUserIDAndCodeHash success not found and error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectMFASelect(mock, "user_backup_codes").WillReturnRows(userBackupCodeRows())
		expectMFASelect(mock, "user_backup_codes").WillReturnError(gorm.ErrRecordNotFound)
		expectMFASelect(mock, "user_backup_codes").WillReturnError(errors.New("db error"))
		repo := NewUserBackupCodeRepository(db)

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
		expectMFAUpdate(mock, "user_backup_codes").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		mock.ExpectBegin()
		expectMFADelete(mock, "user_backup_codes").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		repo := NewUserBackupCodeRepository(db)

		require.NoError(t, repo.MarkUsed(1))
		require.NoError(t, repo.DeleteAllByUserID(mfaTestUserID))
		assertExpectationsMet(t, mock)
	})
}

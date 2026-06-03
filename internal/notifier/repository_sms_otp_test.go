package notifier

import (
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSMSOtpRepository(t *testing.T) {
	t.Run("constructor and WithTx", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		repo := NewSMSOtpRepository(db)
		require.NotNil(t, repo)
		assert.NotNil(t, repo.(*smsOtpRepository).WithTx(db))
	})

	t.Run("FindValidByPhone success not found and error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectNotifierSelect(mock, "sms_otps").WillReturnRows(smsOtpRows())
		expectNotifierSelect(mock, "sms_otps").WillReturnError(gorm.ErrRecordNotFound)
		expectNotifierSelect(mock, "sms_otps").WillReturnError(errors.New("db down"))
		repo := NewSMSOtpRepository(db)

		got, err := repo.FindValidByPhone("+15551234567")
		require.NoError(t, err)
		require.NotNil(t, got)
		got, err = repo.FindValidByPhone("+15551234567")
		require.NoError(t, err)
		assert.Nil(t, got)
		got, err = repo.FindValidByPhone("+15551234567")
		require.Error(t, err)
		assert.Nil(t, got)
		assertNotifierExpectationsMet(t, mock)
	})

	t.Run("RecordFailure and MarkUsed update rows", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		expectNotifierUpdate(mock, "sms_otps").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		mock.ExpectBegin()
		expectNotifierUpdate(mock, "sms_otps").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		repo := NewSMSOtpRepository(db)

		require.NoError(t, repo.RecordFailure(1, 3))
		require.NoError(t, repo.MarkUsed(1))
		assertNotifierExpectationsMet(t, mock)
	})
}

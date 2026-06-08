package notifier

import (
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserOTPModel(t *testing.T) {
	t.Run("table name", func(t *testing.T) {
		assert.Equal(t, "user_otps", UserOTP{}.TableName())
	})

	t.Run("before create assigns UUID", func(t *testing.T) {
		model := &UserOTP{}
		require.NoError(t, model.BeforeCreate(nil))
		assert.NotEqual(t, "00000000-0000-0000-0000-000000000000", model.UserOTPUUID.String())
	})

	t.Run("before create preserves existing UUID", func(t *testing.T) {
		id := testTenantUUID
		model := &UserOTP{UserOTPUUID: id}
		require.NoError(t, model.BeforeCreate(nil))
		assert.Equal(t, id, model.UserOTPUUID)
	})
}

func TestUserOTPRepository(t *testing.T) {
	t.Run("WithTx", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		repo := NewUserOTPRepository(db)
		assert.NotNil(t, repo.(*userOTPRepository).WithTx(db))
	})

	t.Run("FindValid multiple outcomes", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserOTPRepository(db)

		t.Run("found", func(t *testing.T) {
			expectNotifierSelect(mock, "user_otps").WillReturnRows(userOTPRows())
			record, err := repo.FindValid("sms", "+15551234567")
			require.NoError(t, err)
			require.NotNil(t, record)
			assert.Equal(t, int64(1), record.UserOTPID)
		})

		t.Run("not found", func(t *testing.T) {
			expectNotifierSelect(mock, "user_otps").WillReturnError(gorm.ErrRecordNotFound)
			record, err := repo.FindValid("sms", "+15551234567")
			require.NoError(t, err)
			assert.Nil(t, record)
		})

		t.Run("db error", func(t *testing.T) {
			expectNotifierSelect(mock, "user_otps").WillReturnError(errors.New("db down"))
			_, err := repo.FindValid("sms", "+15551234567")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "db down")
		})
	})

	t.Run("RecordFailure", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserOTPRepository(db)
		mock.ExpectBegin()
		expectNotifierUpdate(mock, "user_otps").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		require.NoError(t, repo.RecordFailure(1, 3))
	})

	t.Run("MarkUsed", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserOTPRepository(db)
		mock.ExpectBegin()
		expectNotifierUpdate(mock, "user_otps").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		require.NoError(t, repo.MarkUsed(1))
	})
}

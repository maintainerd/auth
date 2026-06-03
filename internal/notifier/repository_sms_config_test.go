package notifier

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSMSConfigRepository(t *testing.T) {
	t.Run("constructor and WithTx", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		repo := NewSMSConfigRepository(db)
		require.NotNil(t, repo)
		assert.NotNil(t, repo.(*smsConfigRepository).WithTx(db))
	})

	t.Run("FindByTenantID success not found and error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectNotifierSelect(mock, "sms_config").WillReturnRows(smsConfigRows())
		expectNotifierSelect(mock, "sms_config").WillReturnError(gorm.ErrRecordNotFound)
		expectNotifierSelect(mock, "sms_config").WillReturnError(errors.New("db down"))
		repo := NewSMSConfigRepository(db)

		got, err := repo.FindByTenantID(testTenantID)
		require.NoError(t, err)
		require.NotNil(t, got)
		got, err = repo.FindByTenantID(testTenantID)
		require.NoError(t, err)
		assert.Nil(t, got)
		got, err = repo.FindByTenantID(testTenantID)
		require.Error(t, err)
		assert.Nil(t, got)
		assertNotifierExpectationsMet(t, mock)
	})
}

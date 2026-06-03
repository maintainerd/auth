package notifier

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestEmailConfigRepository(t *testing.T) {
	t.Run("constructor and WithTx", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		repo := NewEmailConfigRepository(db)
		require.NotNil(t, repo)
		assert.NotNil(t, repo.(*emailConfigRepository).WithTx(db))
	})

	t.Run("FindByTenantID success not found and error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectNotifierSelect(mock, "email_config").WillReturnRows(emailConfigRows())
		expectNotifierSelect(mock, "email_config").WillReturnError(gorm.ErrRecordNotFound)
		expectNotifierSelect(mock, "email_config").WillReturnError(errors.New("db down"))
		repo := NewEmailConfigRepository(db)

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

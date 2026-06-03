package iam

import (
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func policyRows(values ...driver.Value) *sqlmock.Rows {
	if len(values) == 0 {
		values = []driver.Value{int64(1), testResourceUUID.String(), tenantID, "policy", "desc", []byte(`{"version":"v1","statement":[]}`), "v1", "active", false, time.Now(), time.Now()}
	}
	return sqlmock.NewRows([]string{
		"policy_id", "policy_uuid", "tenant_id", "name", "description", "document", "version", "status", "is_system", "created_at", "updated_at",
	}).AddRow(values...)
}

func TestPolicyRepository(t *testing.T) {
	t.Run("constructor and WithTx", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		repo := NewPolicyRepository(db)
		require.NotNil(t, repo)
		assert.NotNil(t, repo.(*policyRepository).WithTx(db))
	})

	t.Run("single row lookups success not found and error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		for i := 0; i < 3; i++ {
			expectAnySelect(mock, "policies").WillReturnRows(policyRows())
			expectAnySelect(mock, "policies").WillReturnError(gorm.ErrRecordNotFound)
			expectAnySelect(mock, "policies").WillReturnError(errors.New("db error"))
		}
		repo := NewPolicyRepository(db)

		for _, run := range []func() (*Policy, error){
			func() (*Policy, error) { return repo.FindByUUIDAndTenantID(testResourceUUID, tenantID) },
			func() (*Policy, error) { return repo.FindByName("policy", tenantID) },
			func() (*Policy, error) { return repo.FindByNameAndVersion("policy", "v1", tenantID) },
		} {
			got, err := run()
			require.NoError(t, err)
			require.NotNil(t, got)
			got, err = run()
			require.NoError(t, err)
			assert.Nil(t, got)
			got, err = run()
			require.Error(t, err)
			assert.Nil(t, got)
		}
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindSystemPolicies returns rows", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnySelect(mock, "policies").WillReturnRows(policyRows())

		got, err := NewPolicyRepository(db).FindSystemPolicies(tenantID)

		require.NoError(t, err)
		assert.Len(t, got, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("status mutations and delete", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		for i := 0; i < 3; i++ {
			mock.ExpectBegin()
			expectAnyUpdate(mock, "policies").WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()
		}
		repo := NewPolicyRepository(db)

		require.NoError(t, repo.SetStatusByUUID(testResourceUUID, tenantID, "active"))
		require.NoError(t, repo.SetSystemStatusByUUID(testResourceUUID, tenantID, true))
		require.NoError(t, repo.DeleteByUUIDAndTenantID(testResourceUUID, tenantID))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindPaginated applies filters", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnyCount(mock, "policies").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		expectAnySelect(mock, "policies").WillReturnRows(policyRows())
		name := "policy"
		isSystem := true

		got, err := NewPolicyRepository(db).FindPaginated(PolicyRepositoryGetFilter{
			TenantID: tenantID, Name: &name, Description: &name, Version: &name,
			Status: []string{"active"}, IsSystem: &isSystem, ServiceID: &testResourceUUID, Page: 1, Limit: 10,
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Len(t, got.Data, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

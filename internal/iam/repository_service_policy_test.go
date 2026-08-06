package iam

import (
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func servicePolicyRows(values ...driver.Value) *sqlmock.Rows {
	if len(values) == 0 {
		values = []driver.Value{int64(1), testResourceUUID.String(), int64(2), int64(3), time.Now()}
	}
	return sqlmock.NewRows([]string{
		"service_policy_id", "service_policy_uuid", "service_id", "policy_id", "created_at",
	}).AddRow(values...)
}

func TestServicePolicyRepository(t *testing.T) {
	t.Run("constructor and WithTx", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		repo := NewServicePolicyRepository(db)
		require.NotNil(t, repo)
		assert.NotNil(t, repo.(*servicePolicyRepository).WithTx(db))
	})

	t.Run("FindByServiceAndPolicy success not found and error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnySelect(mock, "service_policies").WillReturnRows(servicePolicyRows())
		expectAnySelect(mock, "service_policies").WillReturnError(gorm.ErrRecordNotFound)
		expectAnySelect(mock, "service_policies").WillReturnError(errors.New("db error"))
		repo := NewServicePolicyRepository(db)

		got, err := repo.FindByServiceAndPolicy(2, 3)
		require.NoError(t, err)
		require.NotNil(t, got)
		got, err = repo.FindByServiceAndPolicy(2, 3)
		require.NoError(t, err)
		assert.Nil(t, got)
		got, err = repo.FindByServiceAndPolicy(2, 3)
		require.Error(t, err)
		assert.Nil(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("delete and collection lookups", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "service_policies".*`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		expectAnySelect(mock, "policies").WillReturnRows(policyRows())
		repo := NewServicePolicyRepository(db)

		require.NoError(t, repo.DeleteByServiceAndPolicy(2, 3))
		policies, err := repo.FindPoliciesByServiceID(2)
		require.NoError(t, err)
		assert.Len(t, policies, 1)
		// The FindServicesByPolicyID case went with the method — zero-caller and
		// tenant-unscoped (see repository_service_policy.go).
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// servicePolicySortColumns replaced the global union allowlist, which accepted
	// columns `service_policies` does not have (name, status, updated_at) and
	// turned sort_by=name into a Postgres 42703 → 500.
	t.Run("FindPaginated sorting is limited to the service_policies table", func(t *testing.T) {
		for _, sortBy := range []string{"name", "status", "updated_at", "email"} {
			assert.Equal(t, "created_at DESC",
				database.SanitizeOrderIn(servicePolicySortColumns, sortBy, SortOrderDesc, "created_at DESC"),
				"%q is not a service_policies column and must fall back to the default", sortBy)
		}
		assert.Equal(t, "policy_id ASC",
			database.SanitizeOrderIn(servicePolicySortColumns, "policy_id", "asc", "created_at DESC"))
	})

	t.Run("FindPaginated applies filters", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnyCount(mock, "service_policies").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		expectAnySelect(mock, "service_policies").WillReturnRows(servicePolicyRows())
		serviceID := int64(2)
		policyID := int64(3)

		got, err := NewServicePolicyRepository(db).FindPaginated(ServicePolicyRepositoryGetFilter{
			ServiceID: &serviceID, PolicyID: &policyID, Page: 1, Limit: 10,
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Len(t, got.Data, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

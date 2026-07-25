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

func serviceRows(values ...driver.Value) *sqlmock.Rows {
	if len(values) == 0 {
		values = []driver.Value{int64(1), testResourceUUID.String(), "service", "Service", "desc", "v1", "active", false, time.Now(), time.Now()}
	}
	return sqlmock.NewRows([]string{
		"service_id", "service_uuid", "name", "display_name", "description", "version", "status", "is_system", "created_at", "updated_at",
	}).AddRow(values...)
}

func TestServiceRepository(t *testing.T) {
	t.Run("constructor and WithTx", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		repo := NewServiceRepository(db)
		require.NotNil(t, repo)
		assert.NotNil(t, repo.(*serviceRepository).WithTx(db))
	})

	t.Run("name lookups success not found and error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		for i := 0; i < 2; i++ {
			expectAnySelect(mock, "services").WillReturnRows(serviceRows())
			expectAnySelect(mock, "services").WillReturnError(gorm.ErrRecordNotFound)
			expectAnySelect(mock, "services").WillReturnError(errors.New("db error"))
		}
		repo := NewServiceRepository(db)

		for _, run := range []func() (*Service, error){
			func() (*Service, error) { return repo.FindByName("service") },
			func() (*Service, error) { return repo.FindByNameAndTenantID("service", tenantID) },
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

	t.Run("FindByTenantID returns rows", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnySelect(mock, "services").WillReturnRows(serviceRows())

		got, err := NewServiceRepository(db).FindByTenantID(tenantID)

		require.NoError(t, err)
		assert.Len(t, got, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindPaginated applies filters", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnyCount(mock, "services").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		expectAnySelect(mock, "services").WillReturnRows(serviceRows())
		name := "service"
		flag := true

		got, err := NewServiceRepository(db).FindPaginated(ServiceRepositoryGetFilter{
			Name: &name, DisplayName: &name, Description: &name, Version: &name, TenantID: ptrInt64(tenantID),
			Status: []string{"active"}, IsSystem: &flag, Page: 1, Limit: 10,
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Len(t, got.Data, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindServicesByPolicyUUID applies filters", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnyCount(mock, "services").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		expectAnyCount(mock, "services").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		expectAnySelect(mock, "services").WillReturnRows(serviceRows())
		name := "service"
		flag := true

		got, err := NewServiceRepository(db).FindServicesByPolicyUUID(testResourceUUID, ServiceRepositoryGetFilter{
			Name: &name, DisplayName: &name, Description: &name, Version: &name, Status: []string{"active"},
			IsSystem: &flag, Page: 1, Limit: 10,
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Len(t, got.Data, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindServicesByPolicyUUID count error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnyCount(mock, "services").WillReturnError(errors.New("count error"))

		got, err := NewServiceRepository(db).FindServicesByPolicyUUID(testResourceUUID, ServiceRepositoryGetFilter{})

		require.Error(t, err)
		assert.Nil(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("SetStatusByUUID updates and CountPoliciesByServiceID counts", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		expectAnyUpdate(mock, "services").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		expectAnyCount(mock, "service_policies").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		repo := NewServiceRepository(db)
		require.NoError(t, repo.SetStatusByUUID(testResourceUUID, tenantID, "active"))
		count, err := repo.CountPoliciesByServiceID(1)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func ptrInt64(v int64) *int64 { return &v }

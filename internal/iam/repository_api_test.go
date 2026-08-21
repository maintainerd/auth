package iam

import (
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func expectAnySelect(mock sqlmock.Sqlmock, table string) *sqlmock.ExpectedQuery {
	return mock.ExpectQuery(`SELECT .* FROM "` + table + `".*`)
}

func expectAnyCount(mock sqlmock.Sqlmock, table string) *sqlmock.ExpectedQuery {
	return mock.ExpectQuery(`SELECT count\(\*\) FROM "` + table + `".*`)
}

func expectAnyUpdate(mock sqlmock.Sqlmock, table string) *sqlmock.ExpectedExec {
	return mock.ExpectExec(`UPDATE "` + table + `".*`)
}

func expectAnyDelete(mock sqlmock.Sqlmock, table string) *sqlmock.ExpectedExec {
	return mock.ExpectExec(`UPDATE "` + table + `".*"deleted_at".*`)
}

func apiRows(values ...driver.Value) *sqlmock.Rows {
	if len(values) == 0 {
		values = []driver.Value{int64(1), testResourceUUID.String(), tenantID, int64(0), "api", "API", "desc", "svc:api", "active", false, time.Now(), time.Now()}
	}
	return sqlmock.NewRows([]string{
		"api_id", "api_uuid", "tenant_id", "service_id", "name", "display_name", "description", "identifier", "status", "is_system", "created_at", "updated_at",
	}).AddRow(values...)
}

func TestAPIRepository(t *testing.T) {
	t.Run("constructor and WithTx", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		repo := NewAPIRepository(db)

		require.NotNil(t, repo)
		assert.NotNil(t, repo.(*apiRepository).WithTx(db))
	})

	t.Run("FindByUUIDAndTenantID success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnySelect(mock, "apis").WillReturnRows(apiRows())

		got, err := NewAPIRepository(db).FindByUUIDAndTenantID(testResourceUUID, tenantID)

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, testResourceUUID, got.APIUUID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByUUIDAndTenantID not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnySelect(mock, "apis").WillReturnError(gorm.ErrRecordNotFound)

		got, err := NewAPIRepository(db).FindByUUIDAndTenantID(testResourceUUID, tenantID)

		require.NoError(t, err)
		assert.Nil(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByUUIDAndTenantID query error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnySelect(mock, "apis").WillReturnError(errors.New("db error"))

		got, err := NewAPIRepository(db).FindByUUIDAndTenantID(testResourceUUID, tenantID)

		require.Error(t, err)
		assert.Nil(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByName success and not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnySelect(mock, "apis").WillReturnRows(apiRows())
		expectAnySelect(mock, "apis").WillReturnError(gorm.ErrRecordNotFound)

		repo := NewAPIRepository(db)
		got, err := repo.FindByName("api", tenantID)
		require.NoError(t, err)
		require.NotNil(t, got)

		missing, err := repo.FindByName("missing", tenantID)
		require.NoError(t, err)
		assert.Nil(t, missing)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByName query error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnySelect(mock, "apis").WillReturnError(errors.New("db error"))

		got, err := NewAPIRepository(db).FindByName("api", tenantID)

		require.Error(t, err)
		assert.Nil(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByIdentifier returns row or error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnySelect(mock, "apis").WillReturnRows(apiRows())
		expectAnySelect(mock, "apis").WillReturnError(errors.New("db error"))

		repo := NewAPIRepository(db)
		got, err := repo.FindByIdentifier("svc:api", tenantID)
		require.NoError(t, err)
		require.NotNil(t, got)

		got, err = repo.FindByIdentifier("svc:missing", tenantID)
		require.Error(t, err)
		// On a query error the repo returns (nil, err), like every sibling finder.
		assert.Nil(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindPaginated applies filters", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnyCount(mock, "apis").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		expectAnySelect(mock, "apis").WillReturnRows(apiRows())
		name := "api"
		identifier := "svc:api"
		serviceID := int64(7)
		statuses := []string{"active"}
		isSystem := true

		got, err := NewAPIRepository(db).FindPaginated(APIRepositoryGetFilter{
			TenantID: tenantID, Name: &name, DisplayName: &name, Identifier: &identifier,
			ServiceID: &serviceID, Status: statuses, IsSystem: &isSystem, Page: 1, Limit: 10,
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Len(t, got.Data, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("SetStatusByUUID updates status", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		expectAnyUpdate(mock, "apis").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := NewAPIRepository(db).SetStatusByUUID(testResourceUUID, tenantID, "active")

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CountByServiceID returns count", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnyCount(mock, "apis").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

		count, err := NewAPIRepository(db).CountByServiceID(7, tenantID)

		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DeleteByUUIDAndTenantID soft deletes", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "apis"`) + `.*`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := NewAPIRepository(db).DeleteByUUIDAndTenantID(testResourceUUID, tenantID)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

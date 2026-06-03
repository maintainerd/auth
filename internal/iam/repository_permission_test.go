package iam

import (
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func permissionRows(values ...driver.Value) *sqlmock.Rows {
	if len(values) == 0 {
		values = []driver.Value{int64(1), testResourceUUID.String(), tenantID, int64(9), "permission", "desc", "active", false, false, time.Now(), time.Now()}
	}
	return sqlmock.NewRows([]string{
		"permission_id", "permission_uuid", "tenant_id", "api_id", "name", "description", "status", "is_default", "is_system", "created_at", "updated_at",
	}).AddRow(values...)
}

func TestPermissionRepository(t *testing.T) {
	t.Run("constructor and WithTx", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		repo := NewPermissionRepository(db)
		require.NotNil(t, repo)
		assert.NotNil(t, repo.(*permissionRepository).WithTx(db))
	})

	t.Run("FindByUUIDAndTenantID success not found and error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnySelect(mock, "permissions").WillReturnRows(permissionRows())
		expectAnySelect(mock, "apis").WillReturnRows(apiRows(int64(9), uuid.New().String(), tenantID, int64(0), "api", "API", "desc", "rest", "svc:api", "active", false, time.Now(), time.Now()))
		expectAnySelect(mock, "permissions").WillReturnError(gorm.ErrRecordNotFound)
		expectAnySelect(mock, "permissions").WillReturnError(errors.New("db error"))

		repo := NewPermissionRepository(db)
		got, err := repo.FindByUUIDAndTenantID(testResourceUUID, tenantID)
		require.NoError(t, err)
		require.NotNil(t, got)

		got, err = repo.FindByUUIDAndTenantID(testResourceUUID, tenantID)
		require.NoError(t, err)
		assert.Nil(t, got)

		got, err = repo.FindByUUIDAndTenantID(testResourceUUID, tenantID)
		require.Error(t, err)
		assert.Nil(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByName success not found and error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnySelect(mock, "permissions").WillReturnRows(permissionRows())
		expectAnySelect(mock, "permissions").WillReturnError(gorm.ErrRecordNotFound)
		expectAnySelect(mock, "permissions").WillReturnError(errors.New("db error"))

		repo := NewPermissionRepository(db)
		got, err := repo.FindByName("permission", tenantID)
		require.NoError(t, err)
		require.NotNil(t, got)

		got, err = repo.FindByName("missing", tenantID)
		require.NoError(t, err)
		assert.Nil(t, got)

		got, err = repo.FindByName("permission", tenantID)
		require.Error(t, err)
		assert.Nil(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindPaginated applies filters", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnyCount(mock, "permissions").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		expectAnySelect(mock, "permissions").WillReturnRows(permissionRows())
		expectAnySelect(mock, "apis").WillReturnRows(apiRows(int64(9), uuid.New().String(), tenantID, int64(0), "api", "API", "desc", "rest", "svc:api", "active", false, time.Now(), time.Now()))
		name := "permission"
		apiID := int64(9)
		roleID := int64(8)
		status := "active"
		flag := true

		got, err := NewPermissionRepository(db).FindPaginated(PermissionRepositoryGetFilter{
			TenantID: tenantID, Name: &name, Description: &name, APIID: &apiID, RoleID: &roleID,
			Status: &status, IsDefault: &flag, IsSystem: &flag, Page: 1, Limit: 10,
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Len(t, got.Data, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DeleteByUUIDAndTenantID success not found and error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		expectAnyDelete(mock, "permissions").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		mock.ExpectBegin()
		expectAnyDelete(mock, "permissions").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
		mock.ExpectBegin()
		expectAnyDelete(mock, "permissions").WillReturnError(errors.New("db error"))
		mock.ExpectRollback()

		repo := NewPermissionRepository(db)
		require.NoError(t, repo.DeleteByUUIDAndTenantID(testResourceUUID, tenantID))
		require.ErrorIs(t, repo.DeleteByUUIDAndTenantID(testResourceUUID, tenantID), gorm.ErrRecordNotFound)
		require.Error(t, repo.DeleteByUUIDAndTenantID(testResourceUUID, tenantID))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

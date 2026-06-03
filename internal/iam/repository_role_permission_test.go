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

func rolePermissionRows(values ...driver.Value) *sqlmock.Rows {
	if len(values) == 0 {
		values = []driver.Value{int64(1), testResourceUUID.String(), int64(2), int64(3), time.Now()}
	}
	return sqlmock.NewRows([]string{
		"role_permission_id", "role_permission_uuid", "role_id", "permission_id", "created_at",
	}).AddRow(values...)
}

func TestRolePermissionRepository(t *testing.T) {
	t.Run("constructor WithTx and Assign", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO "role_permissions".*RETURNING`).WillReturnRows(rolePermissionRows())
		mock.ExpectCommit()
		repo := NewRolePermissionRepository(db)

		got, err := repo.Assign(&RolePermission{RoleID: 2, PermissionID: 3})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.NotNil(t, repo.(*rolePermissionRepository).WithTx(db))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByRoleAndPermission success not found and error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnySelect(mock, "role_permissions").WillReturnRows(rolePermissionRows())
		expectAnySelect(mock, "role_permissions").WillReturnError(gorm.ErrRecordNotFound)
		expectAnySelect(mock, "role_permissions").WillReturnError(errors.New("db error"))
		repo := NewRolePermissionRepository(db)

		got, err := repo.FindByRoleAndPermission(2, 3)
		require.NoError(t, err)
		require.NotNil(t, got)
		got, err = repo.FindByRoleAndPermission(2, 3)
		require.NoError(t, err)
		assert.Nil(t, got)
		got, err = repo.FindByRoleAndPermission(2, 3)
		require.Error(t, err)
		assert.Nil(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindAll helpers return rows", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnySelect(mock, "role_permissions").WillReturnRows(rolePermissionRows())
		expectAnySelect(mock, "role_permissions").WillReturnRows(rolePermissionRows())
		repo := NewRolePermissionRepository(db)

		byRole, err := repo.FindAllByRoleID(2)
		require.NoError(t, err)
		assert.Len(t, byRole, 1)
		byPermission, err := repo.FindAllByPermissionID(3)
		require.NoError(t, err)
		assert.Len(t, byPermission, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("delete and update helpers", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "role_permissions".*`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		mock.ExpectBegin()
		expectAnyUpdate(mock, "role_permissions").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		repo := NewRolePermissionRepository(db)

		require.NoError(t, repo.RemoveByRoleAndPermission(2, 3))
		require.NoError(t, repo.SetDefaultStatusByUUID(testResourceUUID, true))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

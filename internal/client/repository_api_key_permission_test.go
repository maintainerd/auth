package client

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPIKeyPermissionRepository(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewAPIKeyPermissionRepository(gdb)
	assert.NotNil(t, repo)
}

func TestAPIKeyPermissionRepository_WithTx(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewAPIKeyPermissionRepository(gdb)
	txRepo := repo.WithTx(gdb)
	assert.NotNil(t, txRepo)
}

func TestAPIKeyPermissionRepository_FindByAPIKeyAPIAndPermission(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "api_key_permissions" WHERE.*api_key_api_id = \$1.*permission_id = \$2`).
			WithArgs(int64(1), int64(2), 1).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_permission_id", "api_key_permission_uuid", "api_key_api_id", "permission_id", "created_at"}).
				AddRow(1, id, 1, 2, now))
		result, err := NewAPIKeyPermissionRepository(gdb).FindByAPIKeyAPIAndPermission(1, 2)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(1), result.APIKeyAPIID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "api_key_permissions" WHERE.*api_key_api_id = \$1.*permission_id = \$2`).
			WithArgs(int64(1), int64(2), 1).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_permission_id", "api_key_permission_uuid", "api_key_api_id"}))
		result, err := NewAPIKeyPermissionRepository(gdb).FindByAPIKeyAPIAndPermission(1, 2)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "api_key_permissions" WHERE.*api_key_api_id = \$1.*permission_id = \$2`).
			WithArgs(int64(1), int64(2), 1).
			WillReturnError(assert.AnError)
		result, err := NewAPIKeyPermissionRepository(gdb).FindByAPIKeyAPIAndPermission(1, 2)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAPIKeyPermissionRepository_RemoveByAPIKeyAPIAndPermission(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "api_key_permissions" WHERE.*api_key_api_id = \$1.*permission_id = \$2`).
			WithArgs(int64(1), int64(2)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		err := NewAPIKeyPermissionRepository(gdb).RemoveByAPIKeyAPIAndPermission(1, 2)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "api_key_permissions" WHERE.*api_key_api_id = \$1.*permission_id = \$2`).
			WithArgs(int64(1), int64(2)).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()
		err := NewAPIKeyPermissionRepository(gdb).RemoveByAPIKeyAPIAndPermission(1, 2)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAPIKeyPermissionRepository_FindByAPIKeyAPIID(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "api_key_permissions" WHERE.*api_key_api_id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_permission_id", "api_key_permission_uuid", "api_key_api_id", "permission_id", "created_at"}).
				AddRow(1, id, 1, 2, now))

		mock.ExpectQuery(`SELECT \* FROM "permissions" WHERE "permissions"\."permission_id" = \$1`).
			WithArgs(int64(2)).
			WillReturnRows(sqlmock.NewRows([]string{"permission_id", "permission_uuid", "name", "status", "created_at", "updated_at"}).
				AddRow(2, uuid.New(), "test-perm", "active", now, now))

		result, err := NewAPIKeyPermissionRepository(gdb).FindByAPIKeyAPIID(1)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns empty", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "api_key_permissions" WHERE.*api_key_api_id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_permission_id", "api_key_permission_uuid", "api_key_api_id", "permission_id", "created_at"}))
		result, err := NewAPIKeyPermissionRepository(gdb).FindByAPIKeyAPIID(1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "api_key_permissions" WHERE.*api_key_api_id = \$1`).
			WithArgs(int64(1)).
			WillReturnError(assert.AnError)
		result, err := NewAPIKeyPermissionRepository(gdb).FindByAPIKeyAPIID(1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

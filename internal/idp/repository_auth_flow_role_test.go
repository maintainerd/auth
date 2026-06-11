package idp

import (
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthFlowRoleRepository(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewAuthFlowRoleRepository(gdb)
	assert.NotNil(t, repo)
}

func TestAuthFlowRoleRepository_WithTx(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewAuthFlowRoleRepository(gdb)
	txRepo := repo.WithTx(gdb)
	assert.NotNil(t, txRepo)
}

func TestAuthFlowRoleRepository_FindByAuthFlowID(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "auth_flow_roles" WHERE .*auth_flow_id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"auth_flow_role_id", "auth_flow_role_uuid", "auth_flow_id", "role_id", "created_at"}).
				AddRow(1, uuid.New(), int64(1), int64(10), now).
				AddRow(2, uuid.New(), int64(1), int64(20), now))
		mock.ExpectQuery(`SELECT \* FROM "roles" WHERE "roles"\."role_id" IN \(\$1,\$2\)`).
			WithArgs(int64(10), int64(20)).
			WillReturnRows(sqlmock.NewRows([]string{"role_id", "role_uuid", "name", "created_at", "updated_at"}).
				AddRow(10, uuid.New(), "admin", now, now).
				AddRow(20, uuid.New(), "user", now, now))
		repo := NewAuthFlowRoleRepository(gdb)
		result, err := repo.FindByAuthFlowID(1)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty returns empty slice", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "auth_flow_roles" WHERE .*auth_flow_id = \$1`).
			WithArgs(int64(99)).
			WillReturnRows(sqlmock.NewRows([]string{"auth_flow_role_id", "auth_flow_role_uuid", "auth_flow_id", "role_id"}))
		repo := NewAuthFlowRoleRepository(gdb)
		result, err := repo.FindByAuthFlowID(99)
		require.NoError(t, err)
		assert.Empty(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "auth_flow_roles" WHERE .*auth_flow_id = \$1`).
			WithArgs(int64(1)).
			WillReturnError(errors.New("db error"))
		repo := NewAuthFlowRoleRepository(gdb)
		result, err := repo.FindByAuthFlowID(1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAuthFlowRoleRepository_FindByAuthFlowIDPaginated(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "auth_flow_roles" WHERE .*auth_flow_id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
		mock.ExpectQuery(`SELECT \* FROM "auth_flow_roles" WHERE .*auth_flow_id = \$1.*LIMIT`).
			WithArgs(int64(1), 10).
			WillReturnRows(sqlmock.NewRows([]string{"auth_flow_role_id", "auth_flow_role_uuid", "auth_flow_id", "role_id", "created_at"}).
				AddRow(1, uuid.New(), int64(1), int64(10), now).
				AddRow(2, uuid.New(), int64(1), int64(20), now))
		mock.ExpectQuery(`SELECT \* FROM "roles" WHERE "roles"\."role_id" IN \(\$1,\$2\)`).
			WithArgs(int64(10), int64(20)).
			WillReturnRows(sqlmock.NewRows([]string{"role_id", "role_uuid", "name", "created_at", "updated_at"}).
				AddRow(10, uuid.New(), "admin", now, now).
				AddRow(20, uuid.New(), "user", now, now))
		repo := NewAuthFlowRoleRepository(gdb)
		result, total, err := repo.FindByAuthFlowIDPaginated(1, 1, 10)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, int64(2), total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "auth_flow_roles" WHERE .*auth_flow_id = \$1`).
			WithArgs(int64(1)).
			WillReturnError(errors.New("db error"))
		repo := NewAuthFlowRoleRepository(gdb)
		result, total, err := repo.FindByAuthFlowIDPaginated(1, 1, 10)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, int64(0), total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "auth_flow_roles" WHERE .*auth_flow_id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
		mock.ExpectQuery(`SELECT \* FROM "auth_flow_roles" WHERE .*auth_flow_id = \$1.*LIMIT`).
			WithArgs(int64(1), 10).
			WillReturnError(errors.New("query error"))
		repo := NewAuthFlowRoleRepository(gdb)
		result, total, err := repo.FindByAuthFlowIDPaginated(1, 1, 10)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, int64(0), total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAuthFlowRoleRepository_DeleteByAuthFlowIDAndRoleID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "auth_flow_roles" WHERE .*auth_flow_id = \$1 AND role_id = \$2`).
			WithArgs(int64(1), int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		repo := NewAuthFlowRoleRepository(gdb)
		err := repo.DeleteByAuthFlowIDAndRoleID(1, 10)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "auth_flow_roles" WHERE .*auth_flow_id = \$1 AND role_id = \$2`).
			WithArgs(int64(1), int64(10)).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()
		repo := NewAuthFlowRoleRepository(gdb)
		err := repo.DeleteByAuthFlowIDAndRoleID(1, 10)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAuthFlowRoleRepository_FindByAuthFlowIDAndRoleID(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "auth_flow_roles" WHERE .*auth_flow_id = \$1 AND role_id = \$2`).
			WithArgs(int64(1), int64(10), 1).
			WillReturnRows(sqlmock.NewRows([]string{"auth_flow_role_id", "auth_flow_role_uuid", "auth_flow_id", "role_id", "created_at"}).
				AddRow(1, uuid.New(), int64(1), int64(10), now))
		repo := NewAuthFlowRoleRepository(gdb)
		result, err := repo.FindByAuthFlowIDAndRoleID(1, 10)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(1), result.AuthFlowID)
		assert.Equal(t, int64(10), result.RoleID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "auth_flow_roles" WHERE .*auth_flow_id = \$1 AND role_id = \$2`).
			WithArgs(int64(1), int64(99), 1).
			WillReturnRows(sqlmock.NewRows([]string{"auth_flow_role_id", "auth_flow_role_uuid", "auth_flow_id", "role_id"}))
		repo := NewAuthFlowRoleRepository(gdb)
		result, err := repo.FindByAuthFlowIDAndRoleID(1, 99)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "auth_flow_roles" WHERE .*auth_flow_id = \$1 AND role_id = \$2`).
			WithArgs(int64(1), int64(10), 1).
			WillReturnError(errors.New("db error"))
		repo := NewAuthFlowRoleRepository(gdb)
		result, err := repo.FindByAuthFlowIDAndRoleID(1, 10)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

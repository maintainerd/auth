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

func TestNewRegistrationFlowRoleRepository(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewRegistrationFlowRoleRepository(gdb)
	assert.NotNil(t, repo)
}

func TestRegistrationFlowRoleRepository_WithTx(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewRegistrationFlowRoleRepository(gdb)
	txRepo := repo.WithTx(gdb)
	assert.NotNil(t, txRepo)
}

func TestRegistrationFlowRoleRepository_FindByRegistrationFlowID(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "registration_flow_roles" WHERE .*registration_flow_id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_role_id", "registration_flow_role_uuid", "registration_flow_id", "role_id", "created_at"}).
				AddRow(1, uuid.New(), int64(1), int64(10), now).
				AddRow(2, uuid.New(), int64(1), int64(20), now))
		mock.ExpectQuery(`SELECT \* FROM "roles" WHERE "roles"\."role_id" IN \(\$1,\$2\)`).
			WithArgs(int64(10), int64(20)).
			WillReturnRows(sqlmock.NewRows([]string{"role_id", "role_uuid", "name", "created_at", "updated_at"}).
				AddRow(10, uuid.New(), "admin", now, now).
				AddRow(20, uuid.New(), "user", now, now))
		repo := NewRegistrationFlowRoleRepository(gdb)
		result, err := repo.FindByRegistrationFlowID(1)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty returns empty slice", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "registration_flow_roles" WHERE .*registration_flow_id = \$1`).
			WithArgs(int64(99)).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_role_id", "registration_flow_role_uuid", "registration_flow_id", "role_id"}))
		repo := NewRegistrationFlowRoleRepository(gdb)
		result, err := repo.FindByRegistrationFlowID(99)
		require.NoError(t, err)
		assert.Empty(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "registration_flow_roles" WHERE .*registration_flow_id = \$1`).
			WithArgs(int64(1)).
			WillReturnError(errors.New("db error"))
		repo := NewRegistrationFlowRoleRepository(gdb)
		result, err := repo.FindByRegistrationFlowID(1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRegistrationFlowRoleRepository_FindByRegistrationFlowIDPaginated(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "registration_flow_roles" WHERE .*registration_flow_id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
		mock.ExpectQuery(`SELECT \* FROM "registration_flow_roles" WHERE .*registration_flow_id = \$1.*LIMIT`).
			WithArgs(int64(1), 10).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_role_id", "registration_flow_role_uuid", "registration_flow_id", "role_id", "created_at"}).
				AddRow(1, uuid.New(), int64(1), int64(10), now).
				AddRow(2, uuid.New(), int64(1), int64(20), now))
		mock.ExpectQuery(`SELECT \* FROM "roles" WHERE "roles"\."role_id" IN \(\$1,\$2\)`).
			WithArgs(int64(10), int64(20)).
			WillReturnRows(sqlmock.NewRows([]string{"role_id", "role_uuid", "name", "created_at", "updated_at"}).
				AddRow(10, uuid.New(), "admin", now, now).
				AddRow(20, uuid.New(), "user", now, now))
		repo := NewRegistrationFlowRoleRepository(gdb)
		result, err := repo.FindByRegistrationFlowIDPaginated(1, 1, 10)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Data, 2)
		assert.Equal(t, int64(2), result.Total)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 10, result.Limit)
		assert.Equal(t, 1, result.TotalPages)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "registration_flow_roles" WHERE .*registration_flow_id = \$1`).
			WithArgs(int64(1)).
			WillReturnError(errors.New("db error"))
		repo := NewRegistrationFlowRoleRepository(gdb)
		result, err := repo.FindByRegistrationFlowIDPaginated(1, 1, 10)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "registration_flow_roles" WHERE .*registration_flow_id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
		mock.ExpectQuery(`SELECT \* FROM "registration_flow_roles" WHERE .*registration_flow_id = \$1.*LIMIT`).
			WithArgs(int64(1), 10).
			WillReturnError(errors.New("query error"))
		repo := NewRegistrationFlowRoleRepository(gdb)
		result, err := repo.FindByRegistrationFlowIDPaginated(1, 1, 10)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty page returns a zero-total result", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "registration_flow_roles" WHERE .*registration_flow_id = \$1`).
			WithArgs(int64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`SELECT \* FROM "registration_flow_roles" WHERE .*registration_flow_id = \$1.*LIMIT`).
			WithArgs(int64(7), 10).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_role_id", "registration_flow_role_uuid", "registration_flow_id", "role_id"}))
		repo := NewRegistrationFlowRoleRepository(gdb)
		result, err := repo.FindByRegistrationFlowIDPaginated(7, 1, 10)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Empty(t, result.Data)
		assert.Equal(t, int64(0), result.Total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// DeleteByRegistrationFlowID clears a flow's whole role membership. It exists
// because registration_flows is soft-deleted and a soft delete does not fire the
// FK cascade, so without this the child rows outlive the parent.
func TestRegistrationFlowRoleRepository_DeleteByRegistrationFlowID(t *testing.T) {
	t.Run("success deletes every row for the flow", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "registration_flow_roles" WHERE .*registration_flow_id = \$1`).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 3))
		mock.ExpectCommit()
		repo := NewRegistrationFlowRoleRepository(gdb)
		require.NoError(t, repo.DeleteByRegistrationFlowID(1))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no rows is not an error (idempotent)", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "registration_flow_roles" WHERE .*registration_flow_id = \$1`).
			WithArgs(int64(99)).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
		repo := NewRegistrationFlowRoleRepository(gdb)
		require.NoError(t, repo.DeleteByRegistrationFlowID(99))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "registration_flow_roles" WHERE .*registration_flow_id = \$1`).
			WithArgs(int64(1)).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()
		repo := NewRegistrationFlowRoleRepository(gdb)
		require.Error(t, repo.DeleteByRegistrationFlowID(1))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRegistrationFlowRoleRepository_DeleteByRegistrationFlowIDAndRoleID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "registration_flow_roles" WHERE .*registration_flow_id = \$1 AND role_id = \$2`).
			WithArgs(int64(1), int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		repo := NewRegistrationFlowRoleRepository(gdb)
		err := repo.DeleteByRegistrationFlowIDAndRoleID(1, 10)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "registration_flow_roles" WHERE .*registration_flow_id = \$1 AND role_id = \$2`).
			WithArgs(int64(1), int64(10)).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()
		repo := NewRegistrationFlowRoleRepository(gdb)
		err := repo.DeleteByRegistrationFlowIDAndRoleID(1, 10)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRegistrationFlowRoleRepository_FindByRegistrationFlowIDAndRoleID(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "registration_flow_roles" WHERE .*registration_flow_id = \$1 AND role_id = \$2`).
			WithArgs(int64(1), int64(10), 1).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_role_id", "registration_flow_role_uuid", "registration_flow_id", "role_id", "created_at"}).
				AddRow(1, uuid.New(), int64(1), int64(10), now))
		repo := NewRegistrationFlowRoleRepository(gdb)
		result, err := repo.FindByRegistrationFlowIDAndRoleID(1, 10)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(1), result.RegistrationFlowID)
		assert.Equal(t, int64(10), result.RoleID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "registration_flow_roles" WHERE .*registration_flow_id = \$1 AND role_id = \$2`).
			WithArgs(int64(1), int64(99), 1).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_role_id", "registration_flow_role_uuid", "registration_flow_id", "role_id"}))
		repo := NewRegistrationFlowRoleRepository(gdb)
		result, err := repo.FindByRegistrationFlowIDAndRoleID(1, 99)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "registration_flow_roles" WHERE .*registration_flow_id = \$1 AND role_id = \$2`).
			WithArgs(int64(1), int64(10), 1).
			WillReturnError(errors.New("db error"))
		repo := NewRegistrationFlowRoleRepository(gdb)
		result, err := repo.FindByRegistrationFlowIDAndRoleID(1, 10)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

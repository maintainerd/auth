package user

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserRoleRepository(t *testing.T) {
	db, _ := newMockGormDB(t)
	repo := NewUserRoleRepository(db)
	require.NotNil(t, repo)
}

func TestUserRoleRepository_WithTx(t *testing.T) {
	db, _ := newMockGormDB(t)
	repo := NewUserRoleRepository(db)
	txDB, _ := newMockGormDB(t)
	txRepo := repo.WithTx(txDB)
	require.NotNil(t, txRepo)
	assert.NotSame(t, repo, txRepo)
}

func TestUserRoleRepository_FindByUserID(t *testing.T) {
	t.Run("success with results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRoleRepository(db)

		rows := sqlmock.NewRows([]string{"user_role_id", "user_role_uuid", "user_id", "role_id"}).
			AddRow(1, testResourceUUID, 42, 10).
			AddRow(2, testUserUUID, 42, 20)
		mock.ExpectQuery(`SELECT \* FROM "user_roles" WHERE user_id = \$1`).
			WithArgs(int64(42)).
			WillReturnRows(rows)

		result, err := repo.FindByUserID(42)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRoleRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_roles" WHERE user_id = \$1`).
			WithArgs(int64(99)).
			WillReturnRows(sqlmock.NewRows([]string{"user_role_id", "user_role_uuid", "user_id", "role_id"}))

		result, err := repo.FindByUserID(99)
		require.NoError(t, err)
		assert.Len(t, result, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRoleRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_roles" WHERE user_id = \$1`).
			WithArgs(int64(42)).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindByUserID(42)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRoleRepository_FindByUserIDAndRoleID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRoleRepository(db)

		rows := sqlmock.NewRows([]string{"user_role_id", "user_role_uuid", "user_id", "role_id"}).
			AddRow(1, testResourceUUID, 42, 10)
		mock.ExpectQuery(`SELECT \* FROM "user_roles" WHERE user_id = \$1 AND role_id = \$2 ORDER BY "user_roles"\."user_role_id" LIMIT \$3`).
			WithArgs(int64(42), int64(10), 1).
			WillReturnRows(rows)

		result, err := repo.FindByUserIDAndRoleID(42, 10)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(10), result.RoleID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRoleRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_roles" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"user_role_id", "user_role_uuid", "user_id", "role_id"}))

		result, err := repo.FindByUserIDAndRoleID(99, 10)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRoleRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_roles" WHERE`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindByUserIDAndRoleID(42, 10)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRoleRepository_FindDefaultRolesByUserID(t *testing.T) {
	t.Run("success with results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRoleRepository(db)

		rows := sqlmock.NewRows([]string{"user_role_id", "user_role_uuid", "user_id", "role_id"}).
			AddRow(1, testResourceUUID, 42, 10)
		mock.ExpectQuery(`SELECT \* FROM "user_roles" WHERE user_id = \$1 AND is_default = true`).
			WithArgs(int64(42)).
			WillReturnRows(rows)

		result, err := repo.FindDefaultRolesByUserID(42)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRoleRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_roles" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"user_role_id", "user_role_uuid", "user_id", "role_id"}))

		result, err := repo.FindDefaultRolesByUserID(99)
		require.NoError(t, err)
		assert.Len(t, result, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRoleRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_roles" WHERE`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindDefaultRolesByUserID(42)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRoleRepository_DeleteByUserID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRoleRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "user_roles" WHERE user_id = \$1`).
			WithArgs(int64(42)).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		err := repo.DeleteByUserID(42)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRoleRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "user_roles" WHERE user_id = \$1`).
			WithArgs(int64(42)).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()

		err := repo.DeleteByUserID(42)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRoleRepository_DeleteByUserIDAndRoleID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRoleRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "user_roles" WHERE user_id = \$1 AND role_id = \$2`).
			WithArgs(int64(42), int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.DeleteByUserIDAndRoleID(42, 10)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserRoleRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "user_roles" WHERE`).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()

		err := repo.DeleteByUserIDAndRoleID(42, 10)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

package user

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserPasswordHistoryRepository(t *testing.T) {
	db, _ := newMockGormDB(t)
	repo := NewUserPasswordHistoryRepository(db)
	require.NotNil(t, repo)
}

func TestUserPasswordHistoryRepository_WithTx(t *testing.T) {
	db, _ := newMockGormDB(t)
	repo := NewUserPasswordHistoryRepository(db)
	txDB, _ := newMockGormDB(t)
	txRepo := repo.WithTx(txDB)
	require.NotNil(t, txRepo)
	assert.NotSame(t, repo, txRepo)
}

func TestUserPasswordHistoryRepository_AddEntry(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserPasswordHistoryRepository(db)

		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO "user_password_history" \("history_uuid","user_id","password_hash","created_at"\) VALUES \(\$1,\$2,\$3,\$4\) RETURNING "history_id"`).
			WithArgs(sqlmock.AnyArg(), int64(42), "hashed-password", sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"history_id"}).AddRow(1))
		mock.ExpectCommit()

		err := repo.AddEntry(42, "hashed-password")
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserPasswordHistoryRepository(db)

		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO "user_password_history"`).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()

		err := repo.AddEntry(42, "hashed-password")
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserPasswordHistoryRepository_FindRecentHashes(t *testing.T) {
	t.Run("success with results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserPasswordHistoryRepository(db)

		rows := sqlmock.NewRows([]string{"history_id", "user_id", "password_hash"}).
			AddRow(1, 42, "hash-1").
			AddRow(2, 42, "hash-2")
		mock.ExpectQuery(`SELECT \* FROM "user_password_history" WHERE user_id = \$1 ORDER BY created_at DESC LIMIT \$2`).
			WithArgs(int64(42), 3).
			WillReturnRows(rows)

		result, err := repo.FindRecentHashes(42, 3)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "hash-1", result[0])
		assert.Equal(t, "hash-2", result[1])
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserPasswordHistoryRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_password_history" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"history_id", "user_id", "password_hash"}))

		result, err := repo.FindRecentHashes(99, 3)
		require.NoError(t, err)
		assert.Len(t, result, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserPasswordHistoryRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_password_history" WHERE`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindRecentHashes(42, 3)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserPasswordHistoryRepository_PruneExcess(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserPasswordHistoryRepository(db)

		mock.ExpectExec(`DELETE FROM user_password_history`).
			WillReturnResult(sqlmock.NewResult(0, 2))

		err := repo.PruneExcess(42, 5)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserPasswordHistoryRepository(db)

		mock.ExpectExec(`DELETE FROM user_password_history`).
			WillReturnError(errors.New("db error"))

		err := repo.PruneExcess(42, 5)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

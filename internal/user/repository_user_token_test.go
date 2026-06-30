package user

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserTokenRepository(t *testing.T) {
	db, _ := newMockGormDB(t)
	repo := NewUserTokenRepository(db)
	require.NotNil(t, repo)
}

func TestUserTokenRepository_WithTx(t *testing.T) {
	db, _ := newMockGormDB(t)
	repo := NewUserTokenRepository(db)
	txDB, _ := newMockGormDB(t)
	txRepo := repo.WithTx(txDB)
	require.NotNil(t, txRepo)
	assert.NotSame(t, repo, txRepo)
}

func TestUserTokenRepository_FindByUserID(t *testing.T) {
	t.Run("success with results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		rows := sqlmock.NewRows([]string{"user_token_id", "user_token_uuid", "user_id", "token_type"}).
			AddRow(1, testResourceUUID, 42, shared.TokenTypeSession).
			AddRow(2, testUserUUID, 42, shared.TokenTypeEmailVerification)
		mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE user_id = \$1`).
			WithArgs(int64(42)).
			WillReturnRows(rows)

		result, err := repo.FindByUserID(42)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE user_id = \$1`).
			WithArgs(int64(99)).
			WillReturnRows(sqlmock.NewRows([]string{"user_token_id"}))

		result, err := repo.FindByUserID(99)
		require.NoError(t, err)
		assert.Len(t, result, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE user_id = \$1`).
			WithArgs(int64(42)).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindByUserID(42)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserTokenRepository_FindActiveTokensByUserID(t *testing.T) {
	t.Run("success with results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		rows := sqlmock.NewRows([]string{"user_token_id", "user_token_uuid", "user_id", "is_revoked"}).
			AddRow(1, testResourceUUID, 42, false)
		mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE user_id = \$1 AND is_revoked = false AND \(expires_at IS NULL OR expires_at > \$2\)`).
			WithArgs(int64(42), sqlmock.AnyArg()).
			WillReturnRows(rows)

		result, err := repo.FindActiveTokensByUserID(42)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"user_token_id"}))

		result, err := repo.FindActiveTokensByUserID(99)
		require.NoError(t, err)
		assert.Len(t, result, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserTokenRepository_FindByUserIDAndTokenType(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		rows := sqlmock.NewRows([]string{"user_token_id", "user_token_uuid", "user_id", "token_type"}).
			AddRow(1, testResourceUUID, 42, shared.TokenTypeEmailVerification)
		mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE user_id = \$1 AND token_type = \$2`).
			WithArgs(int64(42), shared.TokenTypeEmailVerification).
			WillReturnRows(rows)

		result, err := repo.FindByUserIDAndTokenType(42, shared.TokenTypeEmailVerification)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"user_token_id"}))

		result, err := repo.FindByUserIDAndTokenType(99, "unknown")
		require.NoError(t, err)
		assert.Len(t, result, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserTokenRepository_RevokeByUUID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)
		tokenUUID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "user_tokens" SET "is_revoked"=\$1,"updated_at"=\$2 WHERE user_token_uuid = \$3`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.RevokeByUUID(tokenUUID)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "user_tokens" SET .+ WHERE user_token_uuid = \$[0-9]+`).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()

		err := repo.RevokeByUUID(uuid.New())
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserTokenRepository_RevokeAllByUserID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "user_tokens" SET "is_revoked"=\$1,"updated_at"=\$2 WHERE user_id = \$3`).
			WillReturnResult(sqlmock.NewResult(0, 3))
		mock.ExpectCommit()

		err := repo.RevokeAllByUserID(42)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "user_tokens" SET .+ WHERE user_id = \$[0-9]+`).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()

		err := repo.RevokeAllByUserID(42)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserTokenRepository_DeleteByUserID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "user_tokens" WHERE user_id = \$1`).
			WithArgs(int64(42)).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		err := repo.DeleteByUserID(42)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "user_tokens" WHERE user_id = \$1`).
			WithArgs(int64(42)).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()

		err := repo.DeleteByUserID(42)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserTokenRepository_DeleteExpiredTokens(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)
		before := time.Now()

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "user_tokens".*`).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "user_tokens".*`).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		err := repo.DeleteExpiredTokens(before)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "user_tokens".*`).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()

		err := repo.DeleteExpiredTokens(time.Now())
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserTokenRepository_FindActiveSessions(t *testing.T) {
	t.Run("success with results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		rows := sqlmock.NewRows([]string{"user_token_id", "user_token_uuid", "user_id", "token_type"}).
			AddRow(1, testResourceUUID, 42, shared.TokenTypeSession)
		mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE user_id = \$1 AND token_type = \$2 AND is_revoked = false AND \(absolute_expires_at IS NULL OR absolute_expires_at > \$3\) ORDER BY created_at ASC`).
			WithArgs(int64(42), shared.TokenTypeSession, sqlmock.AnyArg()).
			WillReturnRows(rows)

		result, err := repo.FindActiveSessions(42)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"user_token_id"}))

		result, err := repo.FindActiveSessions(99)
		require.NoError(t, err)
		assert.Len(t, result, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserTokenRepository_FindActiveSessionByUUID(t *testing.T) {
	sessionUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		rows := sqlmock.NewRows([]string{"user_token_id", "user_token_uuid", "user_id", "token_type"}).
			AddRow(1, sessionUUID, 42, shared.TokenTypeSession)
		mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE user_id = \$1 AND user_token_uuid = \$2 AND token_type = \$3 AND is_revoked = false AND \(absolute_expires_at IS NULL OR absolute_expires_at > \$4\) ORDER BY "user_tokens"\."user_token_id" LIMIT \$5`).
			WithArgs(int64(42), sessionUUID, shared.TokenTypeSession, sqlmock.AnyArg(), 1).
			WillReturnRows(rows)

		result, err := repo.FindActiveSessionByUUID(42, sessionUUID)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, sessionUUID, result.UserTokenUUID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"user_token_id"}))

		result, err := repo.FindActiveSessionByUUID(99, uuid.New())
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_tokens" WHERE`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindActiveSessionByUUID(42, uuid.New())
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserTokenRepository_CountActiveSessions(t *testing.T) {
	t.Run("success with count", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "user_tokens" WHERE user_id = \$1 AND token_type = \$2 AND is_revoked = false AND \(absolute_expires_at IS NULL OR absolute_expires_at > \$3\)`).
			WithArgs(int64(42), shared.TokenTypeSession, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		result, err := repo.CountActiveSessions(42)
		require.NoError(t, err)
		assert.Equal(t, int64(5), result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("zero count", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "user_tokens" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		result, err := repo.CountActiveSessions(99)
		require.NoError(t, err)
		assert.Equal(t, int64(0), result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserTokenRepository_TouchSession(t *testing.T) {
	sessionUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)
		now := time.Now()
		userID := int64(1)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "user_tokens" SET "last_used_at"=\$1,"updated_at"=\$2 WHERE user_token_uuid = \$3 AND user_id = \$4 AND token_type = \$5 AND is_revoked = false`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.TouchSession(userID, sessionUUID, now)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "user_tokens" SET .+ WHERE user_token_uuid = \$[0-9]+`).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()

		err := repo.TouchSession(int64(1), uuid.New(), time.Now())
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserTokenRepository_RevokeSessionByUUID(t *testing.T) {
	sessionUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "user_tokens" SET "is_revoked"=\$1,"updated_at"=\$2 WHERE user_id = \$3 AND user_token_uuid = \$4 AND token_type = \$5`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.RevokeSessionByUUID(42, sessionUUID)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "user_tokens" SET .+ WHERE user_id = \$[0-9]+`).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()

		err := repo.RevokeSessionByUUID(42, uuid.New())
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserTokenRepository_RevokeAllSessionsByUserID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "user_tokens" SET "is_revoked"=\$1,"updated_at"=\$2 WHERE user_id = \$3 AND token_type = \$4`).
			WillReturnResult(sqlmock.NewResult(0, 3))
		mock.ExpectCommit()

		err := repo.RevokeAllSessionsByUserID(42)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserTokenRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "user_tokens" SET .+ WHERE user_id = \$[0-9]+`).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()

		err := repo.RevokeAllSessionsByUserID(42)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

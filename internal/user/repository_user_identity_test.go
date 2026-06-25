package user

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserIdentityRepository(t *testing.T) {
	db, _ := newMockGormDB(t)
	repo := NewUserIdentityRepository(db)
	require.NotNil(t, repo)
}

func TestUserIdentityRepository_WithTx(t *testing.T) {
	db, _ := newMockGormDB(t)
	repo := NewUserIdentityRepository(db)
	txDB, _ := newMockGormDB(t)
	txRepo := repo.WithTx(txDB)
	require.NotNil(t, txRepo)
	assert.NotSame(t, repo, txRepo)
}

func TestUserIdentityRepository_FindByUserID(t *testing.T) {
	t.Run("success with results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserIdentityRepository(db)

		rows := sqlmock.NewRows([]string{"user_identity_id", "user_identity_uuid", "user_id", "provider", "sub"}).
			AddRow(1, testResourceUUID, 42, "google", "sub-1").
			AddRow(2, testUserUUID, 42, "github", "sub-2")
		mock.ExpectQuery(`SELECT \* FROM "user_identities" WHERE user_id = \$1`).
			WithArgs(int64(42)).
			WillReturnRows(rows)

		result, err := repo.FindByUserID(42)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserIdentityRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_identities" WHERE user_id = \$1`).
			WithArgs(int64(99)).
			WillReturnRows(sqlmock.NewRows([]string{"user_identity_id", "user_identity_uuid", "user_id"}))

		result, err := repo.FindByUserID(99)
		require.NoError(t, err)
		assert.Len(t, result, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserIdentityRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_identities" WHERE user_id = \$1`).
			WithArgs(int64(42)).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindByUserID(42)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserIdentityRepository_FindUserIdentitiesPaginated(t *testing.T) {
	t.Run("success with results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserIdentityRepository(db)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "user_identities" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
		mock.ExpectQuery(`SELECT \* FROM "user_identities" WHERE .+ LIMIT \$[0-9]+`).
			WillReturnRows(sqlmock.NewRows([]string{"user_identity_id", "user_identity_uuid", "user_id", "provider"}).
				AddRow(1, testResourceUUID, 42, "google").
				AddRow(2, testUserUUID, 42, "github"))

		result, err := repo.FindUserIdentitiesPaginated(GetUserIdentitiesFilter{
			UserID: 42, Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), result.Total)
		assert.Len(t, result.Data, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserIdentityRepository(db)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "user_identities" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`SELECT \* FROM "user_identities" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"user_identity_id", "user_identity_uuid", "user_id"}))

		result, err := repo.FindUserIdentitiesPaginated(GetUserIdentitiesFilter{
			UserID: 42, Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), result.Total)
		assert.Len(t, result.Data, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserIdentityRepository(db)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "user_identities" WHERE`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindUserIdentitiesPaginated(GetUserIdentitiesFilter{
			UserID: 42, Page: 1, Limit: 10, SortBy: "created_at", SortOrder: SortOrderDesc,
		})
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserIdentityRepository_FindByUserIDAndClientID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserIdentityRepository(db)

		rows := sqlmock.NewRows([]string{"user_identity_id", "user_identity_uuid", "user_id", "client_id"}).
			AddRow(1, testResourceUUID, 42, 5)
		mock.ExpectQuery(`SELECT \* FROM "user_identities" WHERE user_id = \$1 AND client_id = \$2 ORDER BY "user_identities"\."user_identity_id" LIMIT \$3`).
			WithArgs(int64(42), int64(5), 1).
			WillReturnRows(rows)

		result, err := repo.FindByUserIDAndClientID(42, 5)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(5), result.ClientID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserIdentityRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_identities" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"user_identity_id", "user_identity_uuid", "user_id", "client_id"}))

		result, err := repo.FindByUserIDAndClientID(99, 5)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserIdentityRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_identities" WHERE`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindByUserIDAndClientID(42, 5)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserIdentityRepository_FindByUserIDAndProvider(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserIdentityRepository(db)

		rows := sqlmock.NewRows([]string{"user_identity_id", "user_identity_uuid", "user_id", "provider"}).
			AddRow(1, testResourceUUID, 42, "google")
		mock.ExpectQuery(`SELECT \* FROM "user_identities" WHERE user_id = \$1 AND provider = \$2 ORDER BY "user_identities"\."user_identity_id" LIMIT \$3`).
			WithArgs(int64(42), "google", 1).
			WillReturnRows(rows)

		result, err := repo.FindByUserIDAndProvider(42, "google")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "google", result.Provider)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserIdentityRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_identities" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"user_identity_id", "user_identity_uuid", "user_id", "provider"}))

		result, err := repo.FindByUserIDAndProvider(99, "google")
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserIdentityRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_identities" WHERE`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindByUserIDAndProvider(42, "google")
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserIdentityRepository_FindByIdentityProviderID(t *testing.T) {
	t.Run("success with results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserIdentityRepository(db)

		rows := sqlmock.NewRows([]string{"user_identity_id", "user_identity_uuid", "identity_provider_id"}).
			AddRow(1, testResourceUUID, 3).
			AddRow(2, testUserUUID, 3)
		mock.ExpectQuery(`SELECT \* FROM "user_identities" WHERE identity_provider_id = \$1`).
			WithArgs(int64(3)).
			WillReturnRows(rows)

		result, err := repo.FindByIdentityProviderID(3)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserIdentityRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_identities" WHERE identity_provider_id = \$1`).
			WithArgs(int64(99)).
			WillReturnRows(sqlmock.NewRows([]string{"user_identity_id", "user_identity_uuid", "identity_provider_id"}))

		result, err := repo.FindByIdentityProviderID(99)
		require.NoError(t, err)
		assert.Len(t, result, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserIdentityRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_identities" WHERE identity_provider_id = \$1`).
			WithArgs(int64(3)).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindByIdentityProviderID(3)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserIdentityRepository_DeleteByUserID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserIdentityRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "user_identities" WHERE user_id = \$1`).
			WithArgs(int64(42)).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		err := repo.DeleteByUserID(42)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserIdentityRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "user_identities" WHERE user_id = \$1`).
			WithArgs(int64(42)).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()

		err := repo.DeleteByUserID(42)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

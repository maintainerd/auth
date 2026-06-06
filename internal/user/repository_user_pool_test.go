package user

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserPoolRepository(t *testing.T) {
	db, _ := newMockGormDB(t)
	repo := NewUserPoolRepository(db)
	require.NotNil(t, repo)
}

func TestUserPoolRepository_WithTx(t *testing.T) {
	db, _ := newMockGormDB(t)
	repo := NewUserPoolRepository(db)
	txDB, _ := newMockGormDB(t)
	txRepo := repo.WithTx(txDB)
	require.NotNil(t, txRepo)
	assert.NotSame(t, repo, txRepo)
}

func TestUserPoolRepository_FindByIdentifier(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserPoolRepository(db)

		rows := sqlmock.NewRows([]string{"user_pool_id", "user_pool_uuid", "tenant_id", "identifier", "name"}).
			AddRow(1, testResourceUUID, 1, "my-pool", "My Pool")
		mock.ExpectQuery(`SELECT \* FROM "user_pools" WHERE \(tenant_id = \$1 AND identifier = \$2 AND deleted_at IS NULL\) AND "user_pools"\."deleted_at" IS NULL ORDER BY "user_pools"\."user_pool_id" LIMIT \$3`).
			WithArgs(int64(1), "my-pool", 1).
			WillReturnRows(rows)

		result, err := repo.FindByIdentifier(1, "my-pool")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "my-pool", result.Identifier)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserPoolRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_pools" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"user_pool_id"}))

		result, err := repo.FindByIdentifier(1, "missing")
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserPoolRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_pools" WHERE`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindByIdentifier(1, "my-pool")
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserPoolRepository_FindSystem(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserPoolRepository(db)

		rows := sqlmock.NewRows([]string{"user_pool_id", "user_pool_uuid", "tenant_id", "is_system", "name"}).
			AddRow(2, testUserUUID, 1, true, "System Pool")
		mock.ExpectQuery(`SELECT \* FROM "user_pools" WHERE \(tenant_id = \$1 AND is_system = \$2 AND deleted_at IS NULL\) AND "user_pools"\."deleted_at" IS NULL ORDER BY "user_pools"\."user_pool_id" LIMIT \$3`).
			WithArgs(int64(1), true, 1).
			WillReturnRows(rows)

		result, err := repo.FindSystem(1)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.IsSystem)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserPoolRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_pools" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"user_pool_id"}))

		result, err := repo.FindSystem(1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserPoolRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_pools" WHERE`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindSystem(1)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserPoolRepository_FindAllByTenantID(t *testing.T) {
	t.Run("success with results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserPoolRepository(db)

		rows := sqlmock.NewRows([]string{"user_pool_id", "user_pool_uuid", "tenant_id", "name"}).
			AddRow(1, testResourceUUID, 1, "Pool A").
			AddRow(2, testUserUUID, 1, "Pool B")
		mock.ExpectQuery(`SELECT \* FROM "user_pools" WHERE \(tenant_id = \$1 AND deleted_at IS NULL\) AND "user_pools"\."deleted_at" IS NULL`).
			WithArgs(int64(1)).
			WillReturnRows(rows)

		result, err := repo.FindAllByTenantID(1)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty results", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserPoolRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_pools" WHERE`).
			WillReturnRows(sqlmock.NewRows([]string{"user_pool_id", "user_pool_uuid"}))

		result, err := repo.FindAllByTenantID(1)
		require.NoError(t, err)
		assert.Len(t, result, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := NewUserPoolRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "user_pools" WHERE`).
			WillReturnError(errors.New("db error"))

		_, err := repo.FindAllByTenantID(1)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

package client

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyRepository_FindByUUIDAndTenantID(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "api_keys" WHERE.*api_key_uuid = \$1.*tenant_id = \$2.*AND "api_keys"\."deleted_at" IS NULL`).
			WithArgs(id.String(), int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_id", "api_key_uuid", "tenant_id", "name", "key_hash", "key_prefix", "status", "created_at", "updated_at"}).
				AddRow(1, id, 1, "test-key", "hash123", "kp_abc", "active", now, now))
		result, err := NewAPIKeyRepository(gdb).FindByUUIDAndTenantID(id.String(), 1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-key", result.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "api_keys" WHERE.*api_key_uuid = \$1.*tenant_id = \$2.*AND "api_keys"\."deleted_at" IS NULL`).
			WithArgs(id.String(), int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_id", "api_key_uuid", "tenant_id"}))
		result, err := NewAPIKeyRepository(gdb).FindByUUIDAndTenantID(id.String(), 1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "api_keys" WHERE.*api_key_uuid = \$1.*tenant_id = \$2`).
			WithArgs(id.String(), int64(1), 1).
			WillReturnError(assert.AnError)
		result, err := NewAPIKeyRepository(gdb).FindByUUIDAndTenantID(id.String(), 1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAPIKeyRepository_FindByKeyPrefix(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "api_keys" WHERE.*key_prefix = \$1.*AND "api_keys"\."deleted_at" IS NULL`).
			WithArgs("kp_abc123", 1).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_id", "api_key_uuid", "tenant_id", "name", "key_hash", "key_prefix", "status", "created_at", "updated_at"}).
				AddRow(1, id, 1, "test-key", "hash123", "kp_abc123", "active", now, now))
		result, err := NewAPIKeyRepository(gdb).FindByKeyPrefix("kp_abc123")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "kp_abc123", result.KeyPrefix)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "api_keys" WHERE.*key_prefix = \$1.*AND "api_keys"\."deleted_at" IS NULL`).
			WithArgs("kp_xyz", 1).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_id", "api_key_uuid", "tenant_id"}))
		result, err := NewAPIKeyRepository(gdb).FindByKeyPrefix("kp_xyz")
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "api_keys" WHERE.*key_prefix = \$1`).
			WithArgs("kp_abc123", 1).
			WillReturnError(assert.AnError)
		result, err := NewAPIKeyRepository(gdb).FindByKeyPrefix("kp_abc123")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNewAPIKeyRepository(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewAPIKeyRepository(gdb)
	assert.NotNil(t, repo)
}

func TestAPIKeyRepository_WithTx(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewAPIKeyRepository(gdb)
	txRepo := repo.WithTx(gdb)
	assert.NotNil(t, txRepo)
}

func TestAPIKeyRepository_FindByKeyHash(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "api_keys" WHERE.*key_hash = \$1.*AND "api_keys"\."deleted_at" IS NULL`).
			WithArgs("hash123", 1).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_id", "api_key_uuid", "tenant_id", "name", "key_hash", "key_prefix", "status", "created_at", "updated_at"}).
				AddRow(1, id, 1, "test-key", "hash123", "kp_abc", "active", now, now))
		result, err := NewAPIKeyRepository(gdb).FindByKeyHash("hash123")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "hash123", result.KeyHash)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "api_keys" WHERE.*key_hash = \$1.*AND "api_keys"\."deleted_at" IS NULL`).
			WithArgs("nonexistent", 1).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_id", "api_key_uuid", "tenant_id"}))
		result, err := NewAPIKeyRepository(gdb).FindByKeyHash("nonexistent")
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "api_keys" WHERE.*key_hash = \$1`).
			WithArgs("hash123", 1).
			WillReturnError(assert.AnError)
		result, err := NewAPIKeyRepository(gdb).FindByKeyHash("hash123")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAPIKeyRepository_DeleteByUUIDAndTenantID(t *testing.T) {
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "api_keys" SET.*"deleted_at"=.*WHERE.*api_key_uuid = \$\d+.*tenant_id = \$\d+`).
			WithArgs(sqlmock.AnyArg(), id.String(), int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		err := NewAPIKeyRepository(gdb).DeleteByUUIDAndTenantID(id.String(), 1)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "api_keys" SET.*"deleted_at"=.*WHERE.*api_key_uuid = \$\d+.*tenant_id = \$\d+`).
			WithArgs(sqlmock.AnyArg(), id.String(), int64(1)).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()
		err := NewAPIKeyRepository(gdb).DeleteByUUIDAndTenantID(id.String(), 1)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAPIKeyRepository_FindPaginated(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "api_keys" WHERE.*tenant_id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "api_keys" WHERE.*tenant_id = \$1.*ORDER BY created_at DESC.*LIMIT \$\d+`).
			WithArgs(int64(1), 10).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_id", "api_key_uuid", "tenant_id", "name", "key_hash", "key_prefix", "status", "created_at", "updated_at"}).
				AddRow(1, id, 1, "test-key", "hash123", "kp_abc", "active", now, now))
		result, err := NewAPIKeyRepository(gdb).FindPaginated(APIKeyRepositoryGetFilter{
			TenantID:  1,
			Page:      1,
			Limit:     10,
			SortBy:    "created_at",
			SortOrder: "DESC",
		})
		require.NoError(t, err)
		assert.Len(t, result.Data, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "api_keys" WHERE.*tenant_id = \$1`).
			WithArgs(int64(1)).
			WillReturnError(assert.AnError)
		result, err := NewAPIKeyRepository(gdb).FindPaginated(APIKeyRepositoryGetFilter{
			TenantID:  1,
			Page:      1,
			Limit:     10,
			SortBy:    "created_at",
			SortOrder: "DESC",
		})
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

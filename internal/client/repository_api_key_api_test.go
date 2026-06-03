package client

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyAPIRepository_FindByAPIKeyUUID(t *testing.T) {
	now := time.Now()
	apiKeyUUID := uuid.New()
	apiUUID := uuid.New()
	apiKeyAPIUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "api_key_apis" JOIN api_keys ON api_keys\.api_key_id = api_key_apis\.api_key_id WHERE api_keys\.api_key_uuid = \$1`).
			WithArgs(apiKeyUUID).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_api_id", "api_key_api_uuid", "api_key_id", "api_id", "created_at"}).
				AddRow(1, apiKeyAPIUUID, 1, 1, now))

		mock.ExpectQuery(`SELECT \* FROM "apis" WHERE "apis"\."api_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"api_id", "api_uuid", "name", "created_at", "updated_at"}).
				AddRow(1, apiUUID, "test-api", now, now))

		mock.ExpectQuery(`SELECT \* FROM "api_key_permissions" WHERE "api_key_permissions"\."api_key_api_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_permission_id", "api_key_api_id", "permission_id", "created_at"}))

		result, err := NewAPIKeyAPIRepository(gdb).FindByAPIKeyUUID(apiKeyUUID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns empty", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "api_key_apis" JOIN api_keys ON api_keys\.api_key_id = api_key_apis\.api_key_id WHERE api_keys\.api_key_uuid = \$1`).
			WithArgs(apiKeyUUID).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_api_id", "api_key_api_uuid", "api_key_id", "api_id", "created_at"}))
		result, err := NewAPIKeyAPIRepository(gdb).FindByAPIKeyUUID(apiKeyUUID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "api_key_apis" JOIN api_keys ON api_keys\.api_key_id = api_key_apis\.api_key_id WHERE api_keys\.api_key_uuid = \$1`).
			WithArgs(apiKeyUUID).
			WillReturnError(assert.AnError)
		result, err := NewAPIKeyAPIRepository(gdb).FindByAPIKeyUUID(apiKeyUUID)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNewAPIKeyAPIRepository(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewAPIKeyAPIRepository(gdb)
	assert.NotNil(t, repo)
}

func TestAPIKeyAPIRepository_WithTx(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewAPIKeyAPIRepository(gdb)
	txRepo := repo.WithTx(gdb)
	assert.NotNil(t, txRepo)
}

func TestAPIKeyAPIRepository_FindByAPIKeyAndAPI(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "api_key_apis" WHERE.*api_key_id = \$1.*api_id = \$2`).
			WithArgs(int64(1), int64(2), 1).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_api_id", "api_key_api_uuid", "api_key_id", "api_id", "created_at"}).
				AddRow(1, id, 1, 2, now))
		result, err := NewAPIKeyAPIRepository(gdb).FindByAPIKeyAndAPI(1, 2)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(1), result.APIKeyID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "api_key_apis" WHERE.*api_key_id = \$1.*api_id = \$2`).
			WithArgs(int64(1), int64(2), 1).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_api_id", "api_key_api_uuid", "api_key_id"}))
		result, err := NewAPIKeyAPIRepository(gdb).FindByAPIKeyAndAPI(1, 2)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "api_key_apis" WHERE.*api_key_id = \$1.*api_id = \$2`).
			WithArgs(int64(1), int64(2), 1).
			WillReturnError(assert.AnError)
		result, err := NewAPIKeyAPIRepository(gdb).FindByAPIKeyAndAPI(1, 2)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAPIKeyAPIRepository_FindByAPIKeyUUIDPaginated(t *testing.T) {
	now := time.Now()
	apiKeyUUID := uuid.New()
	apiKeyAPIUUID := uuid.New()
	apiUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "api_key_apis" JOIN api_keys ON api_keys\.api_key_id = api_key_apis\.api_key_id WHERE.*api_keys\.api_key_uuid = \$1`).
			WithArgs(apiKeyUUID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT .* FROM "api_key_apis" JOIN api_keys ON api_keys\.api_key_id = api_key_apis\.api_key_id WHERE.*api_keys\.api_key_uuid = \$1.*ORDER BY.*created_at.*DESC.*LIMIT \$\d+`).
			WithArgs(apiKeyUUID, 10).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_api_id", "api_key_api_uuid", "api_key_id", "api_id", "created_at"}).
				AddRow(1, apiKeyAPIUUID, 1, 1, now))

		mock.ExpectQuery(`SELECT \* FROM "apis" WHERE "apis"\."api_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"api_id", "api_uuid", "name", "created_at", "updated_at"}).
				AddRow(1, apiUUID, "test-api", now, now))

		mock.ExpectQuery(`SELECT \* FROM "api_key_permissions" WHERE "api_key_permissions"\."api_key_api_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_permission_id", "api_key_api_id", "permission_id", "created_at"}))

		result, err := NewAPIKeyAPIRepository(gdb).FindByAPIKeyUUIDPaginated(apiKeyUUID, 1, 10, "created_at", "DESC")
		require.NoError(t, err)
		assert.Len(t, result.Data, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "api_key_apis" JOIN api_keys ON api_keys\.api_key_id = api_key_apis\.api_key_id WHERE.*api_keys\.api_key_uuid = \$1`).
			WithArgs(apiKeyUUID).
			WillReturnError(assert.AnError)
		result, err := NewAPIKeyAPIRepository(gdb).FindByAPIKeyUUIDPaginated(apiKeyUUID, 1, 10, "created_at", "DESC")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAPIKeyAPIRepository_FindByAPIKeyUUIDAndAPIUUID(t *testing.T) {
	now := time.Now()
	apiKeyUUID := uuid.New()
	apiUUID := uuid.New()
	apiKeyAPIUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "api_key_apis" JOIN api_keys ON api_keys\.api_key_id = api_key_apis\.api_key_id JOIN apis ON apis\.api_id = api_key_apis\.api_id WHERE.*api_keys\.api_key_uuid = \$1.*apis\.api_uuid = \$2`).
			WithArgs(apiKeyUUID, apiUUID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_api_id", "api_key_api_uuid", "api_key_id", "api_id", "created_at"}).
				AddRow(1, apiKeyAPIUUID, 1, 1, now))

		mock.ExpectQuery(`SELECT \* FROM "apis" WHERE "apis"\."api_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"api_id", "api_uuid", "name", "created_at", "updated_at"}).
				AddRow(1, apiUUID, "test-api", now, now))

		mock.ExpectQuery(`SELECT \* FROM "api_key_permissions" WHERE "api_key_permissions"\."api_key_api_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_permission_id", "api_key_api_id", "permission_id", "created_at"}))

		result, err := NewAPIKeyAPIRepository(gdb).FindByAPIKeyUUIDAndAPIUUID(apiKeyUUID, apiUUID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "api_key_apis" JOIN api_keys ON api_keys\.api_key_id = api_key_apis\.api_key_id JOIN apis ON apis\.api_id = api_key_apis\.api_id WHERE.*api_keys\.api_key_uuid = \$1.*apis\.api_uuid = \$2`).
			WithArgs(apiKeyUUID, apiUUID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_api_id", "api_key_api_uuid", "api_key_id"}))
		result, err := NewAPIKeyAPIRepository(gdb).FindByAPIKeyUUIDAndAPIUUID(apiKeyUUID, apiUUID)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "api_key_apis" JOIN api_keys ON api_keys\.api_key_id = api_key_apis\.api_key_id JOIN apis ON apis\.api_id = api_key_apis\.api_id WHERE.*api_keys\.api_key_uuid = \$1.*apis\.api_uuid = \$2`).
			WithArgs(apiKeyUUID, apiUUID, 1).
			WillReturnError(assert.AnError)
		result, err := NewAPIKeyAPIRepository(gdb).FindByAPIKeyUUIDAndAPIUUID(apiKeyUUID, apiUUID)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAPIKeyAPIRepository_RemoveByAPIKeyAndAPI(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "api_key_apis" WHERE.*api_key_id = \$1.*api_id = \$2`).
			WithArgs(int64(1), int64(2)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		err := NewAPIKeyAPIRepository(gdb).RemoveByAPIKeyAndAPI(1, 2)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "api_key_apis" WHERE.*api_key_id = \$1.*api_id = \$2`).
			WithArgs(int64(1), int64(2)).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()
		err := NewAPIKeyAPIRepository(gdb).RemoveByAPIKeyAndAPI(1, 2)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAPIKeyAPIRepository_RemoveByAPIKeyUUIDAndAPIUUID(t *testing.T) {
	apiKeyUUID := uuid.New()
	apiUUID := uuid.New()
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "api_key_apis" JOIN api_keys ON api_keys\.api_key_id = api_key_apis\.api_key_id JOIN apis ON apis\.api_id = api_key_apis\.api_id WHERE.*api_keys\.api_key_uuid = \$1.*apis\.api_uuid = \$2`).
			WithArgs(apiKeyUUID, apiUUID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_api_id", "api_key_api_uuid", "api_key_id", "api_id", "created_at"}).
				AddRow(1, uuid.New(), 1, 1, now))

		mock.ExpectQuery(`SELECT \* FROM "apis" WHERE "apis"\."api_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"api_id", "api_uuid", "name", "created_at", "updated_at"}).
				AddRow(1, apiUUID, "test-api", now, now))

		mock.ExpectQuery(`SELECT \* FROM "api_key_permissions" WHERE "api_key_permissions"\."api_key_api_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_permission_id", "api_key_api_id", "permission_id", "created_at"}))

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "api_key_apis" WHERE "api_key_apis"\."api_key_api_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := NewAPIKeyAPIRepository(gdb).RemoveByAPIKeyUUIDAndAPIUUID(apiKeyUUID, apiUUID)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error on find", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "api_key_apis" JOIN api_keys ON api_keys\.api_key_id = api_key_apis\.api_key_id JOIN apis ON apis\.api_id = api_key_apis\.api_id WHERE.*api_keys\.api_key_uuid = \$1.*apis\.api_uuid = \$2`).
			WithArgs(apiKeyUUID, apiUUID, 1).
			WillReturnError(assert.AnError)
		err := NewAPIKeyAPIRepository(gdb).RemoveByAPIKeyUUIDAndAPIUUID(apiKeyUUID, apiUUID)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "api_key_apis" JOIN api_keys ON api_keys\.api_key_id = api_key_apis\.api_key_id JOIN apis ON apis\.api_id = api_key_apis\.api_id WHERE.*api_keys\.api_key_uuid = \$1.*apis\.api_uuid = \$2`).
			WithArgs(apiKeyUUID, apiUUID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"api_key_api_id", "api_key_api_uuid", "api_key_id"}))
		err := NewAPIKeyAPIRepository(gdb).RemoveByAPIKeyUUIDAndAPIUUID(apiKeyUUID, apiUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API key API relationship not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

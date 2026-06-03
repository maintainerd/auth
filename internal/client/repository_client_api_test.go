package client

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientAPIRepository_FindByClientUUID(t *testing.T) {
	now := time.Now()
	clientUUID := uuid.New()
	apiUUID := uuid.New()
	clientAPIUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "client_apis" JOIN clients ON clients\.client_id = client_apis\.client_id WHERE clients\.client_uuid = \$1`).
			WithArgs(clientUUID).
			WillReturnRows(sqlmock.NewRows([]string{"client_api_id", "client_api_uuid", "client_id", "api_id", "created_at"}).
				AddRow(1, clientAPIUUID, 1, 1, now))

		mock.ExpectQuery(`SELECT \* FROM "apis" WHERE "apis"\."api_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"api_id", "api_uuid", "name", "created_at", "updated_at"}).
				AddRow(1, apiUUID, "test-api", now, now))

		mock.ExpectQuery(`SELECT \* FROM "client_permissions" WHERE "client_permissions"\."client_api_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"client_permission_id", "client_api_id", "permission_id", "created_at"}))

		result, err := NewClientAPIRepository(gdb).FindByClientUUID(clientUUID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns empty", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "client_apis" JOIN clients ON clients\.client_id = client_apis\.client_id WHERE clients\.client_uuid = \$1`).
			WithArgs(clientUUID).
			WillReturnRows(sqlmock.NewRows([]string{"client_api_id", "client_api_uuid", "client_id", "api_id", "created_at"}))
		result, err := NewClientAPIRepository(gdb).FindByClientUUID(clientUUID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "client_apis" JOIN clients ON clients\.client_id = client_apis\.client_id WHERE clients\.client_uuid = \$1`).
			WithArgs(clientUUID).
			WillReturnError(assert.AnError)
		result, err := NewClientAPIRepository(gdb).FindByClientUUID(clientUUID)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNewClientAPIRepository(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewClientAPIRepository(gdb)
	assert.NotNil(t, repo)
}

func TestClientAPIRepository_WithTx(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewClientAPIRepository(gdb)
	txRepo := repo.WithTx(gdb)
	assert.NotNil(t, txRepo)
}

func TestClientAPIRepository_FindByClientAndAPI(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "client_apis" WHERE.*client_id = \$1.*api_id = \$2`).
			WithArgs(int64(1), int64(2), 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_api_id", "client_api_uuid", "client_id", "api_id", "created_at"}).
				AddRow(1, id, 1, 2, now))
		result, err := NewClientAPIRepository(gdb).FindByClientAndAPI(1, 2)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(1), result.ClientID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "client_apis" WHERE.*client_id = \$1.*api_id = \$2`).
			WithArgs(int64(1), int64(2), 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_api_id", "client_api_uuid", "client_id"}))
		result, err := NewClientAPIRepository(gdb).FindByClientAndAPI(1, 2)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "client_apis" WHERE.*client_id = \$1.*api_id = \$2`).
			WithArgs(int64(1), int64(2), 1).
			WillReturnError(assert.AnError)
		result, err := NewClientAPIRepository(gdb).FindByClientAndAPI(1, 2)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientAPIRepository_FindByClientUUIDAndAPIUUID(t *testing.T) {
	now := time.Now()
	clientUUID := uuid.New()
	apiUUID := uuid.New()
	clientAPIUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "client_apis" JOIN clients ON clients\.client_id = client_apis\.client_id JOIN apis ON apis\.api_id = client_apis\.api_id WHERE.*clients\.client_uuid = \$1.*apis\.api_uuid = \$2`).
			WithArgs(clientUUID, apiUUID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_api_id", "client_api_uuid", "client_id", "api_id", "created_at"}).
				AddRow(1, clientAPIUUID, 1, 1, now))

		mock.ExpectQuery(`SELECT \* FROM "apis" WHERE "apis"\."api_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"api_id", "api_uuid", "name", "created_at", "updated_at"}).
				AddRow(1, apiUUID, "test-api", now, now))

		mock.ExpectQuery(`SELECT \* FROM "client_permissions" WHERE "client_permissions"\."client_api_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"client_permission_id", "client_api_id", "permission_id", "created_at"}))

		result, err := NewClientAPIRepository(gdb).FindByClientUUIDAndAPIUUID(clientUUID, apiUUID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "client_apis" JOIN clients ON clients\.client_id = client_apis\.client_id JOIN apis ON apis\.api_id = client_apis\.api_id WHERE.*clients\.client_uuid = \$1.*apis\.api_uuid = \$2`).
			WithArgs(clientUUID, apiUUID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_api_id", "client_api_uuid", "client_id"}))
		result, err := NewClientAPIRepository(gdb).FindByClientUUIDAndAPIUUID(clientUUID, apiUUID)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "client_apis" JOIN clients ON clients\.client_id = client_apis\.client_id JOIN apis ON apis\.api_id = client_apis\.api_id WHERE.*clients\.client_uuid = \$1.*apis\.api_uuid = \$2`).
			WithArgs(clientUUID, apiUUID, 1).
			WillReturnError(assert.AnError)
		result, err := NewClientAPIRepository(gdb).FindByClientUUIDAndAPIUUID(clientUUID, apiUUID)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientAPIRepository_RemoveByClientAndAPI(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "client_apis" WHERE.*client_id = \$1.*api_id = \$2`).
			WithArgs(int64(1), int64(2)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		err := NewClientAPIRepository(gdb).RemoveByClientAndAPI(1, 2)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "client_apis" WHERE.*client_id = \$1.*api_id = \$2`).
			WithArgs(int64(1), int64(2)).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()
		err := NewClientAPIRepository(gdb).RemoveByClientAndAPI(1, 2)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientAPIRepository_RemoveByClientUUIDAndAPIUUID(t *testing.T) {
	clientUUID := uuid.New()
	apiUUID := uuid.New()
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "client_apis" JOIN clients ON clients\.client_id = client_apis\.client_id JOIN apis ON apis\.api_id = client_apis\.api_id WHERE.*clients\.client_uuid = \$1.*apis\.api_uuid = \$2`).
			WithArgs(clientUUID, apiUUID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_api_id", "client_api_uuid", "client_id", "api_id", "created_at"}).
				AddRow(1, uuid.New(), 1, 1, now))

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "client_apis" WHERE "client_apis"\."client_api_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := NewClientAPIRepository(gdb).RemoveByClientUUIDAndAPIUUID(clientUUID, apiUUID)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "client_apis" JOIN clients ON clients\.client_id = client_apis\.client_id JOIN apis ON apis\.api_id = client_apis\.api_id WHERE.*clients\.client_uuid = \$1.*apis\.api_uuid = \$2`).
			WithArgs(clientUUID, apiUUID, 1).
			WillReturnError(assert.AnError)
		err := NewClientAPIRepository(gdb).RemoveByClientUUIDAndAPIUUID(clientUUID, apiUUID)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

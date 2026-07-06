package client

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientURIRepository_FindByUUIDAndTenantID(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "client_uris" WHERE.*client_uri_uuid = \$1.*tenant_id = \$2`).
			WithArgs(id.String(), int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_uri_id", "client_uri_uuid", "tenant_id", "client_id", "uri", "type", "created_at", "updated_at"}).
				AddRow(1, id, 1, 1, "https://example.com/callback", "redirect_uri", now, now))
		result, err := NewClientURIRepository(gdb).FindByUUIDAndTenantID(id.String(), 1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "https://example.com/callback", result.URI)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "client_uris" WHERE.*client_uri_uuid = \$1.*tenant_id = \$2`).
			WithArgs(id.String(), int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_uri_id", "client_uri_uuid", "tenant_id"}))
		result, err := NewClientURIRepository(gdb).FindByUUIDAndTenantID(id.String(), 1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "client_uris" WHERE.*client_uri_uuid = \$1.*tenant_id = \$2`).
			WithArgs(id.String(), int64(1), 1).
			WillReturnError(assert.AnError)
		result, err := NewClientURIRepository(gdb).FindByUUIDAndTenantID(id.String(), 1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNewClientURIRepository(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewClientURIRepository(gdb)
	assert.NotNil(t, repo)
}

func TestClientURIRepository_WithTx(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewClientURIRepository(gdb)
	txRepo := repo.WithTx(gdb)
	assert.NotNil(t, txRepo)
}

func TestClientURIRepository_FindByURIAndType(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "client_uris" WHERE.*uri = \$1.*type = \$2.*client_id = \$3.*tenant_id = \$4`).
			WithArgs("https://example.com/callback", "redirect_uri", int64(1), int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_uri_id", "client_uri_uuid", "tenant_id", "client_id", "uri", "type", "created_at", "updated_at"}).
				AddRow(1, id, 1, 1, "https://example.com/callback", "redirect_uri", now, now))
		result, err := NewClientURIRepository(gdb).FindByURIAndType("https://example.com/callback", "redirect_uri", 1, 1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "https://example.com/callback", result.URI)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "client_uris" WHERE.*uri = \$1.*type = \$2.*client_id = \$3.*tenant_id = \$4`).
			WithArgs("https://nonexistent.com", "redirect_uri", int64(1), int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_uri_id", "client_uri_uuid", "tenant_id"}))
		result, err := NewClientURIRepository(gdb).FindByURIAndType("https://nonexistent.com", "redirect_uri", 1, 1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "client_uris" WHERE.*uri = \$1.*type = \$2.*client_id = \$3.*tenant_id = \$4`).
			WithArgs("https://example.com/callback", "redirect_uri", int64(1), int64(1), 1).
			WillReturnError(assert.AnError)
		result, err := NewClientURIRepository(gdb).FindByURIAndType("https://example.com/callback", "redirect_uri", 1, 1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientURIRepository_FindByClientIDAndType(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "client_uris" WHERE.*client_id = \$1.*type = \$2.*tenant_id = \$3`).
			WithArgs(int64(1), "redirect_uri", int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"client_uri_id", "client_uri_uuid", "tenant_id", "client_id", "uri", "type", "created_at", "updated_at"}).
				AddRow(1, id, 1, 1, "https://example.com/callback", "redirect_uri", now, now))
		result, err := NewClientURIRepository(gdb).FindByClientIDAndType(1, "redirect_uri", 1)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty returns empty", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "client_uris" WHERE.*client_id = \$1.*type = \$2.*tenant_id = \$3`).
			WithArgs(int64(1), "redirect_uri", int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"client_uri_id", "client_uri_uuid", "tenant_id"}))
		result, err := NewClientURIRepository(gdb).FindByClientIDAndType(1, "redirect_uri", 1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "client_uris" WHERE.*client_id = \$1.*type = \$2.*tenant_id = \$3`).
			WithArgs(int64(1), "redirect_uri", int64(1)).
			WillReturnError(assert.AnError)
		result, err := NewClientURIRepository(gdb).FindByClientIDAndType(1, "redirect_uri", 1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientURIRepository_DeleteByUUIDAndTenantID(t *testing.T) {
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "client_uris" SET.*"deleted_at"=.*WHERE.*client_uri_uuid = .*tenant_id = .*"client_uris"\."deleted_at" IS NULL`).
			WithArgs(sqlmock.AnyArg(), id.String(), int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		err := NewClientURIRepository(gdb).DeleteByUUIDAndTenantID(id.String(), 1)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "client_uris" SET.*"deleted_at"=.*WHERE.*client_uri_uuid = .*tenant_id = .*"client_uris"\."deleted_at" IS NULL`).
			WithArgs(sqlmock.AnyArg(), id.String(), int64(1)).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()
		err := NewClientURIRepository(gdb).DeleteByUUIDAndTenantID(id.String(), 1)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found when rows affected zero", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "client_uris" SET.*"deleted_at"=.*WHERE.*client_uri_uuid = .*tenant_id = .*"client_uris"\."deleted_at" IS NULL`).
			WithArgs(sqlmock.AnyArg(), id.String(), int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
		err := NewClientURIRepository(gdb).DeleteByUUIDAndTenantID(id.String(), 1)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

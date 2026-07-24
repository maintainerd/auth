package client

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cipRows() *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows([]string{
		"client_identity_provider_id", "client_identity_provider_uuid", "tenant_id", "client_id",
		"identity_provider_id", "is_default", "enabled", "display_order", "created_at", "updated_at",
	}).AddRow(1, uuid.New(), 1, 10, 100, true, true, 0, now, now)
}

// cipTwoEnabledRows returns the client's full connection list with a SECOND
// enabled connection, so removing or disabling the first still leaves a way to
// sign in. Needed by the "client stays usable" invariant.
func cipTwoEnabledRows() *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows([]string{
		"client_identity_provider_id", "client_identity_provider_uuid", "tenant_id", "client_id",
		"identity_provider_id", "is_default", "enabled", "display_order", "created_at", "updated_at",
	}).
		AddRow(1, uuid.New(), 1, 10, 100, true, true, 0, now, now).
		AddRow(2, uuid.New(), 1, 10, 101, false, true, 1, now, now)
}

// cipOnlyOneEnabledRow returns a connection list containing ONLY the connection
// under mutation, so disabling or removing it must be refused.
func cipOnlyOneEnabledRow() *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows([]string{
		"client_identity_provider_id", "client_identity_provider_uuid", "tenant_id", "client_id",
		"identity_provider_id", "is_default", "enabled", "display_order", "created_at", "updated_at",
	}).AddRow(1, uuid.New(), 1, 10, 100, true, true, 0, now, now)
}

func idpPreloadRows() *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows([]string{
		"identity_provider_id", "identity_provider_uuid", "tenant_id", "name", "is_system", "created_at", "updated_at",
	}).AddRow(100, uuid.New(), 1, "google", false, now, now)
}

// idpPreloadSystemRows is the IdentityProvider preload for the built-in system
// provider (is_system = true), used to assert the remove guard.
func idpPreloadSystemRows() *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows([]string{
		"identity_provider_id", "identity_provider_uuid", "tenant_id", "name", "is_system", "created_at", "updated_at",
	}).AddRow(100, uuid.New(), 1, "maintainerd", true, now, now)
}

// cipEmptyRows is an empty client_identity_providers result set.
func cipEmptyRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"client_identity_provider_id"})
}

// cipInsertRow is the RETURNING row for a connection INSERT.
func cipInsertRow() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"client_identity_provider_id"}).AddRow(int64(1))
}

func TestNewClientIdentityProviderRepository(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	assert.NotNil(t, NewClientIdentityProviderRepository(gdb))
}

func TestClientIdentityProviderRepository_WithTx(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewClientIdentityProviderRepository(gdb)
	assert.NotNil(t, repo.WithTx(gdb))
}

func TestClientIdentityProviderRepository_FindByUUIDAndTenantID(t *testing.T) {
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE.*client_identity_provider_uuid = \$1.*tenant_id = \$2`).
			WithArgs(id.String(), int64(1), 1).
			WillReturnRows(cipRows())
		mock.ExpectQuery(`SELECT \* FROM "identity_providers"`).
			WillReturnRows(idpPreloadRows())
		result, err := NewClientIdentityProviderRepository(gdb).FindByUUIDAndTenantID(id.String(), 1)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(10), result.ClientID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE.*client_identity_provider_uuid = \$1.*tenant_id = \$2`).
			WithArgs(id.String(), int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_identity_provider_id"}))
		result, err := NewClientIdentityProviderRepository(gdb).FindByUUIDAndTenantID(id.String(), 1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE.*client_identity_provider_uuid = \$1.*tenant_id = \$2`).
			WithArgs(id.String(), int64(1), 1).
			WillReturnError(assert.AnError)
		result, err := NewClientIdentityProviderRepository(gdb).FindByUUIDAndTenantID(id.String(), 1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientIdentityProviderRepository_FindByClientAndProvider(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE.*client_id = \$1.*identity_provider_id = \$2`).
			WithArgs(int64(10), int64(100), 1).
			WillReturnRows(cipRows())
		result, err := NewClientIdentityProviderRepository(gdb).FindByClientAndProvider(10, 100)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE.*client_id = \$1.*identity_provider_id = \$2`).
			WithArgs(int64(10), int64(100), 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_identity_provider_id"}))
		result, err := NewClientIdentityProviderRepository(gdb).FindByClientAndProvider(10, 100)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE.*client_id = \$1.*identity_provider_id = \$2`).
			WithArgs(int64(10), int64(100), 1).
			WillReturnError(assert.AnError)
		result, err := NewClientIdentityProviderRepository(gdb).FindByClientAndProvider(10, 100)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientIdentityProviderRepository_FindByClientID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE.*client_id = \$1.*ORDER BY display_order`).
			WithArgs(int64(10)).
			WillReturnRows(cipRows())
		mock.ExpectQuery(`SELECT \* FROM "identity_providers"`).
			WillReturnRows(idpPreloadRows())
		result, err := NewClientIdentityProviderRepository(gdb).FindByClientID(10)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE.*client_id = \$1.*ORDER BY display_order`).
			WithArgs(int64(10)).
			WillReturnError(assert.AnError)
		result, err := NewClientIdentityProviderRepository(gdb).FindByClientID(10)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientIdentityProviderRepository_UnsetDefaultForClient(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "client_identity_providers" SET.*is_default.*WHERE.*client_id = \$\d`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		err := NewClientIdentityProviderRepository(gdb).UnsetDefaultForClient(10, 5)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "client_identity_providers" SET`).WillReturnError(assert.AnError)
		mock.ExpectRollback()
		err := NewClientIdentityProviderRepository(gdb).UnsetDefaultForClient(10, 5)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientIdentityProviderRepository_DeleteByUUIDAndTenantID(t *testing.T) {
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		// Soft delete: gorm.DeletedAt makes Delete emit an UPDATE of deleted_at.
		mock.ExpectExec(`UPDATE "client_identity_providers" SET "deleted_at"=.*WHERE.*client_identity_provider_uuid = \$\d.*tenant_id = \$\d`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		err := NewClientIdentityProviderRepository(gdb).DeleteByUUIDAndTenantID(id.String(), 1)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found when rows affected zero", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "client_identity_providers" SET "deleted_at"=`).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
		err := NewClientIdentityProviderRepository(gdb).DeleteByUUIDAndTenantID(id.String(), 1)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "client_identity_providers" SET "deleted_at"=`).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()
		err := NewClientIdentityProviderRepository(gdb).DeleteByUUIDAndTenantID(id.String(), 1)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

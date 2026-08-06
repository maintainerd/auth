package client

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientRepository_FindByUUIDAndTenantID(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "clients" WHERE .*client_uuid = \$1.*tenant_id = \$2.*AND "clients"\."deleted_at" IS NULL`).
			WithArgs(id, int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "status", "created_at", "updated_at"}).
				AddRow(1, id, 1, 1, "test-client", "active", now, now))

		mock.ExpectQuery(`SELECT \* FROM "client_uris" WHERE "client_uris"\."client_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"client_uri_id", "client_uri_uuid", "tenant_id", "client_id", "uri", "type", "created_at", "updated_at"}))

		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE "client_identity_providers"\."client_id" = \$1.*"client_identity_providers"\."deleted_at" IS NULL`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"client_identity_provider_id", "client_identity_provider_uuid", "tenant_id", "client_id", "identity_provider_id", "is_default", "enabled", "display_order", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), 1, 1, 1, true, true, 0, now, now))

		mock.ExpectQuery(`SELECT \* FROM "identity_providers" WHERE "identity_providers"\."identity_provider_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "tenant_id", "name", "status", "created_at", "updated_at"}).
				AddRow(1, 1, "test-idp", "active", now, now))

		mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE "tenants"\."tenant_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name", "status", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), "test-tenant", "active", now, now))

		result, err := NewClientRepository(gdb).FindByUUIDAndTenantID(id, 1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-client", result.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "clients" WHERE .*client_uuid = \$1.*tenant_id = \$2.*AND "clients"\."deleted_at" IS NULL`).
			WithArgs(id, int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "tenant_id"}))
		result, err := NewClientRepository(gdb).FindByUUIDAndTenantID(id, 1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "clients" WHERE .*client_uuid = \$1.*tenant_id = \$2`).
			WithArgs(id, int64(1), 1).
			WillReturnError(assert.AnError)
		result, err := NewClientRepository(gdb).FindByUUIDAndTenantID(id, 1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientRepository_FindByNameAndIdentityProvider(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "clients" JOIN client_identity_providers.*WHERE.*clients\.name = \$1.*clients\.tenant_id = \$2.*client_identity_providers\.identity_provider_id = \$3.*client_identity_providers\.deleted_at IS NULL.*AND "clients"\."deleted_at" IS NULL`).
			WithArgs("test-client", int64(1), int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "tenant_id", "name", "status", "created_at", "updated_at"}).
				AddRow(1, id, 1, "test-client", "active", now, now))
		result, err := NewClientRepository(gdb).FindByNameAndIdentityProvider("test-client", 1, 1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-client", result.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "clients" JOIN client_identity_providers.*WHERE.*clients\.name = \$1.*clients\.tenant_id = \$2.*client_identity_providers\.identity_provider_id = \$3.*client_identity_providers\.deleted_at IS NULL.*AND "clients"\."deleted_at" IS NULL`).
			WithArgs("test-client", int64(1), int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "tenant_id"}))
		result, err := NewClientRepository(gdb).FindByNameAndIdentityProvider("test-client", 1, 1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "clients" JOIN client_identity_providers.*WHERE.*clients\.name = \$1.*clients\.tenant_id = \$2.*client_identity_providers\.identity_provider_id = \$3`).
			WithArgs("test-client", int64(1), int64(1), 1).
			WillReturnError(assert.AnError)
		result, err := NewClientRepository(gdb).FindByNameAndIdentityProvider("test-client", 1, 1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientRepository_FindByNameAndTenantID(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "clients" WHERE.*name = \$1.*tenant_id = \$2.*AND "clients"\."deleted_at" IS NULL`).
			WithArgs("test-client", int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "status", "created_at", "updated_at"}).
				AddRow(1, id, 1, 1, "test-client", "active", now, now))

		mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE "tenants"\."tenant_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name", "status", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), "test-tenant", "active", now, now))

		// Enabled connections are preloaded: callers create user identities
		// against the client's provider, and there is no identity_provider_id
		// column on clients to fall back to.
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE "client_identity_providers"\."client_id" = \$1 AND enabled = \$2`).
			WithArgs(int64(1), true).
			WillReturnRows(sqlmock.NewRows([]string{"client_identity_provider_id", "client_identity_provider_uuid", "tenant_id", "client_id", "identity_provider_id", "is_default", "enabled", "display_order", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), 1, 1, 4, true, true, 0, now, now))

		mock.ExpectQuery(`SELECT \* FROM "identity_providers" WHERE "identity_providers"\."identity_provider_id" = \$1`).
			WithArgs(int64(4)).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "identity_provider_uuid", "tenant_id", "name", "status", "created_at", "updated_at"}).
				AddRow(4, uuid.New(), 1, "built-in", "active", now, now))

		result, err := NewClientRepository(gdb).FindByNameAndTenantID("test-client", 1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-client", result.Name)
		assert.Equal(t, int64(4), result.DefaultConnectedIdentityProviderID())
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "clients" WHERE.*name = \$1.*tenant_id = \$2.*AND "clients"\."deleted_at" IS NULL`).
			WithArgs("test-client", int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "tenant_id"}))
		result, err := NewClientRepository(gdb).FindByNameAndTenantID("test-client", 1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "clients" WHERE.*name = \$1.*tenant_id = \$2`).
			WithArgs("test-client", int64(1), 1).
			WillReturnError(assert.AnError)
		result, err := NewClientRepository(gdb).FindByNameAndTenantID("test-client", 1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientRepository_FindSystem(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "clients" JOIN tenants.*WHERE.*clients\.is_system = \$1.*clients\.status = \$2.*clients\.name = \$3.*tenants\.is_system = \$4.*AND "clients"\."deleted_at" IS NULL`).
			WithArgs(true, "active", shared.SystemClientNameAuthConsole, true, 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "status", "is_system", "created_at", "updated_at"}).
				AddRow(1, id, 1, 1, shared.SystemClientNameAuthConsole, "active", true, now, now))

		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE "client_identity_providers"\."client_id" = \$1 AND enabled = \$2.*"client_identity_providers"\."deleted_at" IS NULL`).
			WithArgs(int64(1), true).
			WillReturnRows(sqlmock.NewRows([]string{"client_identity_provider_id", "client_identity_provider_uuid", "tenant_id", "client_id", "identity_provider_id", "is_default", "enabled", "display_order", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), 1, 1, 1, true, true, 0, now, now))

		mock.ExpectQuery(`SELECT \* FROM "identity_providers" WHERE "identity_providers"\."identity_provider_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "tenant_id", "name", "status", "created_at", "updated_at"}).
				AddRow(1, 1, "test-idp", "active", now, now))

		mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE "tenants"\."tenant_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name", "status", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), "system-tenant", "active", now, now))
		mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE "tenants"\."tenant_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name", "status", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), "system-tenant", "active", now, now))

		result, err := NewClientRepository(gdb).FindSystem()
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, shared.SystemClientNameAuthConsole, result.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "clients" JOIN tenants`).
			WithArgs(true, "active", shared.SystemClientNameAuthConsole, true, 1).
			WillReturnError(assert.AnError)
		result, err := NewClientRepository(gdb).FindSystem()
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "clients" JOIN tenants.*WHERE.*clients\.is_system = \$1.*clients\.status = \$2.*clients\.name = \$3.*tenants\.is_system = \$4.*AND "clients"\."deleted_at" IS NULL`).
			WithArgs(true, "active", shared.SystemClientNameAuthConsole, true, 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "tenant_id"}))
		result, err := NewClientRepository(gdb).FindSystem()
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNewClientRepository(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewClientRepository(gdb)
	assert.NotNil(t, repo)
}

func TestClientRepository_WithTx(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewClientRepository(gdb)
	txRepo := repo.WithTx(gdb)
	assert.NotNil(t, txRepo)
}

func TestClientRepository_FindByClientID(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "clients" WHERE.*client_id = \$1.*tenant_id = \$2.*AND "clients"\."deleted_at" IS NULL`).
			WithArgs("1", int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "status", "created_at", "updated_at"}).
				AddRow(1, id, 1, 1, "test-client", "active", now, now))
		result, err := NewClientRepository(gdb).FindByClientID("1", 1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(1), result.ClientID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "clients" WHERE.*client_id = \$1.*tenant_id = \$2.*AND "clients"\."deleted_at" IS NULL`).
			WithArgs("999", int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "tenant_id"}))
		result, err := NewClientRepository(gdb).FindByClientID("999", 1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "clients" WHERE.*client_id = \$1.*tenant_id = \$2`).
			WithArgs("1", int64(1), 1).
			WillReturnError(assert.AnError)
		result, err := NewClientRepository(gdb).FindByClientID("1", 1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientRepository_FindAllByTenantID(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "clients" WHERE clients\.tenant_id = \$1 AND "clients"\."deleted_at" IS NULL`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "status", "created_at", "updated_at"}).
				AddRow(1, id, 1, 1, "test-client", "active", now, now))
		result, err := NewClientRepository(gdb).FindAllByTenantID(1)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty returns empty", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "clients" WHERE clients\.tenant_id = \$1 AND "clients"\."deleted_at" IS NULL`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "tenant_id"}))
		result, err := NewClientRepository(gdb).FindAllByTenantID(1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "clients" WHERE clients\.tenant_id = \$1`).
			WithArgs(int64(1)).
			WillReturnError(assert.AnError)
		result, err := NewClientRepository(gdb).FindAllByTenantID(1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientRepository_FindDefaultByTenantID(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "clients" WHERE.*tenant_id = \$1.*is_default = true.*status = \$2.*AND "clients"\."deleted_at" IS NULL`).
			WithArgs(int64(1), "active", 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "status", "is_default", "created_at", "updated_at"}).
				AddRow(1, id, 1, 1, "default-client", "active", true, now, now))
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE "client_identity_providers"\."client_id" = \$1 AND enabled = \$2.*"client_identity_providers"\."deleted_at" IS NULL`).
			WithArgs(int64(1), true).
			WillReturnRows(sqlmock.NewRows([]string{"client_identity_provider_id", "client_identity_provider_uuid", "tenant_id", "client_id", "identity_provider_id", "is_default", "enabled", "display_order", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), 1, 1, 1, true, true, 0, now, now))
		mock.ExpectQuery(`SELECT \* FROM "identity_providers" WHERE "identity_providers"\."identity_provider_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "tenant_id", "name", "status", "created_at", "updated_at"}).
				AddRow(1, 1, "test-idp", "active", now, now))
		mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE "tenants"\."tenant_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name", "status", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), "test-tenant", "active", now, now))
		mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE "tenants"\."tenant_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name", "status", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), "test-tenant", "active", now, now))
		result, err := NewClientRepository(gdb).FindDefaultByTenantID(1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsDefault)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "clients" WHERE.*tenant_id = \$1.*is_default = true.*status = \$2.*AND "clients"\."deleted_at" IS NULL`).
			WithArgs(int64(1), "active", 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "tenant_id"}))
		result, err := NewClientRepository(gdb).FindDefaultByTenantID(1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "clients" WHERE.*tenant_id = \$1.*is_default = true.*status = \$2`).
			WithArgs(int64(1), "active", 1).
			WillReturnError(assert.AnError)
		result, err := NewClientRepository(gdb).FindDefaultByTenantID(1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientRepository_SetStatusByUUID(t *testing.T) {
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "clients" SET.*status.*=.*\$\d+.*WHERE.*client_uuid = \$\d+.*tenant_id = \$\d+`).
			WithArgs("inactive", sqlmock.AnyArg(), id, int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		err := NewClientRepository(gdb).SetStatusByUUID(id, 1, "inactive")
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "clients" SET.*status.*=.*\$\d+.*WHERE.*client_uuid = \$\d+.*tenant_id = \$\d+`).
			WithArgs("inactive", sqlmock.AnyArg(), id, int64(1)).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()
		err := NewClientRepository(gdb).SetStatusByUUID(id, 1, "inactive")
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientRepository_FindByClientIDAndIdentityProvider(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "clients" JOIN client_identity_providers.*JOIN identity_providers.*WHERE.*clients\.identifier = \$1.*clients\.status = \$2.*identity_providers\.identifier = \$3.*identity_providers\.status = \$4.*client_identity_providers\.enabled = \$5.*client_identity_providers\.deleted_at IS NULL.*AND "clients"\."deleted_at" IS NULL`).
			WithArgs("client-ident", "active", "idp-ident", "active", true, 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "identifier", "status", "created_at", "updated_at"}).
				AddRow(1, id, 1, 1, "test-client", "client-ident", "active", now, now))

		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE "client_identity_providers"\."client_id" = \$1 AND enabled = \$2.*"client_identity_providers"\."deleted_at" IS NULL`).
			WithArgs(int64(1), true).
			WillReturnRows(sqlmock.NewRows([]string{"client_identity_provider_id", "client_identity_provider_uuid", "tenant_id", "client_id", "identity_provider_id", "is_default", "enabled", "display_order", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), 1, 1, 1, true, true, 0, now, now))

		mock.ExpectQuery(`SELECT \* FROM "identity_providers" WHERE "identity_providers"\."identity_provider_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "tenant_id", "name", "status", "created_at", "updated_at"}).
				AddRow(1, 1, "test-idp", "active", now, now))

		mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE "tenants"\."tenant_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name", "status", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), "test-tenant", "active", now, now))
		mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE "tenants"\."tenant_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name", "status", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), "test-tenant", "active", now, now))

		result, err := NewClientRepository(gdb).FindByClientIDAndIdentityProvider("client-ident", "idp-ident")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-client", result.Name)
		require.NotNil(t, result.Identifier)
		assert.Equal(t, "client-ident", *result.Identifier)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "clients" JOIN client_identity_providers.*JOIN identity_providers.*WHERE.*clients\.identifier = \$1.*clients\.status = \$2.*identity_providers\.identifier = \$3.*identity_providers\.status = \$4.*client_identity_providers\.enabled = \$5.*client_identity_providers\.deleted_at IS NULL.*AND "clients"\."deleted_at" IS NULL`).
			WithArgs("missing-client", "active", "idp-ident", "active", true, 1).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "tenant_id"}))
		result, err := NewClientRepository(gdb).FindByClientIDAndIdentityProvider("missing-client", "idp-ident")
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "clients" JOIN client_identity_providers.*JOIN identity_providers.*WHERE.*clients\.identifier = \$1.*clients\.status = \$2.*identity_providers\.identifier = \$3.*identity_providers\.status = \$4.*client_identity_providers\.enabled = \$5`).
			WithArgs("client-ident", "active", "idp-ident", "active", true, 1).
			WillReturnError(assert.AnError)
		result, err := NewClientRepository(gdb).FindByClientIDAndIdentityProvider("client-ident", "idp-ident")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientRepository_DeleteByUUIDAndTenantID(t *testing.T) {
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "clients" SET.*"deleted_at"=.*WHERE.*client_uuid = \$\d+.*tenant_id = \$\d+`).
			WithArgs(sqlmock.AnyArg(), id, int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		err := NewClientRepository(gdb).DeleteByUUIDAndTenantID(id, 1)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "clients" SET.*"deleted_at"=.*WHERE.*client_uuid = \$\d+.*tenant_id = \$\d+`).
			WithArgs(sqlmock.AnyArg(), id, int64(1)).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()
		err := NewClientRepository(gdb).DeleteByUUIDAndTenantID(id, 1)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found when rows affected zero", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "clients" SET.*"deleted_at"=.*WHERE.*client_uuid = \$\d+.*tenant_id = \$\d+`).
			WithArgs(sqlmock.AnyArg(), id, int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
		err := NewClientRepository(gdb).DeleteByUUIDAndTenantID(id, 1)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientRepository_FindPaginated(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "clients" WHERE.*clients\.tenant_id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT .* FROM "clients" WHERE.*clients\.tenant_id = \$1.*ORDER BY clients\.created_at DESC.*LIMIT \$\d+`).
			WithArgs(int64(1), 10).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "status", "created_at", "updated_at"}).
				AddRow(1, id, 1, 1, "test-client", "active", now, now))
		mock.ExpectQuery(`SELECT \* FROM "client_uris" WHERE "client_uris"\."client_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"client_uri_id", "client_uri_uuid", "tenant_id", "client_id", "uri", "type", "created_at", "updated_at"}))
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE "client_identity_providers"\."client_id" = \$1 AND enabled = \$2.*"client_identity_providers"\."deleted_at" IS NULL`).
			WithArgs(int64(1), true).
			WillReturnRows(sqlmock.NewRows([]string{"client_identity_provider_id", "client_identity_provider_uuid", "tenant_id", "client_id", "identity_provider_id", "is_default", "enabled", "display_order", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), 1, 1, 1, true, true, 0, now, now))
		mock.ExpectQuery(`SELECT \* FROM "identity_providers" WHERE "identity_providers"\."identity_provider_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "tenant_id", "name", "status", "created_at", "updated_at"}).
				AddRow(1, 1, "test-idp", "active", now, now))
		mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE "tenants"\."tenant_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name", "status", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), "test-tenant", "active", now, now))
		result, err := NewClientRepository(gdb).FindPaginated(ClientRepositoryGetFilter{
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
		mock.ExpectQuery(`SELECT count\(\*\) FROM "clients" WHERE.*clients\.tenant_id = \$1`).
			WithArgs(int64(1)).
			WillReturnError(assert.AnError)
		result, err := NewClientRepository(gdb).FindPaginated(ClientRepositoryGetFilter{
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

	t.Run("with all filter options", func(t *testing.T) {
		isDef := true
		isSys := false
		idpID := int64(5)
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "clients" JOIN client_identity_providers.*WHERE.*clients\.tenant_id = \$1.*clients\.status IN \(\$2\).*clients\.is_default = \$3.*clients\.is_system = \$4.*clients\.client_type IN \(\$5\).*client_identity_providers\.identity_provider_id = \$6.*client_identity_providers\.deleted_at IS NULL`).
			WithArgs(int64(1), "active", true, false, "public", int64(5)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT .* FROM "clients" JOIN client_identity_providers.*WHERE.*clients\.tenant_id = \$1.*clients\.status IN \(\$2\).*clients\.is_default = \$3.*clients\.is_system = \$4.*clients\.client_type IN \(\$5\).*client_identity_providers\.identity_provider_id = \$6.*client_identity_providers\.deleted_at IS NULL.*ORDER BY clients\.created_at DESC.*LIMIT \$\d+`).
			WithArgs(int64(1), "active", true, false, "public", int64(5), 10).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "status", "created_at", "updated_at"}).
				AddRow(1, id, 1, 5, "test-client", "active", now, now))
		mock.ExpectQuery(`SELECT \* FROM "client_uris" WHERE "client_uris"\."client_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"client_uri_id", "client_uri_uuid", "tenant_id", "client_id", "uri", "type", "created_at", "updated_at"}))
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE "client_identity_providers"\."client_id" = \$1 AND enabled = \$2.*"client_identity_providers"\."deleted_at" IS NULL`).
			WithArgs(int64(1), true).
			WillReturnRows(sqlmock.NewRows([]string{"client_identity_provider_id", "client_identity_provider_uuid", "tenant_id", "client_id", "identity_provider_id", "is_default", "enabled", "display_order", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), 1, 1, 5, true, true, 0, now, now))
		mock.ExpectQuery(`SELECT \* FROM "identity_providers" WHERE "identity_providers"\."identity_provider_id" = \$1`).
			WithArgs(int64(5)).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "tenant_id", "name", "status", "created_at", "updated_at"}).
				AddRow(5, 1, "test-idp", "active", now, now))
		mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE "tenants"\."tenant_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name", "status", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), "test-tenant", "active", now, now))
		result, err := NewClientRepository(gdb).FindPaginated(ClientRepositoryGetFilter{
			TenantID:           1,
			Status:             []string{"active"},
			IsDefault:          &isDef,
			IsSystem:           &isSys,
			ClientType:         []string{"public"},
			IdentityProviderID: &idpID,
			Page:               1,
			Limit:              10,
			SortBy:             "created_at",
			SortOrder:          "DESC",
		})
		require.NoError(t, err)
		assert.Len(t, result.Data, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

package tenant

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTenantRepository_FindByName(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE name = \$1 AND "tenants"."deleted_at" IS NULL ORDER BY "tenants"."tenant_id" LIMIT \$2`).
			WithArgs("acme", 1).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name", "status", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), "acme", "active", now, now))

		result, err := NewTenantRepository(db).FindByName("acme")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "acme", result.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE name = \$1 AND "tenants"."deleted_at" IS NULL ORDER BY "tenants"."tenant_id" LIMIT \$2`).
			WithArgs("missing", 1).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name"}))

		result, err := NewTenantRepository(db).FindByName("missing")

		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE name = \$1`).
			WithArgs("acme", 1).
			WillReturnError(assert.AnError)

		result, err := NewTenantRepository(db).FindByName("acme")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTenantRepository_FindByIdentifier(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE identifier = \$1 AND "tenants"."deleted_at" IS NULL ORDER BY "tenants"."tenant_id" LIMIT \$2`).
			WithArgs("acme", 1).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name", "identifier", "status", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), "acme", "acme", "active", now, now))

		result, err := NewTenantRepository(db).FindByIdentifier("acme")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "acme", result.Identifier)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE identifier = \$1 AND "tenants"."deleted_at" IS NULL ORDER BY "tenants"."tenant_id" LIMIT \$2`).
			WithArgs("missing", 1).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "identifier"}))

		result, err := NewTenantRepository(db).FindByIdentifier("missing")

		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE identifier = \$1`).
			WithArgs("acme", 1).
			WillReturnError(assert.AnError)

		result, err := NewTenantRepository(db).FindByIdentifier("acme")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTenantRepository_FindSystem(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE is_system = \$1 AND "tenants"."deleted_at" IS NULL ORDER BY "tenants"."tenant_id" LIMIT \$2`).
			WithArgs(true, 1).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name", "is_system", "status", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), "system", true, "active", now, now))

		result, err := NewTenantRepository(db).FindSystem()

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.IsSystem)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE is_system = \$1 AND "tenants"."deleted_at" IS NULL ORDER BY "tenants"."tenant_id" LIMIT \$2`).
			WithArgs(true, 1).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name"}))

		result, err := NewTenantRepository(db).FindSystem()

		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE is_system = \$1`).
			WithArgs(true, 1).
			WillReturnError(assert.AnError)

		result, err := NewTenantRepository(db).FindSystem()

		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTenantRepository_FindPaginated(t *testing.T) {
	db, mock := newMockGormDB(t)
	now := time.Now()
	name := "acme"
	system := false

	mock.ExpectQuery(`SELECT count\(\*\) FROM "tenants" WHERE.*name.*display_name.*description.*identifier.*status IN.*is_system`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE.*name.*display_name.*description.*identifier.*status IN.*is_system.*ORDER BY name ASC LIMIT`).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name", "display_name", "description", "identifier", "status", "is_system", "created_at", "updated_at"}).
			AddRow(1, uuid.New(), "acme", "Acme", "desc", "acme", "active", false, now, now))

	result, err := NewTenantRepository(db).FindPaginated(TenantRepositoryGetFilter{
		Name:        &name,
		DisplayName: &name,
		Description: &name,
		Identifier:  &name,
		Status:      []string{"active"},
		IsSystem:    &system,
		Page:        1,
		Limit:       10,
		SortBy:      "name",
		SortOrder:   "asc",
	})

	require.NoError(t, err)
	require.Len(t, result.Data, 1)
	assert.Equal(t, "acme", result.Data[0].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantRepository_Mutations(t *testing.T) {
	t.Run("WithTx returns repository bound to tx", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tx := db.Begin()
		require.NoError(t, tx.Error)

		repo := NewTenantRepository(db).WithTx(tx)

		assert.NotNil(t, repo)
		require.NoError(t, tx.Rollback().Error)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("SetStatusByUUID updates status", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		tenantUUID := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "tenants" SET "status"=\$1,"updated_at"=\$2 WHERE tenant_uuid = \$3 AND "tenants"."deleted_at" IS NULL`).
			WithArgs("inactive", sqlmock.AnyArg(), tenantUUID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := NewTenantRepository(db).SetStatusByUUID(tenantUUID, "inactive")

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("SetSystemStatusByUUID updates system flag", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		tenantUUID := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "tenants" SET "is_system"=\$1,"updated_at"=\$2 WHERE tenant_uuid = \$3 AND "tenants"."deleted_at" IS NULL`).
			WithArgs(true, sqlmock.AnyArg(), tenantUUID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := NewTenantRepository(db).SetSystemStatusByUUID(tenantUUID, true)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTenantRepository_DeleteCascade(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT set_config\('maintainerd\.allow_auth_event_delete', \$1, true\)`).
			WithArgs("tenant_delete").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE "tenant_members" SET "deleted_at"=\$1 WHERE tenant_id = \$2 AND "tenant_members"."deleted_at" IS NULL`).
			WithArgs(sqlmock.AnyArg(), int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectExec(`DELETE FROM "tenant_settings" WHERE tenant_id = \$1`).
			WithArgs(int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := db.Transaction(func(tx *gorm.DB) error {
			return NewTenantRepository(db).DeleteCascade(context.Background(), tx, 10, []any{&TenantMember{}, &TenantSetting{}})
		})

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("delete error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT set_config\('maintainerd\.allow_auth_event_delete', \$1, true\)`).
			WithArgs("tenant_delete").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE "tenant_members" SET "deleted_at"=\$1 WHERE tenant_id = \$2 AND "tenant_members"."deleted_at" IS NULL`).
			WithArgs(sqlmock.AnyArg(), int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectExec(`DELETE FROM "tenant_settings" WHERE tenant_id = \$1`).
			WithArgs(int64(10)).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := db.Transaction(func(tx *gorm.DB) error {
			return NewTenantRepository(db).DeleteCascade(context.Background(), tx, 10, []any{&TenantMember{}, &TenantSetting{}})
		})

		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

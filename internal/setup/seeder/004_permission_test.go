package seeder

import (
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPermissionsIncludesGRPCManagementPermissions(t *testing.T) {
	permissions := defaultPermissions(1, 2)
	names := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		if _, exists := names[permission.Name]; exists {
			t.Fatalf("duplicate permission %q", permission.Name)
		}
		names[permission.Name] = struct{}{}
	}

	for _, name := range []string{
		"client:secret:rotate",
		"user:invite",
		"security-setting:update",
		"tenant-setting:update",
		"email-config:update",
		"sms-config:update",
		"webhook-endpoint:create",
	} {
		t.Run(name, func(t *testing.T) {
			_, exists := names[name]
			assert.True(t, exists)
		})
	}

	require.NotEmpty(t, permissions)
	assert.Equal(t, int64(1), permissions[0].TenantID)
	assert.Equal(t, int64(2), permissions[0].APIID)
}

func TestSeedPermissions(t *testing.T) {
	t.Run("skips existing permissions", func(t *testing.T) {
		db, mock := newSeederMockDB(t)
		mock.ExpectQuery(isSystemTenantQuery()).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"is_system"}).AddRow(true))
		permissions := defaultPermissions(1, 2)
		for i, permission := range permissions {
			mock.ExpectQuery(permissionExistsQuery()).
				WithArgs(permission.Name, int64(1), 1).
				WillReturnRows(sqlmock.NewRows([]string{"permission_id"}).AddRow(int64(i + 1)))
		}

		require.NoError(t, SeedPermissions(db, 1, 2))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("creates missing permissions", func(t *testing.T) {
		db, mock := newSeederMockDB(t)
		mock.ExpectQuery(isSystemTenantQuery()).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"is_system"}).AddRow(true))
		permissions := defaultPermissions(1, 2)
		for i, permission := range permissions {
			mock.ExpectQuery(permissionExistsQuery()).
				WithArgs(permission.Name, int64(1), 1).
				WillReturnRows(sqlmock.NewRows([]string{"permission_id"}))
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "permissions"`)).
				WillReturnRows(sqlmock.NewRows([]string{"permission_id"}).AddRow(int64(i + 1)))
			mock.ExpectCommit()
		}

		require.NoError(t, SeedPermissions(db, 1, 2))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("check error is returned", func(t *testing.T) {
		db, mock := newSeederMockDB(t)
		mock.ExpectQuery(isSystemTenantQuery()).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"is_system"}).AddRow(true))
		permission := defaultPermissions(1, 2)[0]
		mock.ExpectQuery(permissionExistsQuery()).
			WithArgs(permission.Name, int64(1), 1).
			WillReturnError(assert.AnError)

		err := SeedPermissions(db, 1, 2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check permission")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("create error is returned", func(t *testing.T) {
		db, mock := newSeederMockDB(t)
		mock.ExpectQuery(isSystemTenantQuery()).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"is_system"}).AddRow(true))
		permission := defaultPermissions(1, 2)[0]
		mock.ExpectQuery(permissionExistsQuery()).
			WithArgs(permission.Name, int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"permission_id"}))
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "permissions"`)).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := SeedPermissions(db, 1, 2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to seed permission")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func permissionExistsQuery() string {
	return regexp.QuoteMeta(`SELECT * FROM "permissions" WHERE (name = $1 AND tenant_id = $2) AND "permissions"."deleted_at" IS NULL ORDER BY "permissions"."permission_id" LIMIT $3`)
}

func isSystemTenantQuery() string {
	return regexp.QuoteMeta(`SELECT is_system FROM "tenants" WHERE tenant_id = $1`)
}

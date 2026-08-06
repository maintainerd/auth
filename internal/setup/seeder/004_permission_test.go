package seeder

import (
	"database/sql/driver"
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
		expectPruneScan(mock, sqlmock.NewRows([]string{"permission_id", "name"}))

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
		expectPruneScan(mock, sqlmock.NewRows([]string{"permission_id", "name"}))

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

// TestPruneRetiredPermissions covers the half of the catalog contract that
// creating cannot reach. Trimming defaultPermissions only ever fixed databases
// that did not exist yet; a tenant bootstrapped by an earlier build kept every
// retired row — root:impersonate, user:disable, audit:export — so the console
// still offered them, an admin could still build a role out of them, and the
// role still granted nothing. The rows have to be withdrawn, not just stopped
// from being created.
func TestPruneRetiredPermissions(t *testing.T) {
	t.Run("a converged tenant is left untouched", func(t *testing.T) {
		db, mock := newSeederMockDB(t)
		expectPruneScan(mock, sqlmock.NewRows([]string{"permission_id", "name"}))

		require.NoError(t, pruneRetiredPermissions(db, 1, true))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("retired rows are detached from roles and clients, then soft-deleted", func(t *testing.T) {
		db, mock := newSeederMockDB(t)
		expectPruneScan(mock, sqlmock.NewRows([]string{"permission_id", "name"}).
			AddRow(int64(41), "root:impersonate").
			AddRow(int64(42), "audit:export"))

		// The permission row is soft-deleted, so the ON DELETE CASCADE on both
		// join tables never fires: without these two deletes the retired name
		// stays inside the role's membership and inside the client's granted API
		// permissions, and the withdrawal is cosmetic.
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "role_permissions" WHERE permission_id IN ($1,$2)`)).
			WithArgs(int64(41), int64(42)).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "client_permissions" WHERE permission_id IN ($1,$2)`)).
			WithArgs(int64(41), int64(42)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "permissions" SET "deleted_at"=$1 WHERE permission_id IN ($2,$3) AND "permissions"."deleted_at" IS NULL`)).
			WithArgs(sqlmock.AnyArg(), int64(41), int64(42)).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		require.NoError(t, pruneRetiredPermissions(db, 1, true))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("scan error is returned", func(t *testing.T) {
		db, mock := newSeederMockDB(t)
		mock.ExpectQuery(prunePermissionsQuery()).WillReturnError(assert.AnError)

		err := pruneRetiredPermissions(db, 1, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list retired permissions")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("detach failure aborts before the permission is deleted", func(t *testing.T) {
		// Deleting the permission while its grants survive would leave orphaned
		// role_permissions rows pointing at an invisible permission, so the step
		// stops at the first failure rather than half-applying.
		db, mock := newSeederMockDB(t)
		expectPruneScan(mock, sqlmock.NewRows([]string{"permission_id", "name"}).
			AddRow(int64(41), "root:impersonate"))
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "role_permissions"`)).WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := pruneRetiredPermissions(db, 1, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to detach retired permissions from roles")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("only seeded rows are in scope", func(t *testing.T) {
		// is_system = true is the whole safety margin: permissions an operator
		// minted through permission:create are not in the catalog and would
		// otherwise be deleted on the next seeder run.
		db, mock := newSeederMockDB(t)
		mock.ExpectQuery(prunePermissionsQuery()).
			WithArgs(pruneScanArgs(int64(7), true)...).
			WillReturnRows(sqlmock.NewRows([]string{"permission_id", "name"}))

		require.NoError(t, pruneRetiredPermissions(db, 7, true))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRetainedPermissionNamesMatchWhatTheSeederCreates(t *testing.T) {
	// The prune keeps exactly what the create loop would have made. If the two
	// sets ever diverge, one of two failures follows: the prune deletes a name
	// the seeder immediately recreates (an endless churn of grant revocations),
	// or it spares a name the seeder refuses to create (the retired row survives
	// and the catalog is lying again).
	for _, systemTenant := range []bool{true, false} {
		name := "regular tenant"
		if systemTenant {
			name = "system tenant"
		}

		t.Run(name, func(t *testing.T) {
			var created []string
			for _, permission := range defaultPermissions(1, 2) {
				if !isSeedableForTenant(permission.Name, systemTenant) {
					continue
				}
				created = append(created, permission.Name)
			}

			assert.Equal(t, created, retainedPermissionNames(systemTenant))
		})
	}
}

func TestRetainedPermissionNamesExcludeSystemOnlyFromRegularTenants(t *testing.T) {
	// A regular tenant that was bootstrapped before systemOnlyPermissions existed
	// still holds security:rotate-keys, which reaches the deployment-global
	// signing key every other tenant's tokens verify against. Keeping it out of
	// the retained set is what withdraws it.
	retained := retainedPermissionNames(false)

	for _, name := range systemOnlyPermissions {
		assert.NotContains(t, retained, name)
	}
	assert.Subset(t, retainedPermissionNames(true), systemOnlyPermissions)
	assert.Contains(t, retained, "tenant:read", "only the system-only names are withheld")
}

// expectPruneScan queues the retired-permission lookup with its exact arguments,
// so a change to the scope predicate (tenant, is_system, or the retained set)
// fails here instead of silently widening what the step deletes.
func expectPruneScan(mock sqlmock.Sqlmock, rows *sqlmock.Rows) {
	mock.ExpectQuery(prunePermissionsQuery()).
		WithArgs(pruneScanArgs(int64(1), true)...).
		WillReturnRows(rows)
}

func pruneScanArgs(tenantID int64, systemTenant bool) []driver.Value {
	args := []driver.Value{tenantID, true}
	for _, name := range retainedPermissionNames(systemTenant) {
		args = append(args, name)
	}
	return args
}

func prunePermissionsQuery() string {
	return regexp.QuoteMeta(`SELECT * FROM "permissions" WHERE (tenant_id = $1 AND is_system = $2 AND name NOT IN (`)
}

func permissionExistsQuery() string {
	return regexp.QuoteMeta(`SELECT * FROM "permissions" WHERE (name = $1 AND tenant_id = $2) AND "permissions"."deleted_at" IS NULL ORDER BY "permissions"."permission_id" LIMIT $3`)
}

func isSystemTenantQuery() string {
	return regexp.QuoteMeta(`SELECT is_system FROM "tenants" WHERE tenant_id = $1`)
}

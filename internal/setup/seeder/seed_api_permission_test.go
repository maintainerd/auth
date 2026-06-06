package seeder

import (
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// permissionFetchRows builds a result set for the initial
// "SELECT * FROM permissions WHERE tenant_id = ?" query.
func permissionFetchRows(perms ...[3]int64) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{"permission_id", "api_id", "tenant_id", "name"})
	for _, p := range perms {
		// p = {permission_id, api_id, tenant_id}
		rows.AddRow(p[0], p[1], p[2], "perm")
	}
	return rows
}

func TestSeedAPIPermissions(t *testing.T) {
	const tenantID int64 = 1

	t.Run("creates missing api_permission links", func(t *testing.T) {
		db, mock := newSeederMockDB(t)

		mock.ExpectQuery(regexp.QuoteMeta(`FROM "permissions"`)).
			WillReturnRows(permissionFetchRows([3]int64{10, 2, tenantID}, [3]int64{11, 2, tenantID}))

		// permission 10: no existing link -> insert
		mock.ExpectQuery(regexp.QuoteMeta(`FROM "api_permissions"`)).
			WillReturnRows(sqlmock.NewRows([]string{"api_permission_id"}))
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "api_permissions"`)).
			WillReturnRows(sqlmock.NewRows([]string{"api_permission_id"}).AddRow(int64(1)))
		mock.ExpectCommit()

		// permission 11: no existing link -> insert
		mock.ExpectQuery(regexp.QuoteMeta(`FROM "api_permissions"`)).
			WillReturnRows(sqlmock.NewRows([]string{"api_permission_id"}))
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "api_permissions"`)).
			WillReturnRows(sqlmock.NewRows([]string{"api_permission_id"}).AddRow(int64(2)))
		mock.ExpectCommit()

		require.NoError(t, SeedAPIPermissions(db, tenantID))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("skips links that already exist", func(t *testing.T) {
		db, mock := newSeederMockDB(t)

		mock.ExpectQuery(regexp.QuoteMeta(`FROM "permissions"`)).
			WillReturnRows(permissionFetchRows([3]int64{10, 2, tenantID}))

		// Existing link returned -> no insert.
		mock.ExpectQuery(regexp.QuoteMeta(`FROM "api_permissions"`)).
			WillReturnRows(sqlmock.NewRows([]string{"api_permission_id"}).AddRow(int64(99)))

		require.NoError(t, SeedAPIPermissions(db, tenantID))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("skips permissions without an owning api", func(t *testing.T) {
		db, mock := newSeederMockDB(t)

		// api_id = 0 -> permission is skipped, no api_permissions query issued.
		mock.ExpectQuery(regexp.QuoteMeta(`FROM "permissions"`)).
			WillReturnRows(permissionFetchRows([3]int64{10, 0, tenantID}))

		require.NoError(t, SeedAPIPermissions(db, tenantID))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("fetch error is returned", func(t *testing.T) {
		db, mock := newSeederMockDB(t)

		mock.ExpectQuery(regexp.QuoteMeta(`FROM "permissions"`)).
			WillReturnError(assert.AnError)

		err := SeedAPIPermissions(db, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch permissions")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("lookup error is returned", func(t *testing.T) {
		db, mock := newSeederMockDB(t)

		mock.ExpectQuery(regexp.QuoteMeta(`FROM "permissions"`)).
			WillReturnRows(permissionFetchRows([3]int64{10, 2, tenantID}))
		mock.ExpectQuery(regexp.QuoteMeta(`FROM "api_permissions"`)).
			WillReturnError(assert.AnError)

		err := SeedAPIPermissions(db, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check api_permission")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("create error is returned", func(t *testing.T) {
		db, mock := newSeederMockDB(t)

		mock.ExpectQuery(regexp.QuoteMeta(`FROM "permissions"`)).
			WillReturnRows(permissionFetchRows([3]int64{10, 2, tenantID}))
		mock.ExpectQuery(regexp.QuoteMeta(`FROM "api_permissions"`)).
			WillReturnRows(sqlmock.NewRows([]string{"api_permission_id"}))
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "api_permissions"`)).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := SeedAPIPermissions(db, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create api_permission")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

package iam

import (
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func roleRows(values ...driver.Value) *sqlmock.Rows {
	if len(values) == 0 {
		values = []driver.Value{int64(1), testResourceUUID.String(), tenantID, "role", "desc", "active", false, false, time.Now(), time.Now()}
	}
	return sqlmock.NewRows([]string{
		"role_id", "role_uuid", "tenant_id", "name", "description", "status", "is_default", "is_system", "created_at", "updated_at",
	}).AddRow(values...)
}

func TestRoleRepository(t *testing.T) {
	t.Run("constructor and WithTx", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		repo := NewRoleRepository(db)
		require.NotNil(t, repo)
		assert.NotNil(t, repo.(*roleRepository).WithTx(db))
	})

	t.Run("FindByNameAndTenantID and setup lookups success not found and error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		for i := 0; i < 3; i++ {
			expectAnySelect(mock, "roles").WillReturnRows(roleRows())
			expectAnySelect(mock, "roles").WillReturnError(gorm.ErrRecordNotFound)
			expectAnySelect(mock, "roles").WillReturnError(errors.New("db error"))
		}
		repo := NewRoleRepository(db)

		for _, run := range []func() (*Role, error){
			func() (*Role, error) { return repo.FindByNameAndTenantID("role", tenantID) },
			func() (*Role, error) { return repo.FindRegisteredRoleForSetup(tenantID) },
			func() (*Role, error) { return repo.FindSuperAdminRoleForSetup(tenantID) },
		} {
			got, err := run()
			require.NoError(t, err)
			require.NotNil(t, got)
			got, err = run()
			require.NoError(t, err)
			assert.Nil(t, got)
			got, err = run()
			require.Error(t, err)
			assert.Nil(t, got)
		}
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindAllByTenantID returns rows", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnySelect(mock, "roles").WillReturnRows(roleRows())

		got, err := NewRoleRepository(db).FindAllByTenantID(tenantID)

		require.NoError(t, err)
		assert.Len(t, got, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindPaginated applies filters", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnyCount(mock, "roles").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		expectAnySelect(mock, "roles").WillReturnRows(roleRows())
		name := "role"
		flag := true
		status := "active"

		got, err := NewRoleRepository(db).FindPaginated(RoleRepositoryGetFilter{
			TenantID: tenantID, Name: &name, Description: &name, IsDefault: &flag, IsSystem: &flag,
			Status: []string{status}, Page: 1, Limit: 10,
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Len(t, got.Data, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// The "status mutations" case covered SetStatusByUUID / SetDefaultStatusByUUID /
	// SetSystemStatusByUUID, which are gone: all three were zero-caller AND matched
	// on role_uuid with no tenant predicate (see repository_role.go).

	// roleSortColumns replaced the global union allowlist, which accepted columns
	// `roles` does not have (email, username, client_id) and turned
	// GET /roles?sort_by=email into a Postgres 42703 → 500.
	t.Run("FindPaginated sorting is limited to the roles table", func(t *testing.T) {
		for _, sortBy := range []string{"email", "username", "client_id"} {
			assert.Equal(t, "created_at DESC",
				database.SanitizeOrderIn(roleSortColumns, sortBy, SortOrderDesc, "created_at DESC"),
				"%q is not a roles column and must fall back to the default", sortBy)
		}
		assert.Equal(t, "name ASC",
			database.SanitizeOrderIn(roleSortColumns, "name", "asc", "created_at DESC"))
		// The permissions listing sorts permission rows, so it uses that table's set.
		assert.Equal(t, "permissions.created_at DESC",
			database.SanitizeOrderInPrefixed(permissionSortColumns, "permissions.", "is_default", SortOrderDesc, "permissions.created_at DESC"))
	})

	t.Run("GetPermissionsByRoleUUID applies status filter and preloads API", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		expectAnyCount(mock, "permissions").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		expectAnySelect(mock, "permissions").WillReturnRows(permissionRows())
		expectAnySelect(mock, "apis").WillReturnRows(apiRows(int64(9), testResourceUUID.String(), tenantID, int64(0), "api", "API", "desc", "svc:api", "active", false, time.Now(), time.Now()))
		status := "active"

		got, err := NewRoleRepository(db).GetPermissionsByRoleUUID(RoleRepositoryGetPermissionsFilter{
			RoleUUID: testResourceUUID, Status: &status, Page: 1, Limit: 10,
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Len(t, got.Data, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

//go:build integration

package integration_test

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/maintainerd/maintainerd-auth/internal/client"
	"github.com/maintainerd/maintainerd-auth/internal/iam"
)

// Tenant isolation regression tests.
//
// These exercise the REAL tenant-scoped repository code across multiple
// domains. Each expectation regex requires `tenant_id` to appear in the emitted
// SQL, so if a future change drops the tenant predicate from one of these
// finders the query will no longer match the mock, the repository call will
// error, and the test fails — a genuine cross-domain isolation guard rather
// than a documentation stub.
//
// Per-guard behavior (B1–B5: cross-tenant existence oracle, service-id scoping,
// role-permission SQL predicate, identity-client guard, invite flow-client
// tenant check) additionally has dedicated unit tests in each owning package.
//
// Run with: go test -tags integration ./tests/integration/...

func newMockGormDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return db, mock
}

// TestIsolation_CrossTenantLookup_ReturnsNotFound verifies that the primary
// UUID-addressed finders are tenant-scoped: a lookup for a resource that does
// not belong to the caller's tenant resolves to "not found" (nil), and the
// emitted SQL provably includes the tenant_id predicate.
func TestIsolation_CrossTenantLookup_ReturnsNotFound(t *testing.T) {
	const callerTenantID = int64(999) // a tenant that owns none of the rows below

	t.Run("client.FindByUUIDAndTenantID", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := client.NewClientRepository(db)

		// Regex REQUIRES tenant_id in the WHERE clause; zero rows returned.
		mock.ExpectQuery(`SELECT \* FROM "clients" WHERE.*tenant_id.*LIMIT`).
			WillReturnRows(sqlmock.NewRows([]string{"client_id"}))

		got, err := repo.FindByUUIDAndTenantID(uuid.New(), callerTenantID)
		require.NoError(t, err)
		assert.Nil(t, got, "a client owned by another tenant must resolve to not-found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("api.FindByUUIDAndTenantID", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := iam.NewAPIRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "apis" WHERE.*tenant_id.*LIMIT`).
			WillReturnRows(sqlmock.NewRows([]string{"api_id"}))

		got, err := repo.FindByUUIDAndTenantID(uuid.New(), callerTenantID)
		require.NoError(t, err)
		assert.Nil(t, got, "an API owned by another tenant must resolve to not-found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("role.FindByUUIDAndTenantID", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		repo := iam.NewRoleRepository(db)

		mock.ExpectQuery(`SELECT \* FROM "roles" WHERE.*tenant_id.*LIMIT`).
			WillReturnRows(sqlmock.NewRows([]string{"role_id"}))

		got, err := repo.FindByUUIDAndTenantID(uuid.New(), callerTenantID)
		require.NoError(t, err)
		assert.Nil(t, got, "a role owned by another tenant must resolve to not-found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

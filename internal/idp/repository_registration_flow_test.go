package idp

import (
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistrationFlowRepository(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewRegistrationFlowRepository(gdb)
	assert.NotNil(t, repo)
}

func TestRegistrationFlowRepository_WithTx(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewRegistrationFlowRepository(gdb)
	txRepo := repo.WithTx(gdb)
	assert.NotNil(t, txRepo)
}

func TestRegistrationFlowRepository_FindPaginated(t *testing.T) {
	now := time.Now()

	t.Run("success with tenant filter", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "registration_flows" WHERE .*tenant_id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
		mock.ExpectQuery(`SELECT \* FROM "registration_flows" WHERE .*tenant_id = \$1.*ORDER BY.*LIMIT`).
			WithArgs(int64(1), 10).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_id", "registration_flow_uuid", "tenant_id", "name", "client_id", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "flow1", int64(10), now, now).
				AddRow(2, uuid.New(), int64(1), "flow2", int64(20), now, now))
		mock.ExpectQuery(`SELECT \* FROM "clients" WHERE "clients"\."client_id" IN \(\$1,\$2\)`).
			WithArgs(int64(10), int64(20)).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "name", "created_at", "updated_at"}).
				AddRow(10, uuid.New(), "client1", now, now).
				AddRow(20, uuid.New(), "client2", now, now))
		repo := NewRegistrationFlowRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(RegistrationFlowRepositoryGetFilter{
			TenantID:  &tenantID,
			Page:      1,
			Limit:     10,
			SortBy:    "created_at",
			SortOrder: "desc",
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(2), result.Total)
		assert.Len(t, result.Data, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "registration_flows" WHERE .*tenant_id = \$1`).
			WithArgs(int64(1)).
			WillReturnError(errors.New("db error"))
		repo := NewRegistrationFlowRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(RegistrationFlowRepositoryGetFilter{
			TenantID:  &tenantID,
			Page:      1,
			Limit:     10,
			SortBy:    "created_at",
			SortOrder: "desc",
		})
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with status filter", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "registration_flows" WHERE .*status IN`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "registration_flows" WHERE .*status IN.*ORDER BY.*LIMIT`).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_id", "registration_flow_uuid", "tenant_id", "name", "client_id", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "flow1", int64(1), now, now))
		mock.ExpectQuery(`SELECT \* FROM "clients"`).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "name", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), "client1", now, now))

		repo := NewRegistrationFlowRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(RegistrationFlowRepositoryGetFilter{
			TenantID:  &tenantID,
			Status:    []string{"active"},
			Page:      1,
			Limit:     10,
			SortBy:    "created_at",
			SortOrder: "desc",
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with client_id filter", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		clientID := int64(1)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "registration_flows" WHERE .*client_id`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "registration_flows" WHERE .*client_id.*ORDER BY.*LIMIT`).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_id", "registration_flow_uuid", "tenant_id", "name", "client_id", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "flow1", int64(1), now, now))
		mock.ExpectQuery(`SELECT \* FROM "clients"`).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "name", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), "client1", now, now))

		repo := NewRegistrationFlowRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(RegistrationFlowRepositoryGetFilter{
			TenantID:  &tenantID,
			ClientID:  &clientID,
			Page:      1,
			Limit:     10,
			SortBy:    "created_at",
			SortOrder: "desc",
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with name filter", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		name := "flow1"
		mock.ExpectQuery(`SELECT count\(\*\) FROM "registration_flows" WHERE LOWER\(name\) LIKE`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "registration_flows" WHERE LOWER\(name\) LIKE.*ORDER BY.*LIMIT`).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_id", "registration_flow_uuid", "tenant_id", "name", "client_id", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "flow1", int64(1), now, now))
		mock.ExpectQuery(`SELECT \* FROM "clients"`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "name"}).
				AddRow(1, uuid.New(), "client1"))

		repo := NewRegistrationFlowRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(RegistrationFlowRepositoryGetFilter{
			TenantID:  &tenantID,
			Name:      &name,
			Page:      1,
			Limit:     10,
			SortBy:    "created_at",
			SortOrder: "desc",
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// Search spans the two operator-facing handles (name + identifier) and takes
	// precedence over the individual name/identifier filters.
	t.Run("success with search filter spans name and description", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		search := "  Partner  "
		name := "ignored-when-searching"
		mock.ExpectQuery(`SELECT count\(\*\) FROM "registration_flows" WHERE \(LOWER\(name\) LIKE \$1 OR LOWER\(description\) LIKE \$2\)`).
			WithArgs("%partner%", "%partner%", int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "registration_flows" WHERE \(LOWER\(name\) LIKE \$1 OR LOWER\(description\) LIKE \$2\)`).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_id", "registration_flow_uuid", "tenant_id", "name", "client_id", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "partner-signup", int64(1), now, now))
		mock.ExpectQuery(`SELECT \* FROM "clients"`).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "name"}).AddRow(1, uuid.New(), "client1"))

		repo := NewRegistrationFlowRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(RegistrationFlowRepositoryGetFilter{
			TenantID:  &tenantID,
			Search:    &search,
			Name:      &name,
			Page:      1,
			Limit:     10,
			SortBy:    "created_at",
			SortOrder: "desc",
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("whitespace-only search falls back to the name filter", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		search := "   "
		name := "flow1"
		mock.ExpectQuery(`SELECT count\(\*\) FROM "registration_flows" WHERE LOWER\(name\) LIKE`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "registration_flows" WHERE LOWER\(name\) LIKE`).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_id", "registration_flow_uuid", "tenant_id", "name", "client_id", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "flow1", int64(1), now, now))
		mock.ExpectQuery(`SELECT \* FROM "clients"`).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "name"}).AddRow(1, uuid.New(), "client1"))

		repo := NewRegistrationFlowRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(RegistrationFlowRepositoryGetFilter{
			TenantID: &tenantID, Search: &search, Name: &name, Page: 1, Limit: 10,
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with is_system filter", func(t *testing.T) {
		for _, isSystem := range []bool{true, false} {
			t.Run(map[bool]string{true: "true", false: "false"}[isSystem], func(t *testing.T) {
				gdb, mock := newMockGormDBRegex(t)
				mock.ExpectQuery(`SELECT count\(\*\) FROM "registration_flows" WHERE .*is_system = \$2`).
					WithArgs(int64(1), isSystem).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				mock.ExpectQuery(`SELECT \* FROM "registration_flows" WHERE .*is_system = \$2`).
					WillReturnRows(sqlmock.NewRows([]string{"registration_flow_id", "registration_flow_uuid", "tenant_id", "name", "is_system", "client_id", "created_at", "updated_at"}).
						AddRow(1, uuid.New(), int64(1), "flow1", isSystem, int64(1), now, now))
				mock.ExpectQuery(`SELECT \* FROM "clients"`).
					WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "name"}).AddRow(1, uuid.New(), "client1"))

				repo := NewRegistrationFlowRepository(gdb)
				tenantID := int64(1)
				result, err := repo.FindPaginated(RegistrationFlowRepositoryGetFilter{
					TenantID: &tenantID, IsSystem: &isSystem, Page: 1, Limit: 10,
				})
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Len(t, result.Data, 1)
				assert.Equal(t, isSystem, result.Data[0].IsSystem)
				assert.NoError(t, mock.ExpectationsWereMet())
			})
		}
	})

	// Tenant scoping is mandatory: an unscoped list would leak other tenants' flows.
	t.Run("nil tenant_id is rejected", func(t *testing.T) {
		gdb, _ := newMockGormDBRegex(t)
		repo := NewRegistrationFlowRepository(gdb)
		result, err := repo.FindPaginated(RegistrationFlowRepositoryGetFilter{Page: 1, Limit: 10})
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "tenant_id is required")
	})

	t.Run("zero tenant_id is rejected", func(t *testing.T) {
		gdb, _ := newMockGormDBRegex(t)
		repo := NewRegistrationFlowRepository(gdb)
		zero := int64(0)
		result, err := repo.FindPaginated(RegistrationFlowRepositoryGetFilter{TenantID: &zero, Page: 1, Limit: 10})
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "tenant_id is required")
	})

	t.Run("success with empty name skips filter", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		emptyName := ""
		mock.ExpectQuery(`SELECT count\(\*\) FROM "registration_flows"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "registration_flows"`).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_id", "registration_flow_uuid", "tenant_id", "name", "client_id", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "flow1", int64(1), now, now))
		mock.ExpectQuery(`SELECT \* FROM "clients"`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "name"}).
				AddRow(1, uuid.New(), "client1"))

		repo := NewRegistrationFlowRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(RegistrationFlowRepositoryGetFilter{
			TenantID:  &tenantID,
			Name:      &emptyName,
			Page:      1,
			Limit:     10,
			SortBy:    "created_at",
			SortOrder: "desc",
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// FindByNameAndClientTenant resolves the public registration link. The tenant is
// part of the predicate on purpose — matching the client alone proves the flow
// exists, not that it belongs to the requesting tenant.
func TestRegistrationFlowRepository_FindByNameAndClientTenant(t *testing.T) {
	now := time.Now()

	t.Run("success scopes by name, client and tenant", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "registration_flows" WHERE .*name = \$1 AND client_id = \$2 AND tenant_id = \$3`).
			WithArgs("my-flow", int64(1), int64(7), 1).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_id", "registration_flow_uuid", "tenant_id", "client_id", "name", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(7), int64(1), "my-flow", now, now))
		repo := NewRegistrationFlowRepository(gdb)
		result, err := repo.FindByNameAndClientTenant("my-flow", 1, 7)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "my-flow", result.Name)
		assert.Equal(t, int64(7), result.TenantID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "registration_flows" WHERE .*name = \$1 AND client_id = \$2 AND tenant_id = \$3`).
			WithArgs("none", int64(1), int64(7), 1).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_id", "registration_flow_uuid", "tenant_id", "client_id", "name"}))
		repo := NewRegistrationFlowRepository(gdb)
		result, err := repo.FindByNameAndClientTenant("none", 1, 7)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("cross-tenant name is not returned", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		// The same name exists under tenant 7, but tenant 8 asks for it: the
		// tenant predicate is part of the WHERE clause, so nothing comes back.
		mock.ExpectQuery(`SELECT \* FROM "registration_flows" WHERE .*name = \$1 AND client_id = \$2 AND tenant_id = \$3`).
			WithArgs("my-flow", int64(1), int64(8), 1).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_id", "registration_flow_uuid", "tenant_id", "client_id", "name"}))
		repo := NewRegistrationFlowRepository(gdb)
		result, err := repo.FindByNameAndClientTenant("my-flow", 1, 8)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "registration_flows" WHERE .*name = \$1 AND client_id = \$2 AND tenant_id = \$3`).
			WithArgs("my-flow", int64(1), int64(7), 1).
			WillReturnError(errors.New("db error"))
		repo := NewRegistrationFlowRepository(gdb)
		result, err := repo.FindByNameAndClientTenant("my-flow", 1, 7)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRegistrationFlowRepository_FindByUUIDAndTenantID(t *testing.T) {
	now := time.Now()
	flowUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "registration_flows" WHERE .*registration_flow_uuid = \$1 AND tenant_id = \$2`).
			WithArgs(flowUUID, int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_id", "registration_flow_uuid", "tenant_id", "name", "created_at", "updated_at"}).
				AddRow(1, flowUUID, int64(1), "flow name", now, now))
		repo := NewRegistrationFlowRepository(gdb)
		result, err := repo.FindByUUIDAndTenantID(flowUUID, 1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, flowUUID, result.RegistrationFlowUUID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "registration_flows" WHERE .*registration_flow_uuid = \$1 AND tenant_id = \$2`).
			WithArgs(flowUUID, int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_id", "registration_flow_uuid", "tenant_id", "name"}))
		repo := NewRegistrationFlowRepository(gdb)
		result, err := repo.FindByUUIDAndTenantID(flowUUID, 1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "registration_flows" WHERE .*registration_flow_uuid = \$1 AND tenant_id = \$2`).
			WithArgs(flowUUID, int64(1), 1).
			WillReturnError(errors.New("db error"))
		repo := NewRegistrationFlowRepository(gdb)
		result, err := repo.FindByUUIDAndTenantID(flowUUID, 1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with preloads", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "registration_flows" WHERE .*registration_flow_uuid = \$1 AND tenant_id = \$2`).
			WithArgs(flowUUID, int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_id", "registration_flow_uuid", "tenant_id", "name", "client_id", "created_at", "updated_at"}).
				AddRow(1, flowUUID, int64(1), "flow name", int64(10), now, now))
		mock.ExpectQuery(`SELECT \* FROM "clients"`).
			WithArgs(int64(10)).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "name"}).
				AddRow(10, uuid.New(), "client1"))

		repo := NewRegistrationFlowRepository(gdb)
		result, err := repo.FindByUUIDAndTenantID(flowUUID, 1, "Client")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, flowUUID, result.RegistrationFlowUUID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRegistrationFlowRepository_FindByNameAndTenantID(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "registration_flows" WHERE .*name = \$1 AND tenant_id = \$2`).
			WithArgs("test flow", int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_id", "registration_flow_uuid", "tenant_id", "name", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "test flow", now, now))
		repo := NewRegistrationFlowRepository(gdb)
		result, err := repo.FindByNameAndTenantID("test flow", 1)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "test flow", result.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "registration_flows" WHERE .*name = \$1 AND tenant_id = \$2`).
			WithArgs("none", int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_id", "registration_flow_uuid", "tenant_id", "name"}))
		repo := NewRegistrationFlowRepository(gdb)
		result, err := repo.FindByNameAndTenantID("none", 1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("same name in another tenant is not returned", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "registration_flows" WHERE .*name = \$1 AND tenant_id = \$2`).
			WithArgs("test flow", int64(2), 1).
			WillReturnRows(sqlmock.NewRows([]string{"registration_flow_id", "registration_flow_uuid", "tenant_id", "name"}))
		repo := NewRegistrationFlowRepository(gdb)
		result, err := repo.FindByNameAndTenantID("test flow", 2)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "registration_flows" WHERE .*name = \$1 AND tenant_id = \$2`).
			WithArgs("test flow", int64(1), 1).
			WillReturnError(errors.New("db error"))
		repo := NewRegistrationFlowRepository(gdb)
		result, err := repo.FindByNameAndTenantID("test flow", 1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

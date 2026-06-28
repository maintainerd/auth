package idp

import (
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthFlowRepository(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewAuthFlowRepository(gdb)
	assert.NotNil(t, repo)
}

func TestAuthFlowRepository_WithTx(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewAuthFlowRepository(gdb)
	txRepo := repo.WithTx(gdb)
	assert.NotNil(t, txRepo)
}

func TestAuthFlowRepository_FindPaginated(t *testing.T) {
	now := time.Now()

	t.Run("success with tenant filter", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "auth_flows" WHERE .*tenant_id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
		mock.ExpectQuery(`SELECT \* FROM "auth_flows" WHERE .*tenant_id = \$1.*ORDER BY.*LIMIT`).
			WithArgs(int64(1), 10).
			WillReturnRows(sqlmock.NewRows([]string{"auth_flow_id", "auth_flow_uuid", "tenant_id", "name", "client_id", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "flow1", int64(10), now, now).
				AddRow(2, uuid.New(), int64(1), "flow2", int64(20), now, now))
		mock.ExpectQuery(`SELECT \* FROM "clients" WHERE "clients"\."client_id" IN \(\$1,\$2\)`).
			WithArgs(int64(10), int64(20)).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "name", "created_at", "updated_at"}).
				AddRow(10, uuid.New(), "client1", now, now).
				AddRow(20, uuid.New(), "client2", now, now))
		repo := NewAuthFlowRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(AuthFlowRepositoryGetFilter{
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
		mock.ExpectQuery(`SELECT count\(\*\) FROM "auth_flows" WHERE .*tenant_id = \$1`).
			WithArgs(int64(1)).
			WillReturnError(errors.New("db error"))
		repo := NewAuthFlowRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(AuthFlowRepositoryGetFilter{
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
		mock.ExpectQuery(`SELECT count\(\*\) FROM "auth_flows" WHERE .*status IN`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "auth_flows" WHERE .*status IN.*ORDER BY.*LIMIT`).
			WillReturnRows(sqlmock.NewRows([]string{"auth_flow_id", "auth_flow_uuid", "tenant_id", "name", "client_id", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "flow1", int64(1), now, now))
		mock.ExpectQuery(`SELECT \* FROM "clients"`).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "name", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), "client1", now, now))

		repo := NewAuthFlowRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(AuthFlowRepositoryGetFilter{
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
		mock.ExpectQuery(`SELECT count\(\*\) FROM "auth_flows" WHERE .*client_id`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "auth_flows" WHERE .*client_id.*ORDER BY.*LIMIT`).
			WillReturnRows(sqlmock.NewRows([]string{"auth_flow_id", "auth_flow_uuid", "tenant_id", "name", "client_id", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "flow1", int64(1), now, now))
		mock.ExpectQuery(`SELECT \* FROM "clients"`).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "name", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), "client1", now, now))

		repo := NewAuthFlowRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(AuthFlowRepositoryGetFilter{
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

	t.Run("success with identifier filter", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		identifier := "my-flow"
		mock.ExpectQuery(`SELECT count\(\*\) FROM "auth_flows"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "auth_flows"`).
			WillReturnRows(sqlmock.NewRows([]string{"auth_flow_id", "auth_flow_uuid", "tenant_id", "name", "identifier", "client_id", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "flow1", "my-flow", int64(1), now, now))
		mock.ExpectQuery(`SELECT \* FROM "clients"`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "name"}).
				AddRow(1, uuid.New(), "client1"))

		repo := NewAuthFlowRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(AuthFlowRepositoryGetFilter{
			TenantID:   &tenantID,
			Identifier: &identifier,
			Page:       1,
			Limit:      10,
			SortBy:     "created_at",
			SortOrder:  "desc",
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with name filter", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		name := "flow1"
		mock.ExpectQuery(`SELECT count\(\*\) FROM "auth_flows" WHERE LOWER\(name\) LIKE`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "auth_flows" WHERE LOWER\(name\) LIKE.*ORDER BY.*LIMIT`).
			WillReturnRows(sqlmock.NewRows([]string{"auth_flow_id", "auth_flow_uuid", "tenant_id", "name", "client_id", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "flow1", int64(1), now, now))
		mock.ExpectQuery(`SELECT \* FROM "clients"`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "name"}).
				AddRow(1, uuid.New(), "client1"))

		repo := NewAuthFlowRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(AuthFlowRepositoryGetFilter{
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

	t.Run("success with empty name skips filter", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		emptyName := ""
		mock.ExpectQuery(`SELECT count\(\*\) FROM "auth_flows"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "auth_flows"`).
			WillReturnRows(sqlmock.NewRows([]string{"auth_flow_id", "auth_flow_uuid", "tenant_id", "name", "client_id", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "flow1", int64(1), now, now))
		mock.ExpectQuery(`SELECT \* FROM "clients"`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "name"}).
				AddRow(1, uuid.New(), "client1"))

		repo := NewAuthFlowRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(AuthFlowRepositoryGetFilter{
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

func TestAuthFlowRepository_FindByIdentifierAndClientID(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "auth_flows" WHERE .*identifier = \$1 AND client_id = \$2`).
			WithArgs("my-flow", int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"auth_flow_id", "auth_flow_uuid", "tenant_id", "identifier", "client_id", "name", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "my-flow", int64(1), "flow name", now, now))
		repo := NewAuthFlowRepository(gdb)
		result, err := repo.FindByIdentifierAndClientID("my-flow", 1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "my-flow", result.Identifier)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "auth_flows" WHERE .*identifier = \$1 AND client_id = \$2`).
			WithArgs("none", int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"auth_flow_id", "auth_flow_uuid", "tenant_id", "identifier", "client_id", "name"}))
		repo := NewAuthFlowRepository(gdb)
		result, err := repo.FindByIdentifierAndClientID("none", 1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "auth_flows" WHERE .*identifier = \$1 AND client_id = \$2`).
			WithArgs("my-flow", int64(1), 1).
			WillReturnError(errors.New("db error"))
		repo := NewAuthFlowRepository(gdb)
		result, err := repo.FindByIdentifierAndClientID("my-flow", 1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAuthFlowRepository_FindByClientID(t *testing.T) {
	t.Run("returns attached flows in stable order", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "auth_flows" WHERE client_id = \$1.*ORDER BY auth_flow_id ASC`).
			WithArgs(int64(9)).
			WillReturnRows(sqlmock.NewRows([]string{"auth_flow_id", "client_id", "status", "allow_registration"}).
				AddRow(1, 9, shared.StatusActive, true).
				AddRow(2, 9, shared.StatusInactive, false))
		repo := NewAuthFlowRepository(gdb)
		flows, err := repo.FindByClientID(9)
		require.NoError(t, err)
		require.Len(t, flows, 2)
		assert.Equal(t, int64(1), flows[0].AuthFlowID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates database error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "auth_flows" WHERE client_id = \$1`).
			WithArgs(int64(9)).
			WillReturnError(errors.New("db error"))
		repo := NewAuthFlowRepository(gdb)
		flows, err := repo.FindByClientID(9)
		require.Error(t, err)
		assert.Nil(t, flows)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAuthFlowRepository_FindByUUIDAndTenantID(t *testing.T) {
	now := time.Now()
	flowUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "auth_flows" WHERE .*auth_flow_uuid = \$1 AND tenant_id = \$2`).
			WithArgs(flowUUID, int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"auth_flow_id", "auth_flow_uuid", "tenant_id", "name", "created_at", "updated_at"}).
				AddRow(1, flowUUID, int64(1), "flow name", now, now))
		repo := NewAuthFlowRepository(gdb)
		result, err := repo.FindByUUIDAndTenantID(flowUUID, 1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, flowUUID, result.AuthFlowUUID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "auth_flows" WHERE .*auth_flow_uuid = \$1 AND tenant_id = \$2`).
			WithArgs(flowUUID, int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"auth_flow_id", "auth_flow_uuid", "tenant_id", "name"}))
		repo := NewAuthFlowRepository(gdb)
		result, err := repo.FindByUUIDAndTenantID(flowUUID, 1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "auth_flows" WHERE .*auth_flow_uuid = \$1 AND tenant_id = \$2`).
			WithArgs(flowUUID, int64(1), 1).
			WillReturnError(errors.New("db error"))
		repo := NewAuthFlowRepository(gdb)
		result, err := repo.FindByUUIDAndTenantID(flowUUID, 1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with preloads", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "auth_flows" WHERE .*auth_flow_uuid = \$1 AND tenant_id = \$2`).
			WithArgs(flowUUID, int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"auth_flow_id", "auth_flow_uuid", "tenant_id", "name", "client_id", "created_at", "updated_at"}).
				AddRow(1, flowUUID, int64(1), "flow name", int64(10), now, now))
		mock.ExpectQuery(`SELECT \* FROM "clients"`).
			WithArgs(int64(10)).
			WillReturnRows(sqlmock.NewRows([]string{"client_id", "client_uuid", "name"}).
				AddRow(10, uuid.New(), "client1"))

		repo := NewAuthFlowRepository(gdb)
		result, err := repo.FindByUUIDAndTenantID(flowUUID, 1, "Client")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, flowUUID, result.AuthFlowUUID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAuthFlowRepository_FindByName(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "auth_flows" WHERE .*name = \$1`).
			WithArgs("test flow", 1).
			WillReturnRows(sqlmock.NewRows([]string{"auth_flow_id", "auth_flow_uuid", "tenant_id", "name", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "test flow", now, now))
		repo := NewAuthFlowRepository(gdb)
		result, err := repo.FindByName("test flow")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test flow", result.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "auth_flows" WHERE .*name = \$1`).
			WithArgs("none", 1).
			WillReturnRows(sqlmock.NewRows([]string{"auth_flow_id", "auth_flow_uuid", "tenant_id", "name"}))
		repo := NewAuthFlowRepository(gdb)
		result, err := repo.FindByName("none")
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "auth_flows" WHERE .*name = \$1`).
			WithArgs("test flow", 1).
			WillReturnError(errors.New("db error"))
		repo := NewAuthFlowRepository(gdb)
		result, err := repo.FindByName("test flow")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

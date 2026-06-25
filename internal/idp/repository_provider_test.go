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

func TestNewIdentityProviderRepository(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewIdentityProviderRepository(gdb)
	assert.NotNil(t, repo)
}

func TestIdentityProviderRepository_WithTx(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewIdentityProviderRepository(gdb)
	txRepo := repo.WithTx(gdb)
	assert.NotNil(t, txRepo)
}

func TestIdentityProviderRepository_FindByName(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*name = \$1 AND tenant_id = \$2`).
			WithArgs("test", int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "identity_provider_uuid", "tenant_id", "name", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "test", now, now))
		repo := NewIdentityProviderRepository(gdb)
		result, err := repo.FindByName("test", 1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test", result.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*name = \$1 AND tenant_id = \$2`).
			WithArgs("nonexistent", int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "identity_provider_uuid", "tenant_id", "name"}))
		repo := NewIdentityProviderRepository(gdb)
		result, err := repo.FindByName("nonexistent", 1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*name = \$1 AND tenant_id = \$2`).
			WithArgs("test", int64(1), 1).
			WillReturnError(errors.New("db error"))
		repo := NewIdentityProviderRepository(gdb)
		result, err := repo.FindByName("test", 1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestIdentityProviderRepository_FindByIdentifier(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*identifier = \$1`).
			WithArgs("idp-abc", 1).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "identity_provider_uuid", "tenant_id", "identifier", "name", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "idp-abc", "test", now, now))
		repo := NewIdentityProviderRepository(gdb)
		result, err := repo.FindByIdentifier("idp-abc")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "idp-abc", result.Identifier)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*identifier = \$1`).
			WithArgs("nonexistent", 1).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "identity_provider_uuid", "tenant_id", "identifier"}))
		repo := NewIdentityProviderRepository(gdb)
		result, err := repo.FindByIdentifier("nonexistent")
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*identifier = \$1`).
			WithArgs("idp-abc", 1).
			WillReturnError(errors.New("db error"))
		repo := NewIdentityProviderRepository(gdb)
		result, err := repo.FindByIdentifier("idp-abc")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestIdentityProviderRepository_FindDefaultByTenantID(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*tenant_id = \$1 AND is_default = true`).
			WithArgs(int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "identity_provider_uuid", "tenant_id", "name", "is_default", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "default-idp", true, now, now))
		repo := NewIdentityProviderRepository(gdb)
		result, err := repo.FindDefaultByTenantID(1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsDefault)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*tenant_id = \$1 AND is_default = true`).
			WithArgs(int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "identity_provider_uuid", "tenant_id", "name"}))
		repo := NewIdentityProviderRepository(gdb)
		result, err := repo.FindDefaultByTenantID(1)
		require.Error(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*tenant_id = \$1 AND is_default = true`).
			WithArgs(int64(1), 1).
			WillReturnError(errors.New("db error"))
		repo := NewIdentityProviderRepository(gdb)
		result, err := repo.FindDefaultByTenantID(1)
		require.Error(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestIdentityProviderRepository_FindPaginated(t *testing.T) {
	now := time.Now()

	t.Run("success with tenant filter", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "identity_providers" WHERE .*tenant_id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*tenant_id = \$1.*ORDER BY.*LIMIT`).
			WithArgs(int64(1), 10).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "identity_provider_uuid", "tenant_id", "name", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "idp1", now, now).
				AddRow(2, uuid.New(), int64(1), "idp2", now, now))
		mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE "tenants"\."tenant_id" = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), "tenant1", now, now))
		repo := NewIdentityProviderRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(IdentityProviderRepositoryGetFilter{
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
		mock.ExpectQuery(`SELECT count\(\*\) FROM "identity_providers" WHERE .*tenant_id = \$1`).
			WithArgs(int64(1)).
			WillReturnError(errors.New("db error"))
		repo := NewIdentityProviderRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(IdentityProviderRepositoryGetFilter{
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

	t.Run("success with provider filter", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "identity_providers" WHERE provider IN`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE provider IN.*ORDER BY.*LIMIT`).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "identity_provider_uuid", "tenant_id", "name", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "test", now, now))
		mock.ExpectQuery(`SELECT \* FROM "tenants"`).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name"}).
				AddRow(1, uuid.New(), "tenant1"))

		repo := NewIdentityProviderRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(IdentityProviderRepositoryGetFilter{
			TenantID:  &tenantID,
			Provider:  []string{"google"},
			Page:      1,
			Limit:     10,
			SortBy:    "created_at",
			SortOrder: "desc",
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with status filter", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "identity_providers" WHERE .*status IN`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*status IN.*ORDER BY.*LIMIT`).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "identity_provider_uuid", "tenant_id", "name", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "test", now, now))
		mock.ExpectQuery(`SELECT \* FROM "tenants"`).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name"}).
				AddRow(1, uuid.New(), "tenant1"))

		repo := NewIdentityProviderRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(IdentityProviderRepositoryGetFilter{
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

	t.Run("success with is_default filter", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		defaultVal := true
		mock.ExpectQuery(`SELECT count\(\*\) FROM "identity_providers" WHERE .*is_default`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*is_default.*ORDER BY.*LIMIT`).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "identity_provider_uuid", "tenant_id", "name", "is_default", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "test", true, now, now))
		mock.ExpectQuery(`SELECT \* FROM "tenants"`).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name"}).
				AddRow(1, uuid.New(), "tenant1"))

		repo := NewIdentityProviderRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(IdentityProviderRepositoryGetFilter{
			TenantID:  &tenantID,
			IsDefault: &defaultVal,
			Page:      1,
			Limit:     10,
			SortBy:    "created_at",
			SortOrder: "desc",
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with is_system filter", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		systemVal := false
		mock.ExpectQuery(`SELECT count\(\*\) FROM "identity_providers" WHERE .*is_system`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*is_system.*ORDER BY.*LIMIT`).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "identity_provider_uuid", "tenant_id", "name", "is_system", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "test", false, now, now))
		mock.ExpectQuery(`SELECT \* FROM "tenants"`).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name"}).
				AddRow(1, uuid.New(), "tenant1"))

		repo := NewIdentityProviderRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(IdentityProviderRepositoryGetFilter{
			TenantID:  &tenantID,
			IsSystem:  &systemVal,
			Page:      1,
			Limit:     10,
			SortBy:    "created_at",
			SortOrder: "desc",
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with provider_type filter", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		ptype := "social"
		mock.ExpectQuery(`SELECT count\(\*\) FROM "identity_providers" WHERE provider_type`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE provider_type.*ORDER BY.*LIMIT`).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "identity_provider_uuid", "tenant_id", "name", "provider_type", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "test", "social", now, now))
		mock.ExpectQuery(`SELECT \* FROM "tenants"`).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name"}).
				AddRow(1, uuid.New(), "tenant1"))

		repo := NewIdentityProviderRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(IdentityProviderRepositoryGetFilter{
			TenantID:     &tenantID,
			ProviderType: &ptype,
			Page:         1,
			Limit:        10,
			SortBy:       "created_at",
			SortOrder:    "desc",
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with identifier filter", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		id := "idp-abc"
		mock.ExpectQuery(`SELECT count\(\*\) FROM "identity_providers" WHERE identifier`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE identifier.*ORDER BY.*LIMIT`).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "identity_provider_uuid", "tenant_id", "name", "identifier", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "test", "idp-abc", now, now))
		mock.ExpectQuery(`SELECT \* FROM "tenants"`).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name"}).
				AddRow(1, uuid.New(), "tenant1"))

		repo := NewIdentityProviderRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(IdentityProviderRepositoryGetFilter{
			TenantID:   &tenantID,
			Identifier: &id,
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
		name := "test"
		mock.ExpectQuery(`SELECT count\(\*\) FROM "identity_providers"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT .* FROM "identity_providers"`).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "identity_provider_uuid", "tenant_id", "name", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "test", now, now))
		mock.ExpectQuery(`SELECT \* FROM "tenants"`).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name"}).
				AddRow(1, uuid.New(), "tenant1"))

		repo := NewIdentityProviderRepository(gdb)
		tenantID := int64(1)
		result, err := repo.FindPaginated(IdentityProviderRepositoryGetFilter{
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
}

func TestIdentityProviderRepository_FindByTenantAndProvider(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*tenant_id = \$1 AND provider = \$2 AND deleted_at IS NULL`).
			WithArgs(int64(1), "google", 1).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "identity_provider_uuid", "tenant_id", "provider", "name", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "google", "google-idp", now, now))
		repo := NewIdentityProviderRepository(gdb)
		result, err := repo.FindByTenantAndProvider(1, "google")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "google", result.Provider)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*tenant_id = \$1 AND provider = \$2 AND deleted_at IS NULL`).
			WithArgs(int64(1), "unknown", 1).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "identity_provider_uuid", "tenant_id", "provider", "name"}))
		repo := NewIdentityProviderRepository(gdb)
		result, err := repo.FindByTenantAndProvider(1, "unknown")
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*tenant_id = \$1 AND provider = \$2 AND deleted_at IS NULL`).
			WithArgs(int64(1), "google", 1).
			WillReturnError(errors.New("db error"))
		repo := NewIdentityProviderRepository(gdb)
		result, err := repo.FindByTenantAndProvider(1, "google")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestIdentityProviderRepository_FindAllByTenantID(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*tenant_id = \$1 AND deleted_at IS NULL`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "identity_provider_uuid", "tenant_id", "name", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "idp1", now, now).
				AddRow(2, uuid.New(), int64(1), "idp2", now, now))
		repo := NewIdentityProviderRepository(gdb)
		result, err := repo.FindAllByTenantID(1)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty returns empty slice", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*tenant_id = \$1 AND deleted_at IS NULL`).
			WithArgs(int64(99)).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "identity_provider_uuid", "tenant_id", "name"}))
		repo := NewIdentityProviderRepository(gdb)
		result, err := repo.FindAllByTenantID(99)
		require.NoError(t, err)
		assert.Empty(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*tenant_id = \$1 AND deleted_at IS NULL`).
			WithArgs(int64(1)).
			WillReturnError(errors.New("db error"))
		repo := NewIdentityProviderRepository(gdb)
		result, err := repo.FindAllByTenantID(1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestIdentityProviderRepository_FindByIssuer(t *testing.T) {
	now := time.Now()

	t.Run("found active provider", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*issuer = \$1 AND status = \$2`).
			WithArgs("https://idp.example.com", "active", 1).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "identity_provider_uuid", "tenant_id", "name", "issuer", "status", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "idp1", "https://idp.example.com", "active", now, now))
		repo := NewIdentityProviderRepository(gdb)
		result, err := repo.FindByIssuer("https://idp.example.com")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "https://idp.example.com", result.IssuerOrEmpty())
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*issuer = \$1 AND status = \$2`).
			WithArgs("https://missing.example.com", "active", 1).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id"}))
		repo := NewIdentityProviderRepository(gdb)
		result, err := repo.FindByIssuer("https://missing.example.com")
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*issuer = \$1 AND status = \$2`).
			WithArgs("https://idp.example.com", "active", 1).
			WillReturnError(errors.New("db error"))
		repo := NewIdentityProviderRepository(gdb)
		result, err := repo.FindByIssuer("https://idp.example.com")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestIdentityProviderRepository_FindByUUIDSafe(t *testing.T) {
	now := time.Now()
	idpUUID := uuid.New()

	t.Run("found without preloads", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*identity_provider_uuid = \$1`).
			WithArgs(idpUUID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id", "identity_provider_uuid", "tenant_id", "name", "created_at", "updated_at"}).
				AddRow(1, idpUUID, int64(1), "idp1", now, now))
		repo := NewIdentityProviderRepository(gdb)
		result, err := repo.FindByUUIDSafe(idpUUID)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "idp1", result.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*identity_provider_uuid = \$1`).
			WithArgs(idpUUID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_id"}))
		repo := NewIdentityProviderRepository(gdb)
		result, err := repo.FindByUUIDSafe(idpUUID)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_providers" WHERE .*identity_provider_uuid = \$1`).
			WithArgs(idpUUID, 1).
			WillReturnError(errors.New("db error"))
		repo := NewIdentityProviderRepository(gdb)
		result, err := repo.FindByUUIDSafe(idpUUID)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestIdpSafeColumns_OmitsSecret is the structural guarantee behind the
// write-only secret contract: the safe-columns read list must never include the
// encrypted secret columns, and must include the promoted queried columns.
func TestIdpSafeColumns_OmitsSecret(t *testing.T) {
	for _, c := range idpSafeColumns {
		assert.NotEqual(t, "provider_client_secret_encrypted", c)
		assert.NotContains(t, c, "secret")
	}
	assert.Contains(t, idpSafeColumns, "issuer")
	assert.Contains(t, idpSafeColumns, "provider_client_id")
	assert.Contains(t, idpSafeColumns, "allow_jit_provisioning")
}

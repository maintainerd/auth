package branding

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginTemplateRepository_FindByUUIDAndTenantID(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "login_templates" WHERE .*login_template_uuid = \$1.*tenant_id = \$2.*AND "login_templates"\."deleted_at" IS NULL`).
			WithArgs(id, int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"login_template_id", "login_template_uuid", "tenant_id", "name", "template", "status", "is_default", "created_at", "updated_at"}).
				AddRow(1, id, int64(1), "Default", "v1", "active", true, now, now))
		repo := NewLoginTemplateRepository(gdb)
		result, err := repo.FindByUUIDAndTenantID(id, 1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, id, result.LoginTemplateUUID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "login_templates" WHERE .*login_template_uuid = \$1.*tenant_id = \$2.*AND "login_templates"\."deleted_at" IS NULL`).
			WithArgs(id, int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"login_template_id", "login_template_uuid", "tenant_id", "name"}))
		repo := NewLoginTemplateRepository(gdb)
		result, err := repo.FindByUUIDAndTenantID(id, 1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "login_templates" WHERE .*login_template_uuid = \$1.*tenant_id = \$2`).
			WithArgs(id, int64(1), 1).
			WillReturnError(assert.AnError)
		repo := NewLoginTemplateRepository(gdb)
		result, err := repo.FindByUUIDAndTenantID(id, 1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with preloads", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "login_templates" WHERE .*login_template_uuid = \$1.*tenant_id = \$2.*AND "login_templates"\."deleted_at" IS NULL`).
			WithArgs(id, int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"login_template_id", "login_template_uuid", "tenant_id", "name", "status", "created_at", "updated_at"}).
				AddRow(1, id, 1, "Test", "active", now, now))
		repo := NewLoginTemplateRepository(gdb)
		_, _ = repo.FindByUUIDAndTenantID(id, 1, "any_relation")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNewLoginTemplateRepository(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewLoginTemplateRepository(gdb)
	assert.NotNil(t, repo)
}

func TestLoginTemplateRepository_FindPaginated(t *testing.T) {
	gdb, mock := newMockGormDBRegex(t)
	mock.ExpectQuery(`SELECT count\(\*\) FROM "login_templates"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT \* FROM "login_templates"`).
		WillReturnRows(sqlmock.NewRows([]string{"login_template_id", "login_template_uuid", "name", "status", "created_at", "updated_at"}))

	repo := NewLoginTemplateRepository(gdb)
	result, err := repo.FindPaginated(LoginTemplateRepositoryGetFilter{Page: 1, Limit: 10})

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoginTemplateRepository_FindPaginated_WithFilters(t *testing.T) {
	now := time.Now()

	gdb, mock := newMockGormDBRegex(t)
	mock.ExpectQuery(`SELECT count\(\*\) FROM "login_templates"`).
		WithArgs("%modern%", "active", "modern", int64(1), true, false).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT \* FROM "login_templates"`).
		WithArgs("%modern%", "active", "modern", int64(1), true, false, 10).
		WillReturnRows(sqlmock.NewRows([]string{"login_template_id", "login_template_uuid", "name", "status", "created_at", "updated_at"}).
			AddRow(1, uuid.New(), "modern", "active", now, now))

	name := "modern"
	template := "modern"
	active := true
	sys := false
	repo := NewLoginTemplateRepository(gdb)
	result, err := repo.FindPaginated(LoginTemplateRepositoryGetFilter{
		Name:      &name,
		Status:    []string{"active"},
		Template:  &template,
		TenantID:  ptrI64(1),
		IsDefault: &active,
		IsSystem:  &sys,
		Page:      1,
		Limit:     10,
	})

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

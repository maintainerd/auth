package branding

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailTemplateRepository_FindByUUIDAndTenantID(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "email_templates" WHERE .*email_template_uuid = \$1.*tenant_id = \$2.*AND "email_templates"\."deleted_at" IS NULL`).
			WithArgs(id, int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"email_template_id", "email_template_uuid", "tenant_id", "name", "subject", "body_html", "status", "is_default", "created_at", "updated_at"}).
				AddRow(1, id, int64(1), "Welcome", "Hello", "<p>Hi</p>", "active", true, now, now))
		repo := NewEmailTemplateRepository(gdb)
		result, err := repo.FindByUUIDAndTenantID(id, 1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, id, result.EmailTemplateUUID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "email_templates" WHERE .*email_template_uuid = \$1.*tenant_id = \$2.*AND "email_templates"\."deleted_at" IS NULL`).
			WithArgs(id, int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"email_template_id", "email_template_uuid", "tenant_id", "name"}))
		repo := NewEmailTemplateRepository(gdb)
		result, err := repo.FindByUUIDAndTenantID(id, 1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "email_templates" WHERE .*email_template_uuid = \$1.*tenant_id = \$2`).
			WithArgs(id, int64(1), 1).
			WillReturnError(assert.AnError)
		repo := NewEmailTemplateRepository(gdb)
		result, err := repo.FindByUUIDAndTenantID(id, 1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestEmailTemplateRepository_FindByName(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "email_templates" WHERE .*name = \$1.*status = \$2.*AND "email_templates"\."deleted_at" IS NULL`).
			WithArgs("Welcome", shared.StatusActive, 1).
			WillReturnRows(sqlmock.NewRows([]string{"email_template_id", "email_template_uuid", "tenant_id", "name", "subject", "body_html", "status", "is_default", "created_at", "updated_at"}).
				AddRow(1, id, int64(1), "Welcome", "Hello", "<p>Hi</p>", "active", true, now, now))
		repo := NewEmailTemplateRepository(gdb)
		result, err := repo.FindByName("Welcome")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Welcome", result.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "email_templates" WHERE .*name = \$1.*status = \$2.*AND "email_templates"\."deleted_at" IS NULL`).
			WithArgs("Missing", shared.StatusActive, 1).
			WillReturnRows(sqlmock.NewRows([]string{"email_template_id", "email_template_uuid", "tenant_id", "name"}))
		repo := NewEmailTemplateRepository(gdb)
		result, err := repo.FindByName("Missing")
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "email_templates" WHERE .*name = \$1.*status = \$2`).
			WithArgs("Welcome", shared.StatusActive, 1).
			WillReturnError(assert.AnError)
		repo := NewEmailTemplateRepository(gdb)
		result, err := repo.FindByName("Welcome")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNewEmailTemplateRepository(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewEmailTemplateRepository(gdb)
	assert.NotNil(t, repo)
}

func TestEmailTemplateRepository_FindPaginated(t *testing.T) {
	now := time.Now()

	t.Run("success with filters", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "email_templates"`).
			WithArgs("%test%", "active", int64(1), true, false).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "email_templates"`).
			WithArgs("%test%", "active", int64(1), true, false, 10).
			WillReturnRows(sqlmock.NewRows([]string{"email_template_id", "email_template_uuid", "name", "status", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), "Test", "active", now, now))

		name := "test"
		active := true
		sys := false
		repo := NewEmailTemplateRepository(gdb)
		result, err := repo.FindPaginated(EmailTemplateRepositoryGetFilter{
			Name:      &name,
			Status:    []string{"active"},
			TenantID:  ptrI64(1),
			IsDefault: &active,
			IsSystem:  &sys,
			Page:      1,
			Limit:     10,
			SortBy:    "created_at",
			SortOrder: "DESC",
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(1), result.Total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "email_templates"`).
			WillReturnError(assert.AnError)

		repo := NewEmailTemplateRepository(gdb)
		_, err := repo.FindPaginated(EmailTemplateRepositoryGetFilter{Page: 1, Limit: 10})

		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func ptrI64(v int64) *int64 { return &v }

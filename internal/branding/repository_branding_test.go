package branding

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrandingRepository_FindByTenantID(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "branding" WHERE .*tenant_id = \$1.*AND "branding"\."deleted_at" IS NULL`).
			WithArgs(int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"branding_id", "branding_uuid", "tenant_id", "company_name", "created_at", "updated_at"}).
				AddRow(1, uuid.New(), int64(1), "Acme Corp", now, now))
		repo := NewBrandingRepository(gdb)
		result, err := repo.FindByTenantID(1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Acme Corp", result.CompanyName)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "branding" WHERE .*tenant_id = \$1.*AND "branding"\."deleted_at" IS NULL`).
			WithArgs(int64(2), 1).
			WillReturnRows(sqlmock.NewRows([]string{"branding_id", "branding_uuid", "tenant_id", "company_name"}))
		repo := NewBrandingRepository(gdb)
		result, err := repo.FindByTenantID(2)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "branding" WHERE .*tenant_id = \$1`).
			WithArgs(int64(1), 1).
			WillReturnError(assert.AnError)
		repo := NewBrandingRepository(gdb)
		result, err := repo.FindByTenantID(1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNewBrandingRepository(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewBrandingRepository(gdb)
	assert.NotNil(t, repo)
}

func TestBrandingRepository_WithTx(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewBrandingRepository(gdb)
	// WithTx should return a non-nil repo bound to the transaction
	txRepo := repo.WithTx(gdb)
	assert.NotNil(t, txRepo)
}

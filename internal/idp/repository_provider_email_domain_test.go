package idp

import (
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentityProviderEmailDomainRepository_WithTx(t *testing.T) {
	gdb, mock := newMockGormDBRegex(t)
	mock.ExpectQuery(`SELECT .* FROM "identity_provider_email_domains" WHERE .*identity_provider_id = \$1 AND deleted_at IS NULL`).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"identity_provider_email_domain_id"}))
	repo := NewIdentityProviderEmailDomainRepository(gdb).WithTx(gdb)
	_, err := repo.FindByProviderID(9)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIdentityProviderEmailDomainRepository_FindByTenantAndDomain(t *testing.T) {
	now := time.Now()

	t.Run("found", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_provider_email_domains" WHERE .*tenant_id = \$1 AND domain = \$2`).
			WithArgs(int64(1), "example.com", 1).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_email_domain_id", "tenant_id", "identity_provider_id", "domain", "created_at"}).
				AddRow(int64(5), int64(1), int64(9), "example.com", now))
		repo := NewIdentityProviderEmailDomainRepository(gdb)
		result, err := repo.FindByTenantAndDomain(1, "Example.com")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(9), result.IdentityProviderID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_provider_email_domains" WHERE .*tenant_id = \$1 AND domain = \$2`).
			WithArgs(int64(1), "missing.com", 1).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_email_domain_id"}))
		repo := NewIdentityProviderEmailDomainRepository(gdb)
		result, err := repo.FindByTenantAndDomain(1, "missing.com")
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_provider_email_domains" WHERE .*tenant_id = \$1 AND domain = \$2`).
			WithArgs(int64(1), "example.com", 1).
			WillReturnError(errors.New("db error"))
		repo := NewIdentityProviderEmailDomainRepository(gdb)
		result, err := repo.FindByTenantAndDomain(1, "example.com")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestIdentityProviderEmailDomainRepository_FindByProviderID(t *testing.T) {
	now := time.Now()

	t.Run("found", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_provider_email_domains" WHERE .*identity_provider_id = \$1 AND deleted_at IS NULL`).
			WithArgs(int64(9)).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_email_domain_id", "tenant_id", "identity_provider_id", "domain", "created_at"}).
				AddRow(int64(5), int64(1), int64(9), "example.com", now).
				AddRow(int64(6), int64(1), int64(9), "foo.com", now))
		repo := NewIdentityProviderEmailDomainRepository(gdb)
		result, err := repo.FindByProviderID(9)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT .* FROM "identity_provider_email_domains" WHERE .*identity_provider_id = \$1 AND deleted_at IS NULL`).
			WithArgs(int64(9)).
			WillReturnError(errors.New("db error"))
		repo := NewIdentityProviderEmailDomainRepository(gdb)
		_, err := repo.FindByProviderID(9)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestIdentityProviderEmailDomainRepository_ReplaceForProvider(t *testing.T) {
	t.Run("deletes then inserts normalized deduped domains", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		// Hard delete of the existing set.
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "identity_provider_email_domains" WHERE identity_provider_id = \$1`).
			WithArgs(int64(9)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		// Insert of the new set (RETURNING the new ids).
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO "identity_provider_email_domains"`).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_email_domain_id"}).AddRow(int64(1)).AddRow(int64(2)))
		mock.ExpectCommit()
		repo := NewIdentityProviderEmailDomainRepository(gdb)
		// Duplicates / case / whitespace are normalized to two rows.
		err := repo.ReplaceForProvider(1, 9, []string{"Example.com", " example.com ", "foo.com", ""})
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty set only deletes", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "identity_provider_email_domains" WHERE identity_provider_id = \$1`).
			WithArgs(int64(9)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		repo := NewIdentityProviderEmailDomainRepository(gdb)
		err := repo.ReplaceForProvider(1, 9, nil)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("delete error propagates", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "identity_provider_email_domains" WHERE identity_provider_id = \$1`).
			WithArgs(int64(9)).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()
		repo := NewIdentityProviderEmailDomainRepository(gdb)
		err := repo.ReplaceForProvider(1, 9, []string{"example.com"})
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

package idp

import (
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentityProviderAllowedAudienceRepository_ReplaceForProvider(t *testing.T) {
	t.Run("deletes then inserts normalized deduped audiences", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "identity_provider_allowed_audiences" WHERE identity_provider_id = \$1`).
			WithArgs(int64(20)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO "identity_provider_allowed_audiences"`).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_allowed_audience_id"}).AddRow(int64(1)).AddRow(int64(2)))
		mock.ExpectCommit()
		repo := NewIdentityProviderAllowedAudienceRepository(gdb)
		err := repo.ReplaceForProvider(10, 20, []string{"app-1", "  app-2  ", "", "app-1"})
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty set only deletes", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "identity_provider_allowed_audiences" WHERE identity_provider_id = \$1`).
			WithArgs(int64(20)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		repo := NewIdentityProviderAllowedAudienceRepository(gdb)
		err := repo.ReplaceForProvider(10, 20, nil)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("delete error propagates", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "identity_provider_allowed_audiences" WHERE identity_provider_id = \$1`).
			WithArgs(int64(20)).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()
		repo := NewIdentityProviderAllowedAudienceRepository(gdb)
		err := repo.ReplaceForProvider(10, 20, []string{"app-1"})
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestIdentityProviderAllowedAudienceRepository_FindByProviderID(t *testing.T) {
	t.Run("returns attached audiences", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "identity_provider_allowed_audiences" WHERE`).
			WithArgs(int64(20)).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_allowed_audience_id", "identity_provider_id", "audience"}).
				AddRow(1, 20, "app-1").
				AddRow(2, 20, "app-2"))
		repo := NewIdentityProviderAllowedAudienceRepository(gdb)
		audiences, err := repo.FindByProviderID(20)
		require.NoError(t, err)
		require.Len(t, audiences, 2)
		assert.Equal(t, "app-1", audiences[0].Audience)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty when none attached", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "identity_provider_allowed_audiences" WHERE`).
			WithArgs(int64(99)).
			WillReturnRows(sqlmock.NewRows([]string{"identity_provider_allowed_audience_id", "identity_provider_id", "audience"}))
		repo := NewIdentityProviderAllowedAudienceRepository(gdb)
		audiences, err := repo.FindByProviderID(99)
		require.NoError(t, err)
		assert.Empty(t, audiences)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestIdentityProviderAllowedAudienceRepository_WithTx(t *testing.T) {
	gdb, _ := newMockGormDB(t)
	repo := NewIdentityProviderAllowedAudienceRepository(gdb)
	txRepo := repo.WithTx(gdb)
	assert.NotNil(t, txRepo)
}

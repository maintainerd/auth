package tenant

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantSettingRepository_FindByTenantID(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "tenant_settings" WHERE tenant_id = \$1 ORDER BY "tenant_settings"."tenant_setting_id" LIMIT \$2`).
			WithArgs(int64(10), 1).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_setting_id", "tenant_setting_uuid", "tenant_id", "rate_limit_config", "audit_config", "maintenance_config", "feature_flags", "created_at", "updated_at"}).
				AddRow(1, testResourceUUID, 10, []byte(`{"max":100}`), []byte(`{"enabled":true}`), []byte(`{"active":false}`), []byte(`{"beta":true}`), now, now))

		result, err := NewTenantSettingRepository(db).FindByTenantID(10)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(10), result.TenantID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "tenant_settings" WHERE tenant_id = \$1 ORDER BY "tenant_settings"."tenant_setting_id" LIMIT \$2`).
			WithArgs(int64(10), 1).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_setting_id", "tenant_setting_uuid"}))

		result, err := NewTenantSettingRepository(db).FindByTenantID(10)

		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "tenant_settings" WHERE tenant_id = \$1`).
			WithArgs(int64(10), 1).
			WillReturnError(assert.AnError)

		result, err := NewTenantSettingRepository(db).FindByTenantID(10)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTenantSettingRepository_WithTx(t *testing.T) {
	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()
	tx := db.Begin()
	require.NoError(t, tx.Error)

	repo := NewTenantSettingRepository(db).WithTx(tx)

	assert.NotNil(t, repo)
	require.NoError(t, tx.Rollback().Error)
	assert.NoError(t, mock.ExpectationsWereMet())
}

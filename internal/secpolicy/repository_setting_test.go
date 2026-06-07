package secpolicy

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newSecpolicyMockGormDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return db, mock
}

func TestSecuritySettingRepository_FindByTenantID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "security_settings" WHERE tenant_id = \$1 ORDER BY "security_settings"\."security_setting_id" LIMIT \$2`).
			WithArgs(int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"security_setting_id", "security_setting_uuid", "tenant_id"}).
				AddRow(1, testResourceUUID, 1))

		result, err := NewSecuritySettingRepository(db).FindByTenantID(1)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(1), result.TenantID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "security_settings" WHERE tenant_id = \$1`).
			WithArgs(int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"security_setting_id", "security_setting_uuid", "tenant_id"}))

		result, err := NewSecuritySettingRepository(db).FindByTenantID(1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "security_settings" WHERE tenant_id = \$1`).
			WithArgs(int64(1), 1).
			WillReturnError(assert.AnError)

		result, err := NewSecuritySettingRepository(db).FindByTenantID(1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
func TestSecuritySettingRepository_FindPaginated(t *testing.T) {
	t.Run("with TenantID filter", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		tenantID := int64(1)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "security_settings" WHERE tenant_id = \$1`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "security_settings" WHERE tenant_id = \$1 ORDER BY created_at DESC LIMIT \$2`).
			WithArgs(tenantID, 10).
			WillReturnRows(sqlmock.NewRows([]string{"security_setting_id", "security_setting_uuid", "tenant_id"}).
				AddRow(1, testResourceUUID, tenantID))

		result, err := NewSecuritySettingRepository(db).FindPaginated(SecuritySettingRepositoryGetFilter{
			TenantID: &tenantID,
			Page:       1,
			Limit:      10,
		})
		require.NoError(t, err)
		require.Len(t, result.Data, 1)
		assert.Equal(t, tenantID, result.Data[0].TenantID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with all filters", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		tenantID := int64(1)
		version := 2
		createdBy := int64(10)
		updatedBy := int64(20)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "security_settings" WHERE tenant_id = \$1 AND version = \$2 AND created_by = \$3 AND updated_by = \$4`).
			WithArgs(tenantID, version, createdBy, updatedBy).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "security_settings" WHERE tenant_id = \$1 AND version = \$2 AND created_by = \$3 AND updated_by = \$4 ORDER BY created_at DESC LIMIT \$5`).
			WithArgs(tenantID, version, createdBy, updatedBy, 10).
			WillReturnRows(sqlmock.NewRows([]string{"security_setting_id", "security_setting_uuid", "tenant_id"}).
				AddRow(1, testResourceUUID, tenantID))

		result, err := NewSecuritySettingRepository(db).FindPaginated(SecuritySettingRepositoryGetFilter{
			TenantID: &tenantID,
			Version:    &version,
			CreatedBy:  &createdBy,
			UpdatedBy:  &updatedBy,
			Page:       1,
			Limit:      10,
		})
		require.NoError(t, err)
		require.Len(t, result.Data, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSecuritySettingRepository_Mutations(t *testing.T) {
	t.Run("WithTx returns tx-bound repository", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tx := db.Begin()
		require.NoError(t, tx.Error)

		repo := NewSecuritySettingRepository(db).WithTx(tx)
		assert.NotNil(t, repo)
		require.NoError(t, tx.Rollback().Error)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("NewSecuritySettingRepository returns non-nil", func(t *testing.T) {
		db, _ := newSecpolicyMockGormDB(t)
		repo := NewSecuritySettingRepository(db)
		assert.NotNil(t, repo)
	})

	t.Run("IncrementVersion success", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "security_settings" SET "version"=version \+ \$1 WHERE security_setting_id = \$2`).
			WithArgs(1, int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := NewSecuritySettingRepository(db).IncrementVersion(1)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("IncrementVersion error", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "security_settings" SET "version"=version \+ \$1 WHERE security_setting_id = \$2`).
			WithArgs(1, int64(1)).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := NewSecuritySettingRepository(db).IncrementVersion(1)
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

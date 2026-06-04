package secpolicy

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecuritySettingsAuditRepository_FindBySecuritySettingID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "security_settings_audit" WHERE security_setting_id = \$1 ORDER BY created_at DESC`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"security_settings_audit_id", "security_settings_audit_uuid", "user_pool_id", "security_setting_id", "change_type"}).
				AddRow(1, testResourceUUID, 1, 1, "update_mfa_config"))

		result, err := NewSecuritySettingsAuditRepository(db).FindBySecuritySettingID(1)
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "update_mfa_config", result[0].ChangeType)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "security_settings_audit" WHERE security_setting_id = \$1`).
			WithArgs(int64(1)).
			WillReturnError(assert.AnError)

		result, err := NewSecuritySettingsAuditRepository(db).FindBySecuritySettingID(1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSecuritySettingsAuditRepository_FindByUserPoolID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "security_settings_audit" WHERE user_pool_id = \$1 ORDER BY created_at DESC`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"security_settings_audit_id", "security_settings_audit_uuid", "user_pool_id", "security_setting_id", "change_type"}).
				AddRow(1, testResourceUUID, 1, 1, "update_mfa_config"))

		result, err := NewSecuritySettingsAuditRepository(db).FindByUserPoolID(1)
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "update_mfa_config", result[0].ChangeType)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "security_settings_audit" WHERE user_pool_id = \$1`).
			WithArgs(int64(1)).
			WillReturnError(assert.AnError)

		result, err := NewSecuritySettingsAuditRepository(db).FindByUserPoolID(1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSecuritySettingsAuditRepository_FindPaginated(t *testing.T) {
	t.Run("with UserPoolID filter", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		userPoolID := int64(1)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "security_settings_audit" WHERE user_pool_id = \$1`).
			WithArgs(userPoolID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "security_settings_audit" WHERE user_pool_id = \$1 ORDER BY created_at DESC LIMIT \$2`).
			WithArgs(userPoolID, 10).
			WillReturnRows(sqlmock.NewRows([]string{"security_settings_audit_id", "security_settings_audit_uuid", "user_pool_id", "security_setting_id", "change_type"}).
				AddRow(1, testResourceUUID, userPoolID, 1, "update_mfa_config"))

		result, err := NewSecuritySettingsAuditRepository(db).FindPaginated(SecuritySettingsAuditRepositoryGetFilter{
			UserPoolID: &userPoolID,
			Page:       1,
			Limit:      10,
		})
		require.NoError(t, err)
		require.Len(t, result.Data, 1)
		assert.Equal(t, "update_mfa_config", result.Data[0].ChangeType)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with all filters", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		userPoolID := int64(1)
		securitySettingID := int64(5)
		changeType := "update_mfa_config"
		createdBy := int64(10)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "security_settings_audit" WHERE user_pool_id = \$1 AND security_setting_id = \$2 AND change_type = \$3 AND created_by = \$4`).
			WithArgs(userPoolID, securitySettingID, changeType, createdBy).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "security_settings_audit" WHERE user_pool_id = \$1 AND security_setting_id = \$2 AND change_type = \$3 AND created_by = \$4 ORDER BY created_at DESC LIMIT \$5`).
			WithArgs(userPoolID, securitySettingID, changeType, createdBy, 10).
			WillReturnRows(sqlmock.NewRows([]string{"security_settings_audit_id", "security_settings_audit_uuid", "user_pool_id", "security_setting_id", "change_type"}).
				AddRow(1, testResourceUUID, userPoolID, securitySettingID, changeType))

		result, err := NewSecuritySettingsAuditRepository(db).FindPaginated(SecuritySettingsAuditRepositoryGetFilter{
			UserPoolID:        &userPoolID,
			SecuritySettingID: &securitySettingID,
			ChangeType:        &changeType,
			CreatedBy:         &createdBy,
			Page:              1,
			Limit:             10,
		})
		require.NoError(t, err)
		require.Len(t, result.Data, 1)
		assert.Equal(t, changeType, result.Data[0].ChangeType)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSecuritySettingsAuditRepository_Mutations(t *testing.T) {
	t.Run("WithTx returns tx-bound repository", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tx := db.Begin()
		require.NoError(t, tx.Error)

		repo := NewSecuritySettingsAuditRepository(db).WithTx(tx)
		assert.NotNil(t, repo)
		require.NoError(t, tx.Rollback().Error)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("NewSecuritySettingsAuditRepository returns non-nil", func(t *testing.T) {
		db, _ := newSecpolicyMockGormDB(t)
		repo := NewSecuritySettingsAuditRepository(db)
		assert.NotNil(t, repo)
	})
}

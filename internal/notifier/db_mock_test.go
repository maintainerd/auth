package notifier

import (
	"database/sql/driver"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newMockGormDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return db, mock
}

func expectNotifierSelect(mock sqlmock.Sqlmock, table string) *sqlmock.ExpectedQuery {
	return mock.ExpectQuery(`SELECT .* FROM "` + table + `".*`)
}

func expectNotifierUpdate(mock sqlmock.Sqlmock, table string) *sqlmock.ExpectedExec {
	return mock.ExpectExec(`UPDATE "` + table + `".*`)
}

func emailConfigRows(values ...driver.Value) *sqlmock.Rows {
	if len(values) == 0 {
		values = []driver.Value{
			int64(1), testTenantUUID.String(), testTenantID, "smtp", "smtp.example.com", 587,
			"user", "secret", "noreply@example.com", "Maintainerd", "reply@example.com",
			"tls", false, "active", []byte(`{}`), nil, nil, time.Now(), time.Now(), nil,
		}
	}
	return sqlmock.NewRows([]string{
		"email_config_id", "email_config_uuid", "tenant_id", "provider", "host", "port",
		"username", "password_encrypted", "from_address", "from_name", "reply_to",
		"encryption", "test_mode", "status", "metadata", "created_by", "updated_by",
		"created_at", "updated_at", "deleted_at",
	}).AddRow(values...)
}

func smsConfigRows(values ...driver.Value) *sqlmock.Rows {
	if len(values) == 0 {
		values = []driver.Value{
			int64(1), testTenantUUID.String(), testTenantID, "twilio", "AC123", "token",
			"+15551234567", "sender", false, "active", []byte(`{}`), nil, nil,
			time.Now(), time.Now(), nil,
		}
	}
	return sqlmock.NewRows([]string{
		"sms_config_id", "sms_config_uuid", "tenant_id", "provider", "account_sid",
		"auth_token_encrypted", "from_number", "sender_id", "test_mode", "status",
		"metadata", "created_by", "updated_by", "created_at", "updated_at", "deleted_at",
	}).AddRow(values...)
}

func userOTPRows(values ...driver.Value) *sqlmock.Rows {
	if len(values) == 0 {
		values = []driver.Value{
			int64(1), testTenantUUID.String(), int64(42), "sms", "+15551234567", "hash",
			time.Now().Add(time.Minute), false, 0, time.Now(),
		}
	}
	return sqlmock.NewRows([]string{
		"user_otp_id", "user_otp_uuid", "user_id", "channel", "recipient", "otp_hash",
		"expires_at", "used", "failed_attempts", "created_at",
	}).AddRow(values...)
}

func assertNotifierExpectationsMet(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	require.NoError(t, mock.ExpectationsWereMet())
}

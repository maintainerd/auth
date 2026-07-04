package mfa

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
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

func expectMFASelect(mock sqlmock.Sqlmock, table string) *sqlmock.ExpectedQuery {
	return mock.ExpectQuery(`SELECT .* FROM "` + table + `".*`)
}

func expectMFAUpdate(mock sqlmock.Sqlmock, table string) *sqlmock.ExpectedExec {
	return mock.ExpectExec(`UPDATE "` + table + `".*`)
}

func expectMFADelete(mock sqlmock.Sqlmock, table string) *sqlmock.ExpectedExec {
	return mock.ExpectExec(`DELETE FROM "` + table + `".*`)
}

func expectUserTenantIDLookup(mock sqlmock.Sqlmock, tenantID int64) *sqlmock.ExpectedQuery {
	return mock.ExpectQuery(`SELECT .* FROM "user_identities"`).
		WithArgs(mfaTestUserID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id"}).AddRow(tenantID))
}

func userBackupCodeRows(values ...driver.Value) *sqlmock.Rows {
	if len(values) == 0 {
		values = []driver.Value{int64(1), mfaTestCredentialUUID.String(), mfaTestUserID, "hash", false, nil, time.Now()}
	}
	return sqlmock.NewRows([]string{
		"backup_code_id", "backup_code_uuid", "user_id", "code_hash", "used", "used_at", "created_at",
	}).AddRow(values...)
}

func userTOTPSecretRows(values ...driver.Value) *sqlmock.Rows {
	if len(values) == 0 {
		step := int64(10)
		now := time.Now()
		values = []driver.Value{int64(1), mfaTestCredentialUUID.String(), mfaTestUserID, "secret", true, &now, &now, &step, now, now}
	}
	return sqlmock.NewRows([]string{
		"totp_secret_id", "totp_secret_uuid", "user_id", "secret", "is_enabled", "enrolled_at", "last_used_at", "last_used_step", "created_at", "updated_at",
	}).AddRow(values...)
}

func userWebAuthnCredentialRows(values ...driver.Value) *sqlmock.Rows {
	if len(values) == 0 {
		values = []driver.Value{int64(1), mfaTestCredentialUUID.String(), mfaTestUserID, "cred-key", []byte("public"), nil, int64(1), pq.StringArray{"usb"}, true, false, "Security Key", nil, time.Now(), time.Now()}
	}
	return sqlmock.NewRows([]string{
		"credential_id", "credential_uuid", "user_id", "credential_key_id", "public_key", "aaguid", "sign_count", "transport", "is_backup_eligible", "is_backup_state", "name", "last_used_at", "created_at", "updated_at",
	}).AddRow(values...)
}

type mockAuthEventService struct {
	inputs []authevent.AuthEventInput
}

func (m *mockAuthEventService) Log(_ context.Context, input authevent.AuthEventInput) {
	m.inputs = append(m.inputs, input)
}

func (m *mockAuthEventService) FindPaginated(context.Context, authevent.AuthEventRepositoryGetFilter) (*PaginationResult[authevent.AuthEventServiceDataResult], error) {
	return nil, nil
}

func (m *mockAuthEventService) FindByUUID(context.Context, int64, uuid.UUID) (*authevent.AuthEventServiceDataResult, error) {
	return nil, nil
}

func (m *mockAuthEventService) CountByEventType(context.Context, string, int64) (int64, error) {
	return 0, nil
}

func (m *mockAuthEventService) DeleteOlderThan(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func (m *mockAuthEventService) Shutdown() {}

func assertExpectationsMet(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	require.NoError(t, mock.ExpectationsWereMet())
}

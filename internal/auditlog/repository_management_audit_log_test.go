package auditlog

import (
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newAuditLogMockGormDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	return db, mock
}

func auditLogRows(entryUUID uuid.UUID, actorName string, clientName string) *sqlmock.Rows {
	now := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
	return sqlmock.NewRows([]string{
		"management_audit_log_id",
		"management_audit_log_uuid",
		"tenant_id",
		"actor_user_id",
		"actor_client_id",
		"action",
		"resource_type",
		"resource_id",
		"changes",
		"outcome",
		"created_at",
		"actor_user_name",
		"actor_client_name",
	}).AddRow(
		int64(1),
		entryUUID,
		int64(42),
		int64(7),
		nil,
		"user.updated",
		"user",
		"user-123",
		[]byte(`{}`),
		"success",
		now,
		actorName,
		clientName,
	)
}

func TestManagementAuditLogRepository_FindByUUIDAndTenantIDIncludesActorLabels(t *testing.T) {
	db, mock := newAuditLogMockGormDB(t)
	entryUUID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)+`(?s).*actor_user_name.*actor_client_name.*FROM "management_audit_log".*LEFT JOIN users.*LEFT JOIN profiles.*LEFT JOIN clients.*WHERE.*management_audit_log_uuid.*tenant_id.*ORDER BY.*LIMIT`).
		WithArgs(entryUUID, int64(42), 1).
		WillReturnRows(auditLogRows(entryUUID, "Jane Admin", ""))

	got, err := NewManagementAuditLogRepository(db).FindByUUIDAndTenantID(entryUUID, 42)

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, entryUUID, got.ManagementAuditLogUUID)
	require.NotNil(t, got.ActorUserName)
	assert.Equal(t, "Jane Admin", *got.ActorUserName)
}

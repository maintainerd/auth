package authevent

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newAuthEventMockGormDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return db, mock
}

func authEventRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"auth_event_id",
		"auth_event_uuid",
		"tenant_id",
		"ip_address",
		"category",
		"event_type",
		"severity",
		"result",
		"metadata",
		"created_at",
	})
}

func TestAuthEventRepository_FindPaginated(t *testing.T) {
	db, mock := newAuthEventMockGormDB(t)
	tenantID := int64(1)
	actorID := int64(2)
	targetID := int64(3)
	category := AuthEventCategoryAuthn
	eventType := AuthEventTypeLoginSuccess
	severity := AuthEventSeverityInfo
	resultValue := AuthEventResultSuccess
	from := time.Now().Add(-time.Hour).UTC()
	to := time.Now().UTC()
	eventUUID := uuid.New()

	mock.ExpectQuery(`SELECT count\(\*\) FROM "auth_events" WHERE tenant_id = \$1 AND actor_user_id = \$2 AND target_user_id = \$3 AND category = \$4 AND event_type = \$5 AND severity = \$6 AND result = \$7 AND created_at >= \$8 AND created_at <= \$9`).
		WithArgs(tenantID, actorID, targetID, category, eventType, severity, resultValue, from, to).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT \* FROM "auth_events" WHERE tenant_id = \$1 AND actor_user_id = \$2 AND target_user_id = \$3 AND category = \$4 AND event_type = \$5 AND severity = \$6 AND result = \$7 AND created_at >= \$8 AND created_at <= \$9 ORDER BY event_type ASC LIMIT \$10`).
		WithArgs(tenantID, actorID, targetID, category, eventType, severity, resultValue, from, to, 10).
		WillReturnRows(authEventRows().AddRow(1, eventUUID, tenantID, "10.0.0.1", category, eventType, severity, resultValue, []byte(`{}`), to))

	repo := NewAuthEventRepository(db)
	page, err := repo.FindPaginated(AuthEventRepositoryGetFilter{
		TenantID:     &tenantID,
		ActorUserID:  &actorID,
		TargetUserID: &targetID,
		Category:     &category,
		EventType:    &eventType,
		Severity:     &severity,
		Result:       &resultValue,
		DateFrom:     &from,
		DateTo:       &to,
		SortBy:       "event_type",
		SortOrder:    "asc",
		Page:         1,
		Limit:        10,
	})

	require.NoError(t, err)
	require.Len(t, page.Data, 1)
	assert.Equal(t, eventUUID, page.Data[0].AuthEventUUID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthEventRepository_FindByUUIDAndTenantID(t *testing.T) {
	eventUUID := uuid.New()
	now := time.Now().UTC()

	t.Run("success", func(t *testing.T) {
		db, mock := newAuthEventMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "auth_events" WHERE auth_event_uuid = \$1 AND tenant_id = \$2 ORDER BY "auth_events"."auth_event_id" LIMIT \$3`).
			WithArgs(eventUUID.String(), int64(1), 1).
			WillReturnRows(authEventRows().AddRow(1, eventUUID, 1, "10.0.0.1", AuthEventCategoryAuthn, AuthEventTypeLoginSuccess, AuthEventSeverityInfo, AuthEventResultSuccess, []byte(`{}`), now))

		result, err := NewAuthEventRepository(db).FindByUUIDAndTenantID(eventUUID.String(), 1)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, eventUUID, result.AuthEventUUID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		db, mock := newAuthEventMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "auth_events" WHERE auth_event_uuid = \$1 AND tenant_id = \$2 ORDER BY "auth_events"."auth_event_id" LIMIT \$3`).
			WithArgs(eventUUID.String(), int64(1), 1).
			WillReturnRows(authEventRows())

		result, err := NewAuthEventRepository(db).FindByUUIDAndTenantID(eventUUID.String(), 1)

		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newAuthEventMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "auth_events" WHERE auth_event_uuid = \$1 AND tenant_id = \$2`).
			WithArgs(eventUUID.String(), int64(1), 1).
			WillReturnError(assert.AnError)

		result, err := NewAuthEventRepository(db).FindByUUIDAndTenantID(eventUUID.String(), 1)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAuthEventRepository_FindByDateRange(t *testing.T) {
	db, mock := newAuthEventMockGormDB(t)
	from := time.Now().Add(-time.Hour).UTC()
	to := time.Now().UTC()
	eventUUID := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "auth_events" WHERE tenant_id = \$1 AND created_at BETWEEN \$2 AND \$3 ORDER BY created_at DESC`).
		WithArgs(int64(1), from, to).
		WillReturnRows(authEventRows().AddRow(1, eventUUID, 1, "10.0.0.1", AuthEventCategoryAuthn, AuthEventTypeLoginSuccess, AuthEventSeverityInfo, AuthEventResultSuccess, []byte(`{}`), to))

	events, err := NewAuthEventRepository(db).FindByDateRange(1, from, to)

	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, eventUUID, events[0].AuthEventUUID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthEventRepository_DeleteOlderThan(t *testing.T) {
	db, mock := newAuthEventMockGormDB(t)
	cutoff := time.Now().Add(-365 * 24 * time.Hour).UTC()

	mock.ExpectBegin()
	mock.ExpectExec(`SELECT set_config\('maintainerd\.allow_auth_event_delete', \$1, true\)`).
		WithArgs("retention").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM "auth_events" WHERE created_at < \$1`).
		WithArgs(cutoff).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	count, err := NewAuthEventRepository(db).DeleteOlderThan(cutoff)

	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthEventRepository_DeleteExpiredByAuditConfig(t *testing.T) {
	db, mock := newAuthEventMockGormDB(t)
	now := time.Now().UTC()

	mock.ExpectBegin()
	mock.ExpectExec(`SELECT set_config\('maintainerd\.allow_auth_event_delete', \$1, true\)`).
		WithArgs("retention").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM auth_events\s+WHERE auth_events\.created_at < \$1 - \(`).
		WithArgs(now, 90).
		WillReturnResult(sqlmock.NewResult(0, 4))
	mock.ExpectCommit()

	count, err := NewAuthEventRepository(db).DeleteExpiredByAuditConfig(now, 90)

	require.NoError(t, err)
	assert.Equal(t, int64(4), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthEventRepository_ImmutableGenericMutations(t *testing.T) {
	db, _ := newAuthEventMockGormDB(t)
	repo := NewAuthEventRepository(db)

	_, err := repo.CreateOrUpdate(&AuthEvent{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "append-only")

	_, err = repo.UpdateByUUID(uuid.New(), map[string]any{"result": AuthEventResultFailure})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "immutable")

	_, err = repo.UpdateByID(1, map[string]any{"result": AuthEventResultFailure})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "immutable")

	err = repo.DeleteByUUID(uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retention or tenant deletion")

	err = repo.DeleteByID(1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retention or tenant deletion")
}

func TestAuthEventRepository_CountByEventType(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newAuthEventMockGormDB(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "auth_events" WHERE event_type = \$1 AND tenant_id = \$2`).
			WithArgs(AuthEventTypeLoginSuccess, int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		count, err := NewAuthEventRepository(db).CountByEventType(AuthEventTypeLoginSuccess, 1)

		require.NoError(t, err)
		assert.Equal(t, int64(5), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newAuthEventMockGormDB(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "auth_events" WHERE event_type = \$1 AND tenant_id = \$2`).
			WithArgs(AuthEventTypeLoginSuccess, int64(1)).
			WillReturnError(assert.AnError)

		count, err := NewAuthEventRepository(db).CountByEventType(AuthEventTypeLoginSuccess, 1)

		require.Error(t, err)
		assert.Equal(t, int64(0), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAuthEventRepository_WithTx(t *testing.T) {
	db, mock := newAuthEventMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()
	tx := db.Begin()
	require.NoError(t, tx.Error)

	repo := NewAuthEventRepository(db).WithTx(tx)

	assert.NotNil(t, repo)
	require.NoError(t, tx.Rollback().Error)
	assert.NoError(t, mock.ExpectationsWereMet())
}

package webhook

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newWebhookEndpointMockGormDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return db, mock
}

func webhookEndpointRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"webhook_endpoint_id",
		"webhook_endpoint_uuid",
		"tenant_id",
		"url",
		"secret_encrypted",
		"events",
		"max_retries",
		"timeout_seconds",
		"status",
		"description",
		"metadata",
		"last_triggered_at",
		"created_at",
		"updated_at",
		"deleted_at",
	})
}

func addWebhookEndpointRow(rows *sqlmock.Rows, id int64, endpointUUID uuid.UUID, tenantID int64, status string, now time.Time) *sqlmock.Rows {
	return rows.AddRow(id, endpointUUID, tenantID, "https://example.com/hook", "secret", []byte(`["user.created"]`), 3, 30, status, "desc", []byte(`{}`), nil, now, now, nil)
}

func TestNewWebhookEndpointRepository(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newWebhookEndpointMockGormDB(t)

		repo := NewWebhookEndpointRepository(db)

		assert.NotNil(t, repo)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestWebhookEndpointRepository_WithTx(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newWebhookEndpointMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tx := db.Begin()
		require.NoError(t, tx.Error)

		repo := NewWebhookEndpointRepository(db).WithTx(tx)

		assert.NotNil(t, repo)
		require.NoError(t, tx.Rollback().Error)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestWebhookEndpointRepository_FindByTenantID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newWebhookEndpointMockGormDB(t)
		now := time.Now().UTC()
		endpointUUID := uuid.New()
		mock.ExpectQuery(`SELECT \* FROM "webhook_endpoints" WHERE tenant_id = \$1 AND "webhook_endpoints"."deleted_at" IS NULL`).
			WithArgs(int64(1)).
			WillReturnRows(addWebhookEndpointRow(webhookEndpointRows(), 1, endpointUUID, 1, shared.StatusActive, now))

		endpoints, err := NewWebhookEndpointRepository(db).FindByTenantID(1)

		require.NoError(t, err)
		require.Len(t, endpoints, 1)
		assert.Equal(t, endpointUUID, endpoints[0].WebhookEndpointUUID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newWebhookEndpointMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "webhook_endpoints" WHERE tenant_id = \$1`).
			WithArgs(int64(1)).
			WillReturnError(assert.AnError)

		endpoints, err := NewWebhookEndpointRepository(db).FindByTenantID(1)

		require.Error(t, err)
		assert.Nil(t, endpoints)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestWebhookEndpointRepository_FindByUUIDAndTenantID(t *testing.T) {
	endpointUUID := uuid.New()
	now := time.Now().UTC()

	t.Run("success", func(t *testing.T) {
		db, mock := newWebhookEndpointMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "webhook_endpoints" WHERE \(webhook_endpoint_uuid = \$1 AND tenant_id = \$2\) AND "webhook_endpoints"."deleted_at" IS NULL ORDER BY "webhook_endpoints"."webhook_endpoint_id" LIMIT \$3`).
			WithArgs(endpointUUID.String(), int64(1), 1).
			WillReturnRows(addWebhookEndpointRow(webhookEndpointRows(), 1, endpointUUID, 1, shared.StatusActive, now))

		endpoint, err := NewWebhookEndpointRepository(db).FindByUUIDAndTenantID(endpointUUID, 1)

		require.NoError(t, err)
		require.NotNil(t, endpoint)
		assert.Equal(t, endpointUUID, endpoint.WebhookEndpointUUID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found returns nil", func(t *testing.T) {
		db, mock := newWebhookEndpointMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "webhook_endpoints" WHERE \(webhook_endpoint_uuid = \$1 AND tenant_id = \$2\) AND "webhook_endpoints"."deleted_at" IS NULL ORDER BY "webhook_endpoints"."webhook_endpoint_id" LIMIT \$3`).
			WithArgs(endpointUUID.String(), int64(1), 1).
			WillReturnRows(webhookEndpointRows())

		endpoint, err := NewWebhookEndpointRepository(db).FindByUUIDAndTenantID(endpointUUID, 1)

		require.NoError(t, err)
		assert.Nil(t, endpoint)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newWebhookEndpointMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "webhook_endpoints" WHERE \(webhook_endpoint_uuid = \$1 AND tenant_id = \$2\)`).
			WithArgs(endpointUUID.String(), int64(1), 1).
			WillReturnError(assert.AnError)

		endpoint, err := NewWebhookEndpointRepository(db).FindByUUIDAndTenantID(endpointUUID, 1)

		require.Error(t, err)
		assert.Nil(t, endpoint)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestWebhookEndpointRepository_FindPaginated(t *testing.T) {
	t.Run("success with filters", func(t *testing.T) {
		db, mock := newWebhookEndpointMockGormDB(t)
		tenantID := int64(1)
		endpointUUID := uuid.New()
		now := time.Now().UTC()
		mock.ExpectQuery(`SELECT count\(\*\) FROM "webhook_endpoints" WHERE tenant_id = \$1 AND status IN \(\$2\) AND "webhook_endpoints"."deleted_at" IS NULL`).
			WithArgs(tenantID, shared.StatusActive).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "webhook_endpoints" WHERE tenant_id = \$1 AND status IN \(\$2\) AND "webhook_endpoints"."deleted_at" IS NULL ORDER BY status ASC LIMIT \$3`).
			WithArgs(tenantID, shared.StatusActive, 10).
			WillReturnRows(addWebhookEndpointRow(webhookEndpointRows(), 1, endpointUUID, tenantID, shared.StatusActive, now))

		page, err := NewWebhookEndpointRepository(db).FindPaginated(WebhookEndpointRepositoryGetFilter{
			TenantID:  &tenantID,
			Status:    []string{shared.StatusActive},
			Page:      1,
			Limit:     10,
			SortBy:    "status",
			SortOrder: "asc",
		})

		require.NoError(t, err)
		require.Len(t, page.Data, 1)
		assert.Equal(t, int64(1), page.Total)
		assert.Equal(t, endpointUUID, page.Data[0].WebhookEndpointUUID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count error", func(t *testing.T) {
		db, mock := newWebhookEndpointMockGormDB(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "webhook_endpoints" WHERE "webhook_endpoints"."deleted_at" IS NULL`).
			WillReturnError(assert.AnError)

		page, err := NewWebhookEndpointRepository(db).FindPaginated(WebhookEndpointRepositoryGetFilter{})

		require.Error(t, err)
		assert.Nil(t, page)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestWebhookEndpointRepository_FindActiveByTenantID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newWebhookEndpointMockGormDB(t)
		now := time.Now().UTC()
		endpointUUID := uuid.New()
		mock.ExpectQuery(`SELECT \* FROM "webhook_endpoints" WHERE \(tenant_id = \$1 AND status = \$2\) AND "webhook_endpoints"."deleted_at" IS NULL`).
			WithArgs(int64(1), shared.StatusActive).
			WillReturnRows(addWebhookEndpointRow(webhookEndpointRows(), 1, endpointUUID, 1, shared.StatusActive, now))

		endpoints, err := NewWebhookEndpointRepository(db).FindActiveByTenantID(1)

		require.NoError(t, err)
		require.Len(t, endpoints, 1)
		assert.Equal(t, endpointUUID, endpoints[0].WebhookEndpointUUID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newWebhookEndpointMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "webhook_endpoints" WHERE \(tenant_id = \$1 AND status = \$2\)`).
			WithArgs(int64(1), shared.StatusActive).
			WillReturnError(assert.AnError)

		endpoints, err := NewWebhookEndpointRepository(db).FindActiveByTenantID(1)

		require.Error(t, err)
		assert.Nil(t, endpoints)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestWebhookEndpointRepository_UpdateLastTriggeredAt(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newWebhookEndpointMockGormDB(t)
		now := time.Now().UTC()
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "webhook_endpoints" SET "last_triggered_at"=\$1,"updated_at"=\$2 WHERE webhook_endpoint_id = \$3 AND "webhook_endpoints"."deleted_at" IS NULL`).
			WithArgs(now, sqlmock.AnyArg(), int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := NewWebhookEndpointRepository(db).UpdateLastTriggeredAt(1, now)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newWebhookEndpointMockGormDB(t)
		now := time.Now().UTC()
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "webhook_endpoints" SET "last_triggered_at"=\$1,"updated_at"=\$2 WHERE webhook_endpoint_id = \$3 AND "webhook_endpoints"."deleted_at" IS NULL`).
			WithArgs(now, sqlmock.AnyArg(), int64(1)).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := NewWebhookEndpointRepository(db).UpdateLastTriggeredAt(1, now)

		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

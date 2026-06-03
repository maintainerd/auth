package tenant

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantRetentionRunner(t *testing.T) {
	t.Run("nil db returns", func(t *testing.T) {
		StartRetentionRunner(context.Background(), nil, 0, 0)
	})

	t.Run("runs once then exits on cancelled context", func(t *testing.T) {
		db, mock := newMockGormDB(t)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		StartRetentionRunner(ctx, db, 0, 0)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("runTenantRetention deletes expired soft-deleted tenants", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "tenants" WHERE deleted_at IS NOT NULL AND deleted_at < \$1 AND is_system = false`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		runTenantRetention(context.Background(), db, time.Hour)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("loop runs retention on tick and exits on context cancellation", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "tenants" WHERE deleted_at IS NOT NULL AND deleted_at < \$1 AND is_system = false`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		ctx, cancel := context.WithCancel(context.Background())
		ticks := make(chan time.Time)
		done := make(chan struct{})
		go func() {
			runTenantRetentionLoop(ctx, db, time.Hour, ticks)
			close(done)
		}()

		ticks <- time.Now()
		require.Eventually(t, func() bool {
			return mock.ExpectationsWereMet() == nil
		}, time.Second, time.Millisecond)
		cancel()
		<-done

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("runTenantRetention logs and returns on delete error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "tenants" WHERE deleted_at IS NOT NULL AND deleted_at < \$1 AND is_system = false`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		runTenantRetention(context.Background(), db, time.Hour)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

package migration

import (
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCreateAuthEventsAppendOnlyRuleBlocksUpdatesOnly(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	mock.ExpectExec(regexp.QuoteMeta(`
		CREATE OR REPLACE RULE no_update_auth_events
		AS ON UPDATE TO auth_events DO INSTEAD NOTHING
	`)).WillReturnResult(sqlmock.NewResult(0, 0))

	require.NoError(t, CreateAuthEventsAppendOnlyRule(db))
	require.NoError(t, mock.ExpectationsWereMet())
}

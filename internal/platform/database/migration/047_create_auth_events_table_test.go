package migration

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCreateAuthEventsTableAddsImmutabilityTrigger(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = sqlDB.Close() }()

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS auth_events[\s\S]*CREATE OR REPLACE FUNCTION protect_auth_events_immutable\(\)[\s\S]*CREATE TRIGGER trg_protect_auth_events_immutable`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	require.NoError(t, CreateAuthEventsTable(db))
	require.NoError(t, mock.ExpectationsWereMet())
}

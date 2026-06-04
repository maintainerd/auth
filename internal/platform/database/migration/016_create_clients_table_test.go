package migration

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCreateClientTable(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = sqlDB.Close() }()

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS clients").WillReturnResult(sqlmock.NewResult(0, 0))
	require.NoError(t, CreateClientTable(db))
	require.NoError(t, mock.ExpectationsWereMet())
}

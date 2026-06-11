package invite

import (
	"os"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/maintainerd/auth/internal/platform/signedurl"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	_ = signedurl.Configure([]byte("test-secret-key-for-hmac"))
	os.Exit(m.Run())
}

func newMockGormDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return gormDB, mock
}

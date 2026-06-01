//go:build integration

package repository_test

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newMockGormDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return gormDB, mock
}

func TestIntegration_Tenant_FindByUUID(t *testing.T) {
	db, mock := newMockGormDB(t)
	testUUID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE.*ORDER BY.*LIMIT`).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name", "status", "created_at", "updated_at"}).
			AddRow(1, testUUID, "acme", "active", now, now))

	repo := tenant.NewTenantRepository(db)
	result, err := repo.FindByUUID(testUUID)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "acme", result.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIntegration_Tenant_Create(t *testing.T) {
	db, mock := newMockGormDB(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "tenants"`).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id"}).AddRow(1))
	mock.ExpectCommit()

	tModel := &tenant.Tenant{
		TenantUUID: uuid.New(),
		Name:       "new-tenant",
		Status:     "active",
	}
	err := db.Create(tModel).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), tModel.TenantID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIntegration_GORM_Transaction(t *testing.T) {
	db, mock := newMockGormDB(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "tenants"`).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id"}).AddRow(1))
	mock.ExpectCommit()

	err := db.Transaction(func(tx *gorm.DB) error {
		tModel := &tenant.Tenant{
			TenantUUID: uuid.New(),
			Name:       "tx-tenant",
			Status:     "active",
		}
		return tx.Create(tModel).Error
	})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIntegration_GORM_Rollback(t *testing.T) {
	db, mock := newMockGormDB(t)

	mock.ExpectBegin()
	mock.ExpectRollback()

	err := db.Transaction(func(tx *gorm.DB) error {
		return gorm.ErrRecordNotFound
	})
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIntegration_Tenant_Pagination(t *testing.T) {
	db, mock := newMockGormDB(t)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "tenants"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	mock.ExpectQuery(`SELECT \* FROM "tenants"`).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name", "status", "created_at", "updated_at"}).
			AddRow(1, uuid.New(), "t1", "active", time.Now(), time.Now()).
			AddRow(2, uuid.New(), "t2", "inactive", time.Now(), time.Now()))

	var results []tenant.Tenant
	var total int64
	err := db.Model(&tenant.Tenant{}).Count(&total).Error
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	err = db.Offset(0).Limit(10).Find(&results).Error
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

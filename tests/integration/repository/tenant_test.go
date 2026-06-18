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

func TestIntegration_TenantRepository_FindPaginated_WithFilters(t *testing.T) {
	db, mock := newMockGormDB(t)
	now := time.Now()
	status := "active"
	public := true

	mock.ExpectQuery(`SELECT count\(\*\) FROM "tenants" WHERE.*name.*display_name.*description.*identifier.*status IN.*is_public`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE.*name.*display_name.*description.*identifier.*status IN.*is_public.*ORDER BY name ASC LIMIT`).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "tenant_uuid", "name", "display_name", "description", "identifier", "status", "is_public", "created_at", "updated_at"}).
			AddRow(1, uuid.New(), "acme", "Acme", "Acme tenant", "acme", "active", true, now, now))

	repo := tenant.NewTenantRepository(db)
	result, err := repo.FindPaginated(tenant.TenantRepositoryGetFilter{
		Name:        &status,
		DisplayName: &status,
		Description: &status,
		Identifier:  &status,
		Status:      []string{"active"},
		IsPublic:    &public,
		Page:        1,
		Limit:       10,
		SortBy:      "name",
		SortOrder:   "asc",
	})

	require.NoError(t, err)
	require.Len(t, result.Data, 1)
	assert.Equal(t, "acme", result.Data[0].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIntegration_TenantRepository_SetStatusByUUID(t *testing.T) {
	db, mock := newMockGormDB(t)
	tenantUUID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "tenants" SET "status"=\$1,"updated_at"=\$2 WHERE tenant_uuid = \$3 AND "tenants"."deleted_at" IS NULL`).
		WithArgs("inactive", sqlmock.AnyArg(), tenantUUID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := tenant.NewTenantRepository(db)
	err := repo.SetStatusByUUID(tenantUUID, "inactive")

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIntegration_TenantMemberRepository_FindByTenantAndUser(t *testing.T) {
	db, mock := newMockGormDB(t)
	memberUUID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT \* FROM "tenant_members" WHERE \(tenant_id = \$1 AND user_id = \$2\) AND "tenant_members"."deleted_at" IS NULL ORDER BY "tenant_members"."tenant_member_id" LIMIT \$3`).
		WithArgs(int64(10), int64(20), 1).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_member_id", "tenant_member_uuid", "tenant_id", "user_id", "role", "created_at", "updated_at"}).
			AddRow(1, memberUUID, 10, 20, "owner", now, now))

	repo := tenant.NewTenantMemberRepository(db)
	result, err := repo.FindByTenantAndUser(10, 20)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, memberUUID, result.TenantMemberUUID)
	assert.Equal(t, "owner", result.Role)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIntegration_TenantMemberRepository_FindByTenant_IsTenantScoped(t *testing.T) {
	db, mock := newMockGormDB(t)
	now := time.Now()
	role := "member"

	mock.ExpectQuery(`SELECT count\(\*\) FROM "tenant_members" WHERE tenant_id = \$1 AND role = \$2 AND "tenant_members"."deleted_at" IS NULL`).
		WithArgs(int64(10), role).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT \* FROM "tenant_members" WHERE tenant_id = \$1 AND role = \$2 AND "tenant_members"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT \$3`).
		WithArgs(int64(10), role, 10).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_member_id", "tenant_member_uuid", "tenant_id", "user_id", "role", "created_at", "updated_at"}).
			AddRow(1, uuid.New(), 10, 20, "member", now, now))

	repo := tenant.NewTenantMemberRepository(db)
	result, err := repo.FindByTenant(tenant.TenantMemberRepositoryListFilter{
		TenantID: 10,
		Role:     &role,
		Page:     1,
		Limit:    10,
	})

	require.NoError(t, err)
	require.Len(t, result.Data, 1)
	assert.Equal(t, int64(10), result.Data[0].TenantID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIntegration_TenantSettingRepository_FindByTenantID(t *testing.T) {
	db, mock := newMockGormDB(t)
	settingUUID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT \* FROM "tenant_settings" WHERE tenant_id = \$1 ORDER BY "tenant_settings"."tenant_setting_id" LIMIT \$2`).
		WithArgs(int64(10), 1).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_setting_id", "tenant_setting_uuid", "tenant_id", "rate_limit_config", "audit_config", "maintenance_config", "created_at", "updated_at"}).
			AddRow(1, settingUUID, 10, []byte(`{"max":100}`), []byte(`{"enabled":true}`), []byte(`{"active":false}`), now, now))

	repo := tenant.NewTenantSettingRepository(db)
	result, err := repo.FindByTenantID(10)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, settingUUID, result.TenantSettingUUID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

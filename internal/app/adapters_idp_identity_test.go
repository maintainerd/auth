//nolint:staticcheck
//lint:file-ignore SA5011 pre-existing nil-check patterns
package app

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/idp"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newIdentityAdapterDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		_ = sqlDB.Close()
		t.Fatalf("gorm.Open: %v", err)
	}
	return db, mock, func() { _ = sqlDB.Close() }
}

func newIdpUserIdentityRepo(db *gorm.DB) *idpUserIdentityRepo {
	return &idpUserIdentityRepo{
		BaseRepository: database.NewBaseRepository[idp.UserIdentity](db, "user_identity_uuid", "user_identity_id"),
	}
}

// The ON CONFLICT target must name the unique index that actually exists.
// PostgreSQL resolves the arbiter index at PLAN time, so a stale target does not
// merely fail to dedupe — it raises 42P10 on EVERY insert, taking down all
// federated JIT provisioning. Migration 030 keys uniqueness on (tenant_id, sub);
// naming (tenant_id, provider, sub) here is exactly the outage this pins.
//
// A mocked repository cannot catch this, which is why the assertion is on the
// rendered SQL.
func TestCreateByTenantProviderSubIfAbsentTargetsTheRealUniqueIndex(t *testing.T) {
	db, mock, closeDB := newIdentityAdapterDB(t)
	defer closeDB()

	mock.ExpectBegin()
	// UserIdentity carries a primaryKey tag, so gorm issues the INSERT as a
	// Query with RETURNING "user_identity_id"; a returned row = row inserted.
	mock.ExpectQuery(regexp.QuoteMeta(`ON CONFLICT ("tenant_id","sub") DO NOTHING`)).
		WillReturnRows(sqlmock.NewRows([]string{"user_identity_id"}).AddRow(int64(1)))
	mock.ExpectCommit()

	repo := newIdpUserIdentityRepo(db)
	_, created, err := repo.CreateByTenantProviderSubIfAbsent(&idp.UserIdentity{
		TenantID:           1,
		UserID:             7,
		IdentityProviderID: 3,
		Provider:           "google",
		Sub:                "sub-123",
	})
	if err != nil {
		t.Fatalf("CreateByTenantProviderSubIfAbsent: %v", err)
	}
	if !created {
		t.Fatal("expected created=true when the insert produced a row")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// On conflict the row must be re-read by the SAME key the conflict fired on.
// Looking it up by (tenant, provider, sub) would miss a row holding this sub
// under a DIFFERENT provider: the caller would then see no owner, conclude
// nothing was wrong, and continue with a freshly created user that has no
// external identity — silently provisioning a duplicate account per login.
func TestCreateByTenantProviderSubIfAbsentResolvesOwnerAcrossProviders(t *testing.T) {
	db, mock, closeDB := newIdentityAdapterDB(t)
	defer closeDB()

	ownerUUID := uuid.New()

	mock.ExpectBegin()
	// RETURNING with no rows = the insert was refused by the conflict target.
	mock.ExpectQuery(regexp.QuoteMeta(`ON CONFLICT ("tenant_id","sub") DO NOTHING`)).
		WillReturnRows(sqlmock.NewRows([]string{"user_identity_id"}))
	mock.ExpectCommit()

	// Keyed on (tenant_id, sub) only — no provider predicate.
	mock.ExpectQuery(`SELECT \* FROM "user_identities" WHERE tenant_id = \$1 AND sub = \$2`).
		WithArgs(int64(1), "sub-123", 1).
		WillReturnRows(sqlmock.NewRows([]string{"user_identity_id", "user_identity_uuid", "tenant_id", "user_id", "identity_provider_id", "provider", "sub"}).
			AddRow(int64(9), ownerUUID, int64(1), int64(42), int64(8), "okta", "sub-123"))

	repo := newIdpUserIdentityRepo(db)
	existing, created, err := repo.CreateByTenantProviderSubIfAbsent(&idp.UserIdentity{
		TenantID:           1,
		UserID:             7,
		IdentityProviderID: 3,
		Provider:           "google",
		Sub:                "sub-123",
	})
	if err != nil {
		t.Fatalf("CreateByTenantProviderSubIfAbsent: %v", err)
	}
	if created {
		t.Fatal("expected created=false when the insert was refused")
	}
	//lint:file-ignore SA5011 pre-existing nil-check patterns
	if existing == nil {
		t.Fatal("the conflicting owner must be returned, otherwise the caller cannot detect the collision")
	}
	if existing.UserID != 42 || existing.IdentityProviderID != 8 {
		t.Fatalf("expected the okta owner (user 42, idp 8), got user %d idp %d", existing.UserID, existing.IdentityProviderID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

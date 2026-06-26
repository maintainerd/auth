package app

import (
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/maintainerd/auth/internal/shared"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestOAuthClientRepoFindByIdentifierPreloadsClientURIs(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer sqlDB.Close()

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}

	now := time.Now()
	requirePKCE := true
	clientUUID := uuid.New()
	tenantUUID := uuid.New()
	uriUUID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "clients" WHERE clients.identifier = $1 AND clients.status = $2 AND "clients"."deleted_at" IS NULL ORDER BY "clients"."client_id" LIMIT $3`)).
		WithArgs("local-app-5141", shared.StatusActive, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"client_id", "client_uuid", "tenant_id", "name", "display_name", "client_type", "domain", "identifier",
			"status", "is_default", "is_system", "token_endpoint_auth_method", "grant_types", "response_types",
			"require_consent", "require_pkce", "allowed_scopes", "created_at", "updated_at",
		}).AddRow(
			int64(3), clientUUID, int64(1), "local-app-5141", "Localhost 5141 Test App", shared.ClientTypeSPA,
			"http://localhost:5141", "local-app-5141", shared.StatusActive, false, false, "none",
			pqArray("authorization_code", "refresh_token"), pqArray("code"), false, requirePKCE,
			pqArray("openid", "profile", "email"), now, now,
		))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "client_uris" WHERE "client_uris"."client_id" = $1`)).
		WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{
			"client_uri_id", "client_uri_uuid", "tenant_id", "client_id", "uri", "type", "created_at", "updated_at",
		}).AddRow(int64(10), uriUUID, int64(1), int64(3), "http://localhost:5141/callback", shared.ClientURITypeRedirect, now, now))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants" WHERE "tenants"."tenant_id" = $1`)).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"tenant_id", "tenant_uuid", "name", "display_name", "identifier", "status", "is_system", "created_at", "updated_at",
		}).AddRow(int64(1), tenantUUID, "maintainerd", "Maintainerd Auth", "maintainerd", shared.StatusActive, true, now, now))

	client, err := newOAuthClientRepo(db).FindByIdentifier("local-app-5141")
	if err != nil {
		t.Fatalf("FindByIdentifier error: %v", err)
	}
	if client == nil {
		t.Fatal("FindByIdentifier returned nil")
	}
	if client.ClientURIs == nil || len(*client.ClientURIs) != 1 {
		t.Fatalf("ClientURIs = %#v, want one preloaded URI", client.ClientURIs)
	}
	if got := (*client.ClientURIs)[0].URI; got != "http://localhost:5141/callback" {
		t.Fatalf("redirect URI = %q", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func pqArray(values ...string) driver.Value {
	value, err := pq.Array(values).Value()
	if err != nil {
		panic(err)
	}
	return value
}

package runner_test

import (
	"os"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/runner"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestMigrationsApplyToFreshDatabase applies the whole migration set to an EMPTY
// database.
//
// Migrations are edited IN PLACE under the pre-release create-only policy, and
// the runner skips any version already recorded in schema_migrations. That means
// an existing dev database silently keeps the old shape and never exercises an
// edit — running from scratch is the only thing that proves the set is still
// coherent. Opt-in so the normal suite needs no database; two ways in:
//
//	MIGCHECK_DSN="host=localhost port=5433 user=... password=... dbname=migcheck sslmode=disable" \
//	  go test ./internal/platform/runner/ -run TestMigrationsApplyToFreshDatabase -count=1
//
// or, from inside the dev container, MIGCHECK_FROM_APP_CONFIG=1 to reuse the
// service's own configuration. That path exists so nobody has to copy the
// database password out of .env and into a shell command (where it lands in
// history and process listings) just to check the migrations — godotenv.Load
// never overrides an already-set variable, so DB_NAME can be redirected at a
// throwaway database while the credentials stay inside the process:
//
//	docker exec -e MIGCHECK_FROM_APP_CONFIG=1 -e DB_NAME=migcheck m9d-auth-dev \
//	  go test ./internal/platform/runner/ -run TestMigrationsApplyToFreshDatabase -count=1
func TestMigrationsApplyToFreshDatabase(t *testing.T) {
	dsn := os.Getenv("MIGCHECK_DSN")
	if dsn == "" && os.Getenv("MIGCHECK_FROM_APP_CONFIG") != "" {
		// go test runs each package with the package directory as the working
		// directory, and godotenv resolves .env relative to it — without this the
		// loader silently finds no file and every required variable reads as unset.
		if err := os.Chdir("../../.."); err != nil {
			t.Fatalf("chdir to module root: %v", err)
		}
		if err := config.Init(); err != nil {
			t.Fatalf("config.Init: %v", err)
		}
		// Guard against pointing a schema-mutating run at the real database:
		// this test is only meaningful against a scratch one anyway.
		if config.DBName == "" || config.DBName == "maintainerd" {
			t.Fatalf("refusing to run migrations against DB_NAME=%q; set DB_NAME to a throwaway database", config.DBName)
		}
		dsn = config.GetDBConnectionString()
	}
	if dsn == "" {
		t.Skip("MIGCHECK_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Error)})
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	// Ask the server which database this actually is rather than trusting the
	// DSN string. MIGCHECK_DSN is free-form (key=value or URL) and was not
	// checked at all, so the drop below could otherwise be aimed at the real
	// database by a single mistyped variable.
	var current string
	if err := db.Raw("SELECT current_database()").Scan(&current).Error; err != nil {
		t.Fatalf("current_database: %v", err)
	}
	if current == "" || current == "maintainerd" {
		t.Fatalf("refusing to drop the schema of database %q; point this at a throwaway database", current)
	}

	// Actually make the database empty. Without this the run is a no-op on every
	// invocation after the first: RunMigrations skips any version already in
	// schema_migrations, so the test passed in 0.02s having applied nothing and
	// having proved nothing — the exact false green this test exists to prevent.
	if err := db.Exec("DROP SCHEMA public CASCADE").Error; err != nil {
		t.Fatalf("drop schema: %v", err)
	}
	if err := db.Exec("CREATE SCHEMA public").Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}

	// Prove the reset landed before crediting the run, so a future change that
	// stops emptying the database fails here instead of passing vacuously.
	var trackingTable *string
	if err := db.Raw("SELECT to_regclass('public.schema_migrations')::text").Scan(&trackingTable).Error; err != nil {
		t.Fatalf("check tracking table: %v", err)
	}
	if trackingTable != nil {
		t.Fatalf("schema_migrations still present after reset; database is not empty")
	}

	if err := runner.RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	// A migration set that silently applied nothing would otherwise look
	// identical to one that applied cleanly.
	var applied int64
	if err := db.Raw("SELECT COUNT(*) FROM schema_migrations").Scan(&applied).Error; err != nil {
		t.Fatalf("count applied migrations: %v", err)
	}
	if applied == 0 {
		t.Fatal("no migrations were applied to the empty database")
	}
	t.Logf("applied %d migrations to an empty database", applied)
}

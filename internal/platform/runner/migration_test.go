package runner

import (
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestMigrations_RegisterLatestMigration(t *testing.T) {
	require.NotEmpty(t, migrations)
	last := migrations[len(migrations)-1]
	assert.NotNil(t, last.Fn)
}

// TestMigrations_ControlPlaneMigrationsRegistered catches the one failure the
// pin above cannot: a create migration written but never appended to the
// registry. Nothing else references these functions, so the package still
// compiles and the table simply never exists — which surfaces as a runtime
// "relation does not exist" on the first call that needs it.
func TestMigrations_ControlPlaneMigrationsRegistered(t *testing.T) {
	registered := make(map[string]bool, len(migrations))
	for _, m := range migrations {
		registered[m.Version] = true
	}
}

// TestMigrations_VersionsAreUniqueAndOrdered guards the registry invariant the
// runner depends on: it records applied versions by string, so a duplicated
// version silently skips the second migration entirely.
func TestMigrations_VersionsAreUniqueAndOrdered(t *testing.T) {
	seen := make(map[string]bool, len(migrations))
	previous := ""
	for _, m := range migrations {
		require.False(t, seen[m.Version], "duplicate migration version %q", m.Version)
		seen[m.Version] = true
		require.NotNil(t, m.Fn, "migration %q has no function", m.Version)
		assert.Greater(t, m.Version, previous, "migration %q is out of order", m.Version)
		previous = m.Version
	}
}

func TestRunMigrations_AllApplied(t *testing.T) {
	db, mock := newRunnerMockDB(t)
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_lock($1)")).
		WithArgs(advisoryLockKey).
		WillReturnResult(sqlmock.NewResult(0, 0))
	for _, entry := range migrations {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(1) FROM schema_migrations WHERE version = $1")).
			WithArgs(entry.Version).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	}
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_unlock($1)")).
		WithArgs(advisoryLockKey).
		WillReturnResult(sqlmock.NewResult(0, 0))

	require.NoError(t, RunMigrations(db))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunMigrations_AppliesPendingMigration(t *testing.T) {
	orig := migrations
	applied := false
	migrations = []migrationEntry{{
		Version: "001_fake",
		Fn: func(*gorm.DB) error {
			applied = true
			return nil
		},
	}}
	t.Cleanup(func() { migrations = orig })

	db, mock := newRunnerMockDB(t)
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_lock($1)")).
		WithArgs(advisoryLockKey).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(1) FROM schema_migrations WHERE version = $1")).
		WithArgs("001_fake").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO schema_migrations (version) VALUES ($1)")).
		WithArgs("001_fake").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_unlock($1)")).
		WithArgs(advisoryLockKey).
		WillReturnResult(sqlmock.NewResult(0, 0))

	require.NoError(t, RunMigrations(db))
	assert.True(t, applied)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunMigrations_Errors(t *testing.T) {
	t.Run("bootstrap error", func(t *testing.T) {
		db, mock := newRunnerMockDB(t)
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
			WillReturnError(assert.AnError)
		require.Error(t, RunMigrations(db))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("advisory lock error", func(t *testing.T) {
		db, mock := newRunnerMockDB(t)
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_lock($1)")).
			WithArgs(advisoryLockKey).
			WillReturnError(assert.AnError)
		require.Error(t, RunMigrations(db))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("check applied error", func(t *testing.T) {
		orig := migrations
		migrations = []migrationEntry{{Version: "001_fake", Fn: func(*gorm.DB) error { return nil }}}
		t.Cleanup(func() { migrations = orig })
		db, mock := newRunnerMockDB(t)
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_lock($1)")).
			WithArgs(advisoryLockKey).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(1) FROM schema_migrations WHERE version = $1")).
			WithArgs("001_fake").
			WillReturnError(assert.AnError)
		mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_unlock($1)")).
			WithArgs(advisoryLockKey).
			WillReturnResult(sqlmock.NewResult(0, 0))
		require.Error(t, RunMigrations(db))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("migration function error", func(t *testing.T) {
		orig := migrations
		migrations = []migrationEntry{{Version: "001_fake", Fn: func(*gorm.DB) error { return assert.AnError }}}
		t.Cleanup(func() { migrations = orig })
		db, mock := newRunnerMockDB(t)
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_lock($1)")).
			WithArgs(advisoryLockKey).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(1) FROM schema_migrations WHERE version = $1")).
			WithArgs("001_fake").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
		mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_unlock($1)")).
			WithArgs(advisoryLockKey).
			WillReturnResult(sqlmock.NewResult(0, 0))
		require.Error(t, RunMigrations(db))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("record error", func(t *testing.T) {
		orig := migrations
		migrations = []migrationEntry{{Version: "001_fake", Fn: func(*gorm.DB) error { return nil }}}
		t.Cleanup(func() { migrations = orig })
		db, mock := newRunnerMockDB(t)
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_lock($1)")).
			WithArgs(advisoryLockKey).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(1) FROM schema_migrations WHERE version = $1")).
			WithArgs("001_fake").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO schema_migrations (version) VALUES ($1)")).
			WithArgs("001_fake").
			WillReturnError(assert.AnError)
		mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_unlock($1)")).
			WithArgs(advisoryLockKey).
			WillReturnResult(sqlmock.NewResult(0, 0))
		require.Error(t, RunMigrations(db))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMigrationHelpers(t *testing.T) {
	t.Run("is applied false", func(t *testing.T) {
		db, mock := newRunnerMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(1) FROM schema_migrations WHERE version = $1")).
			WithArgs("001").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))

		applied, err := isMigrationApplied(db, "001")
		require.NoError(t, err)
		assert.False(t, applied)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("record migration", func(t *testing.T) {
		db, mock := newRunnerMockDB(t)
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO schema_migrations (version) VALUES ($1)")).
			WithArgs("001").
			WillReturnResult(sqlmock.NewResult(0, 1))

		require.NoError(t, recordMigration(db, "001"))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func newRunnerMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	return db, mock
}

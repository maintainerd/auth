package config

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/retry"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// InitDB opens a connection to the PostgreSQL database, waits for it to be
// reachable (retrying with exponential backoff), enforces SSL in production,
// applies connection-pool limits, and registers the OTel plugin.
func InitDB(ctx context.Context) (*gorm.DB, error) {
	if AppEnv == "production" && DBSSLMode == "disable" {
		return nil, fmt.Errorf("database SSL is disabled (DB_SSLMODE=disable) — not allowed in production")
	}

	// gorm.Open with the postgres driver is lazy — it does not dial the server.
	// The actual connection attempt happens on the first Ping or query.
	db, err := gorm.Open(postgres.Open(GetDBConnectionString()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database driver: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Apply pool limits before the ping attempts so they are active from the
	// first connection the pool establishes.
	sqlDB.SetMaxOpenConns(DBMaxOpenConns)
	sqlDB.SetMaxIdleConns(DBMaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(DBConnMaxLifetimeSec) * time.Second)
	sqlDB.SetConnMaxIdleTime(90 * time.Second)

	if err := retry.WithBackoff(ctx, "postgres", 10, 2*time.Second, func() error {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return sqlDB.PingContext(pingCtx)
	}); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Use(otelgorm.NewPlugin()); err != nil {
		return nil, fmt.Errorf("failed to register otelgorm plugin: %w", err)
	}

	slog.Info("Database connected",
		"max_open_conns", DBMaxOpenConns,
		"max_idle_conns", DBMaxIdleConns,
		"conn_max_lifetime_sec", DBConnMaxLifetimeSec,
		"statement_timeout_ms", DBStatementTimeoutMs,
	)
	return db, nil
}

func GetDBConnectionString() string {
	// The `options` value contains a space, so it must be single-quoted in the
	// keyword/value DSN — otherwise the driver splits it and Postgres receives a
	// bare `-c` (FATAL: invalid command-line argument for server process: -c).
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s options='-c statement_timeout=%d'",
		DBHost, DBPort, DBUser, DBPassword, DBName, DBSSLMode, DBStatementTimeoutMs,
	)
}

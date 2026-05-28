package config

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// InitDB opens a connection to the PostgreSQL database, enforces SSL in
// production, applies connection-pool limits, and registers the OTel plugin.
func InitDB() (*gorm.DB, error) {
	if AppEnv == "production" && DBSSLMode == "disable" {
		return nil, fmt.Errorf("database SSL is disabled (DB_SSLMODE=disable) — not allowed in production")
	}

	db, err := gorm.Open(postgres.Open(GetDBConnectionString()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Use(otelgorm.NewPlugin()); err != nil {
		return nil, fmt.Errorf("failed to register otelgorm plugin: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(DBMaxOpenConns)
	sqlDB.SetMaxIdleConns(DBMaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(DBConnMaxLifetimeSec) * time.Second)

	slog.Info("Database connected",
		"max_open_conns", DBMaxOpenConns,
		"max_idle_conns", DBMaxIdleConns,
		"conn_max_lifetime_sec", DBConnMaxLifetimeSec,
		"statement_timeout_ms", DBStatementTimeoutMs,
	)
	return db, nil
}

func GetDBConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s options=-c statement_timeout=%d",
		DBHost, DBPort, DBUser, DBPassword, DBName, DBSSLMode, DBStatementTimeoutMs,
	)
}

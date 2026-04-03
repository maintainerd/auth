package config

import (
	"fmt"
	"log/slog"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// InitDB opens a connection to the PostgreSQL database and returns it together
// with any error. It no longer calls os.Exit so that main() can decide how to
// handle initialization failures.
func InitDB() (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(GetDBConnectionString()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	slog.Info("Database connected")
	return db, nil
}

func GetDBConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		DBHost, DBPort, DBUser, DBPassword, DBName, DBSSLMode,
	)
}

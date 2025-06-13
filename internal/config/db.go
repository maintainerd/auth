package config

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {
	db, err := gorm.Open(postgres.Open(GetDBConnectionString()), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Failed to connect to DB:", err)
	}

	fmt.Println("✅ Database connected")
	return db
}

func GetDBConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		DBHost, DBPort, DBUser, DBPassword, DBName, DBSSLMode,
	)
}

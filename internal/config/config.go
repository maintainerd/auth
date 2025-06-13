package config

import (
	"log"

	"github.com/joho/godotenv"
)

var (
	// JWT
	JWTPrivateKey []byte
	JWTPublicKey  []byte

	// DB Config
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
)

func Init() {
	// Load environment variables first
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ .env file not found, relying on environment variables")
	}

	// JWT Config
	JWTPrivateKey = []byte(GetEnv("JWT_PRIVATE_KEY"))
	JWTPublicKey = []byte(GetEnv("JWT_PUBLIC_KEY"))

	// DB Config
	DBHost = GetEnv("DB_HOST")
	DBPort = GetEnv("DB_PORT")
	DBUser = GetEnv("DB_USER")
	DBPassword = GetEnv("DB_PASSWORD")
	DBName = GetEnv("DB_NAME")
	DBSSLMode = GetEnvOrDefault("DB_SSLMODE", "disable")
}

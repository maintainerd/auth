package config

import (
	"log"
	"strconv"

	"github.com/joho/godotenv"
)

var (
	// APP
	AppVersion         string
	AppPublicHostname  string
	AppPrivateHostname string

	// FRONTEND
	AccountHostname string
	AuthHostname    string

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

	// Email Config
	SMTPHost      string
	SMTPPort      int
	SMTPUser      string
	SMTPPass      string
	SMTPFromEmail string
	SMTPFromName  string
	EmailLogo     string
)

func Init() {
	// Load environment variables first
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ .env file not found, relying on environment variables")
	}
	// App Config
	AppVersion = GetEnv("APP_VERSION")
	AppPublicHostname = GetEnv("APP_PUBLIC_HOSTNAME")
	AppPrivateHostname = GetEnv("APP_PRIVATE_HOSTNAME")

	// Frontend Config
	AccountHostname = GetEnv("ACCOUNT_HOSTNAME")
	AuthHostname = GetEnv("AUTH_HOSTNAME")

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

	// Email Config
	SMTPHost = GetEnv("SMTP_HOST")
	portStr := GetEnv("SMTP_PORT")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("❌ Invalid SMTP_PORT: %v", err)
	}
	SMTPPort = port
	SMTPUser = GetEnv("SMTP_USER")
	SMTPPass = GetEnv("SMTP_PASS")
	SMTPFromEmail = GetEnvOrDefault("SMTP_FROM_EMAIL", "noreply@maintainerd.com")
	SMTPFromName = GetEnvOrDefault("SMTP_FROM_NAME", "Maintainerd")
	EmailLogo = GetEnvOrDefault("EMAIL_LOGO_URL", "https://avatars.githubusercontent.com/u/215448978?s=400&u=f6f4016d81d3ef54ea34cd9cf3028a8ca1183afc&v=4")
}

package config

import (
	"fmt"
	"log/slog"
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

	// JWT Configuration
	JWTPrivateKey []byte
	JWTPublicKey  []byte

	// Secret Management
	SecretProvider string // "env", "aws_ssm", "aws_secrets", "vault", "azure_kv"
	SecretPrefix   string // Prefix for secret names in external providers

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

// Init loads all configuration from environment variables (and an optional .env file).
// It returns an error for any missing required variable so that main() can decide
// how to handle the failure — nothing in this package calls os.Exit.
func Init() error {
	// Load environment variables first (best-effort; not required in production)
	if err := godotenv.Load(); err != nil {
		slog.Warn(".env file not found, relying on environment variables")
	}

	// Secret management provider (optional with defaults)
	SecretProvider = GetEnvOrDefault("SECRET_PROVIDER", "env")
	SecretPrefix = GetEnvOrDefault("SECRET_PREFIX", "maintainerd/auth")

	if err := ValidateSecretProvider(); err != nil {
		return fmt.Errorf("secret provider validation failed: %w", err)
	}

	if err := initSecretManager(); err != nil {
		return fmt.Errorf("failed to initialize secret manager: %w", err)
	}

	// App Config
	var err error
	if AppVersion, err = GetEnv("APP_VERSION"); err != nil {
		return err
	}
	if AppPublicHostname, err = GetEnv("APP_PUBLIC_HOSTNAME"); err != nil {
		return err
	}
	if AppPrivateHostname, err = GetEnv("APP_PRIVATE_HOSTNAME"); err != nil {
		return err
	}

	// Frontend Config
	if AccountHostname, err = GetEnv("ACCOUNT_HOSTNAME"); err != nil {
		return err
	}
	if AuthHostname, err = GetEnv("AUTH_HOSTNAME"); err != nil {
		return err
	}

	// JWT Config — loaded via the configured secret provider
	slog.Info("Loading JWT keys from secret provider")
	if JWTPrivateKey, err = loadSecret("JWT_PRIVATE_KEY"); err != nil {
		return fmt.Errorf("failed to load JWT private key: %w", err)
	}
	if JWTPublicKey, err = loadSecret("JWT_PUBLIC_KEY"); err != nil {
		return fmt.Errorf("failed to load JWT public key: %w", err)
	}
	slog.Info("JWT keys loaded successfully")

	// DB Config
	if DBHost, err = GetEnv("DB_HOST"); err != nil {
		return err
	}
	if DBPort, err = GetEnv("DB_PORT"); err != nil {
		return err
	}
	if DBUser, err = GetEnv("DB_USER"); err != nil {
		return err
	}
	if DBPassword, err = GetEnv("DB_PASSWORD"); err != nil {
		return err
	}
	if DBName, err = GetEnv("DB_NAME"); err != nil {
		return err
	}
	DBSSLMode = GetEnvOrDefault("DB_SSLMODE", "disable")

	// Email Config
	if SMTPHost, err = GetEnv("SMTP_HOST"); err != nil {
		return err
	}
	smtpPortStr, err := GetEnv("SMTP_PORT")
	if err != nil {
		return err
	}
	SMTPPort, err = strconv.Atoi(smtpPortStr)
	if err != nil {
		return fmt.Errorf("invalid SMTP_PORT %q: %w", smtpPortStr, err)
	}
	if SMTPUser, err = GetEnv("SMTP_USER"); err != nil {
		return err
	}
	if SMTPPass, err = GetEnv("SMTP_PASS"); err != nil {
		return err
	}
	SMTPFromEmail = GetEnvOrDefault("SMTP_FROM_EMAIL", "noreply@maintainerd.com")
	SMTPFromName = GetEnvOrDefault("SMTP_FROM_NAME", "Maintainerd")
	EmailLogo = GetEnvOrDefault("EMAIL_LOGO_URL", "https://avatars.githubusercontent.com/u/215448978?s=400&u=f6f4016d81d3ef54ea34cd9cf3028a8ca1183afc&v=4")

	return nil
}

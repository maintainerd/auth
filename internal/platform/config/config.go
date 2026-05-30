package config

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/joho/godotenv"
)

var (
	// APP
	AppEnv             string // "development" or "production"; defaults "development"
	AppVersion         string
	AppPublicHostname  string
	AppPrivateHostname string

	// Application Encryption Key (AES-256)
	AppEncryptionKey []byte

	// Logging
	LogLevel string // "debug", "info", "warn", "error"; defaults "info"

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

	// DB Pool Config
	DBMaxOpenConns       int // DB_MAX_OPEN_CONNS; default 25
	DBMaxIdleConns       int // DB_MAX_IDLE_CONNS; default 10
	DBConnMaxLifetimeSec int // DB_CONN_MAX_LIFETIME_SEC; default 300

	// DB Statement Timeout
	DBStatementTimeoutMs int // DB_STATEMENT_TIMEOUT_MS; default 30000

	// Cookie Config
	CookieSecure   bool   // defaults true; set COOKIE_SECURE=false for local dev
	CookieSameSite string // "strict", "lax", or "none"; defaults "strict"

	// Email Config
	SMTPHost      string
	SMTPPort      int
	SMTPUser      string
	SMTPPass      string
	SMTPFromEmail string
	SMTPFromName  string
	EmailLogo     string

	// Email Provider
	EmailProvider string // EMAIL_PROVIDER; default "smtp"
	EmailAPIKey   string // EMAIL_API_KEY
	EmailDomain   string // EMAIL_DOMAIN (Mailgun)
	EmailRegion   string // EMAIL_REGION (SES); default "us-east-1"

	// SMS Config
	SMSProvider      string // SMS_PROVIDER; "twilio", "sns", "vonage"
	TwilioAccountSID string // TWILIO_ACCOUNT_SID
	TwilioAuthToken  string // TWILIO_AUTH_TOKEN
	TwilioFromNumber string // TWILIO_FROM_NUMBER
	SNSRegion        string // SNS_REGION; default "us-east-1"
	VonageAPIKey     string // VONAGE_API_KEY
	VonageAPISecret  string // VONAGE_API_SECRET
	VonageFrom       string // VONAGE_FROM
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
	AppEnv = GetEnvOrDefault("APP_ENV", "development")
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

	// Application encryption key — loaded via the configured secret provider.
	slog.Info("Loading application encryption key from secret provider")
	var encErr error
	AppEncryptionKey, encErr = loadSecret("APP_ENCRYPTION_KEY")
	if encErr != nil {
		return fmt.Errorf("failed to load APP_ENCRYPTION_KEY: %w", encErr)
	}
	if len(AppEncryptionKey) != 32 {
		return fmt.Errorf("APP_ENCRYPTION_KEY must be 32 bytes (AES-256), got %d", len(AppEncryptionKey))
	}
	slog.Info("Application encryption key loaded successfully")

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
	DBMaxOpenConns = parseIntDefault(GetEnvOrDefault("DB_MAX_OPEN_CONNS", "25"), 25)
	DBMaxIdleConns = parseIntDefault(GetEnvOrDefault("DB_MAX_IDLE_CONNS", "10"), 10)
	DBConnMaxLifetimeSec = parseIntDefault(GetEnvOrDefault("DB_CONN_MAX_LIFETIME_SEC", "300"), 300)
	DBStatementTimeoutMs = parseIntDefault(GetEnvOrDefault("DB_STATEMENT_TIMEOUT_MS", "30000"), 30000)

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
	EmailProvider = GetEnvOrDefault("EMAIL_PROVIDER", "smtp")
	EmailAPIKey = GetEnvOrDefault("EMAIL_API_KEY", "")
	EmailDomain = GetEnvOrDefault("EMAIL_DOMAIN", "")
	EmailRegion = GetEnvOrDefault("EMAIL_REGION", "us-east-1")

	// SMS Config
	SMSProvider = GetEnvOrDefault("SMS_PROVIDER", "")
	TwilioAccountSID = GetEnvOrDefault("TWILIO_ACCOUNT_SID", "")
	TwilioAuthToken = GetEnvOrDefault("TWILIO_AUTH_TOKEN", "")
	TwilioFromNumber = GetEnvOrDefault("TWILIO_FROM_NUMBER", "")
	SNSRegion = GetEnvOrDefault("SNS_REGION", "us-east-1")
	VonageAPIKey = GetEnvOrDefault("VONAGE_API_KEY", "")
	VonageAPISecret = GetEnvOrDefault("VONAGE_API_SECRET", "")
	VonageFrom = GetEnvOrDefault("VONAGE_FROM", "")

	// Cookie Config
	CookieSecure = GetEnvOrDefault("COOKIE_SECURE", "true") != "false"
	CookieSameSite = GetEnvOrDefault("COOKIE_SAMESITE", "strict")

	// Logging
	LogLevel = GetEnvOrDefault("LOG_LEVEL", "info")

	return nil
}

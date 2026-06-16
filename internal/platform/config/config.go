package config

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/joho/godotenv"
	"github.com/maintainerd/auth/internal/platform/signedurl"
)

// Package-level configuration variables are populated exactly once by Init()
// at application startup and are read-only thereafter. Tests that need to
// override values should use t.Setenv before calling Init() or manipulate
// the exported vars directly via a test helper.
var (
	// APP
	AppEnv             string // "development" or "production"; defaults "development"
	AppVersion         string
	AppPublicHostname  string
	AppPrivateHostname string
	ManagementPort     string // MANAGEMENT_PORT; default ":8082"
	GRPCTLSCertFile    string
	GRPCTLSKeyFile     string
	GRPCClientCAFile   string
	GRPCRequireMTLS    bool

	// Application Encryption Key (AES-256)
	AppEncryptionKey []byte

	// Logging
	LogLevel string // "debug", "info", "warn", "error"; defaults "info"

	// FRONTEND
	AppFrontendIdentityHostname string
	AppFrontendConsoleHostname  string

	// JWT Configuration
	JWTPrivateKey               []byte
	JWTPublicKey                []byte
	JWTKeyRotationPeriodSeconds int
	SecretRefreshPeriodSeconds  int

	// HMAC Secret for signed URLs
	HMACSecretKey []byte

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
)

// Init loads all configuration from environment variables (and an optional .env file).
// It returns an error for any missing required variable so that main() can decide
// how to handle the failure — nothing in this package calls os.Exit.
func Init() error {
	// Load environment variables first (best-effort; not required in production)
	if err := godotenv.Load(); err != nil {
		slog.Warn("no .env file found, using system environment", "err", err)
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
	ManagementPort = normalizeListenAddr(GetEnvOrDefault("MANAGEMENT_PORT", "8082"))
	GRPCTLSCertFile = GetEnvOrDefault("GRPC_TLS_CERT_FILE", "")
	GRPCTLSKeyFile = GetEnvOrDefault("GRPC_TLS_KEY_FILE", "")
	GRPCClientCAFile = GetEnvOrDefault("GRPC_CLIENT_CA_FILE", "")
	GRPCRequireMTLS = strings.EqualFold(GetEnvOrDefault("GRPC_REQUIRE_MTLS", "false"), "true")

	// Frontend Config
	if AppFrontendIdentityHostname, err = GetEnv("APP_FRONTEND_IDENTITY_HOSTNAME"); err != nil {
		return err
	}
	if AppFrontendConsoleHostname, err = GetEnv("APP_FRONTEND_CONSOLE_HOSTNAME"); err != nil {
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
	JWTKeyRotationPeriodSeconds = parseIntDefault(GetEnvOrDefault("JWT_KEY_ROTATION_PERIOD_SECONDS", "86400"), 86400)
	SecretRefreshPeriodSeconds = parseIntDefault(GetEnvOrDefault("SECRET_REFRESH_PERIOD_SECONDS", "300"), 300)

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

	// HMAC secret key for signed URLs — loaded via the configured secret provider.
	slog.Info("Loading HMAC secret key from secret provider")
	if HMACSecretKey, err = loadSecret("HMAC_SECRET_KEY"); err != nil {
		return fmt.Errorf("failed to load HMAC_SECRET_KEY: %w", err)
	}
	if err := signedurl.Configure(HMACSecretKey); err != nil {
		return fmt.Errorf("failed to configure signed URL signer: %w", err)
	}
	slog.Info("HMAC secret key loaded successfully")

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

	// Cookie Config
	CookieSecure = GetEnvOrDefault("COOKIE_SECURE", "true") != "false"
	CookieSameSite = GetEnvOrDefault("COOKIE_SAMESITE", "strict")

	// Logging
	LogLevel = GetEnvOrDefault("LOG_LEVEL", "info")

	return nil
}

type Config struct {
	AppEnv             string
	AppVersion         string
	AppPublicHostname  string
	AppPrivateHostname string
	ManagementPort     string
	GRPCTLSCertFile    string
	GRPCTLSKeyFile     string
	GRPCClientCAFile   string
	GRPCRequireMTLS    bool
	AppEncryptionKey   []byte

	LogLevel string

	AppFrontendIdentityHostname string
	AppFrontendConsoleHostname  string

	JWTPrivateKey               []byte
	JWTPublicKey                []byte
	JWTKeyRotationPeriodSeconds int
	SecretRefreshPeriodSeconds  int
	HMACSecretKey               []byte

	SecretProvider string
	SecretPrefix   string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	DBMaxOpenConns       int
	DBMaxIdleConns       int
	DBConnMaxLifetimeSec int
	DBStatementTimeoutMs int

	CookieSecure   bool
	CookieSameSite string
}

func GetConfig() Config {
	return Config{
		AppEnv:                      AppEnv,
		AppVersion:                  AppVersion,
		AppPublicHostname:           AppPublicHostname,
		AppPrivateHostname:          AppPrivateHostname,
		ManagementPort:              ManagementPort,
		GRPCTLSCertFile:             GRPCTLSCertFile,
		GRPCTLSKeyFile:              GRPCTLSKeyFile,
		GRPCClientCAFile:            GRPCClientCAFile,
		GRPCRequireMTLS:             GRPCRequireMTLS,
		AppEncryptionKey:            AppEncryptionKey,
		LogLevel:                    LogLevel,
		AppFrontendIdentityHostname: AppFrontendIdentityHostname,
		AppFrontendConsoleHostname:  AppFrontendConsoleHostname,
		JWTPrivateKey:               JWTPrivateKey,
		JWTPublicKey:                JWTPublicKey,
		JWTKeyRotationPeriodSeconds: JWTKeyRotationPeriodSeconds,
		SecretRefreshPeriodSeconds:  SecretRefreshPeriodSeconds,
		HMACSecretKey:               HMACSecretKey,
		SecretProvider:              SecretProvider,
		SecretPrefix:                SecretPrefix,
		DBHost:                      DBHost,
		DBPort:                      DBPort,
		DBUser:                      DBUser,
		DBPassword:                  DBPassword,
		DBName:                      DBName,
		DBSSLMode:                   DBSSLMode,
		DBMaxOpenConns:              DBMaxOpenConns,
		DBMaxIdleConns:              DBMaxIdleConns,
		DBConnMaxLifetimeSec:        DBConnMaxLifetimeSec,
		DBStatementTimeoutMs:        DBStatementTimeoutMs,
		CookieSecure:                CookieSecure,
		CookieSameSite:              CookieSameSite,
	}
}

func normalizeListenAddr(port string) string {
	port = strings.TrimSpace(port)
	if port == "" {
		return ""
	}
	if strings.Contains(port, ":") {
		return port
	}
	return ":" + port
}

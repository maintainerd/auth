package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	// setRequiredEnv sets the minimum env vars needed for Init() to succeed
	// with the "env" secret provider.
	setRequiredEnv := func(t *testing.T) {
		t.Helper()
		t.Setenv("SECRET_PROVIDER", "env")
		t.Setenv("APP_VERSION", "1.0.0")
		t.Setenv("APP_PUBLIC_HOSTNAME", "https://pub.example.com")
		t.Setenv("APP_PRIVATE_HOSTNAME", "https://priv.example.com")
		t.Setenv("ACCOUNT_HOSTNAME", "https://account.example.com")
		t.Setenv("AUTH_HOSTNAME", "https://auth.example.com")
		t.Setenv("JWT_PRIVATE_KEY", "private-key-data")
		t.Setenv("JWT_PUBLIC_KEY", "public-key-data")
		t.Setenv("APP_ENCRYPTION_KEY", "12345678901234567890123456789012")
		t.Setenv("HMAC_SECRET_KEY", "hmac-secret-data")
		t.Setenv("DB_HOST", "localhost")
		t.Setenv("DB_PORT", "5432")
		t.Setenv("DB_USER", "postgres")
		t.Setenv("DB_PASSWORD", "pass")
		t.Setenv("DB_NAME", "authdb")
		t.Setenv("SMTP_HOST", "smtp.example.com")
		t.Setenv("SMTP_PORT", "587")
		t.Setenv("SMTP_USER", "user")
		t.Setenv("SMTP_PASS", "pass")
	}

	saveGlobals := func(t *testing.T) {
		t.Helper()
		origSM := activeSecretManager
		origProvider := SecretProvider
		origPrefix := SecretPrefix
		origAppVersion := AppVersion
		origAppPubHost := AppPublicHostname
		origAppPrivHost := AppPrivateHostname
		origManagementPort := ManagementPort
		origAccountHost := AccountHostname
		origAuthHost := AuthHostname
		origJWTPriv := JWTPrivateKey
		origJWTPub := JWTPublicKey
		origDBHost := DBHost
		origDBPort := DBPort
		origDBUser := DBUser
		origDBPass := DBPassword
		origDBName := DBName
		origDBSSL := DBSSLMode
		origSMTPHost := SMTPHost
		origSMTPPort := SMTPPort
		origSMTPUser := SMTPUser
		origSMTPPass := SMTPPass
		origSMTPFromEmail := SMTPFromEmail
		origSMTPFromName := SMTPFromName
		origEmailLogo := EmailLogo
		origEncKey := AppEncryptionKey
		origHMACKey := HMACSecretKey
		t.Cleanup(func() {
			activeSecretManager = origSM
			SecretProvider = origProvider
			SecretPrefix = origPrefix
			AppVersion = origAppVersion
			AppPublicHostname = origAppPubHost
			AppPrivateHostname = origAppPrivHost
			ManagementPort = origManagementPort
			AccountHostname = origAccountHost
			AuthHostname = origAuthHost
			JWTPrivateKey = origJWTPriv
			JWTPublicKey = origJWTPub
			DBHost = origDBHost
			DBPort = origDBPort
			DBUser = origDBUser
			DBPassword = origDBPass
			DBName = origDBName
			DBSSLMode = origDBSSL
			SMTPHost = origSMTPHost
			SMTPPort = origSMTPPort
			SMTPUser = origSMTPUser
			SMTPPass = origSMTPPass
			SMTPFromEmail = origSMTPFromEmail
			SMTPFromName = origSMTPFromName
			EmailLogo = origEmailLogo
			AppEncryptionKey = origEncKey
			HMACSecretKey = origHMACKey
		})
	}

	t.Run("success with all required vars", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)

		err := Init()
		require.NoError(t, err)

		assert.Equal(t, "1.0.0", AppVersion)
		assert.Equal(t, "https://pub.example.com", AppPublicHostname)
		assert.Equal(t, "https://priv.example.com", AppPrivateHostname)
		assert.Equal(t, ":8082", ManagementPort)
		assert.Equal(t, "https://account.example.com", AccountHostname)
		assert.Equal(t, "https://auth.example.com", AuthHostname)
		assert.Equal(t, []byte("private-key-data"), JWTPrivateKey)
		assert.Equal(t, []byte("public-key-data"), JWTPublicKey)
		assert.Equal(t, "localhost", DBHost)
		assert.Equal(t, "5432", DBPort)
		assert.Equal(t, "postgres", DBUser)
		assert.Equal(t, "pass", DBPassword)
		assert.Equal(t, "authdb", DBName)
		assert.Equal(t, "disable", DBSSLMode)
		assert.Equal(t, "smtp.example.com", SMTPHost)
		assert.Equal(t, 587, SMTPPort)
		assert.Equal(t, "user", SMTPUser)
		assert.Equal(t, "pass", SMTPPass)
		assert.Equal(t, []byte("12345678901234567890123456789012"), AppEncryptionKey)
	})

	t.Run("invalid secret provider", func(t *testing.T) {
		saveGlobals(t)
		t.Setenv("SECRET_PROVIDER", "bad_provider")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "secret provider validation failed")
	})

	t.Run("initSecretManager failure", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("SECRET_PROVIDER", "gcp")
		t.Setenv("GCP_PROJECT_ID", "test-project")
		// GCP client creation will fail without credentials

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize secret manager")
	})

	t.Run("missing APP_VERSION", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("APP_VERSION", "")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "APP_VERSION")
	})

	t.Run("missing APP_PUBLIC_HOSTNAME", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("APP_PUBLIC_HOSTNAME", "")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "APP_PUBLIC_HOSTNAME")
	})

	t.Run("missing APP_PRIVATE_HOSTNAME", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("APP_PRIVATE_HOSTNAME", "")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "APP_PRIVATE_HOSTNAME")
	})

	t.Run("missing ACCOUNT_HOSTNAME", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("ACCOUNT_HOSTNAME", "")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ACCOUNT_HOSTNAME")
	})

	t.Run("missing AUTH_HOSTNAME", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("AUTH_HOSTNAME", "")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "AUTH_HOSTNAME")
	})

	t.Run("missing JWT_PRIVATE_KEY", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("JWT_PRIVATE_KEY", "")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "JWT private key")
	})

	t.Run("missing JWT_PUBLIC_KEY", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("JWT_PUBLIC_KEY", "")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "JWT public key")
	})

	t.Run("missing DB_HOST", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("DB_HOST", "")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DB_HOST")
	})

	t.Run("missing DB_PORT", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("DB_PORT", "")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DB_PORT")
	})

	t.Run("missing DB_USER", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("DB_USER", "")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DB_USER")
	})

	t.Run("missing DB_PASSWORD", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("DB_PASSWORD", "")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DB_PASSWORD")
	})

	t.Run("missing DB_NAME", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("DB_NAME", "")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DB_NAME")
	})

	t.Run("missing SMTP_HOST", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("SMTP_HOST", "")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SMTP_HOST")
	})

	t.Run("missing SMTP_PORT", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("SMTP_PORT", "")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SMTP_PORT")
	})

	t.Run("invalid SMTP_PORT", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("SMTP_PORT", "not-a-number")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid SMTP_PORT")
	})

	t.Run("missing SMTP_USER", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("SMTP_USER", "")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SMTP_USER")
	})

	t.Run("missing SMTP_PASS", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("SMTP_PASS", "")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SMTP_PASS")
	})

	t.Run("defaults for optional vars", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)

		err := Init()
		require.NoError(t, err)

		assert.Equal(t, "noreply@maintainerd.com", SMTPFromEmail)
		assert.Equal(t, "Maintainerd", SMTPFromName)
		assert.NotEmpty(t, EmailLogo)
	})

	t.Run("APP_ENCRYPTION_KEY wrong size", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("APP_ENCRYPTION_KEY", "tooshort")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "APP_ENCRYPTION_KEY must be 32 bytes")
	})

	t.Run("missing APP_ENCRYPTION_KEY", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("APP_ENCRYPTION_KEY", "")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "APP_ENCRYPTION_KEY")
	})

	t.Run("missing HMAC_SECRET_KEY", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("HMAC_SECRET_KEY", "")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "HMAC_SECRET_KEY")
	})

	t.Run("custom SMS_DAILY_SEND_LIMIT", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("SMS_DAILY_SEND_LIMIT", "500")

		err := Init()
		require.NoError(t, err)
		assert.Equal(t, 500, SMSDailySendLimit)
	})

	t.Run("SMS_DAILY_SEND_LIMIT disabled", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("SMS_DAILY_SEND_LIMIT", "0")

		err := Init()
		require.NoError(t, err)
		assert.Equal(t, 0, SMSDailySendLimit)
	})

	t.Run("custom MANAGEMENT_PORT", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("MANAGEMENT_PORT", "9090")

		err := Init()
		require.NoError(t, err)
		assert.Equal(t, ":9090", ManagementPort)
	})

	t.Run("custom LOG_LEVEL", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("LOG_LEVEL", "debug")

		err := Init()
		require.NoError(t, err)
		assert.Equal(t, "debug", LogLevel)
	})

	t.Run("custom COOKIE_SECURE false", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("COOKIE_SECURE", "false")

		err := Init()
		require.NoError(t, err)
		assert.False(t, CookieSecure)
	})

	t.Run("custom COOKIE_SECURE true", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("COOKIE_SECURE", "true")

		err := Init()
		require.NoError(t, err)
		assert.True(t, CookieSecure)
	})

	t.Run("custom COOKIE_SAMESITE lax", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("COOKIE_SAMESITE", "lax")

		err := Init()
		require.NoError(t, err)
		assert.Equal(t, "lax", CookieSameSite)
	})

	t.Run("custom EMAIL_PROVIDER", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("EMAIL_PROVIDER", "mailgun")

		err := Init()
		require.NoError(t, err)
		assert.Equal(t, "mailgun", EmailProvider)
	})

	t.Run("custom DB pool config", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("DB_MAX_OPEN_CONNS", "50")
		t.Setenv("DB_MAX_IDLE_CONNS", "20")
		t.Setenv("DB_CONN_MAX_LIFETIME_SEC", "600")

		err := Init()
		require.NoError(t, err)
		assert.Equal(t, 50, DBMaxOpenConns)
		assert.Equal(t, 20, DBMaxIdleConns)
		assert.Equal(t, 600, DBConnMaxLifetimeSec)
	})

	t.Run("custom EMAIL_API_KEY set", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("EMAIL_API_KEY", "key-123")

		err := Init()
		require.NoError(t, err)
		assert.Equal(t, "key-123", EmailAPIKey)
	})
}

func TestGetConfig(t *testing.T) {
	saveGlobals := func(t *testing.T) {
		t.Helper()
		orig := AppEnv
		t.Cleanup(func() { AppEnv = orig })
	}
	saveGlobals(t)
	AppEnv = "production"

	cfg := GetConfig()
	assert.Equal(t, "production", cfg.AppEnv)
	assert.NotNil(t, cfg)
}

func TestNormalizeListenAddr(t *testing.T) {
	tests := []struct {
		name string
		port string
		want string
	}{
		{"with colon", ":8080", ":8080"},
		{"without colon", "8080", ":8080"},
		{"empty", "", ""},
		{"whitespace trimmed", " 9090 ", ":9090"},
		{"already has colon and path", "0.0.0.0:8080", "0.0.0.0:8080"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, normalizeListenAddr(tc.port))
		})
	}
}

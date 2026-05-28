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
		t.Cleanup(func() {
			activeSecretManager = origSM
			SecretProvider = origProvider
			SecretPrefix = origPrefix
			AppVersion = origAppVersion
			AppPublicHostname = origAppPubHost
			AppPrivateHostname = origAppPrivHost
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
}

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
		t.Setenv("APP_FRONTEND_IDENTITY_HOSTNAME", "https://account.example.com")
		t.Setenv("APP_FRONTEND_CONSOLE_HOSTNAME", "https://auth.example.com")
		t.Setenv("JWT_PRIVATE_KEY", "private-key-data")
		t.Setenv("JWT_PUBLIC_KEY", "public-key-data")
		t.Setenv("APP_ENCRYPTION_KEY", "12345678901234567890123456789012")
		t.Setenv("HMAC_SECRET_KEY", "hmac-secret-data")
		t.Setenv("DB_HOST", "localhost")
		t.Setenv("DB_PORT", "5432")
		t.Setenv("DB_USER", "postgres")
		t.Setenv("DB_PASSWORD", "pass")
		t.Setenv("DB_NAME", "authdb")
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
		origAccountHost := AppFrontendIdentityHostname
		origAuthHost := AppFrontendConsoleHostname
		origJWTPriv := JWTPrivateKey
		origJWTPub := JWTPublicKey
		origDBHost := DBHost
		origDBPort := DBPort
		origDBUser := DBUser
		origDBPass := DBPassword
		origDBName := DBName
		origDBSSL := DBSSLMode
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
			AppFrontendIdentityHostname = origAccountHost
			AppFrontendConsoleHostname = origAuthHost
			JWTPrivateKey = origJWTPriv
			JWTPublicKey = origJWTPub
			DBHost = origDBHost
			DBPort = origDBPort
			DBUser = origDBUser
			DBPassword = origDBPass
			DBName = origDBName
			DBSSLMode = origDBSSL
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
		assert.Equal(t, "https://account.example.com", AppFrontendIdentityHostname)
		assert.Equal(t, "https://auth.example.com", AppFrontendConsoleHostname)
		assert.Equal(t, []byte("private-key-data"), JWTPrivateKey)
		assert.Equal(t, []byte("public-key-data"), JWTPublicKey)
		assert.Equal(t, "localhost", DBHost)
		assert.Equal(t, "5432", DBPort)
		assert.Equal(t, "postgres", DBUser)
		assert.Equal(t, "pass", DBPassword)
		assert.Equal(t, "authdb", DBName)
		assert.Equal(t, "disable", DBSSLMode)
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

	t.Run("missing APP_VERSION defaults instead of erroring", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("APP_VERSION", "")
		AppVersion = "" // no build-injected value

		err := Init()
		require.NoError(t, err)
		assert.Equal(t, "dev", AppVersion)
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
		t.Setenv("APP_FRONTEND_IDENTITY_HOSTNAME", "")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "APP_FRONTEND_IDENTITY_HOSTNAME")
	})

	t.Run("missing AUTH_HOSTNAME", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)
		t.Setenv("APP_FRONTEND_CONSOLE_HOSTNAME", "")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "APP_FRONTEND_CONSOLE_HOSTNAME")
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

	t.Run("defaults for optional vars", func(t *testing.T) {
		saveGlobals(t)
		setRequiredEnv(t)

		err := Init()
		require.NoError(t, err)

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

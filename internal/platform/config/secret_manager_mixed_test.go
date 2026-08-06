package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The mixed model: non-sensitive configuration stays in plain environment
// variables while credentials follow SECRET_PROVIDER. That only works if
// credentials actually route through the provider — DB_PASSWORD, REDIS_PASSWORD,
// SETUP_BOOTSTRAP_TOKEN and APP_ENCRYPTION_KEYS_PREVIOUS used to call os.Getenv
// directly, so an operator running Vault or Secrets Manager still had to leave
// them in the environment.
//
// With SECRET_PROVIDER=env this must stay byte-identical to reading the
// variable, so the default deployment is unaffected.
func TestCredentialsRouteThroughProvider(t *testing.T) {
	t.Run("env provider reads the variable exactly as before", func(t *testing.T) {
		useEnvSecretManager(t)
		t.Setenv("DB_PASSWORD", "s3cr3t")

		got, err := LoadSecretString("DB_PASSWORD")
		require.NoError(t, err)
		assert.Equal(t, "s3cr3t", got)
	})

	t.Run("a required credential that is unset is an error", func(t *testing.T) {
		useEnvSecretManager(t)
		t.Setenv("DB_PASSWORD", "")

		_, err := LoadSecretString("DB_PASSWORD")
		require.Error(t, err)
	})

	// An empty Redis password is a legitimate local setup, so optional
	// credentials must resolve to "" rather than failing startup.
	t.Run("an optional credential that is unset resolves empty", func(t *testing.T) {
		useEnvSecretManager(t)
		t.Setenv("REDIS_PASSWORD", "")

		got, err := LoadSecretStringOptional("REDIS_PASSWORD")
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("an optional credential that is set is returned", func(t *testing.T) {
		useEnvSecretManager(t)
		t.Setenv("REDIS_PASSWORD", "hunter2")

		got, err := LoadSecretStringOptional("REDIS_PASSWORD")
		require.NoError(t, err)
		assert.Equal(t, "hunter2", got)
	})

	// A remote store being unreachable must NOT be read as "this credential is
	// intentionally unset" — that would silently start the app with no Redis
	// password against a store that requires one.
	t.Run("a remote provider failure is an error, not an absent secret", func(t *testing.T) {
		saveActiveSecretManager(t)
		activeSecretManager = &failingSecretManager{}

		_, err := LoadSecretStringOptional("REDIS_PASSWORD")
		require.Error(t, err, "an outage must not be mistaken for an unset credential")
	})

	t.Run("credentials honour the same base64 handling as other secrets", func(t *testing.T) {
		useEnvSecretManager(t)
		t.Setenv("DB_PASSWORD", "base64:aHVudGVyMg==")

		got, err := LoadSecretString("DB_PASSWORD")
		require.NoError(t, err)
		assert.Equal(t, "hunter2", got)
	})
}

// failingSecretManager stands in for an unreachable remote store.
type failingSecretManager struct{}

func (f *failingSecretManager) GetSecret(string) ([]byte, error) {
	return nil, assertRemoteOutage{}
}
func (f *failingSecretManager) GetSecretString(string) (string, error) {
	return "", assertRemoteOutage{}
}

type assertRemoteOutage struct{}

func (assertRemoteOutage) Error() string { return "vault: connection refused" }

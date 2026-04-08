package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── envSecretManager ────────────────────────────────────────────────────

func TestEnvSecretManager_GetSecret(t *testing.T) {
	sm := &envSecretManager{}

	t.Run("returns plain value", func(t *testing.T) {
		t.Setenv("TEST_SM_PLAIN", "my-secret")
		data, err := sm.GetSecret("TEST_SM_PLAIN")
		require.NoError(t, err)
		assert.Equal(t, []byte("my-secret"), data)
	})

	t.Run("decodes base64 prefix", func(t *testing.T) {
		raw := "hello world"
		encoded := base64.StdEncoding.EncodeToString([]byte(raw))
		t.Setenv("TEST_SM_B64", "base64:"+encoded)
		data, err := sm.GetSecret("TEST_SM_B64")
		require.NoError(t, err)
		assert.Equal(t, []byte(raw), data)
	})

	t.Run("error on invalid base64", func(t *testing.T) {
		t.Setenv("TEST_SM_B64_BAD", "base64:!!!invalid!!!")
		_, err := sm.GetSecret("TEST_SM_B64_BAD")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode base64")
	})

	t.Run("error when not set", func(t *testing.T) {
		_, err := sm.GetSecret("TEST_SM_NOT_SET")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not set")
	})
}

func TestEnvSecretManager_GetSecretString(t *testing.T) {
	sm := &envSecretManager{}

	t.Run("returns string value", func(t *testing.T) {
		t.Setenv("TEST_SM_STR", "value")
		val, err := sm.GetSecretString("TEST_SM_STR")
		require.NoError(t, err)
		assert.Equal(t, "value", val)
	})

	t.Run("propagates error", func(t *testing.T) {
		_, err := sm.GetSecretString("TEST_SM_STR_MISSING")
		require.Error(t, err)
	})
}

// ─── fileSecretManager ───────────────────────────────────────────────────

func TestFileSecretManager_GetSecret(t *testing.T) {
	dir := t.TempDir()
	sm := &fileSecretManager{basePath: dir}

	t.Run("reads file content", func(t *testing.T) {
		err := os.WriteFile(filepath.Join(dir, "jwt-private-key"), []byte("secret-data"), 0600)
		require.NoError(t, err)

		data, err := sm.GetSecret("JWT_PRIVATE_KEY")
		require.NoError(t, err)
		assert.Equal(t, []byte("secret-data"), data)
	})

	t.Run("error when file missing", func(t *testing.T) {
		_, err := sm.GetSecret("NONEXISTENT_KEY")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read secret file")
	})
}

func TestFileSecretManager_GetSecretString(t *testing.T) {
	dir := t.TempDir()
	sm := &fileSecretManager{basePath: dir}

	t.Run("trims whitespace", func(t *testing.T) {
		err := os.WriteFile(filepath.Join(dir, "my-key"), []byte("  data \n"), 0600)
		require.NoError(t, err)

		val, err := sm.GetSecretString("MY_KEY")
		require.NoError(t, err)
		assert.Equal(t, "data", val)
	})

	t.Run("propagates error", func(t *testing.T) {
		_, err := sm.GetSecretString("MISSING_KEY")
		require.Error(t, err)
	})
}

// ─── ValidateSecretProvider ──────────────────────────────────────────────

func TestValidateSecretProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		wantErr  bool
	}{
		{"env", "env", false},
		{"file", "file", false},
		{"aws_secrets", "aws_secrets", false},
		{"aws_ssm", "aws_ssm", false},
		{"vault", "vault", false},
		{"gcp", "gcp", false},
		{"azure_kv", "azure_kv", false},
		{"invalid", "unknown", true},
		{"empty", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			origProvider := SecretProvider
			t.Cleanup(func() { SecretProvider = origProvider })

			SecretProvider = tc.provider
			err := ValidateSecretProvider()
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid SECRET_PROVIDER")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── loadSecret ──────────────────────────────────────────────────────────

func TestLoadSecret(t *testing.T) {
	t.Run("nil manager returns error", func(t *testing.T) {
		saveActiveSecretManager(t)
		activeSecretManager = nil

		_, err := loadSecret("ANY_KEY")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("success on first attempt", func(t *testing.T) {
		saveActiveSecretManager(t)
		activeSecretManager = &mockSecretManager{
			getSecretFn: func(key string) ([]byte, error) {
				return []byte("value"), nil
			},
		}

		data, err := loadSecret("MY_KEY")
		require.NoError(t, err)
		assert.Equal(t, []byte("value"), data)
	})

	t.Run("empty secret returns error", func(t *testing.T) {
		saveActiveSecretManager(t)
		activeSecretManager = &mockSecretManager{
			getSecretFn: func(key string) ([]byte, error) {
				return []byte{}, nil
			},
		}

		_, err := loadSecret("EMPTY_KEY")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is empty")
	})

	t.Run("retries on failure and succeeds", func(t *testing.T) {
		saveActiveSecretManager(t)
		callCount := 0
		activeSecretManager = &mockSecretManager{
			getSecretFn: func(key string) ([]byte, error) {
				callCount++
				if callCount < 2 {
					return nil, fmt.Errorf("transient error")
				}
				return []byte("recovered"), nil
			},
		}

		data, err := loadSecret("RETRY_KEY")
		require.NoError(t, err)
		assert.Equal(t, []byte("recovered"), data)
		assert.Equal(t, 2, callCount)
	})

	t.Run("fails after 3 attempts", func(t *testing.T) {
		saveActiveSecretManager(t)
		activeSecretManager = &mockSecretManager{
			getSecretFn: func(key string) ([]byte, error) {
				return nil, fmt.Errorf("persistent error")
			},
		}

		_, err := loadSecret("FAIL_KEY")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "after 3 attempts")
	})
}

// ─── initSecretManager ──────────────────────────────────────────────────

func TestInitSecretManager(t *testing.T) {
	t.Run("sets activeSecretManager for env provider", func(t *testing.T) {
		saveActiveSecretManager(t)
		origProvider := SecretProvider
		t.Cleanup(func() { SecretProvider = origProvider })

		SecretProvider = "env"
		err := initSecretManager()
		require.NoError(t, err)
		require.NotNil(t, activeSecretManager)
		_, ok := activeSecretManager.(*envSecretManager)
		assert.True(t, ok)
	})

	t.Run("returns error on failure", func(t *testing.T) {
		saveActiveSecretManager(t)
		origProvider := SecretProvider
		t.Cleanup(func() { SecretProvider = origProvider })

		activeSecretManager = nil
		SecretProvider = "gcp"
		// GCP_PROJECT_ID not set → newSecretManager fails

		err := initSecretManager()
		require.Error(t, err)
	})
}

// ─── newSecretManager ────────────────────────────────────────────────────

func TestNewSecretManager(t *testing.T) {
	t.Run("env provider", func(t *testing.T) {
		origProvider := SecretProvider
		t.Cleanup(func() { SecretProvider = origProvider })
		SecretProvider = "env"

		sm, err := newSecretManager()
		require.NoError(t, err)
		_, ok := sm.(*envSecretManager)
		assert.True(t, ok)
	})

	t.Run("file provider", func(t *testing.T) {
		origProvider := SecretProvider
		t.Cleanup(func() { SecretProvider = origProvider })
		SecretProvider = "file"
		t.Setenv("SECRET_FILE_PATH", "/tmp/test-secrets")

		sm, err := newSecretManager()
		require.NoError(t, err)
		fsm, ok := sm.(*fileSecretManager)
		require.True(t, ok)
		assert.Equal(t, "/tmp/test-secrets", fsm.basePath)
	})

	t.Run("file provider uses default path", func(t *testing.T) {
		origProvider := SecretProvider
		t.Cleanup(func() { SecretProvider = origProvider })
		SecretProvider = "file"

		sm, err := newSecretManager()
		require.NoError(t, err)
		fsm, ok := sm.(*fileSecretManager)
		require.True(t, ok)
		assert.Equal(t, "/run/secrets", fsm.basePath)
	})

	t.Run("unknown provider falls back to env", func(t *testing.T) {
		origProvider := SecretProvider
		t.Cleanup(func() { SecretProvider = origProvider })
		SecretProvider = "unknown_provider"

		sm, err := newSecretManager()
		require.NoError(t, err)
		_, ok := sm.(*envSecretManager)
		assert.True(t, ok)
	})

	t.Run("gcp without project ID returns error", func(t *testing.T) {
		origProvider := SecretProvider
		t.Cleanup(func() { SecretProvider = origProvider })
		SecretProvider = "gcp"
		// GCP_PROJECT_ID not set

		_, err := newSecretManager()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GCP_PROJECT_ID")
	})

	t.Run("azure_kv without vault URL returns error", func(t *testing.T) {
		origProvider := SecretProvider
		t.Cleanup(func() { SecretProvider = origProvider })
		SecretProvider = "azure_kv"
		// AZURE_KEYVAULT_URL not set

		_, err := newSecretManager()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "AZURE_KEYVAULT_URL")
	})

	t.Run("vault with token", func(t *testing.T) {
		origProvider := SecretProvider
		t.Cleanup(func() { SecretProvider = origProvider })
		SecretProvider = "vault"
		t.Setenv("VAULT_TOKEN", "test-token")

		sm, err := newSecretManager()
		require.NoError(t, err)
		_, ok := sm.(*vaultSecretManager)
		assert.True(t, ok)
	})

	t.Run("vault without token or approle creds returns error", func(t *testing.T) {
		origProvider := SecretProvider
		t.Cleanup(func() { SecretProvider = origProvider })
		SecretProvider = "vault"
		// VAULT_TOKEN, VAULT_ROLE_ID, VAULT_SECRET_ID not set

		_, err := newSecretManager()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "AppRole login failed")
	})

	t.Run("aws_secrets provider", func(t *testing.T) {
		origProvider := SecretProvider
		origPrefix := SecretPrefix
		t.Cleanup(func() {
			SecretProvider = origProvider
			SecretPrefix = origPrefix
		})
		SecretProvider = "aws_secrets"
		SecretPrefix = "test/prefix"

		sm, err := newSecretManager()
		require.NoError(t, err)
		asm, ok := sm.(*awsSecretsManager)
		require.True(t, ok)
		assert.Equal(t, "test/prefix", asm.prefix)
	})

	t.Run("aws_ssm provider", func(t *testing.T) {
		origProvider := SecretProvider
		origPrefix := SecretPrefix
		t.Cleanup(func() {
			SecretProvider = origProvider
			SecretPrefix = origPrefix
		})
		SecretProvider = "aws_ssm"
		SecretPrefix = "test/prefix"

		sm, err := newSecretManager()
		require.NoError(t, err)
		ssm, ok := sm.(*awsSSMSecretManager)
		require.True(t, ok)
		assert.Equal(t, "test/prefix", ssm.prefix)
	})

	t.Run("gcp with project ID", func(t *testing.T) {
		origProvider := SecretProvider
		t.Cleanup(func() { SecretProvider = origProvider })
		SecretProvider = "gcp"
		t.Setenv("GCP_PROJECT_ID", "test-project")

		// Will error because no GCP credentials, but covers the branch
		_, err := newSecretManager()
		require.Error(t, err)
	})

	t.Run("azure_kv with vault URL", func(t *testing.T) {
		origProvider := SecretProvider
		t.Cleanup(func() { SecretProvider = origProvider })
		SecretProvider = "azure_kv"
		t.Setenv("AZURE_KEYVAULT_URL", "https://test.vault.azure.net")

		// May succeed or fail depending on credential chain availability
		_, _ = newSecretManager()
	})
}

package config

import (
	"context"
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── newAzureKeyVaultManager ─────────────────────────────────────────────

func TestNewAzureKeyVaultManager(t *testing.T) {
	t.Run("success with mock client", func(t *testing.T) {
		orig := newAzureClient
		t.Cleanup(func() { newAzureClient = orig })
		newAzureClient = func(_ string) (azureSecretsClient, error) {
			return &mockAzureSecretsClient{}, nil
		}

		sm, err := newAzureKeyVaultManager("https://test.vault.azure.net")
		require.NoError(t, err)
		require.NotNil(t, sm)
	})

	t.Run("client creation error", func(t *testing.T) {
		orig := newAzureClient
		t.Cleanup(func() { newAzureClient = orig })
		newAzureClient = func(_ string) (azureSecretsClient, error) {
			return nil, fmt.Errorf("no credentials")
		}

		_, err := newAzureKeyVaultManager("https://test.vault.azure.net")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no credentials")
	})
}

// ─── secretName ─────────────────────────────────────────────────────────

func TestAzureKeyVaultManager_SecretName(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{"standard key", "JWT_PRIVATE_KEY", "jwt-private-key"},
		{"simple key", "DB_PASSWORD", "db-password"},
		{"single word", "SECRET", "secret"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sm := &azureKeyVaultManager{}
			assert.Equal(t, tc.want, sm.secretName(tc.key))
		})
	}
}

// ─── GetSecret ──────────────────────────────────────────────────────────

func TestAzureKeyVaultManager_GetSecret(t *testing.T) {
	t.Run("returns secret value", func(t *testing.T) {
		sm := &azureKeyVaultManager{
			client: &mockAzureSecretsClient{
				getSecretFn: func(_ context.Context, name string, version string) (azsecrets.GetSecretResponse, error) {
					assert.Equal(t, "my-key", name)
					assert.Equal(t, "", version)
					return azsecrets.GetSecretResponse{
						Secret: azsecrets.Secret{
							Value: stringPtr("azure-secret"),
						},
					}, nil
				},
			},
		}

		data, err := sm.GetSecret("MY_KEY")
		require.NoError(t, err)
		assert.Equal(t, []byte("azure-secret"), data)
	})

	t.Run("nil value", func(t *testing.T) {
		sm := &azureKeyVaultManager{
			client: &mockAzureSecretsClient{
				getSecretFn: func(_ context.Context, _ string, _ string) (azsecrets.GetSecretResponse, error) {
					return azsecrets.GetSecretResponse{
						Secret: azsecrets.Secret{Value: nil},
					}, nil
				},
			},
		}

		_, err := sm.GetSecret("K")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "has no value")
	})

	t.Run("api error", func(t *testing.T) {
		sm := &azureKeyVaultManager{
			client: &mockAzureSecretsClient{
				getSecretFn: func(_ context.Context, _ string, _ string) (azsecrets.GetSecretResponse, error) {
					return azsecrets.GetSecretResponse{}, fmt.Errorf("unauthorized")
				},
			},
		}

		_, err := sm.GetSecret("K")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get secret")
	})
}

func TestAzureKeyVaultManager_GetSecretString(t *testing.T) {
	t.Run("returns string", func(t *testing.T) {
		sm := &azureKeyVaultManager{
			client: &mockAzureSecretsClient{
				getSecretFn: func(_ context.Context, _ string, _ string) (azsecrets.GetSecretResponse, error) {
					return azsecrets.GetSecretResponse{
						Secret: azsecrets.Secret{Value: stringPtr("val")},
					}, nil
				},
			},
		}

		val, err := sm.GetSecretString("K")
		require.NoError(t, err)
		assert.Equal(t, "val", val)
	})

	t.Run("propagates error", func(t *testing.T) {
		sm := &azureKeyVaultManager{
			client: &mockAzureSecretsClient{
				getSecretFn: func(_ context.Context, _ string, _ string) (azsecrets.GetSecretResponse, error) {
					return azsecrets.GetSecretResponse{}, fmt.Errorf("fail")
				},
			},
		}

		_, err := sm.GetSecretString("K")
		require.Error(t, err)
	})
}

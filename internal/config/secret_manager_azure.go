package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

// ──────────────────────────────────── Azure Key Vault provider ─────────────
//
// Configuration env vars:
//   AZURE_KEYVAULT_URL – Key Vault endpoint URL, e.g. https://my-vault.vault.azure.net (required)
//
// Authentication: DefaultAzureCredential, which tries in order:
//   1. Environment variables (AZURE_TENANT_ID + AZURE_CLIENT_ID + AZURE_CLIENT_SECRET)
//   2. Workload Identity (AKS)
//   3. Managed Identity
//   4. Azure CLI (local development)
//
// Secret naming: <key-lowercased-hyphens>
// Azure Key Vault names allow only lowercase letters, numbers and hyphens.
// e.g. JWT_PRIVATE_KEY → jwt-private-key

// azureSecretsClient abstracts the Azure Key Vault secrets API for testability.
type azureSecretsClient interface {
	GetSecret(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error)
}

// newAzureClient creates credential and client for Azure Key Vault. Replaceable in tests.
var newAzureClient = func(vaultURL string) (azureSecretsClient, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("Azure Key Vault: failed to create credential: %w", err)
	}
	client, err := azsecrets.NewClient(vaultURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("Azure Key Vault: failed to create client for %q: %w", vaultURL, err)
	}
	return client, nil
}

type azureKeyVaultManager struct {
	client azureSecretsClient
}

func newAzureKeyVaultManager(vaultURL string) (*azureKeyVaultManager, error) {
	client, err := newAzureClient(vaultURL)
	if err != nil {
		return nil, err
	}
	return &azureKeyVaultManager{client: client}, nil
}

func (a *azureKeyVaultManager) secretName(key string) string {
	return strings.ToLower(strings.ReplaceAll(key, "_", "-"))
}

func (a *azureKeyVaultManager) GetSecret(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), secretFetchTimeout)
	defer cancel()

	name := a.secretName(key)
	// Empty version string fetches the latest version.
	result, err := a.client.GetSecret(ctx, name, "", nil)
	if err != nil {
		return nil, fmt.Errorf("Azure Key Vault: failed to get secret %q: %w", name, err)
	}
	if result.Value == nil {
		return nil, fmt.Errorf("Azure Key Vault: secret %q has no value", name)
	}
	return []byte(*result.Value), nil
}

func (a *azureKeyVaultManager) GetSecretString(key string) (string, error) {
	data, err := a.GetSecret(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

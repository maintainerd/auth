package config

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	awsssm "github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/googleapis/gax-go/v2"
	vaultapi "github.com/hashicorp/vault/api"

	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// ─── Generic SecretManager mock ──────────────────────────────────────────

type mockSecretManager struct {
	getSecretFn func(key string) ([]byte, error)
}

func (m *mockSecretManager) GetSecret(key string) ([]byte, error) {
	return m.getSecretFn(key)
}

func (m *mockSecretManager) GetSecretString(key string) (string, error) {
	data, err := m.GetSecret(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ─── AWS Secrets Manager mock ────────────────────────────────────────────

type mockAWSSecretsClient struct {
	getSecretValueFn func(ctx context.Context, params *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error)
}

func (m *mockAWSSecretsClient) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	return m.getSecretValueFn(ctx, params)
}

// ─── AWS SSM mock ────────────────────────────────────────────────────────

type mockAWSSSMClient struct {
	getParameterFn func(ctx context.Context, params *awsssm.GetParameterInput) (*awsssm.GetParameterOutput, error)
}

func (m *mockAWSSSMClient) GetParameter(ctx context.Context, params *awsssm.GetParameterInput, _ ...func(*awsssm.Options)) (*awsssm.GetParameterOutput, error) {
	return m.getParameterFn(ctx, params)
}

// ─── Vault KV v2 mock ───────────────────────────────────────────────────

type mockVaultKVReader struct {
	getFn func(ctx context.Context, secretPath string) (*vaultapi.KVSecret, error)
}

func (m *mockVaultKVReader) Get(ctx context.Context, secretPath string) (*vaultapi.KVSecret, error) {
	return m.getFn(ctx, secretPath)
}

// ─── GCP Secret Manager mock ────────────────────────────────────────────

type mockGCPSMClient struct {
	accessSecretVersionFn func(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest) (*secretmanagerpb.AccessSecretVersionResponse, error)
}

func (m *mockGCPSMClient) AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, _ ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	return m.accessSecretVersionFn(ctx, req)
}

// ─── Azure Key Vault mock ───────────────────────────────────────────────

type mockAzureSecretsClient struct {
	getSecretFn func(ctx context.Context, name string, version string) (azsecrets.GetSecretResponse, error)
}

func (m *mockAzureSecretsClient) GetSecret(ctx context.Context, name string, version string, _ *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
	return m.getSecretFn(ctx, name, version)
}

// ─── Helpers ─────────────────────────────────────────────────────────────

func saveActiveSecretManager(t *testing.T) {
	t.Helper()
	orig := activeSecretManager
	t.Cleanup(func() { activeSecretManager = orig })
}

func stringPtr(s string) *string { return &s }

package config

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// SecretManager is the interface all secret providers must implement.
// Providers are selected at startup via the SECRET_PROVIDER environment variable.
type SecretManager interface {
	GetSecret(key string) ([]byte, error)
	GetSecretString(key string) (string, error)
}

// activeSecretManager is the resolved provider, initialized once by initSecretManager.
var activeSecretManager SecretManager

// secretFetchTimeout caps each external provider call.
const secretFetchTimeout = 10 * time.Second

// ────────────────────────────────────────────────── env provider ──────────

type envSecretManager struct{}

func (e *envSecretManager) GetSecret(key string) ([]byte, error) {
	value := os.Getenv(key)
	if value == "" {
		return nil, fmt.Errorf("environment variable %q is not set", key)
	}
	if strings.HasPrefix(value, "base64:") {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, "base64:"))
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 secret %q: %w", key, err)
		}
		return decoded, nil
	}
	return []byte(value), nil
}

func (e *envSecretManager) GetSecretString(key string) (string, error) {
	data, err := e.GetSecret(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ────────────────────────────────────────────── file provider ──────────

// fileSecretManager reads secrets from files — useful for Docker secrets
// mounted under /run/secrets or a custom directory (SECRET_FILE_PATH).
// Key names are lowercased and underscores replaced with hyphens.
// e.g. JWT_PRIVATE_KEY → <base-path>/jwt-private-key
type fileSecretManager struct{ basePath string }

func (f *fileSecretManager) GetSecret(key string) ([]byte, error) {
	name := strings.ToLower(strings.ReplaceAll(key, "_", "-"))
	path := fmt.Sprintf("%s/%s", f.basePath, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret file %q: %w", path, err)
	}
	return data, nil
}

func (f *fileSecretManager) GetSecretString(key string) (string, error) {
	data, err := f.GetSecret(key)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// ────────────────────────────────────────── factory & lifecycle ──────────

// initSecretManager creates the active provider from the current configuration
// and stores it in activeSecretManager. Must be called once from config.Init()
// after SecretProvider and SecretPrefix are set.
func initSecretManager() error {
	sm, err := newSecretManager()
	if err != nil {
		return err
	}
	activeSecretManager = sm
	return nil
}

// newSecretManager constructs the SecretManager for the configured SECRET_PROVIDER.
//
// Supported values (SECRET_PROVIDER):
//
//	env        – environment variables (default)
//	file       – files under SECRET_FILE_PATH (default /run/secrets)
//	aws_secrets – AWS Secrets Manager
//	aws_ssm    – AWS SSM Parameter Store
//	vault      – HashiCorp Vault (KV v2)
//	gcp        – GCP Secret Manager
//	azure_kv   – Azure Key Vault
func newSecretManager() (SecretManager, error) {
	switch SecretProvider {
	case "env":
		slog.Info("Secret provider: environment variables")
		return &envSecretManager{}, nil

	case "file":
		basePath := GetEnvOrDefault("SECRET_FILE_PATH", "/run/secrets")
		slog.Info("Secret provider: file", "path", basePath)
		return &fileSecretManager{basePath: basePath}, nil

	case "aws_secrets":
		region := GetEnvOrDefault("AWS_REGION", "us-east-1")
		slog.Info("Secret provider: AWS Secrets Manager", "region", region, "prefix", SecretPrefix)
		return newAWSSecretsManager(region, SecretPrefix)

	case "aws_ssm":
		region := GetEnvOrDefault("AWS_REGION", "us-east-1")
		slog.Info("Secret provider: AWS SSM Parameter Store", "region", region, "prefix", SecretPrefix)
		return newAWSSSMSecretManager(region, SecretPrefix)

	case "vault":
		address := GetEnvOrDefault("VAULT_ADDR", "http://localhost:8200")
		token := os.Getenv("VAULT_TOKEN")
		mount := GetEnvOrDefault("VAULT_MOUNT", "secret")
		slog.Info("Secret provider: HashiCorp Vault", "address", address, "mount", mount, "prefix", SecretPrefix)
		return newVaultSecretManager(address, token, SecretPrefix, mount)

	case "gcp":
		projectID, err := GetEnv("GCP_PROJECT_ID")
		if err != nil {
			return nil, fmt.Errorf("GCP Secret Manager requires GCP_PROJECT_ID: %w", err)
		}
		slog.Info("Secret provider: GCP Secret Manager", "project", projectID)
		return newGCPSecretManager(projectID)

	case "azure_kv":
		vaultURL, err := GetEnv("AZURE_KEYVAULT_URL")
		if err != nil {
			return nil, fmt.Errorf("Azure Key Vault requires AZURE_KEYVAULT_URL: %w", err)
		}
		slog.Info("Secret provider: Azure Key Vault", "url", vaultURL)
		return newAzureKeyVaultManager(vaultURL)

	default:
		slog.Warn("Unknown SECRET_PROVIDER, falling back to environment variables", "provider", SecretProvider)
		return &envSecretManager{}, nil
	}
}

// loadSecret fetches a secret through the active provider with up to 3 retries.
func loadSecret(key string) ([]byte, error) {
	if activeSecretManager == nil {
		return nil, fmt.Errorf("secret manager not initialized; ensure initSecretManager is called first")
	}

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		secret, err := activeSecretManager.GetSecret(key)
		if err == nil {
			if len(secret) == 0 {
				return nil, fmt.Errorf("secret %q is empty", key)
			}
			slog.Info("Loaded secret", "key", key, "bytes", len(secret))
			return secret, nil
		}
		lastErr = err
		if attempt < 3 {
			slog.Warn("Failed to load secret, retrying", "key", key, "attempt", attempt, "error", err)
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	return nil, fmt.Errorf("failed to load secret %q after 3 attempts: %w", key, lastErr)
}

// ValidateSecretProvider returns an error if SECRET_PROVIDER is not a known value.
func ValidateSecretProvider() error {
	valid := []string{"env", "file", "aws_secrets", "aws_ssm", "vault", "gcp", "azure_kv"}
	for _, p := range valid {
		if SecretProvider == p {
			return nil
		}
	}
	return fmt.Errorf("invalid SECRET_PROVIDER %q, must be one of: %v", SecretProvider, valid)
}

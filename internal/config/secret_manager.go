package config

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// SecretManager interface for different secret providers
type SecretManager interface {
	GetSecret(key string) ([]byte, error)
	GetSecretString(key string) (string, error)
}

// EnvironmentSecretManager loads secrets from environment variables
type EnvironmentSecretManager struct{}

func (e *EnvironmentSecretManager) GetSecret(key string) ([]byte, error) {
	value := os.Getenv(key)
	if value == "" {
		return nil, fmt.Errorf("environment variable %s is not set", key)
	}

	// Handle base64 encoded secrets (useful for binary data)
	if strings.HasPrefix(value, "base64:") {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, "base64:"))
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 secret %s: %w", key, err)
		}
		return decoded, nil
	}

	return []byte(value), nil
}

func (e *EnvironmentSecretManager) GetSecretString(key string) (string, error) {
	data, err := e.GetSecret(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FileSecretManager loads secrets from files (useful for Docker secrets)
type FileSecretManager struct {
	BasePath string
}

func (f *FileSecretManager) GetSecret(key string) ([]byte, error) {
	// Convert environment variable name to file path
	// JWT_PRIVATE_KEY -> /run/secrets/jwt_private_key
	filename := strings.ToLower(strings.ReplaceAll(key, "_", "_"))
	filepath := fmt.Sprintf("%s/%s", f.BasePath, filename)

	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret file %s: %w", filepath, err)
	}

	return data, nil
}

func (f *FileSecretManager) GetSecretString(key string) (string, error) {
	data, err := f.GetSecret(key)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// AWS Systems Manager Parameter Store (placeholder for future implementation)
type AWSSSMSecretManager struct {
	Region string
	Prefix string
}

func (a *AWSSSMSecretManager) GetSecret(key string) ([]byte, error) {
	// TODO: Implement AWS SSM Parameter Store integration
	// This would use AWS SDK to fetch parameters
	return nil, fmt.Errorf("AWS SSM integration not implemented yet")
}

func (a *AWSSSMSecretManager) GetSecretString(key string) (string, error) {
	data, err := a.GetSecret(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// AWS Secrets Manager (placeholder for future implementation)
type AWSSecretsManager struct {
	Region string
	Prefix string
}

func (a *AWSSecretsManager) GetSecret(key string) ([]byte, error) {
	// TODO: Implement AWS Secrets Manager integration
	return nil, fmt.Errorf("AWS Secrets Manager integration not implemented yet")
}

func (a *AWSSecretsManager) GetSecretString(key string) (string, error) {
	data, err := a.GetSecret(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// HashiCorp Vault (placeholder for future implementation)
type VaultSecretManager struct {
	Address string
	Token   string
	Prefix  string
}

func (v *VaultSecretManager) GetSecret(key string) ([]byte, error) {
	// TODO: Implement HashiCorp Vault integration
	return nil, fmt.Errorf("HashiCorp Vault integration not implemented yet")
}

func (v *VaultSecretManager) GetSecretString(key string) (string, error) {
	data, err := v.GetSecret(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// getSecretManager returns the appropriate secret manager based on configuration
func getSecretManager() SecretManager {
	switch SecretProvider {
	case "env":
		log.Println("🔐 Using environment variable secret provider")
		return &EnvironmentSecretManager{}

	case "file":
		basePath := GetEnvOrDefault("SECRET_FILE_PATH", "/run/secrets")
		log.Printf("🔐 Using file secret provider (path: %s)", basePath)
		return &FileSecretManager{BasePath: basePath}

	case "aws_ssm":
		region := GetEnvOrDefault("AWS_REGION", "us-east-1")
		log.Printf("🔐 Using AWS SSM Parameter Store (region: %s, prefix: %s)", region, SecretPrefix)
		return &AWSSSMSecretManager{Region: region, Prefix: SecretPrefix}

	case "aws_secrets":
		region := GetEnvOrDefault("AWS_REGION", "us-east-1")
		log.Printf("🔐 Using AWS Secrets Manager (region: %s, prefix: %s)", region, SecretPrefix)
		return &AWSSecretsManager{Region: region, Prefix: SecretPrefix}

	case "vault":
		address := GetEnvOrDefault("VAULT_ADDR", "http://localhost:8200")
		log.Printf("🔐 Using HashiCorp Vault (address: %s, prefix: %s)", address, SecretPrefix)
		return &VaultSecretManager{
			Address: address,
			Token:   os.Getenv("VAULT_TOKEN"),
			Prefix:  SecretPrefix,
		}

	default:
		log.Printf("⚠️ Unknown secret provider '%s', falling back to environment variables", SecretProvider)
		return &EnvironmentSecretManager{}
	}
}

// loadSecret loads a secret using the configured secret manager
func loadSecret(key string) ([]byte, error) {
	manager := getSecretManager()

	// Add retry logic for production resilience
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		secret, err := manager.GetSecret(key)
		if err == nil {
			// Validate secret is not empty
			if len(secret) == 0 {
				return nil, fmt.Errorf("secret %s is empty", key)
			}

			log.Printf("✅ Successfully loaded secret: %s (length: %d bytes)", key, len(secret))
			return secret, nil
		}

		lastErr = err
		if attempt < 3 {
			log.Printf("⚠️ Failed to load secret %s (attempt %d/3): %v", key, attempt, err)
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	return nil, fmt.Errorf("failed to load secret %s after 3 attempts: %w", key, lastErr)
}

// loadSecretString is a convenience function for string secrets
func loadSecretString(key string) (string, error) {
	data, err := loadSecret(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ValidateSecretProvider validates the secret provider configuration
func ValidateSecretProvider() error {
	validProviders := []string{"env", "file", "aws_ssm", "aws_secrets", "vault"}

	for _, provider := range validProviders {
		if SecretProvider == provider {
			return nil
		}
	}

	return fmt.Errorf("invalid secret provider '%s', must be one of: %v", SecretProvider, validProviders)
}

package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	awsssm "github.com/aws/aws-sdk-go-v2/service/ssm"
)

// awsLoadDefaultConfig is the AWS config loader, replaceable in tests.
var awsLoadDefaultConfig = awsconfig.LoadDefaultConfig

// ─────────────────────────────────── AWS Secrets Manager provider ──────────
//
// Configuration env vars:
//   AWS_REGION             – AWS region (default: us-east-1)
//   SECRET_PREFIX          – Secret name prefix (default: maintainerd/auth)
//   AWS_ACCESS_KEY_ID      – Access key (optional; uses IAM role when omitted)
//   AWS_SECRET_ACCESS_KEY  – Secret key (optional; uses IAM role when omitted)
//
// Secret naming: <SECRET_PREFIX>/<key-lowercased-hyphens>
// e.g. JWT_PRIVATE_KEY → maintainerd/auth/jwt-private-key

// awsSecretsClient abstracts the AWS Secrets Manager API for testability.
type awsSecretsClient interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

type awsSecretsManager struct {
	prefix string
	client awsSecretsClient
}

func newAWSSecretsManager(region, prefix string) (*awsSecretsManager, error) {
	cfg, err := awsLoadDefaultConfig(context.Background(), awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("AWS Secrets Manager: failed to load AWS config: %w", err)
	}
	return &awsSecretsManager{
		prefix: prefix,
		client: secretsmanager.NewFromConfig(cfg),
	}, nil
}

func (a *awsSecretsManager) secretID(key string) string {
	name := strings.ToLower(strings.ReplaceAll(key, "_", "-"))
	if a.prefix != "" {
		return fmt.Sprintf("%s/%s", a.prefix, name)
	}
	return name
}

func (a *awsSecretsManager) GetSecret(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), secretFetchTimeout)
	defer cancel()

	id := a.secretID(key)
	out, err := a.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(id),
	})
	if err != nil {
		return nil, fmt.Errorf("AWS Secrets Manager: failed to get %q: %w", id, err)
	}
	if out.SecretString != nil {
		return []byte(*out.SecretString), nil
	}
	if out.SecretBinary != nil {
		return out.SecretBinary, nil
	}
	return nil, fmt.Errorf("AWS Secrets Manager: secret %q has no value", id)
}

func (a *awsSecretsManager) GetSecretString(key string) (string, error) {
	data, err := a.GetSecret(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ─────────────────────────────── AWS SSM Parameter Store provider ──────────
//
// Configuration env vars:
//   AWS_REGION             – AWS region (default: us-east-1)
//   SECRET_PREFIX          – Parameter path prefix (default: maintainerd/auth)
//   AWS_ACCESS_KEY_ID      – Access key (optional; uses IAM role when omitted)
//   AWS_SECRET_ACCESS_KEY  – Secret key (optional; uses IAM role when omitted)
//
// Parameter naming: /<SECRET_PREFIX>/<key-lowercased-hyphens>
// e.g. JWT_PRIVATE_KEY → /maintainerd/auth/jwt-private-key
// SecureString parameters are automatically decrypted via the default KMS key.

// awsSSMClient abstracts the AWS SSM Parameter Store API for testability.
type awsSSMClient interface {
	GetParameter(ctx context.Context, params *awsssm.GetParameterInput, optFns ...func(*awsssm.Options)) (*awsssm.GetParameterOutput, error)
}

type awsSSMSecretManager struct {
	prefix string
	client awsSSMClient
}

func newAWSSSMSecretManager(region, prefix string) (*awsSSMSecretManager, error) {
	cfg, err := awsLoadDefaultConfig(context.Background(), awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("AWS SSM: failed to load AWS config: %w", err)
	}
	return &awsSSMSecretManager{
		prefix: prefix,
		client: awsssm.NewFromConfig(cfg),
	}, nil
}

func (a *awsSSMSecretManager) paramPath(key string) string {
	name := strings.ToLower(strings.ReplaceAll(key, "_", "-"))
	prefix := strings.Trim(a.prefix, "/")
	if prefix != "" {
		return fmt.Sprintf("/%s/%s", prefix, name)
	}
	return "/" + name
}

func (a *awsSSMSecretManager) GetSecret(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), secretFetchTimeout)
	defer cancel()

	path := a.paramPath(key)
	out, err := a.client.GetParameter(ctx, &awsssm.GetParameterInput{
		Name:           aws.String(path),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("AWS SSM: failed to get parameter %q: %w", path, err)
	}
	if out.Parameter == nil {
		return nil, fmt.Errorf("AWS SSM: parameter %q not found", path)
	}
	if out.Parameter.Value == nil {
		return nil, fmt.Errorf("AWS SSM: parameter %q has no value", path)
	}
	return []byte(*out.Parameter.Value), nil
}

func (a *awsSSMSecretManager) GetSecretString(key string) (string, error) {
	data, err := a.GetSecret(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

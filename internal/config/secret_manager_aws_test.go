package config

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	awsssm "github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── awsSecretsManager constructors ──────────────────────────────────────

func TestNewAWSSecretsManager(t *testing.T) {
	t.Run("config load error", func(t *testing.T) {
		orig := awsLoadDefaultConfig
		t.Cleanup(func() { awsLoadDefaultConfig = orig })
		awsLoadDefaultConfig = func(_ context.Context, _ ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
			return aws.Config{}, fmt.Errorf("mock config error")
		}

		_, err := newAWSSecretsManager("us-east-1", "test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load AWS config")
	})
}

func TestNewAWSSSMSecretManager(t *testing.T) {
	t.Run("config load error", func(t *testing.T) {
		orig := awsLoadDefaultConfig
		t.Cleanup(func() { awsLoadDefaultConfig = orig })
		awsLoadDefaultConfig = func(_ context.Context, _ ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
			return aws.Config{}, fmt.Errorf("mock config error")
		}

		_, err := newAWSSSMSecretManager("us-east-1", "test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load AWS config")
	})
}

// ─── awsSecretsManager ──────────────────────────────────────────────────

func TestAWSSecretsManager_SecretID(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		key    string
		want   string
	}{
		{"with prefix", "maintainerd/auth", "JWT_PRIVATE_KEY", "maintainerd/auth/jwt-private-key"},
		{"no prefix", "", "DB_PASSWORD", "db-password"},
		{"complex key", "app", "SMTP_FROM_EMAIL", "app/smtp-from-email"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sm := &awsSecretsManager{prefix: tc.prefix}
			assert.Equal(t, tc.want, sm.secretID(tc.key))
		})
	}
}

func TestAWSSecretsManager_GetSecret(t *testing.T) {
	t.Run("returns secret string", func(t *testing.T) {
		sm := &awsSecretsManager{
			prefix: "test",
			client: &mockAWSSecretsClient{
				getSecretValueFn: func(_ context.Context, params *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
					assert.Equal(t, "test/my-key", aws.ToString(params.SecretId))
					return &secretsmanager.GetSecretValueOutput{
						SecretString: stringPtr("secret-value"),
					}, nil
				},
			},
		}

		data, err := sm.GetSecret("MY_KEY")
		require.NoError(t, err)
		assert.Equal(t, []byte("secret-value"), data)
	})

	t.Run("returns secret binary", func(t *testing.T) {
		sm := &awsSecretsManager{
			prefix: "test",
			client: &mockAWSSecretsClient{
				getSecretValueFn: func(_ context.Context, _ *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
					return &secretsmanager.GetSecretValueOutput{
						SecretBinary: []byte("binary-data"),
					}, nil
				},
			},
		}

		data, err := sm.GetSecret("BIN_KEY")
		require.NoError(t, err)
		assert.Equal(t, []byte("binary-data"), data)
	})

	t.Run("error when no value", func(t *testing.T) {
		sm := &awsSecretsManager{
			prefix: "test",
			client: &mockAWSSecretsClient{
				getSecretValueFn: func(_ context.Context, _ *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
					return &secretsmanager.GetSecretValueOutput{}, nil
				},
			},
		}

		_, err := sm.GetSecret("EMPTY_KEY")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "has no value")
	})

	t.Run("api error", func(t *testing.T) {
		sm := &awsSecretsManager{
			prefix: "test",
			client: &mockAWSSecretsClient{
				getSecretValueFn: func(_ context.Context, _ *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
					return nil, fmt.Errorf("access denied")
				},
			},
		}

		_, err := sm.GetSecret("MY_KEY")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get")
	})
}

func TestAWSSecretsManager_GetSecretString(t *testing.T) {
	t.Run("returns string", func(t *testing.T) {
		sm := &awsSecretsManager{
			prefix: "",
			client: &mockAWSSecretsClient{
				getSecretValueFn: func(_ context.Context, _ *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
					return &secretsmanager.GetSecretValueOutput{SecretString: stringPtr("val")}, nil
				},
			},
		}

		val, err := sm.GetSecretString("K")
		require.NoError(t, err)
		assert.Equal(t, "val", val)
	})

	t.Run("propagates error", func(t *testing.T) {
		sm := &awsSecretsManager{
			prefix: "",
			client: &mockAWSSecretsClient{
				getSecretValueFn: func(_ context.Context, _ *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
					return nil, fmt.Errorf("boom")
				},
			},
		}

		_, err := sm.GetSecretString("K")
		require.Error(t, err)
	})
}

// ─── awsSSMSecretManager ────────────────────────────────────────────────

func TestAWSSSMSecretManager_ParamPath(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		key    string
		want   string
	}{
		{"with prefix", "maintainerd/auth", "JWT_PRIVATE_KEY", "/maintainerd/auth/jwt-private-key"},
		{"no prefix", "", "DB_PASSWORD", "/db-password"},
		{"prefix with slashes", "/app/config/", "MY_SECRET", "/app/config/my-secret"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sm := &awsSSMSecretManager{prefix: tc.prefix}
			assert.Equal(t, tc.want, sm.paramPath(tc.key))
		})
	}
}

func TestAWSSSMSecretManager_GetSecret(t *testing.T) {
	t.Run("returns parameter value", func(t *testing.T) {
		sm := &awsSSMSecretManager{
			prefix: "test",
			client: &mockAWSSSMClient{
				getParameterFn: func(_ context.Context, params *awsssm.GetParameterInput) (*awsssm.GetParameterOutput, error) {
					assert.Equal(t, "/test/my-param", aws.ToString(params.Name))
					assert.True(t, aws.ToBool(params.WithDecryption))
					return &awsssm.GetParameterOutput{
						Parameter: &ssmtypes.Parameter{
							Value: stringPtr("param-value"),
						},
					}, nil
				},
			},
		}

		data, err := sm.GetSecret("MY_PARAM")
		require.NoError(t, err)
		assert.Equal(t, []byte("param-value"), data)
	})

	t.Run("nil parameter", func(t *testing.T) {
		sm := &awsSSMSecretManager{
			prefix: "test",
			client: &mockAWSSSMClient{
				getParameterFn: func(_ context.Context, _ *awsssm.GetParameterInput) (*awsssm.GetParameterOutput, error) {
					return &awsssm.GetParameterOutput{Parameter: nil}, nil
				},
			},
		}

		_, err := sm.GetSecret("NIL_PARAM")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("nil value", func(t *testing.T) {
		sm := &awsSSMSecretManager{
			prefix: "test",
			client: &mockAWSSSMClient{
				getParameterFn: func(_ context.Context, _ *awsssm.GetParameterInput) (*awsssm.GetParameterOutput, error) {
					return &awsssm.GetParameterOutput{
						Parameter: &ssmtypes.Parameter{Value: nil},
					}, nil
				},
			},
		}

		_, err := sm.GetSecret("NO_VAL")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "has no value")
	})

	t.Run("api error", func(t *testing.T) {
		sm := &awsSSMSecretManager{
			prefix: "test",
			client: &mockAWSSSMClient{
				getParameterFn: func(_ context.Context, _ *awsssm.GetParameterInput) (*awsssm.GetParameterOutput, error) {
					return nil, fmt.Errorf("parameter not found")
				},
			},
		}

		_, err := sm.GetSecret("MISSING")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get parameter")
	})
}

func TestAWSSSMSecretManager_GetSecretString(t *testing.T) {
	t.Run("returns string", func(t *testing.T) {
		sm := &awsSSMSecretManager{
			prefix: "",
			client: &mockAWSSSMClient{
				getParameterFn: func(_ context.Context, _ *awsssm.GetParameterInput) (*awsssm.GetParameterOutput, error) {
					return &awsssm.GetParameterOutput{
						Parameter: &ssmtypes.Parameter{Value: stringPtr("val")},
					}, nil
				},
			},
		}

		val, err := sm.GetSecretString("K")
		require.NoError(t, err)
		assert.Equal(t, "val", val)
	})

	t.Run("propagates error", func(t *testing.T) {
		sm := &awsSSMSecretManager{
			prefix: "",
			client: &mockAWSSSMClient{
				getParameterFn: func(_ context.Context, _ *awsssm.GetParameterInput) (*awsssm.GetParameterOutput, error) {
					return nil, fmt.Errorf("boom")
				},
			},
		}

		_, err := sm.GetSecretString("K")
		require.Error(t, err)
	})
}

package config

import (
	"context"
	"fmt"
	"testing"

	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── newGCPSecretManager ────────────────────────────────────────────────

func TestNewGCPSecretManager(t *testing.T) {
	t.Run("success with mock client", func(t *testing.T) {
		orig := newGCPClient
		t.Cleanup(func() { newGCPClient = orig })
		newGCPClient = func(_ context.Context) (gcpSMClient, error) {
			return &mockGCPSMClient{}, nil
		}

		sm, err := newGCPSecretManager("test-project")
		require.NoError(t, err)
		assert.Equal(t, "test-project", sm.projectID)
	})

	t.Run("client creation error", func(t *testing.T) {
		orig := newGCPClient
		t.Cleanup(func() { newGCPClient = orig })
		newGCPClient = func(_ context.Context) (gcpSMClient, error) {
			return nil, fmt.Errorf("no credentials")
		}

		_, err := newGCPSecretManager("test-project")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create client")
	})
}

// ─── secretVersion ──────────────────────────────────────────────────────

func TestGCPSecretManager_SecretVersion(t *testing.T) {
	tests := []struct {
		name      string
		projectID string
		key       string
		want      string
	}{
		{"standard key", "my-project", "JWT_PRIVATE_KEY", "projects/my-project/secrets/jwt-private-key/versions/latest"},
		{"simple key", "proj", "DB_PASSWORD", "projects/proj/secrets/db-password/versions/latest"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sm := &gcpSecretManager{projectID: tc.projectID}
			assert.Equal(t, tc.want, sm.secretVersion(tc.key))
		})
	}
}

// ─── GetSecret ──────────────────────────────────────────────────────────

func TestGCPSecretManager_GetSecret(t *testing.T) {
	t.Run("returns payload data", func(t *testing.T) {
		sm := &gcpSecretManager{
			projectID: "proj",
			client: &mockGCPSMClient{
				accessSecretVersionFn: func(_ context.Context, req *secretmanagerpb.AccessSecretVersionRequest) (*secretmanagerpb.AccessSecretVersionResponse, error) {
					assert.Equal(t, "projects/proj/secrets/my-key/versions/latest", req.Name)
					return &secretmanagerpb.AccessSecretVersionResponse{
						Payload: &secretmanagerpb.SecretPayload{
							Data: []byte("secret-data"),
						},
					}, nil
				},
			},
		}

		data, err := sm.GetSecret("MY_KEY")
		require.NoError(t, err)
		assert.Equal(t, []byte("secret-data"), data)
	})

	t.Run("api error", func(t *testing.T) {
		sm := &gcpSecretManager{
			projectID: "proj",
			client: &mockGCPSMClient{
				accessSecretVersionFn: func(_ context.Context, _ *secretmanagerpb.AccessSecretVersionRequest) (*secretmanagerpb.AccessSecretVersionResponse, error) {
					return nil, fmt.Errorf("not found")
				},
			},
		}

		_, err := sm.GetSecret("K")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to access")
	})

	t.Run("nil payload", func(t *testing.T) {
		sm := &gcpSecretManager{
			projectID: "proj",
			client: &mockGCPSMClient{
				accessSecretVersionFn: func(_ context.Context, _ *secretmanagerpb.AccessSecretVersionRequest) (*secretmanagerpb.AccessSecretVersionResponse, error) {
					return &secretmanagerpb.AccessSecretVersionResponse{Payload: nil}, nil
				},
			},
		}

		_, err := sm.GetSecret("K")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil payload")
	})

	t.Run("nil payload data", func(t *testing.T) {
		sm := &gcpSecretManager{
			projectID: "proj",
			client: &mockGCPSMClient{
				accessSecretVersionFn: func(_ context.Context, _ *secretmanagerpb.AccessSecretVersionRequest) (*secretmanagerpb.AccessSecretVersionResponse, error) {
					return &secretmanagerpb.AccessSecretVersionResponse{
						Payload: &secretmanagerpb.SecretPayload{Data: nil},
					}, nil
				},
			},
		}

		_, err := sm.GetSecret("K")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil payload")
	})
}

func TestGCPSecretManager_GetSecretString(t *testing.T) {
	t.Run("returns string", func(t *testing.T) {
		sm := &gcpSecretManager{
			projectID: "proj",
			client: &mockGCPSMClient{
				accessSecretVersionFn: func(_ context.Context, _ *secretmanagerpb.AccessSecretVersionRequest) (*secretmanagerpb.AccessSecretVersionResponse, error) {
					return &secretmanagerpb.AccessSecretVersionResponse{
						Payload: &secretmanagerpb.SecretPayload{Data: []byte("val")},
					}, nil
				},
			},
		}

		val, err := sm.GetSecretString("K")
		require.NoError(t, err)
		assert.Equal(t, "val", val)
	})

	t.Run("propagates error", func(t *testing.T) {
		sm := &gcpSecretManager{
			projectID: "proj",
			client: &mockGCPSMClient{
				accessSecretVersionFn: func(_ context.Context, _ *secretmanagerpb.AccessSecretVersionRequest) (*secretmanagerpb.AccessSecretVersionResponse, error) {
					return nil, fmt.Errorf("fail")
				},
			},
		}

		_, err := sm.GetSecretString("K")
		require.Error(t, err)
	})
}

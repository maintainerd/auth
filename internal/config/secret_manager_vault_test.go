package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── secretPath ──────────────────────────────────────────────────────────

func TestVaultSecretManager_SecretPath(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		key    string
		want   string
	}{
		{"with prefix", "maintainerd/auth", "JWT_PRIVATE_KEY", "maintainerd/auth/jwt-private-key"},
		{"no prefix", "", "DB_PASSWORD", "db-password"},
		{"prefix with slashes", "/app/", "MY_KEY", "app/my-key"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sm := &vaultSecretManager{prefix: tc.prefix}
			assert.Equal(t, tc.want, sm.secretPath(tc.key))
		})
	}
}

// ─── GetSecret ───────────────────────────────────────────────────────────

func TestVaultSecretManager_GetSecret(t *testing.T) {
	t.Run("returns field value", func(t *testing.T) {
		sm := &vaultSecretManager{
			prefix: "app",
			mount:  "secret",
			field:  "value",
			kv: &mockVaultKVReader{
				getFn: func(_ context.Context, secretPath string) (*vaultapi.KVSecret, error) {
					assert.Equal(t, "app/my-key", secretPath)
					return &vaultapi.KVSecret{
						Data: map[string]interface{}{"value": "secret-data"},
					}, nil
				},
			},
		}

		data, err := sm.GetSecret("MY_KEY")
		require.NoError(t, err)
		assert.Equal(t, []byte("secret-data"), data)
	})

	t.Run("api error", func(t *testing.T) {
		sm := &vaultSecretManager{
			prefix: "",
			mount:  "secret",
			field:  "value",
			kv: &mockVaultKVReader{
				getFn: func(_ context.Context, _ string) (*vaultapi.KVSecret, error) {
					return nil, fmt.Errorf("permission denied")
				},
			},
		}

		_, err := sm.GetSecret("K")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read")
	})

	t.Run("nil secret", func(t *testing.T) {
		sm := &vaultSecretManager{
			prefix: "",
			mount:  "secret",
			field:  "value",
			kv: &mockVaultKVReader{
				getFn: func(_ context.Context, _ string) (*vaultapi.KVSecret, error) {
					return nil, nil
				},
			},
		}

		_, err := sm.GetSecret("K")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("nil data", func(t *testing.T) {
		sm := &vaultSecretManager{
			prefix: "",
			mount:  "secret",
			field:  "value",
			kv: &mockVaultKVReader{
				getFn: func(_ context.Context, _ string) (*vaultapi.KVSecret, error) {
					return &vaultapi.KVSecret{Data: nil}, nil
				},
			},
		}

		_, err := sm.GetSecret("K")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("missing field", func(t *testing.T) {
		sm := &vaultSecretManager{
			prefix: "",
			mount:  "secret",
			field:  "value",
			kv: &mockVaultKVReader{
				getFn: func(_ context.Context, _ string) (*vaultapi.KVSecret, error) {
					return &vaultapi.KVSecret{
						Data: map[string]interface{}{"other": "data"},
					}, nil
				},
			},
		}

		_, err := sm.GetSecret("K")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing field")
		assert.Contains(t, err.Error(), "other")
	})

	t.Run("non-string field value", func(t *testing.T) {
		sm := &vaultSecretManager{
			prefix: "",
			mount:  "secret",
			field:  "value",
			kv: &mockVaultKVReader{
				getFn: func(_ context.Context, _ string) (*vaultapi.KVSecret, error) {
					return &vaultapi.KVSecret{
						Data: map[string]interface{}{"value": 12345},
					}, nil
				},
			},
		}

		_, err := sm.GetSecret("K")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a string")
	})
}

func TestVaultSecretManager_GetSecretString(t *testing.T) {
	t.Run("returns string", func(t *testing.T) {
		sm := &vaultSecretManager{
			prefix: "",
			mount:  "secret",
			field:  "value",
			kv: &mockVaultKVReader{
				getFn: func(_ context.Context, _ string) (*vaultapi.KVSecret, error) {
					return &vaultapi.KVSecret{
						Data: map[string]interface{}{"value": "str"},
					}, nil
				},
			},
		}

		val, err := sm.GetSecretString("K")
		require.NoError(t, err)
		assert.Equal(t, "str", val)
	})

	t.Run("propagates error", func(t *testing.T) {
		sm := &vaultSecretManager{
			prefix: "",
			mount:  "secret",
			field:  "value",
			kv: &mockVaultKVReader{
				getFn: func(_ context.Context, _ string) (*vaultapi.KVSecret, error) {
					return nil, fmt.Errorf("fail")
				},
			},
		}

		_, err := sm.GetSecretString("K")
		require.Error(t, err)
	})
}

// ─── vaultAppRoleLogin ──────────────────────────────────────────────────

func TestVaultAppRoleLogin(t *testing.T) {
	t.Run("missing role ID", func(t *testing.T) {
		// Neither VAULT_ROLE_ID nor VAULT_SECRET_ID set
		cfg := vaultapi.DefaultConfig()
		client, err := vaultapi.NewClient(cfg)
		require.NoError(t, err)

		err = vaultAppRoleLogin(client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must both be set")
	})

	t.Run("missing secret ID", func(t *testing.T) {
		t.Setenv("VAULT_ROLE_ID", "role-123")
		// VAULT_SECRET_ID not set

		cfg := vaultapi.DefaultConfig()
		client, err := vaultapi.NewClient(cfg)
		require.NoError(t, err)

		err = vaultAppRoleLogin(client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must both be set")
	})

	t.Run("server returns auth token", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/auth/approle/login" {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"auth": map[string]interface{}{
						"client_token":   "test-token-abc",
						"lease_duration": 3600,
					},
				})
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		t.Setenv("VAULT_ROLE_ID", "role-123")
		t.Setenv("VAULT_SECRET_ID", "secret-456")

		cfg := vaultapi.DefaultConfig()
		cfg.Address = srv.URL
		client, err := vaultapi.NewClient(cfg)
		require.NoError(t, err)

		err = vaultAppRoleLogin(client)
		require.NoError(t, err)
		assert.Equal(t, "test-token-abc", client.Token())
	})

	t.Run("server returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errors": ["internal error"]}`))
		}))
		defer srv.Close()

		t.Setenv("VAULT_ROLE_ID", "role-123")
		t.Setenv("VAULT_SECRET_ID", "secret-456")

		cfg := vaultapi.DefaultConfig()
		cfg.Address = srv.URL
		client, err := vaultapi.NewClient(cfg)
		require.NoError(t, err)

		err = vaultAppRoleLogin(client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "AppRole login request failed")
	})

	t.Run("server returns nil auth", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			// No "auth" field in response
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{},
			})
		}))
		defer srv.Close()

		t.Setenv("VAULT_ROLE_ID", "role-123")
		t.Setenv("VAULT_SECRET_ID", "secret-456")

		cfg := vaultapi.DefaultConfig()
		cfg.Address = srv.URL
		client, err := vaultapi.NewClient(cfg)
		require.NoError(t, err)

		err = vaultAppRoleLogin(client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no auth token")
	})
}

// ─── newVaultSecretManager ──────────────────────────────────────────────

func TestNewVaultSecretManager(t *testing.T) {
	t.Run("with token", func(t *testing.T) {
		sm, err := newVaultSecretManager("http://localhost:8200", "my-token", "app", "secret")
		require.NoError(t, err)
		require.NotNil(t, sm)
		assert.Equal(t, "app", sm.prefix)
		assert.Equal(t, "secret", sm.mount)
		assert.Equal(t, "value", sm.field)
	})

	t.Run("empty mount defaults to secret", func(t *testing.T) {
		sm, err := newVaultSecretManager("http://localhost:8200", "my-token", "", "")
		require.NoError(t, err)
		assert.Equal(t, "secret", sm.mount)
	})

	t.Run("custom field", func(t *testing.T) {
		t.Setenv("VAULT_SECRET_FIELD", "data")
		sm, err := newVaultSecretManager("http://localhost:8200", "my-token", "", "secret")
		require.NoError(t, err)
		assert.Equal(t, "data", sm.field)
	})

	t.Run("approle login with httptest", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"auth": map[string]interface{}{
					"client_token":   "approle-token",
					"lease_duration": 3600,
				},
			})
		}))
		defer srv.Close()

		t.Setenv("VAULT_ROLE_ID", "role-id")
		t.Setenv("VAULT_SECRET_ID", "secret-id")

		sm, err := newVaultSecretManager(srv.URL, "", "app", "secret")
		require.NoError(t, err)
		require.NotNil(t, sm)
	})

	t.Run("approle login failure", func(t *testing.T) {
		_, err := newVaultSecretManager("http://localhost:8200", "", "", "secret")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "AppRole login failed")
	})

	t.Run("client creation error", func(t *testing.T) {
		orig := vaultNewClient
		t.Cleanup(func() { vaultNewClient = orig })
		vaultNewClient = func(_ *vaultapi.Config) (*vaultapi.Client, error) {
			return nil, fmt.Errorf("tls error")
		}

		_, err := newVaultSecretManager("http://localhost:8200", "token", "", "secret")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create client")
	})
}

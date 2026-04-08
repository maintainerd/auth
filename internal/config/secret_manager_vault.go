package config

import (
	"context"
	"fmt"
	"strings"

	vaultapi "github.com/hashicorp/vault/api"
)

// ─────────────────────────────────── HashiCorp Vault provider ──────────────
//
// Configuration env vars:
//   VAULT_ADDR    – Vault server address (default: http://localhost:8200)
//   VAULT_TOKEN   – Static token (optional; use AppRole instead in production)
//   VAULT_MOUNT   – KV v2 mount path (default: secret)
//   SECRET_PREFIX – Path prefix within the mount (default: maintainerd/auth)
//
// AppRole authentication (used when VAULT_TOKEN is empty):
//   VAULT_ROLE_ID   – AppRole role ID
//   VAULT_SECRET_ID – AppRole secret ID
//
// Secret path: <VAULT_MOUNT>/data/<SECRET_PREFIX>/<key-lowercased-hyphens>
// e.g. JWT_PRIVATE_KEY → secret/data/maintainerd/auth/jwt-private-key
//
// Each secret must have a "value" field (configurable via VAULT_SECRET_FIELD).
// Example Vault write:
//   vault kv put secret/maintainerd/auth/jwt-private-key value=@private.pem

// vaultKVReader abstracts the Vault KV v2 read API for testability.
type vaultKVReader interface {
	Get(ctx context.Context, secretPath string) (*vaultapi.KVSecret, error)
}

// vaultNewClient creates a Vault API client. Replaceable in tests.
var vaultNewClient = vaultapi.NewClient

type vaultSecretManager struct {
	prefix string
	mount  string
	field  string
	kv     vaultKVReader
}

func newVaultSecretManager(address, token, prefix, mount string) (*vaultSecretManager, error) {
	cfg := vaultapi.DefaultConfig()
	cfg.Address = address

	client, err := vaultNewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("Vault: failed to create client: %w", err)
	}

	if token != "" {
		client.SetToken(token)
	} else {
		// Fall back to AppRole authentication.
		if err := vaultAppRoleLogin(client); err != nil {
			return nil, fmt.Errorf("Vault: AppRole login failed (set VAULT_TOKEN or VAULT_ROLE_ID+VAULT_SECRET_ID): %w", err)
		}
	}

	if mount == "" {
		mount = "secret"
	}

	field := GetEnvOrDefault("VAULT_SECRET_FIELD", "value")

	return &vaultSecretManager{prefix: prefix, mount: mount, field: field, kv: client.KVv2(mount)}, nil
}

// vaultAppRoleLogin authenticates with Vault using AppRole credentials.
func vaultAppRoleLogin(client *vaultapi.Client) error {
	roleID := GetEnvOrDefault("VAULT_ROLE_ID", "")
	secretID := GetEnvOrDefault("VAULT_SECRET_ID", "")
	if roleID == "" || secretID == "" {
		return fmt.Errorf("VAULT_ROLE_ID and VAULT_SECRET_ID must both be set for AppRole auth")
	}

	secret, err := client.Logical().Write("auth/approle/login", map[string]interface{}{
		"role_id":   roleID,
		"secret_id": secretID,
	})
	if err != nil {
		return fmt.Errorf("AppRole login request failed: %w", err)
	}
	if secret == nil || secret.Auth == nil {
		return fmt.Errorf("AppRole login returned no auth token")
	}
	client.SetToken(secret.Auth.ClientToken)
	return nil
}

func (v *vaultSecretManager) secretPath(key string) string {
	name := strings.ToLower(strings.ReplaceAll(key, "_", "-"))
	prefix := strings.Trim(v.prefix, "/")
	if prefix != "" {
		return fmt.Sprintf("%s/%s", prefix, name)
	}
	return name
}

func (v *vaultSecretManager) GetSecret(key string) ([]byte, error) {
	path := v.secretPath(key)
	ctx, cancel := context.WithTimeout(context.Background(), secretFetchTimeout)
	defer cancel()
	secret, err := v.kv.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("Vault: failed to read %q at mount %q: %w", path, v.mount, err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("Vault: secret %q not found", path)
	}

	val, ok := secret.Data[v.field]
	if !ok {
		keys := make([]string, 0, len(secret.Data))
		for k := range secret.Data {
			keys = append(keys, k)
		}
		return nil, fmt.Errorf("Vault: secret %q missing field %q (found: %v)", path, v.field, keys)
	}

	str, ok := val.(string)
	if !ok {
		return nil, fmt.Errorf("Vault: field %q in secret %q is not a string (got %T)", v.field, path, val)
	}
	return []byte(str), nil
}

func (v *vaultSecretManager) GetSecretString(key string) (string, error) {
	data, err := v.GetSecret(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

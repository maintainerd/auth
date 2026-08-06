package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

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

	// client and the AppRole credentials are retained so an expired token can be
	// replaced without restarting the process. Vault tokens ALWAYS have a TTL —
	// AppRole-issued ones are often minutes — and the token used to be set once
	// at startup and never renewed. Once it lapsed every read returned 403, so
	// the periodic secret-refresh runner silently stopped picking up rotations
	// and the only symptom was a warning line every few minutes. reauth is nil
	// for a static VAULT_TOKEN, where there is nothing to re-authenticate with.
	client *vaultapi.Client
	reauth func(*vaultapi.Client) error
	mu     sync.Mutex
}

func newVaultSecretManager(address, token, prefix, mount string) (*vaultSecretManager, error) {
	cfg := vaultapi.DefaultConfig()
	cfg.Address = address

	client, err := vaultNewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("vault: failed to create client: %w", err)
	}

	var reauth func(*vaultapi.Client) error
	if token != "" {
		client.SetToken(token)
	} else {
		// Fall back to AppRole authentication, and remember how so the token can
		// be renewed later.
		if err := vaultAppRoleLogin(client); err != nil {
			return nil, fmt.Errorf("vault: AppRole login failed (set VAULT_TOKEN or VAULT_ROLE_ID+VAULT_SECRET_ID): %w", err)
		}
		reauth = vaultAppRoleLogin
	}

	if mount == "" {
		mount = "secret"
	}

	field := GetEnvOrDefault("VAULT_SECRET_FIELD", "value")

	return &vaultSecretManager{
		prefix: prefix,
		mount:  mount,
		field:  field,
		kv:     client.KVv2(mount),
		client: client,
		reauth: reauth,
	}, nil
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

	secret, err := v.read(path)
	if err != nil && v.isAuthFailure(err) {
		// Self-heal an expired or revoked token: re-authenticate once and retry.
		// Recovering here rather than on a timer means the fix applies to
		// revocation too, and there is no background goroutine to supervise.
		if rerr := v.relogin(); rerr != nil {
			return nil, fmt.Errorf("vault: read %q failed and re-authentication failed: %w (original: %v)", path, rerr, err)
		}
		secret, err = v.read(path)
	}
	if err != nil {
		var respErr *vaultapi.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("vault: secret %q: %w", path, ErrSecretNotFound)
		}
		return nil, fmt.Errorf("vault: failed to read %q at mount %q: %w", path, v.mount, err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("vault: secret %q: %w", path, ErrSecretNotFound)
	}

	val, ok := secret.Data[v.field]
	if !ok {
		keys := make([]string, 0, len(secret.Data))
		for k := range secret.Data {
			keys = append(keys, k)
		}
		return nil, fmt.Errorf("vault: secret %q missing field %q (found: %v)", path, v.field, keys)
	}

	str, ok := val.(string)
	if !ok {
		return nil, fmt.Errorf("vault: field %q in secret %q is not a string (got %T)", v.field, path, val)
	}
	return []byte(str), nil
}

func (v *vaultSecretManager) read(path string) (*vaultapi.KVSecret, error) {
	ctx, cancel := context.WithTimeout(context.Background(), secretFetchTimeout)
	defer cancel()
	return v.kv.Get(ctx, path)
}

// isAuthFailure reports whether an error is Vault rejecting our token, as
// opposed to the secret being absent or the server being unreachable. Only the
// former is worth re-authenticating for.
func (v *vaultSecretManager) isAuthFailure(err error) bool {
	if err == nil || v.reauth == nil {
		return false
	}
	var respErr *vaultapi.ResponseError
	if errors.As(err, &respErr) {
		return respErr.StatusCode == http.StatusForbidden || respErr.StatusCode == http.StatusUnauthorized
	}
	// The KVv2 helper does not always surface a typed error.
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "permission denied") || strings.Contains(msg, "invalid token")
}

func (v *vaultSecretManager) relogin() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.reauth == nil || v.client == nil {
		return fmt.Errorf("no AppRole credentials configured; a static VAULT_TOKEN cannot be renewed")
	}
	if err := v.reauth(v.client); err != nil {
		return err
	}
	slog.Info("vault: re-authenticated after token expiry")
	// KVv2 borrows the client's token at call time, so the existing handle is
	// still valid — refreshed here only to keep the mount binding explicit.
	v.kv = v.client.KVv2(v.mount)
	return nil
}

func (v *vaultSecretManager) GetSecretString(key string) (string, error) {
	data, err := v.GetSecret(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

package config

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
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

// ErrSecretNotFound means the provider answered definitively that the key does
// not exist — as opposed to the provider being unreachable, unauthorized, or
// otherwise broken. Only the first justifies falling back to another source;
// treating an outage as "not configured" would silently boot the service on a
// stale environment value while the operator believes it is reading Vault.
//
// Every provider must map its store's genuine 404 onto this and nothing else.
var ErrSecretNotFound = errors.New("secret not found")

// envFallbackEnabled reports whether a secret absent from the configured
// provider may be read from the environment instead.
//
// Default ON so a deployment can adopt a secret manager incrementally: move
// JWT keys into Vault today, leave the rest in the environment, migrate the
// remainder later. Set SECRET_STRICT=true once everything is migrated to make
// the provider authoritative — after that a missing secret is a startup
// failure rather than a silent fall back.
func envFallbackEnabled() bool {
	return !strings.EqualFold(GetEnvOrDefault("SECRET_STRICT", "false"), "true")
}

// secretSource is used only for the startup audit line.
type secretSource string

const (
	sourceProvider secretSource = "provider"
	sourceEnvFall  secretSource = "env-fallback"
)

// secretFetchTimeout caps each external provider call.
const secretFetchTimeout = 10 * time.Second

// ────────────────────────────────────────────────── env provider ──────────

type envSecretManager struct{}

func (e *envSecretManager) GetSecret(key string) ([]byte, error) {
	value := os.Getenv(key)
	if value == "" {
		return nil, fmt.Errorf("environment variable %q: %w", key, ErrSecretNotFound)
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
	data, err := os.ReadFile(path) // #nosec G304 -- file secret manager reads from configured directory
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("secret file %q: %w", path, ErrSecretNotFound)
		}
		// A permission error or an I/O fault is NOT "absent" — surface it.
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
		address := GetEnvOrDefault("VAULT_ADDR", "http://localhost:8200") // nosemgrep
		token := os.Getenv("VAULT_TOKEN")
		mount := GetEnvOrDefault("VAULT_MOUNT", "secret")
		// Every secret in the system crosses this connection, including the JWT
		// signing key. The default address is plaintext for local development, so
		// an operator who sets SECRET_PROVIDER=vault and forgets VAULT_ADDR would
		// otherwise ship secrets in the clear without a word.
		if err := requireSecureSecretTransport("VAULT_ADDR", address); err != nil {
			return nil, err
		}
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
			return nil, fmt.Errorf("azure key vault requires AZURE_KEYVAULT_URL: %w", err)
		}
		slog.Info("Secret provider: Azure Key Vault", "url", vaultURL)
		return newAzureKeyVaultManager(vaultURL)

	default:
		// Fail closed. Silently falling back to environment variables meant a typo
		// in SECRET_PROVIDER ("hashicorp" instead of "vault") started the app
		// reading env vars while the operator believed it was reading Vault — and
		// if stale dev values happened to be present in the environment, it would
		// boot with them. config.Init already rejects unknown values via
		// ValidateSecretProvider; this is the second line of defence for any
		// caller that reaches the factory directly.
		return nil, fmt.Errorf("unknown SECRET_PROVIDER %q", SecretProvider)
	}
}

// resolveSecret fetches a key from the configured provider, falling back to the
// environment ONLY when the provider positively reports the key as absent.
// Returns the value, where it came from, and whether it was found at all.
func resolveSecret(key string) ([]byte, secretSource, bool, error) {
	raw, err := fetchFromProvider(key)
	switch {
	case err == nil:
		return raw, sourceProvider, true, nil
	case !errors.Is(err, ErrSecretNotFound):
		// Unreachable, unauthorized, malformed — a real failure. Never fall
		// back: that is the difference between "not configured" and "the secret
		// store is down", and conflating them is how a service quietly starts
		// with the wrong credential.
		return nil, "", false, err
	}

	// Definitively absent from the provider.
	if _, isEnv := activeSecretManager.(*envSecretManager); isEnv {
		return nil, "", false, nil // env WAS the provider; nothing to fall back to
	}
	if !envFallbackEnabled() {
		return nil, "", false, nil
	}
	if value := os.Getenv(key); value != "" {
		return []byte(value), sourceEnvFall, true, nil
	}
	return nil, "", false, nil
}

// fetchFromProvider calls the active provider, retrying only for remote stores
// where a failure may be transient.
func fetchFromProvider(key string) ([]byte, error) {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		secret, err := activeSecretManager.GetSecret(key)
		if err == nil {
			return secret, nil
		}
		lastErr = err
		// A definitive "absent" will not change on retry, and neither will a
		// local read.
		if errors.Is(err, ErrSecretNotFound) || providerIsLocal() {
			return nil, err
		}
		if attempt < 3 {
			slog.Warn("Failed to load secret, retrying", "key", key, "attempt", attempt, "error", err)
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
	return nil, fmt.Errorf("failed to load secret %q after 3 attempts: %w", key, lastErr)
}

// loadSecret fetches a required secret.
func loadSecret(key string) ([]byte, error) {
	if activeSecretManager == nil {
		return nil, fmt.Errorf("secret manager not initialized; ensure initSecretManager is called first")
	}

	raw, source, found, err := resolveSecret(key)
	if err != nil {
		return nil, err
	}
	if !found {
		if envFallbackEnabled() {
			return nil, fmt.Errorf("secret %q not found in provider %q or the environment", key, SecretProvider)
		}
		return nil, fmt.Errorf("secret %q not found in provider %q (SECRET_STRICT is on, so the environment is not consulted)", key, SecretProvider)
	}

	normalized, nerr := normalizeSecret(key, raw)
	if nerr != nil {
		return nil, nerr
	}
	if len(normalized) == 0 {
		return nil, fmt.Errorf("secret %q is empty", key)
	}
	// Record WHERE each secret came from. Without this an operator cannot answer
	// "is this credential actually being served by Vault?" — which is exactly
	// what an auditor asks. The value is never logged, only its origin.
	slog.Info("Loaded secret", "key", key, "source", string(source))
	return normalized, nil
}

// normalizeSecret applies the value handling that must be IDENTICAL across
// every provider, because swapping SECRET_PROVIDER is supposed to be
// transparent to the application.
//
// Two behaviours used to differ per provider:
//
//   - `base64:` decoding was implemented only by the env provider, so a secret
//     stored base64-encoded worked locally and silently arrived still-encoded
//     from AWS, Vault, GCP or Azure.
//   - Trailing whitespace was stripped by GetSecretString but NOT by GetSecret,
//     which returns the raw bytes used for key material. A secret file written
//     the obvious way (`echo value > secret`) carries a trailing newline, so the
//     encryption key the app used was one byte longer than the one the operator
//     stored — a silent mismatch, not an error.
func normalizeSecret(key string, raw []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(raw)
	if encoded, ok := bytes.CutPrefix(trimmed, []byte("base64:")); ok {
		decoded, err := base64.StdEncoding.DecodeString(string(encoded))
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 secret %q: %w", key, err)
		}
		return decoded, nil
	}
	return trimmed, nil
}

// providerIsLocal reports whether the active provider resolves secrets without
// a network call. Retrying a local read is pointless — os.Getenv and a missing
// file are deterministic — and the backoff would add three seconds to a
// misconfiguration that should fail instantly.
func providerIsLocal() bool {
	switch activeSecretManager.(type) {
	case *envSecretManager, *fileSecretManager:
		return true
	default:
		return false
	}
}

// loadSecretOptional fetches a secret that is allowed to be absent, returning
// nil when it is not configured. Any other failure (a malformed base64 payload,
// a provider that is unreachable) is still an error — "the store is down" must
// never be silently read as "this credential is unset".
func loadSecretOptional(key string) ([]byte, error) {
	if activeSecretManager == nil {
		return nil, fmt.Errorf("secret manager not initialized; ensure initSecretManager is called first")
	}
	raw, source, found, err := resolveSecret(key)
	if err != nil {
		return nil, fmt.Errorf("failed to load optional secret %q: %w", key, err)
	}
	if !found {
		return nil, nil
	}
	normalized, nerr := normalizeSecret(key, raw)
	if nerr != nil {
		return nil, nerr
	}
	if len(normalized) == 0 {
		return nil, nil
	}
	slog.Info("Loaded optional secret", "key", key, "source", string(source))
	return normalized, nil
}

// LoadSecretOptional is the exported form of loadSecretOptional, for secrets
// that may legitimately be unset (an empty Redis password, for example).
func LoadSecretOptional(key string) ([]byte, error) {
	return loadSecretOptional(key)
}

// LoadSecretStringOptional is LoadSecretOptional for text values.
func LoadSecretStringOptional(key string) (string, error) {
	data, err := loadSecretOptional(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// LoadSecretString fetches a required secret as text.
func LoadSecretString(key string) (string, error) {
	data, err := loadSecret(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// LoadSecret is the exported entry point for runtime secret refresh.
// It re-fetches a secret through the active provider with up to 3 retries,
// identical to the behaviour used at startup.
func LoadSecret(key string) ([]byte, error) {
	return loadSecret(key)
}

// requireSecureSecretTransport refuses a cleartext secret-store endpoint
// outside development. Loopback stays permitted so a local Vault dev server
// still works, and any TLS endpoint is accepted anywhere.
func requireSecureSecretTransport(name, rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%s is not a valid URL: %w", name, err)
	}
	if strings.EqualFold(u.Scheme, "https") {
		return nil
	}
	host := u.Hostname()
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		slog.Warn("Secret store reached over plaintext HTTP on loopback — acceptable for local development only", "var", name, "address", rawURL)
		return nil
	}
	// Read APP_ENV directly rather than the package var: Init() constructs the
	// secret manager BEFORE it assigns AppEnv, so the var is still empty here.
	if strings.EqualFold(GetEnvOrDefault("APP_ENV", "production"), "production") {
		return fmt.Errorf("%s must use https in production (got %q): secrets would cross the network in cleartext", name, rawURL)
	}
	slog.Warn("Secret store reached over plaintext HTTP — this must be https before production", "var", name, "address", rawURL)
	return nil
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

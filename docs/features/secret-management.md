# Secret Management

> A pluggable secret-loading layer that reads every credential the service needs (JWT keys, encryption keys, DB password, HMAC key, bootstrap token) from one of seven interchangeable providers selected by `SECRET_PROVIDER`, with normalized value handling and optional environment fallback.

| | |
|---|---|
| **Status** | Implemented |
| **Code** | `internal/platform/config` (`secret_manager.go`, `secret_manager_aws.go`, `secret_manager_vault.go`, `secret_manager_gcp.go`, `secret_manager_azure.go`); `internal/platform/runner/secret_refresh.go` |
| **Endpoints** | n/a (startup + background loading; no HTTP surface) |
| **Storage** | n/a (secrets live in the external provider, not in Postgres) |
| **Config** | `SECRET_PROVIDER`, `SECRET_PREFIX`, `SECRET_FILE_PATH`, `SECRET_STRICT`, `SECRET_REFRESH_PERIOD_SECONDS`, plus per-provider vars (`AWS_REGION`, `VAULT_*`, `GCP_PROJECT_ID`, `AZURE_KEYVAULT_URL`) |

## Overview

Every sensitive value the process needs is fetched through a single `SecretManager` interface (`secret_manager.go:17`) rather than read directly from the environment. The concrete provider is chosen once at startup from `SECRET_PROVIDER` (`config.go:134`), so non-sensitive config stays in plain env vars while credentials can be pulled from a managed store without touching call sites. With the default `SECRET_PROVIDER=env` the behavior is byte-identical to reading env vars (`config.go:199-204`), so a fresh deployment needs no secret manager at all.

Seven providers ship:

| `SECRET_PROVIDER` | Backend | Constructor | Network? |
|---|---|---|---|
| `env` (default) | Process environment variables | `envSecretManager` (`secret_manager.go:59`) | local |
| `file` | Files under a directory (Docker/K8s secrets) | `fileSecretManager` (`secret_manager.go:83`) | local |
| `aws_secrets` | AWS Secrets Manager | `newAWSSecretsManager` (`secret_manager_aws.go:41`) | remote |
| `aws_ssm` | AWS SSM Parameter Store (SecureString auto-decrypted) | `newAWSSSMSecretManager` (`secret_manager_aws.go:117`) | remote |
| `vault` | HashiCorp Vault (KV v2) | `newVaultSecretManager` (`secret_manager_vault.go:60`) | remote |
| `gcp` | GCP Secret Manager | `newGCPSecretManager` (`secret_manager_gcp.go:45`) | remote |
| `azure_kv` | Azure Key Vault | `newAzureKeyVaultManager` (`secret_manager_azure.go:52`) | remote |

## How it works

### Startup selection

1. `config.Init()` reads `SECRET_PROVIDER` (default `env`) and `SECRET_PREFIX` (default `maintainerd/auth`) (`config.go:134-135`).
2. `ValidateSecretProvider()` rejects any value outside the seven known providers (`secret_manager.go:400-408`); an unknown value is a startup error.
3. `initSecretManager()` → `newSecretManager()` constructs the concrete provider and stores it in the package-level `activeSecretManager` (`secret_manager.go:112-193`). This runs **before** most other config is parsed, so the loaders below can be used for the rest of `Init()`.

### Key → provider-name mapping

A logical key such as `JWT_PRIVATE_KEY` is mapped to a provider-native name by lowercasing and replacing underscores with hyphens (`jwt-private-key`), then combined with `SECRET_PREFIX` where the provider supports it:

| Provider | Resulting name for `JWT_PRIVATE_KEY` | Code |
|---|---|---|
| `env` | `JWT_PRIVATE_KEY` (raw env var, unmodified) | `secret_manager.go:62` |
| `file` | `<SECRET_FILE_PATH>/jwt-private-key` | `secret_manager.go:86-87` |
| `aws_secrets` | `maintainerd/auth/jwt-private-key` | `secret_manager_aws.go:52-58` |
| `aws_ssm` | `/maintainerd/auth/jwt-private-key` | `secret_manager_aws.go:128-135` |
| `vault` | `<mount>/data/maintainerd/auth/jwt-private-key`, field `value` | `secret_manager_vault.go:119-126` |
| `gcp` | `projects/<GCP_PROJECT_ID>/secrets/jwt-private-key/versions/latest` (**prefix ignored**) | `secret_manager_gcp.go:55-58` |
| `azure_kv` | `jwt-private-key` (**prefix ignored**) | `secret_manager_azure.go:60-62` |

GCP and Azure do not apply `SECRET_PREFIX`; scope access with IAM/vault policies instead (`secret_manager_gcp.go:27-28`).

### Fetch path (`loadSecret` / `LoadSecretOptional`)

1. **`resolveSecret`** (`secret_manager.go:198`) calls the active provider via **`fetchFromProvider`** (`secret_manager.go:226`).
2. Remote providers retry up to **3 times** with linear backoff (1s, 2s); local providers (`env`, `file`) and definitive "not found" answers are not retried (`secret_manager.go:228-244`, `providerIsLocal` at `:308`). Each remote call is capped at a **10s timeout** (`secretFetchTimeout`, `secret_manager.go:55`).
3. **Not-found vs. failure is strictly distinguished.** Only `ErrSecretNotFound` (`secret_manager.go:32`) — the store positively answering "no such key" — permits fallback. Every provider maps only its genuine 404 onto it (AWS `ResourceNotFoundException`/`ParameterNotFound`, Vault HTTP 404, GCP `NotFound`, Azure HTTP 404); throttling, auth errors, and network faults propagate as real errors and **never** fall back (`secret_manager.go:203-208`).
4. **Environment fallback.** If the provider is not `env` and the key is definitively absent, and `SECRET_STRICT` is off (the default), the value is read from `os.Getenv(key)` as a fallback (`secret_manager.go:211-221`). This lets a deployment migrate secrets into a manager incrementally. `SECRET_STRICT=true` disables the fallback and makes the provider authoritative (`envFallbackEnabled`, `secret_manager.go:42-44`).
5. **Normalization** (`normalizeSecret`, `secret_manager.go:292`) runs identically for every provider so swapping `SECRET_PROVIDER` is transparent:
   - Leading/trailing whitespace is trimmed (fixes the trailing newline from `echo value > secret`).
   - A `base64:` prefix triggers standard base64 decoding of the remainder.
   An empty result after normalization is an error for required secrets (`secret_manager.go:268-269`) and treated as "unset" for optional ones (`:336-338`).
6. The origin of each loaded secret (`provider` vs `env-fallback`) is logged for audit — the value is never logged, only its source (`secret_manager.go:271-274`).

### Required vs optional loaders

- `LoadSecret` / `LoadSecretString` — required; a missing value is a hard error whose message reflects whether strict mode consulted the environment (`secret_manager.go:257-262`).
- `LoadSecretOptional` / `LoadSecretStringOptional` — may be absent; return `nil`/`""` when unset but still error on a real provider failure (`secret_manager.go:321-347`).

### Secrets loaded through this layer

| Key | Loader | Required | Site |
|---|---|---|---|
| `JWT_PRIVATE_KEY` / `JWT_PUBLIC_KEY` | `loadSecret` | yes | `config.go:219-224` |
| `APP_ENCRYPTION_KEY` | `loadSecret` (must be 32 bytes / AES-256) | yes | `config.go:232-238` |
| `APP_ENCRYPTION_KEYS_PREVIOUS` | `LoadSecretStringOptional` (comma-separated retired keys) | no | `config.go:242-258` |
| `HMAC_SECRET_KEY` | `loadSecret` (signed-URL signer) | yes | `config.go:263-268` |
| `DB_PASSWORD` | `LoadSecretString` | yes | `config.go:281` |
| `SETUP_BOOTSTRAP_TOKEN` | `LoadSecretStringOptional` | no | `config.go:205` |
| `REDIS_PASSWORD` | `LoadSecretStringOptional` | no | `redis.go:23` |

### Background refresh

`StartSecretRefreshRunner` (`secret_refresh.go:19`) re-fetches secrets on a ticker (`SECRET_REFRESH_PERIOD_SECONDS`, default 300; floored to 5m if non-positive — `workers.go:72-76`). Currently it re-reads `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY` and, only when they differ from the in-memory copy, swaps them and calls `jwt.InitJWTKeys()` so rotated keys take effect without a restart (`secret_refresh.go:38-62`). A read failure is logged and skipped, not fatal.

## Implementation

- **Interface + factory + shared logic**: `internal/platform/config/secret_manager.go` — `SecretManager` interface (`:17`), `newSecretManager` factory (`:132`), `resolveSecret`/`fetchFromProvider` (`:198`, `:226`), `normalizeSecret` (`:292`), exported `LoadSecret*` entry points (`:345-372`), `ValidateSecretProvider` (`:400`).
- **AWS**: `secret_manager_aws.go` — Secrets Manager (`awsSecretsManager`, `:36`) returns `SecretString` or `SecretBinary`; SSM Parameter Store (`awsSSMSecretManager`, `:112`) reads with `WithDecryption: true`. Both use `awsconfig.LoadDefaultConfig` (env keys or IAM role) with region from `AWS_REGION`.
- **Vault**: `secret_manager_vault.go` — KV v2 via `client.KVv2(mount)`. Auth by static `VAULT_TOKEN` or AppRole (`VAULT_ROLE_ID` + `VAULT_SECRET_ID`, `:98-117`). On a 401/403 it re-authenticates once and retries (`GetSecret` → `isAuthFailure` → `relogin`, `:128-204`) so an expired AppRole token self-heals; a static token cannot be renewed. Reads the `value` field (override `VAULT_SECRET_FIELD`).
- **GCP**: `secret_manager_gcp.go` — `AccessSecretVersion` against `.../versions/latest` using Application Default Credentials.
- **Azure**: `secret_manager_azure.go` — `azsecrets` client with `DefaultAzureCredential` (env → workload identity → managed identity → Azure CLI).
- **Refresh runner**: `internal/platform/runner/secret_refresh.go`, started in `cmd/server/workers.go:72-76`.
- **Config wiring**: `internal/platform/config/config.go` — `Init()` selects the provider (`:133-143`) then loads each secret (`:205-281`).

## Configuration

| Env var | Default | Applies to | Purpose |
|---|---|---|---|
| `SECRET_PROVIDER` | `env` | all | Selects the provider: `env` / `file` / `aws_secrets` / `aws_ssm` / `vault` / `gcp` / `azure_kv`. Unknown value fails startup. |
| `SECRET_PREFIX` | `maintainerd/auth` | AWS, Vault | Name/path prefix for secrets. Ignored by `env`, `file`, `gcp`, `azure_kv`. |
| `SECRET_STRICT` | `false` | all remote | `true` disables env fallback — a secret missing from the provider is a startup failure. |
| `SECRET_FILE_PATH` | `/run/secrets` | `file` | Directory holding one file per secret (`jwt-private-key`, …). |
| `SECRET_REFRESH_PERIOD_SECONDS` | `300` | refresh runner | How often refreshable secrets (JWT keys) are re-read; floored to 5m if ≤ 0. |
| `AWS_REGION` | `us-east-1` | `aws_secrets`, `aws_ssm` | AWS region. |
| `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` | (unset) | AWS | Static credentials; omit to use the instance IAM role. |
| `GCP_PROJECT_ID` | (required) | `gcp` | GCP project; startup fails if unset. |
| `AZURE_KEYVAULT_URL` | (required) | `azure_kv` | Key Vault endpoint URL; startup fails if unset. |
| `VAULT_ADDR` | `http://localhost:8200` | `vault` | Vault address. Must be `https` in production (see below). |
| `VAULT_TOKEN` | (unset) | `vault` | Static token; if empty, AppRole login is used. |
| `VAULT_ROLE_ID` / `VAULT_SECRET_ID` | (unset) | `vault` | AppRole credentials (both required when `VAULT_TOKEN` is empty). |
| `VAULT_MOUNT` | `secret` | `vault` | KV v2 mount path. |
| `VAULT_SECRET_FIELD` | `value` | `vault` | Field within each Vault secret to read. |

There are no per-tenant secret-management settings; provider selection is process-global. See also [Environment variables](../contributing/environment-variables.md).

## Security considerations

- **Cleartext-transport guard.** For the Vault provider, `requireSecureSecretTransport` rejects a non-`https`, non-loopback `VAULT_ADDR` when `APP_ENV=production` — otherwise the JWT signing key and every other secret would cross the network in the clear (`secret_manager.go:161-163`, `:377-397`). Loopback HTTP is allowed with a warning for local dev; non-loopback HTTP outside production warns.
- **Fail-closed provider selection.** An unknown/typo'd `SECRET_PROVIDER` is rejected twice — by `ValidateSecretProvider` in `Init()` and by the factory's `default` case (`secret_manager.go:183-192`) — rather than silently falling back to env vars, which could otherwise boot the service on stale local values while the operator believes it is reading Vault.
- **Outage ≠ absent.** Provider errors that are not a definitive 404 never trigger env fallback, so "the secret store is down" cannot be mistaken for "this credential is unset" and start the service with the wrong (or stale env) credential (`secret_manager.go:203-208`, `:26-32`).
- **Values are never logged**, only their origin (`provider` / `env-fallback`) for audit (`secret_manager.go:273-274`).
- **Encryption-key length is enforced**: `APP_ENCRYPTION_KEY` must be exactly 32 bytes (AES-256) and each retired key likewise (`config.go:236-238`, `:252-253`).
- **Vault token self-healing** re-authenticates via AppRole on a 401/403 so a lapsed short-TTL token does not silently stop secret refresh (`secret_manager_vault.go:50-58`, `:128-204`).
- **`SECRET_STRICT=true`** makes a managed provider fully authoritative once migration is complete, closing the env-fallback path.

## Related

- [Authentication](./authentication.md) — consumes the JWT signing keys loaded here (RS256).
- [Sessions](./sessions.md) — refresh-token and session state protected by `APP_ENCRYPTION_KEY`.
- [Security settings](./security-settings.md) — signed URLs use the `HMAC_SECRET_KEY` loaded here.
- [Setup and bootstrap](./setup-and-bootstrap.md) — `SETUP_BOOTSTRAP_TOKEN` is loaded through this layer.

# Environment Variables — Production Deployment

This document describes every environment variable required to run **Maintainerd Auth** in a production environment.

> **Looking for local development setup?**
> See [`docs/contributing/environment-variables.md`](../contributing/environment-variables.md) instead.

---

## Quick Setup

Copy the full block below as your starting point. Replace every value marked with `← replace` before deploying.
Refer to the relevant section below for instructions on generating each secret.

```env
# =============================================================================
# APP
# =============================================================================
APP_VERSION="v1"
APP_PUBLIC_HOSTNAME="https://auth.yourdomain.com"           # ← replace
APP_PRIVATE_HOSTNAME="https://auth-internal.yourdomain.com" # ← replace

# =============================================================================
# FRONTEND
# =============================================================================
ACCOUNT_HOSTNAME="https://account.yourdomain.com" # ← replace
AUTH_HOSTNAME="https://auth.yourdomain.com"        # ← replace

# =============================================================================
# DATABASE
# =============================================================================
DB_HOST="your-postgres.rds.amazonaws.com" # ← replace
DB_PORT="5432"
DB_USER="maintainerd_auth"                # ← replace
DB_PASSWORD="<generated-32-char-secret>"  # ← replace  (openssl rand -base64 32)
DB_NAME="maintainerd"
DB_SSLMODE="require"
DB_TABLE_PREFIX="md_"

# =============================================================================
# REDIS
# =============================================================================
REDIS_HOST="your-redis.cache.amazonaws.com"                         # ← replace
REDIS_PORT="6379"
REDIS_PASSWORD="<generated-secret>"                                 # ← replace
REDIS_CONNECTION_STRING="rediss://:<generated-secret>@your-redis.cache.amazonaws.com:6379" # ← replace

# =============================================================================
# EMAIL
# =============================================================================
SMTP_HOST="smtp.sendgrid.net"          # ← replace with your provider
SMTP_PORT="587"
SMTP_USER="apikey"                     # ← replace
SMTP_PASS="<retrieved-from-secret-manager>" # ← replace
SMTP_FROM_EMAIL="noreply@yourdomain.com"    # ← replace (must match SPF/DKIM)
SMTP_FROM_NAME="Maintainerd"
EMAIL_LOGO_URL="https://cdn.yourdomain.com/logo.png" # ← replace

# =============================================================================
# SECRET MANAGEMENT
# =============================================================================
SECRET_PROVIDER=aws_secrets       # ← set to your provider: aws_secrets | aws_ssm | vault | gcp | azure_kv | file
SECRET_PREFIX=maintainerd/auth

## --- File provider ---
# SECRET_FILE_PATH=/run/secrets

## --- AWS providers (aws_secrets / aws_ssm) ---
# AWS_REGION=us-east-1            # ← required for aws_secrets / aws_ssm
# AWS_ACCESS_KEY_ID=              # ← prefer IAM roles over static keys
# AWS_SECRET_ACCESS_KEY=

## --- HashiCorp Vault provider (vault) ---
# VAULT_ADDR=https://vault.yourdomain.com # ← required for vault
# VAULT_TOKEN=                            # ← prefer AppRole auth over static tokens
# VAULT_MOUNT=secret
# VAULT_ROLE_ID=                          # ← for AppRole auth (when VAULT_TOKEN is empty)
# VAULT_SECRET_ID=
# VAULT_SECRET_FIELD=value

## --- GCP Secret Manager provider (gcp) ---
# GCP_PROJECT_ID=your-project-id  # ← required for gcp

## --- Azure Key Vault provider (azure_kv) ---
# AZURE_KEYVAULT_URL=https://your-vault.vault.azure.net # ← required for azure_kv
# AZURE_TENANT_ID=                # ← for service principal auth
# AZURE_CLIENT_ID=
# AZURE_CLIENT_SECRET=

# =============================================================================
# JWT  ← generate with: ./scripts/generate-jwt-keys.sh 4096 /tmp/jwt-keys
# Store keys in your secret manager — never leave them as plain files on disk
# =============================================================================
JWT_PRIVATE_KEY="<retrieved-from-secret-manager>" # ← replace
JWT_PUBLIC_KEY="<retrieved-from-secret-manager>"  # ← replace
```

> **Every `← replace` value is required before deployment.** The service will fail to start or operate insecurely if any are left as placeholders.
> Use the [Pre-Deployment Checklist](#pre-deployment-checklist) to verify before going live.

---

## Table of Contents

- [Security Principles](#security-principles)
- [Application](#application)
- [Frontend Hostnames](#frontend-hostnames)
- [Database](#database)
- [Redis](#redis)
- [Email (SMTP)](#email-smtp)
- [Secret Management](#secret-management)
- [JWT Configuration](#jwt-configuration)
- [Checklist](#pre-deployment-checklist)

---

## Security Principles

Before configuring any variable, follow these non-negotiable rules:

- ❌ **Never** store secrets in source code, Docker images, or CI logs.
- ❌ **Never** use default or example values in production.
- ✅ **Always** use a dedicated secret manager (AWS Secrets Manager, HashiCorp Vault, etc.).
- ✅ **Always** rotate credentials on a defined schedule (JWT keys every 90 days, DB passwords every 180 days).
- ✅ **Always** restrict access to secrets using least-privilege IAM policies or Vault policies.
- ✅ **Always** enable TLS for database, Redis, and SMTP connections in production.

---

## Application

| Variable | Required | Description |
|---|---|---|
| `APP_VERSION` | ✅ | API version prefix. Set to `v1` unless you are running a major version migration. |
| `APP_PUBLIC_HOSTNAME` | ✅ | Fully-qualified public base URL, e.g. `https://auth.yourdomain.com`. Must use HTTPS. |
| `APP_PRIVATE_HOSTNAME` | ✅ | Internal base URL, e.g. `https://auth-internal.yourdomain.com`. Must be unreachable from the public internet. |

```env
APP_VERSION="v1"
APP_PUBLIC_HOSTNAME="https://auth.yourdomain.com"
APP_PRIVATE_HOSTNAME="https://auth-internal.yourdomain.com"
```

---

## Frontend Hostnames

| Variable | Required | Description |
|---|---|---|
| `ACCOUNT_HOSTNAME` | ✅ | Production URL of the Account portal. Used for CORS and redirect URIs. Must use HTTPS. |
| `AUTH_HOSTNAME` | ✅ | Production URL of the Auth portal. Must use HTTPS. |

```env
ACCOUNT_HOSTNAME="https://account.yourdomain.com"
AUTH_HOSTNAME="https://auth.yourdomain.com"
```

---

## Database

| Variable | Required | Description |
|---|---|---|
| `DB_HOST` | ✅ | Hostname of your managed PostgreSQL instance (RDS, Cloud SQL, etc.). |
| `DB_PORT` | ✅ | PostgreSQL port. Default: `5432`. |
| `DB_USER` | ✅ | Database user. Use a dedicated, least-privilege user — not a superuser. |
| `DB_PASSWORD` | ✅ | Strong randomly generated password. Minimum 32 characters. Store in your secret manager. |
| `DB_NAME` | ✅ | Database name. |
| `DB_SSLMODE` | ✅ | **Must be `require` or `verify-full` in production.** Never use `disable`. |
| `DB_TABLE_PREFIX` | ❌ | Table name prefix. Default: `md_`. Only change if sharing a schema with other services. |

```env
DB_HOST="your-postgres.rds.amazonaws.com"
DB_PORT="5432"
DB_USER="maintainerd_auth"
DB_PASSWORD="<retrieved-from-secret-manager>"
DB_NAME="maintainerd"
DB_SSLMODE="require"
DB_TABLE_PREFIX="md_"
```

**Generate a strong password:**

```bash
openssl rand -base64 32
```

---

## Redis

| Variable | Required | Description |
|---|---|---|
| `REDIS_CONNECTION_STRING` | ✅ | Full Redis URL. Takes precedence over individual fields. Use `rediss://` (TLS) in production. |
| `REDIS_HOST` | ✅ | Redis hostname (used when connection string is not set). |
| `REDIS_PORT` | ✅ | Redis port. Default: `6379`. |
| `REDIS_PASSWORD` | ✅ | Redis `AUTH` password. Required in production. |

```env
REDIS_HOST="your-redis.cache.amazonaws.com"
REDIS_PORT="6379"
REDIS_PASSWORD="<retrieved-from-secret-manager>"
REDIS_CONNECTION_STRING="rediss://:your-password@your-redis.cache.amazonaws.com:6379"
```

> Use `rediss://` (double-s) for TLS-encrypted connections. Supported by ElastiCache, Redis Cloud, and Upstash.  
> URL-encode special characters in passwords: `@` → `%40`, `#` → `%23`.

---

## Email (SMTP)

| Variable | Required | Description |
|---|---|---|
| `SMTP_HOST` | ✅ | SMTP server hostname. |
| `SMTP_PORT` | ✅ | `587` for STARTTLS, `465` for implicit TLS. Never use port `25` in production. |
| `SMTP_USER` | ✅ | SMTP authentication username. |
| `SMTP_PASS` | ✅ | SMTP password or API key. Store in your secret manager. |
| `SMTP_FROM_EMAIL` | ✅ | Verified sender address. Must match your domain's SPF/DKIM records. |
| `SMTP_FROM_NAME` | ✅ | Display name shown to recipients. |
| `EMAIL_LOGO_URL` | ❌ | Publicly accessible URL of the logo image in HTML emails. Use a CDN URL. |

```env
SMTP_HOST="smtp.sendgrid.net"
SMTP_PORT="587"
SMTP_USER="apikey"
SMTP_PASS="<retrieved-from-secret-manager>"
SMTP_FROM_EMAIL="noreply@yourdomain.com"
SMTP_FROM_NAME="Maintainerd"
EMAIL_LOGO_URL="https://cdn.yourdomain.com/logo.png"
```

> **Recommended providers for production:** [SendGrid](https://sendgrid.com), [Postmark](https://postmarkapp.com), [Resend](https://resend.com), [Mailgun](https://www.mailgun.com).  
> Transactional email providers offer better deliverability, bounce handling, and analytics than raw SMTP.

---

## Secret Management

### Core Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `SECRET_PROVIDER` | ✅ | `env` | Secret backend. Use `env` only for local dev. Production: `aws_secrets`, `aws_ssm`, `vault`, `gcp`, `azure_kv`, or `file`. |
| `SECRET_PREFIX` | ❌ | `maintainerd/auth` | Namespace prefix for secrets in external providers. Not used by `env`, `file`, or `gcp`. |

### Provider-Specific Variables

#### `file` — File-Based Secrets (Docker / Kubernetes)

| Variable | Required | Default | Description |
|---|---|---|---|
| `SECRET_FILE_PATH` | ❌ | `/run/secrets` | Base path for file-based secrets. |

Key names are lowercased with underscores replaced by hyphens.
Example: `JWT_PRIVATE_KEY` → `<SECRET_FILE_PATH>/jwt-private-key`

#### `aws_secrets` — AWS Secrets Manager

| Variable | Required | Default | Description |
|---|---|---|---|
| `AWS_REGION` | ✅ | `us-east-1` | AWS region where secrets are stored. |
| `AWS_ACCESS_KEY_ID` | ❌ | — | Only if IAM roles are unavailable. |
| `AWS_SECRET_ACCESS_KEY` | ❌ | — | Only if IAM roles are unavailable. |

Secret naming: `<SECRET_PREFIX>/<key-lowercased-hyphens>`
Example: `JWT_PRIVATE_KEY` → `maintainerd/auth/jwt-private-key`

```bash
# Store a secret in AWS Secrets Manager
aws secretsmanager create-secret \
  --name "maintainerd/auth/jwt-private-key" \
  --secret-string file:///tmp/jwt-keys/jwt_private.pem
```

#### `aws_ssm` — AWS SSM Parameter Store

| Variable | Required | Default | Description |
|---|---|---|---|
| `AWS_REGION` | ✅ | `us-east-1` | AWS region. |
| `AWS_ACCESS_KEY_ID` | ❌ | — | Only if IAM roles are unavailable. |
| `AWS_SECRET_ACCESS_KEY` | ❌ | — | Only if IAM roles are unavailable. |

Parameter naming: `/<SECRET_PREFIX>/<key-lowercased-hyphens>`
Example: `JWT_PRIVATE_KEY` → `/maintainerd/auth/jwt-private-key`
SecureString parameters are automatically decrypted using the default KMS key.

```bash
# Store a parameter in SSM
aws ssm put-parameter \
  --name "/maintainerd/auth/jwt-private-key" \
  --type SecureString \
  --value file:///tmp/jwt-keys/jwt_private.pem
```

#### `vault` — HashiCorp Vault (KV v2)

| Variable | Required | Default | Description |
|---|---|---|---|
| `VAULT_ADDR` | ✅ | `http://localhost:8200` | Vault server address. **Must use HTTPS in production.** |
| `VAULT_TOKEN` | ❌ | — | Static token. Set this **or** use AppRole below. |
| `VAULT_MOUNT` | ❌ | `secret` | KV v2 mount path. |
| `VAULT_ROLE_ID` | ❌ | — | AppRole role ID (used when `VAULT_TOKEN` is empty). **Recommended for production.** |
| `VAULT_SECRET_ID` | ❌ | — | AppRole secret ID (used when `VAULT_TOKEN` is empty). |
| `VAULT_SECRET_FIELD` | ❌ | `value` | Field name within the KV secret that holds the value. |

Secret path: `<VAULT_MOUNT>/data/<SECRET_PREFIX>/<key-lowercased-hyphens>`

```bash
# Store a secret in Vault
vault kv put secret/maintainerd/auth/jwt-private-key value=@jwt_private.pem
```

> **Always use AppRole authentication in production** — static tokens do not support automatic renewal or revocation.

#### `gcp` — GCP Secret Manager

| Variable | Required | Default | Description |
|---|---|---|---|
| `GCP_PROJECT_ID` | ✅ | — | GCP project ID. |

Authentication uses **Application Default Credentials (ADC)**:
- **GKE / Cloud Run**: Workload Identity is used automatically.
- **Compute Engine**: Attached service account is used.
- **Local development**: `gcloud auth application-default login`.

Secret naming: `projects/<GCP_PROJECT_ID>/secrets/<key-lowercased-hyphens>/versions/latest`

```bash
# Create a secret in GCP Secret Manager
echo -n "$(cat jwt_private.pem)" | \
  gcloud secrets create jwt-private-key --data-file=- --project=my-project
```

> `SECRET_PREFIX` is not used by the GCP provider. Use IAM policies to scope access instead.

#### `azure_kv` — Azure Key Vault

| Variable | Required | Default | Description |
|---|---|---|---|
| `AZURE_KEYVAULT_URL` | ✅ | — | Key Vault endpoint, e.g. `https://my-vault.vault.azure.net`. |
| `AZURE_TENANT_ID` | ❌ | — | Azure AD tenant ID (for service principal auth). |
| `AZURE_CLIENT_ID` | ❌ | — | Service principal / managed identity client ID. |
| `AZURE_CLIENT_SECRET` | ❌ | — | Service principal client secret. |

Authentication uses **DefaultAzureCredential**, which tries in order:
1. Environment variables (`AZURE_TENANT_ID` + `AZURE_CLIENT_ID` + `AZURE_CLIENT_SECRET`)
2. Workload Identity (AKS)
3. Managed Identity (App Service, Azure Functions, VM)
4. Azure CLI

Secret naming: `<key-lowercased-hyphens>`
Example: `JWT_PRIVATE_KEY` → `jwt-private-key`

```bash
# Store a secret in Azure Key Vault
az keyvault secret set \
  --vault-name my-vault \
  --name jwt-private-key \
  --file jwt_private.pem
```

> **Use Managed Identity in production** — avoid service principal secrets when possible.

### Provider Recommendations by Platform

| Platform | `SECRET_PROVIDER` | Authentication |
|---|---|---|
| AWS ECS / Lambda | `aws_secrets` or `aws_ssm` | IAM task/execution role |
| GCP GKE / Cloud Run | `gcp` | Workload Identity |
| Azure AKS / App Service | `azure_kv` | Managed Identity |
| Kubernetes (any cloud) | `file` | Kubernetes Secrets mounted as volumes |
| Docker Swarm | `file` | Docker Secrets |
| Self-hosted / bare metal | `vault` | AppRole auth |

---

## JWT Configuration

| Variable | Required | Description |
|---|---|---|
| `JWT_PRIVATE_KEY` | ✅ | PEM-encoded RSA private key. Newlines escaped as `\n` for inline use. Store in your secret manager — never in env files on disk. |
| `JWT_PUBLIC_KEY` | ✅ | PEM-encoded RSA public key. Can be distributed to other services that need to verify tokens. |

**Generate a production key pair:**

```bash
./scripts/generate-jwt-keys.sh 4096 /tmp/jwt-keys
```

**Store keys in AWS Secrets Manager:**

```bash
aws secretsmanager create-secret \
  --name "maintainerd/auth/jwt-private-key" \
  --secret-string file:///tmp/jwt-keys/jwt_private.pem

aws secretsmanager create-secret \
  --name "maintainerd/auth/jwt-public-key" \
  --secret-string file:///tmp/jwt-keys/jwt_public.pem
```

**Store keys in HashiCorp Vault:**

```bash
vault kv put secret/maintainerd/auth/jwt-private-key value=@/tmp/jwt-keys/jwt_private.pem
vault kv put secret/maintainerd/auth/jwt-public-key value=@/tmp/jwt-keys/jwt_public.pem
```

**Store keys in GCP Secret Manager:**

```bash
gcloud secrets create jwt-private-key \
  --data-file=/tmp/jwt-keys/jwt_private.pem --project=my-project

gcloud secrets create jwt-public-key \
  --data-file=/tmp/jwt-keys/jwt_public.pem --project=my-project
```

**Store keys in Azure Key Vault:**

```bash
az keyvault secret set --vault-name my-vault \
  --name jwt-private-key --file /tmp/jwt-keys/jwt_private.pem

az keyvault secret set --vault-name my-vault \
  --name jwt-public-key --file /tmp/jwt-keys/jwt_public.pem
```

**Key rotation procedure:**

1. Generate a new key pair with `./scripts/generate-jwt-keys.sh`.
2. Deploy with **both** the old public key and new public key accepted (grace period).
3. Once all tokens signed with the old key have expired, remove the old public key.
4. Rotate every **90 days** minimum.

---

## Pre-Deployment Checklist

Use this checklist before every production deployment.

- [ ] `DB_SSLMODE` is set to `require` or `verify-full`
- [ ] `REDIS_CONNECTION_STRING` uses `rediss://` (TLS)
- [ ] `APP_PUBLIC_HOSTNAME` and `APP_PRIVATE_HOSTNAME` use HTTPS
- [ ] `SECRET_PROVIDER` is **not** set to `env`
- [ ] JWT keys are stored in the secret manager, not in a `.env` file on disk
- [ ] SMTP credentials are stored in the secret manager
- [ ] If using `vault`, `VAULT_ADDR` uses HTTPS and AppRole auth is configured (not a static token)
- [ ] If using `gcp`, `GCP_PROJECT_ID` is set and Workload Identity is configured
- [ ] If using `azure_kv`, Managed Identity is used (not service principal secrets)
- [ ] Database password is at least 32 characters, randomly generated
- [ ] Redis password is set and strong
- [ ] `JWT_PRIVATE_KEY` permissions are `600` on any filesystem where it is stored
- [ ] No `.env` files are present on the production host
- [ ] Key rotation schedule is documented and owned by a team member


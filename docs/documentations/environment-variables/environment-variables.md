# Environment Variables — Production Deployment

This document describes every environment variable required to run **Maintainerd Auth** in a production environment.

> **Looking for local development setup?**
> See [`docs/contributing/environment-variables.md`](../contributing/environment-variables.md) instead.

---

## Quick Setup

Copy the full block below as your starting point. Replace every value marked with `← replace` before deploying.
Refer to the relevant section below for instructions on generating each secret.

```env
# FRONTEND
# DATABASE
# REDIS
# EMAIL
# SMS
# SECRET MANAGEMENT
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

# Store keys in your secret manager — never leave them as plain files on disk
# OPENTELEMETRY (TRACING)  — optional, disabled by default
> **Every `← replace` value is required before deployment.** The service will fail to start or operate insecurely if any are left as placeholders.
> Use the [Pre-Deployment Checklist](#pre-deployment-checklist) to verify before going live.

---

## Table of Contents

- [Security Principles](#security-principles)
- [Application](#application)
- [Frontend Hostnames](#frontend-hostnames)
- [Database](#database)
- [Redis](#redis)
- [Secret Management](#secret-management)
- [JWT Configuration](#jwt-configuration)
- [OpenTelemetry (Tracing)](#opentelemetry-tracing)
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
| `MANAGEMENT_PORT` | ❌ | Dedicated operations listener for `/metrics`, `/readyz`, `/livez`, and `/openapi.json`. Defaults to `8082`. |

```env
APP_VERSION="v1"
APP_PUBLIC_HOSTNAME="https://auth.yourdomain.com"
APP_PRIVATE_HOSTNAME="https://auth-internal.yourdomain.com"
MANAGEMENT_PORT="8082"
```

---

## Frontend Hostnames

| Variable | Required | Description |
|---|---|---|
| `APP_FRONTEND_IDENTITY_HOSTNAME` | ✅ | Production URL of the Account portal. Used for CORS and redirect URIs. Must use HTTPS. |
| `APP_FRONTEND_CONSOLE_HOSTNAME` | ✅ | Production URL of the Auth portal. Must use HTTPS. |

```env
APP_FRONTEND_IDENTITY_HOSTNAME="https://account.yourdomain.com"
APP_FRONTEND_CONSOLE_HOSTNAME="https://auth.yourdomain.com"
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

## Email & SMS Delivery

Email and SMS credentials are no longer configured via environment variables.
They are managed per-tenant through the admin API and stored in database tables:

| Concern | Table | Admin API |
|---|---|---|
| Email delivery (SMTP, SES, SendGrid, etc.) | `email_config` | `GET/PUT /api/v1/email-config` |
| SMS delivery (Twilio, SNS, Vonage) | `sms_config` | `GET/PUT /api/v1/sms-config` |
| SMS daily budget cap | `sms_config.daily_send_limit` | `PUT /api/v1/sms-config` |
| Email logo branding | `email_config.logo_url` | `PUT /api/v1/email-config` |

Each tenant configures its own credentials. The system tenant's configuration
acts as the global fallback when a tenant has no entry of its own.
Credentials are encrypted at rest using `APP_ENCRYPTION_KEY`.

---

## Secret Management

### Core Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `SECRET_PROVIDER` | ✅ | `env` | Secret backend. Use `env` only for local dev. Production: `aws_secrets`, `aws_ssm`, `vault`, `gcp`, `azure_kv`, or `file`. |
| `SECRET_PREFIX` | ❌ | `maintainerd/auth` | Namespace prefix for secrets in external providers. Not used by `env`, `file`, or `gcp`. |
| `SECRET_REFRESH_PERIOD_SECONDS` | ❌ | `300` | Background hot-reload interval for re-reading refreshable secrets from the active provider. |
| `JWT_KEY_ROTATION_PERIOD_SECONDS` | ❌ | `86400` | Background JWT signing-key rotation interval. Invalid or non-positive values fall back to 24 hours at runtime. |

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

## OpenTelemetry (Tracing)

Maintainerd Auth has built-in [OpenTelemetry](https://opentelemetry.io/) tracing that provides end-to-end distributed observability across HTTP requests, gRPC calls, database queries, Redis commands, and outgoing SMTP email sends.

Tracing is **disabled by default**. Set `OTEL_ENABLED=true` to activate it.

| Variable | Required | Default | Description |
|---|---|---|---|
| `OTEL_ENABLED` | ❌ | `false` | Set to `true` to enable tracing. When `false`, a no-op tracer is installed (zero overhead). |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | ❌ | `localhost:4317` | gRPC endpoint of the OpenTelemetry Collector. In production, point to your collector sidecar or cluster service. |
| `OTEL_SERVICE_NAME` | ❌ | `maintainerd-auth` | Service name attached to every span. Useful for distinguishing multiple instances. |

> All standard `OTEL_*` environment variables defined by the [OpenTelemetry SDK specification](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/) are supported automatically. Key production variables include:
>
> | Variable | Example | Purpose |
> |---|---|---|
> | `OTEL_EXPORTER_OTLP_HEADERS` | `Authorization=Bearer <token>` | Authenticate with managed tracing backends (e.g. Grafana Cloud, Honeycomb). |
> | `OTEL_EXPORTER_OTLP_INSECURE` | `false` | Whether to skip TLS verification. **Must be `false` in production.** |
> | `OTEL_TRACES_SAMPLER` | `parentbased_traceidratio` | Sampling strategy. Use head-based sampling to control volume. |
> | `OTEL_TRACES_SAMPLER_ARG` | `0.1` | Sampler argument (e.g. 10% sampling rate). |

### What is instrumented

| Layer | Instrumentation |
|---|---|
| HTTP (REST) | Automatic spans for every inbound request (method, route, status code). |
| gRPC | Automatic spans for every inbound RPC. |
| PostgreSQL | Automatic spans for every query and transaction. |
| Redis | Automatic spans for every Redis command. |
| SMTP (email) | Explicit spans for outgoing email sends (host, port, recipient, subject). |
| Logs | `trace_id` and `span_id` are injected into structured JSON log output for correlation. |

### Recommended collector architecture

Deploy the [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) as a sidecar or DaemonSet and export traces to your backend of choice:

| Backend | Notes |
|---|---|
| [Grafana Tempo](https://grafana.com/oss/tempo/) | Open-source, pairs with Grafana for visualization. |
| [Jaeger](https://www.jaegertracing.io/) | Open-source, self-hosted. |
| [Honeycomb](https://www.honeycomb.io/) | Managed, excellent query engine. |
| [Datadog](https://www.datadoghq.com/) | Managed, full-stack observability. |
| [AWS X-Ray](https://aws.amazon.com/xray/) | Native AWS integration. |
| [GCP Cloud Trace](https://cloud.google.com/trace) | Native GCP integration. |

**Example (Kubernetes sidecar)**

```env
OTEL_ENABLED="true"
OTEL_EXPORTER_OTLP_ENDPOINT="localhost:4317"   # collector sidecar
OTEL_SERVICE_NAME="maintainerd-auth"
```

**Example (Grafana Cloud)**

```env
OTEL_ENABLED="true"
OTEL_EXPORTER_OTLP_ENDPOINT="tempo-us-central1.grafana.net:443"
OTEL_EXPORTER_OTLP_HEADERS="Authorization=Basic <base64-encoded-credentials>"
OTEL_SERVICE_NAME="maintainerd-auth"
```

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
- [ ] If `OTEL_ENABLED=true`, `OTEL_EXPORTER_OTLP_ENDPOINT` points to a reachable collector and `OTEL_EXPORTER_OTLP_INSECURE` is not `true`

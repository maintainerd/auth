# Environment Variables

This document describes every environment variable recognised by **Maintainerd Auth**.  
Copy `.env.example` to `.env` (or `.env.local` for local overrides) and fill in the values described below before starting the service.

```bash
cp .env.example .env
```

> **Security notice** — Never commit a `.env` file that contains real credentials.
> All three files (`.env`, `.env.local`, `.env.example`) are listed in `.gitignore` and must stay that way.

---

## Quick Setup

Copy the full block below into your `.env` file. The defaults work out of the box with `docker-compose up`.
Only the marked variables **require your own values** — read the relevant section below for instructions on generating them.

```env
# =============================================================================
# APP
# =============================================================================
APP_VERSION="v1"
APP_PUBLIC_HOSTNAME="http://public.api.maintainerd.auth"
APP_PRIVATE_HOSTNAME="http://private.api.maintainerd.auth"

# =============================================================================
# FRONTEND
# =============================================================================
ACCOUNT_HOSTNAME="http://account.maintainerd.local"
AUTH_HOSTNAME="http://auth.maintainerd.local"

# =============================================================================
# DATABASE
# =============================================================================
DB_HOST="postgres-db"
DB_PORT="5432"
DB_USER="devuser"
DB_PASSWORD="Pass123"
DB_NAME="maintainerd"
DB_SSLMODE="disable"
DB_TABLE_PREFIX="md_"

# =============================================================================
# REDIS
# =============================================================================
REDIS_HOST="redis-db"
REDIS_PORT="6379"
REDIS_PASSWORD="Pass123"
REDIS_CONNECTION_STRING="redis://:Pass123@redis-db:6379"

# =============================================================================
# EMAIL  ← replace with your own SMTP credentials
# See: #email-smtp for Gmail App Password instructions
# =============================================================================
SMTP_HOST="smtp.gmail.com"
SMTP_PORT="587"
SMTP_USER="you@gmail.com"
SMTP_PASS="your-app-password"
SMTP_FROM_EMAIL="you@gmail.com"
SMTP_FROM_NAME="Maintainerd"
EMAIL_LOGO_URL=""

# =============================================================================
# SECRET MANAGEMENT
# =============================================================================
SECRET_PROVIDER=env
SECRET_PREFIX=maintainerd/auth

## --- File provider (Docker / Kubernetes Secrets) ---
# SECRET_FILE_PATH=/run/secrets

## --- AWS providers (aws_secrets / aws_ssm) ---
# AWS_REGION=us-east-1
# AWS_ACCESS_KEY_ID=
# AWS_SECRET_ACCESS_KEY=

## --- HashiCorp Vault provider (vault) ---
# VAULT_ADDR=http://localhost:8200
# VAULT_TOKEN=
# VAULT_MOUNT=secret
# VAULT_ROLE_ID=
# VAULT_SECRET_ID=
# VAULT_SECRET_FIELD=value

## --- GCP Secret Manager provider (gcp) ---
# GCP_PROJECT_ID=

## --- Azure Key Vault provider (azure_kv) ---
# AZURE_KEYVAULT_URL=
# AZURE_TENANT_ID=
# AZURE_CLIENT_ID=
# AZURE_CLIENT_SECRET=

# =============================================================================
# JWT  ← generate your own keys, see: #generating-a-key-pair
# Run: ./scripts/generate-jwt-keys.sh  then paste the output of keys/jwt_env_vars.txt here
# =============================================================================
JWT_PRIVATE_KEY=""
JWT_PUBLIC_KEY=""

# =============================================================================
# OPENTELEMETRY (TRACING)  — optional, disabled by default
# =============================================================================
OTEL_ENABLED="false"
# OTEL_EXPORTER_OTLP_ENDPOINT="localhost:4317"
# OTEL_SERVICE_NAME="maintainerd-auth"
```

> **Variables that need your attention before first run:**
> - 📧 `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM_EMAIL` — your email credentials ([instructions](#email-smtp))
> - 🔑 `JWT_PRIVATE_KEY`, `JWT_PUBLIC_KEY` — run `./scripts/generate-jwt-keys.sh` ([instructions](#generating-a-key-pair))
>
> Everything else works as-is with Docker Compose.

---

## Table of Contents

- [Application](#application)
- [Frontend Hostnames](#frontend-hostnames)
- [Database](#database)
- [Redis](#redis)
- [Email (SMTP)](#email-smtp)
- [Secret Management](#secret-management)
- [JWT Configuration](#jwt-configuration)
- [OpenTelemetry (Tracing)](#opentelemetry-tracing)

---

## Application

Controls the API versioning and the public/private base URLs that the service advertises to clients and internal callers.

| Variable | Required | Default | Description |
|---|---|---|---|
| `APP_VERSION` | ✅ | `v1` | API version prefix used in every route path (e.g. `/v1/…`). |
| `APP_PUBLIC_HOSTNAME` | ✅ | — | Fully-qualified base URL of the **public** REST API, reachable from the internet or the frontend. |
| `APP_PRIVATE_HOSTNAME` | ✅ | — | Fully-qualified base URL of the **internal** REST API, reachable only within the private network / service mesh. |

**Example**

```env
APP_VERSION="v1"
APP_PUBLIC_HOSTNAME="http://localhost:8081"
APP_PRIVATE_HOSTNAME="http://localhost:8080"
```

> For Docker Compose local development, use the service names defined in `docker-compose.yml` as hostnames:
> ```env
> APP_PUBLIC_HOSTNAME="http://public.api.maintainerd.auth"
> APP_PRIVATE_HOSTNAME="http://private.api.maintainerd.auth"
> ```

---

## Frontend Hostnames

Hostnames of the frontend applications that consume this API.  
Used internally for CORS policies, redirect URIs, and email link generation.

| Variable | Required | Default | Description |
|---|---|---|---|
| `ACCOUNT_HOSTNAME` | ✅ | — | Base URL of the **Account** portal (profile management, billing). |
| `AUTH_HOSTNAME` | ✅ | — | Base URL of the **Auth** portal (login, registration, password reset). |

**Example**

```env
ACCOUNT_HOSTNAME="http://localhost:3001"
AUTH_HOSTNAME="http://localhost:3000"
```

> For Docker Compose:
> ```env
> ACCOUNT_HOSTNAME="http://account.maintainerd.local"
> AUTH_HOSTNAME="http://auth.maintainerd.local"
> ```

---

## Database

PostgreSQL connection settings.

| Variable | Required | Default | Description |
|---|---|---|---|
| `DB_HOST` | ✅ | `localhost` | Hostname or IP of the PostgreSQL server. |
| `DB_PORT` | ✅ | `5432` | TCP port PostgreSQL listens on. |
| `DB_USER` | ✅ | — | Database username. |
| `DB_PASSWORD` | ✅ | — | Password for `DB_USER`. Use a strong, randomly generated password in production. |
| `DB_NAME` | ✅ | `maintainerd` | Name of the database. |
| `DB_SSLMODE` | ✅ | `disable` | PostgreSQL SSL mode. Set to `require` or `verify-full` in production. |
| `DB_TABLE_PREFIX` | ❌ | `md_` | Optional prefix prepended to every table name. Useful when sharing a schema with other services. |

**Example (local Docker)**

```env
DB_HOST="localhost"
DB_PORT="5432"
DB_USER="devuser"
DB_PASSWORD="change-me-locally"
DB_NAME="maintainerd"
DB_SSLMODE="disable"
DB_TABLE_PREFIX="md_"
```

**Generating a secure password**

```bash
openssl rand -base64 32
```

---

## Redis

Redis is used for distributed rate-limiting, session caching, and pub/sub.

| Variable | Required | Default | Description |
|---|---|---|---|
| `REDIS_CONNECTION_STRING` | ✅ | — | Full Redis URL. Takes precedence over the individual fields below when set. Format: `redis://[:password@]host:port[/db]`. |
| `REDIS_HOST` | ✅ | `localhost` | Redis server hostname (used when `REDIS_CONNECTION_STRING` is not set). |
| `REDIS_PORT` | ✅ | `6379` | Redis port. |
| `REDIS_PASSWORD` | ❌ | — | Redis `AUTH` password. Leave empty if Redis has no password (development only). |

**Example (local Docker)**

```env
REDIS_HOST="localhost"
REDIS_PORT="6379"
REDIS_PASSWORD="change-me-locally"
REDIS_CONNECTION_STRING="redis://:change-me-locally@localhost:6379"
```

> `REDIS_CONNECTION_STRING` must URL-encode special characters in the password.  
> For example, `@` becomes `%40`, `#` becomes `%23`.
>
> ```bash
> # Quick encoder
> python3 -c "import urllib.parse; print(urllib.parse.quote('your-password', safe=''))"
> ```

---

## Email (SMTP)

Outgoing email settings for transactional mail (invitations, password resets, verification).

| Variable | Required | Default | Description |
|---|---|---|---|
| `SMTP_HOST` | ✅ | — | SMTP server hostname. |
| `SMTP_PORT` | ✅ | `587` | SMTP port. Use `587` for STARTTLS (recommended) or `465` for implicit TLS. |
| `SMTP_USER` | ✅ | — | SMTP authentication username (usually the sender email address). |
| `SMTP_PASS` | ✅ | — | SMTP authentication password or app-specific password. **See below for generation steps.** |
| `SMTP_FROM_EMAIL` | ✅ | — | The `From` email address shown to recipients. |
| `SMTP_FROM_NAME` | ✅ | `Maintainerd` | The `From` display name shown to recipients. |
| `EMAIL_LOGO_URL` | ❌ | — | Publicly accessible URL of the logo image embedded in HTML email templates. |

**Example**

```env
SMTP_HOST="smtp.example.com"
SMTP_PORT="587"
SMTP_USER="noreply@example.com"
SMTP_PASS="your-smtp-password"
SMTP_FROM_EMAIL="noreply@example.com"
SMTP_FROM_NAME="Maintainerd"
EMAIL_LOGO_URL="https://example.com/logo.png"
```

**Generating a Gmail App Password**

If you are using Gmail as your SMTP provider you must use an **App Password**, not your account password.

1. Enable **2-Step Verification** on your Google account → <https://myaccount.google.com/security>
2. Go to **Manage your Google Account** → **Security** → **2-Step Verification** → **App passwords**
3. Choose **Mail** as the app and your device type, then click **Generate**.
4. Copy the 16-character password (spaces are cosmetic; omit them) and set it as `SMTP_PASS`.

> Other providers (SendGrid, Postmark, Resend, Mailgun) expose SMTP credentials through their dashboards.  
> Using a dedicated transactional email service is strongly recommended for production.

---

## Secret Management

Maintainerd Auth supports pluggable secret backends so sensitive values (JWT keys, database passwords) can be stored outside of environment variables in production.

### Core Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `SECRET_PROVIDER` | ✅ | `env` | Secret backend to use. One of: `env`, `file`, `aws_secrets`, `aws_ssm`, `vault`, `gcp`, `azure_kv`. |
| `SECRET_PREFIX` | ❌ | `maintainerd/auth` | Namespace prefix for secrets in external providers. Not used by `env`, `file`, or `gcp`. |

### Provider-Specific Variables

#### `env` — Environment Variables (default)

No additional variables. Secrets are read directly from environment variables.
Supports `base64:` prefix for binary secrets (e.g. `JWT_PRIVATE_KEY=base64:LS0tLS1C…`).

#### `file` — File-Based Secrets (Docker / Kubernetes)

| Variable | Required | Default | Description |
|---|---|---|---|
| `SECRET_FILE_PATH` | ❌ | `/run/secrets` | Base directory where secret files are mounted. |

Key names are lowercased with underscores replaced by hyphens.
Example: `JWT_PRIVATE_KEY` → `<SECRET_FILE_PATH>/jwt-private-key`

#### `aws_secrets` — AWS Secrets Manager

| Variable | Required | Default | Description |
|---|---|---|---|
| `AWS_REGION` | ✅ | `us-east-1` | AWS region where secrets are stored. |
| `AWS_ACCESS_KEY_ID` | ❌ | — | AWS access key. Prefer IAM roles over static credentials. |
| `AWS_SECRET_ACCESS_KEY` | ❌ | — | AWS secret key. |

Secret naming: `<SECRET_PREFIX>/<key-lowercased-hyphens>`
Example: `JWT_PRIVATE_KEY` → `maintainerd/auth/jwt-private-key`

#### `aws_ssm` — AWS SSM Parameter Store

| Variable | Required | Default | Description |
|---|---|---|---|
| `AWS_REGION` | ✅ | `us-east-1` | AWS region. |
| `AWS_ACCESS_KEY_ID` | ❌ | — | AWS access key. Prefer IAM roles. |
| `AWS_SECRET_ACCESS_KEY` | ❌ | — | AWS secret key. |

Parameter naming: `/<SECRET_PREFIX>/<key-lowercased-hyphens>`
Example: `JWT_PRIVATE_KEY` → `/maintainerd/auth/jwt-private-key`
SecureString parameters are automatically decrypted.

#### `vault` — HashiCorp Vault (KV v2)

| Variable | Required | Default | Description |
|---|---|---|---|
| `VAULT_ADDR` | ❌ | `http://localhost:8200` | Vault server address. |
| `VAULT_TOKEN` | ❌ | — | Static token. Set this **or** use AppRole below. |
| `VAULT_MOUNT` | ❌ | `secret` | KV v2 mount path. |
| `VAULT_ROLE_ID` | ❌ | — | AppRole role ID (used when `VAULT_TOKEN` is empty). |
| `VAULT_SECRET_ID` | ❌ | — | AppRole secret ID (used when `VAULT_TOKEN` is empty). |
| `VAULT_SECRET_FIELD` | ❌ | `value` | Field name within the KV secret that holds the value. |

Secret path: `<VAULT_MOUNT>/data/<SECRET_PREFIX>/<key-lowercased-hyphens>`
Example: `JWT_PRIVATE_KEY` → `secret/data/maintainerd/auth/jwt-private-key`

Each secret must have a field (default: `value`) containing the actual secret data:
```bash
vault kv put secret/maintainerd/auth/jwt-private-key value=@jwt_private.pem
```

#### `gcp` — GCP Secret Manager

| Variable | Required | Default | Description |
|---|---|---|---|
| `GCP_PROJECT_ID` | ✅ | — | GCP project ID. |

Authentication uses **Application Default Credentials (ADC)**:
- **GKE / Cloud Run**: Workload Identity is used automatically.
- **Local development**: Run `gcloud auth application-default login`.

Secret naming: `projects/<GCP_PROJECT_ID>/secrets/<key-lowercased-hyphens>/versions/latest`
Example: `JWT_PRIVATE_KEY` → `projects/my-project/secrets/jwt-private-key/versions/latest`

> `SECRET_PREFIX` is not used by the GCP provider. Use IAM policies to scope access.

#### `azure_kv` — Azure Key Vault

| Variable | Required | Default | Description |
|---|---|---|---|
| `AZURE_KEYVAULT_URL` | ✅ | — | Key Vault endpoint, e.g. `https://my-vault.vault.azure.net`. |
| `AZURE_TENANT_ID` | ❌ | — | Azure AD tenant ID (for service principal auth). |
| `AZURE_CLIENT_ID` | ❌ | — | Service principal client ID. |
| `AZURE_CLIENT_SECRET` | ❌ | — | Service principal client secret. |

Authentication uses **DefaultAzureCredential**, which tries in order:
1. Environment variables (`AZURE_TENANT_ID` + `AZURE_CLIENT_ID` + `AZURE_CLIENT_SECRET`)
2. Workload Identity (AKS)
3. Managed Identity
4. Azure CLI (local development)

Secret naming: `<key-lowercased-hyphens>`
Example: `JWT_PRIVATE_KEY` → `jwt-private-key`

> Azure Key Vault names only allow lowercase letters, numbers, and hyphens.

### Provider Quick-Reference

| Environment | Recommended `SECRET_PROVIDER` |
|---|---|
| Local development | `env` |
| Docker Compose / Docker Swarm | `file` (Docker Secrets) |
| Kubernetes | `file` (Kubernetes Secrets) |
| AWS ECS / Lambda | `aws_secrets` or `aws_ssm` |
| GCP GKE / Cloud Run | `gcp` |
| Azure AKS / App Service | `azure_kv` |
| Self-hosted / on-prem | `vault` |

**Example (local)**

```env
SECRET_PROVIDER=env
SECRET_PREFIX=maintainerd/auth
```

**Example (AWS)**

```env
SECRET_PROVIDER=aws_secrets
SECRET_PREFIX=maintainerd/auth
AWS_REGION=us-east-1
# Prefer an IAM role attached to the task/instance over static keys
# AWS_ACCESS_KEY_ID=...
# AWS_SECRET_ACCESS_KEY=...
```

**Example (Vault with AppRole)**

```env
SECRET_PROVIDER=vault
SECRET_PREFIX=maintainerd/auth
VAULT_ADDR=http://localhost:8200
VAULT_MOUNT=secret
VAULT_ROLE_ID=your-role-id
VAULT_SECRET_ID=your-secret-id
```

**Example (GCP)**

```env
SECRET_PROVIDER=gcp
GCP_PROJECT_ID=my-project-id
```

**Example (Azure)**

```env
SECRET_PROVIDER=azure_kv
AZURE_KEYVAULT_URL=https://my-vault.vault.azure.net
```

---

## JWT Configuration

RSA key pair used to sign and verify JSON Web Tokens.  
The private key signs tokens; the public key verifies them.

| Variable | Required | Description |
|---|---|---|
| `JWT_PRIVATE_KEY` | ✅ | PEM-encoded RSA private key. Newlines must be escaped as `\n` when stored inline. |
| `JWT_PUBLIC_KEY` | ✅ | PEM-encoded RSA public key. Same escaping rule applies. |

> ⚠️ **Never share or commit your private key.** It grants the ability to mint arbitrary tokens for your system.

### Generating a key pair

Use the included script to generate a production-quality RSA-4096 key pair:

```bash
# Default: 4096-bit key, output to ./keys/
./scripts/generate-jwt-keys.sh

# Custom key size and output directory
./scripts/generate-jwt-keys.sh 4096 /tmp/jwt-keys
```

The script produces:

| File | Purpose |
|---|---|
| `jwt_private.pem` | RSA private key (permissions: `600`) |
| `jwt_public.pem` | RSA public key (permissions: `644`) |
| `jwt_env_vars.txt` | Ready-to-paste `.env` lines with `\n`-escaped keys |
| `key_fingerprints.txt` | SHA-256 fingerprints for verification |

Copy the contents of `jwt_env_vars.txt` directly into your `.env` file:

```bash
cat keys/jwt_env_vars.txt >> .env
```

### Generating manually with OpenSSL

```bash
# 1. Generate private key (RSA 4096-bit)
openssl genrsa -out jwt_private.pem 4096

# 2. Derive public key
openssl rsa -in jwt_private.pem -pubout -out jwt_public.pem

# 3. Format for .env (escapes newlines to \n)
echo -n 'JWT_PRIVATE_KEY="' && \
  awk 'NF {sub(/\r/, ""); printf "%s\\n",$0;}' jwt_private.pem && \
  echo '"'

echo -n 'JWT_PUBLIC_KEY="' && \
  awk 'NF {sub(/\r/, ""); printf "%s\\n",$0;}' jwt_public.pem && \
  echo '"'
```

### Storing keys in production

Inline PEM in environment variables is acceptable for local development.  
In production, use one of the following approaches instead:

| Approach | How |
|---|---|
| **AWS Secrets Manager** | Store the raw PEM as a secret value; set `SECRET_PROVIDER=aws_secrets` |
| **AWS SSM Parameter Store** | Store as `SecureString`; set `SECRET_PROVIDER=aws_ssm` |
| **HashiCorp Vault** | Store under `SECRET_PREFIX`; set `SECRET_PROVIDER=vault` |
| **GCP Secret Manager** | Create a secret with the PEM contents; set `SECRET_PROVIDER=gcp` |
| **Azure Key Vault** | Store the PEM as a secret; set `SECRET_PROVIDER=azure_kv` |
| **Docker / Kubernetes Secrets** | Mount as files; set `SECRET_PROVIDER=file` and `SECRET_FILE_PATH` |
| **Base64 inline** | Prefix the value with `base64:` — e.g. `JWT_PRIVATE_KEY=base64:LS0tLS1C…` |

### Key rotation

- Rotate JWT keys at least every **90 days** in production.
- During rotation, keep the old public key active until all tokens signed with it have expired.
- Update `JWT_PUBLIC_KEY` to the new public key and deploy; then remove the old key.

---

## OpenTelemetry (Tracing)

Maintainerd Auth has built-in [OpenTelemetry](https://opentelemetry.io/) tracing. When enabled, the service exports distributed traces covering HTTP requests, gRPC calls, database queries, Redis commands, and outgoing SMTP email sends.

Tracing is **disabled by default**. Set `OTEL_ENABLED=true` to activate it.

| Variable | Required | Default | Description |
|---|---|---|---|
| `OTEL_ENABLED` | ❌ | `false` | Set to `true` to enable tracing. When `false`, a no-op tracer is installed (zero overhead). |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | ❌ | `localhost:4317` | gRPC endpoint of the OpenTelemetry Collector (or compatible backend like Jaeger, Tempo). |
| `OTEL_SERVICE_NAME` | ❌ | `maintainerd-auth` | Service name attached to every span. |

> All standard `OTEL_*` environment variables defined by the [OpenTelemetry SDK specification](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/) are supported automatically (e.g. `OTEL_EXPORTER_OTLP_HEADERS`, `OTEL_EXPORTER_OTLP_INSECURE`, `OTEL_TRACES_SAMPLER`).

### What is instrumented

| Layer | Instrumentation | Details |
|---|---|---|
| HTTP (REST) | Automatic | Every inbound request gets a span with method, route, status code. |
| gRPC | Automatic | Every inbound RPC gets a span via `otelgrpc`. |
| PostgreSQL | Automatic | Every query/transaction gets a span via `otelgorm`. |
| Redis | Automatic | Every Redis command gets a span via `redisotel`. |
| SMTP (email) | Explicit | Outgoing email sends are wrapped in a span with host, port, recipient, and subject attributes. |
| Logs | Correlation | `trace_id` and `span_id` are automatically injected into structured JSON log output. |

### Local development with Jaeger

The easiest way to view traces locally is with [Jaeger](https://www.jaegertracing.io/):

```bash
# Start Jaeger all-in-one (receives OTLP on port 4317)
docker run -d --name jaeger \
  -p 4317:4317 \
  -p 16686:16686 \
  jaegertracing/all-in-one:latest
```

Then set these in your `.env`:

```env
OTEL_ENABLED="true"
OTEL_EXPORTER_OTLP_ENDPOINT="localhost:4317"
OTEL_SERVICE_NAME="maintainerd-auth"
```

Open <http://localhost:16686> to browse traces.


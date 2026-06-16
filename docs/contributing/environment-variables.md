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
APP_FRONTEND_IDENTITY_HOSTNAME="http://account.maintainerd.local"
APP_FRONTEND_CONSOLE_HOSTNAME="http://auth.maintainerd.local"

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
# SECRET MANAGEMENT
# =============================================================================
# EMAIL and SMS credentials are now per-tenant via admin API (email_config / sms_config tables)
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
- [SMS](#sms)
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
| `MANAGEMENT_PORT` | ❌ | `8082` | Dedicated operations listener for `/metrics`, `/readyz`, `/livez`, and `/openapi.json`. |

**Example**

```env
APP_VERSION="v1"
APP_PUBLIC_HOSTNAME="http://localhost:8081"
APP_PRIVATE_HOSTNAME="http://localhost:8080"
MANAGEMENT_PORT="8082"
```

> For Docker Compose local development, use the service names defined in `docker-compose.yml` as hostnames:
> ```env
> APP_PUBLIC_HOSTNAME="http://public.api.maintainerd.auth"
> APP_PRIVATE_HOSTNAME="http://private.api.maintainerd.auth"
> MANAGEMENT_PORT="8082"
> ```

---

## Frontend Hostnames

Hostnames of the frontend applications that consume this API.  
Used internally for CORS policies, redirect URIs, and email link generation.

| Variable | Required | Default | Description |
|---|---|---|---|
| `APP_FRONTEND_IDENTITY_HOSTNAME` | ✅ | — | Base URL of the **Account** portal (profile management, billing). |
| `APP_FRONTEND_CONSOLE_HOSTNAME` | ✅ | — | Base URL of the **Auth** portal (login, registration, password reset). |

**Example**

```env
APP_FRONTEND_IDENTITY_HOSTNAME="http://localhost:3001"
APP_FRONTEND_CONSOLE_HOSTNAME="http://localhost:3000"
```

> For Docker Compose:
> ```env
> APP_FRONTEND_IDENTITY_HOSTNAME="http://account.maintainerd.local"
> APP_FRONTEND_CONSOLE_HOSTNAME="http://auth.maintainerd.local"
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

## Email & SMS Delivery

Email and SMS credentials are now managed per-tenant through the admin API.
See [environment-variables.md](../documentations/environment-variables/environment-variables.md#email--sms-delivery) for details.

- **Email**: Configured per-tenant via `GET/PUT /api/v1/email-config` → stored in `email_config` table.
- **SMS**: Configured per-tenant via `GET/PUT /api/v1/sms-config` → stored in `sms_config` table.
- **SMS daily budget**: Per-tenant via `sms_config.daily_send_limit` (default: `1000`).

Credentials are encrypted at rest using `APP_ENCRYPTION_KEY`.

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

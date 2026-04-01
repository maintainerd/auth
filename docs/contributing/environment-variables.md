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
# SECRET_FILE_PATH=/run/secrets
# AWS_REGION=us-east-1
# AWS_ACCESS_KEY_ID=
# AWS_SECRET_ACCESS_KEY=
# VAULT_ADDR=
# VAULT_TOKEN=

# =============================================================================
# JWT  ← generate your own keys, see: #generating-a-key-pair
# Run: ./scripts/generate-jwt-keys.sh  then paste the output of keys/jwt_env_vars.txt here
# =============================================================================
JWT_PRIVATE_KEY=""
JWT_PUBLIC_KEY=""
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

| Variable | Required | Default | Description |
|---|---|---|---|
| `SECRET_PROVIDER` | ✅ | `env` | Secret backend to use. One of: `env`, `file`, `aws_ssm`, `aws_secrets`, `vault`. |
| `SECRET_PREFIX` | ❌ | `maintainerd/auth` | Namespace prefix for secrets in external providers. Not used when `SECRET_PROVIDER=env`. |
| `SECRET_FILE_PATH` | ❌ | `/run/secrets` | Base directory for file-based secrets (Docker Secrets, Kubernetes Secrets). Only used when `SECRET_PROVIDER=file`. |
| `AWS_REGION` | ❌ | — | AWS region. Required for `aws_ssm` and `aws_secrets` providers. |
| `AWS_ACCESS_KEY_ID` | ❌ | — | AWS access key. Prefer IAM roles over static credentials in production. |
| `AWS_SECRET_ACCESS_KEY` | ❌ | — | AWS secret key. |
| `VAULT_ADDR` | ❌ | — | HashiCorp Vault server address (e.g. `https://vault.company.com`). Required for `vault` provider. |
| `VAULT_TOKEN` | ❌ | — | Vault authentication token. Prefer AppRole or Kubernetes auth in production. |

**Provider quick-reference**

| Environment | Recommended `SECRET_PROVIDER` |
|---|---|
| Local development | `env` |
| Docker Compose / Docker Swarm | `file` (Docker Secrets) |
| Kubernetes | `file` (Kubernetes Secrets) |
| AWS ECS / Lambda | `aws_secrets` or `aws_ssm` |
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
| **Docker / Kubernetes Secrets** | Mount as files; set `SECRET_PROVIDER=file` and `SECRET_FILE_PATH` |
| **Base64 inline** | Prefix the value with `base64:` — e.g. `JWT_PRIVATE_KEY=base64:LS0tLS1C…` |

### Key rotation

- Rotate JWT keys at least every **90 days** in production.
- During rotation, keep the old public key active until all tokens signed with it have expired.
- Update `JWT_PUBLIC_KEY` to the new public key and deploy; then remove the old key.


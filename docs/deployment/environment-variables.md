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
SECRET_PROVIDER=aws_secrets       # ← set to your provider: aws_secrets | aws_ssm | vault | file
SECRET_PREFIX=maintainerd/auth
# SECRET_FILE_PATH=/run/secrets
# AWS_REGION=us-east-1            # ← required for aws_secrets / aws_ssm
# AWS_ACCESS_KEY_ID=              # ← prefer IAM roles over static keys
# AWS_SECRET_ACCESS_KEY=
# VAULT_ADDR=https://vault.yourdomain.com # ← required for vault
# VAULT_TOKEN=                            # ← prefer AppRole auth over static tokens

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

| Variable | Required | Default | Description |
|---|---|---|---|
| `SECRET_PROVIDER` | ✅ | `env` | Secret backend. Use `env` only for local dev. In production use `aws_secrets`, `aws_ssm`, `vault`, or `file`. |
| `SECRET_PREFIX` | ❌ | `maintainerd/auth` | Namespace prefix for secrets in external providers. |
| `SECRET_FILE_PATH` | ❌ | `/run/secrets` | Base path for file-based secrets. Required when `SECRET_PROVIDER=file`. |
| `AWS_REGION` | ❌ | — | AWS region. Required for `aws_ssm` and `aws_secrets`. Prefer IAM roles over static keys. |
| `AWS_ACCESS_KEY_ID` | ❌ | — | AWS access key. Only if IAM roles are unavailable. |
| `AWS_SECRET_ACCESS_KEY` | ❌ | — | AWS secret key. Only if IAM roles are unavailable. |
| `VAULT_ADDR` | ❌ | — | HashiCorp Vault address. Required for `vault` provider. |
| `VAULT_TOKEN` | ❌ | — | Vault token. Prefer AppRole or Kubernetes auth over static tokens. |

**Provider recommendations by platform:**

| Platform | `SECRET_PROVIDER` |
|---|---|
| AWS ECS / Lambda | `aws_secrets` or `aws_ssm` |
| Kubernetes | `file` (Kubernetes Secrets) |
| Docker Swarm | `file` (Docker Secrets) |
| Self-hosted / bare metal | `vault` |

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
vault kv put secret/maintainerd/auth \
  jwt_private_key=@/tmp/jwt-keys/jwt_private.pem \
  jwt_public_key=@/tmp/jwt-keys/jwt_public.pem
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
- [ ] Database password is at least 32 characters, randomly generated
- [ ] Redis password is set and strong
- [ ] `JWT_PRIVATE_KEY` permissions are `600` on any filesystem where it is stored
- [ ] No `.env` files are present on the production host
- [ ] Key rotation schedule is documented and owned by a team member


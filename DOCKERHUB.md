# maintainerd-auth

**Open-source, self-hostable OAuth 2.0 / OpenID Connect provider + identity broker — in a single container.**

One image bundles the Go backend, the admin console, and the hosted login UI (compiled in via `go:embed`, no nginx, no sidecars). Bring a **PostgreSQL** and a **Redis**, and you have a full identity & access platform: authentication, MFA, social/enterprise/SAML federation, multi-tenancy, and fine-grained authorization.

- 📦 **Source & full docs:** https://github.com/maintainerd/maintainerd-auth
- 🔑 **Every environment variable:** https://github.com/maintainerd/maintainerd-auth/blob/main/docs/contributing/environment-variables.md
- 🐛 **Issues / questions:** https://github.com/maintainerd/maintainerd-auth/issues
- 📜 **License:** Apache-2.0

---

## Supported tags

| Tag | Meaning |
|-----|---------|
| `0.1.0` | Exact release (recommended — pin this) |
| `0.1` | Latest patch of the `0.1` line |
| `latest` | Most recent stable release |

**Architectures:** `linux/amd64`, `linux/arm64`. Each image carries SLSA provenance + an SBOM attestation.

---

## What's inside

A single process serves four surfaces, each on its own port:

| Port | Surface | Expose publicly? |
|------|---------|------------------|
| `3000` | Admin console | ✅ operators |
| `3001` | Hosted login / identity UI | ✅ end users |
| `8081` | Data plane — OAuth2/OIDC issuer + public API | ✅ where your issuer must resolve |
| `8080` | Control plane — management API | ❌ keep internal |
| `8082` | Health checks + Prometheus `/metrics` | ❌ keep internal |

You provide PostgreSQL and Redis; they are **not** in this image.

---

## Quick start

The app reads its config from a `/.env` file at startup, so both methods below mount one in. First, generate your keys and secrets:

```bash
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out jwt_private.pem 2>/dev/null
openssl pkey -in jwt_private.pem -pubout -out jwt_public.pem 2>/dev/null

cat > .env <<EOF
APP_ENV=development
APP_PUBLIC_HOSTNAME=http://localhost:8081
APP_PRIVATE_HOSTNAME=http://localhost:8080
APP_FRONTEND_CONSOLE_HOSTNAME=http://localhost:3000
APP_FRONTEND_IDENTITY_HOSTNAME=http://localhost:3001
COOKIE_SECURE=false
DB_HOST=postgres
DB_PORT=5432
DB_USER=maintainerd
DB_PASSWORD=change-me
DB_NAME=maintainerd
REDIS_ADDR=redis:6379
APP_ENCRYPTION_KEY=base64:$(openssl rand -base64 32)
HMAC_SECRET_KEY=base64:$(openssl rand -base64 32)
JWT_PRIVATE_KEY="$(awk '{printf "%s\\n", $0}' jwt_private.pem)"
JWT_PUBLIC_KEY="$(awk '{printf "%s\\n", $0}' jwt_public.pem)"
EOF
```

### Option A — docker compose (recommended: includes Postgres + Redis)

```yaml
services:
  postgres:
    image: postgres:17-alpine
    environment:
      POSTGRES_USER: maintainerd
      POSTGRES_PASSWORD: change-me
      POSTGRES_DB: maintainerd
    volumes: [pgdata:/var/lib/postgresql/data]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U maintainerd"]
      interval: 5s
      retries: 10

  redis:
    image: redis:7-alpine

  auth:
    image: xreyc/maintainerd-auth:0.1.0   # or :latest
    depends_on:
      postgres: { condition: service_healthy }
      redis: { condition: service_started }
    ports:
      - "3000:3000"   # admin console
      - "3001:3001"   # hosted login
      - "8081:8081"   # OAuth2/OIDC issuer + public API
    volumes:
      - ./.env:/.env:ro

volumes:
  pgdata:
```

```bash
docker compose up -d
```

### Option B — docker run (you supply Postgres + Redis)

Point `DB_HOST` / `REDIS_ADDR` in your `.env` at reachable instances, then:

```bash
docker run -d --name maintainerd-auth \
  -p 3000:3000 -p 3001:3001 -p 8081:8081 \
  -v "$PWD/.env:/.env:ro" \
  xreyc/maintainerd-auth:0.1.0
```

> Mount the `.env` (don't use `--env-file`): the app un-escapes the multi-line PEM keys when it reads the file itself.

### Finish setup

Open **http://localhost:3000** and complete the setup wizard to create your first tenant and admin. Your OIDC discovery document is at **http://localhost:8081/.well-known/openid-configuration**.

---

## Environment variables

### Required — the app won't start without these

| Variable | Description |
|----------|-------------|
| `APP_PUBLIC_HOSTNAME` | Public base URL — this is your **OIDC issuer** (`iss`) and must match discovery. |
| `APP_PRIVATE_HOSTNAME` | Control-plane (management) base URL. |
| `APP_FRONTEND_CONSOLE_HOSTNAME` | Admin console URL. |
| `APP_FRONTEND_IDENTITY_HOSTNAME` | Hosted login URL. |
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` | PostgreSQL connection. |
| `JWT_PRIVATE_KEY` / `JWT_PUBLIC_KEY` | RS256 signing keypair (PEM). |
| `APP_ENCRYPTION_KEY` | AES-256 key for encryption-at-rest — **exactly 32 bytes** (a `base64:`-prefixed value is decoded first). |
| `HMAC_SECRET_KEY` | HMAC key for signed URLs (accepts `base64:`). |

### Common options

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | `development` | Set `production` for stricter runtime checks. |
| `REDIS_ADDR` | `redis-db:6379` | Redis `host:port` (Redis is required infrastructure). |
| `REDIS_PASSWORD` / `REDIS_TLS` | — / `false` | Redis auth / TLS. |
| `DB_SSLMODE` | `disable` | Set `require` in production (`disable` is rejected when `APP_ENV=production`). |
| `COOKIE_SECURE` | `true` | Set `false` only for local HTTP. |
| `COOKIE_SAMESITE` | `lax` | Keep `lax` — needed for federated SSO redirects. |
| `CORS_ALLOWED_ORIGINS` | — | Extra allowed origins (comma-separated). |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error`. |
| `MANAGEMENT_PORT` | `8082` | Health + `/metrics` port. |

> **Secrets** (`JWT_*`, `APP_ENCRYPTION_KEY`, `HMAC_SECRET_KEY`, `DB_PASSWORD`) accept a `base64:` prefix and can be sourced from **AWS Secrets Manager / SSM, HashiCorp Vault, Azure Key Vault, GCP Secret Manager, or mounted files** instead of env — set `SECRET_PROVIDER`.
>
> 👉 **The complete list — every variable, default, and production note — is in the [Environment Variables reference](https://github.com/maintainerd/maintainerd-auth/blob/main/docs/contributing/environment-variables.md).**

---

## Production notes

- Terminate TLS in front of the container; set real HTTPS hostnames, `COOKIE_SECURE=true`, and `DB_SSLMODE=require`.
- Keep `8080` (control plane) and `8082` (metrics) on an internal network.
- Use a managed secret provider rather than plaintext env in production.
- Health endpoints: `GET /readyz` and `/livez` on `8081`/`8080`/`8082`.

---

<sub>Built by <a href="https://github.com/xreyc">Reyco Seguma (@xreyc)</a> and the Maintainerd community · Apache-2.0</sub>

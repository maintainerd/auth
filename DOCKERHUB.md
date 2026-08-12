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
| `latest` | Current build — used by the quick start |
| `0.1.0` | The pre-release version (moving during testing; pin `latest` for the newest) |

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

Run locally behind nginx with clean HTTPS hostnames (no ports), using this image + PostgreSQL + Redis. Full walkthrough: the [repo README](https://github.com/maintainerd/maintainerd-auth#quick-start).

Download these into one folder — [`docker-compose.yml`](https://raw.githubusercontent.com/maintainerd/maintainerd-auth/main/examples/quickstart/docker-compose.yml), [`.env.example`](https://raw.githubusercontent.com/maintainerd/maintainerd-auth/main/examples/quickstart/.env.example), [`nginx.conf`](https://raw.githubusercontent.com/maintainerd/maintainerd-auth/main/examples/quickstart/nginx.conf), [`setup.sh`](https://raw.githubusercontent.com/maintainerd/maintainerd-auth/main/examples/quickstart/setup.sh) — then:

```bash
cp .env.example .env
chmod +x setup.sh && ./setup.sh          # generates your keys + a local TLS cert

sudo tee -a /etc/hosts >/dev/null <<'EOF'
127.0.0.1 console.auth.maintainerd.local identity.auth.maintainerd.local console-api.auth.maintainerd.local identity-api.auth.maintainerd.local
EOF

docker compose up -d
```

**First run 👉** open **https://console.auth.maintainerd.local/setup/tenant** and create your first **tenant** and **admin** (accept the one-time self-signed-cert warning).

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

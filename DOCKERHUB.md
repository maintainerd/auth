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

Ready-made sample files (compose + env + key generator) live in the repo. Download them, generate your own keys, and run — full walkthrough in the [repo README](https://github.com/maintainerd/maintainerd-auth#quick-start).

```bash
mkdir maintainerd-auth && cd maintainerd-auth
base=https://raw.githubusercontent.com/maintainerd/maintainerd-auth/main/examples/quickstart
curl -O "$base/.env.example" -O "$base/docker-compose.yml" -O "$base/generate-secrets.sh"
cp .env.example .env

# generate YOUR OWN keys/secrets into .env (runs openssl locally)
chmod +x generate-secrets.sh && ./generate-secrets.sh

# map the local hostnames (one time)
sudo tee -a /etc/hosts >/dev/null <<'EOF'
127.0.0.1 console.auth.maintainerd.local identity.auth.maintainerd.local console-api.auth.maintainerd.local identity-api.auth.maintainerd.local
EOF

docker compose up -d
```

**First run 👉** open **http://console.auth.maintainerd.local:3000** and complete the setup wizard to create your first **tenant** and **admin**. OIDC discovery: **http://identity-api.auth.maintainerd.local:8081/.well-known/openid-configuration**.

> Prefer `docker run`, or a split-host / production layout? See the [full quick-start guide](https://github.com/maintainerd/maintainerd-auth/tree/main/examples/quickstart).

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

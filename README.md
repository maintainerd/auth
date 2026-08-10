<div align="left">
  <img src="https://github.com/user-attachments/assets/8ecfd8bd-e8df-4fe5-a291-bd6192c23a5d" alt="Maintainerd Auth" height="70">
</div>

<br clear="left">

[![Release](https://img.shields.io/github/v/release/maintainerd/maintainerd-auth?logo=github&label=release)](https://github.com/maintainerd/maintainerd-auth/releases/latest)
[![Docker Hub](https://img.shields.io/docker/v/xreyc/maintainerd-auth?logo=docker&label=docker%20hub&sort=semver)](https://hub.docker.com/r/xreyc/maintainerd-auth)
[![CI](https://github.com/maintainerd/maintainerd-auth/actions/workflows/ci.yml/badge.svg)](https://github.com/maintainerd/maintainerd-auth/actions/workflows/ci.yml)
[![Security](https://github.com/maintainerd/maintainerd-auth/actions/workflows/security.yml/badge.svg)](https://github.com/maintainerd/maintainerd-auth/actions/workflows/security.yml)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/maintainerd/maintainerd-auth/badge)](https://scorecard.dev/viewer/?uri=github.com/maintainerd/maintainerd-auth)
[![Coverage](https://codecov.io/gh/maintainerd/maintainerd-auth/graph/badge.svg)](https://codecov.io/gh/maintainerd/maintainerd-auth)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)

**maintainerd-auth** is an open-source, self-hostable identity and access platform — a single container that delivers authentication, federation, and fine-grained authorization for your applications, services, and APIs.

It ships as **one all-in-one image**: the Go backend plus the admin console and hosted login UI, compiled into a single binary (no nginx, no sidecars). Bring your own PostgreSQL and Redis, and you have a full OAuth 2.0 / OpenID Connect provider and identity broker.

---

## Features

- **Full OAuth 2.0 + OIDC** — authorization code (PKCE), client credentials, device, token exchange, PAR, CIBA, dynamic client registration, and DPoP
- **JWT (RS256)** with multi-key JWKS and automatic key rotation
- **Multi-factor authentication** — TOTP, WebAuthn/passkeys, SMS OTP, backup codes, and step-up auth
- **Federation** — broker sign-in over **OIDC, OAuth 2.0, and SAML 2.0**: Google, Microsoft, GitHub, GitLab, LinkedIn, Facebook, X (Twitter), Auth0, Cognito, any standards-compliant IdP, and maintainerd-to-maintainerd. Includes JIT provisioning, identity linking, and home-realm discovery
- **Fine-grained access control** — RBAC with granular permissions, plus IAM services, APIs, policies, service-token policy bundles, and service-to-service authorization
- **Multi-tenant** — full tenant isolation, per-tenant configuration, API keys, and invite flows
- **Session management** — refresh-token rotation, family revocation, reuse detection, and concurrent-session limits
- **Webhook delivery** — auth-event notifications signed with HMAC-SHA256, with replay protection
- **Audit logging** — structured auth events with retention, per-tenant isolation, and PII redaction
- **Pluggable secret management** — env vars, AWS Secrets Manager / SSM, HashiCorp Vault, Azure Key Vault, GCP Secret Manager, or mounted files
- **Pluggable email delivery** — SMTP, SES, SendGrid, Postmark, Mailgun, Resend (configured per tenant)
- **OpenTelemetry** — traces, metrics, and a Prometheus endpoint

---

## Quick Start

Run the released image with a PostgreSQL and a Redis. No clone required.

**1. Generate your keys and secrets into a `.env`:**

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

**2. Create `docker-compose.yml`:**

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
      - "3001:3001"   # hosted login (identity)
      - "8081:8081"   # data plane — OAuth2/OIDC issuer + public API
    volumes:
      - ./.env:/.env:ro   # the app reads this at startup

volumes:
  pgdata:
```

**3. Start it:**

```bash
docker compose up -d
```

**4. Finish setup:** open **http://localhost:3000** and complete the setup wizard to create your first tenant and admin. Your OIDC discovery document is served at **http://localhost:8081/.well-known/openid-configuration**.

> **Production:** set `COOKIE_SECURE=true`, real HTTPS hostnames, `APP_ENV=production`, `DB_SSLMODE=require`, and a managed secret provider. See [Environment Variables](docs/contributing/environment-variables.md).

---

## Ports

The image serves each surface on its own port so browser origins stay isolated and auth cookies are host-only.

| Port | Surface | Expose publicly? |
|------|---------|------------------|
| `3000` | Admin console SPA | Yes — operators |
| `3001` | Hosted login / identity SPA | Yes — end users |
| `8081` | Data plane — OAuth2/OIDC issuer + public API | Yes — where the issuer must resolve |
| `8080` | Control plane — management API | **No** — keep internal (the console reaches it in-process) |
| `8082` | Management — health checks + Prometheus `/metrics` | **No** — keep internal |

---

## Documentation

| Document | |
|----------|---|
| [Environment Variables](docs/contributing/environment-variables.md) | Every configuration variable, with defaults |
| [Operator Runbook](docs/documentations/devops/operator-runbook.md) | Install, first-run bootstrap, backups, and upgrades |
| [Architecture](docs/documentations/architecture/architecture.md) | System design and data flow |
| [Service-to-Service Authorization](docs/documentations/service-to-service-authorization/service-to-service-authorization.md) | IAM policy bundles and local authorization |
| [API Reference](docs/openapi.yaml) | OpenAPI 3.1 spec (also served at `/openapi.json`) |
| [Getting Started (contributors)](docs/contributing/getting-started.md) | Local development environment |

---

## Building from source

You only need this for development — the released image is the supported way to run maintainerd-auth.

```bash
git clone https://github.com/maintainerd/maintainerd-auth.git
cd maintainerd-auth

go test ./...                       # run the test suite
go build -tags embedassets ./cmd/server   # build the all-in-one binary (embeds the SPAs)
```

The two SPAs live under `web/console` and `web/identity` and are compiled into the binary via `go:embed` under the `embedassets` build tag.

---

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) and the [getting-started guide](docs/contributing/getting-started.md) before opening a pull request.

---

## License

Copyright 2026 Reyco Seguma.

Licensed under the Apache License 2.0. See [LICENSE](LICENSE) for the license terms and [NOTICE](NOTICE) for attribution.

---

<p align="center">
  <em>Built by <a href="https://github.com/xreyc">Reyco Seguma (@xreyc)</a> and the Maintainerd community.</em>
</p>

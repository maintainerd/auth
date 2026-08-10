<div align="left">
  <img src="https://github.com/user-attachments/assets/8ecfd8bd-e8df-4fe5-a291-bd6192c23a5d" alt="Maintainerd Auth" height="70">
</div>

<br clear="left">

[![Release](https://img.shields.io/github/v/release/maintainerd/maintainerd-auth?logo=github&label=release&color=blue)](https://github.com/maintainerd/maintainerd-auth/releases/latest)
[![Docker](https://img.shields.io/docker/v/xreyc/maintainerd-auth?logo=docker&label=docker&sort=semver&color=blue)](https://hub.docker.com/r/xreyc/maintainerd-auth)
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

Run the released image locally behind nginx — clean HTTPS hostnames, **no ports** — with PostgreSQL + Redis. For local testing; in production you front it with your own TLS.

**1. Download these four files** into one empty folder:

[`docker-compose.yml`](https://raw.githubusercontent.com/maintainerd/maintainerd-auth/main/examples/quickstart/docker-compose.yml) · [`.env.example`](https://raw.githubusercontent.com/maintainerd/maintainerd-auth/main/examples/quickstart/.env.example) · [`nginx.conf`](https://raw.githubusercontent.com/maintainerd/maintainerd-auth/main/examples/quickstart/nginx.conf) · [`setup.sh`](https://raw.githubusercontent.com/maintainerd/maintainerd-auth/main/examples/quickstart/setup.sh)

**2. Run:**

```bash
cp .env.example .env
chmod +x setup.sh && ./setup.sh          # generates your keys + a local TLS cert

sudo tee -a /etc/hosts >/dev/null <<'EOF'
127.0.0.1 console.auth.maintainerd.local identity.auth.maintainerd.local console-api.auth.maintainerd.local identity-api.auth.maintainerd.local
EOF

docker compose up -d
```

**3. Open the setup wizard 👉 https://console.auth.maintainerd.local/setup/tenant** and create your first **tenant** and **admin** (accept the one-time self-signed-cert warning).

| | URL |
|---|---|
| Setup wizard — **start here** | https://console.auth.maintainerd.local/setup/tenant |
| Admin console | https://console.auth.maintainerd.local |
| Hosted login (end users) | https://identity.auth.maintainerd.local |
| OIDC discovery | https://identity-api.auth.maintainerd.local/.well-known/openid-configuration |

> This is a **local** setup for trying it out. In production, front it with your own TLS + real hostnames, keep `APP_ENV=production` and `DB_SSLMODE=require`, and source secrets from a manager. See [Environment Variables](docs/contributing/environment-variables.md).

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

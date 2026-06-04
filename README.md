<div align="left">
  <img src="https://github.com/user-attachments/assets/8ecfd8bd-e8df-4fe5-a291-bd6192c23a5d" alt="Maintainerd Auth" height="70">
</div>

<br clear="left">

<p>
  <a href="https://github.com/maintainerd/maintainerd-auth/actions/workflows/ci.yml">
    <img src="https://github.com/maintainerd/maintainerd-auth/actions/workflows/ci.yml/badge.svg" alt="CI">
  </a>
  <a href="https://github.com/maintainerd/maintainerd-auth/actions/workflows/security.yml">
    <img src="https://github.com/maintainerd/maintainerd-auth/actions/workflows/security.yml/badge.svg" alt="Security">
  </a>
  <a href="https://goreportcard.com/report/github.com/maintainerd/maintainerd-auth">
    <img src="https://goreportcard.com/badge/github.com/maintainerd/maintainerd-auth" alt="Go Report Card">
  </a>
  <a href="https://scorecard.dev/viewer/?uri=github.com/maintainerd/maintainerd-auth">
    <img src="https://api.scorecard.dev/projects/github.com/maintainerd/maintainerd-auth/badge" alt="OpenSSF Scorecard">
  </a>
  <a href="https://codecov.io/gh/maintainerd/maintainerd-auth">
    <img src="https://codecov.io/gh/maintainerd/maintainerd-auth/graph/badge.svg" alt="Coverage">
  </a>
</p>

An open-source, self-hostable identity and access platform. Delivers identity and access management (IAM) for applications, services, and APIs — from authentication and federation to fine-grained authorization.

---

## Features

- **Full OAuth 2.0 + OIDC** — authorization code (PKCE), client credentials, device, token exchange, PAR, CIBA, dynamic client registration, DPoP
- **JWT (RS256)** with multi-key JWKS and automatic key rotation
- **Multi-factor authentication** — TOTP, WebAuthn/passkeys, SMS OTP, backup codes, step-up auth
- **Federation** — OIDC upstream connectors (Google, Microsoft, Apple, GitHub, GitLab), JIT provisioning, identity linking, home-realm discovery
- **Fine-grained access control** — RBAC with granular permissions, plus an IAM resource model of services, APIs, permissions, and policies
- **Multi-tenant** — full tenant isolation, API keys, invite flows
- **Session management** — refresh token rotation, family revocation, reuse detection, concurrent session limits
- **Webhook delivery** — auth event notifications with HMAC-SHA256 signatures and replay protection
- **Audit logging** — structured auth events with retention, per-tenant isolation, and PII redaction
- **Pluggable secret management** — env vars, AWS Secrets Manager, SSM, HashiCorp Vault, Azure Key Vault, GCP Secret Manager
- **Pluggable email providers** — SMTP, SES, SendGrid, Postmark, Mailgun, Resend
- **OpenTelemetry** — traces, metrics, and Prometheus endpoint

---

## Quick Start

```bash
git clone https://github.com/maintainerd/maintainerd-auth.git
cd maintainerd-auth

cp .env.example .env
# Edit .env with your database, Redis, and JWT key settings

docker compose up --build -d
```

The management API is available at `http://localhost:8080/api/v1` and the public auth API at `http://localhost:8081/api/v1`.

### JWT keys

```bash
./scripts/generate-jwt-keys.sh
cat keys/jwt_env_vars.txt >> .env
```

---

## Documentation

| Document | |
|---|---|
| [Getting Started](docs/contributing/getting-started.md) | Set up your local development environment |
| [Environment Variables](docs/contributing/environment-variables.md) | All configuration variables |
| [API Reference](docs/apis/) | Full REST API reference |
| [Architecture](docs/architecture/) | System design and data flow |

---

## Contributing

Contributions are welcome. Please read the [getting started guide](docs/contributing/getting-started.md) before opening a pull request.

```bash
# Fork the repo, then:
git clone https://github.com/<your-username>/maintainerd-auth.git
cd maintainerd-auth

./scripts/dev.sh start   # start the full local stack
go test ./...            # run tests
```

---

## Related Projects

- [`maintainerd/auth-console`](https://github.com/maintainerd/auth-console) — Admin console UI
- [`maintainerd/auth-identity`](https://github.com/maintainerd/auth-identity) — End-user identity portal

---

## License

MIT — see [LICENSE](LICENSE) for details.

---

<p align="center">
  <em>Built by <a href="https://github.com/xreyc">@xreyc</a> and the Maintainerd community.</em>
</p>

<div align="left">
  <img src="https://github.com/user-attachments/assets/8ecfd8bd-e8df-4fe5-a291-bd6192c23a5d" alt="Maintainerd Auth" height="70">
</div>

<br clear="left">

<p>
  <a href="https://github.com/maintainerd/auth/actions/workflows/ci.yml">
    <img src="https://github.com/maintainerd/auth/actions/workflows/ci.yml/badge.svg" alt="CI">
  </a>
  <a href="https://github.com/maintainerd/auth/actions/workflows/security.yml">
    <img src="https://github.com/maintainerd/auth/actions/workflows/security.yml/badge.svg" alt="Security">
  </a>
  <a href="https://goreportcard.com/report/github.com/maintainerd/auth">
    <img src="https://goreportcard.com/badge/github.com/maintainerd/auth" alt="Go Report Card">
  </a>
  <a href="https://scorecard.dev/viewer/?uri=github.com/maintainerd/auth">
    <img src="https://api.scorecard.dev/projects/github.com/maintainerd/auth/badge" alt="OpenSSF Scorecard">
  </a>
  <a href="https://www.bestpractices.dev/projects/TODO">
    <img src="https://img.shields.io/badge/openssf_best_practices-in_progress-yellow?logo=opensourcesecurityfoundation&logoColor=white" alt="OpenSSF Best Practices">
  </a>
  <a href="https://codecov.io/gh/maintainerd/auth">
    <img src="https://codecov.io/gh/maintainerd/auth/graph/badge.svg" alt="Coverage">
  </a>
</p>

## Overview

Maintainerd Auth handles user registration, login, multi-tenancy, role-based access control, and token issuance so that your other services don't have to.

It exposes two interfaces:

- **REST API** — for web and mobile clients
- **gRPC API** — for internal service-to-service communication

It can be used in three ways:

| Mode | Description |
|---|---|
| **Standalone** | Deploy it as a dedicated auth service in front of your own application |
| **Microservice** | Other services authenticate requests by calling this service over gRPC or REST |
| **Maintainerd ecosystem** | Plug it in as the identity layer alongside other Maintainerd services |

---

## Features

- **JWT authentication** with RSA key pairs (RS256)
- **Multi-tenant** support — organisations, tenants, and users are fully isolated
- **Role-based access control (RBAC)** with granular permissions
- **Dual API surface** — private management API (`:8080`) and public auth API (`:8081`)
- **Invite-based registration** and standard open registration
- **Transactional email** for verification, password reset, and invitations
- **Distributed rate limiting** via Redis
- **Pluggable secret management** — env vars, AWS Secrets Manager, SSM, HashiCorp Vault, or file-based secrets
- **Docker-first** with a production multi-stage `Dockerfile` and a local development `Dockerfile.local`

---

## Quick Start

### Option 1 — Docker Compose (recommended)

The fastest way to get everything running locally, including PostgreSQL, Redis, RabbitMQ, and an Nginx proxy.

```bash
git clone https://github.com/maintainerd/auth.git
cd auth

# Set up your environment
cp .env.example .env
# Follow docs/contributing/environment-variables.md to fill in your values

# Start all services
docker-compose up --build -d
```

Services available after startup:

| Service | Address |
|---|---|
| Public REST API | `http://localhost:80/api/v1` (via Nginx) |
| Private REST API | `http://localhost:8080/api/v1` |
| PostgreSQL | `localhost:5433` |
| Redis | `localhost:6379` |
| RabbitMQ management | `http://localhost:15672` |

### Option 2 — Bare metal

```bash
# Prerequisites: Go 1.26+, PostgreSQL, Redis

git clone https://github.com/maintainerd/auth.git
cd auth

cp .env.example .env
# Edit .env with your local database and Redis credentials

go run cmd/server/main.go
```

### Health check

```bash
curl http://localhost:8080/health
# {"status":"ok"}

curl http://localhost:8080/ready
# {"status":"ready"}
```

---

## Architecture

Maintainerd Auth runs two HTTP servers behind an Nginx proxy:

```
                  ┌─────────────────────────────────────┐
                  │              Nginx                   │
                  │  api.maintainerd.auth        → :8080 │  (private / management)
                  │  api.public.maintainerd.auth → :8081 │  (public / auth only)
                  └─────────────────────────────────────┘
                              │               │
               ┌──────────────┘               └──────────────┐
               ▼                                             ▼
        Private API (:8080)                       Public API (:8081)
        All routes including                      Auth routes only
        management & setup                        (login, register, etc.)
```

Other services communicate with Maintainerd Auth over **gRPC** for token validation and user lookups, keeping service-to-service calls fast and typed.

**Data layer:**

| Component | Role |
|---|---|
| PostgreSQL | Persistent storage for users, tenants, roles, permissions |
| Redis | Distributed rate limiting and session caching |
| RabbitMQ | Async event publishing (email dispatch, audit events) |

---

## Configuration

Copy the environment file and follow the documentation to fill in your values:

```bash
cp .env.example .env
```

See [`docs/contributing/environment-variables.md`](docs/contributing/environment-variables.md) for the full variable reference and a ready-to-use Quick Setup block.

### JWT keys

Generate a key pair before starting the service:

```bash
./scripts/generate-jwt-keys.sh
cat keys/jwt_env_vars.txt >> .env
```

---

## Deployment

Build the production image:

```bash
docker build -t maintainerd/auth:latest .
```

Run it:

```bash
docker run -d \
  --name maintainerd-auth \
  -p 8080:8080 \
  -p 8081:8081 \
  --env-file .env \
  maintainerd/auth:latest
```

For full environment variable guidance including secret management options (AWS Secrets Manager, HashiCorp Vault, Kubernetes Secrets), see [`docs/deployment/environment-variables.md`](docs/deployment/environment-variables.md).

---

## Documentation

| Document | Description |
|---|---|
| [Contributing — Getting Started](docs/contributing/getting-started.md) | Set up your local development environment |
| [Contributing — Environment Variables](docs/contributing/environment-variables.md) | All variables for local development |
| [Deployment — Environment Variables](docs/deployment/environment-variables.md) | All variables for production deployment |

---

## Contributing

Contributions are welcome. Please read the [getting started guide](docs/contributing/getting-started.md) before opening a pull request.

```bash
# Fork the repo, then:
git clone https://github.com/<your-username>/auth.git
cd auth

./scripts/dev.sh start   # start the full local stack
go test ./...            # run tests
```

---

## Related Projects

- [`maintainerd/core`](https://github.com/maintainerd/core) — Core platform services
- [`maintainerd/contracts`](https://github.com/maintainerd/contracts) — Shared gRPC contracts
- [`maintainerd/web`](https://github.com/maintainerd/web) — Web dashboard *(coming soon)*

---

## License

MIT — see [LICENSE](LICENSE) for details.

---

<p align="center">
  <em>Built by <a href="https://github.com/xreyc">@xreyc</a> and the Maintainerd community.</em>
</p>

<p align="center">
  <sub>Security scanning powered by</sub>
  <br>
  <a href="https://scorecard.dev/viewer/?uri=github.com/maintainerd/auth"><img src="https://img.shields.io/badge/OpenSSF_Scorecard-grey?logo=opensourcesecurityfoundation&logoColor=white" alt="OpenSSF Scorecard"></a>
  <a href="https://www.bestpractices.dev/projects/TODO"><img src="https://img.shields.io/badge/OpenSSF_Best_Practices-grey?logo=opensourcesecurityfoundation&logoColor=white" alt="OpenSSF Best Practices"></a>
  <a href="https://semgrep.dev"><img src="https://img.shields.io/badge/Semgrep-grey?logo=semgrep&logoColor=white" alt="Semgrep"></a>
  <a href="https://snyk.io"><img src="https://img.shields.io/badge/Snyk-grey?logo=snyk&logoColor=white" alt="Snyk"></a>
  <a href="https://github.com/features/security"><img src="https://img.shields.io/badge/CodeQL-grey?logo=github&logoColor=white" alt="CodeQL"></a>
  <a href="https://codecov.io"><img src="https://img.shields.io/badge/Codecov-grey?logo=codecov&logoColor=white" alt="Codecov"></a>
</p>

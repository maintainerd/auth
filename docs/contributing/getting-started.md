# Getting Started

Welcome to **Maintainerd Auth**! This guide walks you through setting up a local development environment so you can run, test, and contribute to the project.

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Fork & Clone](#fork--clone)
- [Environment Setup](#environment-setup)
- [Option A — Docker Compose (Recommended)](#option-a--docker-compose-recommended)
- [Option B — Bare Metal](#option-b--bare-metal)
- [Generate JWT Keys](#generate-jwt-keys)
- [Verify the Service is Running](#verify-the-service-is-running)
- [Available Commands](#available-commands)
- [Project Structure](#project-structure)
- [Submitting a Pull Request](#submitting-a-pull-request)

---

## Prerequisites

Make sure the following tools are installed before you begin.

| Tool | Minimum Version | Notes |
|---|---|---|
| [Go](https://go.dev/dl/) | **1.26+** | Required for bare-metal runs and `make` targets |
| [Docker](https://docs.docker.com/get-docker/) | 24+ | Required for Option A |
| [Docker Compose](https://docs.docker.com/compose/install/) | v2+ | Required for Option A (`docker compose` or `docker-compose`) |
| [OpenSSL](https://www.openssl.org/) | any | Required for JWT key generation |
| [protoc](https://grpc.io/docs/protoc-installation/) + Go plugins | any | Only needed if you modify `.proto` files |
| [Git](https://git-scm.com/) | any | — |

> **macOS:** Install Go and OpenSSL via [Homebrew](https://brew.sh/): `brew install go openssl`  
> **Windows:** Use [WSL2](https://learn.microsoft.com/en-us/windows/wsl/) for the best experience.

---

## Fork & Clone

1. **Fork** the repository on GitHub — click the **Fork** button at the top-right of the repo page.

2. **Clone** your fork locally:

```bash
git clone https://github.com/<your-username>/auth.git
cd auth
```

3. Add the upstream remote so you can keep your fork in sync:

```bash
git remote add upstream https://github.com/maintainerd/auth.git
```

---

## Environment Setup

Copy the sample environment file and fill in your local values:

```bash
cp .env.example .env
```

> See **[`docs/contributing/environment-variables.md`](./environment-variables.md)** for a full description of every variable and how to generate secrets.

At minimum you need to:

1. Set your database credentials (`DB_*`)
2. Set your Redis password (`REDIS_*`)
3. Generate and set JWT keys (`JWT_PRIVATE_KEY`, `JWT_PUBLIC_KEY`) — see [Generate JWT Keys](#generate-jwt-keys) below

---

## Option A — Docker Compose (Recommended)

This is the fastest way to get a fully working environment including PostgreSQL, Redis, RabbitMQ, and an Nginx proxy.

```bash
# Start all services (builds the image on first run)
./scripts/dev.sh start

# Or equivalently
docker-compose up --build -d
```

Services started:

| Service | Local address |
|---|---|
| Auth API | `http://localhost:80` (via Nginx) |
| Auth API (direct) | `http://localhost:8080` |
| PostgreSQL | `localhost:5433` |
| Redis | `localhost:6379` |
| RabbitMQ management UI | `http://localhost:15672` |

**Useful dev script commands:**

```bash
./scripts/dev.sh logs          # Tail logs for all services
./scripts/dev.sh logs auth     # Tail logs for the auth service only
./scripts/dev.sh reload        # Rebuild & restart only the auth container
./scripts/dev.sh status        # Show container status
./scripts/dev.sh shell         # Open a shell inside the auth container
./scripts/dev.sh stop          # Stop all services
./scripts/dev.sh restart       # Full stop + rebuild + start
./scripts/dev.sh clean         # Destroy all containers, images & volumes
```

---

## Option B — Bare Metal

Use this if you prefer to manage PostgreSQL and Redis yourself or run the Go binary directly.

**1. Install dependencies:**

```bash
go mod download
```

**2. Make sure PostgreSQL and Redis are running locally, then update your `.env` accordingly:**

```env
DB_HOST=localhost
DB_PORT=5432
REDIS_HOST=localhost
REDIS_PORT=6379
```

**3. Run the service:**

```bash
# Using the Makefile
make run

# Or directly
go run cmd/server/main.go
```

---

## Generate JWT Keys

The service requires an RSA key pair to sign and verify JWTs.  
Use the included script to generate one:

```bash
./scripts/generate-jwt-keys.sh
```

This creates a `./keys/` directory containing:

- `jwt_private.pem` — private signing key
- `jwt_public.pem` — public verification key
- `jwt_env_vars.txt` — ready-to-paste `.env` lines

Append the generated keys to your `.env`:

```bash
cat keys/jwt_env_vars.txt >> .env
```

> See [`docs/contributing/environment-variables.md` → JWT Configuration](./environment-variables.md#jwt-configuration) for full details and manual OpenSSL instructions.

---

## Verify the Service is Running

```bash
curl http://localhost:8080/health
# Expected: {"status":"ok"}

curl http://localhost:8080/ready
# Expected (when DB and Redis are healthy): {"status":"ready"}
# Returns 503 with {"status":"not ready","reason":"..."} if a dependency is down.
```

---

## Available Commands

```bash
make run          # Run the service locally
make build        # Compile the binary to ./bin/auth
make clean        # Remove build artifacts
make proto        # Regenerate Go code from .proto files
make proto-clean  # Remove all generated proto files
make tidy         # Run go mod tidy
```

---

## Project Structure

```
.
├── cmd/server/         # Application entry point
├── internal/
│   ├── app/            # Application wiring / DI
│   ├── config/         # Environment config loading
│   ├── contract/       # gRPC .proto definitions
│   ├── dto/            # Request / response data transfer objects
│   ├── gen/            # Generated gRPC code (do not edit manually)
│   ├── grpc/           # gRPC server handlers
│   ├── handler/        # REST HTTP handlers
│   ├── middleware/      # HTTP middleware (auth, rate-limit, logging…)
│   ├── model/          # Database models
│   ├── repository/     # Data access layer
│   ├── route/          # Route registration
│   ├── runner/         # Background workers / migration runner
│   ├── service/        # Business logic layer
│   ├── startup/        # Boot sequence
│   ├── templates/      # HTML email templates
│   └── util/           # Shared utilities
├── scripts/
│   ├── dev.sh                  # Development helper script
│   └── generate-jwt-keys.sh   # JWT key generation script
├── docs/
│   ├── contributing/           # Contributor documentation (you are here)
│   └── deployment/             # Production deployment documentation
├── docker-compose.yml
├── Dockerfile                  # Production image
├── Dockerfile.local            # Development image (with hot reload)
└── Makefile
```

---

## Submitting a Pull Request

1. **Sync** your fork with upstream before starting:

```bash
git fetch upstream
git checkout main
git merge upstream/main
```

2. **Create a feature branch** — use a descriptive name:

```bash
git checkout -b fix/login-rate-limit
git checkout -b feat/oauth2-provider
```

3. **Make your changes** — keep commits focused and atomic.

4. **Run tests** before pushing:

```bash
go test ./...
```

5. **Push** your branch and open a Pull Request against `maintainerd/auth:main`.

6. Fill in the PR template and link any related issues.

> For significant changes, open an issue first to discuss the approach before investing time in implementation.


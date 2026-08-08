<div align="left">
  <img src="https://github.com/user-attachments/assets/8ecfd8bd-e8df-4fe5-a291-bd6192c23a5d" alt="Maintainerd Auth Console" height="70">
</div>

<br clear="left">

<p>
  <a href="https://github.com/maintainerd/maintainerd-auth-console/actions/workflows/ci.yml">
    <img src="https://github.com/maintainerd/maintainerd-auth-console/actions/workflows/ci.yml/badge.svg" alt="CI">
  </a>
  <a href="https://github.com/maintainerd/maintainerd-auth-console/actions/workflows/security.yml">
    <img src="https://github.com/maintainerd/maintainerd-auth-console/actions/workflows/security.yml/badge.svg" alt="Security">
  </a>
  <a href="https://scorecard.dev/viewer/?uri=github.com/maintainerd/maintainerd-auth-console">
    <img src="https://api.scorecard.dev/projects/github.com/maintainerd/maintainerd-auth-console/badge" alt="OpenSSF Scorecard">
  </a>
  <a href="https://www.bestpractices.dev/projects/TODO">
    <img src="https://img.shields.io/badge/openssf_best_practices-in_progress-yellow?logo=opensourcesecurityfoundation&logoColor=white" alt="OpenSSF Best Practices">
  </a>
  <a href="https://codecov.io/gh/maintainerd/maintainerd-auth-console">
    <img src="https://codecov.io/gh/maintainerd/maintainerd-auth-console/graph/badge.svg" alt="Coverage">
  </a>
</p>

## Overview

Maintainerd Auth Console is the management dashboard for [`maintainerd-auth`](https://github.com/maintainerd/maintainerd-auth) — a multi-tenant authentication and authorization service. It is the operator-facing surface for administering tenants, users, roles, OAuth2 clients, identity providers, policies, and security settings.

It targets the **internal management API** (`maintainerd-auth` port `:8080`) and is the equivalent of the Auth0 / Okta / Keycloak admin console for the Maintainerd ecosystem.

<img width="1916" height="1013" alt="image" src="https://github.com/user-attachments/assets/2eeff397-9321-4733-b62d-79aee517c162" />

It can be used in three ways:

| Mode | Description |
|---|---|
| **Standalone** | Deploy it next to a single `maintainerd-auth` instance as the dedicated admin UI |
| **Multi-tenant** | One console fronts an `maintainerd-auth` instance hosting many tenants, with the active tenant scoped by URL |
| **Maintainerd ecosystem** | Plug it in as the identity admin layer alongside other Maintainerd services |

> End-user authentication flows (login widget, consent screen, OAuth2 redirects, hosted MFA challenge) are **not** part of this console. They belong to a separate public-facing app that talks to `maintainerd-auth` port `:8081`.

---

## Features

- **Multi-tenant management** — tenants, members, and tenant-scoped settings via URL-driven tenant resolution
- **Users & profiles** — full lifecycle: create, edit, status changes, role assignment, profile management
- **RBAC** — roles, permissions, and policies with assignment UIs
- **OAuth2 / OIDC clients** — confidential and public clients, redirect-URI management, token TTLs
- **API keys** — issue, scope, and rotate machine credentials
- **Federation** — identity provider and social provider configuration
- **Security center** — password policies, session management, threat detection, IP restrictions, security headers
- **Branding** — login, email, and SMS template editors per tenant
- **Audit & analytics** — log monitoring, notifications, dashboard analytics
- **Cookie-based session auth** — no tokens in `localStorage`, no CORS in development (Vite proxy)

---

## Quick Start

### Prerequisites

- **Node.js 22** (`>=22 <23`, see `.nvmrc`; Vite 7 + React 19)
- A running [`maintainerd-auth`](https://github.com/maintainerd/maintainerd-auth) backend exposing port `:8080`

### Run the console

```bash
git clone https://github.com/maintainerd/maintainerd-auth-console.git
cd maintainerd-auth-console

# Install dependencies
npm install

# Start the dev server (Vite proxies /api/* to the backend)
npm run dev
```

The dev server prints a URL — typically `http://localhost:5173`.

On first run against a fresh `maintainerd-auth` instance you will be redirected through the setup wizard (`/setup/tenant` → `/setup/admin` → `/setup/profile`) to bootstrap the first tenant and admin.

### Production build

```bash
npm run build       # type-check (tsc -b) + bundle into dist/
npm run preview     # serve the build locally for smoke testing
```

---

## Architecture

The console is a single-page React application with a layered architecture and strict one-way dependency flow:

```
                  ┌───────────────────────────────────┐
                  │      Browser (SPA, Vite-served)   │
                  └─────────────────┬─────────────────┘
                                    │
              ┌─────────────────────▼─────────────────────┐
              │   Pages  →  Components (Radix + Tailwind) │
              │                       │                    │
              │                       ▼                    │
              │                     Hooks                  │
              │      ┌───────────────┴───────────────┐     │
              │      ▼                               ▼     │
              │  Redux Toolkit              TanStack Query │
              │  (auth, tenant)             (server state) │
              │                       │                    │
              │                       ▼                    │
              │              Service Layer (Axios)         │
              └─────────────────────┬─────────────────────┘
                                    │  withCredentials: true
                                    ▼
                  maintainerd-auth — Management Port :8080
                                    │
                                    ▼
                        PostgreSQL · Redis · RabbitMQ
```

**Layer responsibilities:**

| Layer | Role |
|---|---|
| Pages (`src/pages/`) | Route-bound views; consume hooks, render UI |
| Components (`src/components/`) | Reusable primitives (ui, layout, sidebar, data-table, …) |
| Hooks (`src/hooks/`) | Bridge between Redux/Query and the service layer |
| Services (`src/services/api/`) | Typed HTTP calls; the only layer that knows about Axios |
| Store (`src/store/`) | Redux Toolkit slices for session-scoped state |

See [`docs/architecture.md`](docs/architecture.md) for the full breakdown.

---

## Configuration

The console is configured via `.env` at the project root (copy `.env.example` to
`.env` to start). In development, the Vite dev server proxies `/api/*` and
`/public-api/*` to the backend, so these base URLs are only used for production
builds. See `.env.example` for the full list.

| Variable                       | Default                                             | Description                                                     |
| ------------------------------ | --------------------------------------------------- | --------------------------------------------------------------- |
| `VITE_AUTH_API_BASE_URL`       | `https://private-api.auth.maintainerd.local/api/v1` | Production base URL of the internal management API (port `:8080`) |
| `VITE_AUTH_PUBLIC_API_BASE_URL`| `https://public-api.auth.maintainerd.local/api/v1`  | Production base URL of the public API (port `:8081`, OAuth bootstrap) |
| `VITE_AUTH_IDENTITY_BASE_URL`  | `https://identity.auth.maintainerd.local`           | Base URL of the hosted identity (public login) UI               |

In production these can also be injected at container start via `window.__ENV__`
(see `docker-entrypoint.sh`), letting a single built image target different API
origins without a rebuild.

### Pointing the dev proxy at your backend

`vite.config.ts` ships proxies targeting `https://private-api.auth.maintainerd.local`
(internal API) and `https://public-api.auth.maintainerd.local` (public API). If
your backend is reachable elsewhere (e.g. bare-metal on `localhost:8080`), update
the `target` in `vite.config.ts`:

```ts
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:8080',
      changeOrigin: true,
      secure: false,
    }
  }
}
```

See [`docs/getting-started.md`](docs/getting-started.md) for the full local-development walkthrough.

---

## Deployment

The console builds to a static `dist/` directory that can be served from any CDN, object store, or static-file server.

```bash
npm run build
# Outputs dist/index.html + assets
```

Serve it behind the same Nginx (or equivalent) that fronts the management port, so cookies set by `maintainerd-auth` are same-site for the console origin.

### Run via Docker

The repository ships a production `Dockerfile` (Node build stage → Nginx serve
stage) with a committed, hardened `nginx.conf` (SPA fallback, gzip, immutable
asset caching, and CSP + security headers). Nginx listens on port `8080` as a
non-root user.

```bash
# Build the image
docker build -t maintainerd-auth-console .

# Run it, injecting API origins at container start (window.__ENV__)
docker run --rm -p 8080:8080 \
  -e VITE_AUTH_API_BASE_URL="https://private-api.auth.maintainerd.local/api/v1" \
  -e VITE_AUTH_PUBLIC_API_BASE_URL="https://public-api.auth.maintainerd.local/api/v1" \
  -e VITE_AUTH_IDENTITY_BASE_URL="https://identity.auth.maintainerd.local" \
  maintainerd-auth-console
```

`docker-entrypoint.sh` writes `/config.js` from those environment variables on
start, so the same image can target different environments without a rebuild.
The full local stack (backend, console, identity, nginx) is wired up in the
[`maintainerd-dev`](https://github.com/maintainerd/maintainerd-dev) Docker Compose environment.

---

## Documentation

| Document | Description |
|---|---|
| [Architecture](docs/architecture.md) | Layered architecture, request lifecycle, multi-tenancy, error handling |
| [Getting Started](docs/getting-started.md) | Local dev environment, dev proxy, scripts, project structure |
| [Feature List](docs/feature-list.md) | Backend capabilities mapped to console implementation status |
| [Checklist](docs/checklist.md) | Architecture & code-quality roadmap |

---

## Contributing

Contributions are welcome. Please read the [getting started guide](docs/getting-started.md) before opening a pull request.

```bash
# Fork the repo, then:
git clone https://github.com/<your-username>/maintainerd-auth-console.git
cd maintainerd-auth-console

npm install
npm run dev      # start the dev server
npm run lint     # lint the project
npm run build    # type-check + production build
```

---

## Related Projects

- [`maintainerd-auth`](https://github.com/maintainerd/maintainerd-auth) — Authentication & authorization backend (the API this console manages)
- [`maintainerd-auth-identity`](https://github.com/maintainerd/maintainerd-auth-identity) — Hosted identity (public login) UI, the sibling front-end
- [`maintainerd-dev`](https://github.com/maintainerd/maintainerd-dev) — Local development environment (Docker Compose, nginx, setup scripts)

---

## License

Copyright 2026 Reyco Seguma.

Licensed under the Apache License 2.0. See [LICENSE](LICENSE) for the
license terms and [NOTICE](NOTICE) for attribution.

---

<p align="center">
  <em>Built by <a href="https://github.com/xreyc">Reyco Seguma (@xreyc)</a> and the Maintainerd community.</em>
</p>

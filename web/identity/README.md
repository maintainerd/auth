<div align="left">
  <img src="https://github.com/user-attachments/assets/8ecfd8bd-e8df-4fe5-a291-bd6192c23a5d" alt="Maintainerd Auth Identity" height="70">
</div>

<br clear="left">

<p>
  <a href="https://github.com/maintainerd/maintainerd-auth-identity/actions/workflows/ci.yml">
    <img src="https://github.com/maintainerd/maintainerd-auth-identity/actions/workflows/ci.yml/badge.svg" alt="CI">
  </a>
  <a href="https://github.com/maintainerd/maintainerd-auth-identity/actions/workflows/security.yml">
    <img src="https://github.com/maintainerd/maintainerd-auth-identity/actions/workflows/security.yml/badge.svg" alt="Security">
  </a>
  <a href="https://scorecard.dev/viewer/?uri=github.com/maintainerd/maintainerd-auth-identity">
    <img src="https://api.scorecard.dev/projects/github.com/maintainerd/maintainerd-auth-identity/badge" alt="OpenSSF Scorecard">
  </a>
  <a href="https://www.bestpractices.dev/projects/TODO">
    <img src="https://img.shields.io/badge/openssf_best_practices-in_progress-yellow?logo=opensourcesecurityfoundation&logoColor=white" alt="OpenSSF Best Practices">
  </a>
  <a href="https://codecov.io/gh/maintainerd/maintainerd-auth-identity">
    <img src="https://codecov.io/gh/maintainerd/maintainerd-auth-identity/graph/badge.svg" alt="Coverage">
  </a>
</p>

## Overview

Maintainerd Auth Identity is the public-facing authentication frontend for [`maintainerd-auth`](https://github.com/maintainerd/maintainerd-auth) — a multi-tenant authentication and authorization service. It provides end-user login, registration, magic link, and invite-based authentication flows, serving as the identity UI for external client applications (OAuth2, direct login, etc.).

It targets the **public API** (`maintainerd-auth` port `:8081`) and is the equivalent of the hosted login / signup widget of the Maintainerd ecosystem.

> Operator-facing administration (tenant management, user management, security settings, OAuth2 client config, etc.) lives in the separate [`maintainerd-auth-console`](https://github.com/maintainerd/maintainerd-auth-console) app, which targets the internal management port `:8080`.

---

## Features

- **Password login** — email + password with MFA step-up support (TOTP, SMS, passkey, email OTP, backup codes)
- **Magic link login** — passwordless sign-in via email link
- **Self-registration** — public sign-up with tenant-controlled password policies
- **Invite-based registration** — signed invite URL flow with token validation
- **Forgot / reset password** — email-based password reset flow
- **TOTP — email verification step**
- **Cookie-based session auth** — no tokens in `localStorage`, no CORS in development (Vite proxy)
- **Multi-tenant** — public login is client-scoped (`/login?client_id=...`); the backend resolves the tenant from that client

---

## Quick Start

### Prerequisites

- **Node.js 22** (`>=22 <23`; see `.nvmrc`) — Vite 7 + React 19
- A running [`maintainerd-auth`](https://github.com/maintainerd/maintainerd-auth) backend exposing the **public port** `:8081`

### Run the identity app

```bash
git clone https://github.com/maintainerd/maintainerd-auth-identity.git
cd maintainerd-auth-identity

# Install dependencies
npm install

# Start the dev server (Vite proxies /api/* to the backend public port)
npm run dev
```

The dev server prints a URL — typically `http://localhost:5174`.

First-run bootstrap of the initial tenant and admin is handled by the internal admin app, [`maintainerd-auth-console`](https://github.com/maintainerd/maintainerd-auth-console) (which targets the internal `:8080` surface where the setup endpoints live). Once a tenant and admin exist, users authenticate here in the identity app.

### Production build

```bash
npm run build       # type-check (tsc -b) + bundle into dist/
npm run preview     # serve the build locally for smoke testing
```

---

## Architecture

The identity app is a single-page React application with a layered architecture:

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
                   maintainerd-auth — Public Port :8081
                                     │
                                     ▼
                         PostgreSQL · Redis · RabbitMQ
```

**Layer responsibilities:**

| Layer | Role |
|---|---|
| Pages (`src/pages/`) | Route-bound views; consume hooks, render UI |
| Components (`src/components/`) | Reusable primitives (ui, layout, form, step-up) |
| Hooks (`src/hooks/`) | Bridge between Redux/Query and the service layer |
| Services (`src/services/api/`) | Typed HTTP calls; the only layer that knows about Axios |
| Store (`src/store/`) | Redux Toolkit slices for session-scoped state (auth, tenant) |

---

## Configuration

The identity app is configured via `.env` at the project root (copy `.env.example`). In development, the Vite dev server proxies `/api/*` to the backend, so `VITE_AUTH_API_BASE_URL` is only used for production builds. For published Docker images the same value is injected at container start (see **Run via Docker**), so a single image can point at any backend without rebuilding.

| Variable                  | Example (`.env.example`)                             | Description                                        |
| ------------------------- | ---------------------------------------------------- | -------------------------------------------------- |
| `VITE_AUTH_API_BASE_URL`  | `https://public-api.auth.maintainerd.local/api/v1`   | Base URL of the public API (`maintainerd-auth` :8081) |

### Pointing the dev proxy at your backend

`vite.config.ts` ships a proxy targeting `https://public-api.auth.maintainerd.local` (the nginx host from the [`maintainerd-dev`](https://github.com/maintainerd/maintainerd-dev) environment). If your backend is reachable elsewhere (e.g. bare-metal on `localhost:8081`), update the `target`:

```ts
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:8081',
      changeOrigin: true,
      secure: false,
    }
  }
}
```

---

## Deployment

The identity app builds to a static `dist/` directory that can be served from any CDN, object store, or static-file server.

```bash
npm run build
# Outputs dist/index.html + assets
```

### Run via Docker

The included multi-stage `Dockerfile` (Node build stage → non-root Nginx serve stage on port `8080`) supports **runtime configuration** — the API base URL is injected when the container starts, so one image works against any backend without rebuilding.

```bash
# Build the image
docker build -t maintainerd-auth-identity .

# Run it, pointing at your backend's public API
docker run --rm -p 8080:8080 \
  -e VITE_AUTH_API_BASE_URL="https://your-backend.example.com/api/v1" \
  maintainerd-auth-identity
```

The entrypoint writes the value into `config.js` (`window.__ENV__`) at startup; the build-time `.env` value is used as the fallback.

---

## Documentation

| Document | Description |
|---|---|
| [Architecture](docs/architecture.md) | Layered architecture, request lifecycle, multi-tenancy, error handling |
| [Getting Started](docs/getting-started.md) | Local dev environment, dev proxy, scripts, project structure |

---

## Contributing

Contributions are welcome. Please read the [getting started guide](docs/getting-started.md) before opening a pull request.

```bash
# Fork the repo, then:
git clone https://github.com/<your-username>/maintainerd-auth-identity.git
cd maintainerd-auth-identity

npm install
npm run dev      # start the dev server
npm run lint     # lint the project
npm run build    # type-check + production build
```

---

## Related Projects

- [`maintainerd-auth`](https://github.com/maintainerd/maintainerd-auth) — Authentication & authorization backend (the API this app uses)
- [`maintainerd-auth-console`](https://github.com/maintainerd/maintainerd-auth-console) — Admin management console (targets the internal port `:8080`)
- [`maintainerd-dev`](https://github.com/maintainerd/maintainerd-dev) — Local dev environment (docker compose, nginx, setup scripts) for the full stack

---

## License

Copyright 2026 Reyco Seguma.

Licensed under the Apache License 2.0. See [LICENSE](LICENSE) for the
license terms and [NOTICE](NOTICE) for attribution.

---

<p align="center">
  <em>Built by <a href="https://github.com/xreyc">Reyco Seguma (@xreyc)</a> and the Maintainerd community.</em>
</p>

# Getting Started

Welcome to **Maintainerd Auth Console**! This guide walks you through setting up a local development environment so you can run, test, and contribute to the project.

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Fork & Clone](#fork--clone)
- [Install Dependencies](#install-dependencies)
- [Environment Setup](#environment-setup)
- [Running the Backend](#running-the-backend)
- [Running the Console](#running-the-console)
- [How the Dev Proxy Works](#how-the-dev-proxy-works)
- [Available Scripts](#available-scripts)
- [Project Structure](#project-structure)
- [Submitting a Pull Request](#submitting-a-pull-request)

---

## Prerequisites

Make sure the following tools are installed before you begin.

| Tool                                                       | Minimum Version | Notes                                                  |
| ---------------------------------------------------------- | --------------- | ------------------------------------------------------ |
| [Node.js](https://nodejs.org/)                             | **20.x LTS+**   | Vite 7 and React 19 require Node 20 or newer           |
| [npm](https://www.npmjs.com/) / pnpm / yarn                | npm 10+         | This project ships a `package-lock.json`               |
| [Git](https://git-scm.com/)                                | any             | —                                                      |
| Running `maintainerd-auth` backend                         | see its README  | The console will not function without the backend      |

> The console targets the **management port** (default `:8080`) of `maintainerd-auth`. You do not need to expose the public auth port (`:8081`) for the console to work.

---

## Fork & Clone

1. **Fork** the repository on GitHub — click the **Fork** button at the top-right of the repo page.

2. **Clone** your fork locally:

```bash
git clone https://github.com/<your-username>/maintainerd-auth-console.git
cd maintainerd-auth-console
```

3. Add the upstream remote so you can keep your fork in sync:

```bash
git remote add upstream https://github.com/maintainerd/auth-console.git
```

---

## Install Dependencies

```bash
npm install
```

This installs every runtime and dev dependency listed in `package.json`. Lock-file integrity is enforced — please commit `package-lock.json` updates that result from real dependency changes only.

---

## Environment Setup

Create a local `.env` file at the project root:

```bash
cp .env.example .env   # if .env.example exists, otherwise create the file directly
```

Supported variables:

| Variable                  | Default                                       | Description                                                    |
| ------------------------- | --------------------------------------------- | -------------------------------------------------------------- |
| `VITE_AUTH_API_BASE_URL`  | `http://api.maintainerd.auth/api/v1`          | Production base URL of the management API. Ignored in dev.     |

> **Development mode does not use `VITE_AUTH_API_BASE_URL`.** Vite serves the API under the relative path `/api/v1` and proxies it through to the backend. See [How the Dev Proxy Works](#how-the-dev-proxy-works) below.

---

## Running the Backend

Start `maintainerd-auth` first. The shortest path is its Docker Compose stack:

```bash
# in your maintainerd-auth checkout
cp .env.example .env
docker-compose up --build -d
```

After it boots you should be able to hit:

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

If you run the backend in bare-metal mode or behind a different host name, point the dev proxy at it (see below).

---

## Running the Console

```bash
npm run dev
```

The Vite dev server prints a URL — typically `http://localhost:5173`. Open it in your browser.

On first run with a fresh backend you will be redirected to the setup wizard (`/setup/tenant` → `/setup/admin` → `/setup/profile`) to bootstrap the first tenant and admin user. After that, log in with the admin credentials you just created.

---

## How the Dev Proxy Works

`vite.config.ts` ships a proxy that rewrites `/api/*` calls from the dev server to the backend:

```ts
server: {
  proxy: {
    '/api': {
      target: 'http://api.maintainerd.auth',
      changeOrigin: true,
      secure: false,
      // …
    }
  }
}
```

This means:

- The console always issues requests to **the same origin it was served from**, which avoids browser CORS in development.
- Cookies set by the backend are accepted as same-site, which is necessary for the cookie-based session auth.
- `import.meta.env.DEV` causes `src/services/api/config.ts` to use the relative `/api/v1` base URL rather than `VITE_AUTH_API_BASE_URL`.

If your backend is reachable at a different host or port, update the `target` value in `vite.config.ts`. Common alternatives:

```ts
target: 'http://localhost:8080'  // backend running bare-metal on host
target: 'http://host.docker.internal:8080'  // backend in Docker, console on host
```

`http://api.maintainerd.auth` (the default) assumes the recommended local Nginx setup from the backend repo. If you use it, add the host to `/etc/hosts`:

```
127.0.0.1   api.maintainerd.auth
```

---

## Available Scripts

```bash
npm run dev        # Start the Vite dev server with HMR
npm run build      # Type-check (tsc -b) and produce a production build in dist/
npm run preview    # Serve the production build locally for smoke testing
npm run lint       # Run ESLint over the project
```

The `build` step runs `tsc -b` first; type errors will fail the build. There is no separate type-check script — keep your editor's TypeScript server running while developing.

---

## Project Structure

```
.
├── docs/                      # This documentation
├── public/                    # Static assets served as-is
├── src/
│   ├── App.tsx                # Route table + provider tree + bootstrap effects
│   ├── main.tsx               # Entry point; wraps <App/> with <Provider/> + <BrowserRouter/>
│   ├── components/            # Reusable UI (ui, layout, sidebar, header, data-table, …)
│   ├── pages/                 # One folder per route group (users/, roles/, security/, …)
│   ├── hooks/                 # Custom hooks: Redux bindings, TanStack Query hooks
│   ├── services/api/          # HTTP service layer; one folder per backend resource
│   │   ├── client.ts          # Axios instance + ApiError class
│   │   ├── config.ts          # Base URL + endpoint paths
│   │   └── <resource>/        # fetchX, createX, updateX, deleteX + types
│   ├── store/                 # Redux Toolkit slices (auth, tenant) + typed hooks
│   ├── lib/                   # Cross-cutting helpers (queryClient, cn, …)
│   ├── utils/                 # Pure utility functions (tenant resolution, formatters)
│   ├── constants/             # App-wide constants
│   ├── types/                 # Shared TypeScript types
│   └── styles/                # Global CSS, Tailwind layer overrides
├── package.json
├── tsconfig.json
├── vite.config.ts
└── README.md
```

For a deeper architectural walkthrough see [`architecture.md`](./architecture.md). For a status report against the backend's feature surface see [`feature-list.md`](./feature-list.md).

---

## Submitting a Pull Request

1. Create a feature branch: `git checkout -b feature/<short-description>`.
2. Make your changes and run `npm run lint` and `npm run build` locally before pushing.
3. Open a PR against `main` with a clear description of the change and screenshots for any UI work.
4. Address review feedback, then squash-merge.

If you are about to refactor a large area, please open an issue or discussion first so we can align on the approach — see [`checklist.md`](./checklist.md) for areas that already have a planned direction.

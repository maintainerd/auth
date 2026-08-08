# Architecture

This document describes the internal architecture of **Maintainerd Auth Console** — the management/admin SPA for the [`maintainerd-auth`](https://github.com/maintainerd/auth) identity service. If you are new to the codebase, read this first to understand how the pieces fit together before diving into any specific module.

---

## Table of Contents

- [High-Level Overview](#high-level-overview)
- [Relationship to the Backend](#relationship-to-the-backend)
- [Architecture Diagram](#architecture-diagram)
- [Request Lifecycle](#request-lifecycle)
- [Entry Point & Bootstrap](#entry-point--bootstrap)
- [Layer-by-Layer Breakdown](#layer-by-layer-breakdown)
  - [Routing](#routing)
  - [Pages](#pages)
  - [Components](#components)
  - [Hooks](#hooks)
  - [Service Layer](#service-layer)
  - [HTTP Client](#http-client)
  - [Global State (Redux)](#global-state-redux)
  - [Server State (TanStack Query)](#server-state-tanstack-query)
- [Multi-Tenancy](#multi-tenancy)
- [Authentication Flow](#authentication-flow)
- [Error Handling](#error-handling)
- [Path Aliases](#path-aliases)
- [Technology Stack](#technology-stack)

---

## High-Level Overview

The console is a **single-page React application** that follows a layered architecture with strict one-way dependency flow:

```
Pages / Components
       ↓
   Hooks  (Redux + Query bindings)
       ↓
  Service Layer  (typed API calls)
       ↓
   HTTP Client  (axios + interceptors)
       ↓
  maintainerd-auth (Management Port :8080)
```

Each layer only depends on the layer directly below it. Pages never call the HTTP client directly; hooks never reach into pages; the service layer knows nothing about React. This keeps the concerns isolated and makes every layer independently testable.

---

## Relationship to the Backend

`maintainerd-auth` exposes two HTTP servers behind an Nginx proxy:

| Port  | Surface                | Audience                                       |
| ----- | ---------------------- | ---------------------------------------------- |
| 8080  | **Management API**     | Admins, internal tooling, **this console**     |
| 8081  | **Public Auth API**    | End users, external client apps (login, OAuth) |

This console targets **only port 8080**. It is the operator-facing dashboard — the equivalent of the Auth0 / Okta / Keycloak admin console. End-user authentication flows (login widget, consent screen, OAuth2 redirects) are handled by a separate public-facing frontend that talks to port 8081.

Practical consequences for this codebase:

- All endpoints under `src/services/api/` resolve against the management base URL (`VITE_AUTH_API_BASE_URL`, defaulting to `http://api.maintainerd.auth/api/v1`).
- The console's own login uses the management `/login` endpoint, not the OAuth2 `/oauth/authorize` flow.
- Routes such as `/oauth/token`, `/oauth/authorize`, `/.well-known/*`, and per-client login screens are intentionally **not** part of this app.

---

## Architecture Diagram

```
┌───────────────────────────────────────────────────────────────┐
│                       Browser (SPA)                            │
│                                                                │
│   React Router  →  Pages  →  Components (Radix + Tailwind)     │
│                       │                                         │
│                       ▼                                         │
│                     Hooks                                       │
│           ┌───────────┴───────────┐                             │
│           ▼                       ▼                             │
│   Redux Toolkit slice      TanStack Query                       │
│   (auth, tenant)           (everything else: server state)      │
│           │                       │                             │
│           └───────────┬───────────┘                             │
│                       ▼                                         │
│                Service Layer                                    │
│       (src/services/api/<resource>/index.ts)                    │
│                       │                                         │
│                       ▼                                         │
│              Axios HTTP Client                                  │
│       baseURL + interceptors + ApiError class                   │
└───────────────────────┬───────────────────────────────────────┘
                        │  withCredentials: true (cookie auth)
                        ▼
        maintainerd-auth — Management Port :8080
                        │
                        ▼
              PostgreSQL · Redis · RabbitMQ
```

---

## Request Lifecycle

A typical `GET /users` page render flows like this:

1. The user navigates to `/:tenantId/users`.
2. `App.tsx` matches the route and renders `UsersPage` inside `PrivateLayout`.
3. `UsersPage` calls the `useUsers(params)` hook from `src/hooks/useUsers.ts`.
4. The hook returns a `useQuery` result keyed by `userKeys.list(params)`.
5. On first render, `queryFn` invokes `fetchUsers(params)` from `src/services/api/users/index.ts`.
6. `fetchUsers` builds the URL with `URLSearchParams` and calls `get<ApiResponse<UserListResponseInterface>>(endpoint)`.
7. The Axios instance attaches credentials, sends the request, and on success unwraps `response.data`.
8. The hook returns `{ data, isLoading, error }`; the page renders rows.
9. Errors are normalized to an `ApiError` with `status`, `message`, and `responseData` fields.

Mutations (create/update/delete) follow the same pipeline but additionally call `queryClient.invalidateQueries({ queryKey: userKeys.lists() })` on success.

---

## Entry Point & Bootstrap

`src/main.tsx` mounts the tree with the providers required by every page:

```
<Provider store={store}>
  <BrowserRouter>
    <App />
  </BrowserRouter>
</Provider>
```

`App.tsx` then runs two bootstrap effects:

1. **Auth init** — calls `initializeAuth()` once on mount. This dispatches `initializeAuthAsync`, which hits `/profile` to validate the session cookie and hydrate the Redux `auth` slice.
2. **Tenant init** — runs whenever `location.pathname` or `location.search` changes. `determineTenantIdentifier` extracts the tenant from the URL, then `initializeFromLocation` resolves the tenant via the API (with `lastTenantIdentifierRef` guarding redundant calls).

The `QueryClientProvider` wraps `<Routes>` so every page inside has access to the shared `QueryClient` instance from `src/lib/queryClient.ts`.

---

## Layer-by-Layer Breakdown

### Routing

Routes live in a single `<Routes>` block in `App.tsx` and follow a tenant-scoped pattern:

```
/login                     → public
/register                  → public
/forgot-password           → public
/reset-password            → public
/setup/:step               → first-run bootstrap
/:tenantId/dashboard       → private (tenant-scoped)
/:tenantId/users           → private
/:tenantId/security/*      → private
/:tenantId/branding/*      → private
…
```

Two `PrivateLayout` mount points exist: a default-width one (dashboard, profile) and a `fullWidth` one for management pages with dense tables.

### Pages

`src/pages/<resource>/` follows a consistent shape per CRUD resource:

```
src/pages/users/
├── index.tsx          # list page (table + filters)
├── details.tsx        # read view
├── form.tsx           # create / edit form
└── profile-form.tsx   # nested resource form (where applicable)
```

Pages are declarative — they consume hooks and render UI. They do **not** import from `services/` directly.

### Components

`src/components/` is grouped by purpose, not by feature:

| Folder       | Purpose                                                            |
| ------------ | ------------------------------------------------------------------ |
| `ui/`        | shadcn-style primitives wrapping Radix UI (Button, Dialog, etc.)   |
| `layout/`    | `PrivateLayout`, page shells, sidebar/header composition           |
| `sidebar/`   | Navigation entries and user menu                                   |
| `header/`    | Top bar, tenant switcher, breadcrumbs                              |
| `data-table/`| Generic table built on `@tanstack/react-table`                     |
| `form/`      | Reusable form fields wired to `react-hook-form` + `yup` resolvers  |
| `dialog/`    | Confirmation and form modals                                       |
| `card/`      | Reusable card/section primitives                                   |
| `badges/`    | Status pills (active, suspended, etc.)                             |
| `auth/`      | Auth-specific composites (e.g. profile dropdown)                   |
| `icon/`      | App-specific icon wrappers                                         |
| `container/` | Containerized layout helpers                                       |
| `navigation/`| Tabs, breadcrumb, navigation primitives                            |

### Hooks

`src/hooks/use<Resource>.ts` is where Redux and Query are bridged:

- For **server state** (users, roles, services, etc.), each hook file exports:
  - A `<resource>Keys` factory for query keys.
  - One `useQuery` hook per read endpoint.
  - One `useMutation` hook per write endpoint, each invalidating the right keys on success.
- For **session state** (`useAuth`, `useTenant`), hooks wrap `useAppDispatch` / `useAppSelector` and expose imperative methods backed by Redux thunks.

### Service Layer

`src/services/api/<resource>/index.ts` is the *only* place that talks to HTTP. Each module exports plain async functions returning typed payloads:

```ts
export async function fetchUsers(
  params?: UserQueryParamsInterface
): Promise<UserListResponseInterface> { … }
```

`src/services/index.ts` re-exports the public surface of every resource module. Types live alongside the functions in `<resource>/types.ts`.

### HTTP Client

`src/services/api/client.ts` exposes thin `get` / `post` / `put` / `patch` / `deleteRequest` wrappers around a shared Axios instance configured with:

- `baseURL` from `src/services/api/config.ts` (env-driven).
- `withCredentials: true` so the auth cookie travels on every call.
- A response interceptor that converts every Axios error into an `ApiError` carrying `status`, `code`, and the original `responseData` envelope.

Endpoint paths are centralized in `API_ENDPOINTS` in `src/services/api/config.ts`. No service file should hard-code a path.

### Global State (Redux)

Redux Toolkit is reserved for **session-scoped concerns** that the entire app reacts to:

- `src/store/auth/` — current profile, `isAuthenticated`, `isInitialized`, login/logout/forgot/reset thunks.
- `src/store/tenant/` — currently active tenant and its metadata.

`src/store/hooks.ts` exports the typed `useAppDispatch` and `useAppSelector` aliases.

### Server State (TanStack Query)

Everything else — users, roles, clients, identity providers, policies, templates, settings — is server state and lives in TanStack Query. The `QueryClient` is created in `src/lib/queryClient.ts` and reused across the app.

Query keys follow the hierarchical factory convention (see `userKeys` in `src/hooks/useUsers.ts`) so invalidation can target a precise scope (`detail(id)`, `lists()`, `roles(id)`, etc.).

---

## Multi-Tenancy

The app is multi-tenant by URL: every authenticated page is mounted under `/:tenantId/...`. `determineTenantIdentifier` (`src/utils/tenant.ts`) extracts the identifier from the path or query string and `useTenant` resolves it via the API.

Switching tenants is just a navigation — when `location.pathname` changes, `App.tsx`'s tenant effect re-runs `initializeFromLocation` and the Redux `tenant` slice is updated. Query caches are not tenant-partitioned today; see [`checklist.md`](./checklist.md) for the planned cache-key change.

---

## Authentication Flow

1. The user submits credentials on `/login`.
2. `useAuth().login()` dispatches `loginAsync` → `authService.login()` → `POST /login`.
3. The backend sets an HTTP-only cookie containing the access token.
4. `loginAsync` immediately follows up with `fetchProfile()` so the Redux `auth.profile` is populated.
5. On every subsequent request, Axios sends the cookie automatically (`withCredentials: true`).
6. On app reload, `initializeAuthAsync` calls `validateAuthentication()` to rehydrate the session.

Logout calls `POST /logout`, clears `auth` and `tenant` slices, and the user is redirected to `/login`.

---

## Error Handling

The `ApiError` class is the single error type produced by the service layer:

```ts
class ApiError extends Error {
  status: number
  code?: string
  responseData?: { error, details, success }
}
```

Pages and hooks should narrow on `ApiError` rather than parsing string messages. Toast notifications go through `useToast` (`src/hooks/useToast.ts`) so the user-facing copy stays consistent.

> **Known inconsistency:** several services additionally inspect `response.success` and `throw new Error(response.message)` on failure. This mixes `Error` and `ApiError` instances at the call site — see [`checklist.md`](./checklist.md) for the planned consolidation.

---

## Path Aliases

`@/*` maps to `src/*` in both `tsconfig.json` and `vite.config.ts`. Always import via the alias:

```ts
import { useAuth } from '@/hooks/useAuth'
import { fetchUsers } from '@/services/api/users'
```

Relative imports (`../../services/...`) are reserved for sibling files inside the same feature folder.

---

## Technology Stack

| Layer            | Library                                              |
| ---------------- | ---------------------------------------------------- |
| Framework        | React 19, TypeScript 5.9                             |
| Build tool       | Vite 7                                               |
| Routing          | React Router 7                                       |
| Global state     | Redux Toolkit 2 + react-redux 9                      |
| Server state     | TanStack Query 5                                     |
| HTTP             | axios 1                                              |
| Forms            | react-hook-form 7 + yup 1 (`@hookform/resolvers`)    |
| Tables           | `@tanstack/react-table` 8                            |
| UI primitives    | Radix UI, Tailwind CSS 4, `tw-animate-css`           |
| Icons            | `lucide-react`, `@tabler/icons-react`                |
| Charts           | recharts                                             |
| Date utilities   | `date-fns`, `react-day-picker`                       |
| Toasts           | `react-toastify`                                     |
| Lint             | ESLint 9 + `typescript-eslint`                       |

For the development workflow, environment variables, and proxy setup see [`getting-started.md`](./getting-started.md). For the gap analysis vs. the backend feature set see [`feature-list.md`](./feature-list.md).

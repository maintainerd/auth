# Refactor — Architecture

> **Purpose:** Capture the *structural* changes the codebase needs — the ones that
> aren't a one-line fix but a deliberate re-shaping of layers, boundaries, and
> conventions. Companion to the task-level [`backlog.md`](./backlog.md) and the
> prose [`../checklist.md`](../checklist.md).
>
> **Verdict:** The project does **not** need a ground-up rewrite. The layering
> (`pages → hooks → services/api → client`) is sound. What it needs is **targeted
> re-architecture** in five areas: state-management boundaries, the service layer,
> error/resilience flow, module/feature boundaries, and configuration.

**Audit date:** 2026-06-06

---

## Current architecture (as observed)

```
main.tsx
  └─ <Provider store>            ← Redux Toolkit (auth, tenant)
       └─ <BrowserRouter>
            └─ <App>
                 └─ <QueryClientProvider>   ← TanStack Query (all other server state)
                      └─ <Routes>           ← ~70 EAGERLY imported pages
                           └─ <PrivateLayout> → <ProtectedRoute>
                                └─ page → hook → services/api/<resource> → client(axios)
```

- **Global state:** Redux Toolkit slices for `auth` and `tenant` (thunks + extra-reducers), wrapped by hand-written hooks (`useAuth`, `useTenant`).
- **Server state:** TanStack Query everywhere else, via per-resource hooks (`useUsers`, `useRoles`, …) with query-key factories.
- **HTTP:** single axios instance in `services/api/client.ts` with an interceptor and an `ApiError` class; per-resource `index.ts` modules wrap each endpoint.
- **Forms:** react-hook-form + yup resolvers; schemas in `lib/validations`.

This is a reasonable shape. The problems are *seams*, not foundations.

---

## Problem 1 — Two overlapping state systems with a hand-rolled wrapper layer

**Symptom.** Auth/tenant live in Redux; everything else lives in TanStack Query.
`useAuth` re-implements thunk dispatch + `try/catch` + **silent error swallowing**
(`catch (error) { return false }`, `catch { return null }`), duplicating what
TanStack Query already does for server state. The auth *profile* is server state
being cached in Redux.

**Why it's a problem.** Two caching/loading/error models to reason about; the
wrapper hooks add a third hand-maintained layer where errors get dropped (see
[backlog.md](./backlog.md) P0-3); contributors won't know where new state belongs.

**Target.**

- Adopt one explicit rule and write it into `architecture.md`:
  - **TanStack Query** = *server state* (anything fetched from `maintainerd-auth`), including the auth **profile** and tenant **details**.
  - **Redux** = *client/session state only* (e.g. "is initialized", active-tenant identifier derived from URL, UI prefs) — small, synchronous, non-server.
- Migrate `auth.profile` and tenant lookups to TanStack Query hooks (`useProfile` already exists — consolidate onto it). Keep Redux only for the genuinely-client bits, or drop Redux entirely if nothing client-only remains.
- Delete the hand-rolled try/catch wrappers; let Query's `error`/`isError` carry failures to the error layer (Problem 3).

**Migration sketch.**
1. Introduce `useProfile`/`useSession` Query hooks as the source of truth.
2. Point components at them; leave Redux as a thin shim that reads from Query (or URL) during transition.
3. Remove the auth/tenant thunks + slices once no component reads them.

---

## Problem 2 — Service layer is copy-paste boilerplate

**Symptom.** Every `services/api/<resource>/index.ts` repeats the same three things:
build query string from params, call the verb, then
`if (response.success && response.data) return response.data; throw new Error(...)`.
Plus the client itself fabricates fake `{ success: true }` on empty bodies, and
exposes both named exports *and* an aggregate object (`apiClient`, `authService`).

**Why it's a problem.** ~22 resources × the same boilerplate = high change-cost and
drift; the fabricated-success hides real empty/`204` responses (correctness bug);
two import styles fragment usage.

**Target — a generic resource factory + shared helpers.**

```ts
// services/api/_lib/unwrap.ts
export function unwrap<T>(res: ApiResponse<T>, action: string): T {
  if (res.success && res.data !== undefined) return res.data
  throw new ApiError({ message: typeof res.error === 'string' ? res.error : `Failed to ${action}`, status: 0 })
}

// services/api/_lib/query.ts
export const buildQuery = (params?: Record<string, unknown>) => { /* the existing param loop, once */ }

// services/api/_lib/resource.ts
export function createResourceApi<TRow, TCreate, TUpdate>(base: string) {
  return {
    list:   (p?: QueryParams) => get<ApiResponse<PaginatedResponse<TRow>>>(`${base}${buildQuery(p)}`).then(r => unwrap(r, 'list')),
    get:    (id: string)      => get<ApiResponse<TRow>>(`${base}/${id}`).then(r => unwrap(r, 'fetch')),
    create: (d: TCreate)      => post<ApiResponse<TRow>>(base, d).then(r => unwrap(r, 'create')),
    update: (id, d: TUpdate)  => put<ApiResponse<TRow>>(`${base}/${id}`, d).then(r => unwrap(r, 'update')),
    remove: (id: string)      => deleteRequest<ApiResponse<void>>(`${base}/${id}`).then(r => unwrap(r, 'delete')),
  }
}
```

- Resources with bespoke endpoints (users, auth) compose the factory + a few custom calls.
- Fix the client to return `response.data` verbatim (no synthetic success — [backlog.md](./backlog.md) P0-2).
- Standardize on **named exports**; drop the aggregate objects ([backlog.md](./backlog.md) P2-5).

**Outcome.** A new resource becomes ~20 lines; unwrap/throw/query logic lives in one tested place.

---

## Problem 3 — No resilience or error-propagation backbone

**Symptom.** No `ErrorBoundary`, no `Suspense`, errors swallowed in hooks, the HTTP
interceptor uses `data as any`, and there's no error reporting. A thrown render
error blanks the entire SPA; a failed fetch is indistinguishable from "no data".

**Target — a single error pipeline.**

```
axios interceptor → typed ApiError ─┬─► TanStack Query error state ─► UI (toast / inline)
                                     └─► Error reporter (Sentry-style transport)
React render error ─► <ErrorBoundary> (app-level + per-route) ─► fallback UI + reporter
```

- Top-level `ErrorBoundary` in `App.tsx`; per-route boundary inside `PrivateLayout` so one page can fail without nuking the shell ([backlog.md](./backlog.md) P1-4).
- One `mapApiError(err): string` helper consumed by every `useMutation.onError` and form ([backlog.md](./backlog.md) P2-7).
- Distinguish expected `401`/empty (return `null`, no report) from unexpected errors (report) — kills the silent `catch {}` pattern ([backlog.md](./backlog.md) P0-3).
- Type the interceptor payload instead of `as any` ([backlog.md](./backlog.md) P1-10).

---

## Problem 4 — Feature boundaries leak (mock data + giant components)

**Symptom.** Production page `constants.ts` files ship **fake data** (e.g.
`reyco.seguma`, `jane.doe`, dicebear avatar URLs, ~400 lines of sample log rows,
sample analytics), mixed in the same file as real config constants. Several
components are enormous (`ClientAddOrUpdateForm` 1084 lines, `LoginTemplateForm`
791, `UserProfileForm` 591).

**Why it's a problem.** Demo data is bundled and can render in the real UI; the
"constants" files conflate config and fixtures; 1000-line components can't be
unit-tested or reasoned about.

**Target — a consistent feature-module shape.**

```
pages/<feature>/
  index.ts            ← barrel (default export)
  <Feature>Page.tsx   ← thin: layout + composition, < ~250 lines
  components/          ← presentational sub-parts (split big forms here)
  form/               ← sub-forms + a form hook (extract from the 1000-liners)
  hooks/              ← feature-local hooks
  constants.ts        ← REAL config only — NO fixtures
```

- Evict all fixtures to `test/fixtures/`, MSW handlers, or Storybook — never imported by shipped components ([backlog.md](./backlog.md) P1-6).
- Decompose the oversized forms into section components + a `use<Feature>Form` hook ([backlog.md](./backlog.md) P2-3).
- Remove dicebear runtime calls.

---

## Problem 5 — Configuration & bootstrap are implicit

**Symptom.** Env vars read inline (`import.meta.env.VITE_...`) with a **plaintext-HTTP**
production fallback; `~70` pages imported eagerly; `debug.ts` self-executes on
import and attaches to `window`.

**Target.**

- `src/lib/env.ts`: validate required env once (yup/zod), export a typed `config`; fail fast in prod ([backlog.md](./backlog.md) P0-4, P1-8).
- Code-split routes with `React.lazy` + `Suspense`; keep auth pages eager ([backlog.md](./backlog.md) P1-5).
- Delete `debug.ts` / gate behind explicit opt-in ([backlog.md](./backlog.md) P0-1).

---

## Proposed target architecture

```
main.tsx
  └─ <ErrorBoundary>                         ← app-level resilience (NEW)
       └─ <QueryClientProvider>              ← single source of SERVER state
            └─ [<Provider store>]            ← OPTIONAL: client-only state, or removed
                 └─ <BrowserRouter>
                      └─ <App>  (env validated at startup)
                           └─ <Suspense> + <Routes>   ← LAZY route chunks (NEW)
                                └─ <PrivateLayout> → <ProtectedRoute> → <RouteErrorBoundary>
                                     └─ page → feature hook ──► useQuery/useMutation
                                                                    │
                                  services/api/<resource> = createResourceApi(base)  (NEW factory)
                                                                    │
                                  client.ts (axios) → interceptor → ApiError ──► error reporter (NEW)
```

---

## Sequencing (low-risk → high-value)

| Phase | Goal | Depends on |
|---|---|---|
| **A** | Error backbone: boundaries + reporter + `mapApiError`; stop swallowing | — |
| **B** | Service factory + fix client `unwrap`; single import style | A (error types) |
| **C** | Config: env validation, lazy routes, kill `debug.ts` | — |
| **D** | State consolidation: move profile/tenant to Query, shrink/remove Redux | A, B |
| **E** | Feature-module cleanup: evict fixtures, split giant components | B |

Phases A–C are mechanical and safe to do first; D is the largest behavioral change
and should land only after tests exist ([backlog.md](./backlog.md) P1-1) to catch regressions.

---

## Explicit non-goals

- **Not** switching frameworks, bundler, router, or UI kit — all are modern and appropriate.
- **Not** replacing TanStack Query or react-hook-form/yup.
- **Not** a big-bang rewrite — every phase above is independently shippable behind the existing layer seams.

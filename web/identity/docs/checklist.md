# maintainerd-auth-identity — Architecture & Code-Quality Checklist

A complete, opinionated checklist to take the console from its current "ship-fast" state to a production-grade, well-tested, secure admin app aligned with the standards of the [`maintainerd-auth`](https://github.com/maintainerd/auth) backend. Items already in place are pre-checked; unchecked items are concrete refactors or additions to plan into upcoming milestones.

This document is the *how* counterpart to [`feature-list.md`](./feature-list.md) (which is the *what*) and [`architecture.md`](./architecture.md) (which is the *current shape*).

**Legend**

| Marker | Meaning |
|---|---|
| `[x]` | Done |
| `[~]` | Partially done |
| `[ ]` | To do |
| 🔴 | Critical — fix before fronting the console to real customers |
| 🟡 | High priority |
| 🟢 | Medium / nice-to-have |
| ⚪ | Low / optional |

---

## Table of Contents

1. [Project Layout & Naming Conventions](#1-project-layout--naming-conventions)
2. [Function, Hook & Variable Naming](#2-function-hook--variable-naming)
3. [TypeScript Discipline](#3-typescript-discipline)
4. [Service Layer (HTTP)](#4-service-layer-http)
5. [Error Handling](#5-error-handling)
6. [State Management — Redux vs TanStack Query](#6-state-management--redux-vs-tanstack-query)
7. [Hooks Layer](#7-hooks-layer)
8. [Routing, Layouts & Code-Splitting](#8-routing-layouts--code-splitting)
9. [Multi-Tenancy](#9-multi-tenancy)
10. [Forms & Validation](#10-forms--validation)
11. [UI / Design System](#11-ui--design-system)
12. [Accessibility (WCAG 2.2 AA)](#12-accessibility-wcag-22-aa)
13. [Internationalization & Localization](#13-internationalization--localization)
14. [Testing Strategy](#14-testing-strategy)
15. [Tooling, Lint & Formatting](#15-tooling-lint--formatting)
16. [Build, Bundle & Performance](#16-build-bundle--performance)
17. [CI/CD — Mirror the Backend Pipeline](#17-cicd--mirror-the-backend-pipeline)
18. [Security Scanning & Supply-Chain](#18-security-scanning--supply-chain)
19. [Observability & Error Reporting](#19-observability--error-reporting)
20. [Frontend Security Hygiene](#20-frontend-security-hygiene)
21. [Documentation & Governance](#21-documentation--governance)
22. [Release & Versioning](#22-release--versioning)
23. [Quick-Win Sequence](#23-quick-win-sequence)

---

## 1. Project Layout & Naming Conventions

### Folder structure

- [x] `src/` is split by concern (`pages`, `components`, `hooks`, `services`, `store`, `lib`, `utils`, `types`, `constants`, `styles`, `assets`).
- [x] Path alias `@/*` → `src/*` configured in both `tsconfig.json` and `vite.config.ts`.
- [x] Per-resource colocation in `services/api/<resource>/{index,types}.ts`.
- [x] Per-feature colocation in `pages/<feature>/{index,components,details,form,hooks}/`.
- [x] 🟡 **Standardize folder casing.** All folders MUST be `kebab-case` (already mostly true). Verify on every new feature.
- [x] 🟡 **Resource-folder names must be plural** when they represent collections (`users/`, `roles/`, `clients/`). Settings-style singletons stay singular (`security-settings/`, `session-settings/`).
- [x] 🟡 **Service folders must match the API resource path**, not the UI label. `services/api/auth-client/` should be renamed to `services/api/clients/` to match `/api/v1/clients` and the `pages/clients/` folder. Divergence is a bug magnet.
- [x] 🟡 The placeholder folder `src/services/api/types/` (empty, with a sibling `types.ts` shell) should be removed. Per-resource types live in `services/api/<resource>/types.ts`.
- [x] 🟢 **Adopt a feature-folder convention** for any page that grows beyond ~3 files: `pages/<feature>/{index.tsx,components/,details/,form/,hooks/,types.ts,constants.ts}`. Pattern is partially in place — make it the rule.
- [ ] 🟢 **Where to put what** — codify in `CONTRIBUTING.md`:
  - `src/lib/` — third-party wrappers and framework glue (`queryClient.ts`, `validations/`).
  - `src/utils/` — pure, framework-agnostic helpers (`tenant.ts`, `formatDate.ts`).
  - `src/constants/` — only literal constants (no code, no React).
  - `src/types/` — only globally shared types. Per-resource types stay co-located.

### File naming

| Layer | Convention | Example |
|---|---|---|
| React components | `PascalCase.tsx` | `UserListTable.tsx` |
| Hooks | `camelCase.ts` starting with `use` | `useUsers.ts` |
| Services / utilities / constants | `camelCase.ts` | `client.ts`, `tenant.ts` |
| Types-only files | `types.ts` (per folder) | `services/api/users/types.ts` |
| Index barrels | `index.ts` / `index.tsx` | `services/api/users/index.ts` |
| Tests | mirrors source + `.test.ts(x)` | `useUsers.test.ts` |
| Stories | mirrors source + `.stories.tsx` | `Button.stories.tsx` |

- [ ] 🟡 Document and enforce via ESLint (`eslint-plugin-unicorn`'s `filename-case`).
- [x] 🟡 Avoid `index.tsx` for non-barrel pages; prefer `pages/users/UsersPage.tsx` re-exported via `pages/users/index.ts` so component names appear in stack traces.

### Type & symbol naming

- [x] 🟡 **Drop the `Interface` / `Type` suffix from type names** (`UserType`, `UserListResponseInterface`). Prefer plain `User`, `UserListResponse`. The suffix adds noise without disambiguation. Suffix is retained only for categorical union aliases that aren't backend resources (e.g. `ApiType`, `ProviderType`, `ClientType`, `OnboardingType`, `TemplateType`, `ClientUriType`).
- [x] 🟡 **DTO vs domain naming.** Wire shapes (matching backend payloads exactly) live in `<resource>/types.ts` and use the `Dto` suffix only when a separate domain shape exists. Prefer one shape per resource unless mapping is needed.
- [x] 🟢 Boolean variables prefixed `is`, `has`, `should`, `can` (`isActive`, `hasUnsavedChanges`, `canDelete`).
- [x] 🟢 Event handlers prefixed `handle` for local handlers and `on` for prop names (`<Button onClick={handleSubmit}>`).


## 2. Function, Hook & Variable Naming

### Functions

| Pattern | Use for | Example |
|---|---|---|
| `verbNoun` (action) | mutating / side-effectful functions | `createUser`, `revokeSession` |
| `getNoun` / `fetchNoun` | reads | `getUserById`, `fetchUsersList` |
| `buildNoun` | pure construction | `buildQueryString` |
| `mapNounToOther` | pure transforms | `mapUserDtoToUser` |
| `assertCondition` | invariants | `assertTenantSelected` |
| `selectNoun` | selectors returning a value | `selectActiveTenant` |

- [ ] 🟡 **Service-layer verbs are standardized.** Every CRUD service exposes the same surface so call sites read identically across resources:
  - `list(params)` — paginated read
  - `getById(id)` — single read
  - `create(input)` — POST
  - `update(id, input)` — PUT/PATCH
  - `remove(id)` — DELETE (avoid `delete`, it's reserved)
  - `setStatus(id, status)` — status mutation
- [ ] 🟡 Avoid `data` and `result` as variable names at service boundaries — use the resource name (`user`, `users`, `roles`).
- [ ] 🟡 Async functions explicitly suffix `Async` only when disambiguation is needed (`loginAsync` is acceptable for Redux thunks; plain functions stay unsuffixed).

### Hooks

| Pattern | Use for | Example |
|---|---|---|
| `use<Resource>` | Aggregate hook returning all queries+mutations for a resource | `useUsers()` |
| `use<Resource>List(params)` | Specific list query | `useUsersList(params)` |
| `use<Resource>(id)` | Specific single-item query | `useUser(id)` |
| `useCreate<Resource>` / `useUpdate<Resource>` / `useDelete<Resource>` | Single-purpose mutation hooks | `useCreateUser` |
| `use<Feature>` | Feature-level UI hook (no service call) | `useDebouncedSearch` |

- [ ] 🟡 **Pick one shape per resource hook file.** The current pattern (`useUsers.ts` exporting many sub-hooks like `useUsersList`, `useUser`, `useCreateUser`) is good — replicate it everywhere consistently. Don't mix "fat aggregate hook" and "many small hooks" in the same file.
- [ ] 🟡 Hook return shape MUST be an object, never a positional array — improves diff-friendliness and forward compatibility.

### Variables & constants

- [ ] 🟢 `SCREAMING_SNAKE_CASE` only for top-level literal constants exported from `src/constants/` or `src/services/api/config.ts`.
- [ ] 🟢 Enums replaced with `as const` objects + `keyof typeof` unions. TypeScript enums are discouraged (tree-shaking + `verbatimModuleSyntax` issues).
- [ ] 🟢 Magic numbers and strings extracted into named constants (`DEFAULT_PAGE_SIZE`, `MAX_LOGIN_ATTEMPTS`).

### Components

- [ ] 🟡 Component **default-export** only for top-level page components routed by React Router. All other components use **named exports** so refactor tools rename consistently.
- [ ] 🟡 Props type named `<ComponentName>Props`, defined immediately above the component, never inlined in the function signature for non-trivial components.
- [ ] 🟢 Avoid `React.FC` — define component signature directly: `export function Button(props: ButtonProps) { ... }`.

---

## 3. TypeScript Discipline

- [x] `strict: true`, `noUnusedLocals`, `noUnusedParameters`, `noFallthroughCasesInSwitch`, `verbatimModuleSyntax` already enabled in `tsconfig.app.json`.
- [ ] 🔴 **Eliminate every `any`.** Concrete offenders:
  - `src/services/api/client.ts:42` — `error.response.data as any` → introduce `BackendErrorPayload` type.
  - `src/services/api/client.ts:95,103,119` — `data?: any` on `post`/`put`/`patch` → use `unknown` or a generic `<TBody>`.
  - `src/store/auth/actions.ts:30,43,92,109` — `catch (error: any)` → `catch (error: unknown)` and narrow on `error instanceof ApiError`.
- [ ] 🟡 Add `noUncheckedIndexedAccess: true` and `exactOptionalPropertyTypes: true` to `tsconfig.app.json`.
- [ ] 🟡 Add a `typecheck` npm script (`tsc -b --noEmit`) so CI can fail fast without producing a build artifact.
- [ ] 🟡 Forbid non-null assertions (`!`) outside narrow, documented zones (test setup) via `@typescript-eslint/no-non-null-assertion: error`.
- [ ] 🟡 Replace `as` casts with type predicates / narrowing helpers (`isApiError(e)`, `assertIsUser(x)`).
- [ ] 🟢 Adopt `@typescript-eslint/consistent-type-imports` to keep type-only imports erasable.
- [ ] 🟢 Centralize Axios body/response generics once `BackendEnvelope<T>` shape is settled (see §4).
- [ ] 🟢 Generate types from a backend OpenAPI spec (`openapi-typescript`) — replaces every hand-written `<resource>/types.ts` file.


---

## 4. Service Layer (HTTP)

### What's already there

- [x] Single Axios instance with response interceptor that emits `ApiError`.
- [x] `withCredentials: true` for cookie-auth.
- [x] Endpoint paths centralized in `API_ENDPOINTS` (`services/api/config.ts`).
- [x] Typed wrappers `get` / `post` / `put` / `patch` / `deleteRequest`.

### What needs fixing

- [ ] 🔴 **Eliminate the duplicated CRUD boilerplate.** Every resource module re-implements the same shape:

  ```ts
  const queryParams = new URLSearchParams()
  if (params) Object.entries(params).forEach(([k, v]) => …)
  const endpoint = `${API_ENDPOINTS.X}${qs ? `?${qs}` : ''}`
  const response = await get<ApiResponse<…>>(endpoint)
  if (response.success && response.data) return response.data
  throw new Error(response.message || '…')
  ```

  Replace with a generic factory in `src/services/api/factory.ts`:

  ```ts
  export function createResourceService<TList, TItem, TCreate, TUpdate>(opts: {
    base: string
    statusPath?: string
  }): ResourceService<TList, TItem, TCreate, TUpdate>
  ```

  → produces `list`, `getById`, `create`, `update`, `remove`, `setStatus`. Hand-written services are reserved for endpoints that don't fit the pattern (nested resources, bulk endpoints).

- [ ] 🟡 **Stop unwrapping `ApiResponse<T>` inside every service.** Either (a) make the backend always return `T` and remove the envelope, or (b) unwrap once in the response interceptor. Today every service contains the same `if (response.success && response.data) return response.data; throw new Error(...)` block.
- [ ] 🟡 A typed `buildQuery(params: Record<string, unknown>)` helper to replace the inline `URLSearchParams` loop. Lives in `src/services/api/query.ts`.
- [ ] 🟡 Remove the `apiClient` re-export object — it duplicates the named exports and obscures tree-shaking.
- [ ] 🟡 Rename `services/api/auth-client/` → `services/api/clients/` (matches backend route + UI folder).
- [ ] 🟡 Add a request interceptor that attaches `X-Tenant-Id` (or similar) from the active-tenant store, so service files don't have to know about tenancy.
- [ ] 🟢 Per-call abort signal support so list searches can cancel in-flight requests on input change.
- [ ] 🟢 Request/response logging in dev only, behind a feature flag (`VITE_DEBUG_HTTP`).
- [ ] 🟢 Replace hand-written types with generated types from a backend OpenAPI spec (`openapi-typescript` → `src/services/api/_generated/schema.d.ts`).

### Per-resource refactor checklist

For every folder under `src/services/api/<resource>/` (currently 25 of them), a refactor PR should:

- [ ] Migrate `index.ts` to `createResourceService(...)` from the factory.
- [ ] Move query keys into `keys.ts` co-located with the service.
- [ ] Drop unused fields from `types.ts`; remove `Type`/`Interface` suffixes.
- [ ] Update `src/hooks/use<Resource>.ts` to use `createCrudHooks(keys, service)` (see §7).
- [ ] Add a Vitest spec covering the service (`services/api/<resource>/index.test.ts`).

---

## 5. Error Handling

- [x] `ApiError` class with `status`, `code`, `responseData`.
- [x] Single Axios interceptor produces `ApiError` for every transport-level failure.
- [ ] 🔴 **Fix the inconsistent error path.** Services currently throw both `ApiError` (from the interceptor on 4xx/5xx) **and** plain `Error` (from `if (!response.success) throw new Error(...)` on 2xx-with-success-false). Hooks then have to handle two shapes.
  → Either treat `success: false` as an error in the interceptor (preferred), or wrap it in a `BusinessError` class. Document the convention in `architecture.md`.
- [ ] 🔴 **Auto-redirect on 401** from the interceptor — clear Redux auth slice, clear Query cache, navigate to `/login`. Must happen exactly once even when several requests 401 in parallel.
- [ ] 🟡 A single `useErrorToast(error)` hook that knows how to render either error shape with a fallback message.
- [ ] 🟡 Route-level `<ErrorBoundary />` (e.g. `react-error-boundary`) with a friendly fallback UI; reset on route change.
- [ ] 🟡 Network-offline detector with a global banner driven by `navigator.onLine`.
- [ ] 🟡 Surface backend `request_id` from `ApiError.responseData` in error toasts and console logs so support can correlate.
- [ ] 🟢 Exponential-backoff retry on idempotent reads (configure in `QueryClient` defaults; today there's no central config).
- [ ] 🟢 Differentiate user-actionable errors (4xx → toast) from system errors (5xx → boundary + report) in a single `mapErrorToUI(err)` helper.

---

## 6. State Management — Redux vs TanStack Query

- [x] Redux Toolkit for session-scoped global state (`auth`, `tenant`).
- [x] TanStack Query for server state (per-resource).
- [ ] 🟡 **Move `auth` to TanStack Query** if you want a single state model. A `useSession()` hook backed by `useQuery(['session'])` plus a `setQueryData` on login/logout achieves the same result with less ceremony than Redux thunks.
- [ ] 🟡 If keeping Redux, **remove duplication between thunks and Query hooks.** Pick one entry point per action and expose it from `useAuth()`.
- [ ] 🟡 Decide on the `tenant` slice: keep as Redux (current) or expose as a Query plus a router-driven `useActiveTenant()`. Don't keep both.
- [ ] 🟡 **Standardize the `<resource>Keys` factory shape across all hook files.** The factory already in `useUsers.ts` is the model — replicate everywhere:

  ```ts
  export const userKeys = {
    all: ['users'] as const,
    lists: () => [...userKeys.all, 'list'] as const,
    list: (params: UserListParams) => [...userKeys.lists(), params] as const,
    details: () => [...userKeys.all, 'detail'] as const,
    detail: (id: string) => [...userKeys.details(), id] as const,
  }
  ```

- [ ] 🔴 **Tenant-aware Query keys.** Most keys today are `['users', 'list', params]` — they don't include the active tenant, so switching tenants can briefly show stale data. Prefix every key with the active tenant id: `[tenantId, 'users', 'list', params]`. Easiest approach: have the factory receive `tenantId` as a parameter and call it from a `useScopedQueryKeys()` hook.
- [ ] 🟡 Configure `QueryClient` defaults centrally (`staleTime`, `gcTime`, `retry`, `refetchOnWindowFocus`) in `src/lib/queryClient.ts` instead of relying on per-call defaults.
- [ ] 🟢 Persist Query cache to `sessionStorage` (`@tanstack/query-sync-storage-persister`) for snappier tenant switches.
- [ ] ⚪ Consider replacing Redux entirely with Zustand if Redux only ever holds 2 slices.


---

## 7. Hooks Layer

- [x] One `use<Resource>.ts` per service module.
- [x] Mutation `onSuccess` invalidates the right keys.
- [ ] 🟡 **Extract a `createCrudHooks(keys, service)` helper** so each `useUsers.ts` / `useRoles.ts` / etc. is ~10 lines instead of ~300. Pair with the service factory in §4 — the two refactors compound.
- [ ] 🟡 Co-locate the `<resource>Keys` factory next to the service module (`src/services/api/<resource>/keys.ts`) so hooks and services share one source of truth.
- [ ] 🟡 Provide `onMutate` optimistic-update defaults for status toggles (activate / deactivate / suspend) — currently every mutation refetches.
- [ ] 🟢 A `useDebouncedQuery(params, delay)` wrapper that builds on the existing `useDebouncedSearch` for list pages.
- [ ] 🟢 A shared `useResourceTable(resource, columns)` hook that unifies the list page pattern (search + filter + paginate + table state) — most pages re-implement this.

---

## 8. Routing, Layouts & Code-Splitting

- [x] `<Routes>` declared once in `App.tsx`.
- [x] `PrivateLayout` shared across protected routes (with a `fullWidth` variant).
- [ ] 🔴 **Replace placeholder routes** that point at `<DashboardPage />`: `events`, `webhooks`, `branding` (root). They silently render the dashboard today.
- [ ] 🟡 **Extract the route table to `src/routes.tsx`** (array of `{ path, element, layout, requires }`). `App.tsx` is already 200+ lines and any new feature adds 4 imports.
- [ ] 🟡 **Lazy-load route components** with `React.lazy` + `<Suspense>` per top-level feature; today every page is in the initial bundle.
- [ ] 🟡 A real `<RequireAuth />` guard component instead of relying on bootstrap effects to gate access.
- [ ] 🟡 A real `<RequireTenant />` guard for `/:tenantId/...` that redirects when the tenant fails to resolve.
- [ ] 🟡 A `<RequirePermission permissions={[...]} />` guard for permission-gated routes (e.g. tenant settings).
- [ ] 🟢 404 page mounted on `path="*"`.
- [ ] 🟢 Persist last-visited tenant in `localStorage` so `/` redirects to `/<lastTenant>/dashboard` instead of `/login` for returning users.
- [ ] 🟢 Reflect filter / search / page state in URL query strings so list pages are shareable.

---

## 9. Multi-Tenancy

- [x] Tenant resolved from URL path / query.
- [x] Tenant cached in Redux to avoid re-fetching.
- [ ] 🔴 Encode the active tenant in **Query keys** (see §6).
- [ ] 🟡 Encode the active tenant in **request headers** via Axios interceptor (services stop caring about it).
- [ ] 🟡 Resolve the `:tenantId` URL segment **once** via a route loader; today both `App.tsx` and individual pages re-derive it.
- [ ] 🟡 Tenant switcher with explicit "switch tenant" action that clears Query caches and navigates.
- [ ] 🟢 Cross-tenant guard rails: warn if a stored Query key doesn't match the URL tenant after navigation.

---

## 10. Forms & Validation

- [x] `react-hook-form` + `yup` resolvers via `@hookform/resolvers`.
- [ ] 🟡 Extract shared field components (`<TextField>`, `<SelectField>`, `<SwitchField>`, `<TextareaField>`, `<DatePickerField>`) so forms compose primitives instead of re-wiring `Controller` per field.
- [ ] 🟡 Centralize validation messages and per-field copy (i18n-ready) in `src/lib/validations/messages.ts`.
- [ ] 🟡 A consistent submit handler that maps `ApiError.responseData.details` → per-field errors via `setError`.
- [ ] 🟡 Adopt `zod` (or stay on `yup`) and pick one — don't have both. Document in `architecture.md`.
- [ ] 🟢 Optimistic UI on simple toggles (status, default-flag).
- [ ] 🟢 Form-level dirty-tracking so navigation away prompts "unsaved changes".
- [ ] 🟢 Reusable confirm-dialog for destructive submits (delete / revoke / rotate).

---

## 11. UI / Design System

- [x] Radix UI primitives wrapped in `components/ui/`.
- [x] Tailwind CSS 4 + `tw-animate-css`.
- [x] Reusable `data-table`, `dialog`, `card`, `header`, `sidebar`, `navigation`, `badges`.
- [ ] 🟡 Audit `components/ui/` for components that drifted from shadcn defaults; document deviations in `components/ui/README.md`.
- [ ] 🟡 Storybook (or Ladle) for `components/ui/` + form fields. Acts as the design-system source of truth.
- [ ] 🟡 Empty-state component used consistently across list pages (today some pages render nothing for empty results).
- [ ] 🟡 Loading-skeleton component family (table-row skeleton, card skeleton) — replace ad-hoc spinners.
- [ ] 🟢 Dark-mode theme provider + toggle (Tailwind already configured for it).
- [ ] 🟢 Design tokens (color, spacing, radius) centralized in CSS variables and re-exposed via Tailwind config.
- [ ] 🟢 Icon strategy — pick one of `lucide-react` or `@tabler/icons-react`; both are imported today.

---

## 12. Accessibility (WCAG 2.2 AA)

- [ ] 🟡 Manual + automated **WCAG 2.2 AA** pass over key flows (login, setup, user CRUD, security pages).
- [ ] 🟡 Keyboard-navigation pass: modal focus traps, focus return on close, `tabindex` hygiene, skip-to-content link in `PrivateLayout`.
- [ ] 🟡 ARIA labels for all icon-only buttons (sidebar, table action menus).
- [ ] 🟡 `prefers-reduced-motion` respected for any non-essential animations.
- [ ] 🟡 Form fields have programmatically associated labels (RHF `Controller` + `<Label htmlFor>`).
- [ ] 🟡 Color-contrast audit of all status badges + sidebar selected state (4.5:1 min for body, 3:1 for large text / UI).
- [ ] 🟢 `axe-core` integrated into Vitest + Playwright (`@axe-core/playwright`); fail CI on violations.
- [ ] 🟢 RTL support audit (Tailwind logical properties, `dir="rtl"`).
- [ ] 🟢 Screen-reader pass on the dashboard and at least one list page (NVDA + VoiceOver).

---

## 13. Internationalization & Localization

- [ ] 🟡 Adopt an i18n framework (`react-intl` or `i18next`) with extraction tooling, even if only `en` ships initially. Without it, retrofitting later is painful.
- [ ] 🟡 All user-visible strings extracted to a message catalog. Hardcoded English in JSX is forbidden via ESLint.
- [ ] 🟢 Number / date / currency formatting via `Intl.*` instead of ad-hoc `toFixed` / template strings.
- [ ] 🟢 Locale stored per-user (preferred), fall back to browser locale.
- [ ] ⚪ Pluralization + ICU message format support.


---

## 14. Testing Strategy

The repo currently has **zero `*.test.ts(x)` files** — testing is the single biggest gap.

### Stack to adopt

| Layer | Tool |
|---|---|
| Unit + component tests | **Vitest** + **@testing-library/react** + **@testing-library/jest-dom** |
| HTTP mocking | **MSW** (Mock Service Worker) — same handlers reused in tests + Storybook |
| End-to-end | **Playwright** |
| Accessibility | **@axe-core/playwright** + **vitest-axe** |
| Visual regression (later) | **Chromatic** or **Percy** via Storybook |

### Coverage targets (CI gates)

| Tier | Threshold | Layers covered |
|---|---|---|
| Tier 1 (must pass) | 70% statements, 60% branches | Service factory, error mapping, hook factory, query keys |
| Tier 2 (should pass) | 50% on touched files | Per-resource hooks, primitive UI components |
| Tier 3 (nice-to-have) | smoke E2E green | Login, setup wizard, user CRUD, role assignment |

### What to write first (in order)

- [ ] 🔴 `src/services/api/client.test.ts` — 200/4xx/5xx/network/timeout paths produce the correct `ApiError`.
- [ ] 🔴 `src/services/api/factory.test.ts` (after the factory exists) — covers list/get/create/update/remove/setStatus uniformly. One test file then implicitly covers most resources.
- [ ] 🔴 `src/lib/queryClient.test.ts` — defaults are applied, retry policy, gc time.
- [ ] 🟡 `src/hooks/useAuth.test.ts` — login, logout, session bootstrap on mount, 401 handling.
- [ ] 🟡 Hook tests for `useUsers`, `useRoles`, `useClients` (smoke) — pattern replicated to remaining resources.
- [ ] 🟡 Integration test for `App.tsx` bootstrap effects (auth init, tenant init).
- [ ] 🟡 Component tests for `components/ui/` primitives + form-field wrappers.
- [ ] 🟡 **Playwright E2E** covering: setup wizard, login, user CRUD, role assignment, password-policy edit, OAuth client create.
- [ ] 🟢 Visual regression (Chromatic / Percy) on critical pages once Storybook lands.
- [ ] 🟢 Mutation testing (`stryker`) on the service factory once stable.

### Conventions

- [ ] 🟡 Test files co-located with source (`Foo.tsx` ↔ `Foo.test.tsx`).
- [ ] 🟡 No conditional logic in tests — one assertion per behavior, AAA layout (Arrange / Act / Assert).
- [ ] 🟡 MSW handlers live in `src/test/mocks/handlers/<resource>.ts` and are imported by both Vitest and Storybook.
- [ ] 🟡 Test data factories in `src/test/factories/` (e.g. `userFactory.ts`) — never hand-roll fixtures inline.

---

## 15. Tooling, Lint & Formatting

### Today

- [x] ESLint 9 with `typescript-eslint`, `react-hooks`, `react-refresh`.
- [x] Strict TS via `tsc -b`.

### Add

- [ ] 🟡 **`typecheck` script** — `tsc -b --noEmit`. Used by CI to fail on types without producing a build.
- [ ] 🟡 **Prettier** + an `.editorconfig`. Pick exactly one of (Prettier) or (ESLint stylistic) and stick with it. Recommend Prettier + `eslint-config-prettier`.
- [ ] 🟡 **Pre-commit hooks** via `husky` + `lint-staged` (lint + typecheck + format on changed files). Mirrors the backend's `pre-commit` discipline.
- [ ] 🟡 **`commitlint` + Conventional Commits.** Backend uses Conventional Commits — frontend should too.
- [ ] 🟡 Stricter ESLint rules (add to `eslint.config.js`):
  - `@typescript-eslint/consistent-type-imports: error`
  - `@typescript-eslint/no-explicit-any: error`
  - `@typescript-eslint/no-non-null-assertion: error`
  - `@typescript-eslint/no-floating-promises: error` (requires type-aware lint config)
  - `react/jsx-no-leaked-render: error`
  - `import/order` (alphabetized, grouped, newlines between groups)
  - `no-restricted-imports` — forbid `services/*` imports from inside `pages/*` (must go through hooks)
  - `unicorn/filename-case` — enforce kebab-case folders / camelCase files / PascalCase components
- [ ] 🟢 `eslint-plugin-tailwindcss` for class-name ordering and unknown-class detection.
- [ ] 🟢 `eslint-plugin-jsx-a11y` for accessibility lint rules.
- [ ] 🟢 `knip` or `ts-prune` to find unused exports (e.g. the empty `services/api/types.ts` shell, the `apiClient` re-export).
- [ ] 🟢 `tsconfig` `noUncheckedIndexedAccess: true` + `exactOptionalPropertyTypes: true`.

---

## 16. Build, Bundle & Performance

- [x] Vite 7 with React 19.
- [ ] 🟡 Code-split per route via `React.lazy` (see §8).
- [ ] 🟡 Vendor-chunk strategy in `vite.config.ts` — split `react`, `recharts`, `@radix-ui/*`, `react-day-picker` into their own chunks.
- [ ] 🟡 Bundle-size budget check in CI (`rollup-plugin-visualizer` produces a stats file; compare against a baseline).
- [ ] 🟡 Stamp build with `VITE_BUILD_SHA` and `VITE_BUILD_TIME` so support can pin a regression to a commit.
- [ ] 🟢 Image optimization pipeline for `public/` assets (e.g. `vite-imagetools`).
- [ ] 🟢 Preload / prefetch hints for the post-login dashboard.
- [ ] 🟢 HTTP/2 push hints via the host (Nginx `http2_push_preload`).
- [ ] ⚪ Brotli pre-compression of `dist/` assets in CI.


---

## 17. CI/CD — Mirror the Backend Pipeline

The backend ships three GitHub Actions workflows: `ci.yml`, `security.yml`, `scorecard.yml`. The console must reach **feature parity** with these — same job names, same triggers, same SARIF uploads, same Codecov gate. Below is a concrete recipe.

### Pipeline overview

```
.github/
└── workflows/
    ├── ci.yml         # lint · typecheck · test · build (every PR + push)
    ├── security.yml   # CodeQL · Semgrep · Snyk · Gitleaks (every PR + nightly cron)
    └── scorecard.yml  # OpenSSF Scorecard (weekly + main pushes)
```

### `ci.yml` — required jobs (mirrors backend)

| Job | Purpose | Fails the build when |
|---|---|---|
| `lint` | ESLint over `src/` | any lint error |
| `typecheck` | `tsc -b --noEmit` | any TS error |
| `test` | Vitest with coverage → upload to **Codecov** | tests fail, coverage below threshold |
| `build` | `npm run build` (production bundle) | bundle fails or exceeds size budget |
| `e2e` (optional, gated) | Playwright against a built preview | E2E suite red |

- [ ] 🔴 `.github/workflows/ci.yml` with the four jobs above. Triggered on `push` to `main`/`develop` and every `pull_request`.
- [ ] 🔴 `actions/setup-node@v4` with `node-version: 20` and `cache: 'npm'`.
- [ ] 🔴 `npm ci` (not `npm install`) for reproducibility.
- [ ] 🔴 **Codecov upload** via `codecov/codecov-action@v5` with `fail_ci_if_error: true` and `flags: unittests` — same convention as backend.
- [ ] 🟡 `concurrency: group: ci-${{ github.ref }} cancel-in-progress: true` to cancel stale runs on force-push.
- [ ] 🟡 Cache Vite/Vitest output (`~/.cache/vite`, `node_modules/.vite`).
- [ ] 🟡 `permissions: contents: read` at the top of the workflow (same hardening as backend).
- [ ] 🟡 Job matrix is unnecessary for frontend (single Node 20 LTS); avoid the temptation to over-matrix.

### Recommended `ci.yml` skeleton

```yaml
name: CI

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

permissions:
  contents: read

concurrency:
  group: ci-${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

jobs:
  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: 20, cache: npm }
      - run: npm ci
      - run: npm run lint

  typecheck:
    name: Typecheck
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: 20, cache: npm }
      - run: npm ci
      - run: npm run typecheck

  test:
    name: Test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: 20, cache: npm }
      - run: npm ci
      - run: npm run test -- --coverage --run
      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v5
        with:
          files: coverage/lcov.info
          flags: unittests
          name: maintainerd-auth-identity
          fail_ci_if_error: true
        env:
          CODECOV_TOKEN: ${{ secrets.CODECOV_TOKEN }}

  build:
    name: Build
    runs-on: ubuntu-latest
    needs: [lint, typecheck, test]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: 20, cache: npm }
      - run: npm ci
      - run: npm run build
      - name: Bundle size report
        run: npx vite-bundle-visualizer --json -o bundle-stats.json || true
      - uses: actions/upload-artifact@v4
        with: { name: dist, path: dist, retention-days: 7 }
```

### Branch protection on `main`

- [ ] 🔴 Require PR reviews (minimum 1).
- [ ] 🔴 Require status checks to pass: `lint`, `typecheck`, `test`, `build`, `CodeQL`, `Semgrep`, `Snyk`, `Gitleaks`.
- [ ] 🔴 Require branches to be up to date before merging.
- [ ] 🔴 Require signed commits (matches backend).
- [ ] 🟡 Require linear history (no merge commits).
- [ ] 🟡 Dismiss stale reviews on new commits.

### Preview & deploy

- [ ] 🟡 **Preview deploys per PR** (Vercel / Cloudflare Pages / Netlify). Comment the preview URL on the PR.
- [ ] 🟡 **Production deploy from `main`** with build-info baked into the bundle (`VITE_BUILD_SHA`, `VITE_BUILD_TIME`).
- [ ] 🟡 Optional `release.yml` workflow that creates a GitHub Release with auto-generated changelog (Conventional Commits → `release-please` or `changesets`).
- [ ] 🟢 **Lighthouse CI** on key pages (login, dashboard, user list). Fail PR if scores drop below baseline.
- [ ] 🟢 **OpenSSF Best Practices** badge once CI is green for 30 days. The README already references the in-progress badge.


---

## 18. Security Scanning & Supply-Chain

The backend's `security.yml` runs **CodeQL · Semgrep · Snyk · Gitleaks** on every PR and nightly. The console must run an equivalent stack — same job names where possible, same nightly cron, same SARIF uploads to GitHub Code Scanning.

### `security.yml` jobs

| Job | Tool | What it scans |
|---|---|---|
| `codeql` | GitHub CodeQL (`javascript-typescript`) | semantic SAST for the JS/TS source tree |
| `semgrep` | Semgrep CI | OWASP Top 10, JWT, XSS, secrets, react-specific rulesets |
| `snyk` | Snyk Open Source + Snyk Code | npm dependencies + first-party JS code |
| `gitleaks` | Gitleaks CLI | secrets across **full git history** (`fetch-depth: 0`) |
| `npm-audit` | `npm audit --omit=dev` | npm advisory DB on prod deps |
| `osv-scanner` | OSV-Scanner | additional supply-chain coverage on `package-lock.json` |

- [ ] 🔴 `.github/workflows/security.yml` mirroring the backend file structure (same triggers: `push`, `pull_request`, daily `02:00 UTC` cron).
- [ ] 🔴 All scan jobs upload SARIF to `github/codeql-action/upload-sarif@v4` so results land in the **Security → Code Scanning** tab.
- [ ] 🔴 `continue-on-error: true` on the scan steps + an "Ensure SARIF file exists" fallback so the upload step always runs (same pattern as backend Semgrep/Gitleaks jobs).
- [ ] 🔴 `permissions: { contents: read, security-events: write }` at the workflow level.
- [ ] 🔴 **CodeQL config** uses `languages: javascript-typescript` and `queries: security-extended,security-and-quality` — matches backend's posture for Go.
- [ ] 🔴 **Semgrep rulesets** — pull at minimum:
  - `p/javascript`
  - `p/typescript`
  - `p/react`
  - `p/owasp-top-ten`
  - `p/secrets`
  - `p/xss`
  - `p/jwt`
  - `p/insecure-transport`
- [ ] 🔴 **Snyk** with `--severity-threshold=high` + SARIF output, same as backend.
- [ ] 🔴 **Gitleaks** with full-history `fetch-depth: 0`, redacted output, custom `.gitleaks.toml` config copied from the backend.
- [ ] 🟡 **Dependency review** action (`actions/dependency-review-action@v4`) on every PR — fails the PR if a new dep brings in a known vulnerability.
- [ ] 🟡 **OSV-Scanner** GitHub Action as belt-and-suspenders alongside Snyk (different advisory sources catch different vulnerabilities).
- [ ] 🟡 **Trivy** scan against the production Docker image once it's published (matches a likely backend addition).
- [ ] 🟢 **`npm-audit-resolver`** committed to suppress accepted-risk advisories with documented justifications.
- [ ] 🟢 **SBOM generation** on release (CycloneDX via `npm sbom` or `@cyclonedx/cyclonedx-npm`) attached to GitHub Releases.

### `scorecard.yml` — OpenSSF Scorecard

- [ ] 🔴 Copy `scorecard.yml` verbatim from the backend (only the project name needs to change). Triggers: weekly cron + push to `main`.
- [ ] 🔴 `id-token: write` permission for OIDC publishing.
- [ ] 🔴 `publish_results: true` so the score appears at https://scorecard.dev/viewer/?uri=github.com/maintainerd/auth-console.
- [ ] 🟡 Target a baseline score of **8.0+** (matches the backend's posture). Iterate on findings until reached.

### Supply-chain hygiene (codebase)

- [ ] 🟡 Pin GitHub Actions to **commit SHA**, not `@v4` tags (mitigates action repo takeover). Use a Renovate config to keep them current.
- [ ] 🟡 `package.json` `"engines"` field pinned to `node: ">=20"` and `npm: ">=10"`.
- [ ] 🟡 Renovate (or Dependabot) configured with grouping for safe minor/patch updates.
- [ ] 🟡 `.npmrc` with `engine-strict=true`, `fund=false`, `audit=false` (audit is run in CI instead).
- [ ] 🟢 `package-lock.json` is the only lockfile (no committed `yarn.lock` / `pnpm-lock.yaml`).
- [ ] 🟢 Forbid `postinstall` scripts on third-party deps via `--ignore-scripts` in CI installs (`npm ci --ignore-scripts`).

---

## 19. Observability & Error Reporting

- [ ] 🟡 **Frontend error reporting** (Sentry / Bugsnag / Highlight) wired to a route-level error boundary. Capture `ApiError.responseData.request_id` as a tag for correlation with backend logs.
- [ ] 🟡 **Web-Vitals reporter** (LCP / INP / CLS / TTFB) sent to backend or a metrics provider.
- [ ] 🟡 Sourcemaps uploaded to the error tracker on every production build (so stack traces are de-minified).
- [ ] 🟢 User-action breadcrumbs (route changes, button clicks on destructive actions, form submits).
- [ ] 🟢 Custom analytics events for product metrics (PostHog / Mixpanel / Plausible). Honor a per-tenant consent flag.
- [ ] 🟢 `console.log` / `console.warn` stripped from production builds via Vite's `esbuild.drop`.

---

## 20. Frontend Security Hygiene

- [x] HTTP-only cookie session (no `localStorage` tokens).
- [x] `withCredentials: true` on Axios.
- [ ] 🔴 **Strict CSP for production builds** delivered by the host (Nginx). No `unsafe-inline`, no `unsafe-eval`. Style-nonces or hashes for any inline `<style>` injected by Tailwind.
- [ ] 🔴 **Subresource Integrity (SRI)** hashes for any external scripts (today there are none — keep it that way; if any are added, SRI is mandatory).
- [ ] 🔴 **No tokens in URLs** (no `?token=` query strings) — enforced by lint rule + service review.
- [ ] 🟡 **Secret-handling UX** for OAuth client secrets and API keys:
  - Display the secret **once at creation**, never on subsequent reads.
  - Offer "copy" + "I have stored this" confirmation.
  - List views show a masked indicator (`••••••••XYZ`).
  - Rotation flow that revokes the old secret server-side.
- [ ] 🟡 **No `dangerouslySetInnerHTML`** outside vetted template-preview components, and inside those, sanitize via DOMPurify.
- [ ] 🟡 **Click-jacking protection** — `X-Frame-Options: DENY` (or `frame-ancestors 'none'` in CSP) on the production host config.
- [ ] 🟡 Disable the dev `console.log` proxy logger (`vite.config.ts`) in production builds (verify the conditional is correct).
- [ ] 🟡 **Auth state never persisted to `localStorage`/`sessionStorage`** — only Redux in-memory (already true; codify in `architecture.md`).
- [ ] 🟡 **Form autocomplete attributes** correct (`autocomplete="current-password"` / `"new-password"` / `"one-time-code"`).
- [ ] 🟢 `Permissions-Policy` header to disable unused browser features (camera, microphone, geolocation, payment).
- [ ] 🟢 `Referrer-Policy: strict-origin-when-cross-origin`.
- [ ] 🟢 `Cross-Origin-Opener-Policy: same-origin`, `Cross-Origin-Resource-Policy: same-origin`.
- [ ] 🟢 PII in audit/log views masked by default; reveal-on-click with audit trail (matches backend audit posture).
- [ ] 🟢 IP-restriction admin UI must require step-up authentication / re-prompt before saving.


---

## 21. Documentation & Governance

- [x] `docs/architecture.md` — layered architecture, request lifecycle, multi-tenancy.
- [x] `docs/getting-started.md` — local dev environment.
- [x] `docs/feature-list.md` — feature gap analysis.
- [x] `docs/checklist.md` — *this file*.
- [x] `README.md` aligned with backend visual style.
- [x] `CODE_OF_CONDUCT.md`.
- [x] `LICENSE` (Apache License 2.0) and `NOTICE` (original-creator attribution).
- [ ] 🟡 `CONTRIBUTING.md` — branch naming, Conventional Commits, PR template, the conventions in §1 / §2 / §15.
- [ ] 🟡 `SECURITY.md` — vulnerability reporting policy, links to backend security policy, supported versions.
- [ ] 🟡 `.github/PULL_REQUEST_TEMPLATE.md` — checkbox list referencing this checklist where relevant.
- [ ] 🟡 `.github/ISSUE_TEMPLATE/` — bug report, feature request, security report.
- [ ] 🟡 `CODEOWNERS` — at least one reviewer on every PR; security-sensitive paths (`services/api/`, `store/auth/`, `.github/workflows/`) get a dedicated owner.
- [ ] 🟡 Inline JSDoc on shared utilities (`@/utils/tenant`, `@/lib/queryClient`, the upcoming `@/services/api/factory`).
- [ ] 🟢 Storybook deployed alongside the app (Chromatic / GitHub Pages).
- [ ] 🟢 `docs/adr/` (Architecture Decision Records) for material decisions — e.g. *"Why Redux + Query and not Query-only"*, *"Service factory vs hand-written services"*, *"yup vs zod"*.
- [ ] 🟢 `docs/playbooks/` — operator runbooks (rotate a leaked OAuth client secret, switch the active tenant in the URL, recover from a 401-storm).
- [ ] 🟢 `CHANGELOG.md` generated from Conventional Commits.

---

## 22. Release & Versioning

- [ ] 🟡 Adopt **SemVer** for the console (independent from backend versioning, but documented).
- [ ] 🟡 **Conventional Commits** enforced via commitlint (`feat:`, `fix:`, `chore:`, `refactor:`, `docs:`, `test:`, `ci:`, `perf:`, `security:`).
- [ ] 🟡 Auto-generated GitHub Releases via `release-please` or `changesets` driven by Conventional Commits.
- [ ] 🟡 Production Docker image tagged with both git SHA and SemVer tag.
- [ ] 🟢 SBOM attached to every release (CycloneDX).
- [ ] 🟢 Sigstore / cosign signature on the published Docker image.
- [ ] 🟢 Backwards-compat policy documented — at minimum, the console keeps two deprecated route paths working for one minor version after rename.

---

## 23. Quick-Win Sequence

If you want a tight, opinionated rollout, do the items below in order. Each unlocks the next and pays compounding dividends.

### Sprint 1 — Foundation

1. **Service factory + co-located keys** (§4, §6, §7) — collapses ~3,000 lines of duplicated CRUD into ~300.
2. **Normalize error path on `ApiError`** (§5) — unblocks consistent toasts, retries, and the 401-redirect.
3. **Eliminate `any`** in `client.ts` and `store/auth/actions.ts` (§3) — required before strict-er ESLint rules can be enforced.

### Sprint 2 — Tests + CI

4. **Vitest setup + first batch of hook tests** (§14) — once the factory exists, one set of factory tests covers most resources for free.
5. **`.github/workflows/ci.yml`** with `lint · typecheck · test · build` + Codecov upload (§17).
6. **Branch protection on `main`** — require CI to pass + 1 review (§17).

### Sprint 3 — Security parity with backend

7. **`.github/workflows/security.yml`** with CodeQL, Semgrep, Snyk, Gitleaks (§18).
8. **`.github/workflows/scorecard.yml`** copied from backend (§18).
9. **Strict CSP + secret-handling UX** for OAuth clients and API keys (§20).

### Sprint 4 — UX & architecture polish

10. **Route-config + lazy-loaded routes + `RequireAuth`/`RequireTenant`/`RequirePermission` guards** (§8) — removes 100+ lines from `App.tsx` and unlocks code-splitting.
11. **Tenant-aware Query keys + Axios tenant header interceptor** (§6, §9) — fixes staleness and removes tenant plumbing from every call site.
12. **Replace placeholder routes** (`events`, `webhooks`, `branding` root) with real implementations or explicit `<ComingSoon />` pages (§8).

### Sprint 5 — Long tail

13. Storybook + design-system docs (§11).
14. i18n framework (§13).
15. Sentry + Web-Vitals (§19).
16. Playwright E2E for the top-five flows (§14).
17. Prettier + husky + commitlint (§15).
18. OpenSSF Best Practices badge once Scorecard ≥ 8.0 (§18).

Anything not on this list can land incrementally without coordination. The sprints assume one or two engineers and roughly two weeks per sprint at modest pace.

---

*Last updated: 2026-05-08*

# Refinement Backlog — maintainerd-auth-console

> **Purpose:** A prioritized, actionable backlog of everything required to bring this
> project up to best-practice / production-grade standards. Each item lists a
> **priority**, **effort estimate**, the **area**, **why it matters**, and
> **acceptance criteria** so it can be picked up as a standalone task.
>
> This complements the prose audit in [`../checklist.md`](../checklist.md). The
> checklist explains *what good looks like*; this backlog is the *do-list*.
> Structural / architectural items are expanded in
> [`refactor-architecture.md`](./refactor-architecture.md).

**Audit date:** 2026-06-06
**Scope:** `src/` (583 TS/TSX files), build config, tooling, docs.
**Stack:** React 19, TypeScript 5.9, Vite 7, Redux Toolkit, TanStack Query, React Router 7, Tailwind 4, shadcn/ui, react-hook-form + yup, axios.

---

## Legend

| Field | Values |
|---|---|
| **Priority** | `P0` blocker / correctness-or-security · `P1` high · `P2` medium · `P3` nice-to-have |
| **Effort** | `S` < ½ day · `M` ½–2 days · `L` 3–5 days · `XL` > 1 week |

---

## Summary of findings

**What's already good** (keep doing this):

- Clean layered structure: `pages → hooks → services/api → client`.
- Centralized axios instance with a typed `ApiError` class and a response interceptor.
- Cookie-based auth (`withCredentials: true`) — no tokens in `localStorage` (verified: **zero** `localStorage`/`sessionStorage` usage). 
- TanStack Query with a query-key factory pattern (`userKeys`, etc.) and sensible defaults.
- Route protection via a `ProtectedRoute` wrapper around `PrivateLayout`.
- Strict TypeScript flags on (`strict`, `noUnusedLocals`, `noUnusedParameters`, `noFallthroughCasesInSwitch`).
- Per-resource co-located types and a consistent service shape.
- Good in-repo documentation (`architecture.md`, `feature-list.md`, `getting-started.md`, `checklist.md`).

**Top problems** (detailed below):

1. **Zero automated tests** and **no CI** — yet the README advertises CI / coverage / security badges that don't exist.
2. **No error boundaries and no code-splitting** — `App.tsx` eagerly imports ~70 page modules; one render error blanks the whole app.
3. **Mock/dummy data committed into production page code** (fake users, dicebear avatars, sample logs/analytics).
4. **Noisy debug code shipped** — `services/api/debug.ts` auto-logs and attaches `debugApiConnection` to `window`; 40 stray `console.*` calls.
5. **Silent error swallowing** (9+ `catch {}` / `catch → return null`) and **HTTP client fabricates fake success** on empty bodies.
6. **Inconsistent formatting** (mixed tabs/spaces — 15 tab-indented vs 328 space-indented files) with **no Prettier / editorconfig / pre-commit hooks**.
7. **`any` leakage** (32 sites), oversized components (1 file > 1000 lines, several 400–800), and heavy service-layer boilerplate duplication.

---

## P0 — Correctness, security & "the README is lying"

### P0-1 · Remove shipped debug / console noise · `S` · *Tooling/Security*
`src/services/api/debug.ts` runs on import, auto-logs config, and does
`(window as any).debugApiConnection = ...` in dev. Plus **40** `console.*` calls across `src`.
- Delete `debug.ts` (or gate it behind an explicit opt-in dev flag, never auto-run).
- Remove `import './debug'` from `client.ts`.
- Strip stray `console.*`; add the ESLint `no-console` rule (allow `warn`/`error` only).
- **Done when:** `grep -rn "console\." src` returns only intentional `console.error/warn`; no globals attached to `window`.

### P0-2 · Stop fabricating fake success responses in the HTTP client · `S` · *Correctness*
In `src/services/api/client.ts`, every verb does
`return response.data || ({ success: true, message: '...' } as T)`. A genuinely empty
body / `204` now masquerades as a successful payload and lies about its type.
- Return `response.data` as-is; let callers handle empty/`204` explicitly.
- **Done when:** no synthetic `{ success: true }` objects are produced by the client.

### P0-3 · Stop swallowing errors silently · `M` · *Correctness/Observability*
`fetchProfile`, `validateAuthentication`, `initializeAuth`, and several hook
`catch` blocks return `null`/`void` and discard the error. Real failures (network,
500s) become indistinguishable from "not logged in".
- Distinguish **expected** 401/empty (return `null`) from **unexpected** errors (log + surface).
- Route unexpected errors to the error-reporting hook (see P1-7) instead of `catch {}`.
- **Done when:** no bare `catch {}`; each catch either re-throws, reports, or has a comment justifying the swallow.

### P0-4 · Harden the production API base-URL fallback · `S` · *Security/Config*
`config.ts` falls back to `http://api.maintainerd.auth/api/v1` (plain HTTP) in
production when the env var is missing.
- Require `VITE_AUTH_API_BASE_URL` in prod (fail fast / build-time check), or fall back to an HTTPS URL only.
- Add a small env-validation module (see P1-8).
- **Done when:** a prod build with no env var fails loudly rather than silently calling plaintext HTTP.

### P0-5 · Reconcile README badges with reality · `S` · *Docs/Trust*
The README links CI, security, Scorecard, OpenSSF, and Codecov badges, but there
is **no `.github/` directory** and **no tests**. The badges are misleading.
- Either implement the pipelines (P1-1, P1-2, P1-3) or remove/mark the badges as planned.
- **Done when:** every badge in the README points at something that actually runs.

### P0-6 · Add a catch-all `404` route · `S` · *Correctness/UX*
`App.tsx` `<Routes>` has no fallback; unknown URLs render a blank page.
- Add `<Route path="*" element={<NotFound />} />` (public) and a tenant-scoped variant.
- **Done when:** any unmatched URL renders a proper Not-Found screen.

---

## P1 — Quality gates, resilience & developer-experience

### P1-1 · Introduce a test stack and write the first tests · `L` · *Testing*
There are **zero** test files. Add Vitest + React Testing Library + MSW (mock API).
- Unit-test: HTTP client interceptor (P0-2/P0-3 behavior), `utils/tenant`, query-key factories, yup schemas.
- Component-test the auth flows (login, register, reset-password) and one CRUD page end-to-end against MSW.
- **Done when:** `npm test` runs, MSW mocks the API, and critical paths (auth + 1 CRUD resource) are covered.

### P1-2 · Stand up CI · `M` · *CI/CD*
Create `.github/workflows/ci.yml`: install → typecheck (`tsc -b`) → `eslint .` →
`vitest run --coverage` → `vite build`. Gate PRs on it.
- **Done when:** PRs cannot merge red; coverage is uploaded.

### P1-3 · Stand up security scanning · `M` · *Security/CI*
`security.yml`: `npm audit` / OSV scan, CodeQL (JS/TS), dependency review,
secret scanning. Add OpenSSF Scorecard workflow (referenced by README).
- **Done when:** dependency + code scanning run on PR and on a schedule.

### P1-4 · Add error boundaries (+ route-level fallback) · `M` · *Resilience*
No `ErrorBoundary` / `Suspense` anywhere. A single thrown render error blanks the app.
- Add a top-level error boundary in `App.tsx` and per-route boundaries in `PrivateLayout`.
- Wire to the error reporter (P1-7).
- **Done when:** a thrown error in any page shows a recoverable fallback, not a white screen.

### P1-5 · Code-split routes with `React.lazy` · `M` · *Performance*
`App.tsx` statically imports ~70 page modules, so the initial bundle pulls in the
entire admin console up front.
- Convert page imports to `React.lazy` + `<Suspense>`; keep auth pages eager.
- **Done when:** route chunks are split (verify in `vite build` output); initial JS shrinks materially.

### P1-6 · Extract all mock/dummy data out of production code · `M` · *Correctness/Hygiene*
Page `constants.ts` files ship fake records — `reyco.seguma`, `jane.doe`, dicebear
avatars (`https://api.dicebear.com/...`), sample analytics, and ~400 lines of sample
log rows. These get bundled and can render in the real UI.
- Move sample data into test fixtures / Storybook / MSW handlers — never imported by shipped components.
- Split "real config constants" from "demo data" in those files.
- **Done when:** no component imports fake user/log/analytics records; dicebear calls removed from runtime.

### P1-7 · Add an error-reporting / observability hook · `M` · *Observability*
There is no Sentry/equivalent. Combined with P0-3, failures vanish.
- Add a thin reporting wrapper (Sentry or a custom transport) used by error boundaries and the HTTP interceptor.
- **Done when:** unexpected errors are captured with context (tenant, route, request id).

### P1-8 · Validate environment variables at startup · `S` · *Config*
Env vars are read raw (`import.meta.env.VITE_...`) with silent fallbacks.
- Add `src/lib/env.ts` that parses/validates required vars (yup/zod) once and exports a typed config.
- **Done when:** missing/malformed env fails fast with a clear message; no scattered raw `import.meta.env` reads.

### P1-9 · Replace `window.confirm` with the existing AlertDialog · `S` · *UX/Accessibility*
`UserProfileForm.tsx` uses `window.confirm(...)` for a destructive delete while the
project already ships `@radix-ui/react-alert-dialog`.
- Use the accessible `AlertDialog` for all destructive confirmations.
- **Done when:** no `window.confirm`/`alert` in `src`.

### P1-10 · Eliminate `any` and tighten lint · `M` · *TypeScript*
**32** `any` sites including the interceptor's `data as any`, `error: any` in thunks,
and `(window as any)`.
- Type the interceptor's error payload; type thunk `catch (error: unknown)` with a narrowing helper.
- Add ESLint `@typescript-eslint/no-explicit-any` (warn → error) and fix the `ecmaVersion` mismatch (config says `2020`, target is `ES2022`).
- **Done when:** `any` count → ~0; lint enforces it.

---

## P2 — Consistency, structure & maintainability

### P2-1 · Add Prettier + editorconfig + pre-commit hooks · `S` · *Tooling*
Mixed tabs/spaces (15 tab-indented vs 328 space-indented files), no formatter.
- Add Prettier, `.editorconfig`, integrate `eslint-config-prettier`, run a one-time format pass.
- Add Husky + lint-staged to run lint/format/typecheck on commit.
- **Done when:** `prettier --check` is clean and enforced pre-commit + in CI.

### P2-2 · Collapse service-layer boilerplate into a factory · `M` · *Refactor*
Every `services/api/<resource>/index.ts` repeats the same query-param builder and
`if (success && data) return data; throw new Error(...)` block (see
`roles/index.ts`, `users/index.ts`, etc.).
- Extract a generic `createResourceApi<T>()` / shared helpers (`buildQuery`, `unwrap`). See [refactor-architecture.md](./refactor-architecture.md).
- **Done when:** new resources are ~20 lines; the unwrap/throw logic lives in one place.

### P2-3 · Break up oversized components · `L` · *Maintainability*
`ClientAddOrUpdateForm.tsx` (1084 lines), `LoginTemplateForm.tsx` (791),
`UserProfileForm.tsx` (591), `ApiKeyAddOrUpdateForm.tsx` (533), and others exceed
sane limits and are untestable as units.
- Split into sub-forms/sections + hooks; target < ~250 lines per component.
- **Done when:** no page/component file > ~300 lines (or each exception is justified).

### P2-4 · Decide Redux vs TanStack Query boundaries · `M` · *Architecture*
Both Redux Toolkit (auth, tenant) and TanStack Query (everything else) are in use;
`useAuth` hand-wraps thunks with their own try/catch/swallow. See
[refactor-architecture.md](./refactor-architecture.md).
- Document the rule: **Redux = client/session state, TanStack = server state**; remove overlap.
- **Done when:** the boundary is written down and the code follows it; no duplicated server-state caching.

### P2-5 · Drop redundant "backward-compat" object exports · `S` · *Hygiene*
`apiClient`, `authService`, etc. duplicate the named exports, giving two ways to
import the same thing.
- Pick named exports; remove the aggregate objects (or vice versa) and update imports.
- **Done when:** one import style per module.

### P2-6 · Remove or relocate one-off migration scripts · `S` · *Hygiene*
`scripts/*.py` (`rename_pages.py`, `rename_types.py`, `pluralize_services.py`) were
one-time refactors and shouldn't live in the app repo root long-term.
- Move to a `tools/` archive with a README, or delete now that they've run.
- **Done when:** the repo root no longer ships completed migration scripts as if they're part of the app.

### P2-7 · Standardize error handling in hooks & forms · `M` · *Consistency*
Mutation hooks and forms handle errors ad hoc; some toast, some swallow.
- Centralize API-error → user-message mapping (one helper) used by all `useMutation` `onError` + forms.
- **Done when:** every mutation surfaces failures consistently via the toast/error layer.

---

## P3 — Polish & long-tail

### P3-1 · Accessibility pass (WCAG 2.2 AA) · `L` · *A11y*
Only 21 files use `aria-*`/`role`. Add `eslint-plugin-jsx-a11y`, audit focus
management in dialogs/menus, color contrast, and keyboard nav. (See checklist §12.)

### P3-2 · Internationalization scaffolding · `L` · *i18n*
All strings are hardcoded English. Introduce `react-i18next` and externalize strings
incrementally. (See checklist §13.)

### P3-3 · Bundle/performance budget · `M` · *Performance*
After P1-5, add a bundle-size check in CI (e.g. `rollup-plugin-visualizer` + size budget) and lazy-load heavy deps (`recharts`, `react-day-picker`).

### P3-4 · Storybook for the design system · `M` · *DX/UI*
`components/ui/*` (shadcn) and shared components have no isolated docs; Storybook would also host the mock data evicted in P1-6.

### P3-5 · CONTRIBUTING / PR template / CODEOWNERS · `S` · *Governance*
`CODE_OF_CONDUCT.md`, `LICENSE`, and `NOTICE` exist; add `CONTRIBUTING.md`, PR/issue templates, and `CODEOWNERS`.

---

## Suggested sequencing

| Sprint | Theme | Items |
|---|---|---|
| **1** | Stop the bleeding | P0-1, P0-2, P0-3, P0-4, P0-5, P0-6 |
| **2** | Quality gates | P1-1, P1-2, P1-3, P2-1 |
| **3** | Resilience & perf | P1-4, P1-5, P1-6, P1-7, P1-8 |
| **4** | Type & UX cleanup | P1-9, P1-10, P2-5, P2-6, P2-7 |
| **5** | Architecture | P2-2, P2-3, P2-4 (track in [refactor-architecture.md](./refactor-architecture.md)) |
| **6+** | Long tail | P3-1 … P3-5 |

---

## Quick metrics snapshot (2026-06-06)

| Metric | Value |
|---|---|
| TS/TSX files | 583 |
| Test files | **0** |
| `console.*` calls | 40 |
| `any` usages | 32 |
| `eslint-disable` / `@ts-*` suppressions | 3 |
| Bare/swallowing `catch` | 9+ |
| Largest component | `ClientAddOrUpdateForm.tsx` — 1084 lines |
| Files with mock data | users, analytics, signup-flows, log-monitoring constants |
| CI workflows | **0** (`.github/` absent) |
| Prettier / editorconfig / hooks | **none** |

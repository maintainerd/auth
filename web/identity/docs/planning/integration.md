# Backend Integration Backlog — aligning with `maintainerd-auth`

> **Purpose:** Track the work needed to align this app (`maintainerd-auth-identity`)
> with the current `maintainerd-auth` backend API. The backend has moved ahead;
> several console endpoints no longer match, the setup flow is partially wired, and
> a number of console pages still run on mock data while the backend already exposes
> the real endpoints.
>
> Companion to [`backlog.md`](./backlog.md) (general code quality) and
> [`refactor-architecture.md`](./refactor-architecture.md) (structure). This file is
> **integration-specific**.

**Audit date:** 2026-06-06
**Backend:** `../maintainerd-auth` — Go 1.26 + chi, internal API on **:8080** (VPN-only), public API on **:8081**.
**Authoritative sources:** backend `internal/*/routes.go` (actual routes) and `docs/openapi.yaml` (spec — see [INT-2](#int-2--openapi-spec-diverges-from-actual-routes), it currently drifts from the routes).
**Console targets:** the **internal** management API (`/api/v1` on :8080), cookie-authenticated.

Priority/effort legend matches [`backlog.md`](./backlog.md): `P0`–`P3`, `S/M/L/XL`.

---

## How the console talks to the backend (verified)

- **Base path:** `/api/v1` on the internal router. Dev uses the Vite proxy (`/api` → `http://api.maintainerd.auth`); prod uses `VITE_AUTH_API_BASE_URL`.
- **Auth transport:** cookie. The console sends `withCredentials: true` and `X-Token-Delivery: cookie` on login/register; the backend `JWTAuthMiddleware` accepts an `access_token` **cookie** (or `Authorization: Bearer`). ✅ This works against the internal router.
- **CSRF:** the backend applies `CSRFMiddleware` (`__Host-csrf` cookie ↔ `X-CSRF-Token` header) **only on the public router (:8081)** for cookie-auth state-changing routes. The internal router (:8080) does **not** require CSRF. The console sends no CSRF token today — fine for :8080, **broken if ever pointed at :8081** (see [INT-9](#int-9--csrf-not-handled-only-matters-if-pointed-at-public-port)).
- **Authorization:** management routes enforce **fine-grained permissions** (e.g. `security-setting:read`, `ip-restriction-rule:create`) via middleware. The console does not yet gate its UI on the caller's permissions (see [INT-10](#int-10--make-the-ui-permission-aware)).

---

## P0 — Setup flow (first-run bootstrap)

The console's setup wizard (`/setup/tenant` → `/setup/admin` → `/setup/profile`) creates
the tenant, admin, and profile. The three `create_*` calls are aligned, but the
**lifecycle** around them is not.

### INT-1 · Wire `isSetupCompleted()` to `GET /setup/status` · `S` · *P0*
`src/services/api/setup/index.ts` currently does:
```ts
export async function isSetupCompleted(): Promise<boolean> {
  // ...
  return false   // ← hardcoded
}
```
It never calls the backend. The backend now exposes **`GET /setup/status`**.
- Replace the stub with a real call to `/setup/status` and branch the wizard on it.
- **Done when:** a fresh backend routes the user into setup; an already-provisioned backend skips it.

### INT-2 (setup) · Call `POST /setup/complete` to lock setup · `S` · *P0*
The backend added **`POST /setup/complete`** which "explicitly locks setup once
bootstrap records are provisioned." The console never calls it, so setup is never
locked after the admin/profile are created.
- After `create_profile` succeeds, call `/setup/complete`.
- **Done when:** finishing the wizard locks setup server-side (re-running setup is rejected).

### INT-3 · Decide on `POST /setup/register-control-service` · `S` · *P0/P1*
Backend exposes **`POST /setup/register-control-service`** (optional controller
registration before setup is locked). The console has no UI/step for it.
- Confirm with backend owners whether the console should offer this step; if yes, add it to the wizard before `complete`.
- **Done when:** the decision is recorded and (if needed) a step exists.

**Backend setup surface (actual):** `GET /setup/status`, `POST /setup/complete`,
`POST /setup/register-control-service`, `POST /setup/create_tenant`,
`POST /setup/create_admin`, `POST /setup/create_profile`.
**Console uses:** only the three `create_*`. → add `status`, `complete`, (maybe) `register-control-service`.

---

## P0 — Endpoint mismatches that break features today

### INT-4 · Fix Security Settings paths — console calls routes that don't exist · `M` · *P0*
The console splits security settings into separate services hitting:

| Console calls | Backend has | Status |
|---|---|---|
| `GET/PUT /security-settings/general` | `…/mfa` | ❌ **wrong path** (no `/general`) |
| `GET/PUT /security-settings/password` | `…/password` | ✅ |
| `GET/PUT /security-settings/session` | `…/session` | ✅ |
| `GET/PUT /security-settings/threat` | `…/threat` | ✅ |
| `GET/PUT /security-settings/ip` | *(none)* — IP is `/ip-restriction-rules` | ❌ **wrong path** |
| *(missing)* | `…/lockout` | ⚠️ backend-only |
| *(missing)* | `…/registration` | ⚠️ backend-only |
| *(missing)* | `…/token` | ⚠️ backend-only |

- Rename `general` → `mfa`; remove `/security-settings/ip` and point IP UI at the top-level `/ip-restriction-rules` resource (already a separate console service — consolidate).
- Add services/UI for `lockout`, `registration`, `token`.
- **Done when:** every security-settings call maps to a real backend route; no 404s.

### INT-5 · Security-settings updates require MFA step-up — console has no step-up flow · `L` · *P0*
Every **`PUT`** under `/security-settings/*` is guarded by `middleware.RequireStepUp`.
The console has **no MFA / step-up** implementation, so **all security-settings saves
will be rejected**.
- Implement the step-up challenge/verify flow (`POST /mfa/step-up/challenge`, `POST /mfa/step-up/verify`) and retry the guarded request after a successful step-up.
- **Done when:** saving any security setting triggers step-up when required and succeeds afterward.

### INT-6 · Remove the dead `/refresh` endpoint · `S` · *P0*
`API_ENDPOINTS.AUTH.REFRESH = '/refresh'` has **no matching backend route**. Token
refresh on the backend happens via cookie rotation / the OAuth token endpoint, not a
REST `/refresh`.
- Delete the constant and any code path assuming it; confirm the real refresh mechanism with backend owners and document it.
- **Done when:** no console code references a non-existent `/refresh`.

---

## P1 — Mock-data & placeholder pages the backend can now power

These pages render committed mock data ([`backlog.md`](./backlog.md) P1-6) even though
the backend exposes the real endpoints. Integration + de-mocking should happen together.

### INT-7 · Connect the mock/placeholder pages to real endpoints · `L` · *P1*

| Console page | Today | Backend endpoint(s) to use |
|---|---|---|
| **Logs** (`log-monitoring`) | ~400 lines of mock rows | `GET /auth-events`, `GET /auth-events/count`, `GET /auth-events/{uuid}` |
| **Analytics / Monitoring** | mock | `GET /auth-events/count` (+ any metrics surface) |
| **Notifications** | mock | `/email-config`, `/sms-config` |
| **Session management** | mock | `/security-settings/session` (policy) + `/account/sessions` (live sessions) |
| **Events** (route → DashboardPage placeholder) | not built | `/event-routes`, `/event-types`, `/tenant-event-types` |
| **Webhooks** (route → DashboardPage placeholder) | not built | `/webhook-endpoints`, `/webhook-endpoints/{uuid}/status`, `/webhook-replay` |

- **Done when:** each page reads live data; its mock fixtures are deleted/moved to test fixtures.

---

## P1/P2 — Backend capabilities with no console surface

The backend exposes whole feature areas the console doesn't yet manage. Triage with
product before building, but they should be on the radar.

### INT-8 · Plan console coverage for un-surfaced backend features · `M` (planning) + `XL` (build) · *P1/P2*

| Area | Backend routes | Notes |
|---|---|---|
| **MFA management** | `/mfa/*` (TOTP enroll/verify, WebAuthn register/auth, backup codes, step-up, `admin/users/{uuid}/reset`) | Prereq for [INT-5](#int-5--security-settings-updates-require-mfa-step-up--console-has-no-step-up-flow). |
| **Invites** | `/invite`, `/register/invite` | Admin-issued user invitations. |
| **Tenant members** | `/tenants/{tenant_uuid}/members` | Member management per tenant. |
| **Tenant settings** | `/tenant-settings` | Distinct from `/security-settings`. |
| **User self-service account** | `/account` (sessions, email change, export, backup codes), `/recovery` | Some may belong to the public app, not the console — confirm scope. |
| **User settings** | `/user-settings` | Per-user prefs. |
| **Federation / identity linking** | `/account/identities`, `/federation`, `/identity_providers/.../oauth2/callback` | Console manages IdPs; linking may be end-user scope. |
| **OAuth consent grants** | `/oauth/consent/grants`, `/oauth/consent/grants/{uuid}` | Admin view of granted consents. |

- **Done when:** each area has a decision (build now / later / out-of-scope) recorded here.

---

## P2 — Robustness of the integration layer

### INT-2 · OpenAPI spec diverges from actual routes · `M` · *P2*
`maintainerd-auth/docs/openapi.yaml` does **not** match the registered chi routes:

| Spec (`openapi.yaml`) | Actual route (`routes.go`) |
|---|---|
| `/auth/login`, `/auth/register`, … | `/login`, `/register`, … (no `/auth` prefix on internal router) |
| `/identity-providers` (kebab) | `/identity_providers` (snake) |
| `/tenants/{tenant_uuid}` only | `/tenant` **and** `/tenants` both exist |
| *(omits)* setup, signup_flows, templates, branding, security-settings, notifier, webhooks, events | all registered in `router.go` |

The console currently matches the **code**, not the spec. But this drift is a trap:
generating a client/types from `openapi.json` today would produce wrong paths.
- **Action (mostly backend-side, track here):** ask backend owners to regenerate `openapi.yaml` from the live routes (the server serves `/openapi.json`). Then [INT-11](#int-11--generate-types--client-from-openapi) becomes safe.
- **Done when:** `openapi.json` matches `routes.go` for every path the console uses.

### INT-9 · CSRF not handled (only matters if pointed at public port) · `S` · *P2*
If the console is ever deployed against the **public** API (:8081) instead of the
internal one, its cookie-auth `POST/PUT/PATCH/DELETE` calls will be rejected — the
public router requires the `__Host-csrf` cookie echoed in an `X-CSRF-Token` header.
- Add an axios request interceptor that reads the `__Host-csrf` cookie and sets `X-CSRF-Token` on unsafe methods. Harmless on :8080, required on :8081.
- **Done when:** the console works against either port without 403s.

### INT-10 · Make the UI permission-aware · `M` · *P2*
The backend enforces per-endpoint permissions (`<resource>:read|create|update|delete`,
`security-setting:*`, etc.). The console shows actions regardless, so users hit 403s.
- Fetch the caller's effective permissions/policy bundle (`GET /services/me/policy-bundle` or the profile) and hide/disable actions accordingly.
- **Done when:** the UI only offers actions the current user is authorized for.

### INT-11 · Generate types / client from OpenAPI · `M` · *P2*
The console hand-maintains request/response types per resource — they drift from the
backend as it changes (this whole backlog is the cost of that drift).
- Once [INT-2](#int-2--openapi-spec-diverges-from-actual-routes) is fixed, generate types (e.g. `openapi-typescript`) from `/openapi.json` in CI and feed the service layer / factory (see [`refactor-architecture.md`](./refactor-architecture.md) Problem 2).
- **Done when:** request/response types are generated, not hand-written, and drift fails the build.

### INT-12 · Replace fragile endpoint string-building · `S` · *P2*
The tenant service builds paths like `` `${API_ENDPOINTS.TENANT}s` `` (`/tenant` → `/tenants`)
by appending a literal `s`. This is brittle and obscures which routes are hit.
- Declare explicit constants (`TENANTS: '/tenants'`, `TENANT: '/tenant'`) rather than string-concatenating.
- **Done when:** no endpoint is derived by appending characters to another endpoint.

---

## Endpoint alignment reference

**Aligned today (console config ↔ backend routes):**
`/services`, `/apis`, `/permissions`, `/policies`, `/roles`, `/users`, `/clients`,
`/identity_providers`, `/signup_flows`, `/email_templates`,
`/sms_templates`, `/login_templates`, `/ip-restriction-rules`, `/tenant` + `/tenants`,
auth `/login` `/register` `/logout` `/forgot-password` `/reset-password` `/profile`,
setup `create_tenant` `create_admin` `create_profile`.

**Needs action:** see INT-1…INT-12 above.

**Backend auth routes the console doesn't use (confirm if needed):**
`/email-verification/send|verify`, `/magic-link/send|verify`, `/sms-login/send|verify`,
`/register/invite`.

---

## Suggested sequencing

| Step | Items |
|---|---|
| **1 — Unblock setup** | INT-1, INT-2(setup `/complete`), INT-3 |
| **2 — Stop the 404s** | INT-4, INT-6, INT-12 |
| **3 — Unblock settings saves** | INT-5 (+ INT-8 MFA prerequisite) |
| **4 — De-mock with real data** | INT-7 |
| **5 — Hardening** | INT-9, INT-10 |
| **6 — Kill the drift** | INT-2 (spec), INT-11 |
| **7 — Expand coverage** | INT-8 (per product triage) |

> **Cross-refs:** mock-data removal is [`backlog.md`](./backlog.md) P1-6; the service-layer
> factory that INT-11 feeds is [`refactor-architecture.md`](./refactor-architecture.md) Problem 2;
> env/prod base-URL handling is [`backlog.md`](./backlog.md) P0-4.

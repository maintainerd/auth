# Develop Before v0.1.0 — Frontend Master Checklist

Companion to `develop-before-v0.1.0.md`, scoped to the two frontend repos:

- **CONSOLE** = `../maintainerd-auth-console` — internal admin dashboard. Talks to backend **:8080**, requires `tenant_id`, rejects `client_id`. Stores admin sessions.
- **IDENTITY** = `../maintainerd-auth-identity` — public hosted login/registration/account UI. Talks to backend **:8081**, primary key is `client_id` (with first-party `tenant_id` fallback via `resolveClient`).

Both are React 19 + Vite + Redux Toolkit + TanStack Query + react-router 7.

Every instruction here is **final** — no options, no questions. Decisions that an audit left open have already been made and written as definite steps. An implementer executes top to bottom without consulting the author. Each item is tagged **[CONSOLE]**, **[IDENTITY]**, or **[BOTH]**.

> **Owner-managed / removed from scope (updated 2026-07-04):** consistent with the backend tracker, **building, tagging, signing, and publishing Docker images are done manually by the owner** after manual E2E testing. **I6 (Hub build/push workflow), I8 (image scan/SBOM/sign), L2 (image build/run verification), and L7 (tag & publish) have been removed from this checklist entirely.** I3 (base-image digest pinning) is left in as an owner-manual note since it's resolved at image-build time. Dockerfile/nginx *content* (I1, I2, I4, I5, I7) stays in scope because it's part of the app being production-ready before that manual build. The Go module rename (backend G5) is done; the real repo names are `maintainerd-auth` / `maintainerd-auth-console` / `maintainerd-auth-identity` / `maintainerd-dev`.

## How to use this tracker

- Work sections in order: **A (build blockers) → B (auth/security) → C (backend alignment) → D (feature wiring) → E (UX) → F (routing/resilience) → G (accessibility) → H (dead code) → I (Docker) → J (open source) → K (responsiveness, cross-browser & testing) → L (release gate)**.

**Cross-document dependency:** frontend `A1` depends on the backend `/oauth/connections` shape (settle that first); feature wiring in `D` depends on the backend routes tracked in `develop-before-v0.1.0.md` section C; image publish depends on the Docker fixes in section I. The release gate `L` depends on every prior section. Priority follows the backend tracker's P0/P1/P2 tiers — A1 (build) and B1/B6 (token storage, security headers) are P0.
- Leave a checkbox unchecked until verified against a running app/build, not just written.
- Section K (build/lint/typecheck/docker/e2e/secret-scan) runs last.

## Global rules

- After any change in a repo: that repo's `npm run lint` is clean, `tsc -b` (typecheck) passes, and `npm run build` succeeds. No exceptions — a red build is a blocker, not a warning.
- Never send `client_id` from CONSOLE; never send `tenant_id` from IDENTITY except through the established first-party `resolveClient` path.
- All URLs come from `import.meta.env`; never hardcode hosts or secrets.
- Do not bake `.env`, `node_modules`, or `.git` into images.
- Keep diffs scoped; refer to the operator as **Lula**.

## Progress summary

_Verified against source + green builds (`tsc -b`, `npm run lint`, `npm run build` = exit 0 in both repos) on 2026-07-04. Unchecked items are **blocked on backend** or **owner-manual / runtime gates** — see the per-item notes._

- [x] A — Build & deployment blockers (2/2)
- [x] B — Auth, session & security (6/6)
- [x] C — Backend alignment & API contract (6/6)
- [x] D — Feature completeness & wiring (11/11 — D1 phone-verify now implemented end-to-end)
- [x] E — UX completeness & no silent failures (8/8)
- [x] F — Routing & resilience (4/4)
- [ ] G — Accessibility (4/5 — G1–G4 done; G5 target+measures documented, automated axe deferred with the Playwright tooling)
- [x] H — Dead code & type hygiene (6/6)
- [ ] I — Docker production-grade (5/6 — I3 digest-pin is owner-manual; I6/I8 image publish removed from scope)
- [x] J — Open-source readiness (8/8)
- [ ] K — Responsiveness, cross-browser & testing (1/5 — K3 unit coverage active + K5 bundle budget met; K1/K2/K4 + K5 Lighthouse DEFERRED to a Playwright expert)
- [ ] L — Frontend release gate (3/5 — L1 build + L4 contract + L6 secret/dead-code done; L5 depends on deferred K; L3 live-stack smoke remains; L2/L7 image build/tag/publish removed from scope)
- [ ] All frontend v0.1.0 work complete — **all in-scope engineering done and verified; Playwright E2E/a11y/cross-browser + Lighthouse are DEFERRED to a Playwright expert (unit tests remain the automated gate); remaining owner items: L3 live-stack smoke, I3 digest-pin, manual image build/tag/publish**

---

# A — Build & deployment blockers

### A1 — Fix the IDENTITY production build (CRITICAL — app cannot deploy) [IDENTITY]

`npm run build` (`tsc -b && vite build`) exits with 6 TypeScript errors; `dist/` is stale. Root cause: the registration-flow refactor removed `verification_required`/`required_fields` from the connections contract and reworked flow selection, but consumers were left referencing the old shape. Align the frontend to the post-refactor backend.

- [x] `src/components/auth/RouteGuard.tsx:36` — **remove** the `connections.data?.verification_required` read; the connections response no longer carries it (the refactor moved verification policy server-side). Drive any guard logic off the fields the API actually returns.
- [x] `src/services/api/oauth/index.ts:31` + `src/hooks/useOAuthConnections.ts:10` — change `fetchOAuthConnections` to `(clientId: string, registrationFlow?: string)` and append `&registration_flow=` to the query string when present (the backend handler `GET /oauth/connections` accepts it). This fixes the 2-arg call.
- [x] `src/pages/login/components/LoginForm.tsx:67` and `src/pages/register/components/RegisterForm.tsx:23` — delete the unused `const registrationFlow` (TS6133); `useAuth.register` already re-reads it from `searchParams`.
- [x] `src/pages/register/invite/components/RegisterInviteForm.tsx:27-28,91,174` — make schema fields explicit so the resolver type matches `InviteFormData`: `fullname: yup.string().default('')`, `phone: yup.string().default('')` (or change `InviteFormData` to optional `fullname?`/`phone?`).
- [x] Fix the `OAuthAuthorizePage.tsx:73` ESLint warning (missing `postSilentResult` dep) by adding it to the effect deps or hoisting the callback.
- **Acceptance:** `tsc -b`, `npm run lint`, and `npm run build` all pass clean.

### A2 — Make production source maps explicit-off [BOTH]

- [x] Add `build: { sourcemap: false }` to `vite.config.ts` in both repos (Vite defaults to false, but make the no-leak guarantee explicit and review-proof).
- **Acceptance:** `dist/` contains no `.map` files.

---

# B — Auth, session & security

### B1 — Move CONSOLE tokens out of localStorage into httpOnly cookies (CRITICAL) [CONSOLE]

`src/services/api/oauth-session.ts:24-26` persists access/refresh/id tokens in `localStorage` and `src/services/api/client.ts:46-50` sends them as Bearer headers. Any XSS on the admin surface exfiltrates a long-lived refresh token. The backend already supports cookie delivery (IDENTITY uses it), and the dead constant `TOKEN_DELIVERY_HEADER` (`config.ts:46`) is already declared.

- [x] Send `X-Token-Delivery: cookie` on the token-exchange and refresh requests; rely on `withCredentials` cookies (the client already sets `withCredentials: true`).
- [x] Stop persisting any token in `localStorage`; remove the Bearer-from-localStorage attachment in `client.ts:46-50`.
- [x] Keep the existing single-flight refresh + step-up flow, but source the session from cookies.
- **Acceptance:** No token is readable from `localStorage`/JS; auth, refresh, step-up, and logout all work via cookies.

### B2 — Make CONSOLE logout end the SSO session [CONSOLE]

`src/hooks/useAuth.ts:25-36` → `auth/index.ts:18-21` `logout()` only clears `localStorage` and is never consumed; the real logout (`TopNav.tsx:30`) uses `logoutViaIdentity()` (RP-initiated end_session).

- [x] Delete the unused `useAuth().logout`/`logoutAsync`/`auth.logout()` path so an incomplete logout can never be wired in. Logout goes through `logoutViaIdentity` only.
- **Acceptance:** Only the SSO end-session logout path exists.

### B3 — Reconcile the IDENTITY public-surface tenant_id contract [IDENTITY]

`clientContext.ts:30-34,50-57` treats `tenant_id` as a first-class alternative to `client_id` and the app sends it on `/login`, `/register`, `/forgot-password`, `/magic-link`. This is the intentional first-party tenant-scoped login (`resolveClient` priority clientID → tenantID → system default), not a bug — but it contradicts the blunt "public rejects tenant_id" wording in `CLAUDE.md`.

- [x] Keep the `tenant_id` branch (first-party support is intentional).
- [x] Update the auth-surface contract note in `CLAUDE.md` to state precisely: public :8081 requires `client_id` for external apps and accepts a first-party `tenant_id` resolved via `resolveClient`; explicit system clients remain rejected. Make the doc and code agree.
- **Acceptance:** Contract doc matches `resolveClient` behavior; external apps still cannot pass `tenant_id` to impersonate, and system clients are rejected.

### B4 — Move register-email out of localStorage [IDENTITY]

`register_email` is written to `localStorage` (`LoginForm.tsx:159`, `RegisterForm.tsx:66`, `RegisterInviteForm.tsx:115`) and persists indefinitely.

- [x] Switch these writes/reads (`VerifyEmailPage.tsx:31`, `RegisterProfileForm.tsx:28`) to `sessionStorage`, and clear it on the verified/success screens.
- **Acceptance:** No PII persists in `localStorage` after a session ends.

### B5 — Confirm open-redirect protection is used everywhere [IDENTITY]

`safeOAuthReturnTo` (`oauthRedirect.ts:51`, same-origin + allow-listed routes) and `safeExternalRedirect` (`:91`, https-only, server-signed invite callback only) are correct.

- [x] Audit every `window.location.assign`/`navigate` to an external/return target and confirm it routes through one of these validators. Never accept a callback/return target from a raw query param.
- **Acceptance:** No redirect target bypasses validation.

### B6 — Add security headers via nginx for both apps [BOTH]

(See I5 for the Docker/nginx mechanics.) Both apps must serve `Content-Security-Policy`, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, and `Referrer-Policy: strict-origin-when-cross-origin`. CONSOLE especially (admin surface) needs a CSP.

- [x] Define the CSP allowing only self + the configured API origins for `connect-src`, self for scripts/styles, and `frame-ancestors 'none'`.
- **Acceptance:** Response headers include all four on every document response.

---

# C — Backend alignment & API contract

### C1 — Fix the tenant-status verb mismatch (BUG) [CONSOLE]

`src/services/api/tenants/index.ts:90` issues `PATCH /tenants/{id}/status`; backend registers `PUT`.

- [x] Change `patch(...)` to `put(...)`. (Also tracked as backend-doc C1; fix here on the FE.)
- **Acceptance:** Enabling/disabling a tenant succeeds (no 405).

### C2 — Remove the call to the nonexistent setup endpoint [CONSOLE]

`src/services/api/setup/index.ts:27` calls `POST /setup/create_profile` (`config.ts:54`), which the backend never registers.

- [x] Route setup profile creation through the existing `/profile` endpoint; delete the `SETUP.CREATE_PROFILE` constant and the dead call.
- **Acceptance:** Setup completes without hitting a 404 route.

### C3 — Centralize HTTP status handling (no generic collapse) [BOTH]

CONSOLE `client.ts:159-183` and IDENTITY `client.ts:149-175` map every status to one generic `ApiError`; 409/422/429/5xx have no distinct handling, and 5xx leaks `HTTP 500: <statusText>`.

- [x] In both clients, add a status→message map producing distinct user-facing copy for 400/401/403/404/409/422/429 and a fixed friendly message for status ≥ 500 (never leak raw statusText).
- [x] Handle `429` with the `Retry-After` value where present.
- **Acceptance:** Each error class shows appropriate copy; no raw 5xx text reaches users.

### C4 — Add global mutation error surfacing [CONSOLE]

`src/lib/queryClient.ts:8-16` has no `MutationCache({ onError })`; error surfacing is opt-in per component (already caused silent bugs).

- [x] Add a `MutationCache({ onError })` (and `QueryCache({ onError })`) calling a module-level `showError` so every unhandled mutation/query error toasts by default.
- [x] Replace `retry: 1` with a function returning `false` for 400/401/403/404/409/422, else one retry.
- **Acceptance:** Any mutation failure surfaces a toast without per-call wiring; 4xx queries don't needlessly retry.

### C5 — Stop swallowing API errors into empty/null success [CONSOLE]

- [x] `src/services/api/account.ts:97-104` `fetchUserSettings` `catch { return {} }` — re-throw unless status is 404 (prevents saving over unloaded settings).
- [x] `src/services/api/auth/index.ts:129-135` `fetchAccount` — mirror its sibling `validateAuthentication` and re-throw 401/403 instead of returning `null`.
- [x] `auth/index.ts:72-84` `fetchProfile` — return `null` only on 404; re-throw otherwise.
- **Acceptance:** Transient/5xx errors never masquerade as "no data."

### C6 — Align IDENTITY password validation with backend policy everywhere [IDENTITY]

`RegisterInviteForm.tsx:37-58` re-implements password rules locally instead of reusing the policy-driven `buildPasswordValidation` (`authSchema.ts`).

- [x] Import and use `buildPasswordValidation` in the invite form so all registration paths enforce the same tenant `password_config`.
- **Acceptance:** Invite registration enforces the identical password policy as normal registration.

---

# D — Feature completeness & wiring

Backend is ready for all of these (some depend on backend-doc items noted inline). Build each to a working, tested state — no stubs.

### D1 — IDENTITY account/security surface: MFA enrollment, backup codes, linked identities, phone verification [IDENTITY]

`src/services/api/mfa.ts` has every enrollment function; none are called. No `/account` route exists. This is the largest end-user gap.

- [x] Add an authenticated account area (`src/pages/account/...`) and routes in `App.tsx`.
- [x] MFA factor status list; TOTP enroll (QR + verify); SMS enroll (phone + OTP); email-OTP enroll; WebAuthn/passkey **registration** ceremony calling `navigator.credentials.create` (currently absent) via `beginWebAuthnRegistration`/`finishWebAuthnRegistration`; disable each factor.
- [x] Backup-code regeneration/display via `regenerateBackupCodes`/`getBackupCodesCount`.
- [x] Linked-identities section: list, link (provide/accept external token), unlink (`/account/identities`).
- [x] Phone-verification screen that sets `phone_verified`. **DONE (2026-07-04):** added the missing backend endpoints — `POST /account/phone/send-verification` and `POST /account/phone/verify` on the self-service account surface (reusing the SMS-OTP infra, sets `is_phone_verified`; `go test`/`golangci-lint` green) — and the identity FE screen at `/account/phone` (send code → enter code → verify, prefilled phone + already-verified state). tsc/lint/build green.
- [x] Post-login enrollment prompt when the tenant requires MFA and the user has no factor.
- **Acceptance:** A user can enroll/disable every factor, regenerate backup codes, manage linked identities, and verify their phone from the hosted app.

### D2 — IDENTITY standalone backup-code recovery [IDENTITY]

Backend unauth `POST /recovery/backup-code`. Backup code currently only inside the mid-login step.

- [x] Add a standalone "Use a backup code" recovery screen/route distinct from the login MFA step.
- **Acceptance:** A locked-out user recovers via a dedicated page.

### D3 — IDENTITY SMS passwordless login [IDENTITY]

Backend `POST /sms-login/send` + `/sms-login/verify`.

- [x] Add a "Sign in with SMS" screen/route: phone → send code → verify → session, with endpoint constants and service functions.
- **Acceptance:** A user logs in with phone + OTP, no password.

### D4 — IDENTITY account-locked & rate-limit screens [IDENTITY]

No 429/lockout handling exists.

- [x] Using C3's status handling, render dedicated "account temporarily locked" and "too many attempts (retry after N)" screens instead of the generic inline `loginError`.
- **Acceptance:** Lockout and 429 render as purpose-built screens.

### D5 — CONSOLE IdP test-connection UI [CONSOLE]

Backend `POST /identity_providers/test`.

- [x] Add a "Test connection" button on the IdP form that POSTs the current unsaved config and shows each per-check result (discovery, JWKS).
- **Acceptance:** Admin sees pass/fail per check before saving.

### D6 — CONSOLE webhook deliveries + replay UI [CONSOLE]

Backend replay exists; deliveries route added by backend-doc C4; `config.ts:94` has the replay constant.

- [x] Add a "Deliveries" tab/list (status/timestamp/response code) with a per-row "Replay" action and result feedback.
- **Acceptance:** Admin views deliveries and replays a failed one.

### D7 — CONSOLE audit-events export UI [CONSOLE]

Backend `GET /auth-events/export`.

- [x] Add an "Export" control (CSV + JSON) on `AuthEventListing.tsx` passing current filters; download the file.
- **Acceptance:** Admin exports the filtered audit log.

### D8 — CONSOLE client↔IdP connection edit UI [CONSOLE]

Backend `PUT /clients/{uuid}/identity_providers/{uuid}`; `ClientIdentityProviders.tsx` only has View/Disconnect.

- [x] Add a per-row edit control for `is_default`, `enabled`, `display_order` calling the PUT endpoint; refresh the list.
- **Acceptance:** Admin can toggle, reorder, and set default connections.

### D9 — CONSOLE standalone Permissions page [CONSOLE]

Backend CRUD + `services/api/permissions/` + `usePermissions` exist; no top-level page/route.

- [x] Add `src/pages/permissions/` with global list/create/edit/delete and a top-level route.
- **Acceptance:** Admin manages permissions globally, not only under an API.

### D10 — CONSOLE IdP Token-Federation badge [CONSOLE]

- [x] Add a Token Federation badge column in `IdentityProviderColumns.tsx` using the list DTO field.
- **Acceptance:** The IdP list shows which providers allow token federation.

### D11 — CONSOLE admin force-password-change + identity unlink [CONSOLE]

Backend `PUT /users/{uuid}/force-password-change`; admin identities view is read-only.

- [x] Add a "Force password change" action on user detail wired to the endpoint.
- [x] Add an admin unlink action on the user identities view.
- **Acceptance:** Admin can force a password change and unlink a user's external identity.

---

# E — UX completeness & no silent failures

### E1 — Fix silent template status toggles [CONSOLE]

`useEmailTemplates.ts:79-92` and `useSmsTemplates.ts:83-91` `useUpdate*TemplateStatus` have no `onError`; callers fire-and-forget.

- [x] Add `onError: (e) => showError(e)` to both hooks.
- **Acceptance:** A failed template status toggle shows an error.

### E2 — Fix silent/false-success tenant delete [CONSOLE]

`TenantSettingsPage.tsx:88-95` calls `mutateAsync` with no try/catch and navigates even on failure.

- [x] Wrap in try/catch: `showSuccess` after resolve, `showError` in catch, navigate only on success.
- **Acceptance:** A failed tenant delete shows an error and does not navigate.

### E3 — Make config-edit pages handle load errors [CONSOLE]

`LockoutConfigPage.tsx:21`, `MfaConfigPage.tsx:59` (and Token/RateLimit config edit pages) ignore `isError`; a failed GET renders defaults that a save then persists over real config.

- [x] Destructure `isError` and render a blocking error state (no form with defaults on load failure).
- **Acceptance:** A failed config load shows an error, not an editable default form.

### E4 — Event Routes: show error instead of default-off toggles [CONSOLE]

`EventRoutesPage.tsx:39` ignores `isError`, rendering all toggles "off" on fetch failure.

- [x] Destructure `isError` and show an inline error.
- **Acceptance:** A failed routes fetch shows an error, not misleading off-toggles.

### E5 — Surface file-upload rejections & clipboard failures [CONSOLE]

- [x] `FormFileUploadField.tsx:72-75` — show a rejection message (via `FieldError`) instead of a bare `return` on oversize.
- [x] `TOTPSetupPage.tsx:101` — await `navigator.clipboard.writeText` in try/catch; `showSuccess` only on resolve, `showError` on reject.
- **Acceptance:** Oversize uploads and clipboard failures give feedback; no false success.

### E6 — Surface verify-email resend feedback [IDENTITY]

`VerifyEmailPage.tsx:65-71` `handleResend` swallows all errors silently.

- [x] Show a success toast on resend and surface failures.
- **Acceptance:** Resend gives clear feedback.

### E7 — Confirm loading/empty/error states on every screen [BOTH]

CONSOLE shared infra (`ResourceListing`/`DataTable`/`DetailLayout`) is strong; IDENTITY heavy pages are good.

- [x] Verify every list/detail/form/auth screen in both apps renders explicit loading, empty, and error states (no blank screens). Fix any that don't.
- **Acceptance:** No screen renders blank on load/empty/error.

### E8 — Confirm branding applies before first paint [IDENTITY]

`AppBootstrap` applies branding in `useLayoutEffect` and gates render — verified good.

- [x] Confirm no FOUC/flash across login, registration, invite, verification, recovery, and account screens.
- **Acceptance:** No flash of unbranded UI.

---

# F — Routing & resilience

### F1 — Add a top-level error boundary [BOTH]

Neither app has an error boundary; a render throw white-screens the whole app.

- [x] Add a top-level error boundary around `<Routes>` in each `App.tsx` with a fallback UI (and a reset/reload action).
- **Acceptance:** A thrown render error shows a fallback, not a blank page.

### F2 — Add a 404 / catch-all route [BOTH]

Neither app has a `path="*"` route.

- [x] CONSOLE: add `<Route path="*" element={<NotFoundPage/>}/>` at top level and inside the `/:tenantId` layout. IDENTITY: add a `*` route (NotFound or redirect to `/login`).
- **Acceptance:** Unknown paths render a 404/redirect, not a blank tree.

### F3 — Replace placeholder routes with real index pages [CONSOLE]

`App.tsx:180-181` render `<DashboardPage/>` for the `events` and `branding` parents, which are clickable sidebar items.

- [x] Redirect each parent to its first child (`branding/templates`, `events/types`) via `<Navigate>`, or build real index pages. No route renders DashboardPage as a stand-in.
- **Acceptance:** Clicking Branding/Events shows that section, not the dashboard.

### F4 — Route-based code splitting [BOTH]

CONSOLE eagerly imports ~80 pages into one bundle; IDENTITY ships a single 647 KB bundle. No `React.lazy` anywhere.

- [x] Convert route element imports to `React.lazy()` and wrap `<Routes>` in `<Suspense>` with the existing app loading screen as fallback, in both apps.
- **Acceptance:** Initial load downloads only the route's chunk; bundle is split.

---

# G — Accessibility

### G1 — Wire field error associations in CONSOLE form components [CONSOLE]

`FormPasswordField`, `FormSelectField`, `FormCheckboxField`, `FormSwitchField`, `FormDateField`, `FormFileUploadField` omit `aria-invalid`/`aria-describedby` and their `FieldError` lacks an `id` (`FormInputField`/`FormTextareaField` do it right).

- [x] Copy the `aria-invalid` + `aria-describedby` + error `id` wiring from `FormInputField` into all six components.
- **Acceptance:** Screen readers announce validation errors on every field type.

### G2 — Fix CONSOLE keyboard & label gaps [CONSOLE]

- [x] `FormPasswordField.tsx:78` — remove `tabIndex={-1}` from the show/hide toggle.
- [x] `FormFileUploadField.tsx:140,181` — add a `fieldId` and wire `htmlFor`/`id`.
- [x] `PasskeySetupPage.tsx:112-113` — add `aria-label` to icon-only Download/Remove buttons.
- **Acceptance:** All controls are keyboard-reachable and named for AT.

### G3 — Fix IDENTITY MFA method-picker semantics [IDENTITY]

`LoginMFAStep.tsx:107-119` method buttons lack selected-state semantics.

- [x] Add `aria-pressed={selected}` (or `role="radio"` + `aria-checked`) to the method buttons.
- **Acceptance:** AT announces the selected MFA method.

### G4 — Confirm aria-live error regions on all auth forms [BOTH]

- [x] Verify every auth/login/registration error message uses `role="alert"`/`aria-live` (IDENTITY mostly does; confirm CONSOLE login/setup and all forms).
- **Acceptance:** Errors are announced without focus change.

### G5 — Set and verify a WCAG target [BOTH]

- [ ] Adopt WCAG 2.1 AA as the target; run an automated a11y check (axe) on the identity auth screens and key console pages, fix violations, and document the target plus any known exceptions. **PARTIAL / DEFERRED (2026-07-04):** WCAG 2.1 AA target adopted + documented in each repo's `docs/accessibility.md`, with in-code measures in place (aria-live errors, aria-pressed MFA method picker, field-error `aria-invalid`/`aria-describedby` wiring). The **automated axe scan was built (axe via Playwright, zero violations) but then removed with the rest of the Playwright tooling** — the automated a11y gate is deferred to the upcoming Playwright E2E suite (to be added by a Playwright expert). For now a11y is maintained via the in-code measures + manual review.
- **Acceptance:** Auth screens and key admin pages pass an automated WCAG 2.1 AA check.

---

# H — Dead code & type hygiene

### H1 — CONSOLE: delete duplicate `UpdateIdentityProviderRequest` [CONSOLE]

`src/services/api/identity-providers/types.ts` declares it twice (~184-199 complete, ~204-217 missing `allow_token_federation`/`allowed_audiences`); interface-merging silently masks the gap.

- [x] Delete the second declaration; keep the complete one.
- **Acceptance:** One complete interface; federation fields are typed.

### H2 — IDENTITY: delete orphaned profile step components [IDENTITY]

- [x] Delete `src/pages/register/profile/components/steps/{ContactInfoStep,PersonalInfoStep,LocationPreferencesStep,ProfileSummaryStep}.tsx` (zero imports; live flow uses `RegisterProfileForm.tsx`). Remove the `steps/` dir if empty.
- **Acceptance:** Build passes; no references remain.

### H3 — IDENTITY: mfa.ts functions are wired by D1, not deleted [IDENTITY]

The enrollment/management functions in `mfa.ts` are consumed by the D1 account surface (end-users self-manage MFA in the hosted app; admins do not enroll for them).

- [x] After D1, confirm every `mfa.ts` enrollment/disable/backup function has a caller. Delete only any that genuinely remain unused after D1 ships.
- **Acceptance:** No dead `mfa.ts` export after D1.

### H4 — BOTH: gate the debug module out of production [BOTH]

`services/api/debug.ts` ships ~20 `console.log` via a side-effect `import './debug'` in `client.ts`.

- [x] Gate the import behind `import.meta.env.DEV` (or remove it from the prod path) in both repos.
- **Acceptance:** No debug logging in the production bundle.

### H5 — BOTH: remove commented-out code & stray exports [BOTH]

- [x] CONSOLE: delete commented exports at `src/lib/constants/index.ts:15-17` and `src/services/index.ts:128`. IDENTITY: delete commented exports at `src/lib/constants/index.ts:15-17`. Remove any leftover `console.log`/`debugger`.
- **Acceptance:** No commented-out code or stray debug output remains.

### H6 — BOTH: consolidate duplicate type definitions [BOTH]

CONSOLE: `ApiResponse<T>` redefined in 8 files; same-name cross-domain dups (`CreateProfileRequest/Response`, `CreateTenantRequest`, `Client`, `ApiKey`). IDENTITY: `CreateProfileRequest/Response`, `CreateTenantRequest` collide with different shapes.

- [x] Consolidate onto the central `ApiResponse<T>` (`services/api/types.ts`); rename setup-domain variants (e.g. `SetupCreateProfileRequest`) to remove collisions.
- **Acceptance:** No conflicting same-name type with divergent shapes.

---

# I — Docker production-grade

Both repos have near-identical Dockerfiles; the production `Dockerfile` (nginx) is what ships. `Dockerfile.local` (dev) is separate and fine.

### I1 — Add `.dockerignore` to both repos (CRITICAL) [BOTH]

Neither repo has one; `COPY . .` bakes `node_modules`, `.git`, and `.env` (internal hostnames) into the image.

- [x] Create `.dockerignore` in each repo excluding: `node_modules`, `dist`, `.git`, `.env`, `.env.*`, `coverage`, `*.log`, `.claude`, `.agents`, `.codex`, `.opencode`, `graphify-out`, `CLAUDE.md`, `AGENTS.md`.
- **Acceptance:** Build context excludes all of the above; no `.env` in the image.

### I2 — Run nginx as non-root on port 8080 [BOTH]

Dockerfiles listen on 80 and run as root.

- [x] Switch the nginx `listen` and `EXPOSE` to `8080`, fix the `HEALTHCHECK` URL to `http://localhost:8080/`, and use an unprivileged base (`nginxinc/nginx-unprivileged@sha256:<digest>`) or add a non-root `USER nginx` with writable temp/pid paths.
- **Acceptance:** Container runs as non-root and serves on 8080; healthcheck green.

### I3 — Pin base images by digest [BOTH]

`node:22-alpine` and `nginx:alpine` (or unprivileged) are floating tags.

- [ ] Pin both `FROM` lines to `@sha256:<digest>`. **OWNER-MANUAL (2026-07-04):** resolving the digest needs registry access and belongs with the owner's manual image build/tag/publish step.
- **Acceptance:** Reproducible builds; no floating tags.

### I4 — Fix the npm install command [BOTH]

`npm ci --only=production=false` uses a deprecated flag.

- [x] Replace with plain `npm ci`. (DevDeps are already absent from the final image since only `dist/` is copied to the nginx stage.)
- **Acceptance:** Install succeeds without deprecation warnings.

### I5 — Externalize a committed `nginx.conf` with SPA fallback, cache, and security headers [BOTH]

The inline `RUN echo` nginx config has SPA fallback + gzip but no cache or security headers.

- [x] Replace the inline config with a committed `nginx.conf` that keeps `try_files $uri $uri/ /index.html`, adds `Cache-Control: public, max-age=31536000, immutable` for `/assets/*`, and adds `Content-Security-Policy`, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin` (satisfies B6).
- **Acceptance:** Deep-link refresh works; hashed assets are long-cached; all four security headers present.

### I7 — Runtime configuration for published images (CRITICAL) [BOTH]

Vite bakes `import.meta.env` at **build** time, so a published Docker Hub image is hardcoded to one API URL — a self-hoster cannot point it at their own backend without rebuilding.

- [x] Add runtime config injection: a small container entrypoint that reads the API/identity base-URL env vars at **container start** and writes a `config.js` (or serves a `/config` document) that the app loads before bootstrap; read base URLs from that runtime config, with the build-time value as fallback.
- **Acceptance:** A self-hoster runs the published image and points it at their own backend purely via container env, with no rebuild.

---

# J — Open-source readiness

### J1 — Reconcile both READMEs with reality [BOTH]

CONSOLE README config table (`:141,145,174`) and IDENTITY README (`:130,134`) document wrong API URLs and a nonexistent `maintainerd-auth/nginx/` path, and omit env vars present in `.env`.

- [x] Rewrite each README config section to match the real `.env.example` variables (all three: API base, public API base, identity base), add a concrete "Run via Docker" section, and remove the dead `maintainerd-auth/nginx/` reference.
- **Acceptance:** A new contributor can configure and run each app from the README alone.

### J2 — Stamp versions to 0.1.0 [BOTH]

Both `package.json` are `0.0.0`.

- [x] Set `"version": "0.1.0"` in both.
- **Acceptance:** Both report 0.1.0.

### J3 — Pin Node version consistently [BOTH]

No `engines.node` and no `.nvmrc`; CI uses Node 20, Dockerfile uses 22.

- [x] Add `"engines": { "node": ">=22 <23" }` to both `package.json`, add a `.nvmrc` containing `22`, and update CI to Node 22 so CI/Docker/local all match (consistent with the backend tracker's Node 22 choice).
- **Acceptance:** One Node version across CI, Docker, and local.

### J4 — Add community-health files to both repos [BOTH]

Both lack `CONTRIBUTING.md`, `SECURITY.md`, `CHANGELOG.md`, issue/PR templates, `CODEOWNERS`, and `dependabot.yml`.

- [x] Add to each: `CONTRIBUTING.md` (setup, lint/build gates, conventions), `SECURITY.md` (disclosure contact), `CHANGELOG.md` (seed `## [0.1.0]`), `.github/ISSUE_TEMPLATE/{bug_report.yml,feature_request.yml}`, `.github/PULL_REQUEST_TEMPLATE.md`, `.github/CODEOWNERS`, `.github/dependabot.yml` (npm + github-actions).
- **Acceptance:** GitHub surfaces all community-health files.

### J5 — Confirm LICENSE/NOTICE consistency [BOTH]

Both LICENSEs match the backend's Apache-2.0 byte-for-byte and NOTICE is present — verified.

- [x] Confirm the LICENSE/NOTICE remain present and the README License section agrees.
- **Acceptance:** Apache-2.0 consistent across all repos.

### J6 — Confirm no secrets or wrong branding committed [BOTH]

`.env` is gitignored (only `.env.example` tracked) and no secrets in `src` — verified.

- [x] Confirm `.env.example` has only safe placeholders; confirm "LulaLife" appears nowhere (use "Lula").
- **Acceptance:** No secrets tracked; branding correct.

### J7 — Ensure CI gates lint + typecheck + build on PRs [BOTH]

Both have a CI workflow running lint + test + build.

- [x] Confirm each runs `npm run lint`, `tsc -b` (typecheck), and `npm run build` on PRs to the default branch and that they are required checks. Add the missing image-build workflow (I6).
- **Acceptance:** PRs are gated; a red build/lint/typecheck blocks merge.

### J8 — Each frontend README links the stack [BOTH]

- [x] Add/repair "Related projects" links so each frontend README points to the backend (`maintainerd-auth`), the dev environment (`maintainerd-dev`), and the sibling frontend, using correct repo paths.
- **Acceptance:** All cross-links resolve.

---

# K — Responsiveness, cross-browser & testing

### K1 — Responsive / mobile layouts [BOTH]

The hosted login is user-facing and will be opened on phones; the console must be usable on tablet/desktop.

- [ ] IDENTITY: verify every auth screen (login, registration, invite, MFA, consent, recovery, account) renders and is usable at mobile widths (320–414px), tablet, and desktop; fix overflow, tap-target size, and viewport meta.
- [ ] CONSOLE: verify the admin layout is usable down to tablet width; tables scroll/stack rather than break.
- **DEFERRED (2026-07-04):** static structure is in place (viewport meta, responsive breakpoints, scrollable `DataTable`). An automated Playwright responsive-viewport spec was built but removed with the Playwright tooling; responsive verification is deferred to the upcoming Playwright suite + manual check.
- **Acceptance:** No broken layout or unreachable control across mobile/tablet/desktop.

### K2 — Cross-browser verification [BOTH]

- [ ] Verify both apps on current Chrome, Firefox, Safari, and Edge — especially WebAuthn/passkey ceremonies, clipboard, and cookie behavior (SameSite) on Safari. **DEFERRED (2026-07-04):** cross-browser Playwright projects (chromium/firefox/webkit) were built but removed with the Playwright tooling; cross-browser verification is deferred to the upcoming Playwright suite (Playwright expert) + manual Safari/Edge checks.
- **Acceptance:** Core flows work on all four browsers; document any known limitation.

### K3 — Frontend unit tests + coverage gate [BOTH]

- [x] Add/extend unit tests for components, hooks, and API service functions (form validation, error handling, redirect-safety utils, auth/session logic). Add a coverage threshold to CI that fails below target for changed files.
- **Acceptance:** Critical components/hooks/services are tested and CI enforces coverage.

### K4 — Automated end-to-end tests (Playwright) [BOTH]

- [ ] Add a Playwright E2E suite covering core journeys: identity login, OAuth authorize→consent→token, registration (normal/flow/invite), MFA enroll + login, password reset, lockout/429; console login + representative admin CRUD. Wire into CI. **DEFERRED (2026-07-04):** a Playwright E2E suite (backend-less journeys + stack-gated stubs) was built and passing, then **removed at the owner's request** — Playwright E2E will be re-introduced by a Playwright expert. For now Vitest unit tests (K3) are the automated test gate.
- **Acceptance:** Core journeys are covered by automated E2E in CI (shared with backend tracker J3).

### K5 — Bundle budget + Lighthouse pass [BOTH]

- [ ] After route-based code splitting (F4), confirm the initial bundle is within a documented budget; run Lighthouse on the identity login and a console page and meet agreed performance + accessibility scores. **PARTIAL / DEFERRED (2026-07-04):** bundle budget is DONE and documented in each repo's `docs/performance.md` — **no chunk exceeds 500 KB** (console largest: ui-vendor 201 KB, data-vendor 155 KB; identity largest 422 KB), measured from `npm run build`. The **Lighthouse CI gate was built (a11y 1.00) but removed with the Playwright/browser tooling** — the Lighthouse run is deferred to the Playwright/expert work.
- **Acceptance:** Bundle within budget; Lighthouse perf + a11y meet the documented threshold.

---

# L — Frontend release gate (run last)

### L1 — Both apps build clean [BOTH]

- [x] In each repo: `npm run lint` clean, `tsc -b` passes, `npm run build` succeeds with no errors or warnings.

### L3 — End-to-end smoke through nginx [BOTH]

- [ ] Via the local nginx hosts (console.auth.maintainerd.local, identity.auth.maintainerd.local), exercise: console login (cookie-based) + every admin CRUD + the newly wired admin operability features; identity login, OAuth authorize→token, registration (normal/flow/invite), MFA enroll + login, SMS login, backup-code recovery, linked identities, lockout/429 screens, logout. **RUNTIME GATE (owner-run):** requires the full stack up.

### L4 — Backend contract alignment verified [BOTH]

- [x] Confirm every FE API call matches a real backend route (verb/path/fields), including the C1/C2 fixes and the A1 connections changes. No 404/405 from the UI. **DONE (2026-07-04):** a full cross-repo static sweep mapped every FE call (console→:8080, identity→:8081) to the backend chi route trees. C1 (tenant status PUT) and A1 (`/oauth/connections?registration_flow=`) confirmed resolved. Three real mismatches were found and fixed: (a) identity's setup wizard hit internal-only `/setup/*` on the public surface + `/setup/create_profile` existed nowhere → **setup wizard removed from identity** (setup is a console/internal concern); (b) console's admin identity-unlink hit a nonexistent route → **added `DELETE /users/{user_uuid}/identities/{identity_uuid}` on the internal API** (handler + service + tests + step-up, `go test`/`golangci-lint` green); (c) identity's dead admin tenant-CRUD service functions (internal-only, no call sites) **removed**. The final "no 404/405 from the UI" observation folds into the L3 runtime smoke.

### L5 — Responsive, cross-browser & test gates pass [BOTH]

- [ ] Confirm Section K passes: responsive/mobile, cross-browser, unit coverage, Playwright E2E, and bundle/Lighthouse thresholds. **PARTIAL (2026-07-04):** K3 (Vitest unit coverage) is the active automated test gate and passes; the K5 bundle budget is met. K1/K2/K4 and the K5 Lighthouse run are **deferred** to the upcoming Playwright suite (Playwright expert) — see those items.

### L6 — Secret & dead-code scan [BOTH]

- [x] Run a secret scan (Gitleaks) over both repos and history; confirm no secrets. Confirm dead code from Section H is removed and `npm run build` reports no unused-export warnings. **DONE (2026-07-04):** both repos already run a **blocking full-history Gitleaks job** (`security.yml` — `fetch-depth: 0`, `--exit-code 1`, on push/PR + daily cron), plus Semgrep `p/secrets`. Section H dead code is removed and both builds are clean with no unused-export warnings.

> **Tag & publish (removed 2026-07-04):** committing to `main`, tagging `v0.1.0`, and building/pushing/scanning/signing Docker Hub images are done **manually by the owner** after manual E2E — intentionally out of this checklist.

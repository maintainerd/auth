# Frontend Adjustments — Backend Alignment Checklist

Companion to `develop-before-v0.1.0-frontend.md`. This document is the **authoritative gap analysis** of every backend change made during the v0.1.0 development cycle and exactly what each frontend must do to stay aligned.

**Date audited:** 2026-07-08  
**Backend state:** post all planning-doc passes (registration-flows-remaining, develop-before-v0.1.0.md, develop-before-v0.1.0-frontend.md)  
**Frontends audited:** `maintainerd-auth-console` and `maintainerd-auth-identity`

---

## How to use this document

- Work **P0 first**, then P1, then P2.
- Each item shows: the change, the files to touch, the exact steps, and the acceptance criteria.
- "Already aligned" items are documented for completeness — do not re-implement them.
- After every change: `npm run lint`, `tsc -b`, `npm run build` must all pass green in the affected repo.

---

## Progress summary

- [ ] **CONSOLE — P0 (3 items):** invite callback_url · auth-events export · branding logo upload
- [ ] **CONSOLE — P1 (3 items):** auth-events count · logo URL prefixing · invite/event-route detail endpoints
- [ ] **CONSOLE — P2 (1 item):** keyset pagination awareness
- [ ] **IDENTITY — P0 (0 items):** all critical flows are already aligned
- [ ] **IDENTITY — P1 (1 item):** registration flow `required_fields` rendering verification
- [ ] **CONSOLE — REMOVALS:** API keys full feature removal (Part 7)
- [ ] **CONSOLE — SCHEMA TYPES:** TypeScript type updates from DB schema changes (Part 8)
- [ ] **CONSOLE — NEW FEATURES:** New backend entities needing console pages (Part 9)
- [ ] **CONSOLE — SIDENAV:** Navigation hierarchy update (Part 10)
- [ ] **Verification pass:** API-by-API smoke against the running stack

---

## Part 1 — Already implemented (do not re-implement)

These backend changes are **already reflected** in the frontends. This section exists so the implementer does not waste time or introduce regressions by re-implementing them.

### Console — already aligned

| Area | Backend change | Console state |
|---|---|---|
| Client `branding_id` + `allow_registration` | Added to `clients` table + all DTOs | `CreateClientRequest` / `UpdateClientRequest` / `Client` types all carry these fields; client form wires them |
| Registration flows rename (`auth_flow` → `registration_flow`) | Tables, routes, DTOs renamed | Console routes, sidebar, API service, types, hooks, validation all say "Registration Flow" |
| `PUT /tenants/{id}/status` verb | Backend is `PUT` (not `PATCH`) | `updateTenantStatus` uses `put()` — fixed as C1 |
| Webhook deliveries `GET /webhook-endpoints/{id}/deliveries` | Added in backend C4 | `fetchDeliveryHistory()` present in `webhooks/index.ts` |
| Webhook replay `POST /webhook-replay` | Existing | `replayDelivery()` present |
| User admin identity unlink `DELETE /users/{id}/identities/{identityId}` | Added in backend L4 | `unlinkUserIdentity()` present in `users/index.ts` |
| Admin phone verify `PATCH /users/{id}/verify-phone` | Added to user routes | `verifyUserPhone()` present in `users/index.ts` |
| IdP test-connection `POST /identity_providers/test` | Existing | "Test Connection" button wired on IdP form |
| Token federation badge | `allow_token_federation` in list DTO | Badge column present in `IdentityProviderColumns.tsx` |
| Client↔IdP connection edit `PUT /clients/{id}/identity_providers/{connId}` | Existing | Per-row edit panel in `ClientIdentityProviders.tsx` |
| Standalone Permissions page `/permissions` | Existing CRUD | Page + routes wired |
| Force password change `PUT /users/{id}/force-password-change` | Existing | Action on user detail page |
| `setup/create_profile` removed | Endpoint never existed | Removed; setup profile goes through `/profile` |
| `PATCH /webhook-endpoints/{id}/status` verb | Backend is `PATCH` | Uses `patch()` |
| Step-up on API permission mutations | Middleware present | Step-up dialog triggered correctly |

### Identity — already aligned

| Area | Backend change | Identity state |
|---|---|---|
| `POST /oauth/authorize/continue` | Added in D8 | `continueOAuth(requestId)` calls `POST /oauth/authorize/continue`; `RegisterForm` calls it after registration when `request_id` in URL |
| `ApiError.requestId` extraction | `request_id` in login_required error body | `client.ts` parses `data.request_id` into `ApiError.requestId`; `OAuthAuthorizePage` reads it and redirects to `/register?request_id=...` on `screen_hint=signup` |
| `OAuthConnections` branding field | `ClientBrandingResponse` added | `AppBootstrap` reads `connections.branding` and applies via `applyBranding()`; falls back to `currentTenant.branding` |
| `verification_required` + `required_fields` removed from connections | Removed from DTO | Identity `OAuthConnections` type has neither field |
| Invite `callback_url` from invite-context endpoint | `GET /invite?invite_token=` returns `callback_url` | `fetchInviteContext()` + `RegisterInviteForm` reads `ctx.callback_url`; stored in sessionStorage; consumed by `LoginSuccessPage` |
| `safeExternalRedirect` for invite callbacks | Exact-match validator on backend | Identity uses `safeExternalRedirect` (https-only) on all invite callbacks |
| Registration flow `registration_flow` param on `POST /register` | Backend thread through auth service | `auth/index.ts` appends `registration_flow` query param when `data.registrationFlow` is set |
| `POST /account/phone/send-verification` + `POST /account/phone/verify` | Added in D1/backend D1 | Endpoint constants defined in `config.ts`; phone verification page at `/account/phone` |
| MFA self-service enrollment (all factors) | Existing MFA routes | Full enrollment UI at `/account/mfa` |
| SMS passwordless login `POST /sms-login/send` + `/verify` | Existing routes | SMS login page at `/sms-login` |
| Backup code recovery `POST /recovery/backup-code` | Existing | Recovery page at `/recovery` |
| Linked identities `GET/POST/DELETE /account/identities*` | Existing | Linked identities page at `/account/identities` |
| Account locked (423) + rate limit (429) screens | Existing error codes | Dedicated `/account-locked` + `/too-many-requests` pages; axios interceptor redirects |
| `screen_hint=signup` on OAuth authorize → redirect to `/register?request_id=...` | `screen_hint` accepted on authorize | `OAuthAuthorizePage` reads `screen_hint` from params and routes correctly |
| Setup wizard removed from identity | Internal-only concern | Identity has no `/setup/*` routes |

---

## Part 2 — Console changes required

### P0 — C-CON-01 · Invite form: add `callback_url` field

**What:** The backend `SendInviteRequest` DTO accepts an optional `callback_url` (validated exact-match against the flow's client redirect URIs at creation time). The console type and form do not include it.

**Why:** Without this, admins cannot set a post-invite-registration redirect. The backend will silently ignore any callback, and the invited user lands on the identity default post-login screen.

**Backend contract:**
```
POST /invite
Body: { email: string, registration_flow_uuid?: string, callback_url?: string }
Validation: if callback_url is set AND a registration_flow_uuid is set, the URL must be an exact registered redirect_uri of the flow's linked client.
```

**Files to change:**

- [ ] **`src/services/api/invites/types.ts`**
  - Add `callback_url?: string` to `SendInviteRequest`:
    ```ts
    export interface SendInviteRequest {
      email: string
      registration_flow_uuid?: string
      callback_url?: string
    }
    ```

- [ ] **`src/pages/invites/form/InviteForm.tsx`**
  - After the registration flow selector, add an optional "Post-registration callback URL" input field.
  - The field should only be enabled when a `registration_flow_uuid` is selected (because the URL is validated against that flow's client redirect URIs).
  - Add to Yup schema: `callback_url: yup.string().url('Must be a valid URL').optional()`
  - Add to `onSubmit` payload: `callback_url: data.callback_url || undefined`
  - Add helper text: "Must exactly match a redirect URI registered on the flow's client. Leave blank to use the identity app's default post-registration screen."

- [ ] **`src/hooks/useInvites.ts`** (or wherever the `sendInvite` mutation lives)
  - Confirm `sendInvite(data: SendInviteRequest)` passes through `callback_url` — no change needed if the payload is passed directly.

**Acceptance:** Creating an invite with a `callback_url` that does not match a registered redirect URI returns a 400 from the backend (surfaced as an error toast). A valid URL is stored and the invited user is redirected there after account completion.

---

### P0 — C-CON-02 · Auth events export: add service function + export button

**What:** The backend has a fully implemented `GET /auth-events/export?format=csv|json` endpoint. The console has no service function for it, no endpoint constant, and no UI trigger.

**Backend contract:**
```
GET /auth-events/export?format=csv   → application/octet-stream, Content-Disposition: attachment; filename="auth_events.csv"
GET /auth-events/export?format=json  → application/octet-stream, filename="auth_events.json"
Query params: same filters as GET /auth-events (user_id, event_type, from, to, etc.)
Auth: requires management client session (internal :8080)
```

**Files to change:**

- [ ] **`src/services/api/config.ts`**
  - Add endpoint constant:
    ```ts
    AUTH_EVENTS_EXPORT: '/auth-events/export',
    ```

- [ ] **`src/services/api/auth-events/index.ts`**
  - Add export function that triggers a file download:
    ```ts
    export async function exportAuthEvents(
      format: 'csv' | 'json',
      params?: Omit<AuthEventQueryParams, 'page' | 'limit'>
    ): Promise<void> {
      const queryParams = new URLSearchParams({ format })
      if (params?.user_id) queryParams.set('user_id', params.user_id)
      if (params?.event_type) queryParams.set('event_type', params.event_type)
      if (params?.from) queryParams.set('from', params.from)
      if (params?.to) queryParams.set('to', params.to)

      const response = await apiClient.get(
        `${API_ENDPOINTS.AUTH_EVENTS_EXPORT}?${queryParams.toString()}`,
        { responseType: 'blob' }
      )
      const filename = format === 'csv' ? 'auth_events.csv' : 'auth_events.json'
      const url = URL.createObjectURL(new Blob([response.data]))
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      a.click()
      URL.revokeObjectURL(url)
    }
    ```
  - Note: Use `responseType: 'blob'` so axios does not try to JSON-parse the stream.

- [ ] **`src/pages/logs/AuthEventListing.tsx`**
  - Add an "Export" dropdown button (CSV / JSON) in the listing toolbar.
  - On click, call `exportAuthEvents(format, currentFilters)`.
  - Show a loading state on the button during download; surface errors via toast.
  - The button should respect the currently applied filters (date range, event type, user filter) so the export matches what is visible on screen.

**Acceptance:** Clicking "Export CSV" downloads `auth_events.csv` with the filtered data. Clicking "Export JSON" downloads `auth_events.json`. If the export fails, a toast error appears and no file download starts.

---

### P0 — C-CON-03 · Branding: add logo file upload support

**What:** The backend create/update branding handlers accept optional `logo_data` (base64-encoded image) and `logo_content_type` fields. When present, the backend stores the binary in the DB and sets `logo_url` to the relative path `/public/branding/{uuid}/logo`. The console `BrandingRequest` type and the branding form only support a URL string — binary upload is completely absent.

Additionally, when the backend returns `logo_url` as the relative path `/public/branding/{uuid}/logo`, the console cannot display it correctly because that path resolves against the console's own origin, not the public backend. The path must be prefixed with `PUBLIC_BASE_URL`.

**Backend contract (create/update branding):**
```
POST /branding    (authenticated, :8080)
PUT  /branding/{uuid}

Body (JSON):
{
  name: string,
  layout: string,
  company_name: string,
  logo_url?: string,          // pass a URL directly (external or the path returned by backend)
  logo_data?: string,         // base64-encoded image bytes (PNG/JPEG/WebP, max 256 KB)
  logo_content_type?: string, // "image/png" | "image/jpeg" | "image/webp"
  favicon_url?: string,
  support_url?: string,
  privacy_policy_url?: string,
  terms_of_service_url?: string,
  metadata?: BrandingMetadata
}
When logo_data is accepted, backend sets logo_url = "/public/branding/{uuid}/logo"
```

**Backend contract (serve logo — public, no auth):**
```
GET /public/branding/{branding_uuid}/logo
→ streams binary with Content-Type, ETag, Cache-Control
Base: PUBLIC_BASE_URL (not the internal API)
```

**Files to change:**

- [ ] **`src/services/api/branding/types.ts`**
  - Add the two new fields to `BrandingRequest`:
    ```ts
    export interface BrandingRequest {
      name: string
      layout: string
      company_name: string
      logo_url?: string
      logo_data?: string         // base64-encoded; mutually exclusive with logo_url
      logo_content_type?: string // "image/png" | "image/jpeg" | "image/webp"
      favicon_url?: string
      support_url?: string
      privacy_policy_url?: string
      terms_of_service_url?: string
      metadata?: BrandingMetadata
    }
    ```

- [ ] **`src/utils/branding.ts`** (or `src/lib/utils/` — wherever URL helpers live)
  - Add a helper to resolve backend-relative logo URLs to full public API URLs:
    ```ts
    export function resolveBrandingLogoUrl(logoUrl: string | null | undefined): string | null {
      if (!logoUrl) return null
      if (logoUrl.startsWith('/public/branding/')) {
        return `${API_CONFIG.PUBLIC_BASE_URL.replace('/api/v1', '')}${logoUrl}`
      }
      return logoUrl
    }
    ```
  - The backend's `PUBLIC_BASE_URL` constant (from `config.ts`) is `https://public-api.auth.maintainerd.local/api/v1`; strip `/api/v1` to get the origin.

- [ ] **All places in the console that render `branding.logo_url` in an `<img>` tag**
  - Replace direct `src={branding.logo_url}` with `src={resolveBrandingLogoUrl(branding.logo_url) ?? ''}`.
  - Files likely affected: branding list/detail/preview components, the console theme provider (if it injects the logo into the sidebar/navbar), and client detail pages that display the client's branding logo.

- [ ] **Branding create/edit form** (find the form component under `src/pages/branding/`)
  - Replace the plain "Logo URL" text input with a dual-mode field:
    - **Option A — URL:** text input for an external URL (existing behavior).
    - **Option B — File upload:** file input accepting `image/png,image/jpeg,image/webp`, max 256 KB.
  - When a file is selected:
    1. Validate MIME type is `image/png`, `image/jpeg`, or `image/webp`.
    2. Validate size ≤ 256 KB (262 144 bytes). Show a clear error if over limit.
    3. Convert to base64: `const base64 = await toBase64(file)` (implement a `toBase64(File): Promise<string>` helper using `FileReader`).
    4. Set `logo_data = base64` and `logo_content_type = file.type` in the form state.
    5. Clear `logo_url` (the two modes are mutually exclusive).
  - When a URL is typed, clear `logo_data` and `logo_content_type`.
  - Show a preview of the selected image before saving.
  - On form submit, include `logo_data` + `logo_content_type` in the `BrandingRequest` payload if a file was chosen.

**Acceptance:** An admin can upload a PNG/JPEG/WebP logo ≤ 256 KB via the branding form. The logo is stored in the DB and displayed correctly in both the console and the identity app. A file over 256 KB or of the wrong type shows a validation error before submission. Brandings with external `logo_url` strings continue to work unchanged.

---

### P1 — C-CON-04 · Auth events count endpoint

**What:** The backend exposes `GET /auth-events/count?event_type=<type>` returning `{ count: number }`. Useful for dashboard counters or event-type metrics.

**Backend contract:**
```
GET /auth-events/count?event_type=<event_type_string>
→ { success: true, data: { count: number }, message: "Auth event count retrieved successfully" }
Auth: internal :8080
```

**Files to change:**

- [ ] **`src/services/api/config.ts`**
  - Add: `AUTH_EVENTS_COUNT: '/auth-events/count'`

- [ ] **`src/services/api/auth-events/index.ts`**
  - Add:
    ```ts
    export async function fetchAuthEventCount(eventType: string): Promise<number> {
      const response = await get<ApiResponse<{ count: number }>>(
        `${API_ENDPOINTS.AUTH_EVENTS_COUNT}?event_type=${encodeURIComponent(eventType)}`
      )
      return response.data?.count ?? 0
    }
    ```

- [ ] **`src/hooks/useAuthEvents.ts`** (if it exists) or a new `useAuthEventCount.ts`
  - Add a TanStack Query hook:
    ```ts
    export function useAuthEventCount(eventType: string) {
      return useQuery({
        queryKey: ['auth-events', 'count', eventType],
        queryFn: () => fetchAuthEventCount(eventType),
        enabled: !!eventType,
      })
    }
    ```

- [ ] **Dashboard page / summary** — wire count queries for key event types (e.g., `login_success`, `login_failure`, `mfa_enrolled`) if the dashboard design calls for per-event-type metrics.

**Acceptance:** `useAuthEventCount('login_failure')` returns a number that matches the backend's stored count.

---

### P1 — C-CON-05 · Invite detail: add `fetchInviteById` service function

**What:** Backend C7 added `GET /invite/{invite_uuid}` (authenticated, tenant-scoped) so console invite detail pages can load directly by ID without depending on the list. The console does not have a `fetchInviteById` function.

**Backend contract:**
```
GET /invite/{invite_uuid}   (internal :8080, management client required)
→ ApiResponse<InviteResponse>  (same shape as list items)
```

**Files to change:**

- [ ] **`src/services/api/invites/index.ts`**
  - Add:
    ```ts
    export async function fetchInviteById(inviteId: string): Promise<InviteResponse> {
      const response = await get<ApiResponse<InviteResponse>>(
        `${API_ENDPOINTS.INVITE}/${inviteId}`
      )
      return unwrap(response, 'fetch invite')
    }
    ```

- [ ] **`src/hooks/useInvites.ts`**
  - Add:
    ```ts
    export function useInvite(inviteId: string | undefined) {
      return useQuery({
        queryKey: ['invites', inviteId],
        queryFn: () => fetchInviteById(inviteId!),
        enabled: !!inviteId,
      })
    }
    ```

- [ ] **Invite detail page** (`src/pages/invites/:inviteId/InviteDetailPage.tsx` or equivalent)
  - Replace any approach that requires the list to be loaded first; use `useInvite(inviteId)` directly.

**Acceptance:** Navigating directly to `/:tenantId/invites/:inviteId` loads the invite detail without first fetching the full list.

---

### P1 — C-CON-06 · Event routes detail: add `fetchEventRouteById` service function

**What:** Backend C7 added `GET /event-routes/{uuid}` for event route detail. Verify whether this endpoint is already called; if not, add the service function.

**Backend contract:**
```
GET /event-routes/{route_uuid}   (internal :8080)
→ ApiResponse<EventRoute>
```

**Files to change:**

- [ ] **Locate the event routes service file** (likely `src/services/api/event-routes/index.ts`)
  - If `fetchEventRouteById(routeId: string)` does not exist, add it:
    ```ts
    export async function fetchEventRouteById(routeId: string): Promise<EventRoute> {
      const response = await get<ApiResponse<EventRoute>>(
        `${API_ENDPOINTS.EVENT_ROUTES}/${routeId}`
      )
      return unwrap(response, 'fetch event route')
    }
    ```

- [ ] Add a corresponding TanStack Query hook if event route detail pages load by ID.

**Acceptance:** Navigating directly to `/:tenantId/events/routes/:routeId` loads the route without requiring the full list to be loaded first.

---

### P2 — C-CON-07 · Auth events + users: verify keyset pagination compatibility

**What:** The backend switched `auth_events` list and user list queries to keyset pagination (cursor-based) for performance at 1M+ rows. The standard response now includes `next_cursor` alongside `rows`. The console currently uses offset pagination (`page`, `total_pages`).

**Action required — verify, do not blindly implement:**

- [ ] Make a real `GET /auth-events` and `GET /users` request against the running backend and inspect the exact JSON response shape.
  - If response still includes `{ rows, total, page, limit, total_pages }` → no change needed (the keyset is internal, the HTTP shape is unchanged).
  - If response returns `{ rows, next_cursor }` (without `total_pages`) → the console pagination controls break and must be updated:
    - Replace page-number pagination with "Load more" / infinite scroll using `next_cursor`.
    - Update `AuthEventQueryParams` and `UserQueryParams` types to add `after_id?: number` and remove `page?: number` for these two resources.
    - The query cache key must include `after_id` to correctly cache pages.

**Note:** Low-volume admin tables (clients, roles, tenants, permissions, etc.) still use offset pagination — no change for those.

---

## Part 3 — Identity changes required

### P1 — I-ID-01 · Registration flow `required_fields` rendering — verify end-to-end

**What:** When a user reaches `/register` via an OAuth authorize redirect with `screen_hint=signup&registration_flow=<identifier>`, the backend threads the `registration_flow` identifier into `POST /register` as a query parameter, enforces the flow's `required_fields`, and applies `verification_required`. The identity frontend passes `registration_flow` correctly, but the form rendering of dynamic `required_fields` (e.g., making `phone` or `fullname` required vs optional) should be verified.

**Backend behavior:**
- `registration_flow` is an identifier string passed as `?registration_flow=<id>` on `POST /register`.
- The backend enforces `required_fields` server-side (rejects missing fields with 422).
- A safe public context endpoint exists: `GET /oauth/connections?client_id=<id>&registration_flow=<id>` does NOT return required_fields (they were removed from the connections response). The registration flow's required fields are now enforcement-only on the backend.

**Files to verify:**

- [ ] **`src/pages/register/components/RegisterForm.tsx`**
  - Confirm: when `registration_flow` is present in URL params, the form does **not** try to fetch required_fields from `connections` (that field no longer exists in the response).
  - Confirm: `fullname` and `phone` fields are always rendered (both optional in the baseline registration form) — the backend enforces any "required" constraints server-side and returns 422 if a required field is missing.
  - The frontend does not need to dynamically hide/show fields based on a flow's required_fields; it shows all baseline fields and lets the backend validate.
  - If any code still reads `connections.required_fields` or `connections.verification_required`, delete it.

- [ ] **`src/services/api/oauth/types.ts`**
  - Confirm `OAuthConnections` does NOT have `verification_required` or `required_fields` fields. If they exist, delete them.

- [ ] **`src/hooks/useOAuthConnections.ts`**
  - Confirm the hook does not consume or re-export `verification_required` / `required_fields`.

- [ ] **`src/components/auth/RouteGuard.tsx`**
  - Confirm no guard logic reads `connections.data?.verification_required`. (Was removed as part of planning doc A1 — verify it stayed removed.)

**Acceptance:** A user visiting `/oauth/authorize?...&screen_hint=signup&registration_flow=seller` is redirected to `/register?request_id=...&registration_flow=seller`. The registration form renders normally. After submitting all fields, `POST /register?client_id=...&registration_flow=seller` is called. If the flow requires `phone` and the user left it blank, the backend returns 422 which surfaces as a form error.

---

## Part 4 — API-by-API alignment table

Cross-reference of every public/internal API endpoint against what each frontend calls. Use this during the live-stack smoke test (release gate L3).

### Internal API (:8080) — console calls

| Endpoint | Console service function | Status |
|---|---|---|
| `GET /setup/status` | `fetchSetupStatus()` | ✅ Aligned |
| `POST /setup/create_tenant` | `createTenantSetup()` | ✅ Aligned |
| `POST /setup/create_admin` | `createAdminSetup()` | ✅ Aligned |
| `POST /setup/complete` | `completeSetup()` | ✅ Aligned |
| `GET /account` | `fetchAccount()` / Redux `initializeAuthAsync` | ✅ Aligned |
| `GET /profile` | `fetchProfile()` | ✅ Aligned |
| `POST /profile` | `createProfile()` | ✅ Aligned |
| `GET /user-settings` | `fetchUserSettings()` | ✅ Aligned |
| `POST /user-settings` | `updateUserSettings()` | ✅ Aligned |
| `GET /account/sessions` | `fetchAccountSessions()` | ✅ Aligned |
| `DELETE /account/sessions/{id}` | `revokeAccountSession()` | ✅ Aligned |
| `DELETE /account/sessions` | `revokeAllAccountSessions()` (step-up) | ✅ Aligned |
| `POST /account/email/change` | `changeEmail()` | ✅ Aligned |
| `POST /account/email/verify` | `verifyEmailChange()` | ✅ Aligned |
| `PUT /account/username` | `updateUsername()` | ✅ Aligned |
| `GET /account/export` | `exportAccountData()` | ✅ Aligned |
| `DELETE /account` | `deleteAccount()` (step-up) | ✅ Aligned |
| `GET /mfa/status` | `fetchMFAStatus()` | ✅ Aligned |
| `POST /mfa/totp/enroll` | `enrollTOTP()` | ✅ Aligned |
| `POST /mfa/totp/verify` | `verifyTOTP()` | ✅ Aligned |
| `DELETE /mfa/totp` | `disableTOTP()` | ✅ Aligned |
| `GET /mfa/backup-codes/count` | `getBackupCodesCount()` | ✅ Aligned |
| `POST /mfa/backup-codes/regenerate` | `regenerateBackupCodes()` | ✅ Aligned |
| `POST /mfa/webauthn/register/begin` | `beginWebAuthnRegistration()` | ✅ Aligned |
| `POST /mfa/webauthn/register/finish` | `finishWebAuthnRegistration()` | ✅ Aligned |
| `POST /mfa/webauthn/auth/begin` | `beginWebAuthnAuth()` | ✅ Aligned |
| `DELETE /mfa/webauthn/{uuid}` | `deleteWebAuthnCredential()` | ✅ Aligned |
| `POST /mfa/sms/enroll` | `enrollSMS()` | ✅ Aligned |
| `POST /mfa/sms/verify` | `verifySMSEnroll()` | ✅ Aligned |
| `DELETE /mfa/sms` | `disableSMS()` | ✅ Aligned |
| `POST /mfa/email-otp/enroll` | `enrollEmailOTP()` | ✅ Aligned |
| `POST /mfa/email-otp/verify` | `verifyEmailOTPEnroll()` | ✅ Aligned |
| `DELETE /mfa/email-otp` | `disableEmailOTP()` | ✅ Aligned |
| `POST /mfa/reset` | `resetMFA()` | ✅ Aligned |
| `POST /mfa/step-up/challenge` | `requestStepUpChallenge()` | ✅ Aligned |
| `POST /mfa/step-up/send-sms` | `sendStepUpSMS()` | ✅ Aligned |
| `POST /mfa/step-up/send-email-otp` | `sendStepUpEmailOTP()` | ✅ Aligned |
| `POST /mfa/step-up/verify` | `verifyStepUp()` | ✅ Aligned |
| `POST /mfa/admin/users/{id}/reset` | `adminResetMFA()` | ✅ Aligned |
| `POST /mfa/admin/users/{id}/reset/{method}` | `adminResetMFAMethod()` | ✅ Aligned |
| `GET /tenant` | Redux `fetchDefaultTenantAsync` | ✅ Aligned |
| `GET /tenant/{identifier}` | Redux `fetchTenantByIdentifierAsync` | ✅ Aligned |
| `GET /tenants` | `fetchTenants()` | ✅ Aligned |
| `GET /tenants/{id}` | `fetchTenantById()` | ✅ Aligned |
| `POST /tenants` | `createTenant()` | ✅ Aligned |
| `PUT /tenants/{id}` | `updateTenant()` | ✅ Aligned |
| `PUT /tenants/{id}/status` | `updateTenantStatus()` (uses `put`) | ✅ Aligned (C1 fix) |
| `DELETE /tenants/{id}` | `deleteTenant()` | ✅ Aligned |
| `GET /tenants/{id}/members` | `fetchTenantMembers()` | ✅ Aligned |
| `POST /tenants/{id}/members` | `addTenantMember()` | ✅ Aligned |
| `PATCH /tenants/{id}/members/{id}/role` | `updateTenantMemberRole()` | ✅ Aligned |
| `DELETE /tenants/{id}/members/{id}` | `removeTenantMember()` | ✅ Aligned |
| `GET /users` | `fetchUsers()` | ✅ Aligned |
| `GET /users/{id}` | `fetchUserById()` | ✅ Aligned |
| `POST /users` | `createUser()` | ✅ Aligned |
| `PUT /users/{id}` | `updateUser()` | ✅ Aligned |
| `DELETE /users/{id}` | `deleteUser()` | ✅ Aligned |
| `PATCH /users/{id}/status` | `updateUserStatus()` | ✅ Aligned |
| `GET /users/{id}/roles` | `fetchUserRoles()` | ✅ Aligned |
| `POST /users/{id}/roles` | `addUserRoles()` | ✅ Aligned |
| `DELETE /users/{id}/roles/{id}` | `removeUserRole()` | ✅ Aligned |
| `GET /users/{id}/identities` | `fetchUserIdentities()` | ✅ Aligned |
| `DELETE /users/{id}/identities/{id}` | `unlinkUserIdentity()` | ✅ Aligned |
| `GET /users/{id}/profiles` | `fetchUserProfiles()` | ✅ Aligned |
| `POST /users/{id}/profiles` | `createUserProfile()` | ✅ Aligned |
| `PUT /users/{id}/profiles/{id}` | `updateUserProfile()` | ✅ Aligned |
| `DELETE /users/{id}/profiles/{id}` | `deleteUserProfile()` | ✅ Aligned |
| `PUT /users/{id}/profiles/{id}/set-default` | `setDefaultUserProfile()` | ✅ Aligned |
| `PATCH /users/{id}/verify-email` | `verifyUserEmail()` | ✅ Aligned |
| `PATCH /users/{id}/verify-phone` | `verifyUserPhone()` | ✅ Aligned |
| `PATCH /users/{id}/complete-account` | `completeUserAccount()` | ✅ Aligned |
| `PUT /users/{id}/force-password-change` | `forcePasswordChange()` | ✅ Aligned |
| `GET /users/{id}/sessions` | `fetchUserSessions()` | ✅ Aligned |
| `DELETE /users/{id}/sessions/{id}` | `revokeUserSession()` | ✅ Aligned |
| `GET /users/{id}/mfa` | `fetchUserMFA()` | ✅ Aligned |
| `GET /roles` | `fetchRoles()` | ✅ Aligned |
| `GET /roles/{id}` | `fetchRoleById()` | ✅ Aligned |
| `POST /roles` | `createRole()` | ✅ Aligned |
| `PUT /roles/{id}` | `updateRole()` | ✅ Aligned |
| `DELETE /roles/{id}` | `deleteRole()` | ✅ Aligned |
| `PUT /roles/{id}/status` | `updateRoleStatus()` | ✅ Aligned |
| `GET /roles/{id}/permissions` | `fetchRolePermissions()` | ✅ Aligned |
| `POST /roles/{id}/permissions` | `addRolePermissions()` | ✅ Aligned |
| `DELETE /roles/{id}/permissions/{id}` | `removeRolePermission()` | ✅ Aligned |
| `GET /permissions` | `fetchPermissions()` | ✅ Aligned |
| `GET /permissions/{id}` | `fetchPermissionById()` | ✅ Aligned |
| `POST /permissions` | `createPermission()` | ✅ Aligned |
| `PUT /permissions/{id}` | `updatePermission()` | ✅ Aligned |
| `DELETE /permissions/{id}` | `deletePermission()` | ✅ Aligned |
| `PUT /permissions/{id}/status` | `updatePermissionStatus()` | ✅ Aligned |
| `GET /services` | `fetchServices()` | ✅ Aligned |
| `POST /services` | `createService()` | ✅ Aligned |
| `PUT /services/{id}` | `updateService()` | ✅ Aligned |
| `DELETE /services/{id}` | `deleteService()` | ✅ Aligned |
| `PUT /services/{id}/status` | `updateServiceStatus()` | ✅ Aligned |
| `GET /apis` | `fetchApis()` | ✅ Aligned |
| `POST /apis` | `createApi()` | ✅ Aligned |
| `PUT /apis/{id}` | `updateApi()` | ✅ Aligned |
| `DELETE /apis/{id}` | `deleteApi()` | ✅ Aligned |
| `GET /policies` | `fetchPolicies()` | ✅ Aligned |
| `POST /policies` | `createPolicy()` | ✅ Aligned |
| `PUT /policies/{id}` | `updatePolicy()` | ✅ Aligned |
| `DELETE /policies/{id}` | `deletePolicy()` | ✅ Aligned |
| `GET /identity_providers` | `fetchIdentityProviders()` | ✅ Aligned |
| `POST /identity_providers` | `createIdentityProvider()` | ✅ Aligned |
| `PUT /identity_providers/{id}` | `updateIdentityProvider()` | ✅ Aligned |
| `DELETE /identity_providers/{id}` | `deleteIdentityProvider()` | ✅ Aligned |
| `PUT /identity_providers/{id}/status` | `updateIdentityProviderStatus()` | ✅ Aligned |
| `POST /identity_providers/test` | `testIdentityProvider()` | ✅ Aligned |
| `GET /clients` | `fetchClients()` | ✅ Aligned |
| `GET /clients/{id}` | `fetchClientById()` | ✅ Aligned |
| `POST /clients` | `createClient()` (incl. `branding_id`, `allow_registration`) | ✅ Aligned |
| `PUT /clients/{id}` | `updateClient()` (incl. `branding_id`, `allow_registration`) | ✅ Aligned |
| `DELETE /clients/{id}` | `deleteClient()` | ✅ Aligned |
| `PUT /clients/{id}/status` | `updateClientStatus()` | ✅ Aligned |
| `POST /clients/{id}/rotate-secret` | `rotateClientSecret()` | ✅ Aligned |
| `GET /clients/{id}/uris` | `fetchClientUris()` | ✅ Aligned |
| `POST /clients/{id}/uris` | `addClientUri()` | ✅ Aligned |
| `PUT /clients/{id}/uris/{id}` | `updateClientUri()` | ✅ Aligned |
| `DELETE /clients/{id}/uris/{id}` | `deleteClientUri()` | ✅ Aligned |
| `GET /clients/{id}/apis` | `fetchClientApis()` | ✅ Aligned |
| `POST /clients/{id}/apis` | `addClientApis()` | ✅ Aligned |
| `DELETE /clients/{id}/apis/{id}` | `removeClientApi()` | ✅ Aligned |
| `POST /clients/{id}/apis/{id}/permissions` | `addClientApiPermissions()` | ✅ Aligned |
| `DELETE /clients/{id}/apis/{id}/permissions/{id}` | `removeClientApiPermission()` | ✅ Aligned |
| `POST /clients/{id}/identity_providers` | `addClientIdentityProvider()` | ✅ Aligned |
| `PUT /clients/{id}/identity_providers/{id}` | `updateClientIdentityProvider()` | ✅ Aligned |
| `DELETE /clients/{id}/identity_providers/{id}` | `removeClientIdentityProvider()` | ✅ Aligned |
| `GET /api_keys` | `fetchApiKeys()` | ❌ REMOVE — endpoint deleted from backend (Part 7) |
| `POST /api_keys` | `createApiKey()` | ❌ REMOVE — endpoint deleted from backend (Part 7) |
| `PUT /api_keys/{id}` | `updateApiKey()` | ❌ REMOVE — endpoint deleted from backend (Part 7) |
| `DELETE /api_keys/{id}` | `deleteApiKey()` | ❌ REMOVE — endpoint deleted from backend (Part 7) |
| `PUT /api_keys/{id}/status` | `updateApiKeyStatus()` | ❌ REMOVE — endpoint deleted from backend (Part 7) |
| `GET /api_keys/{id}/apis` | `fetchApiKeyApis()` | ❌ REMOVE — endpoint deleted from backend (Part 7) |
| `POST /api_keys/{id}/apis` | `addApiKeyApis()` | ❌ REMOVE — endpoint deleted from backend (Part 7) |
| `GET /webhook-endpoints` | `fetchWebhooks()` | ✅ Aligned |
| `POST /webhook-endpoints` | `createWebhook()` | ✅ Aligned |
| `PUT /webhook-endpoints/{id}` | `updateWebhook()` | ✅ Aligned |
| `DELETE /webhook-endpoints/{id}` | `deleteWebhook()` | ✅ Aligned |
| `PATCH /webhook-endpoints/{id}/status` | `updateWebhookStatus()` | ✅ Aligned |
| `GET /webhook-endpoints/{id}/subscriptions` | `fetchWebhookSubscriptions()` | ✅ Aligned |
| `POST /webhook-endpoints/{id}/subscriptions` | `addWebhookSubscription()` | ✅ Aligned |
| `DELETE /webhook-endpoints/{id}/subscriptions` | `removeWebhookSubscription()` | ✅ Aligned |
| `GET /webhook-endpoints/{id}/deliveries` | `fetchDeliveryHistory()` | ✅ Aligned |
| `POST /webhook-replay` | `replayDelivery()` | ✅ Aligned |
| `GET /event-types` | `fetchEventTypes()` | ✅ Aligned |
| `GET /tenant-event-types` | `fetchTenantEventTypes()` | ✅ Aligned |
| `PUT /tenant-event-types` | `setTenantEventType()` | ✅ Aligned |
| `GET /event-routes` | `fetchEventRoutes()` | ✅ Aligned |
| `POST /event-routes` | `createEventRoute()` | ✅ Aligned |
| `PUT /event-routes/{id}` | `updateEventRoute()` | ✅ Aligned |
| `DELETE /event-routes/{id}` | `deleteEventRoute()` | ✅ Aligned |
| `GET /event-routes/{id}` | `fetchEventRouteById()` | ⚠️ Verify (C-CON-06) |
| `GET /registration_flows` | `fetchRegistrationFlows()` | ✅ Aligned |
| `POST /registration_flows` | `createRegistrationFlow()` | ✅ Aligned |
| `PUT /registration_flows/{id}` | `updateRegistrationFlow()` | ✅ Aligned |
| `DELETE /registration_flows/{id}` | `deleteRegistrationFlow()` | ✅ Aligned |
| `PATCH /registration_flows/{id}/status` | `updateRegistrationFlowStatus()` | ✅ Aligned |
| `GET /registration_flows/{id}/roles` | `fetchRegistrationFlowRoles()` | ✅ Aligned |
| `POST /registration_flows/{id}/roles` | `addRegistrationFlowRoles()` | ✅ Aligned |
| `DELETE /registration_flows/{id}/roles/{id}` | `removeRegistrationFlowRole()` | ✅ Aligned |
| `GET /invite` | `fetchInvites()` | ✅ Aligned |
| `GET /invite/{id}` | `fetchInviteById()` | ❌ Missing (C-CON-05) |
| `POST /invite` | `sendInvite()` — missing `callback_url` | ❌ Missing field (C-CON-01) |
| `POST /invite/{id}/resend` | `resendInvite()` | ✅ Aligned |
| `DELETE /invite/{id}` | `deleteInvite()` | ✅ Aligned |
| `GET /branding` | `fetchBrandings()` | ✅ Aligned |
| `POST /branding` | `createBranding()` — missing `logo_data`/`logo_content_type` | ❌ Missing fields (C-CON-03) |
| `PUT /branding/{id}` | `updateBranding()` — missing `logo_data`/`logo_content_type` | ❌ Missing fields (C-CON-03) |
| `PATCH /branding/{id}/activate` | `activateBranding()` | ✅ Aligned |
| `DELETE /branding/{id}` | `deleteBranding()` | ✅ Aligned |
| `GET /email_templates` | `fetchEmailTemplates()` | ✅ Aligned |
| `PUT /email_templates/{id}` | `updateEmailTemplate()` | ✅ Aligned |
| `PATCH /email_templates/{id}/status` | `updateEmailTemplateStatus()` | ✅ Aligned |
| `GET /sms_templates` | `fetchSmsTemplates()` | ✅ Aligned |
| `PUT /sms_templates/{id}` | `updateSmsTemplate()` | ✅ Aligned |
| `PATCH /sms_templates/{id}/status` | `updateSmsTemplateStatus()` | ✅ Aligned |
| `GET /security-settings/mfa` | `fetchMFAConfig()` | ✅ Aligned |
| `PUT /security-settings/mfa` | `updateMFAConfig()` | ✅ Aligned |
| `GET /security-settings/password` | `fetchPasswordPolicies()` | ✅ Aligned |
| `PUT /security-settings/password` | `updatePasswordPolicies()` | ✅ Aligned |
| `GET /security-settings/session` | `fetchSessionSettings()` | ✅ Aligned |
| `PUT /security-settings/session` | `updateSessionSettings()` | ✅ Aligned |
| `GET /security-settings/token` | `fetchTokenConfig()` | ✅ Aligned |
| `PUT /security-settings/token` | `updateTokenConfig()` | ✅ Aligned |
| `GET /security-settings/lockout` | `fetchLockoutConfig()` | ✅ Aligned |
| `PUT /security-settings/lockout` | `updateLockoutConfig()` | ✅ Aligned |
| `GET /security-settings/threat` | `fetchThreatDetection()` | ✅ Aligned |
| `PUT /security-settings/threat` | `updateThreatDetection()` | ✅ Aligned |
| `GET /security-settings/registration` | `fetchRegistrationConfig()` | ✅ Aligned |
| `PUT /security-settings/registration` | `updateRegistrationConfig()` | ✅ Aligned |
| `GET /tenant-settings/audit` | `fetchAuditConfig()` | ✅ Aligned |
| `PUT /tenant-settings/audit` | `updateAuditConfig()` | ✅ Aligned |
| `GET /tenant-settings/maintenance` | `fetchMaintenanceConfig()` | ✅ Aligned |
| `PUT /tenant-settings/maintenance` | `updateMaintenanceConfig()` | ✅ Aligned |
| `GET /tenant-settings/rate-limit` | `fetchRateLimitConfig()` | ✅ Aligned |
| `PUT /tenant-settings/rate-limit` | `updateRateLimitConfig()` | ✅ Aligned |
| `GET /email-config` | `fetchEmailConfig()` | ✅ Aligned |
| `PUT /email-config` | `updateEmailConfig()` | ✅ Aligned |
| `GET /sms-config` | `fetchSmsConfig()` | ✅ Aligned |
| `PUT /sms-config` | `updateSmsConfig()` | ✅ Aligned |
| `GET /ip-restriction-rules` | `fetchIPRestrictionRules()` | ✅ Aligned |
| `POST /ip-restriction-rules` | `createIPRestrictionRule()` | ✅ Aligned |
| `PUT /ip-restriction-rules/{id}` | `updateIPRestrictionRule()` | ✅ Aligned |
| `DELETE /ip-restriction-rules/{id}` | `deleteIPRestrictionRule()` | ✅ Aligned |
| `GET /auth-events` | `fetchAuthEvents()` | ✅ Aligned |
| `GET /auth-events/{id}` | `fetchAuthEventById()` | ✅ Aligned |
| `GET /auth-events/export` | `exportAuthEvents()` | ❌ Missing (C-CON-02) |
| `GET /auth-events/count` | `fetchAuthEventCount()` | ❌ Missing (C-CON-04) |
| `GET /dashboard/summary` | `fetchDashboardSummary()` | ✅ Aligned |

### Public API (:8081) — identity calls

| Endpoint | Identity service function | Status |
|---|---|---|
| `GET /.well-known/openid-configuration` | Not called directly (browser) | ✅ N/A |
| `GET /.well-known/jwks.json` | Not called directly (browser) | ✅ N/A |
| `POST /login` | `authLogin()` | ✅ Aligned |
| `POST /login/mfa/verify` | `verifyMFALogin()` | ✅ Aligned |
| `POST /login/mfa/send-sms` | `sendMFALoginSMS()` | ✅ Aligned |
| `POST /login/mfa/send-email-otp` | `sendMFALoginEmailOtp()` | ✅ Aligned |
| `POST /login/mfa/webauthn/begin` | `beginMFALoginWebAuthn()` | ✅ Aligned |
| `POST /sms-login/send` | `sendSMSLoginCode()` | ✅ Aligned |
| `POST /sms-login/verify` | `verifySMSLogin()` | ✅ Aligned |
| `POST /register` | `authRegister()` (incl. `registration_flow` param) | ✅ Aligned |
| `POST /register/invite` | `authRegisterInvite()` | ✅ Aligned |
| `POST /logout` | `authLogout()` | ✅ Aligned |
| `POST /refresh-token` | axios interceptor | ✅ Aligned |
| `POST /forgot-password` | `authForgotPassword()` | ✅ Aligned |
| `POST /reset-password` | `authResetPassword()` | ✅ Aligned |
| `GET /invite` | `fetchInviteContext()` (by invite_token param) | ✅ Aligned |
| `POST /magic-link/send` | `sendMagicLink()` | ✅ Aligned |
| `POST /magic-link/verify` | `verifyMagicLink()` | ✅ Aligned |
| `POST /email-verification/verify` | inline in `VerifyEmailPage` | ✅ Aligned |
| `POST /email-verification/send` | inline in `VerifyEmailPage` | ✅ Aligned |
| `POST /recovery/backup-code` | `recoverWithBackupCode()` | ✅ Aligned |
| `GET /profile` | `fetchProfile()` | ✅ Aligned |
| `POST /profile` | `createProfile()` | ✅ Aligned |
| `GET /account` | `fetchAccount()` | ✅ Aligned |
| `GET /account/identities` | `fetchAccountIdentities()` | ✅ Aligned |
| `DELETE /account/identities/{id}` | `unlinkAccountIdentity()` | ✅ Aligned |
| `POST /account/identities/link` | `linkAccountIdentity()` | ✅ Aligned |
| `POST /account/phone/send-verification` | `sendPhoneVerification()` | ✅ Aligned |
| `POST /account/phone/verify` | `verifyPhone()` | ✅ Aligned |
| `GET /mfa/status` | `fetchMFAStatus()` | ✅ Aligned |
| `POST /mfa/totp/enroll` | `enrollTOTP()` | ✅ Aligned |
| `POST /mfa/totp/verify` | `verifyTOTPEnroll()` | ✅ Aligned |
| `DELETE /mfa/totp` | `disableTOTP()` | ✅ Aligned |
| `POST /mfa/backup-codes/regenerate` | `regenerateBackupCodes()` | ✅ Aligned |
| `POST /mfa/webauthn/register/begin` | `beginWebAuthnRegistration()` | ✅ Aligned |
| `POST /mfa/webauthn/register/finish` | `finishWebAuthnRegistration()` | ✅ Aligned |
| `POST /mfa/webauthn/auth/begin` | `beginWebAuthnAuth()` | ✅ Aligned |
| `DELETE /mfa/webauthn/{id}` | `deleteWebAuthnCredential()` | ✅ Aligned |
| `POST /mfa/sms/enroll` | `enrollSMSMFA()` | ✅ Aligned |
| `POST /mfa/sms/verify` | `verifySMSMFAEnroll()` | ✅ Aligned |
| `DELETE /mfa/sms` | `disableSMSMFA()` | ✅ Aligned |
| `POST /mfa/email-otp/enroll` | `enrollEmailOTPMFA()` | ✅ Aligned |
| `POST /mfa/email-otp/verify` | `verifyEmailOTPMFAEnroll()` | ✅ Aligned |
| `DELETE /mfa/email-otp` | `disableEmailOTPMFA()` | ✅ Aligned |
| `POST /mfa/step-up/challenge` | `requestStepUpChallenge()` | ✅ Aligned |
| `POST /mfa/step-up/send-sms` | `sendStepUpSMS()` | ✅ Aligned |
| `POST /mfa/step-up/send-email-otp` | `sendStepUpEmailOTP()` | ✅ Aligned |
| `POST /mfa/step-up/verify` | `verifyStepUp()` | ✅ Aligned |
| `GET /oauth/authorize` | via `window.location` redirect | ✅ Aligned |
| `POST /oauth/authorize/continue` | `continueOAuth(requestId)` | ✅ Aligned |
| `GET /oauth/connections` | `fetchOAuthConnections()` (branding field consumed) | ✅ Aligned |
| `GET /oauth/consent/{id}` | `fetchConsentChallenge()` | ✅ Aligned |
| `POST /oauth/consent` | `submitConsent()` | ✅ Aligned |
| `GET /oauth/consent/grants` | `fetchConsentGrants()` | ✅ Aligned |
| `DELETE /oauth/consent/grants/{id}` | `revokeConsentGrant()` | ✅ Aligned |
| `POST /oauth/device` | `submitDeviceUserCode()` | ✅ Aligned |
| `POST /oauth/device/deny` | `denyDeviceUserCode()` | ✅ Aligned |
| `POST /oauth/ciba/approve` | `approveCIBARequest()` | ✅ Aligned |
| `POST /oauth/ciba/deny` | `denyCIBARequest()` | ✅ Aligned |
| `GET /oauth/end_session` | via `window.location.assign` | ✅ Aligned |
| `GET /tenant` | Redux `fetchDefaultTenantAsync` | ✅ Aligned |
| `GET /tenant/{id}` | Redux `fetchTenantByIdentifierAsync` | ✅ Aligned |
| `GET /client` | `fetchPublicClient()` | ✅ Aligned |
| `GET /public/branding` | `fetchPublicBranding()` (if used) | ✅ Aligned |
| `GET /public/branding/{id}/logo` | via `<img src={resolveBrandingLogoUrl(...)}>` | ⚠️ URL must be resolved with PUBLIC_BASE_URL prefix (C-CON-03 applies to identity too if it renders logo from branding.logo_url) |

### Public API (:8081) — console OAuth calls

| Endpoint | Console service function | Status |
|---|---|---|
| `GET /client/console` | `fetchConsoleClient()` (by tenant_id) | ✅ Aligned |
| `POST /oauth/token` | `exchangeAuthorizationCode()` | ✅ Aligned |
| `POST /refresh-token` | axios interceptor | ✅ Aligned |

---

## Part 5 — Identity: logo URL resolution

This applies to both apps. The backend branding handler returns `logo_url = "/public/branding/{uuid}/logo"` (a path-only relative URL) when a logo is uploaded to DB storage. Both frontends must prefix this with `PUBLIC_BASE_URL` (stripped of `/api/v1`) before using it in an `<img>` tag. External `https://` URLs should be used as-is.

**Applies to:**
- Console: anywhere `branding.logo_url` is rendered (C-CON-03 covers this).
- Identity: `AppBootstrap`'s `applyBranding()` call passes `branding.logo_url` to the CSS variable `--branding-logo-url`. Verify this variable is prefixed correctly when the URL starts with `/public/branding/`.

**Check in identity:**

- [ ] **`src/utils/branding.ts`** — Confirm `applyBranding` resolves relative logo paths before setting the CSS variable. If it does not:
  ```ts
  // In applyBranding(), when setting the logo CSS variable:
  const logoUrl = branding.logo_url?.startsWith('/public/branding/')
    ? `${PUBLIC_BASE_URL.replace('/api/v1', '')}${branding.logo_url}`
    : branding.logo_url
  document.documentElement.style.setProperty('--branding-logo-url', logoUrl ? `url('${logoUrl}')` : '')
  ```

---

## Part 6 — Final verification checklist

Run this after implementing all items above.

### Build gates (run in each repo)

- [ ] `npm run lint` exits 0 in `maintainerd-auth-console`
- [ ] `tsc -b` exits 0 in `maintainerd-auth-console`
- [ ] `npm run build` exits 0 in `maintainerd-auth-console`, no `.map` files in `dist/`
- [ ] `npm run lint` exits 0 in `maintainerd-auth-identity`
- [ ] `tsc -b` exits 0 in `maintainerd-auth-identity`
- [ ] `npm run build` exits 0 in `maintainerd-auth-identity`, no `.map` files in `dist/`

### Functional smoke (against running nginx stack)

- [ ] Console: create an invite with a `callback_url` → backend rejects invalid URLs → valid URL is stored → invited user completes registration and lands on the callback URL
- [ ] Console: export auth events as CSV → file downloads with correct headers
- [ ] Console: export auth events as JSON → file downloads as valid JSON
- [ ] Console: upload a PNG logo on a branding template → logo is stored → logo appears correctly in console UI (prefixed URL)
- [ ] Identity: logo from client branding appears correctly after login (resolves via `PUBLIC_BASE_URL` prefix)
- [ ] Identity: `GET /oauth/authorize?...&screen_hint=signup&registration_flow=<id>` → redirected to `/register?request_id=...` → registration completes → `POST /oauth/authorize/continue` → authorization code issued → tokens exchanged
- [ ] Identity: normal login unaffected by the `screen_hint=signup` path
- [ ] Console: verifying keyset pagination response shape for `/auth-events` and `/users` (C-CON-07)

---

## Part 7 — API Keys: Full removal from console

The backend removed the entire API keys feature: migrations 019 (`api_keys`), 020 (`api_key_apis`), and 021 (`api_key_permissions`) were replaced with no-ops and all Go code (models, repos, services, handlers, middleware) was deleted. The console still has a full api-keys feature. Every trace must be removed. The M2M authentication pattern going forward is **OAuth client credentials** (`POST /oauth/token` with `grant_type=client_credentials`).

### Files to delete entirely

- [ ] `src/pages/api-keys/ApiKeysPage.tsx`
- [ ] `src/pages/api-keys/details/` — entire directory
- [ ] `src/pages/api-keys/form/` — entire directory
- [ ] `src/pages/api-keys/index.ts`
- [ ] `src/pages/api-keys/constants.ts`
- [ ] `src/services/api/api-keys/index.ts`
- [ ] `src/services/api/api-keys/types.ts`
- [ ] `src/hooks/useApiKeys.ts`
- [ ] `src/lib/validations/apiKeySchema.ts`

### Files to edit (remove api-key references)

- [ ] **`src/App.tsx`**
  - Remove lazy imports (around lines 52–54):
    ```ts
    const ApiKeysPage = lazy(() => import('./pages/api-keys'))
    const ApiKeyDetailsPage = lazy(() => import('./pages/api-keys/details'))
    const ApiKeyAddOrUpdateForm = lazy(() => import('./pages/api-keys/form'))
    ```
  - Remove routes (around lines 183–186):
    ```tsx
    <Route path="api-keys" element={<ApiKeysPage />} />
    <Route path="api-keys/create" element={<ApiKeyAddOrUpdateForm />} />
    <Route path="api-keys/:id" element={<ApiKeyDetailsPage />} />
    <Route path="api-keys/:id/edit" element={<ApiKeyAddOrUpdateForm />} />
    ```

- [ ] **`src/components/sidebar/constants.tsx`**
  - Remove from Applications section items array:
    ```ts
    { title: "API Keys", route: "/api-keys" },
    ```

- [ ] **`src/services/api/config.ts`**
  - Remove any `API_KEYS` / `api_keys` / `api-keys` endpoint constant(s).

- [ ] **`src/services/index.ts`**
  - Remove any api-key service re-exports.

- [ ] **`src/lib/validations/index.ts`**
  - Remove `apiKeySchema` re-export.

- [ ] **`src/lib/validations/rateLimitConfigSchema.ts`**
  - Inspect: if it references `api_key_rate_limit` or similar, remove those fields.

- [ ] **`src/pages/dashboard/DashboardPage.tsx`**
  - Inspect: if it renders an "API Keys" stat card, remove it.

- [ ] **`src/components/data-table/useServerDataTable.ts` and `useServerDataTable.test.tsx`**
  - Inspect: remove any api-key-specific logic or test cases.

### Migration guidance note

If any page or tooltip in the console refers admins to "API Keys" for machine-to-machine access, replace the copy with guidance to create a **system client** in the Clients page and use the client credentials flow.

**Acceptance:** No `/api-keys` route in the app. No api-keys sidebar entry. `tsc -b` and `npm run build` exit 0.

---

## Part 8 — TypeScript type updates from schema changes

Each item below corresponds to a confirmed backend DTO/model change from `docs/planning/database-restructuring.md`. Find the relevant TypeScript interfaces in `src/services/api/*/types.ts` and apply the changes.

### 8.1 — `User` type: new fields

The backend `UserResponseDTO` now includes these fields (from DB restructuring section 3.1). Add to the `User` interface:

```ts
last_login_at?: string | null        // ISO 8601 — last login timestamp
login_count?: number                 // integer, default 0
email_verified_at?: string | null    // ISO 8601 — when email was verified
phone_verified_at?: string | null    // ISO 8601 — when phone was verified
external_id?: string | null          // SCIM external identifier
created_by?: string | null           // UUID of admin who provisioned the user
updated_by?: string | null           // UUID of admin who last updated the user
```

Display on the user detail page:
- [ ] Show `last_login_at` as "Last login" (formatted date, "Never" if null).
- [ ] Show `login_count` as "Total logins" (integer).
- [ ] Show `email_verified_at` next to the email field (green check / "Unverified").
- [ ] Show `phone_verified_at` next to the phone field.
- [ ] Show `external_id` as a read-only "External ID" field (grayed if null).

### 8.2 — `Client` type: new fields

Add to `CreateClientRequest`, `UpdateClientRequest`, and `Client` response interface (from DB restructuring section 1.6):

```ts
backchannel_logout_uri?: string | null
frontchannel_logout_uri?: string | null
backchannel_logout_session_required?: boolean
dpop_required?: boolean
```

- [ ] Add these four fields to the client create/edit form under an "Advanced / Logout" collapsible section:
  - `backchannel_logout_uri` — URL text input, label "Back-channel logout URI"
  - `frontchannel_logout_uri` — URL text input, label "Front-channel logout URI"
  - `backchannel_logout_session_required` — checkbox, label "Require session parameter on back-channel logout"
  - `dpop_required` — checkbox, label "Require DPoP proof-of-possession"

### 8.3 — Client `grant_types`: remove deprecated values

The backend now enforces a CHECK constraint allowing only: `authorization_code`, `client_credentials`, `refresh_token`, `device_code`, `ciba`, `token-exchange`. The values `implicit` and `password` are no longer valid.

- [ ] Remove `implicit` and `password` from any `grant_types` multi-select dropdown in the client form.
- [ ] Remove them from any Yup/Zod validation `oneOf`/`enum` for grant types.

### 8.4 — `ClientUri` type: rename type values (hyphens → underscores)

The backend changed the `client_uris.type` CHECK constraint from hyphenated to underscore values (DB restructuring section 3.8):

```
'redirect-uri'    →  'redirect_uri'
'origin-uri'      →  'origin_uri'
'logout-uri'      →  'logout_uri'
'login-uri'       →  'login_uri'
'cors-origin-uri' →  'cors_origin_uri'
```

- [ ] Run: `grep -r "redirect-uri\|origin-uri\|logout-uri\|login-uri\|cors-origin-uri" src/` — update every occurrence.
- [ ] Update any `ClientUriType` enum or TypeScript union type.
- [ ] Update any validation schemas that use the old hyphenated values.

### 8.5 — `UserIdentity` type: nullable `client_id` + provisioning fields

From DB restructuring section 1.6 (`user_identities` changes):

```ts
// CHANGE:
client_id: string  →  client_id: string | null

// ADD:
jit_provisioned_at?: string | null
provisioning_source?: 'jit' | 'scim' | 'manual' | 'invite' | 'import' | null
```

- [ ] Update the `UserIdentity` interface.
- [ ] On the user identities list, show a "SCIM" or "JIT" badge when `provisioning_source` is `'scim'` or `'jit'`.

### 8.6 — `IdentityProvider` type: SAML support + certificate expiry

From DB restructuring section 1.6:

```ts
// ADD 'saml' to provider_type union/enum:
type IdentityProviderType = 'oidc' | 'saml' | 'google' | 'github' | ...

// ADD to IdentityProvider response:
certificate_expires_at?: string | null
```

- [ ] Add `'saml'` to the `provider_type` dropdown options in the IdP create/edit form.
- [ ] Show `certificate_expires_at` on the IdP detail page. Add a warning badge if the certificate expires within 30 days.

### 8.7 — `MFAWebAuthnKey` type: transport array + discoverable credential + backup rename

From DB restructuring sections 2.2 and 4.2:

```ts
// CHANGE (transport is now an array):
transport: string   →  transport: string[]

// ADD:
is_discoverable_credential?: boolean

// RENAME (if the field existed with the old name):
is_backup_state?: boolean  →  is_backup_active?: boolean
```

- [ ] When `is_discoverable_credential === true`, label the credential as "Passkey". When false, label it "Security Key".
- [ ] Render `transport` as a comma-joined list (e.g., `"usb, nfc"`).

### 8.8 — `Profile` type: remove product-layer fields

From DB restructuring sections 3.2, 3.9, and 3.27B/C:

```ts
// REMOVE from Profile interface and profile form (if present):
phone           // canonical field is users.phone; no dual ownership
bio
social_links
is_default      // replaced by DB-level UNIQUE constraint
address         // moved to profiles.metadata['address']
city            // moved to profiles.metadata['address']['locality']
country         // moved to profiles.metadata['address']['country']
suffix          // moved to profiles.metadata['name_suffix']
```

- [ ] Remove listed fields from the `Profile` TypeScript interface.
- [ ] Remove corresponding inputs from the user profile edit form. If `phone` was shown in the profile form, leave it as a read-only display referencing `user.phone` — do not remove phone display from the user form.

### 8.9 — `UserSetting` type: remove product-preference fields

From DB restructuring sections 3.10, 3.11, and 3.27A:

```ts
// REMOVE from UserSetting interface and settings form (if present):
marketing_email_consent
sms_notifications_consent
push_notifications_consent
profile_visibility
preferred_contact_method
data_processing_consent      // replaced by user_consents table (see 9.3)
terms_accepted_at            // replaced by user_consents table
privacy_policy_accepted_at   // replaced by user_consents table
emergency_contact_name
emergency_contact_phone
emergency_contact_email
emergency_contact_relation
```

- [ ] Remove listed fields from the `UserSetting` TypeScript interface and from any user settings form/page.

### 8.10 — `RegistrationFlow` type: `required_fields` is now `string[]`

From DB restructuring section 2.3 (JSONB change):

```ts
// CHANGE:
required_fields?: string      →  required_fields?: string[]
```

- [ ] Update the `RegistrationFlow` TypeScript interface.
- [ ] Remove any `JSON.parse(flow.required_fields)` calls — the value now arrives as a real array from the API.
- [ ] If the registration flow form uses a tag/chip input for required fields, it should already work with `string[]`; verify.

### 8.11 — `TenantMember` role: add `admin` tier

From DB restructuring section 4.4:

```ts
// CHANGE:
type TenantMemberRole = 'owner' | 'member'
→
type TenantMemberRole = 'owner' | 'admin' | 'member'
```

- [ ] Update the `TenantMemberRole` type.
- [ ] Add `admin` as an option in the add/edit tenant member role dropdown (between `owner` and `member`).

---

## Part 9 — New backend features needing console pages

These backend entities have API endpoints but no console UI yet. Each item is a new page or detail-tab to build.

### 9.1 — Management Audit Log (new page)

**Backend:** `GET /management-audit-log` (internal :8080, paginated, filterable)  
**Response fields:** `uuid`, `action`, `resource_type`, `resource_id`, `changes` (JSONB diff — `{"after":…}` for creates, `{"update":…,"after":…}` for updates, `{"before":…}` for deletes), `ip_address`, `actor_user_id`, `actor_client_id`, `outcome`, `created_at`

- [ ] Add `MANAGEMENT_AUDIT_LOG: '/management-audit-log'` to `src/services/api/config.ts`
- [ ] Create `src/services/api/audit-log/types.ts` and `index.ts` with `fetchAuditLog(params)` function
- [ ] Create `src/pages/audit-log/AuditLogPage.tsx`:
  - Table columns: timestamp, actor (user or client), action, resource type, resource ID, outcome badge (success/failure/partial)
  - Filters: date range, resource_type dropdown, actor search
  - Click a row → expandable drawer showing the `changes` JSONB diff
- [ ] Register route in `src/App.tsx`: `<Route path="audit-log" element={<AuditLogPage />} />`
- [ ] Add to sidebar under Monitoring (see Part 10 for exact placement)

### 9.2 — Client Roles (new tab on Client detail)

**Backend:** `GET /clients/{uuid}/roles`, `POST /clients/{uuid}/roles` `{ role_uuid: string }`, `DELETE /clients/{uuid}/roles/{role_uuid}` (internal :8080)

- [ ] Add service functions to `src/services/api/clients/index.ts`:
  ```ts
  fetchClientRoles(clientId: string): Promise<Role[]>
  addClientRole(clientId: string, roleUuid: string): Promise<void>
  removeClientRole(clientId: string, roleUuid: string): Promise<void>
  ```
- [ ] Add a "Roles" tab to the Client detail page — same role assignment list + add/remove pattern as user role assignment.

### 9.3 — User Consents (new tab on User detail)

**Backend:** `GET /users/{uuid}/consents` (internal :8080)  
**Response:** paginated list of `{ uuid, consent_type, policy_version, accepted, ip_address, user_agent, created_at }`

- [ ] Add `fetchUserConsents(userId: string)` to `src/services/api/users/index.ts`
- [ ] Add `UserConsent` interface to user types
- [ ] Add a "Consents" tab to the User detail page — table showing consent type (`terms_of_service` / `privacy_policy` / `data_processing`), version accepted, date, IP address

### 9.4 — User Trusted Devices (new tab on User detail)

**Backend:** `GET /users/{uuid}/devices` (internal :8080)  
**Response:** list of `{ uuid, trusted_until, created_at }` (device fingerprint details TBD)

- [ ] Add `fetchUserTrustedDevices(userId: string)` to `src/services/api/users/index.ts`
- [ ] Add a "Trusted Devices" tab to the User detail page — list view with device info and expiry. No admin revoke endpoint exists yet; show read-only.

### 9.5 — Data Erasure Requests (action on User detail)

**Backend:** `POST /users/{uuid}/erasure-requests` (internal :8080, requires `user:delete` permission)  
**Behavior:** Schedules GDPR anonymization of all PII 30 days from now.

- [ ] Add service function: `createErasureRequest(userId: string): Promise<void>`
- [ ] Add an "Erase User Data" danger button to the User detail page, behind a confirmation dialog.
  - Dialog text: "This schedules anonymization of all personal data for this user in 30 days. The account cannot be restored after erasure begins."
  - On confirm: call `createErasureRequest`, show success toast.

### 9.6 — Policy Version History (new tab on Policy detail)

**Backend:** `GET /policies/{uuid}/history` (list, paginated), `GET /policies/{uuid}/history/{version_number}` (single snapshot)

- [ ] Add service functions to `src/services/api/policies/index.ts`:
  ```ts
  fetchPolicyHistory(policyId: string): Promise<PolicyVersionHistory[]>
  fetchPolicyHistoryVersion(policyId: string, version: number): Promise<PolicyVersionHistory>
  ```
- [ ] Add `PolicyVersionHistory` type to policy types: `{ version_number, document, snapshot_at, changed_by_user_id }`
- [ ] Add a "History" tab to the Policy detail page — list of versions with snapshot date; clicking a version shows the policy document at that point in time.

### 9.7 — Workload Identity Federations (new page)

**Backend:** `GET/POST /workload-identity-federations`, `GET/PUT/DELETE /workload-identity-federations/{uuid}` (internal :8080)  
**Key fields:** `name`, `description`, `issuer_url`, `audience`, `subject_claim`, `subject_pattern`, `allowed_scopes`, `attribute_mapping`, `is_active`

- [ ] Create `src/services/api/workload-identity/index.ts` and `types.ts`
- [ ] Add endpoint constant `WORKLOAD_IDENTITY_FEDERATIONS: '/workload-identity-federations'` to `config.ts`
- [ ] Create `src/pages/workload-identity/WorkloadIdentityPage.tsx` — list with create/edit form:
  - `name` (required), `description`, `issuer_url` (required — validated OIDC discovery URL), `audience`, `subject_claim` (default `sub`), `subject_pattern` (glob), `allowed_scopes` (multi-tag input), `attribute_mapping` (JSON editor), `is_active` toggle
- [ ] Register route in `src/App.tsx`: `<Route path="workload-identity" element={<WorkloadIdentityPage />} />`
- [ ] Add to sidebar under Applications (see Part 10)

### 9.8 — SCIM Configurations (new page)

**Backend:** CRUD management endpoints — verify exact paths in `internal/scim/routes.go`; expected: `GET/POST /scim/configurations` and `GET/PUT/DELETE /scim/configurations/{uuid}` on internal :8080  
**Key fields:** `display_name`, `identity_provider_id` (optional link), `base_url`, `sync_users`, `sync_groups`, `sync_direction` (`inbound`/`outbound`/`bidirectional`), `attribute_mapping`, `is_active`, `last_sync_at`, `last_sync_status`

- [ ] Verify exact routes by reading `internal/scim/routes.go`
- [ ] Create `src/services/api/scim/index.ts` and `types.ts`
- [ ] Create `src/pages/scim/SCIMConfigPage.tsx` — list + create/edit form
- [ ] Bearer token handling: on create, show the generated token **one time** in a modal (never display it again). Offer a "Rotate bearer token" action that generates a new one and shows it once.
- [ ] Register route in `src/App.tsx`
- [ ] Add to sidebar under Identity & Access (see Part 10)

---

## Part 10 — Sidenav: assessment and recommended hierarchy

The current sidebar (`src/components/sidebar/constants.tsx`) needs these changes:

1. **Remove** "API Keys" from Applications — backend feature deleted (Part 7).
2. **Replace** with "Workload Identity" in the same position (Part 9.7).
3. **Add** "SCIM" under Authentication in Identity & Access (Part 9.8).
4. **Expand** Monitoring from a bare link to a group with sub-items — add Auth Events and Audit Log.

### Recommended final hierarchy

```
Overview
  - Get Started

Identity & Access
  - User Management
    - Users
    - Roles
    - Invitations
  - Authentication
    - Identity Providers
    - Registration Flows
    - SCIM                           ← NEW (9.8)

Applications & APIs
  - Applications
    - Clients
    - Workload Identity              ← NEW (9.7) — replaces API Keys
  - APIs & Resources
    - Services
    - APIs
    - Policies

Security
  - Security
    - Password Policy
    - Multi-Factor Auth
    - Account Lockout
    - Sessions
    - Tokens
    - Registration
    - Attack Protection
    - IP Restrictions

Branding & Messaging
  - Branding
    - Branding Templates
    - Email Templates
    - SMS Templates
  - Messaging
    - Email Delivery
    - SMS Delivery

Operations
  - Events & Webhooks
    - Webhooks
    - Event Routes
    - Event Types
  - Monitoring                       ← expand from bare link to group
    - Auth Events                    ← was the /logs bare link
    - Audit Log                      ← NEW (9.1)

Administration
  - Organization
    - Tenants
  - Settings
```

### Exact changes to `src/components/sidebar/constants.tsx`

- [ ] **Remove** from Applications `items` array: `{ title: "API Keys", route: "/api-keys" }`
- [ ] **Add** to Applications `items` array: `{ title: "Workload Identity", route: "/workload-identity" }`
- [ ] **Add** to Authentication `items` array: `{ title: "SCIM", route: "/scim" }`
- [ ] **Change** the bare Monitoring entry to a group with sub-items:
  ```ts
  // BEFORE:
  {
    title: "Monitoring",
    route: "/logs",
    icon: TrendingUp,
  },

  // AFTER:
  {
    title: "Monitoring",
    route: "/monitoring",
    icon: TrendingUp,
    items: [
      { title: "Auth Events", route: "/logs" },
      { title: "Audit Log", route: "/audit-log" },
    ],
  },
  ```
- [ ] Import any new icons needed. Suggestions: `Network` or `GitBranch` for Workload Identity; `Database` or `RefreshCw` for SCIM; `ClipboardList` for Audit Log — all available in `lucide-react`.

**Acceptance:** Sidebar has no "API Keys" entry. Workload Identity, SCIM, and Audit Log are navigable from the sidebar. Existing nav items still work. `tsc -b` exits 0.

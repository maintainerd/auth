# Identity, OAuth2, Login & Invitations — Architecture & Plan

> **Status:** Planning / design reference. Mixes **what exists today** (with file references) and
> **what is recommended / still missing**. Another agent should be able to pick this up cold.
> Verify the "exists today" claims against the cited files before building on them — they were
> traced during planning and may have drifted.
>
> **Repos**
> - Backend (this repo): `maintainerd-auth` — Go.
> - Admin console frontend: `maintainerd-auth-console` — React/TS, **internal, behind VPN**.
> - End-user identity frontend: `maintainerd-auth-identity` — React/TS, **public, NOT yet created**.

---

## 1. The two front-end apps (and which auth they use)

| | **auth-console** (admin) | **auth-identity** (end users) |
|---|---|---|
| Audience | Admins / operators | End users (customers of the developers using our platform) |
| Network | **Internal, behind VPN** — end users cannot reach it | **Public** (internet-facing) |
| Backend port/router | Internal API router (JWT bearer) | **Public** router (cookie session + CSRF) |
| Login | **Internal login only** — username/password. **No OAuth2 here.** | **Public login** (username/password) **+ OAuth2** authorization flows |
| Registration / invites | Admin **sends** invites here | Invite **onboarding**, external-app **signup**, OAuth2 all happen here |
| Status | Exists | **To be built** |

**Key rule:** OAuth2 (authorization_code, consent, token) lives only on **auth-identity / the public
backend**. The admin console is a private, first-party tool — it just does internal login and never
participates in OAuth2.

**More rules (decided):**
- **Invites are public-only.** The admin *sends* an invite from the console (`POST /invite`), but
  invite **onboarding/registration happens only on the public app** (`auth-identity`). There must be
  **no internal `/register/invite` route** — remove it (see TODO). Invited users never touch the
  internal/VPN app.
- **Public register fallback.** Public `POST /register` should treat `client_id`/`provider_id` as
  **optional**: when absent, register the user under the **default client of the default (system)
  tenant** — mirroring how the internal `Register()` already falls back to the system/default client.
  (Today the public handler hard-requires them — this needs to change to a fallback.)

---

## 2. OAuth2 vs Login — the two layers (this is the core mental model)

There are **two distinct layers**. They are separate, and one feeds the other.

### Layer 1 — Primary authentication (the IdP's own login/registration)
- The user proves who they are **to our identity provider** via username/password (or registers).
- This establishes a **session** with the IdP (a **cookie** on the public app).
- This is **NOT** OAuth2. It is "are you logged into our identity app" — exactly like "are you
  logged into your Google account."
- Backend: `internal/authn/handler_login.go` → `LoginPublic` (sets a cookie session via
  `internal/platform/cookie`; honors an `X-Token-Delivery` header to choose cookie vs body token).
  `Logout` clears the auth cookies. MFA step (`MFALoginVerify`, `MFALoginSendSMS`,
  `MFALoginWebAuthnBegin`) elevates the session to `acr=2`.

### Layer 2 — OAuth2 / OIDC authorization (delegation to a client app)
- A **client app** (a developer's app, e.g. a medical app) wants tokens for the user.
- It redirects the browser to `GET /oauth/authorize?client_id=…&redirect_uri=…&response_type=code&scope=…`.
- The authorize endpoint **requires an existing Layer-1 session**, then (optional consent) issues a
  short-lived **authorization code** and redirects to the client's **registered** `redirect_uri`.
- The client exchanges the code at `POST /oauth/token` for access/refresh/ID tokens.
- Backend: `internal/oauth/routes.go`
  - `GET /oauth/authorize` — **JWT/session required** (this is the proof of the rule below)
  - `GET /oauth/consent/{challenge_id}`, `POST /oauth/consent`
  - `POST /oauth/token` — unauthenticated, dispatches by `grant_type`
  - mounted at `internal/server/router.go` (`oauth.OAuthInternalRoute`).

### How they connect
```
/oauth/authorize  ──requires──▶  Layer-1 session
   • no session  → identity app shows LOGIN (or SIGNUP) → user authenticates → session created
   • has session → proceed → issue code → redirect to client.redirect_uri?code=…
                                                   │
                                          client → POST /oauth/token → tokens
```
Because `GET /oauth/authorize` requires a JWT/session, **the user must already be logged into
auth-identity before OAuth2 can issue a code** — same as Google. So **login (username/password →
session) is separate from, and prior to, the OAuth2 code flow.** ✅

> Registration / invite-onboarding / external-app signup are all **Layer-1** events (they create the
> account and log the user in), which then typically continue into **Layer-2** (authorize → code →
> back to the developer's app with tokens).

---

## 3. Current backend building blocks (exists today — verify against cited files)

### 3.1 Invitations — **substantially built** (`internal/invite/`)
- **Tables:** `invites` (migration `039_create_invites_table.go`), `invite_roles`
  (`040_create_invite_roles_table.go`).
- **Model** `model_invite.go`: `InviteID`, `InviteUUID`, `TenantID`, `ClientID` (**currently always
  the system client** — see service), `InvitedEmail`, `InvitedByUserID`, `InviteToken` (32-byte
  `crypto.GenerateIdentifier`), `Status` (`pending|accepted|expired|revoked`), `ExpiresAt`
  (default **72h** = `DefaultInviteTTL`), `UsedAt`, audit fields.
  **No `redirect_url` / `callback_url` / `template_id` / `signup_flow_id` field.**
- **Service** `service_invite.go` → `SendInvite(ctx, tenantID, email, userID, roleUUIDs)`:
  - tx: validate system client, validate roles belong to tenant, create `invite` + `invite_roles`.
  - **signed URL:** `signedurl.GenerateSignedURL(AppPrivateHostname + "/register/invite", {invite_token}, 72h)`
    → `signedurl.ConvertToFrontendURL(…, AppFrontendIdentityHostname + "/register/invite")`.
  - sends email via template `internal:user:invite`.
  - **Signature only accepts email + roleUUIDs — no redirect/template parameter.**
- **Repo** `repository_invite.go`: `FindByToken` (roles preloaded), `MarkAsUsed` (→ accepted +
  used_at), `RevokeByUUID` (→ revoked), `FindAllByClientID`, `FindAllByTenantID`.
  **No resend/regenerate method.**
- **Route** `routes.go`: `POST /invite` — **JWT + UserContext only; NOT gated by `user:invite`.** ⚠️
- **Permission** `user:invite` exists (`internal/setup/seeder/004_permission.go`) but is **not
  enforced** on the route.
- Consumed by registration: `internal/authn/service_register.go` → `RegisterInvite` (~L442) and
  `RegisterInvitePublic` (~L608): validate token (pending + not expired), take email from
  `invite.InvitedEmail`, create **active** user with password, assign default role **+ invite
  roles**, `MarkAsUsed`. Handler `POST /register/invite` (`internal/authn/handler_register.go`).

### 3.2 Signed URLs (`internal/platform/signedurl/signedurl.go`)
- `GenerateSignedURL(baseURL, params, ttl)` — HMAC-SHA256 over sorted params + `expires`.
- `ValidateSignedURL(values)` — checks expiry + recomputes HMAC (tamper-evident).
- `ConvertToFrontendURL(apiSignedURL, frontendBase)` — preserves query + signature.

### 3.3 Password reset — the closest **reference pattern** (`internal/authn/service_forgot_password.go`)
- `generateSecureToken(32)` → `UserToken` row, **1h** expiry → signed URL with
  `{token, client_id, provider_id}` → email `internal:user:password:reset` → `GET /reset-password`
  validates signature before allowing the change. Mirror this shape for invite resend if helpful.

### 3.4 Email (`internal/platform/email/email.go`, seeder `010_email_template.go`)
- `email.SendEmail(ctx, db, SendEmailParams{To, Subject, BodyHTML, BodyPlain})`.
- Seeded templates: `internal:user:invite`, `internal:user:password:reset`,
  `internal:user:email:verification`, `internal:user:magic_link`.
- Note: invite email passes `LogoURL` **empty** today (`service_invite.go`). Minor.

### 3.5 User lifecycle fields (`internal/user/model_user.go`)
- `Status` (`active|inactive|suspended`), `IsEmailVerified`, `IsPhoneVerified`,
  `IsProfileCompleted`, `IsAccountCompleted`, `Password` (`*string`, nullable),
  `PasswordChangedAt`, `ForcePasswordChange`, `IsTOTPEnabled`, `IsWebAuthnEnabled`, `MFAEnabledAt`.

### 3.6 Signup Flows — **exists but DORMANT** (`internal/idp/`)
- **Tables:** `signup_flows` (migration `037`), `signup_flow_roles`.
- **Model** `model_signup_flow.go`: `SignupFlowID`, `SignupFlowUUID`, `TenantID`, `Name`,
  `Description`, `Identifier` (unique per tenant), `Config` (JSONB), `Status` (`active|inactive`),
  `ClientID`, audit fields. Roles via `signup_flow_roles`.
- **Service/Handler/Routes** `service_signup_flow.go`, `handler_signup_flow.go`,
  `SignupFlowRoute` in `internal/idp/routes.go`: full CRUD + assign/get/remove roles, gated by
  `signup-flow:read|create|update|delete`.
- **⚠️ Consumed by NOTHING.** Neither register nor invite reads a signup flow. It is admin CRUD only.
- Frontend (`auth-console/src/pages/signup-flows`): fields name/description/status
  (`active|inactive|draft`)/clientId + free-form `Config` key/values + `autoApproved`.

### 3.7 Clients & redirect URIs (`internal/client/`)
- **Client** `model_client.go`: OAuth fields (`GrantTypes`, `ResponseTypes`, `AccessTokenTTL`,
  `RefreshTokenTTL`, `RequireConsent`, `AllowedScopes`, `TokenEndpointAuthMethod`), `ScopeClaimMappings`,
  `ClaimMappers`, `IsSystem`, `IsDefault`, relationships `ClientURIs`, `ClientAPIs`.
  **No default-roles field.**
- **ClientURI** `model_client_uri.go` (migration `016`): `URI` + `Type` ∈
  `redirect-uri | origin-uri | logout-uri | login-uri | cors-origin-uri`.
  → **Clients already own the redirect-URI allow-list.** This is where post-auth redirects must be
  validated.

### 3.8 Roles / permissions baseline (from prior permission work)
- `registered` role = **account/self permissions only** (no admin perms); assigned to every user by
  default. `super-admin` = **all non-self permissions** (no duplication of self perms). Bootstrap
  admin gets **both** roles.
- Relevant permissions: `user:invite` (defined, not yet enforced), `role:restrict-super-admin`
  (hook for privilege guard), `signup-flow:*`.

---

## 4. Target flows (examples — keep these as the canonical reference)

### 4.1 Internal admin login (auth-console, no OAuth2)
```
Admin opens console (VPN) → username/password → internal login endpoint
  → session (JWT/cookie) → admin uses the console
No OAuth2 anywhere in this path.
```

### 4.2 Public login + OAuth2 (auth-identity, "Google-style")
```
Dev app → redirect browser → auth-identity GET /oauth/authorize?client_id&redirect_uri&response_type=code&scope
  ├─ no session → auth-identity shows LOGIN page → POST public login (username/password)
  │                 → (MFA if required → acr=2) → session cookie set
  └─ has session → (consent if client.RequireConsent) → issue authorization code
       → redirect to client.redirect_uri?code=…   (redirect_uri MUST be a registered client_uri)
Dev app backend → POST /oauth/token (code + client creds) → access/refresh/ID tokens
```

### 4.3 External-app signup via a signup flow (self-service registration)
```
Dev app "Register" button → auth-identity signup entry point
   (carries client_id + signup_flow identifier; e.g. via /oauth/authorize with a signup hint,
    or a dedicated /signup?flow=…&client_id=… page)
  → auth-identity shows registration form (email, password, …)
  → backend creates user, assigns the signup_flow's predefined roles, logs user in (session)
  → continue OAuth2 authorize → code → redirect to client.redirect_uri
Result: user registered with the right roles, dev app receives tokens, lands on dev app page.
"Multiple onboarding types" = multiple signup flows (e.g. Doctor / Nurse / Patient) → different roles.
```

### 4.4 Admin invite → onboarding (invited, pre-authorized signup)
```
[auth-console] Admin → Invitations → "Invite user"
  → enter email + OPTIONALLY pick an auth_flow (which carries roles + branding + client + callback)
  → POST /invite  (gated by user:invite + step-up)
  → backend: create invite (status=pending, token, 72h, auth_flow_id nullable). NO invite_roles.
       • with auth_flow  → its roles/branding/client/redirect apply on accept
       • without auth_flow → default registered role only, active branding, no redirect
  → email with SIGNED URL → {AppFrontendIdentityHostname}/register/invite?invite_token=…&expires=…&sig=…

[email] Invited user clicks link
  → opens auth-identity onboarding page (public app)
  → page validates the signed URL, shows onboarding form (username/password ONLY — email is locked)
  → POST /register/invite (public) {invite_token, username, password}
  → backend: validate token (pending + unexpired), create ACTIVE user with invited_email,
       set is_email_verified=TRUE (recommended), grant default role + the auth_flow's roles
       (if any), MarkAsUsed (accepted)
  → log user in (session); if the auth_flow has a callback → redirect there (its selected
       client_uris) via the OAuth code flow; otherwise show success (no external redirect)
Result: user onboarded with predefined roles, lands in the developer's app (e.g. medical app) to
        continue domain-specific data collection (health info, etc.) — NOT collected by the auth app.
```

### 4.5 Resend / re-trigger an expired invite (to build)
```
[auth-console] Admin → Invitations → row "Resend"
  → POST /invite/{invite_uuid}/resend
  → backend: invalidate old token, generate NEW token + NEW expiry, reset status=pending,
       regenerate signed URL, re-send the invite email
```

---

## 5. The Auth Flow decision (rename `signup_flows` → `auth_flows`, KEEP + WIRE IN)

**Decision:** `signup_flows` was meant to be the reusable, client-scoped configuration we want, but
it is currently **dormant** (wired into nothing). **Rename it to `auth_flows`, keep it, and actually
wire it into the login + registration paths.** One concept is reused by **regular signup, invite
onboarding, and oauth2**.

**Why `auth_flows` (not `signup_flows`):** it governs the whole hosted **login + registration
experience** for a client — on the **registration** side it grants special roles; on the **login**
side it only changes the look. So it's broader than signup. (It is a *static config*, NOT a
conditional-auth/rules engine — document that so nobody mistakes it for an Actions/rules pipeline.
Alternative name if that connotation worries us: `auth_experiences`.)

**Target shape (lean) — everything optional with sensible defaults:**
```
auth_flows:       { id, uuid, tenant_id, name, identifier, status,
                    client_id              (FK → clients,     NULLABLE → null = default client),
                    branding_id            (FK → branding,    NULLABLE → null = active/default branding),
                    redirect_client_uri_id (FK → client_uris, NULLABLE → null = no redirect back) }
auth_flow_roles:  { auth_flow_id, role_id }     // special roles granted on registration
```
- **Client is optional** → if not set, registration happens under the **default client** (of the
  default/system tenant).
- **Branding is optional** → if not set, use the tenant's **active** branding (reuse the existing
  `branding` table — do NOT create `auth_brandings`; allow many `branding` rows per tenant).
- **Callback is optional and is a *selected* `client_uris` row** (a pre-registered redirect URI of
  the client — never free text, so it's allow-listed by construction). If none selected → the
  onboarding page **does not redirect back** to any external app (just shows success).
- Drop the unused `Config` JSONB / `autoApproved` / `draft` cruft.

**Usage by mode:**
- **Regular signup:** `auth_flow` optional. With it → grant its roles + its branding + optional
  redirect. Without it → only the default `registered` role + active branding + no redirect.
- **Invite:** `auth_flow` optional. With it → the flow's roles + branding + optional redirect apply.
  Without it → plain onboarding: only the default `registered` role, active branding, no redirect.
- **OAuth2:** `client_id` required (from the `/authorize` request), `auth_flow` optional; branding
  changes the login/consent look only.

The invite **references** a flow by id (`invite.auth_flow_id`, nullable) — it does **not** carry its
own roles. **`invite_roles` is dropped** (redundant: roles come from `auth_flow_roles`).

### 5.1 Branding / theming model (decided)

**Purpose:** let the organization running this open-source app customize the **look** of **both**
`auth-console` (admin) **and** `auth-identity`. **Scope is intentionally minimal for now: colors + a
logo only** — e.g. primary, secondary, accent, panel/background, sidebar, card, top-bar colors, plus
the panel logo. **No custom CSS** (too complicated for now) and **no layout/template customization**
(that's why `login_templates` is dropped). Scoped per tenant.

**Records & lifecycle:**
- **System branding** — seeded at startup, `is_system = true`. **Not updatable, not deletable.** It
  is the initial **active** default and the permanent fallback. Both apps can run on it with zero
  config.
- **Custom branding** — to customize, the admin **creates a new branding row** and **activates** it.
  The app loads the **active** branding's theme. **Exactly one active per tenant** — activating one
  deactivates the others. If no custom is active → fall back to the system branding.
- **Non-active brandings** are not used for the global chrome; they're available to be referenced by
  an `auth_flow.branding_id` for a **custom login/registration look** (per developer app / per flow).
  `auth_flow.branding_id = null` → use the tenant's active/default branding.

**How the apps consume it:**
- Global chrome of **both console and identity** = the tenant's **active** branding.
- A specific hosted login/registration experience = `auth_flow.branding_id` if set, else active.
- `auth-identity` needs to read branding **pre-login (unauthenticated)** to theme the login page →
  expose the active branding (and a flow's branding) via a **public** config/branding read endpoint
  (it's non-sensitive look data). The console reads it authenticated.

**`branding` table changes:**
- ADD `name` (label each theme, e.g. "Acme Dark"), `is_system` (immutable seeded record),
  `is_active` (the default the app loads; **exactly one true per tenant**).
- **DROP `custom_css`** (no CSS in scope).
- Palette: store the minimal color tokens (primary / secondary / accent / panel-bg / sidebar / card /
  top-bar) — recommend a single `colors` JSON (or reuse `metadata`) so adding a token needs no
  migration — plus keep `logo_url`. The extra existing fields not in scope (`company_name`,
  `favicon_url`, `font_family`, `support_url`, `privacy_policy_url`, `terms_of_service_url`) can be
  dropped or left dormant — not used by the minimal theme.

**Login templates:** **DROPPED** — no layout/template customization in scope. Everything is colors +
logo in `branding`. Remove the `login_templates` table, model/handlers/routes, and `login-template:*`
permissions.

---

## 6. Recommendations (security & enterprise best practices)

### 6.1 Invitations (carry over verbatim — must-haves)
1. **Lock the email to the invite.** The user must register under `invited_email` and not be able to
   change it. The backend already takes the email from `invite.InvitedEmail` (good) — just make sure
   the onboarding form doesn't accept an email field.
2. **Mark email verified on accept.** They proved they control the inbox by clicking the link, but
   `RegisterInvite` currently leaves `is_email_verified=false`. Enterprise flows treat invite
   acceptance as email verification.
3. **Privilege-escalation guard.** An inviter should not be able to grant roles more powerful than
   their own (e.g., a user-admin inviting someone as super-admin). This is the classic enterprise
   pitfall — add a check that the invited roles are a subset of what the inviter is allowed to grant.
   (`role:restrict-super-admin` already exists in the catalog as a hook for this.)
4. **Step-up for sensitive invites.** Require step-up MFA on `POST /invite`, at least when
   admin-level roles are attached. Pairs with gating the route on `user:invite`.
5. **Manage pending invites.** The Invitations page should list status
   (pending/accepted/expired/revoked), who invited, expiry — and support **revoke** and **resend**.
   Backend already has `RevokeByUUID`, `FindAllByTenantID`, `invited_by_user_id`; just surface them.
6. **Single-use + sane expiry.** Already single-use (status flips to `accepted`) and 72h TTL — good.
   Enterprises often make TTL configurable (and 7 days is common). Minor.
7. **Onboarding completeness.** Decide if acceptance should also force MFA enrollment / profile
   completion (the model has `is_profile_completed`, `is_account_completed`, `mfa_*` flags to drive
   this). Many enterprise onboardings require MFA setup on first login.

### 6.2 Architecture / OAuth2
8. **Redirect back via the OAuth2 code flow, not a bare redirect.** "Success → go to dev app" means
   complete `/oauth/authorize` → `code` → client's **registered** `redirect_uri`. Gives the dev app
   real tokens and auto-validates the destination against `client_uris` (no open redirect).
9. **Tie signup flow → client.** The "developer's app" *is* an OAuth client. The signup flow
   references the client; roles are the flow's roles; redirect is the client's `redirect_uri`.
10. **Keep the auth app lean.** Identity concerns only (register/login/MFA/tokens). Domain data
    (e.g. medical/health info) is collected by the developer's app AFTER redirect — never by the auth
    app. This is the reason to keep signup flows minimal and to not rebuild complex multi-step signups.

### 6.3 Related (separate track, note only)
11. **Passkey registration options** (not invite-specific): `BeginRegistration` is currently called
    with no `AuthenticatorSelection`, so the web app runs the older "security key" ceremony rather
    than the full passkey UI. Adding `residentKey: "preferred"` + `userVerification: "preferred"`
    (attachment left unset) surfaces Windows Hello / Touch ID / password-manager passkeys. Tracked
    separately from invites.

---

## 7. What's MISSING / TODO

### 7.1 Backend (`maintainerd-auth`)
- [x] **Remove the internal `/register/invite` route + `RegisterInvite` handler.** Invites are
      **public-only** — invite onboarding must happen on `auth-identity` via the public
      `/register/invite` (`RegisterInvitePublic`). The internal invite-register path should not exist.
      (The admin invite-**send** route `POST /invite` stays on the internal/console side.)
- [x] **Public `POST /register` fallback.** Make `client_id`/`provider_id` optional on the public
      register handler; when absent, register under the **default client of the default (system)
      tenant** (mirror internal `Register()`'s existing system/default-client fallback). Today the
      public handler hard-requires them.
- [x] **Rename `signup_flows` → `auth_flows`** (table + model + service + routes + permissions
      `signup-flow:*` → `auth-flow:*`). Make `client_id` **nullable** (null=default client); add
      `branding_id` (nullable→active branding) + `redirect_client_uri_id` (FK→`client_uris`,
      nullable→no redirect); keep `auth_flow_roles`; drop `Config`/`autoApproved`/`draft`.
- [x] **Drop `invite_roles`** (table + model + preload). Add `auth_flow_id` (nullable FK) to
      `invites`. Invite roles now come from the attached flow's `auth_flow_roles`; no flow → default
      `registered` role only.
- [x] **Guard flow deletion:** don't allow deleting an `auth_flow` referenced by pending invites
      (or null-safe fall back to "no special roles" on accept). Mitigates the reference-not-snapshot
      tradeoff.
- [x] **Wire `auth_flows` into login + register + invite** (currently dormant): resolve flow →
      grant roles on register + apply branding on login/register; redirect via the client's
      `redirect_uri` (validated against `client_uris`).
- [x] **Branding theming model:**
  - Add `name`, `is_system`, `is_active` columns to `branding`; **drop `custom_css`**; store the
    minimal color palette (recommend a `colors` JSON) + `logo_url`. (migration + model)
  - **Seed an immutable system branding** at startup (`is_system=true`, initially `is_active=true`).
  - Enforce in the service: **system record is not updatable/deletable** (reject regardless of
    permission); **exactly one active per tenant** (activating one deactivates the rest; never zero —
    fall back to system).
  - Add permissions `branding:create`, `branding:delete`, `branding:activate` (today only
    `branding:read`/`branding:update`).
  - **Public read endpoint** for the active branding (+ a given `auth_flow`'s branding) so the
    public `auth-identity` app can theme login/registration pre-auth (non-sensitive).
  - **DROP `login_templates`** — table + model/handlers/routes + `login-template:*` permissions
    (no layout customization in scope).
- [x] **Gate `POST /invite`** with `PermissionMiddleware(["user:invite"])` (+ `RequireStepUp`).
      Today it's JWT-only (`internal/invite/routes.go`).
- [x] **Privilege-escalation guard** in `SendInvite` — invited roles ⊆ inviter's grantable roles.
- [x] **`SendInvite` takes a signup-flow/template reference** (and/or client) instead of bare
      `roleUUIDs`; snapshot roles (+ client/redirect) onto the invite. Add `signup_flow_id`
      (or `template_id`) + the resolved client to the `invites` table.
- [x] **`ResendInvite(inviteUUID)`** service + `POST /invite/{invite_uuid}/resend` route + repo
      method (new token, new expiry, status→pending, fresh signed URL, re-send email, invalidate old).
- [x] **Mark `is_email_verified=true`** on invite acceptance in `RegisterInvite` /
      `RegisterInvitePublic`.
- [x] **Wire signup flows into signup + invite** (currently dormant): resolve flow → roles → client
      → complete OAuth code flow to the client's `redirect_uri`. Trim the model to lean fields.
- [x] **Validate redirect against `client_uris`** wherever a post-auth/post-registration redirect is
      produced.
- [x] **Configurable invite TTL** (env or per-flow), default 72h or 7 days.
- [x] (Optional) populate invite email `LogoURL` (branding).
- [x] (Optional) onboarding completeness: force MFA enrollment / profile completion via the
      `is_profile_completed` / `is_account_completed` / `mfa_*` flags.
- [x] Confirm/clean the `invite.TenantID` vs system-client tenant relationship in `SendInvite`
      (currently `TenantID` arg vs tenant derived from the system client — ensure they can't diverge).

### 7.2 Frontend — auth-console (admin, exists)
- [ ] **Invitations page** at `/invites` (sidebar entry exists, route/page do not):
      list invites (status/email/roles/invited-by/expiry) with **revoke** + **resend**; "Invite user"
      form (email + pick signup flow/template). Calls `POST /invite`.
- [ ] Optionally a Signup-Flow/Template management UI (or reuse existing `signup-flows` page, trimmed).

### 7.3 Frontend — auth-identity (public, **NOT yet created**)
- [ ] New public app `maintainerd-auth-identity` (works like Google: hosted login + signup pages).
- [ ] **Public login** page → public login endpoint (username/password → session) + MFA step.
- [ ] **OAuth2 authorize/consent** screens (or rely on backend-rendered consent).
- [ ] **Invite onboarding** page at `/register/invite`: validate signed URL, show username/password
      form (email locked), call **public** `POST /register/invite`, then continue OAuth → dev app.
- [ ] **External-app signup** entry (`/signup?flow=…&client_id=…` or via `/oauth/authorize` signup
      hint) → register with the flow's roles → continue OAuth → dev app.
- [ ] Must use the **public** backend endpoints (client_id/provider_id), never internal ones.

---

## 8. Open questions / decisions to confirm

**Decided:**
- `signup_flows` → **`auth_flows`** (governs login look + registration roles; static config, not a
  rules engine). Reuse existing **`branding`** via `auth_flows.branding_id` (nullable); no
  `auth_brandings` table. Keep `login_templates` separate/optional.
- **Invites are public-only** (remove internal `/register/invite`).
- **Public register** falls back to default client/default (system) tenant when `client_id`/
  `provider_id` absent.
- **Branding** = per-tenant theme (colors/font/logo/minimal CSS) for both apps. One immutable
  **system** record (seeded, not updatable/deletable) + optional custom records; **one active** drives
  the global look; non-active records used per-flow via `auth_flows.branding_id`. `login_templates`
  deprecated/unused.

- **Invites reference `auth_flow_id`** (nullable); **`invite_roles` dropped** (roles come from the
  flow). Everything on a flow is optional: client (→ default client), branding (→ active branding),
  callback (→ a selected `client_uris`, else no redirect).

**Still open:**
- Reference-vs-snapshot tradeoff: invites *reference* the flow, so editing/deleting a flow changes
  what a pending invite grants. Mitigation chosen: **guard flow deletion** while pending invites
  reference it (and null-safe fall back to "no special roles"). Confirm this is acceptable vs.
  re-introducing a snapshot.
- Entry mechanism for external signup: a dedicated `/signup` page vs a `screen_hint=signup`-style
  parameter on `/oauth/authorize`.
- Invite TTL: keep 72h, or make configurable / default 7 days?
- Force MFA enrollment at onboarding (yes/no, and for which flows/roles)?
- "Staged user" model (create user at invite time in a pending state) vs current model (user created
  only on acceptance). Current = no ghost users; admins track unaccepted invites on the Invitations
  page. Confirm this is intended.

---

## 9. Table inventory (entire implementation)

> Confirmed table names + migration numbers. "Core" identity/RBAC/MFA tables are reused unchanged;
> their exact names are indicative — verify against migrations when implementing.

### Create / rename / modify
| Table | Action | Notes |
|---|---|---|
| `auth_flows` | **RENAME** from `signup_flows` (mig 037) | make `client_id` **nullable** (null=default client); add `branding_id` (FK→`branding`, nullable, null=active branding) + `redirect_client_uri_id` (FK→`client_uris`, nullable, null=no redirect); drop `config`; keep `name`, `identifier`, `status` |
| `auth_flow_roles` | **RENAME** from `signup_flow_roles` (mig 038) | the ONLY place roles are defined for flows/invites |
| `branding` | **MODIFY** (mig 003) | add `name`, `is_system`, `is_active`; **drop `custom_css`**; minimal color palette (recommend `colors` JSON) + `logo_url`; one active per tenant; immutable system row |
| `invites` | **MODIFY** (mig 039) | add `auth_flow_id` (FK→`auth_flows`, **nullable**); gate send on `user:invite`; add resend; mark email verified on accept |

### Reuse as-is (depended on, no schema change)
| Table | Mig | Used for |
|---|---|---|
| `clients`, `client_uris`, `client_apis`, `client_permissions` | 016 (uris) | OAuth clients + **redirect-URI allow-list** (callbacks pick a `client_uris` row) |
| `email_templates` | 045 | invite + password-reset emails |
| `sms_templates` | 046 | SMS MFA / notifications (not part of branding) |
| `users`, `roles`, `permissions`, `user_roles`, `role_permissions` | core | identity + RBAC |
| `user_identities`, user tokens/sessions | core | identities; sessions + reset/invite tokens |
| MFA tables (TOTP secrets, WebAuthn creds, backup codes, SMS phones) | core | MFA factors |

### Drop
| Table | Mig | Reason |
|---|---|---|
| `login_templates` | **044** | No layout/template customization in scope — theming is colors + logo via `branding` only. Also remove its model/handlers/routes + `login-template:*` permissions. |
| `invite_roles` | **040** | Redundant — invite roles now come from the attached `auth_flow` (`auth_flow_roles`). Drop table + model + any preload/usage. |

> `signup_flows` / `signup_flow_roles` are **renamed** to `auth_flows` / `auth_flow_roles` — not
> dropped (if your migration tooling can't rename, do drop-and-recreate, but treat it as a rename).

---

## 10. Glossary
- **Primary authentication / Layer 1** — the IdP's own login/registration that creates a session.
- **OAuth2 authorization (code flow) / Layer 2** — issuing tokens to a client app after Layer 1.
- **Client** — a registered OAuth app (a developer's app); owns redirect URIs and OAuth settings.
- **Auth flow** (`auth_flows`, formerly `signup_flows`) — a reusable, client-scoped config for the
  hosted login + registration experience: predefined roles granted on registration + an optional
  `branding` for the look. Used by regular signup, invite onboarding, and oauth2.
- **Branding** (`branding`) — the app **theme**: colors, font, logo, minimal custom CSS. Customizes
  **both** console and identity. Per tenant: one immutable **system** record (seeded, not
  updatable/deletable) + optional custom records; exactly **one active** drives the global look.
  Non-active records are used per-flow via `auth_flows.branding_id`.
- **Invite** — a pre-authorized, email-bound, single-use, time-limited signup (token + signed URL).
- **acr=2** — session elevated via MFA/step-up.

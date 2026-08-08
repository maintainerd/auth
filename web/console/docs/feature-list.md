# maintainerd-auth-console — Feature List

A complete inventory of management capabilities exposed (or expected to be exposed) by the **Maintainerd Auth Console**, mapped against the corresponding capabilities of the [`maintainerd-auth`](https://github.com/maintainerd/auth) backend.

This file is the frontend counterpart to [`maintainerd-auth/docs/feature-list.md`](https://github.com/maintainerd/auth/blob/main/docs/feature-list.md). The backend file is the source of truth for what the platform *can* do; this file tracks what the console actually *exposes*.

Legend:

- `[x]` Implemented in the console (page + service + hook)
- `[~]` Partially implemented (notes inline — e.g. route exists but renders a placeholder, or only a subset of the API is wired up)
- `[ ]` Missing / not yet implemented in the console
- 🔴 Critical — required for the console to be useful in production
- 🟡 High priority
- 🟢 Medium / nice-to-have
- ⚪ Low / optional

> **Scope note.** This console targets the **internal management port** (`:8080`) only. End-user OAuth2 flows (login widget, consent screen, RP-initiated logout, hosted MFA challenge) belong in a separate public-facing app and are explicitly **out of scope** here. Items in the backend feature list that only make sense as end-user UI are omitted; items that an admin needs to **configure or audit** are included.

---

## Table of Contents

1. [Bootstrap & Setup](#1-bootstrap--setup)
2. [Console Authentication (Admin Self-Service)](#2-console-authentication-admin-self-service)
3. [Tenancy & Organizations](#3-tenancy--organizations)
4. [Users](#4-users)
5. [Roles & Permissions (RBAC)](#5-roles--permissions-rbac)
6. [OAuth2 / OIDC Client Management](#6-oauth2--oidc-client-management)
7. [API Keys](#7-api-keys)
8. [Identity Providers (Federation)](#8-identity-providers-federation)
9. [Services & APIs](#9-services--apis)
10. [Policies](#10-policies)
11. [Signup Flows](#11-signup-flows)
12. [Multi-Factor Authentication](#12-multi-factor-authentication)
13. [Session Management](#13-session-management)
14. [Password & Credential Policy](#14-password--credential-policy)
15. [Threat Detection & Anomaly](#15-threat-detection--anomaly)
16. [IP Restrictions](#16-ip-restrictions)
17. [Security Settings (Headers, CORS, CSRF, Cookies)](#17-security-settings-headers-cors-csrf-cookies)
18. [JWT, Tokens & Cryptographic Keys](#18-jwt-tokens--cryptographic-keys)
19. [Branding (Login, Email, SMS Templates)](#19-branding-login-email-sms-templates)
20. [Notifications & Email/SMS Delivery](#20-notifications--emailsms-delivery)
21. [Audit Log & Auth Events](#21-audit-log--auth-events)
22. [Webhooks](#22-webhooks)
23. [Analytics & Dashboards](#23-analytics--dashboards)
24. [General Settings](#24-general-settings)
25. [SCIM (Provisioning)](#25-scim-provisioning)
26. [Console Quality (Tooling, Tests, A11y, i18n)](#26-console-quality-tooling-tests-a11y-i18n)

---

## 1. Bootstrap & Setup

- [x] First-run tenant creation page (`pages/setup/tenant`)
- [x] First-run admin creation page (`pages/setup/admin`)
- [x] First-run admin profile page (`pages/setup/profile`)
- [x] `isSetupCompleted` check before login
- [x] `createTenantWithDefaults` helper for default-metadata creation
- [ ] 🟢 Resumable setup state (refresh-tolerant wizard with persisted step)
- [ ] 🟢 Setup-stage progress indicator across the three steps

## 2. Console Authentication (Admin Self-Service)

- [x] Login page (`pages/login`) backed by `loginAsync`
- [x] Register page (`pages/register`) — used during setup and invited admins
- [x] Forgot-password page (`pages/forgot-password`)
- [x] Reset-password page (`pages/reset-password`)
- [x] Cookie-based session (Axios `withCredentials: true`)
- [x] Profile hydration on app load (`initializeAuthAsync`)
- [ ] 🟡 Change-password from inside the console
- [ ] 🟡 Step-up re-auth prompt before destructive actions (delete tenant, rotate keys)
- [ ] 🟡 "Sign out everywhere" button in profile menu (calls revoke-all-sessions)
- [ ] 🟢 Console-side MFA enrollment for admins (TOTP / WebAuthn)
- [ ] 🟢 Idle-timeout banner with auto-logout

## 3. Tenancy & Organizations

- [x] Tenant resolution by URL (`/:tenantId/...` + `determineTenantIdentifier`)
- [x] Active tenant in Redux (`store/tenant`)
- [x] Tenant fetch by identifier and default tenant fetch
- [x] Tenant member listing (`useTenantMembers`)
- [x] Tenant switcher in header
- [ ] 🟡 Tenant settings editor page (rename, metadata, branding defaults)
- [ ] 🟡 Tenant member management UI (invite, remove, change role)
- [ ] 🟡 Create-new-tenant flow from the console (post-bootstrap)
- [ ] 🟢 Tenant deletion with confirmation + cascade summary
- [ ] 🟢 Tenant export (JSON dump for migration / backup)
- [ ] 🟢 Tenant clone (template-based provisioning)
- [ ] 🟢 Hierarchical orgs / sub-organizations / projects view
- [ ] 🟢 Per-tenant feature-flag editor

## 4. Users

- [x] User list page (`pages/users`) with filters and pagination
- [x] User details page (`pages/users/details`)
- [x] Create / edit user form (`pages/users/form`)
- [x] User profile sub-form (`pages/users/profile-form`)
- [x] Update user status (activate / deactivate / suspend) — `useUpdateUserStatus`
- [x] Manual email-verified flag (`useVerifyUserEmail`)
- [x] Manual phone-verified flag (`useVerifyUserPhone`)
- [x] Mark account complete (`useCompleteUserAccount`)
- [x] User profiles CRUD (create, update, delete, set-default)
- [x] User identities listing (`useUserIdentities`)
- [x] Assign / remove user roles
- [ ] 🟡 Send password-reset email from the user details page
- [ ] 🟡 Force password change on next login (toggle)
- [ ] 🟡 Trigger email/phone verification from the admin
- [ ] 🟡 Revoke all sessions for a single user
- [ ] 🟡 Reset MFA (clear enrolled factors) per user
- [ ] 🟢 Bulk user import (CSV / JSON)
- [ ] 🟢 Bulk role assignment
- [ ] 🟢 User-impersonation ("view as user", audit-logged)
- [ ] 🟢 GDPR export per user
- [ ] 🟢 GDPR right-to-erasure with confirmation flow
- [ ] 🟢 Search by linked identity (provider + subject)

## 5. Roles & Permissions (RBAC)

- [x] Role list page (`pages/roles`)
- [x] Role details page (`pages/roles/details`)
- [x] Create / edit role form (`pages/roles/form`)
- [x] Role status update
- [x] Add / remove permissions on a role
- [x] Permission listing (`usePermissions`)
- [~] Permissions admin page (`pages/permissions` exists; surface depth not validated against backend permission catalogue)
- [ ] 🟡 Role cloning (duplicate-with-rename)
- [ ] 🟡 Effective-permissions inspector for a given user
- [ ] 🟡 Hierarchical / inherited roles
- [ ] 🟢 Group model (collection of users distinct from roles)
- [ ] 🟢 ABAC policy editor (attribute-based) alongside RBAC
- [ ] ⚪ Permission diff view between two roles

## 6. OAuth2 / OIDC Client Management

- [x] Client list page (`pages/clients`)
- [x] Client details page
- [x] Create / edit client form
- [x] Client status update
- [ ] 🔴 **Show `client_secret` only once at creation** (and force a regenerate flow afterwards)
- [ ] 🟡 Client-secret rotation UI with grace window
- [ ] 🟡 Per-client allowed-scopes editor
- [ ] 🟡 Per-client allowed-grant-types enforcement UI
- [ ] 🟡 Per-client redirect URI list editor with exact-match validation hint
- [ ] 🟡 Per-client token-TTL override fields
- [ ] 🟡 `require_consent` toggle visible in form
- [ ] 🟢 List & revoke consent grants per user (uses backend `/oauth/consent`)
- [ ] 🟢 Visual indicator for confidential vs public clients
- [ ] 🟢 Test-this-client utility (open `/oauth/authorize` in new tab with chosen scopes)
- [ ] 🟢 OIDC discovery + JWKS viewer (read-only `.well-known/*` browser)

## 7. API Keys

- [x] API key list page (`pages/api-keys`)
- [x] API key details page
- [x] Create / edit API key form
- [x] API key status update
- [x] API key config viewer (`fetchApiKeyConfig`)
- [ ] 🔴 **Show full key only once at creation**, masked thereafter
- [ ] 🟡 Per-key scope editor (which APIs / permissions)
- [ ] 🟡 Per-key expiry date picker + auto-rotation reminder
- [ ] 🟡 Last-used timestamp column in the list
- [ ] 🟢 Revoke + regenerate flow with audit reason

## 8. Identity Providers (Federation)

- [x] Identity provider list / details / form (`pages/identity-providers`)
- [x] Social provider list / details / form (`pages/social-providers`)
- [ ] 🟡 OIDC upstream provider configuration (Google, Microsoft, Apple, GitHub, GitLab) presets
- [ ] 🟡 Generic OAuth2 connector form
- [ ] 🟢 SAML 2.0 SP configuration UI (metadata upload, attribute mapping)
- [ ] 🟢 LDAP / Active Directory connector form
- [ ] 🟢 Attribute-mapping editor (upstream claim → local user field)
- [ ] 🟢 Home-realm-discovery (HRD) rule editor by email domain
- [ ] 🟢 Identity-link / unlink management per user
- [ ] 🟢 Provider connection-test button with diagnostic output

## 9. Services & APIs

- [x] Service list / details / form (`pages/services`)
- [x] API list / details / form (`pages/apis`)
- [x] Service status update
- [x] Assign / remove policy on a service
- [x] API status update
- [ ] 🟡 Per-API scopes/permissions editor surfaced inline
- [ ] 🟢 Service-to-service "trust" matrix view (which services can call which APIs)
- [ ] 🟢 API spec attachment (upload OpenAPI YAML for documentation)

## 10. Policies

- [x] Policy list / details / form (`pages/policies`)
- [x] Policy status update
- [x] Fetch services attached to a policy
- [ ] 🟡 Visual policy-rule editor (currently raw JSON only)
- [ ] 🟡 Policy simulator (given a request, show allow/deny + matching rule)
- [ ] 🟢 Policy diff view across versions
- [ ] 🟢 Policy import / export (JSON)

## 11. Signup Flows

- [x] Signup-flow list / details / form (`pages/signup-flows`)
- [x] Role assignment on signup
- [ ] 🟡 Step-by-step signup-flow builder (drag-and-drop, conditional steps)
- [ ] 🟡 Email-verification step toggle inside the flow editor
- [ ] 🟢 A/B test multiple flows per tenant
- [ ] 🟢 Localization of flow copy (i18n)

## 12. Multi-Factor Authentication

> The backend currently has email OTP utilities only; nothing else is implemented yet. The console has no MFA management UI today.

- [ ] 🟡 Per-tenant MFA policy editor (required / optional / risk-based)
- [ ] 🟡 Per-user enrolled-factors viewer (TOTP, WebAuthn, email OTP)
- [ ] 🟡 Reset-MFA action with audit reason
- [ ] 🟡 TOTP enrollment for console admins themselves
- [ ] 🟢 WebAuthn / passkey registration UI for admins
- [ ] 🟢 Backup-codes generation + display-once flow
- [ ] 🟢 Step-up auth indicator (which sensitive actions trigger re-auth)
- [ ] 🟢 SMS-OTP toggle and provider configuration
- [ ] 🟢 Push-notification 2FA configuration
- [ ] ⚪ Risk-based / adaptive MFA rule editor

## 13. Session Management

- [x] Session-management settings page (`pages/security/session-management`) — global session config (TTLs, concurrent-session limit) backed by `useSessionSettings`
- [ ] 🟡 Active-sessions list **per user** (read from backend `user_token`)
- [ ] 🟡 Revoke-single-session action
- [ ] 🟡 Revoke-all-sessions-for-user action
- [ ] 🟡 Session-revoked-on-password-change toggle
- [ ] 🟡 Session-revoked-on-permission-change toggle
- [ ] 🟢 Idle-timeout (sliding) configuration with preview
- [ ] 🟢 Absolute-lifetime cap configuration
- [ ] 🟢 Trusted-device list per user
- [ ] 🟢 Geo / IP anomaly visualization on session creation

## 14. Password & Credential Policy

- [x] Password-policies page (`pages/security/password-policies`) backed by `usePasswordPolicies`
- [ ] 🟡 Live preview of policy regex against test passwords
- [ ] 🟡 Per-tenant override editor (when global policies exist)
- [ ] 🟢 Pwned-password (HIBP) check toggle
- [ ] 🟢 Password-history depth setting (prevent re-use of last N)
- [ ] 🟢 Credential-rotation reminder configuration
- [ ] 🟢 Force-password-change-on-next-login bulk action

## 15. Threat Detection & Anomaly

- [x] Threat-detection settings page (`pages/security/threat-detection`) backed by `useThreatDetectionSettings`
- [ ] 🟡 Live threat dashboard (recent failed logins, lockouts, suspicious IPs)
- [ ] 🟡 Account-lockout configuration (threshold, lockout duration)
- [ ] 🟢 Brute-force protection rule editor
- [ ] 🟢 Geo-velocity ("impossible travel") detection toggle
- [ ] 🟢 Risk-score visualization per login attempt

## 16. IP Restrictions

- [x] IP-restrictions page (`pages/security/ip-restrictions`)
- [x] IP-restriction rules CRUD (`useIpRestrictionRules`)
- [x] IP-restriction global settings (`useIpRestrictionSettings`)
- [ ] 🟡 CIDR validation feedback in the rule editor
- [ ] 🟢 Per-route IP allow/deny rules (e.g. tighter on management endpoints)
- [ ] 🟢 IP geolocation hint inline with each rule
- [ ] 🟢 Test-rule simulator (paste an IP, see effective rule)

## 17. Security Settings (Headers, CORS, CSRF, Cookies)

- [x] Security-settings page (`pages/security/settings`) backed by `useSecuritySettings`
- [ ] 🟡 CORS allow-list editor with credential-mode awareness
- [ ] 🟡 CSP / security-headers preset selector + custom-overrides
- [ ] 🟡 Cookie-flag editor (Secure / SameSite / `__Host-` prefix)
- [ ] 🟢 HSTS preload toggle with eligibility check
- [ ] 🟢 Rate-limit configuration UI per route family

## 18. JWT, Tokens & Cryptographic Keys

> The backend exposes JWKS and key-rotation tooling but the console does not surface it.

- [ ] 🟡 JWKS viewer (read-only) showing active and retiring keys
- [ ] 🟡 Token TTL settings page (access / refresh / id token, per-client overrides)
- [ ] 🟡 Manual key-rotation button with confirmation flow
- [ ] 🟡 Algorithm selector (RS256 only enforced today; surface read-only)
- [ ] 🟢 Token denylist viewer (revoked `jti`s with TTL)
- [ ] 🟢 Refresh-token family inspector per user
- [ ] 🟢 Audience-restricted access-token viewer (`aud` per resource server)
- [ ] 🟢 Encrypted ID-token (JWE) toggle and key-management

## 19. Branding (Login, Email, SMS Templates)

- [x] Login template list / details / form (`pages/branding/login`)
- [x] Email template list / details / form (`pages/branding/email-templates`)
- [x] SMS template list / details / form (`pages/branding/sms-templates`)
- [~] Branding root route — currently renders the dashboard placeholder; needs a true overview / theme page
- [ ] 🟡 Live preview pane in template editors (render with sample data)
- [ ] 🟡 Per-tenant theme editor (logo, colors, copy)
- [ ] 🟡 Test-send button per template (sends to current admin's email)
- [ ] 🟢 Localized templates per locale (i18n)
- [ ] 🟢 Versioning + rollback for templates

## 20. Notifications & Email/SMS Delivery

- [x] Notifications inbox page (`pages/notifications`)
- [ ] 🟡 Provider configuration UI (SMTP / SES / SendGrid / Postmark / Mailgun / Resend)
- [ ] 🟡 SMS provider configuration UI (Twilio / SNS / Vonage)
- [ ] 🟡 Delivery-log viewer (per-message status, failures, retry count)
- [ ] 🟢 Sandbox-mode toggle for development
- [ ] 🟢 DMARC/SPF/DKIM verification helper for the sender domain

## 21. Audit Log & Auth Events

- [x] Log-monitoring page (`pages/log-monitoring`)
- [ ] 🟡 Filter by actor (user / admin / API key) and event type
- [ ] 🟡 Per-tenant scoped audit view
- [ ] 🟡 Export to CSV / JSON
- [ ] 🟡 Tamper-evident chain indicator (when backend ships HMAC chaining)
- [ ] 🟢 Live tail mode (server-sent events / polling)
- [ ] 🟢 Saved filters / alerts (notify when rule matches)
- [ ] 🟢 SIEM-export configuration UI (S3 / Kinesis / Kafka destination)

## 22. Webhooks

- [~] Route mounted at `/:tenantId/webhooks` but currently renders the dashboard placeholder
- [ ] 🟡 Webhook list / create / edit
- [ ] 🟡 HMAC secret display-once at creation, regenerate flow
- [ ] 🟡 Event-type subscription multi-select
- [ ] 🟡 Recent-deliveries panel (last N attempts + status)
- [ ] 🟢 Manual replay action per delivery
- [ ] 🟢 Dead-letter-queue inspector

## 23. Analytics & Dashboards

- [x] Analytics page (`pages/analytics`) — recharts-based
- [x] Dashboard landing page (`pages/dashboard`)
- [ ] 🟡 Real metrics fed by backend (currently likely placeholder data)
- [ ] 🟡 Login-success / failure trend chart
- [ ] 🟡 Tokens-issued chart
- [ ] 🟡 Active-users (DAU / WAU / MAU) widgets
- [ ] 🟢 Per-tenant breakdown
- [ ] 🟢 Exportable reports

## 24. General Settings

- [x] General settings page (`pages/settings`)
- [ ] 🟡 Tenant-level profile (display name, contact email, support URL)
- [ ] 🟡 Default locale + timezone
- [ ] 🟢 Theme selector (light / dark / system) — see Console Quality below

## 25. SCIM (Provisioning)

> Not implemented in the backend yet; included here so the gap is tracked.

- [ ] 🟢 SCIM bearer-token management UI (issue / revoke)
- [ ] 🟢 Inbound SCIM activity log
- [ ] 🟢 Outbound SCIM destination configuration

## 26. Console Quality (Tooling, Tests, A11y, i18n)

- [x] TypeScript strict mode (via `tsc -b`)
- [x] ESLint with React-Hooks plugin
- [x] Path alias `@/*` configured for both Vite and TS
- [x] Vite proxy for cookie-friendly local dev
- [x] React Query devtools-ready (provider in place; install devtools when needed)
- [ ] 🔴 Unit tests (no `*.test.ts(x)` files in repo today) — Vitest + React Testing Library
- [ ] 🔴 E2E tests (Playwright) covering at minimum: login, setup wizard, user CRUD
- [ ] 🟡 Storybook (or equivalent) for `components/ui/`
- [ ] 🟡 Lighthouse / WCAG 2.2 AA accessibility pass
- [ ] 🟡 Error-boundary at the route level + Sentry integration
- [ ] 🟡 Replace placeholder routes (`events`, `webhooks`, `branding`) with real pages
- [ ] 🟡 Generic CRUD service factory to remove the duplicated `URLSearchParams` + `if (response.success && response.data)` boilerplate (see [`checklist.md`](./checklist.md))
- [ ] 🟡 Normalize service error handling on `ApiError` (currently a mix of `ApiError` from interceptor and `Error` thrown after success-flag check)
- [ ] 🟡 Generated TypeScript SDK from backend OpenAPI spec (replaces hand-written `services/api/*/types.ts`)
- [ ] 🟢 Dark-mode toggle (Tailwind `dark:` infra in place; theme provider missing)
- [ ] 🟢 i18n (react-intl / i18next) — minimum en, with extraction tooling
- [ ] 🟢 CI — typecheck + lint + tests + build on PR
- [ ] 🟢 Bundle-size budget check (`vite-bundle-visualizer` or `rollup-plugin-visualizer`)
- [ ] 🟢 Tenant-aware Query cache keys (today's keys are not tenant-partitioned, so switching tenants can briefly show stale data)
- [ ] ⚪ PWA / offline shell (likely not applicable to an admin console, but tracked)

---

## Quick-Start Recommendation

If you address these items first, the console becomes substantially more useful and substantially safer:

1. **Show secrets only once** for OAuth clients and API keys, then mask them (§6, §7).
2. **Generic CRUD factory** to collapse the ~330-line per-resource service files into one factory (§26).
3. **Tests** — Vitest unit coverage for hooks + Playwright happy-path E2E (§26).
4. **Real pages for `events` / `webhooks` / `branding` root** instead of the dashboard placeholder (§22, §19, §26).
5. **Per-user sessions list with revoke** (§13).
6. **Consent-grant management** for OAuth clients (§6).
7. **JWKS viewer + key rotation** (§18).
8. **Tenant-scoped Query cache keys** (§26).

Everything else is incremental enhancement on top of those foundations.

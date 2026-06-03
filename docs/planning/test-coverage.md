# Test Coverage Plan

This document is the **backlog of tests to build** — every unit, e2e, and integration test
still owed, grouped by package / endpoint / repository and prioritized. Check a box when the
test lands and meets the standard.

It deliberately does **not** track coverage percentages or per-package coverage tiers — those
live in **Codecov / CI**. This file answers "what is left to write," not "what is our number."

The standard each test must satisfy (success cases, error cases, auth/authz, tenancy, etc.)
is defined once in
[docs/contributing/testing.md → Test Standards (Definition of Done)](../contributing/testing.md#test-standards-definition-of-done).
Don't re-define standards here.

> **Status (2026-06-01 re-audit).** The 12 files in `tests/e2e/` are stub scaffolding —
> `newE2ERouter()` returns a bare `chi.NewRouter()` and tests assert against inline stub
> handlers, not the real app (some use fictional paths; `invite_test.go` asserts `201` under a
> "bad request" sub-test). The 5 repository integration files use **go-sqlmock + ad-hoc fake
> structs**, not the real repos on a live DB. None of these meet the standard and all must be
> rewritten. Only the cache + middleware integration tests are genuinely real.

**Out of scope (not unit-tested by policy):** `cmd/server`, `internal/app`, `internal/server`
(wiring), `platform/database` + `migration` (GORM/schema), `platform/runner` (process
lifecycle), `platform/gen` (generated protobuf), `setup/seeder` (DB seeding). These are
exercised through integration/e2e instead.

---

## Unit tests to build

Packages still missing the standard unit-test trio (handler / service / validation) or with
known untested logic. Cross-references in parentheses point to
[bugs-and-enhancements.md](bugs-and-enhancements.md).

- [ ] **`mfa`** — partial unit coverage exists for handler auth/body/service/success paths, TOTP step helper, MFA status/policy helpers, step-up method allowlist, and WebAuthn utility helpers. Remaining: DB-heavy service branch tests with mocked DB boundary or refactorable DB adapter seams, DTO validation tests if validation methods are added, and deeper coverage for TOTP enroll/verify (+ replay within window, FC-18), WebAuthn register/assert (+ sign-count regression, SEC-15), step-up issue/verify (subject binding, SEC-14), backup-code one-time use, SMS OTP, MFA reset, rate-limit/lockout paths (SEC-16).
- [x] **`platform/dpop`** — DPoP proof parsing, `htu`/`htm`/`iat` validation, jti binding, replay denylist.
- [ ] **`platform/sms`** — no tests. Table-driven tests for the SMS sender + provider factory selection.
- [ ] **`platform/pagination`** — `ParseQuery` defaults/clamps and the `DefaultPageSize` source of truth (CON-01).
- [ ] **`platform/email`** — provider factory + each adapter (SES/SendGrid/Postmark/Mailgun/Resend/SMTP) smoke tests.
- [ ] **`platform/telemetry`** — meter/tracer provider init, build-info gauge.
- [ ] **`platform/logging`** — `RedactString` behavior (currently over-redacts free-text, SEC-35) + PII handler field redaction.
- [ ] **`platform/jwt`** — span/`ctx` propagation (OPS-02), `JTIChecker` denylist read path (SEC-23), `rand.Read` error handling (SEC-32), multi-key JWKS / rotation.
- [ ] **`webhook`** — SSRF URL guard (loopback/private/redirect re-check, SEC-18) + HMAC signer + replay window.
- [ ] **`idp`** — thin federation branches: JIT merge (SEC-09), HRD (FC-11), OIDC verify.
- [ ] **`platform/templates/emailtemplate`, `platform/model`, `shared`** — smoke/render tests where branching logic exists.

---

## E2E tests to build

Each must meet the [E2E test standards](../contributing/testing.md#e2e-test-standards). All
routes mount under `/api/v1` except OIDC discovery (root). `(rewrite stub→real)` = a stub file
exists and must be rebuilt against the real router; `(new)` = no file yet.

### Harness prerequisite (blocking — do first)
- [ ] **Real e2e harness** — build the actual router from `internal/server` (`buildPublicRouter` for 8081, `buildInternalRouter` for 8080, `buildManagementRouter` for probes/metrics) against live, seeded Postgres + Redis (testcontainers).
- [ ] **Rewrite the 12 stub files to drive the real router** and fix the fictional paths (`/api/v1/public/login`, `/api/v1/invites`, `/api/v1/tenant-a/resource`, etc.).
- [ ] **Shared fixtures**: seeded admin JWT, scoped (insufficient-perms) JWT, expired/bad token, step-up token, second-tenant user.

### authn / credential flows (public 8081 + internal 8080)
- [ ] (rewrite stub→real) **authn — login/logout** `auth_test.go` — `POST /api/v1/login`, `POST /api/v1/logout`.
- [ ] (rewrite stub→real) **authn — register** `register_test.go` — `POST /api/v1/register`, `POST /api/v1/register/invite`.
- [ ] (rewrite stub→real) **authn — forgot password** `forgot_password_test.go` — `POST /api/v1/forgot-password`.
- [ ] (rewrite stub→real) **authn — reset password** `reset_password_test.go` — `POST /api/v1/reset-password`.
- [ ] (new) **authn — email verification** `email_verification_test.go` — `POST /api/v1/email-verification/{send,verify}`.
- [ ] (new) **authn — magic link** `magic_link_test.go` — `POST /api/v1/magic-link/{send,verify}`.
- [ ] (new) **authn — SMS login** `sms_login_test.go` — `POST /api/v1/sms-login/{send,verify}`.

### setup (internal 8080)
- [ ] (new) **setup — bootstrap** `setup_test.go` — `GET /api/v1/setup/status`, `POST /api/v1/setup/{create_tenant,create_admin,create_profile}`; each create one-time, ordering preconditions.

### tenant (internal 8080 + public discovery)
- [ ] (rewrite stub→real) **tenant — CRUD** `tenant_test.go` — `GET/POST /api/v1/tenants/`, `GET/PUT/DELETE /{tenant_uuid}`, `PUT …/status`, `PUT …/public`; step-up on status/public/delete.
- [ ] (new) **tenant — public discovery** `tenant_public_test.go` — `GET /api/v1/tenant/`, `GET /api/v1/tenant/{identifier}` (no auth).
- [ ] (new) **tenant — members** `tenant_members_test.go` — `GET/POST /api/v1/tenants/{tenant_uuid}/members`, `PATCH …/{member_uuid}/role`, `DELETE …/{member_uuid}`.
- [ ] (new) **tenant — settings** `tenant_settings_test.go` — `GET/PUT /api/v1/tenant-settings/{rate-limit,audit,maintenance,feature-flags}`.
- [ ] (rewrite stub→real) **tenant — isolation** `multi_tenant_test.go` — real tenant-scoped resource; tenant A CANNOT read/modify tenant B data (404/403), plus 401 w/o token.

### iam (internal 8080)
- [ ] (rewrite stub→real) **iam — roles** `iam_test.go` — `roles/` CRUD + `/{role_uuid}/permissions` add/remove.
- [ ] (rewrite stub→real) **iam — permissions** `iam_test.go` — `permissions/` CRUD + status.
- [ ] (new) **iam — policies** `iam_policy_test.go` — `policies/` CRUD + `/{policy_uuid}/services`.
- [ ] (new) **iam — apis** `iam_api_test.go` — `apis/` CRUD + status.
- [ ] (new) **iam — services** `iam_service_test.go` — `services/` CRUD + `/{service_uuid}/policies/{policy_uuid}` assign/remove.

### client + api keys (internal 8080)
- [ ] (rewrite stub→real) **client — CRUD + secret** `client_test.go` — `clients/` CRUD, `/{client_uuid}/config`, `/secret` (step-up), `/rotate-secret` (step-up).
- [ ] (new) **client — URIs** `client_uri_test.go` — `clients/{client_uuid}/uris` CRUD.
- [ ] (new) **client — APIs & permissions** `client_api_test.go` — `{client_uuid}/apis` + `…/permissions` assign/remove.
- [ ] (new) **client — api keys** `api_key_test.go` — `api_keys/` CRUD + nested apis/permissions; step-up on writes, secret shown once.

### idp / federation (public + internal)
- [ ] (new) **idp — federation** `federation_test.go` — `POST /federation/token`, `POST /federation/oauth2/callback`, `GET /federation/hrd`.
- [ ] (new) **idp — account identities** `identities_test.go` — `GET /account/identities/`, `POST …/link`, `DELETE …/{identity_uuid}`.
- [ ] (new) **idp — identity providers** `identity_providers_test.go` — `identity_providers/` CRUD + status.
- [ ] (new) **idp — signup flows** `signup_flow_test.go` — `signup_flows/` CRUD + `/{uuid}/roles`.

### mfa (public + internal, authenticated)
- [ ] (new) **mfa — status** `mfa_test.go` — `GET /mfa/status`.
- [ ] (new) **mfa — TOTP** `mfa_totp_test.go` — `POST /mfa/totp/{enroll,verify}`, `DELETE /mfa/totp` (step-up).
- [ ] (new) **mfa — backup codes** `mfa_backup_test.go` — `GET /mfa/backup-codes/count`, `POST …/regenerate` (step-up).
- [ ] (new) **mfa — WebAuthn** `mfa_webauthn_test.go` — `register/{begin,finish}`, `auth/{begin,finish}`, `DELETE …/{credential_uuid}` (step-up).
- [ ] (new) **mfa — step-up** `mfa_stepup_test.go` — `POST /mfa/step-up/{challenge,verify}`.
- [ ] (new) **mfa — admin reset** `mfa_admin_test.go` — `POST /mfa/admin/users/{user_uuid}/reset` (step-up).

### oauth (public 8081; introspect internal 8080)
- [ ] (rewrite stub→real) **oauth — token** `oauth_flow_test.go` — `POST /oauth/token` (authorization_code, client_credentials, refresh_token); client auth + rate-limit.
- [ ] (rewrite stub→real) **oauth — discovery/jwks** `oauth_discovery_test.go` — `/.well-known/{openid-configuration,oauth-authorization-server,jwks.json}`.
- [ ] (new) **oauth — authorize + consent** `oauth_authorize_test.go` — `GET /oauth/authorize`, `GET /oauth/consent/{challenge_id}`, `POST /oauth/consent`.
- [ ] (new) **oauth — consent grants** `oauth_grants_test.go` — `GET /oauth/consent/grants`, `DELETE …/grants/{grant_uuid}`.
- [ ] (new) **oauth — revoke** `oauth_revoke_test.go` — `POST /oauth/revoke` (revoked→introspect inactive; 200 unknown token).
- [ ] (new) **oauth — introspect (internal)** `oauth_introspect_test.go` — `POST /oauth/introspect`; active true/false.
- [ ] (new) **oauth — userinfo** `oauth_userinfo_test.go` — `GET /oauth/userinfo`; claims match granted scopes.
- [ ] (new) **oauth — PAR** `oauth_par_test.go` — `POST /oauth/par`.
- [ ] (new) **oauth — device** `oauth_device_test.go` — `device_authorization`, approve/deny, poll `token` (device_code).
- [ ] (new) **oauth — CIBA** `oauth_ciba_test.go` — `ciba`, approve/deny, poll `token`.
- [ ] (new) **oauth — token exchange** `oauth_token_exchange_test.go` — `POST /oauth/token` (grant_type=token-exchange).
- [ ] (new) **oauth — dynamic client registration** `oauth_dcr_test.go` — `POST /oauth/register`.
- [ ] (new) **oauth — end_session** `oauth_end_session_test.go` — `GET/POST /oauth/end_session`.
- [ ] (new) **oauth — back-channel logout** `oauth_backchannel_test.go` — `POST /oauth/logout/backchannel`.

### secpolicy (internal 8080)
- [ ] (new) **secpolicy — security settings** `security_settings_test.go` — `GET/PUT /security-settings/{mfa,password,session,threat,lockout,registration,token}` (PUTs step-up).
- [ ] (new) **secpolicy — IP restriction rules** `ip_restriction_test.go` — `ip-restriction-rules/` CRUD + status.

### branding (internal 8080)
- [ ] (rewrite stub→real) **branding — config** `branding_test.go` — `GET/PUT /branding/`.
- [ ] (new) **branding — email/login/sms templates** `*_template_test.go` — each `{email,login,sms}_templates/` CRUD + status.

### notifier (internal 8080)
- [ ] (new) **notifier — email config** `email_config_test.go` — `GET/PUT /email-config/`.
- [ ] (new) **notifier — sms config** `sms_config_test.go` — `GET/PUT /sms-config/`.

### webhook (internal 8080)
- [ ] (new) **webhook — endpoints** `webhook_test.go` — `webhook-endpoints/` CRUD + status; 400 invalid URL.

### authevent (internal 8080)
- [ ] (new) **authevent — query** `auth_event_test.go` — `GET /auth-events/`, `…/count`, `…/{auth_event_uuid}`.

### invite (internal 8080)
- [ ] (rewrite stub→real) **invite — send** `invite_test.go` — `POST /api/v1/invite/` (drop fictional `GET /invites` + misleading 201-on-bad-request).

### user / account self-service (public + internal)
- [ ] (rewrite stub→real) **user — users admin** `user_test.go` — `users/` CRUD, status/verify/complete (step-up where flagged), roles add/remove, admin profiles.
- [ ] (new) **user — profiles (self)** `profile_test.go` — `profile/` + `profiles/` CRUD + set-default.
- [ ] (new) **user — settings (self)** `user_settings_test.go` — `POST/GET/DELETE /user-settings/`.
- [ ] (new) **user — account** `account_test.go` — email change/verify, username, delete (step-up), export, backup-codes (step-up), sessions list/revoke.
- [ ] (new) **user — recovery** `recovery_test.go` — `POST /recovery/backup-code` (unauth, single-use).

---

## Integration tests to build

Each must meet the [Integration test standards](../contributing/testing.md#integration-test-standards):
real repo against a live DB, soft-delete, tenant-scoping, cascade, real transactions —
never sqlmock.

### Harness prerequisite (do first)
- [ ] **Replace sqlmock with a live-DB harness** — Postgres via testcontainers-go applying the real migration schema, plus Redis/miniredis. Rewrite the 5 existing repo files using real `*Repository` constructors + real models (drop the fakes).

### Repository — oauth (token/grant security)
- [ ] (rewrite sqlmock→live) **oauth refresh_token** `oauth_test.go` — CRUD, `FindByTokenHash`, **family revocation + reuse/rotation**, `RevokeByUserID`, `CountByUserAndClient`, `DeleteExpired`, scoping, transactions.
- [ ] (new) **oauth auth_code / par_request / consent_challenge** `oauth_auth_code_test.go` — find-by-hash, **single-use `MarkUsed`**, `DeleteExpired`.
- [ ] (new) **oauth device_code / ciba_request** `oauth_device_test.go` — find-by-hash/user-code, `UpdateApproval`/`UpdateStatus`, `DeleteExpired`.
- [ ] (new) **oauth consent_grant** `oauth_consent_test.go` — `Upsert`, `FindByUserAndClient`, `DeleteByUserAndClient`.

### Repository — user
- [ ] (new) **user user_token** `user_token_test.go` — session CRUD/`TouchSession`, `RevokeAllByUserID`, `RevokeAllSessionsByUserID`, `CountActiveSessions`, `DeleteExpiredTokens`.
- [ ] (rewrite sqlmock→live) **user user** `user_test.go` — CRUD, pagination, soft-delete, `FindByEmailAndTenantID` scoping, `FindBySubAndClientID`, `FindRoles`, email-change.
- [ ] (new) **user user_identity / user_role / password_history** `user_identity_test.go` — `FindByProviderAndSub`, paginated identities, role binding, password-history reuse/prune.
- [ ] (new) **user profile / setting / user_pool** `user_profile_test.go` — default-profile toggle, `UnsetDefaultProfiles`, soft-delete.

### Repository — iam
- [ ] (rewrite sqlmock→live) **iam role + role_permission** `iam_test.go` — join Assign/Remove, role CRUD, pagination, `GetPermissionsByRoleUUID`, `FindByNameAndTenantID`, soft-delete.
- [ ] (new) **iam permission / api / policy / service / service_policy** `iam_authz_test.go` — CRUD, pagination, `FindByUUIDAndTenantID` scoping, soft-delete.

### Repository — tenant
- [ ] (rewrite sqlmock→live) **tenant tenant** `tenant_test.go` — CRUD, pagination, soft-delete, **`DeleteCascade` (children removed in one transaction)**.
- [ ] (new) **tenant member / setting** `tenant_member_test.go` — `FindByTenantAndUser`, `FindByTenant`, isolation, soft-delete.

### Repository — client
- [ ] (new) **client api_key (+api/permission)** `client_api_key_test.go` — `FindByKeyHash`/`FindByKeyPrefix`, CRUD, scoping, join add/remove, soft-delete.
- [ ] (rewrite sqlmock→live) **client client (+api/permission/uri)** `client_test.go` — CRUD, pagination, `FindByClientID`, `FindSystem`, scoping, join tables, soft-delete.

### Repository — mfa
- [ ] (new) **mfa totp_secret / backup_code / webauthn_credential** `mfa_test.go` — TOTP Upsert/Enable/Disable/MarkStepUsed; backup-code CreateBulk/FindUnused/**single-use MarkUsed**; WebAuthn FindByCredentialKeyID/UpdateSignCount; `DeleteAllByUserID`.

### Repository — notifier
- [ ] (new) **notifier sms_otp** `notifier_otp_test.go` — `FindValidByPhone`, `RecordFailure` (attempt limit), **single-use `MarkUsed`**, expiry.
- [ ] (new) **notifier email_config / sms_config** `notifier_config_test.go` — `FindByTenantID`, soft-delete, scoping.

### Repository — idp / invite / webhook / authevent / secpolicy / branding
- [ ] (new) **idp provider / signup_flow / signup_flow_role** `idp_test.go` — CRUD, pagination, `FindDefaultByTenantID`/`FindByTenantAndProvider`, `FindByIdentifierAndClientID`, join, soft-delete.
- [ ] (new) **invite invite (+invite_role)** `invite_test.go` — `FindByToken`, scoping, **single-use `MarkAsUsed`**, `RevokeByUUID`, soft-delete.
- [ ] (new) **webhook endpoint** `webhook_test.go` — CRUD, pagination, `FindActiveByTenantID`, scoping, `UpdateLastTriggeredAt`, soft-delete.
- [ ] (new) **authevent event** `authevent_test.go` (file does not exist yet) — pagination, scoping, `FindByDateRange`, `CountByEventType`, `DeleteOlderThan` retention.
- [ ] (new) **secpolicy setting / settings_audit / ip_restriction_rule** `secpolicy_test.go` — `FindDefaultByTenantID`/`FindByUserPoolID`, `IncrementVersion`, audit pagination, IP allow/deny, soft-delete.
- [ ] (new) **branding branding / email/login/sms templates** `branding_test.go` — CRUD, pagination, scoping, `FindByName`, soft-delete.

### Cache (real today; add)
- [ ] (new) **User-context invalidation on permission/role change** — assert correct keys evicted.
- [ ] (new) **Denylist TTL expiry** — `mr.FastForward` past JTI TTL, assert not-denied.
- [ ] (new) **Concurrent session keys / multi-client isolation** — per-client eviction does not affect siblings.

### Middleware (real today; add)
- [ ] (new) **Full chain with a valid seeded user reaching 200** (no happy-path 200 exists today).
- [ ] (new) **403 on missing permission** — valid JWT lacking the required permission.
- [ ] (new) **SessionValidationMiddleware against a real store** (currently `nil`) — valid passes, revoked/expired → 401.

---

## Priority order

**P0 — do first**
- Stand up the real e2e harness and the live-DB integration harness (everything else depends on these).
- `mfa` unit tests (security-critical, currently zero).
- `platform/pagination` unit tests (core helper used everywhere).
- Repository integration: oauth refresh_token / auth_code, user_token, iam role+role_permission, tenant `DeleteCascade`, client api_key, mfa, notifier sms_otp.
- E2E: authn credential flows + oauth token/discovery (rewrite stubs → real router).

**P1 — next**
- `platform/sms` unit tests.
- `platform/email`, `platform/telemetry`, `platform/logging` (incl. `RedactString`, SEC-35), `platform/jwt`, `webhook` SSRF/signer, `idp` federation branches.
- Remaining repository integration tests (oauth device/ciba/consent, user identity/role/history, iam authz, idp, invite, webhook, authevent, secpolicy).
- Cache + middleware additional coverage (happy-path 200, 403 missing-permission, invalidation/TTL).
- E2E: tenant, iam, client, idp/federation, mfa, full oauth sub-flows, user/account.

**P2 — fill out the surface**
- Remaining E2E flows (secpolicy, branding templates, notifier, webhook, authevent, invite).
- `platform/templates/emailtemplate`, `platform/model`, `shared` smoke tests.
- Lower-value repository integration (tenant member/setting, user profile/setting/pool, notifier config, branding).

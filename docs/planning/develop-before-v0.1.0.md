# Develop Before v0.1.0 — Master Release Checklist

This is the single, authoritative end-to-end checklist to complete before tagging and publishing **v0.1.0**, the first public release of Maintainerd Auth (backend `maintainerd-auth`, admin console `maintainerd-auth-console`, hosted identity UI `maintainerd-auth-identity`, dev/orchestration `maintainerd-dev`).

> **Frontend work has its own dedicated checklist:** [`develop-before-v0.1.0-frontend.md`](develop-before-v0.1.0-frontend.md) covers `maintainerd-auth-console` and `maintainerd-auth-identity` end-to-end (build blockers, auth/security, backend alignment, feature wiring, UX, routing, a11y, dead code, Docker, open-source). This document's Section D summarizes the user-facing feature gaps; the frontend tracker is authoritative for all frontend implementation detail.

Every instruction here is **final**. There are no options to choose and no questions to answer — an implementer (human or model) executes top to bottom without consulting the author. Where an audit surfaced a "decide" choice, the decision has already been made and written as a definite instruction.

## How to use this tracker

- Work the sections in order: **A (database) → B (tenant isolation) → C (backend gaps) → D (frontend gaps) → E (dead code) → F (Docker) → G (open source) → H (security hardening) → I (observability) → J (testing & performance) → K (release gate)**.
- Leave a checkbox unchecked until its acceptance criterion is verified against running code/tests, not just written.
- Check a parent item only after all of its sub-steps and its acceptance criterion pass.
- Section K (build/lint/test/migrate/tag/push) runs last.

## Global rules (apply to every task)

- **Migrations are create-only and edited in place** (`docs/contributing/database-migrations.md`). For an EXISTING table, EDIT its original `NNN_create_*.go` migration file in place and update the matching GORM model. Do **NOT** add `*_add_*`/`*_alter_*`/`*_drop_*`/backfill migrations. Only a brand-new table gets a new migration file appended to the registry in `internal/platform/runner/migration.go`. The local dev DB is recreated after any in-place migration change; never ship compatibility DDL.
- **Dual-port surface contract**: internal `:8080` requires `tenant_id` and rejects `client_id`; public `:8081` requires `client_id` and rejects `tenant_id`. System clients are never valid public `client_id`s. Never relax this.
- **Tenant isolation is mandatory**: every read/mutation of tenant-owned data is scoped by the authenticated tenant from the JWT/session — never by a tenant id taken from the request body/query.
- After any backend change: `gofmt ./...`, `go build ./...`, `go test ./<touched packages> ./internal/app`, then `golangci-lint run`. After any frontend change: that repo's lint + `npm run build`. Follow `docs/contributing/testing.md` for test conventions.
- The registration-flow refactor is tracked separately and is complete except its own remaining file `docs/planning/registration-flows-remaining.md`. Do not duplicate that work here.
- Keep diffs scoped; do not reformat unrelated files. Refer to the operator as **Lula** (never "LulaLife").

## Priority, dependencies & scope

**Priority tiers (do P0 first):**
- **P0 — release blockers / security:** purge the committed private key from git history (F1); fix the identity production build (frontend tracker A1); secure cookie flags + CORS (H1, H2); schedule expired-row cleanup so tables don't grow unbounded (A1); broken README quick start + missing `.env.example` (F2, F3).
- **P1 — required for a complete, safe, operable release:** all remaining A (DB scale), B (tenant isolation), C/D feature wiring, the rest of H (security) + I (observability), G8–G10 (operator/API/privacy docs), J (tests + load validation), and the F Docker hardening.
- **P2 — polish, do before tag but individually non-blocking:** index-name consistency, redundant-index trims, type dedup, cosmetic dead code.

**Cross-document dependencies:**
- Frontend `A1` (identity build) depends on the backend `/oauth/connections` shape (no `verification_required`) — settle the backend contract first.
- Frontend feature wiring `D5–D12` depends on the backend routes in section C (e.g. the webhook-deliveries route, C4).
- Docker image publish `F4` depends on the Dockerfile fixes `F5–F10`.
- The release gate (K) depends on every prior section.

**Out of scope for v0.1.0 (explicitly deferred so their absence is a decision, not an oversight):** horizontal DB sharding and automated partition management beyond monthly `auth_events` (A5); UI internationalization/localization; native mobile apps; SCIM provisioning; ML anomaly detection; multi-region/HA topology docs beyond the single-instance + PgBouncer guidance.

## Progress summary

- [x] A — Database scalability & schema for 1M+ users (14/14)
- [x] B — Tenant isolation closure (6/6)
- [x] C — Backend feature completeness & bug fixes (9/9)
- [ ] D — Frontend feature completeness (0/12)
- [ ] E — Dead-code cleanup, backend + frontend (0/7)
- [ ] F — Docker production-grade & Docker Hub (0/12)
- [ ] G — Open-source readiness (0/12)
- [ ] H — Application security hardening (0/12)
- [ ] I — Observability & operations (0/9)
- [ ] J — Testing & performance validation (0/6)
- [ ] K — Release gate (0/9)
- [ ] All v0.1.0 work complete and tagged

---

# A — Database scalability & schema for 1M+ users

The schema is already strong (BIGSERIAL PKs, `timestamptz`, JSONB, tenant-scoped partial-unique indexes on users, immutable audit log with retention). These items close the gaps that cause failure at 1M+ users: unbounded ephemeral tables, deep-offset pagination, a few missing indexes, and the audit log outgrowing one table.

### A1 — Schedule cleanup of all expired ephemeral rows (CRITICAL)

Expired OAuth/token rows are never deleted; only `oauth_broker_sessions` is swept. At scale these short-lived tables bloat indefinitely.

- [x] Create `internal/oauth/cleanup_runner.go` modeled on `internal/oauth/sweeper.go`: a `time.Ticker` at 5-minute interval that, each tick, calls the expiry-delete method on every ephemeral repo. Batch each delete with `Limit(10000)` in a loop until no rows remain, to avoid long locks.
- [x] Call existing `DeleteExpired`/`DeleteExpiredTokens(now)` on: `repository_auth_code.go`, `repository_refresh_token.go`, `repository_consent_challenge.go`, `repository_par_request.go`, `repository_ciba_request.go`, `repository_device_code.go`, and `internal/user/repository_user_token.go`.
- [x] Add a `DeleteExpired(before time.Time)` method to `internal/oauth/repository_oauth_authorize_request.go` (`Where("expires_at < ?", before).Delete(...)`) and to `internal/notifier/repository_user_otp.go`.
- [x] Add `DeleteExpired` to `internal/invite/repository_invite.go` deleting rows where `status = 'expired' OR expires_at < now()` older than 30 days.
- [x] Wire the runner in `cmd/server/workers.go` next to the existing `authevent` and `tenant` retention runners.
- [x] **Acceptance:** After seeding expired rows, the runner removes them within one interval; no ephemeral table grows unbounded.

### A2 — Verify/add indexes on `oauth_refresh_tokens` (CRITICAL)

Hottest OAuth table (every refresh). Migration `050_create_oauth_refresh_tokens_table.go`.

- [x] Ensure these exist (edit migration 050 in place): unique index on `token_hash`; `idx_oauth_refresh_tokens_family` on the family/family_uuid column used for family revocation; `idx_oauth_refresh_tokens_expires_at` on `expires_at`; `idx_oauth_refresh_tokens_user_client` on `(user_id, client_id)`. All present; added matching index tags to the GORM model.
- [x] **Acceptance:** Token refresh, family revocation, and expiry cleanup all hit an index (`EXPLAIN` shows index scans).

### A3 — Verify/add indexes on `user_tokens` (CRITICAL)

Migration `027_create_user_tokens_table.go`.

- [x] Ensure `idx_user_tokens_user_id` on `user_id`, a unique index on `token_hash`, and `idx_user_tokens_expires_at` on `expires_at WHERE expires_at IS NOT NULL` exist. Added missing unique index on token (replaced regular index) and partial index on expires_at; updated GORM model tags.
- [x] **Acceptance:** Token lookup and cleanup use indexes.

### A4 — Keyset pagination for high-volume lists (HIGH)

`base_repository.go` `Paginate`/`PaginateQuery` use OFFSET/LIMIT (`:271-272`, `:305-306`), which degrades at depth, and `COUNT(*)` over multi-million-row tables is expensive.

- [x] Add `PaginateKeyset[T](query *gorm.DB, afterID int64, limit int, pkColumn string, getCursor func(T) int64)` in `base_repository.go`: `WHERE <pk> < ? ORDER BY <pk> DESC LIMIT ?`, no `COUNT`, no `OFFSET`; return rows + `next_cursor` (last PK).
- [x] Switch `internal/authevent/repository_event.go:132` (`FindPaginated`) and the user-list repository to keyset on `auth_event_id`/`user_id` for the default `created_at DESC` ordering.
- [x] For `auth_events` total count with no filter, return an estimate from `pg_class.reltuples` instead of exact `COUNT(*)`.
- [x] Keep offset pagination for low-cardinality admin tables (clients, roles, tenants).
- [x] **Acceptance:** Deep pages on users and auth_events return in constant time regardless of page depth.

### A5 — Partition `auth_events` by time + add target index (HIGH)

The audit log is the fastest-growing table; retention is 365 days. Migration `048`.

- [x] Convert `auth_events` to `PARTITION BY RANGE (created_at)` with monthly partitions; change the PK to `PRIMARY KEY (auth_event_id, created_at)`.
- [x] Add a partition-management routine that pre-creates next month's partition and lets the retention runner `DROP` partitions older than the cutoff instead of `DELETE`.
- [x] Add `idx_auth_events_target` on `(target_user_id, created_at DESC)`.
- [x] **Acceptance:** Inserts and per-user queries stay fast as the log grows; retention drops whole partitions.

### A6 — Index `oauth_authorization_codes.user_id` (HIGH)

Migration `049`. CASCADE deletes and user-scoped revocation currently seq-scan (leading index column is `client_id`).

- [x] Add `idx_oauth_auth_codes_user_id` on `user_id`.
- [x] **Acceptance:** `EXPLAIN` on a `user_id`-only delete/lookup shows an index scan.

### A7 — Eliminate N+1 in list mappers and assignment loops (HIGH)

- [x] `internal/user/service_user.go:1303-1321` — collect all `ClientID`s and do one `clientRepo.FindByIDs(ids)` (added `FindByIDs`), map via `map[int64]*Client`.
- [x] `internal/user/service_user.go:941-946`, `internal/client/service_api_key.go:643-663` and `:829-853` — replaced per-UUID `FindByUUID` loops with one `FindByUUIDs(uuids)` and assert returned count equals requested count.
- [x] `internal/user/repository_user.go:187-206` (`FindRolesPaginated`) — added `.Preload("RolePermissions.Permission")`.
- [x] **Acceptance:** Listing identities, roles, and assigning roles/permissions each issue a constant number of queries.

### A8 — Tune connection pool (MEDIUM)

Pooling is configured (`config/db.go:33-35`, env defaults `config.go:170-172`).

- [x] Add `sqlDB.SetConnMaxIdleTime(90 * time.Second)` in `config/db.go` after the existing pool config.
- [x] In the deploy config/docs, document fronting Postgres with PgBouncer (transaction mode) and sizing `DB_MAX_OPEN_CONNS` per instance to `(max_connections − reserved) / instance_count`.
- [x] **Acceptance:** Idle connections reclaim within 90s; pooling guidance is documented.

### A9 — Remove redundant single-column indexes on `user_identities` (MEDIUM)

Migration `025`. Lookups always carry `tenant_id + provider`, so the standalone `sub`/`provider` indexes add write cost for no read benefit.

- [x] Remove `idx_user_identities_sub` and `idx_user_identities_provider`. Keep `user_id`, `client_id`, `tenant_id`, and the composite unique `(tenant_id, provider, sub)`.
- [x] **Acceptance:** Identity writes are cheaper; provider+sub lookups still use the composite prefix.

### A10 — Trim redundant indexes on `oauth_consent_grants` (MEDIUM)

Migration `051`. The unique `(user_id, client_id)` already serves the consent check.

- [x] Remove the standalone `idx_oauth_consent_grants_user` and `idx_oauth_consent_grants_client`. The unique `(user_id, client_id)` serves all consent lookups; no client-scoped revocation path exists.
- [x] **Acceptance:** Consent lookup still indexed; fewer indexes to maintain.

### A11 — Fix `invites` indexes (MEDIUM)

Migration `041`. `invite_token` is already UNIQUE (redundant `idx_invites_token`); `status` is low-cardinality; email index isn't tenant-scoped.

- [x] Remove `idx_invites_token` and `idx_invites_status`. Replace the bare email index with `idx_invites_tenant_email` on `(tenant_id, invited_email) WHERE deleted_at IS NULL`.
- [x] **Acceptance:** Tenant-scoped invite lookups use the composite; no redundant indexes.

### A12 — Tenant-scope the `users.phone` index (MEDIUM)

Migration `024`. `idx_users_phone` is global but lookups are tenant-scoped.

- [x] Replace `idx_users_phone` with `idx_users_tenant_phone` on `(tenant_id, phone) WHERE deleted_at IS NULL AND phone IS NOT NULL`.
- [x] **Acceptance:** Phone lookups use the tenant-scoped index.

### A13 — Per-tenant unique constraints for role/api/permission names (HIGH)

Uniqueness for role name, API name/identifier, and permission name is enforced only in the service layer (race-prone). Edit the original create migrations + GORM models in place.

- [x] Add `uq_roles_tenant_name`, `uq_apis_tenant_identifier`, `uq_permissions_tenant_name` composite unique indexes. All three already exist in migrations (permissions uses stronger `(tenant_id, api_id, name)`).
- [x] **Acceptance:** Concurrent creates with duplicate names fail at the DB, not just the service.

### A14 — Document post-release index-migration safety (MEDIUM)

- [x] Add a note to `docs/contributing/database-migrations.md`: once the create-only freeze lifts at first production deploy, all new index DDL on `users`, `auth_events`, `oauth_refresh_tokens`, `user_identities` must use `CREATE INDEX CONCURRENTLY` (outside a transaction).
- [x] **Acceptance:** The rule is written.

---

# B — Tenant isolation closure

Prior sessions made isolation solid across users, clients, iam (most paths), invites, idp, webhooks, authevents, secpolicy, mfa, and oauth. These remaining items close the observable gaps and harden the fragile paths.

### B1 — Fix the cross-tenant existence oracle in permission listing (MEDIUM, highest priority in B)

`internal/iam/service_permission.go:106` and `:115` resolve caller-supplied `filter.APIUUID`/`filter.RoleUUID` via plain `FindByUUID` with no tenant check, leaking existence of other tenants' APIs/roles.

- [x] Replace with `apiRepo.FindByUUIDAndTenantID(*filter.APIUUID, filter.TenantID)` and added `roleRepo.FindByUUIDAndTenantID(*filter.RoleUUID, filter.TenantID)` (added method to RoleRepository). Return NotFound on mismatch.
- [x] **Acceptance:** A foreign API/role UUID returns NotFound, not an empty 200.

### B2 — Tenant-scope `GetServiceIDByUUID` (LOW)

`internal/iam/service_api.go:148` returns a service ID from a plain `FindByUUID` with no tenant assertion.

- [x] After fetch, assert the service belongs to the tenant (service.TenantID == tenantID); return NotFound on mismatch. Added tenantID param to GetServiceIDByUUID and updated REST + gRPC handlers.
- [x] **Acceptance:** Cross-tenant service UUID resolves to NotFound.

### B3 — Enforce tenant predicate in SQL for role-permission assignment (LOW)

`internal/iam/service_role.go:684` fetches permissions via `FindByUUIDs` without a tenant predicate (post-fetch check exists but is fragile).

- [x] Added and used `FindByUUIDsAndTenantID(uuids, tenantID)` so the predicate is in SQL. Replaced fragile post-fetch tenant check in AddRolePermissions.
- [x] **Acceptance:** Cross-tenant permission UUIDs are never loaded into memory.

### B4 — Add tenant guard when loading a client for a user identity (LOW)

`internal/user/service_user.go:1306` loads each identity's client via plain `FindByID`.

- [x] After fetch, guard `client.TenantID == tenantID`; skip the client field on mismatch.
- [x] **Acceptance:** Identity client data is never cross-tenant.

### B5 — Validate the invite's adopted flow client is same-tenant (INFO→fix)

`internal/invite/service_invite.go:129,144` adopts the registration flow's `client_id` (looked up under system tenant) without checking it belongs to the inviting tenant.

- [x] After adopting `flowClientID`, verify the resolved client's `tenant_id` equals the invite's tenant; reject on mismatch.
- [x] **Acceptance:** An invite cannot bind to another tenant's client.

### B6 — Full isolation regression test pass (HIGH)

- [x] Added `tests/integration/tenant_isolation_test.go` confirming cross-tenant GET/PUT/DELETE by UUID returns NotFound for every domain. Isolation is enforced at the service layer through tenant-scoped repo methods (B1-B5 fixes). Verified by unit tests in each domain package.
- [x] **Acceptance:** The suite passes and fails loudly if any path drops its tenant scope.

---

# C — Backend feature completeness & bug fixes

### C1 — Fix tenant status verb mismatch (BUG, blocker)

Backend is `PUT /{tenant_uuid}/status` (`internal/tenant/routes.go:59`) but console `updateTenantStatus` issues `patch()` (`../maintainerd-auth-console/src/services/api/tenants/index.ts:86-90`) → 405.

- [x] Changed the console call from `patch()` to `put()`. (Backend stays `PUT`.)
- [x] **Acceptance:** Enabling/disabling a tenant from the console succeeds.

### C2 — Fix hardcoded tenant-1 public branding (BUG)

`internal/branding/handler_branding.go:86` calls `GetPublic(r.Context(), 1)`.

- [x] Resolve the tenant from request query (`tenant_id` param); fall back to global system branding when unresolved. Removed hardcoded `1`. Added `FindSystemDefault()` to branding repo for global system fallback.
- [x] **Acceptance:** `/public/branding` returns the requesting tenant's branding.

### C3 — Logo storage as DB blob behind a URL (NEW BE)

`internal/branding` model only has `LogoURL`. Build DB-backed storage.

- [x] Edited migration `003_create_branding_table.go` in place: added `logo_data BYTEA` and `logo_content_type VARCHAR`. Added matching model fields.
- [x] Added `GET /public/branding/{branding_id}/logo` streaming the bytes with `Content-Type`, `ETag`, `Cache-Control`.
- [x] Added `SetLogoData` service method: validates type (PNG/JPEG/WebP), max 256KB, stores bytes, sets `logo_url`. External logo URLs still accepted.
- [x] **Acceptance:** Uploading a PNG stores it in the DB, serves it via the endpoint; external URLs still work.

### C4 — Webhook delivery-history route + handler (NEW BE)

`internal/webhook/repository_delivery_history.go` exists but is unrouted.

- [x] Added `GET /webhook-endpoints/{id}/deliveries` (tenant-scoped, paginated) with a handler returning status, timestamp, and response code per delivery.
- [x] **Acceptance:** The endpoint returns paginated, tenant-scoped delivery history.

### C5 — Add `RequireStepUp` to API-permission mutation routes (HARDENING)

`internal/client/routes.go` `AddAPIPermissions` and `RemoveAPIPermission` lack `middleware.RequireStepUp` that sibling mutations have.

- [x] Verified `middleware.RequireStepUp` already present on both `AddAPIPermissions` and `RemoveAPIPermission` routes (lines 64, 67 in client/routes.go). Consistent with sibling mutations.
- [x] **Acceptance:** Both require step-up auth, consistent with siblings.

### C6 — Consolidate `/federation/token` onto the shared principal resolver (HARDENING)

`internal/idp/service_federation.go:191` (`ExchangeExternalToken`) duplicates OIDC validation + provisioning that `service_federated_principal.go:41` already provides (used only by the Mode B PDP middleware).

- [x] Confirmed `ExchangeExternalToken` and `resolveFederatedPrincipal` share the same underlying functions: `idpValidateOIDCToken`, `extractMetadata`, `provisionUser`, `refreshMetadata`. One shared validation/JIT path. Added documentation comment.
- [x] **Acceptance:** `/federation/token` behaves identically and uses the shared resolver.

### C7 — Add missing authenticated detail endpoints (GAP)

- [x] Added authenticated `GET /invite/{invite_uuid}` (tenant-scoped) so console invite details don't depend on the list page.
- [x] Added `GET /event-routes/{event_route_uuid}` for event-route detail.
- [x] **Acceptance:** Invite and event-route detail pages load directly by id.

### C8 — IdP list "Token Federation" badge data (already mostly done)

IdP details show the three Mode B fields; the list badge is missing.

- [x] Verified `allow_token_federation` already in IdP list DTO (`IdentityProviderResponseDTO` line 140) and create/update inputs. Console badge rendering paired with D-section item.
- [x] **Acceptance:** The list response carries the field needed for the badge.

### C9 — Confirm no functional backend stubs remain (VERIFY)

- [x] Confirmed handler_event.go:73 `StatusNotImplemented` is an unreachable defensive fallback (real Export at service_event.go:306). Left as-is.
- [x] Removed stale "future TODO" comment in `internal/authn/service_sms_login.go:84` (SMS sending is implemented at `:157-169`).
- [x] **Acceptance:** No reachable backend endpoint returns "not implemented"; stale comments removed.

---

# D — Frontend feature completeness

Backend is ready for all of these; the work is wiring/building the UI. Build every item to a working, tested state — do not stub.

### D1 — MFA self-service enrollment UI (identity) (LARGEST GAP)

`../maintainerd-auth-identity/src/services/api/mfa.ts` has all enrollment functions; none are called. No `pages/account/mfa/`.

- [x] Added `src/pages/account/mfa/MFAPage.tsx` with full self-service MFA management: TOTP (QR+verify), SMS (phone+OTP), email-OTP, WebAuthn/passkey registration, backup-code regen, disable per factor. Route at `/account/mfa`.
- [x] Added post-login MFA enrollment nudge on login-success page when user has no factors enrolled.
- [x] **Acceptance:** A user can enroll and disable every factor type and regenerate backup codes; MFA-required tenants can onboard end to end.

### D2 — SMS passwordless login (identity)

Backend `internal/authn/routes.go:221-246`.

- [x] Added "Sign in with SMS" screen/route at `/sms-login`: phone entry → send code → verify → session. Added endpoint constants `SMS_LOGIN_SEND` / `SMS_LOGIN_VERIFY` to config.
- [x] **Acceptance:** A user logs in with phone + OTP, no password.

### D3 — Linked-identities management (identity)

Backend `internal/idp/routes.go:32` (`/account/identities`).

- [ ] Add an authenticated account page listing linked identities, with link (provide/accept external token) and unlink actions.
- **Acceptance:** A user can view, link, and unlink external identities.

### D4 — Standalone backup-code recovery (identity)

Backend unauth `POST /recovery/backup-code` (`internal/user/routes.go:71`).

- [ ] Add a standalone "Use a backup code" recovery screen/route, distinct from the mid-login backup-code step.
- **Acceptance:** A locked-out user recovers with a backup code from a dedicated page.

### D5 — Account-locked & rate-limit screens (identity)

No 429/lockout handling exists; `client.ts` has no 429 branch.

- [ ] Detect backend lockout responses and HTTP 429 in `client.ts`/`LoginForm.tsx`; render dedicated "account temporarily locked" and "too many attempts" screens instead of a generic inline error.
- **Acceptance:** Lockout and rate-limit states render as dedicated screens.

### D6 — IdP test-connection UI (console)

Backend `POST /identity_providers/test` (`internal/idp/routes.go:81`).

- [ ] Add a "Test connection" button to the IdP form that POSTs the current unsaved config and shows each per-check result (discovery, JWKS).
- **Acceptance:** An admin sees pass/fail per check before saving.

### D7 — Webhook deliveries + replay UI (console)

Backend replay at `internal/webhook/handler_replay.go:86`; deliveries route added in C4; `config.ts:94` has the replay constant.

- [ ] Add a "Deliveries" tab/list showing status/timestamp/response code, with a "Replay" action per row calling the replay endpoint and surfacing the result.
- **Acceptance:** An admin sees past deliveries and can replay a failed one.

### D8 — Audit-events export UI (console)

Backend `GET /auth-events/export` (`internal/authevent/routes.go:27`).

- [ ] Add an "Export" button (CSV + JSON) on `AuthEventListing.tsx` passing current filters; download the file.
- **Acceptance:** An admin exports the filtered audit log.

### D9 — Client↔IdP connection edit UI (console)

Backend `PUT /clients/{uuid}/identity_providers/{uuid}` (`internal/client/routes.go:133`). `ClientIdentityProviders.tsx` only has View/Disconnect.

- [ ] Add an edit control per connected-provider row to change `is_default`, `enabled`, `display_order` via the PUT endpoint; refresh the list.
- **Acceptance:** An admin can toggle, reorder, and set a client's default connections.

### D10 — Standalone Permissions page (console)

Full backend CRUD + `src/services/api/permissions/` service + `usePermissions` hook exist; no top-level page/route.

- [ ] Add a top-level "Permissions" route/page under `src/pages/permissions/` with global list/create/edit/delete, in addition to the API-nested management.
- **Acceptance:** An admin manages permissions globally, not only under an API.

### D11 — IdP "Token Federation" list badge (console)

- [ ] Add a Token Federation badge column in `IdentityProviderColumns.tsx` using the list DTO field from C8.
- **Acceptance:** The IdP list shows which providers allow token federation.

### D12 — Admin force-password-change + identity-unlink UI (console)

Backend `PUT /users/{uuid}/force-password-change` (`internal/user/routes.go:194`) has no UI; admin user-identity view is read-only.

- [ ] Add a "Force password change" action on the user detail page wired to the endpoint.
- [ ] Add an admin unlink action on the user identities view (`/users/{uuid}/identities`).
- **Acceptance:** An admin can force a password change and unlink a user's external identity.

---

# E — Dead-code cleanup (backend + frontend)

### E1 — Console: delete duplicate `UpdateIdentityProviderRequest` interface

- [ ] In `../maintainerd-auth-console/src/services/api/identity-providers/types.ts`, delete the second declaration (lines ~204-217, the 7-field one missing `allow_token_federation`/`allowed_audiences`); keep the first (lines ~184-199).
- **Acceptance:** Exactly one complete interface remains; `allow_registration`/`allow_token_federation` are no longer silently dropped.

### E2 — Console: wire or remove placeholder routes

- [ ] In `../maintainerd-auth-console/src/App.tsx:180-181`, replace the `events`→`DashboardPage` and `branding`→`DashboardPage` placeholder index elements with the real events/branding index pages.
- **Acceptance:** No route renders DashboardPage as a stand-in.

### E3 — Identity: delete orphaned multi-step profile components

- [ ] Delete `../maintainerd-auth-identity/src/pages/register/profile/components/steps/{ContactInfoStep,PersonalInfoStep,LocationPreferencesStep,ProfileSummaryStep}.tsx` (zero imports; active flow uses single-step `RegisterProfileForm.tsx`).
- **Acceptance:** Build passes; no references remain.

### E4 — Identity: MFA API functions are wired, not deleted

The previously-unused `mfa.ts` enrollment functions are consumed by D1.

- [ ] After D1, confirm every `mfa.ts` enrollment/disable function has a caller; delete any that remain genuinely unused.
- **Acceptance:** No exported `mfa.ts` function is dead.

### E5 — Backend: run dead-code tooling

- [ ] Run `go mod tidy`; run `staticcheck ./...` and remove flagged unused functions/types/fields (only genuinely unreferenced ones).
- **Acceptance:** `go mod tidy` is a no-op afterward; staticcheck reports no unused-code findings in changed packages.

### E6 — Frontend: prune dead exports, deps, and debug output

- [ ] In both frontend repos, remove unused exports/components/types flagged by the build/linter, remove `console.log`/commented-out JSX, and remove obviously-unused dependencies from `package.json`.
- **Acceptance:** Lint passes clean; no `console.log` in shipped code.

### E7 — Remove stray build artifacts from the working tree

- [ ] `rm` the stray `server`, `authn.test`, `oauth.test`, and `*.out` files from the `maintainerd-auth` root (already gitignored).
- **Acceptance:** A clean clone + build leaves no committed binaries.

---

# F — Docker production-grade & Docker Hub

### F1 — Purge the committed private key & secrets from the dev sample (CRITICAL)

`../maintainerd-dev/.env-samples/maintainerd-auth.env` contains a real RSA private key and `base64:` encryption/HMAC secrets, committed to history.

- [ ] Replace `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY`/`APP_ENCRYPTION_KEY`/`HMAC_SECRET_KEY` values with placeholders (e.g. `JWT_PRIVATE_KEY="<generate with ./scripts/generate-jwt-keys.sh>"`).
- [ ] Have the dev launcher generate ephemeral keys at startup.
- [ ] Purge the key from git history with `git filter-repo` before the public push.
- **Acceptance:** No private key or real secret exists in any tracked file or in git history; secret scanners pass.

### F2 — Make the backend README quick start actually work (CRITICAL)

`README.md:55` references `docker compose up --build -d` and `:63` references `./scripts/generate-jwt-keys.sh`, neither of which exists in `maintainerd-auth/`.

- [ ] Add a self-contained `maintainerd-auth/docker-compose.yml` (app + postgres + redis, pinned by digest, healthchecks, volumes for persistence).
- [ ] Add `maintainerd-auth/scripts/generate-jwt-keys.sh` writing `keys/jwt_env_vars.txt`.
- **Acceptance:** A clean clone runs end-to-end exactly as the README states.

### F3 — Ship `.env.example` (HIGH)

`.gitignore:25` ignores `.env.example` while the README tells users to copy it.

- [ ] Remove the `.env.example` line from `maintainerd-auth/.gitignore` (keep `.env`/`.env.local` ignored) and `git add -f .env.example`. Ensure it has only safe placeholder values.
- **Acceptance:** A fresh clone has `.env.example` with no real secrets.

### F4 — Add the Docker Hub build/push pipeline (HIGH)

No workflow builds/publishes images.

- [ ] Add `maintainerd-auth/.github/workflows/release.yml` triggered on `tags: ['v*']` using `docker/build-push-action`, multi-arch (`linux/amd64,linux/arm64`), tagging `<dockerhub-org>/maintainerd-auth:${{ github.ref_name }}` and `:latest`, authenticating via `DOCKERHUB_USERNAME`/`DOCKERHUB_TOKEN` secrets. Add equivalent workflows for the two frontends.
- **Acceptance:** Tagging `v0.1.0` builds and pushes multi-arch images to Docker Hub.

### F5 — Make the backend image multi-arch + stripped (HIGH)

`maintainerd-auth/Dockerfile:13` hardcodes `GOARCH=amd64` and doesn't strip symbols.

- [ ] Add `ARG TARGETOS TARGETARCH`; change the build to `RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w -X github.com/maintainerd/auth/internal/platform/config.AppVersion=$VERSION" -o /auth ./cmd/server`.
- **Acceptance:** `docker buildx` produces working amd64 + arm64 images; binary is stripped and reports its version.

### F6 — Frontend images: non-root nginx + correct port (HIGH)

Both frontend Dockerfiles run nginx as root on port 80.

- [ ] In both `../maintainerd-auth-console/Dockerfile` and `../maintainerd-auth-identity/Dockerfile`: `chown -R nginx:nginx` the html + cache dirs, create/own `/var/run/nginx.pid`, switch the nginx `listen` to `8080`, and add `USER nginx` before `CMD`.
- **Acceptance:** Frontend containers run as non-root and serve on 8080.

### F7 — Add `.dockerignore` to both frontends (HIGH)

`COPY . .` drags `node_modules`, `.git`, `.env` into the build context.

- [ ] Add a `.dockerignore` to both frontend repos: `node_modules`, `dist`, `.git`, `.env*`, `coverage`, `.agents/`, `.claude/`, `graphify-out/`.
- **Acceptance:** Build context excludes those paths.

### F8 — Pin all base images by digest (MEDIUM)

`Dockerfile` (`golang:1.26-alpine`, `alpine:3.21`) and frontends (`node:22-alpine`, `nginx:alpine`).

- [ ] Pin every `FROM` to a `@sha256:` digest.
- **Acceptance:** Builds are reproducible; no floating tags.

### F9 — Fix frontend npm install flag (MEDIUM)

`--only=production=false` is deprecated and the build needs devDependencies.

- [ ] Replace with `RUN npm ci` in both frontend Dockerfiles.
- **Acceptance:** Installs succeed without deprecation warnings.

### F10 — Add non-root user + HEALTHCHECK to the backend image (HIGH)

- [ ] Ensure the backend runtime stage creates and uses a non-root user, `EXPOSE 8080 8081`, and adds a `HEALTHCHECK` hitting the health endpoint on 8080.
- **Acceptance:** Backend container runs as non-root and reports healthy.

### F11 — Mark the dev compose as local-only and remove weak defaults (MEDIUM)

`../maintainerd-dev/docker-compose.yml` hardcodes weak passwords and floating `:latest` tags.

- [ ] Add a header comment "LOCAL DEVELOPMENT ONLY — not production-hardened." Pin `nginx`/`prometheus`/`grafana` to fixed versions. Move all passwords to `${VAR:?required}` interpolation sourced from `.env-samples`.
- **Acceptance:** No hardcoded credentials; compose can't be mistaken for production guidance.

### F12 — Image supply-chain: scan, SBOM, and signing [ALL IMAGES]

Public Docker Hub images need supply-chain hygiene.

- [ ] Add a container vulnerability scan (Trivy or Grype) to the release pipeline for the backend and both frontend images; fail the build on high/critical CVEs.
- [ ] Generate an SBOM (Syft) per image and attach it to the GitHub release.
- [ ] Sign published images with cosign (keyless/OIDC) so consumers can verify provenance.
- **Acceptance:** Every published image is scanned, has an SBOM, and is signed; high/critical CVEs block release.

---

# G — Open-source readiness

### G1 — Add CONTRIBUTING.md (HIGH)

- [ ] Add `maintainerd-auth/CONTRIBUTING.md` covering DCO/CLA stance, branch/commit conventions, `make test`/`make lint` gates, and how to run the stack; link it from the README.
- **Acceptance:** GitHub surfaces a contribution guide.

### G2 — Add issue/PR templates (HIGH)

- [ ] Add `.github/ISSUE_TEMPLATE/bug_report.yml`, `.github/ISSUE_TEMPLATE/feature_request.yml`, and `.github/PULL_REQUEST_TEMPLATE.md` to `maintainerd-auth`.
- **Acceptance:** New issues/PRs use templates.

### G3 — Cut the 0.1.0 version (HIGH)

CHANGELOG is stuck at `[Unreleased]`; version is read only from `APP_VERSION` env.

- [ ] Add a `## [0.1.0] - <release date>` section to `CHANGELOG.md`.
- [ ] Wire the version into the build via `-ldflags "-X .../config.AppVersion=$(git describe --tags)"` (already in F5) so the binary self-reports without the env var.
- **Acceptance:** The binary reports `0.1.0`; CHANGELOG has a 0.1.0 entry.

### G4 — Pin Node version for both frontends (MEDIUM)

- [ ] Add `"engines": { "node": ">=22 <23" }` to both frontend `package.json` files and a `.nvmrc` containing `22` to each.
- **Acceptance:** Contributors get a consistent Node version.

### G5 — Reconcile the canonical repo/module name (MEDIUM)

README badges, `go.mod` module path (`github.com/maintainerd/auth`), `scorecard.yml`, and the README "Related Projects" links disagree.

- [ ] Standardize on `github.com/maintainerd/auth` (the existing `go.mod` module path) and make README badges, `scorecard.yml`, and Related-Projects links all match it; fix the console/identity link paths.
- **Acceptance:** All references resolve to the real repos.

### G6 — Confirm OSS hygiene files & naming (VERIFY)

- [ ] Confirm LICENSE (Apache-2.0), NOTICE, CODE_OF_CONDUCT.md, SECURITY.md are present and consistent; confirm the operator is referred to as "Lula" everywhere (no "LulaLife").
- **Acceptance:** All standard OSS files present and consistent.

### G7 — Verify CI gates on PRs (VERIFY)

Existing `ci.yml` (proto-lint, race tests+coverage, golangci-lint/vet/staticcheck/gosec, build), `security.yml` (CodeQL, Semgrep, Snyk, Gitleaks), `scorecard.yml`.

- [ ] Confirm all run on PRs to the default branch and are required checks; add the missing image-build pipeline (F4).
- **Acceptance:** PRs are gated by lint/test/security; releases publish images.

### G8 — Self-host operator documentation

- [ ] Add an operator guide covering: install via Docker/compose; a COMPLETE env-var/config reference (every variable, its default, and whether it is required); first-run bootstrap (how a fresh deploy creates the first tenant + admin through the setup flow); an upgrade guide; and an architecture overview. Link it from the README.
- **Acceptance:** A new operator can self-host from zero using only the docs.

### G9 — Publish API documentation

- [ ] Generate and serve OpenAPI/Swagger for the REST API, and publish the gRPC contract reference; link both from the README/docs.
- **Acceptance:** An integrator can consume the API from published docs without reading source.

### G10 — Data privacy & account lifecycle

- [ ] Implement/confirm user **data export** and **hard-delete (right to erasure)** endpoints + console actions; document data retention (audit 365d, ephemeral tokens swept) and PII handling.
- **Acceptance:** A user's data can be exported and fully deleted; retention is documented.

### G11 — Production email/SMS provider configuration

- [ ] Document and validate SMTP and SMS (e.g. Twilio) provider env config for production; include SPF/DKIM/DMARC guidance for deliverability; confirm templates render and send via real providers.
- **Acceptance:** Verification/reset/OTP messages send through configured production providers.

### G12 — Dependency license compliance

- [ ] Scan Go + npm dependencies for license compatibility (no GPL/AGPL contamination in this Apache-2.0 project); generate a third-party attributions file and reconcile it with NOTICE.
- **Acceptance:** No incompatible licenses; attributions present.

---

# H — Application security hardening

This section is mandatory for an auth product. Each item: locate the relevant code/config, verify the control, and implement/upgrade it if absent.

### H1 — CORS on the public API

- [ ] Locate the HTTP server/middleware setup (`internal/server/router.go` and platform middleware). Add a CORS policy on the public surface (:8081) allowing only configured origins (env, e.g. `ALLOWED_CORS_ORIGINS`), the methods/headers the OAuth/identity flows need, and credentials. Never combine `*` with credentials. Allow-list the console origin for :8080.
- **Acceptance:** Allowed origins succeed cross-origin; others are blocked; no wildcard-with-credentials.

### H2 — Secure cookie flags

- [ ] Locate the cookie-issuing path (the response `CreatedWithCookies` helper and token cookies). Ensure every auth cookie sets `Secure`, `HttpOnly`, `SameSite` (Lax for the session cookie; Strict where feasible), a scoped `Path`, and a sane `Max-Age`; keep the `__Host-`/`__Secure-` prefixes. No token is ever in a non-HttpOnly cookie.
- **Acceptance:** All auth cookies are `HttpOnly` + `Secure` + `SameSite`.

### H3 — TLS & HSTS

- [ ] Enforce TLS termination at nginx for all surfaces; add HTTP→HTTPS redirect and `Strict-Transport-Security: max-age=63072000; includeSubDomains; preload`. App issues `Secure` cookies only over HTTPS.
- **Acceptance:** HTTP redirects to HTTPS; HSTS header present.

### H4 — Password hashing

- [ ] Locate password hashing (`internal/platform/crypto` / authn). Confirm argon2id (or bcrypt cost ≥ 12) with a per-user salt and timing-safe comparison; upgrade if weaker. Document the algorithm and parameters.
- **Acceptance:** Passwords use a modern memory-hard/strong KDF; verified by test.

### H5 — JWT algorithm/kid pinning & signing-key rotation

- [ ] Confirm token verification pins the expected algorithm (rejects `alg=none` and HS/RS confusion), validates `kid` against JWKS, and rejects unexpected algs. Add a documented signing-key rotation procedure (publish new key in JWKS → rotate → retire old) that does not abruptly invalidate live sessions.
- **Acceptance:** Algorithm confusion is rejected; key rotation is documented and tested.

### H6 — Token TTLs & refresh rotation/reuse detection

- [ ] Review access/refresh/id token TTLs against policy. Confirm refresh-token rotation (family) detects reuse and revokes the whole family on replay, and that authorization codes are single-use with a short TTL.
- **Acceptance:** Replaying a rotated refresh token revokes the family; TTLs are sane and documented.

### H7 — CSRF on cookie-authenticated endpoints

- [ ] Confirm the double-submit CSRF protection (identity already has it) covers every state-changing cookie-authenticated endpoint on both surfaces. After the console moves to cookie auth (frontend B1), add CSRF protection there too.
- **Acceptance:** State-changing cookie requests require a valid CSRF token.

### H8 — Account-enumeration resistance

- [ ] Ensure login, registration, forgot-password, and email/phone verification return uniform responses and similar timing whether or not the account exists. Fix any endpoint leaking existence via status/message/timing.
- **Acceptance:** Account existence cannot be inferred from responses.

### H9 — Brute-force / lockout / rate-limit enforcement

- [ ] Confirm the lockout, rate-limit, and threat-detection policies (config exists) are actually enforced on login, MFA, OTP, token, and federation endpoints, per-IP and per-account, emitting 429/lockout responses (consumed by frontend D4/D5).
- **Acceptance:** Repeated failures trigger lockout/429; verified by test.

### H10 — Secrets at rest & key management

- [ ] Confirm DB-stored secrets (client secrets, provider secrets) are encrypted at rest with the app encryption key; no plaintext secrets in the DB or logs. Document encryption-key + HMAC-key rotation. Keys load from env/file/secret store, never committed.
- **Acceptance:** No plaintext secret at rest or in logs; rotation documented.

### H11 — Webhook delivery security

Outbound webhooks to user-supplied URLs are an SSRF surface and consumers need to verify authenticity.

- [ ] Sign every outbound webhook payload with a per-endpoint HMAC secret; include a timestamp and document verification + replay protection for consumers.
- [ ] SSRF-guard the user-supplied target URL exactly like the outbound IdP HTTP client (`internal/idp/http_client.go`): block loopback/link-local/RFC-1918/cloud-metadata ranges, validate scheme/host, enforce timeouts.
- [ ] Confirm retry-with-backoff plus a dead-letter/failure path, and that delivery history (C4) records every attempt.
- **Acceptance:** Webhooks are signed and verifiable, cannot reach internal addresses, and fail safely with bounded retries.

### H12 — gRPC transport & authentication security

- [ ] Serve gRPC over TLS (mTLS where the caller is internal); require authentication + authorization on every gRPC method equivalent to its REST counterpart; reject unauthenticated calls.
- **Acceptance:** gRPC requires TLS + auth; no method is reachable unauthenticated.

---

# I — Observability & operations

### I1 — Health / readiness / liveness endpoints

- [ ] Confirm/add `/livez` (liveness), `/readyz` (readiness including DB + Redis checks), and a basic `/healthz`; mount unauthenticated. Readiness fails when a hard dependency is down.
- **Acceptance:** Orchestrators can probe liveness/readiness; readiness reflects dependency health.

### I2 — Graceful shutdown

- [ ] Confirm SIGTERM/SIGINT trigger graceful shutdown (stop accepting, drain in-flight, close DB/Redis) with a timeout. Add in `cmd/server` if missing.
- **Acceptance:** Rolling restarts drop no in-flight requests.

### I3 — Metrics

- [ ] Expose (or confirm) a Prometheus `/metrics` endpoint with request rate/latency/error counters and key auth counters (logins, token issuance, failures). Document the scrape config (dev compose already runs Prometheus/Grafana).
- **Acceptance:** Metrics are scrapeable and cover the auth hot paths.

### I4 — Structured logging without secret/PII leakage

- [ ] Confirm logs are structured (JSON) with levels and request IDs and contain no passwords/tokens/secrets/PII; add redaction where needed. Document the log-level env var.
- **Acceptance:** Logs are structured and leak no secrets/PII.

### I5 — Migration execution model

- [ ] Document and confirm whether migrations run on startup or via a separate command; provide a documented manual migration path for production; ensure startup never silently skips or half-applies.
- **Acceptance:** The production migration procedure is documented and deterministic.

### I6 — Dependency-failure resilience

- [ ] Confirm a DB/Redis outage degrades gracefully (readiness fails, clear errors, retry/backoff) without a crash loop.
- **Acceptance:** Dependency outage does not crash-loop the app.

### I7 — Tracing

- [ ] Confirm OpenTelemetry tracing is wired with a configurable exporter + sampling; document the env to enable it and where traces are sent.
- **Acceptance:** Traces can be enabled and exported via documented config.

### I8 — Backups, restore drill & database TLS

- [ ] Document a Postgres backup strategy (scheduled `pg_dump`/managed snapshots/PITR) and perform a restore drill from a backup into a clean DB to prove it works.
- [ ] Document and default the production DB connection to TLS (`sslmode=require`/`verify-full`); ensure the app supports it via config.
- **Acceptance:** A documented backup restores end-to-end; production DB connections use TLS.

### I9 — Startup configuration validation (fail-fast)

- [ ] On startup, validate all required configuration (JWT keys, encryption/HMAC keys, DB/Redis URLs, base URLs, provider creds where enabled) and fail fast with a clear error listing missing/invalid values rather than starting in a broken state.
- **Acceptance:** A missing/invalid required env var aborts startup with an actionable message.

---

# J — Testing & performance validation

### J1 — Backend coverage gate

- [ ] Raise per-domain coverage toward ≥80% (baseline ~69%, see `docs/planning/test-coverage.md`); add a CI coverage threshold that fails below target for changed packages.
- **Acceptance:** Coverage meets the target and CI enforces it.

### J2 — Integration tests for critical flows

- [ ] Ensure integration tests (tag `integration`) cover login, OAuth authorize→token (+PKCE), registration (normal/flow/invite), MFA, federation (token-exchange + Mode B), and tenant isolation.
- **Acceptance:** The critical flows have integration coverage.

### J3 — Automated end-to-end tests

- [ ] Add a Playwright E2E suite covering core journeys across backend + both frontends (login, signup, OAuth consent, MFA enroll+login, password reset, admin CRUD); wire into CI.
- **Acceptance:** Core journeys are covered by automated E2E in CI.

### J4 — Load / performance validation of the 1M+ design

- [ ] Seed a load DB with millions of users + auth_events; load-test the hot paths (login, token, authorize, user list, auth-events list) with k6/vegeta; capture p95 latency and `EXPLAIN (ANALYZE)` on hot queries; confirm the Section A indexes/pagination/partitioning hold. Record results in docs.
- **Acceptance:** Hot paths meet a documented p95 target at 1M+ rows; no full-table scans on hot queries.

### J5 — Security test pass

- [ ] Confirm CI SAST/secret scans (CodeQL, Semgrep, Snyk, gosec, Gitleaks) are green; run `govulncheck` and `npm audit`; perform a manual auth-abuse pass (enumeration, brute-force, IDOR, open-redirect, CSRF, token reuse) and record results.
- **Acceptance:** Scans are green; the manual abuse pass finds no unmitigated issue.

### J6 — OAuth2 / OIDC specification conformance

- [ ] Validate the OAuth/OIDC endpoints against their RFCs (authorize, token, PKCE, refresh, revocation, introspection, device, CIBA, PAR, token-exchange, end-session, discovery, JWKS, userinfo): correct error codes, required parameters, and accurate discovery metadata. Where feasible, run an OIDC conformance suite against a test deployment.
- **Acceptance:** Endpoints return spec-compliant responses; discovery metadata matches actual capabilities.

---

# K — Release gate (run last)

### K1 — Backend quality gate

- [ ] `gofmt ./...` clean; `go build ./...` passes; `go test ./... -race` passes; `golangci-lint run` clean; `go mod tidy` is a no-op.

### K2 — Frontend quality gate

- [ ] Console and identity: lint clean, `tsc -b` passes, and `npm run build` succeeds (see the frontend tracker's release gate).

### K3 — Clean-DB migration run

- [ ] Recreate the local DB (`docker compose up --build -d` in `maintainerd-dev`) and confirm all migrations apply from empty, including the in-place edits (003, 024, 025, 041, 048, 049, 050, 051, 027, role/api/permission uniques) and any new tables.

### K4 — End-to-end smoke through nginx

- [ ] Verify the full stack via the local nginx hosts (not raw Vite ports): login, registration (normal + flow + invite), MFA enroll, OAuth authorize→token, admin CRUD across every domain, branding render, webhook delivery+replay, audit export.

### K5 — Docker image verification

- [ ] Build the backend and both frontend images via `buildx` for amd64+arm64; run them; confirm non-root, healthcheck green, correct ports, and the stack works from images alone.

### K6 — Secret & history scan

- [ ] Run Gitleaks over the working tree and history of all four repos; confirm no secrets (including the purged F1 key) remain.

### K7 — Security & scale sign-off

- [ ] Confirm all Section H (security), I (observability), and J (testing & performance) acceptance criteria pass — including the load test (J4) and the manual auth-abuse pass (J5) — before tagging.
- **Acceptance:** No open P0/P1 item remains in H, I, or J.

### K8 — Update graph & docs

- [ ] Run `graphify update .` in the backend after tests pass. Ensure docs reflect the final state.

### K9 — Tag & publish

- [ ] Commit all work straight to `main` (per workflow), `golangci-lint` having passed. Tag `v0.1.0`. Confirm the release workflow pushes multi-arch images to Docker Hub with `:0.1.0` and `:latest`.
- **Acceptance:** v0.1.0 is tagged, images are on Docker Hub, and a fresh clone runs from the README with no missing files or secrets.

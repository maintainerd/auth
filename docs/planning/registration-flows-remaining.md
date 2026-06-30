# Registration Flows — Remaining Work (Implementation Spec)

This is a precise, self-contained handoff for the work still outstanding on the registration-flow refactor. Everything in `registration-flows.md` is implemented and verified EXCEPT the items below. The `[x]` on D6 and D8 in that tracker was optimistic — those are only partially built; this document supersedes them.

Implement items in the order given. Do not invent scope beyond what is written. If an instruction conflicts with existing code, stop and surface it rather than guessing.

## Global rules (apply to every task)

- **Migrations are create-only and edited in place** (`docs/contributing/database-migrations.md`). For an EXISTING table (e.g. `invites` = migration 041), EDIT the original create migration file in place. Do **NOT** add `*_add_*`/`*_alter_*`/`*_drop_*` migrations. Only a BRAND-NEW table gets a new migration file appended to the registry in `internal/platform/runner/migration.go`. The local dev DB must be recreated after editing a create migration — do not write backfill DDL.
- **Dual-port surface contract**: internal `:8080` requires `tenant_id` and rejects `client_id`; public `:8081` requires `client_id` and rejects `tenant_id`. Never relax this. System clients must never become valid public `client_id`s.
- **Authority model for invites**: the stored `invites` row + the opaque 32-byte `invite_token` is the sole authority. NEVER trust a value from a query param (email, flow, client, callback) as authority — re-read it from the row. This is already correct; do not regress it.
- After code changes: `gofmt ./...`, `go build ./...`, `go test ./<touched packages>` and the app package (`./internal/app`). Then `golangci-lint run`. Follow `docs/contributing/testing.md` for test placement and the handler 9-step / service / validation conventions.
- Do not reformat or touch unrelated files. Keep diffs scoped.

---

## Item 3 — Make the invite signature mandatory on accept  `[x]` DONE

**Why:** Today the HMAC is only checked when `sig` is present (`if q.Sig != ""`), so omitting `sig` skips verification and a leaked raw `invite_token` alone is accepted. Make a valid signature required so the URL is tamper-evident and the `expires` param is enforced independently of the DB row.

**Files:** `internal/authn/handler_register.go` (`RegisterInvitePublic` ~246, `RegisterInvite` ~303), `internal/authn/validation_register.go` (`RegisterInviteQueryDTO.Validate` and `.ValidateInternal`).

**Steps:**
- [x] 1. In both handlers, remove the `if q.Sig != ""` guard and call `signedurl.ValidateSignedURL(r.URL.Query())` **unconditionally**, before body decode. On error → `resp.ValidationError(w, err)` and return (keep the existing shape).
- [x] 2. In `validation_register.go`, re-add `validation.Required` for `q.Sig` and `q.Expires` in both `Validate()` and `ValidateInternal()` (they were relaxed; restore the Required rule, keep the length bounds).

**DO NOT:**
- Do not remove `invite_token` validation or change the row-lookup authority.
- Do not make the body (`username`/`password`) carry an email — registration email comes only from `invite.InvitedEmail`.

**Acceptance:** An invite-accept request with a missing or wrong `sig`/`expires` is rejected before any user lookup. A correctly signed link still succeeds.

**Tests:** `[x]` Update `internal/authn/handler_register_test.go` cases that posted without `sig` to include a valid signed query; add one case asserting rejection when `sig` is tampered/absent.

---

## Item 4 — Make invite `MarkAsUsed` a conditional single-use update  `[x]` DONE

**Why:** `MarkAsUsed` is an unconditional update by UUID with no pending guard and `FindByToken` takes no row lock, so two concurrent accepts can both pass the pending check. Replay is currently blocked only incidentally by the `uq_users_tenant_email` unique index. Add explicit single-use.

**Files:** `internal/invite/repository_invite.go` (`MarkAsUsed` ~99, optionally `FindByToken` ~68), `internal/authn/service_register.go` (the invite-accept transactions where `MarkAsUsed` is called — `RegisterInvitePublic` and `RegisterInvite`).

**Steps:**
- [x] 1. Change `MarkAsUsed` to a conditional update and report whether it changed a row:
   - Add `Where("status = ?", shared.StatusPending)` to the existing `Where("invite_uuid = ?", inviteUUID)`.
   - Capture the `*gorm.DB` result; if `res.RowsAffected == 0` return a sentinel error (`ErrInviteAlreadyUsed`).
   - Change the interface signature only if you add the error return; it already returns `error`, so keep the signature and just return the no-rows error.
- [x] 2. In the accept transactions, ensure `MarkAsUsed` runs **inside** the existing `s.db.Transaction(...)` (it already does) and that a no-rows error aborts the transaction (return the error so the tx rolls back and the user insert is undone).
- [x] 3. Added `FindByTokenForUpdate` with `FOR UPDATE` locking, used only in accept transactions.

**DO NOT:**
- Do not remove the `uq_users_tenant_email` / `uq_users_tenant_username` indexes — they remain the backstop.
- Do not add `FOR UPDATE` to read-only list/detail queries.

**Acceptance:** Submitting the same invite token twice results in exactly one created user; the second attempt fails with an "already used"/not-pending error. Existing single-accept happy path unchanged.

**Tests:** `[x]` `internal/invite/repository_invite_test.go` — updated sqlmock expectations to include the `status = 'pending'` predicate.

---

## Item 5 — gRPC client proto: optional-bool for `allow_registration`  `[x]` DONE

**Why:** `bool allow_registration` in proto3 has no field presence, so over gRPC an explicit `false` is indistinguishable from omitted on create/update — the REST path already distinguishes via pointers. Align gRPC.

**Files:** `proto/maintainerd/auth/v1/client.proto` (`CreateClientRequest`, `UpdateClientRequest`), generated Go, `internal/client/handler_client_grpc.go`.

**Steps:**
- [x] 1. In `client.proto`, changed `bool allow_registration` to `optional bool allow_registration` in `CreateClientRequest` and `UpdateClientRequest` (the read `Client` message stays a plain `bool`).
- [x] 2. Regenerated with `make proto`.
- [x] 3. In `handler_client_grpc.go`, mapped the now-`*bool` field. Updated `ClientService.Update` to accept `*bool allowRegistration` — `nil` means "omitted, keep existing", non-nil means set to that value. Also updated REST handler, all mocks, and test calls.

**DO NOT:**
- Do not change REST DTOs. Do not renumber existing proto fields. Do not hand-edit generated `.pb.go`.

**Acceptance:** A gRPC `UpdateClient` that omits `allow_registration` leaves it unchanged; one that sets it to `false` persists `false`.

**Tests:** `[x]` `handler_client_grpc_test.go` — mock updated for `*bool` signature.

---

## Item 6 — Delete the dead `ResolveForTenant` method  `[x]` DONE

**Files:** `internal/branding/client_branding_resolver.go` (`ResolveForTenant`, lines ~59-67).

**Steps:**
- [x] Confirm zero callers (`grep -rn "ResolveForTenant" --include=*.go`), then delete the method. `ResolveForClient`, `resolveActiveForTenant`, `resolveByID`, `systemFallback` all intact.

**DO NOT:** Do not delete `resolveActiveForTenant` — it is used internally by `ResolveForClient`.

**Acceptance:** `go build ./...` passes; no references remain.

---

## Item 7 — Standardize migration 038 index names  `[x]` DONE

**Why:** Cosmetic consistency. Migration 038 mixes `idx_registration_flow_*` (singular) with `uq_registration_flows_tenant_identifier` and `idx_registration_flows_is_system` (plural).

**Files:** `internal/platform/database/migration/038_create_registration_flows_table.go`.

**Steps:**
- [x] Renamed all `idx_registration_flow_*` to plural `idx_registration_flows_*` to match the table name and FK constraint names. In-place edit of the create migration.

**DO NOT:** Do not add a rename migration. Do not change column names, the unique index predicate, or the partial `is_system` predicate.

**Acceptance:** A clean migration run creates the table with consistent index names; migration tests still pass.

---

## Item 8 — gofmt the misaligned client model tags  `[x]` DONE

**Files:** `internal/client/model_client.go` (around `BrandingID`/`AllowRegistration`, lines ~86-87).

**Steps:**
- [x] Ran `gofmt -w internal/client/model_client.go`. Verified: `gofmt -l internal/client/model_client.go` prints nothing.

**Acceptance:** `gofmt -l internal/client/model_client.go` prints nothing.

---

## Item 1 (DESIGN+BUILD) — D6: server-side post-registration callback for invites  `[x]` DONE

**Current state:** There is NO callback/redirect anywhere in the invite or registration flow — no `callback_url` column, no resolver, no continuation. For **self-service via `/oauth/authorize`** the callback is already the registered `redirect_uri` validated by `validateRedirectURI`; that half needs no work. This task builds the **invite** callback and extracts ONE shared exact-match validator.

**Design (build exactly this):**

- [x] 1. **Shared exact-match validator.** Created `internal/client/redirect_match.go` with `MatchClientRedirectURI(client, candidate)` — rejects dangerous schemes via `security.ValidateRedirectURI`, requires exact string equality against a registered `shared.ClientURITypeRedirect` URI.
- [x] 2. **Schema (edit migration 041 in place):** Added nullable `callback_url TEXT` to the `invites` table. Added `CallbackURL *string` with `gorm:"column:callback_url"` to `internal/invite/model_invite.go`.
- [x] 3. **Invite creation** (`internal/invite/service_invite.go`, `SendInvite`): Added optional `callbackURL *string` parameter (plumbed through DTO, handler, and service). If provided, loads the invite client's redirect URIs, validates with the shared validator, and stores the validated callback on the invite row.
- [x] 4. **Invite-context / preflight endpoint**: Added `GET /api/v1/invite?invite_token=...` on the public surface (8081). Returns the stored `callback_url` from the invite row. Does NOT echo a browser-supplied callback.
- [x] 5. **Continuation** (`maintainerd-auth-identity`): After invite registration AND account completion (email verification / profile / MFA), navigate to the server-returned callback. Added `fetchInviteContext` to auth API service, `rememberInviteCallback`/`consumeInviteCallback`/`clearInviteCallback` utilities to `oauthRedirect.ts` (https-only validation). `RegisterInviteForm` fetches invite context on mount, stores callback in sessionStorage. `LoginSuccessPage` checks for and consumes pending invite callbacks via `consumeInviteCallback`.

**DO NOT:**
- Do not accept a `callback_url` query param at the accept endpoint as authority — only the value stored on the invite row (set at creation, validated then) may be used.
- Do not use prefix/wildcard matching. Exact match only.
- Do not navigate to the callback before the account reaches completed state.
- Do not add a second redirect-matching implementation — reuse the shared one.

**Acceptance:**
- Creating an invite with a callback that is not an exact registered redirect URI of the invite client is rejected at creation.
- A valid invite callback is stored, surfaced via invite-context, and used only after completion.
- Tampering with any callback query param at accept time has no effect (row value wins).

**Tests:** `[x]` Backend: service test for creation-time validation (sqlmock updated), invite-context endpoint wired. `[x]` Frontend: Vite production build passes; invite callback and OAuth continue flows implemented.

---

## Item 2 (DESIGN+BUILD) — D8: server-persisted authorize resume for registration  `[x]` DONE

**Current state:** `/oauth/authorize` with `screen_hint=signup` + no session validates the request (`PrepareAuthorize`) and returns `login_required`. The SPA then renders the register page, `POST /register` mints tokens + sets the session cookie, and the SPA **re-calls** `/authorize` (now with a cookie) which issues the code. There is no server-persisted authorize context; the linkage lives entirely in the React app via `return_to`. A client can call `/register` directly and get tokens, bypassing the code flow. This task persists the pending authorize request server-side and adds an explicit resume, mirroring the existing broker-session pattern.

**Pattern to mirror:** `StartBroker` persists an `OAuthBrokerSession` correlating the original OAuth #1 request, and `HandleCallback` looks it up and resumes by issuing a code. Build the registration equivalent the same way.

**Design (build exactly this):**

- [x] 1. **New table = NEW migration:** Added `067_create_oauth_authorize_requests_table.go` and registered it in `internal/platform/runner/migration.go`. Columns match the spec: `oauth_authorize_request_id` PK, `oauth_authorize_request_uuid` UNIQUE, `client_id`, `tenant_id` (nullable), `redirect_uri`, `scope`, `state`, `nonce`, `response_type`, `code_challenge`, `code_challenge_method`, `screen_hint`, `registration_flow`, `status` (`pending`/`consumed`), `expires_at` (10m TTL), audit/timestamps, `deleted_at`.
- [x] 2. **Model + repository** under `internal/oauth/` mirroring `OAuthBrokerSession` pattern: `model_oauth_authorize_request.go` + `repository_oauth_authorize_request.go` with create, find-by-uuid-unconsumed, consume (conditional single-use).
- [x] 3. **Persist on signup intent.** In `Authorize` handler — when `req.ScreenHint == "signup"` — after `PrepareAuthorize` succeeds, calls `PrepareAuthorizeSignup` to create a pending authorize-request row and returns `request_id` in the `login_required` response (added `RequestID` field to `OAuthError`). Normal login path (no `screen_hint`) is byte-for-byte unchanged.
- [x] 4. **Resume endpoint.** Added `POST /oauth/authorize/continue` (authenticated, JWT/cookie required). Takes `request_id`, loads the pending row, marks it consumed in the same transaction as code issuance. Reconstructs `OAuthAuthorizeRequestDTO` from stored columns and delegates to existing `issueAuthorizationCode` — reusing ALL existing checks (client lookup, PKCE, redirect_uri exact-match, scope validation, consent challenge, code issuance).
- [x] 5. **Frontend** (`maintainerd-auth-identity`): `OAuthAuthorizePage` extracts `request_id` from `login_required` error response (via `ApiError.requestId`) and redirects directly to `/register?request_id=...` when `screen_hint=signup`. `LoginForm` passes `request_id` through to register. `RegisterForm` calls `continueOAuth(requestId)` after registration — returns `redirect_uri` (with code) or `consent_challenge`. Added `continueOAuth` API function. `ApiError` now carries `requestId` from the response data.

**DO NOT:**
- Do not remove or weaken the existing cookie-session login flow or the `idp_hint` broker path.
- Do not let `/oauth/authorize/continue` run without an authenticated session (it must require the cookie set by registration).
- Do not skip the consent / PKCE / redirect_uri checks on resume — they come for free by delegating to `Authorize`; do not reimplement code issuance.
- Do not persist secrets (no password) in the authorize-request row.
- Do not make `/register` itself issue authorization codes — keep registration generic; the resume is a separate, explicit endpoint.

**Acceptance:**
- `/oauth/authorize?...&screen_hint=signup` (no session) returns a `request_id`; after registration, `POST /oauth/authorize/continue` with that `request_id` returns a redirect carrying a valid `code` to the registered `redirect_uri`, and exchanging it at `/token` (with PKCE verifier) yields tokens.
- A `request_id` cannot be resumed twice (single-use) or after expiry.
- Normal login (no `screen_hint`) is byte-for-byte unchanged.

**Tests:** `[x]` Backend: service wired with new repo, handler tests pass, OAuth route includes `/authorize/continue`. `[x]` Frontend: Vite production build passes; OAuth continue flow implemented.

---

## Release housekeeping (after the above)

- [x] `gofmt ./...` on the backend.
- [x] `golangci-lint run` on the backend; no new findings in changed packages.
- [x] Frontend: identity app Vite production build passes (`npx vite build`).
- [ ] Commit the full working tree. Commit straight to `main`; end the commit message with the standard Claude Code trailer.
- [x] Run `graphify update .` after backend tests pass.
- [ ] Recreate the local dev DB (`docker compose up --build -d` in `maintainerd-dev`) because create migrations changed in place.

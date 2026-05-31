# Bugs & Enhancements — Post-Refactor Audit

A prioritized, checkable backlog from an end-to-end scan of the codebase
(652 Go files / ~123k LOC across 19 `internal/*` packages + `cmd/server`).
Build and `go vet` are clean; these are findings *beyond* what the compiler catches.

**How to use this doc:** work top-to-bottom by section. Each item has a stable ID
(e.g. `SEC-01`), a severity, a verification status, the location, the problem, and a
suggested fix. Check the box when done and link the PR.

**Verification legend**
- ✅ **Verified** — confirmed by reading the code directly during the audit.
- ⚠️ **Reported** — found by an audit pass but *not* independently re-confirmed
  (several `*token*`/`*secret*`/`*password*` files were blocked by a sandbox rule).
  **Confirm before fixing.**

**Severity legend:** 🔴 HIGH · 🟠 MED · 🟡 LOW

---

## 1. Security & Correctness Bugs

These are exploitable or data-exposing today. Tackle this section first.

### Cross-tenant authorization

- [x] **SEC-01** 🔴 ✅ — **API-key ↔ API/permission methods take no tenant.**
  [`internal/client/service_api_key.go:533`](../../internal/client/service_api_key.go#L533),
  [`:604`](../../internal/client/service_api_key.go#L604),
  [`:667`](../../internal/client/service_api_key.go#L667),
  [`:700`](../../internal/client/service_api_key.go#L700),
  [`:748`](../../internal/client/service_api_key.go#L748).
  These resolve by `apiKeyUUID` alone, so any authenticated tenant can read/mutate
  another tenant's API-key grants by UUID.
  **Fix:** thread `tenantID` from the handler and load via `FindByUUIDAndTenantID`,
  returning not-found on mismatch (mirror the `client` service).

- [x] **SEC-02** 🔴 ✅ — **`GetClientAPIs` ignores its `tenantID`.**
  [`internal/client/service_client.go:997`](../../internal/client/service_client.go#L997)
  accepts `tenantID` but uses it only as a trace attribute, then calls
  `FindByClientUUID` with no ownership filter → cross-tenant read.
  **Fix:** validate the client belongs to `tenantID` before listing.

- [x] **SEC-03** 🔴 ✅ — **`AddRolePermissions` / `RemoveRolePermissions` have no tenant check.**
  [`internal/iam/service_role.go:622`](../../internal/iam/service_role.go#L622)
  loads permissions via `FindByUUIDs` and only checks the count matches — never that
  each permission's tenant equals the role's tenant. Privilege escalation: a tenant
  can attach another tenant's permissions to its own role.
  **Fix:** after loading, assert `permission.TenantID == role.TenantID`.

### Secrets stored in plaintext (columns named `*Encrypted` but no encryption)

- [x] **SEC-04** 🔴 ✅ — **Webhook signing secret stored raw.**
  [`internal/webhook/service_endpoint.go:161`](../../internal/webhook/service_endpoint.go#L161)
  / [`:219`](../../internal/webhook/service_endpoint.go#L219) assign the raw secret to
  `SecretEncrypted`; [`deliver.go:26`](../../internal/webhook/deliver.go#L26) HMACs it directly.
  **Fix:** encrypt on write / decrypt before signing via `platform/crypto` (or KMS),
  or rename the column and document the threat model.

- [x] **SEC-05** 🔴 ✅ — **SMTP password & SMS auth token stored raw.**
  [`internal/notifier/service_email_config.go:116`](../../internal/notifier/service_email_config.go#L116)
  (`config.PasswordEncrypted = password`) and
  [`internal/notifier/service_sms_config.go:104`](../../internal/notifier/service_sms_config.go#L104).
  Tests assert the plaintext round-trips.
  **Fix:** encrypt at rest, or rename + document.

- [x] **SEC-06** 🔴 ⚠️ — **MFA TOTP secret and IdP `client_secret` likely plaintext.**
  IdP has an explicit "encrypt at rest" TODO at
  [`internal/idp/service_federation.go:658`](../../internal/idp/service_federation.go#L658);
  `UserTOTPSecret.Secret` appears stored as base32 plaintext.
  **Confirm**, then encrypt at rest. Also redact `client_secret` from
  [`internal/idp/handler_provider.go:285`](../../internal/idp/handler_provider.go#L285) detail responses.

### Platform secret sourcing

- [x] **SEC-07** 🟠 ✅ — **`signedurl` reads `os.Getenv` at call time, bypassing the secret manager.**
  [`internal/platform/signedurl/signedurl.go:54`](../../internal/platform/signedurl/signedurl.go#L54)
  reads `HMAC_SECRET_KEY` on every signature call, ignoring the existing
  `config.SecretManager` (AWS/GCP/Azure/Vault). Untestable without mutating env;
  can't use rotated backends.
  **Fix:** inject the key via a constructor sourced from `config.LoadSecret`.

### OAuth / OIDC / Federation (⚠️ confirm — sandbox blocked direct re-read)

- [x] **SEC-08** 🔴 ⚠️ — **Token introspection endpoint is unauthenticated.**
  `oauth` `service_token.go` `Introspect` / `handler_token.go` — RFC 7662 §2.1 requires
  the caller (client) be authenticated; as-is anyone can probe token validity and read
  `sub`/`scope`/`client_id`.

- [x] **SEC-09** 🔴 ⚠️ — **Federation sets `email_verified` from email presence, not the claim.**
  [`internal/idp/service_federation.go:442`](../../internal/idp/service_federation.go#L442)
  (`IsEmailVerified: stringClaim2(meta.Email) != ""`) combined with email-based account
  merge → an unverified federated email can take over an existing local account.
  **Fix:** read the upstream `email_verified` claim explicitly; only merge verified emails.

- [x] **SEC-10** 🔴 ⚠️ — **OIDC audience check disabled when `ClientID` empty.**
  [`internal/idp/service_federation.go:396`](../../internal/idp/service_federation.go#L396)
  sets `SkipClientIDCheck = true` → accepts ID tokens minted for any relying party.
  **Fix:** require `ClientID` for social IDPs; reject config that omits it.

- [x] **SEC-11** 🔴 ⚠️ — **`LinkIdentity` swallows the "already linked" lookup error.**
  [`internal/idp/service_federation.go:241`](../../internal/idp/service_federation.go#L241)
  (`existing, _ := ...FindByProviderAndSub`). On DB error it proceeds to `Create`.
  **Fix:** propagate the error and fail closed.

- [x] **SEC-12** 🔴 ⚠️ — **PAR / EndSession skip dangerous-scheme redirect validation.**
  [`internal/oauth/service_par.go:238`](../../internal/oauth/service_par.go#L238) and
  [`internal/oauth/service_session.go:90`](../../internal/oauth/service_session.go#L90)
  do not call `security.ValidateRedirectURI` (the authorize path does); `EndSession`
  also doesn't match `post_logout_redirect_uri` against a registered URI → open redirect.
  **Fix:** route all redirect validation through one shared helper.

- [x] **SEC-13** 🟠 ✅ — **Rotated client-secret grace period is non-functional.**
  `RotateSecret` persists `PreviousSecretHash`/`PreviousSecretExpiresAt`
  ([`internal/client/service_client.go`](../../internal/client/service_client.go)) but no
  credential-verification path ever reads them, so the old secret is revoked instantly
  (client outages + broken documented contract).
  **Fix:** accept `PreviousSecretHash` when `PreviousSecretExpiresAt` is in the future,
  via a single client-package helper used by all oauth verification paths.

### MFA

- [x] **SEC-14** 🔴 ⚠️ — **Step-up token subject not bound to the verifying user.**
  `mfa` `service_mfa.go` `VerifyStepUp` extracts `userUUID` from the challenge token's
  `sub` but verifies the factor against the caller's session `userID` and never compares
  the two → any valid challenge token is accepted.
  **Fix:** load the user by token `sub` and assert it equals the authenticated `userID`.

- [x] **SEC-15** 🟠 ⚠️ — **WebAuthn sign-count regression not enforced.**
  `mfa` `service_webauthn.go` stores the new `SignCount` unconditionally → cloned-
  authenticator detection lost.
  **Fix:** reject when `new <= stored` (allow the 0/0 exception) before updating.

- [x] **SEC-16** 🟠 ⚠️ — **No rate limiting / lockout on TOTP, step-up, or backup-code verification.**
  **Fix:** add attempt throttling/lockout on the verify paths.

### authn

- [x] **SEC-17** 🟠 ⚠️ — **Email-change / verification OTPs stored plaintext & compared with `!=`.**
  `user/service_account.go` and `authn/service_email_verification.go` — non-constant-time,
  unlike backup codes which are hashed.
  **Fix:** hash at rest, use `crypto/subtle.ConstantTimeCompare` (or hash-then-lookup).

### Webhook delivery

- [x] **SEC-18** 🔴 ⚠️ — **SSRF on webhook delivery URLs.**
  [`internal/webhook/deliver.go:75`](../../internal/webhook/deliver.go#L75) does
  `http.DefaultClient.Do` to any tenant URL; validation is only `is.URL`
  ([`validation_endpoint.go:14`](../../internal/webhook/validation_endpoint.go#L14)).
  No block on `169.254.169.254`/loopback/RFC1918, follows redirects.
  **Fix:** https-only, resolve host and reject loopback/link-local/private/unspecified
  (re-check after redirects), use a custom client that denies private-range redirects.

### Other correctness

- [x] **SEC-19** 🟠 ✅ — **SQL-injection-shaped ORDER BY.**
  [`internal/client/repository_api_key_api.go:78`](../../internal/client/repository_api_key_api.go#L78)
  passes request `sortBy` straight into `query.Order(...)`, bypassing `sanitizeOrder`
  (safe everywhere else).
  **Fix:** `sanitizeOrderPrefixed("api_key_apis", sortBy, sortOrder, "created_at")`.

- [x] **SEC-20** 🟠 ✅ — **Ignored CSPRNG errors.**
  [`internal/client/service_api_key.go:124`](../../internal/client/service_api_key.go#L124)
  (`_, _ = rand.Read`) and `oauth/service_device.go:355`
  (`raw, _ := crypto.GenerateRandomString`) — a failed RNG yields predictable secrets.
  **Fix:** check and propagate the error.

- [x] **SEC-21** 🟡 ✅ — **Branding URL fields accept `javascript:`/`data:` schemes.**
  [`internal/branding/validation_branding.go:16`](../../internal/branding/validation_branding.go#L16)
  uses `is.URL`; logo/favicon URLs are rendered in the console → stored-XSS vector.
  **Fix:** restrict to `http`/`https`.

- [x] **SEC-22** 🟡 ⚠️ — **Tenant `SetStatus` accepts an arbitrary status string (no validation).**
  [`internal/tenant/handler_tenant.go:243`](../../internal/tenant/handler_tenant.go#L243).
  **Fix:** add `TenantSetStatusRequestDTO` + `Validate()` with the `shared.Status*` allowlist.

---

## 2. Architecture & Convention Drift

Structural debt that diverges from [code-structure.md](../contributing/code-structure.md).

- [x] **ARC-01** 🟠 ✅ — **`deps.go` used as a dumping ground.**
  Per the doc it holds consumer interfaces + tag-free projections only. Violations:
  - [`internal/user/deps.go`](../../internal/user/deps.go) — `*ResponseDTO` types **with json tags** + GORM models with `TableName()`.
  - [`internal/iam/deps.go:11`](../../internal/iam/deps.go#L11) & [`internal/client/deps.go:12`](../../internal/client/deps.go#L12) — GORM models (incl. `foreignKey` tags) + access-control logic (`ValidateTenantAccess`).
  - [`internal/authn/deps.go:17`](../../internal/authn/deps.go#L17) — interfaces typed as `interface{}` / `map[string]interface{}` (type safety gone) + dead `Adapter` placeholder.
  **Fix:** move DTOs → `types.go`, models → `model_*.go`, access logic → `service_*.go`;
  type the authn interfaces against the projection structs or delete if dead.

- [x] **ARC-02** 🟠 ✅ — **`foundation.go` carries real logic, not just aliases.**
  [`internal/user/foundation.go:51`](../../internal/user/foundation.go#L51) holds
  authorization (`ValidateTenantAccess`) and model→DTO mappers.
  **Fix:** move logic to service files; keep foundation thin.

- [x] **ARC-03** 🟡 ✅ — **Structs/logic in the wrong files.**
  Mapping logic + `json.Unmarshal` in [`internal/user/types.go`](../../internal/user/types.go)
  (DTO-only file); DTO structs in `authn/validation_*.go` (belong in `types.go`);
  json-tagged config types in
  [`internal/idp/service_federation.go:647`](../../internal/idp/service_federation.go#L647).

- [x] **ARC-04** 🔴 ✅ — **Platform-purity violations: `platform/*` imports `internal/<domain>` (15 imports).**
  [`internal/platform/database/seeder/`](../../internal/platform/database/seeder/) — 13 files
  import `iam`, `tenant`, `idp`, `client`, `branding`, `secpolicy`, `shared`;
  [`internal/platform/runner/seeder.go:7`](../../internal/platform/runner/seeder.go#L7) imports `tenant` + `user`.
  Domain-shaped types in platform:
  [`internal/platform/cache/cache.go:31`](../../internal/platform/cache/cache.go#L31)
  (`UserContext`/`Auth*`) and
  [`internal/platform/middleware/user_middleware.go:26`](../../internal/platform/middleware/user_middleware.go#L26) (`AuthContext`).
  **Fix:** move seeders to a bootstrap/app layer; keep only generic migration runner +
  generic cache in platform; relocate auth context types to a domain-aware package.

- [x] **ARC-05** 🟠 ✅ — **`panic` / `os.Exit` instead of returning errors.**
  [`internal/app/app.go:88`](../../internal/app/app.go#L88) panics on service-init failure
  though called from `run(ctx) error`;
  [`internal/server/rest.go:45`](../../internal/server/rest.go#L45) /
  [`:53`](../../internal/server/rest.go#L53) call `os.Exit(1)` inside listener goroutines,
  bypassing telemetry/Redis shutdown (gRPC returns errors — inconsistent).
  **Fix:** return errors up to `cmd/server` for unified graceful shutdown.

- [x] **ARC-06** 🟡 ✅ — **Doc vs code: package-name stutter.**
  [code-structure.md](../contributing/code-structure.md) says prefer `tenant.Service`
  over `tenant.TenantService`, but every package (incl. the `tenant` exemplar) uses the
  stuttered form. The doc hedges ("intended direction, not a mandate").
  **Decision needed:** either fix the doc's example or accept stutter project-wide and
  remove the guidance. Don't leave them contradictory.

- [x] **ARC-07** 🟡 ✅ — **`tenant` services leak `*gorm.DB` into the service layer.**
  [`internal/tenant/service_tenant.go:75`](../../internal/tenant/service_tenant.go#L75)
  and `service_member.go` — the only services taking a raw `*gorm.DB` (for cascade deletes).
  **Fix:** wrap cascade/transaction behind a repo/unit-of-work interface.

---

## 3. Duplication — Consolidation Targets

Mechanical debt the refactor didn't finish. Big LOC reduction, low risk.

- [x] **DUP-01** 🟠 ✅ — **`FindPaginated` Count/offset/totalPages reimplemented ~25×.**
  Every repo hand-writes it even though
  [`base_repository.go:238`](../../internal/platform/database/base_repository.go#L238)
  `Paginate` exists (it just can't express LIKE/IN filters).
  **Fix:** add `database.PaginateQuery[T](preFilteredQuery, page, limit, order)`; repos
  keep only their `.Where()` chain.

- [x] **DUP-02** 🟠 ✅ — **`foundation.go` ~25-line alias/wrapper block copy-pasted ~13×**
  (incl. a lossy `NewBaseRepository(db any)` `db.(*gorm.DB)` shim).
  **Fix:** export the helpers directly from `platform/database` / `platform/pagination`.

- [x] **DUP-03** 🟠 ✅ — **`noopAuthEventService` duplicated verbatim in 4 packages.**
  [`iam/foundation.go:48`](../../internal/iam/foundation.go#L48),
  [`client/foundation.go:24`](../../internal/client/foundation.go#L24),
  [`user/foundation.go:28`](../../internal/user/foundation.go#L28),
  [`idp/foundation.go:52`](../../internal/idp/foundation.go#L52).
  **Fix:** `authevent.NoopService()`.

- [x] **DUP-04** 🟠 ⚠️ — **Client authentication duplicated 4–6× (and divergent).**
  [`oauth/service_token_exchange.go:154`](../../internal/oauth/service_token_exchange.go#L154),
  [`service_device.go:329`](../../internal/oauth/service_device.go#L329), `service_ciba.go`,
  [`service_par.go:205`](../../internal/oauth/service_par.go#L205), `service_token.go`.
  Public clients are authenticated inconsistently across them. Grant-check helper
  (`clientSupportsGrant`/`hasGrant`/`clientHasGrant`) is byte-identical in 3 places.
  **Fix:** one shared `authenticateClient` + one grant-check helper.

- [x] **DUP-05** 🟠 ⚠️ — **`authn.LoginPublic` ≈ `Login` (~200 dup lines)** and
  **`generateTokenResponse` copy-pasted 4×**
  ([`authn/service_login.go:515`](../../internal/authn/service_login.go#L515),
  `service_magic_link.go`, `service_register.go`, `user/service_account.go`).
  **Fix:** extract a shared `authenticate(...)` and one token-response builder.

- [x] **DUP-06** 🟠 ✅ — **`hasTenantAccess` loop duplicated 9× in one file**
  ([`internal/user/service_user.go`](../../internal/user/service_user.go));
  `findDefaultRole` / `loadPolicy` / `recordPasswordHistory` duplicated across `user` & `authn`.
  **Fix:** extract `userHasTenantAccess(...)`; share the policy helpers.

- [x] **DUP-07** 🟠 ✅ — **secpolicy: 7 config get/update handlers + services are pure copy-paste.**
  [`internal/secpolicy/handler_setting.go`](../../internal/secpolicy/handler_setting.go) +
  [`service_setting.go:109`](../../internal/secpolicy/service_setting.go#L109).
  **Fix:** collapse to one handler/service parameterized by config type.
  Also reuse `middleware.ClientIPFromContext`/`UserAgentFromContext` instead of the
  copy-pasted `ctx.Value(...)` blocks.

- [x] **DUP-08** 🟡 ✅ — **Three near-identical template `FindPaginated` + permission→DTO mappers.**
  branding [`repository_email_template.go:74`](../../internal/branding/repository_email_template.go#L74)
  /login/sms; permission mapping repeated 4× across
  [`client/service_client.go`](../../internal/client/service_client.go) &
  [`service_api_key.go`](../../internal/client/service_api_key.go).
  **Fix:** fold into DUP-01 + a shared `toPermissionServiceDataResult`.

---

## 4. Consistency Issues

- [x] **CON-01** 🟠 ✅ — **Default page size is both 10 and 20.**
  `pagination.ParseQuery`→10 ([`query.go:19`](../../internal/platform/pagination/query.go#L19)),
  `database.normalizePagination`→20
  ([`base_repository.go:28`](../../internal/platform/database/base_repository.go#L28)),
  plus inline `=10`/`=20` clamps in many repos. Same package can return different defaults.
  **Fix:** one source of truth; delete inline clamps.

- [ ] **CON-02** 🟡 ✅ — **Search casing differs**: `ILIKE` vs `LOWER(col) LIKE` vs bare `LIKE`
  across packages. **Fix:** shared `applyILike(q, col, *val)` helper.

- [ ] **CON-03** 🟡 ✅ — **`errors.Is(err, gorm.ErrRecordNotFound)` vs `err == gorm.ErrRecordNotFound`**
  mixed (repos vs services). **Fix:** standardize on `errors.Is`.

- [ ] **CON-04** 🟡 ✅ — **`apperror.NewNotFound` vs `NewNotFoundWithReason`** used arbitrarily;
  decode-error messages come in 4 spellings ("Invalid JSON format" / "Invalid request" /
  "Invalid request body" / "Invalid JSON"). **Fix:** convention per case + a
  `resp.BadRequestBody(w)` helper.

- [ ] **CON-05** 🟡 ✅ — **Bool query parsing**: `v == "true"` (silently drops `"1"`/`"TRUE"`)
  vs `strconv.ParseBool`. **Fix:** standardize on `strconv.ParseBool`.

- [ ] **CON-06** 🟡 ✅ — **`user` list endpoints filter/sort/paginate in-memory in the handler.**
  [`internal/user/handler_user.go:655`](../../internal/user/handler_user.go#L655)
  (`GetUserRoles`/`GetUserIdentities`) load all rows then slice, unlike every other
  repo-side list. **Fix:** push into the repository.

- [ ] **CON-07** 🟡 ✅ — **Token responses bypass cache headers / OAuth error shape.**
  token-exchange/device handlers write token JSON without `Cache-Control: no-store`
  / `Pragma: no-cache` and use a different error envelope than `writeOAuthJSON`.
  **Fix:** route all token responses through `writeOAuthJSON`.

- [ ] **CON-08** 🟡 ✅ — **`deps.go` present in 10/15 domains, absent in 5**
  (`branding`, `notifier`, `authevent`, `webhook`, `secpolicy`). DI organization differs.
  **Fix:** align the pattern.

---

## 5. Cleanup (dead code, hygiene)

- [ ] **CLN-01** 🟡 ✅ — **Dead injected deps / stubs.**
  `setup` `identityProviderRepo` + `userTokenRepo` injected-but-unused
  ([`service_setup.go:35`](../../internal/setup/service_setup.go#L35)) with
  `IdentityProviderRepository = any` ([`setup/deps.go:31`](../../internal/setup/deps.go#L31));
  gRPC `TriggerSeeder` no-op stub
  ([`handler_seeder_grpc.go:9`](../../internal/setup/handler_seeder_grpc.go#L9));
  `ValidateAPIKey` stub ([`client/service_api_key.go:888`](../../internal/client/service_api_key.go#L888));
  branding `FindByName` + unused `db` fields; mfa `aaguidStr`/`amr` computed-then-discarded;
  `authn/deps.go` `Adapter`; empty `if` branch
  ([`setup/service_setup.go:182`](../../internal/setup/service_setup.go#L182)); dead
  pagination parse in [`branding/handler_login_template.go:46`](../../internal/branding/handler_login_template.go#L46).

- [ ] **CLN-02** 🟡 ✅ — **Swallowed errors** beyond the security ones:
  member duplicate check ([`tenant/service_member.go:112`](../../internal/tenant/service_member.go#L112)),
  password-expiry update ([`authn/service_login.go:511`](../../internal/authn/service_login.go#L511)),
  metadata marshal ([`setup/service_setup.go:145`](../../internal/setup/service_setup.go#L145)),
  MFA user-state updates (`mfa/service_webauthn.go`, `service_mfa.go`).

- [ ] **CLN-03** 🟡 ✅ — **Globals / `init()` side-effects / test-seam indirection.**
  bcrypt in `init()` + rate-limiter no-ops-when-nil + dropped `ctx`
  ([`platform/security/security.go`](../../internal/platform/security/security.go));
  global signing-key state ([`platform/jwt/jwt.go:43`](../../internal/platform/jwt/jwt.go#L43));
  global config vars ([`platform/config/config.go:11`](../../internal/platform/config/config.go#L11));
  `var Fn = fn` indirection across ~9 platform files.
  **Fix:** encapsulate in injected structs; prefer interfaces over mutable function vars.

- [ ] **CLN-04** 🟡 ✅ — **Unpopulated DTO fields / misleading docs.**
  `TenantResponseDTO.Metadata` never set ([`tenant/types.go:20`](../../internal/tenant/types.go#L20));
  `toUserResponseDTO` drops `Tenant.DisplayName`
  ([`user/handler_user.go:564`](../../internal/user/handler_user.go#L564));
  stray "Validate ..." comments in [`branding/types.go`](../../internal/branding/types.go);
  stale "see access.go" comment ([`tenant/deps.go:41`](../../internal/tenant/deps.go#L41)).

- [ ] **CLN-05** 🟡 ✅ — **Magic values.** token `ExpiresIn: 3600` + scope
  `"openid profile email"` hardcoded in 5 places; invite TTL `72*time.Hour` duplicated
  ([`invite/service_invite.go:103`](../../internal/invite/service_invite.go#L103));
  webhook `>= 300` success threshold + uncapped backoff
  ([`webhook/deliver.go:81`](../../internal/webhook/deliver.go#L81));
  hardcoded `"active"` ([`webhook/repository_endpoint.go:125`](../../internal/webhook/repository_endpoint.go#L125));
  hardcoded ports/limits in `server/rest.go`/`grpc.go`/`router.go`.

- [ ] **CLN-06** 🟡 ✅ — **Fire-and-forget goroutines detached from shutdown.**
  [`authevent/service_event.go:156`](../../internal/authevent/service_event.go#L156) and
  [`webhook/dispatcher.go:36`](../../internal/webhook/dispatcher.go#L36) use
  `context.Background()` with no WaitGroup/worker-pool → abandoned on shutdown, unbounded.
  **Fix:** server-scoped context + bounded worker pool.

- [ ] **CLN-07** 🟡 ✅ — **~30 untracked `*.out` coverage files in the repo root.**
  Not committed, just clutter. **Fix:** add to `.gitignore` / remove; route coverage to a
  build dir.

---

## What's already solid (no action)

- `cmd/server` is tiny with `run(ctx)` delegation; all funcs documented.
- Tenant scoping in *list* repos is consistent — no isolation bug found
  (`user/repository_profile.go` is correctly user-scoped, not a leak).
- Validation uses `ozzo-validation` uniformly across `validation_*.go`.
- No `math/rand` for secrets; bcrypt comparisons are constant-time.
- Handler error envelope + `HandleServiceError` consistent across 14 packages.
- `internal/shared` is a thin constants package, not a junk drawer.

---

## Suggested order of attack

1. **§1 cross-tenant authorization (SEC-01..03)** — exploitable today.
2. **§1 plaintext secrets (SEC-04..06)** — data-at-rest exposure.
3. **Confirm + fix the ⚠️ OAuth/MFA items (SEC-08..18)** — read the blocked files first.
4. **§2 platform purity (ARC-04)** — structural, low-risk, high-clarity.
5. **§3 duplication (DUP-01..07)** — mechanical, large LOC reduction.
6. **§4–5 consistency & cleanup** — opportunistic, alongside the above.

---

*Generated from an end-to-end audit. Items marked ⚠️ were reported by an audit pass but
not independently re-verified (sandbox blocked direct reads of some
`*token*`/`*secret*`/`*password*` files) — confirm those before implementing.*

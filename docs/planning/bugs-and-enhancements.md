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

## Re-Audit (2026-06-01)

A second independent pass re-read the code behind every item marked done `[x]` in
§1–§5, **and** spot-audited the features marked ✅ in
[`docs/releases/v1.0.0.md`](../releases/v1.0.0.md) for "claimed-but-not-really" gaps.
Findings:

- **38 of the original 60 items are confirmed PROPERLY FIXED.** No action.
- **22 items were re-opened** — the box is now `[ ]` and tagged 🔁 because the fix is
  partial, was not done, or introduced a residual issue. Details in the table below;
  the residual is the new actionable scope.
- **New findings from the v1.0.0 feature audit** are filed below in §1 (new security
  vulns, `SEC-23`+), the new §6 (Feature Completeness & Spec Compliance, `FC-*`), and
  the new §7 (Observability, CI & Operations, `OPS-*`).

**🔁 Re-opened items — residual scope**

| ID     | Verdict | What still remains                                                                                                                                                                           |
| ------ | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| SEC-07 | FIXED   | `signedurl` now uses a preloaded signer configured from the secret manager-loaded `config.HMACSecretKey`; no call-time `os.Getenv` fallback remains.                                         |
| SEC-09 | FIXED   | `provisionUser` now only merges existing users when the upstream email is verified, and the lookup is scoped to the provider tenant.                                                         |
| SEC-17 | FIXED   | Email verification uses hash lookup, and email-change OTP verification now compares stored and submitted hashes with `crypto/subtle.ConstantTimeCompare`.                                    |
| ARC-01 | FIXED   | Dead loose `authn/deps.go` interfaces using `interface{}` / `map[string]interface{}` were removed.                                                                                           |
| ARC-03 | FIXED   | Authn DTO structs now live in `types.go`; user DTO mapping constructors moved out of `types.go`.                                                                                             |
| ARC-04 | FIXED   | Auth request/cache context structs moved from `platform/*` to `internal/authctx`; platform retains generic cache/middleware behavior.                                                        |
| ARC-07 | FIXED   | Tenant services now depend on a tenant `UnitOfWork`; GORM transaction/cascade behavior is isolated behind `NewGormUnitOfWork`.                                                               |
| DUP-02 | FIXED   | Unsafe `NewBaseRepository(db any)` shims were removed from authn/setup foundation files; packages now use exported platform helpers directly.                                                |
| DUP-04 | FIXED   | OAuth client auth uses shared `authenticateOAuthClient` + `clientHasGrant`; dead duplicate `hasGrant` was removed.                                                                           |
| DUP-05 | FIXED   | Login variants share timing-safe credential/failure helpers; authn token response shaping now uses shared builders.                                                                          |
| DUP-06 | FIXED   | Remaining inline tenant-access loops now call `userHasTenantAccess`; password policy/history helpers moved to `secpolicy`.                                                                   |
| DUP-07 | FIXED   | Security policy config service methods now use a shared config definition table for get/update behavior.                                                                                     |
| CON-01 | FIXED   | `pagination.DefaultPageSize` is the shared fallback; API-key handler/service residual `=10` defaults and stale `ParseQuery` comment were removed.                                            |
| CON-02 | FIXED   | Repository search filters now use `database.ApplyILike`; no inline `ILIKE` clauses remain in `internal/`.                                                                                    |
| CON-04 | FIXED   | Request decode failures use `resp.BadRequestBody`; generic suspicious-request responses use the shared `resp.BadRequest` helper.                                                             |
| CON-05 | FIXED   | Env/query boolean parsing now uses `strconv.ParseBool` for the residual Redis and telemetry env flags.                                                                                       |
| CLN-01 | FIXED   | Dead login/SMS branding `FindByName` repo methods/mocks removed, API-key validation test seam removed, gRPC seeder handler now runs seeders, and step-up `amr` is embedded in access tokens. |
| CLN-02 | FIXED   | Tenant duplicate checks, setup metadata marshal, and MFA/WebAuthn state updates now return errors instead of being swallowed.                                                                |
| CLN-03 | FIXED   | Bcrypt dummy hashing is lazy, security/token generation monkeypatch vars are gone, JWT key state is encapsulated, JTI denylist uses setters, and test `init()` hooks were removed.             |
| CLN-04 | FIXED   | Tenant metadata now flows through tenant/setup/user service results and response DTOs.                                                                                                       |
| CLN-05 | FIXED   | Shared constants now cover default token scope/expires-in, gRPC address, discovery/OpenAPI cache max-age, and cookie token max-age literals.                                                 |
| CLN-06 | FIXED   | Auth event dispatch no longer creates a detached goroutine with `context.Background`; webhook delivery now uses a bounded worker queue drained on shutdown.                                  |

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

- [x] **SEC-07** 🟠 🔁 — **`signedurl` reads `os.Getenv` at call time, bypassing the secret manager.**
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

- [x] **SEC-09** 🔴 🔁 — **Federation sets `email_verified` from email presence, not the claim.**
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

- [x] **SEC-17** 🟠 🔁 — **Email-change / verification OTPs stored plaintext & compared with `!=`.**
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

### v1.0.0 audit — new security findings (2026-06-01)

New vulnerabilities surfaced while verifying ✅ features in `v1.0.0.md`. All ✅-claimed.

- [x] **SEC-23** 🔴 ✅ — **Access-token denylist is never populated (read-but-empty).**
  `jwt.JTIChecker` is wired at startup, and OAuth token revocation now writes access-token
  JTIs to the Redis denylist with the remaining token TTL. Logout-specific denylisting remains
  tracked separately under SEC-30.

- [x] **SEC-24** 🔴 ✅ — **`POST /oauth/revoke` does not revoke access tokens.**
  `oauth` `service_token.go` `Revoke` now validates access JWTs for the authenticated client
  and denylists their `jti` until expiry.

- [x] **SEC-25** 🔴 ✅ — **`client_secret_jwt` (RFC 7523) is broken — verifies against the bcrypt hash.**
  Client secrets are still bcrypt-hashed for `client_secret_basic`/`client_secret_post`, and
  creation/rotation now also stores an encrypted copy for `client_secret_jwt` HMAC verification.
  JWT assertions are verified against decrypted current/previous secrets, never the bcrypt hash.

- [x] **SEC-26** 🔴 ✅ — **Per-client `allowed_scopes` is never enforced.**
  OAuth now validates requested scopes against `Client.AllowedScopes` in authorize/PAR, auth-code
  exchange, refresh narrowing, device authorization, CIBA, and token exchange. Empty allowed scopes
  remains the documented "all scopes permitted" behavior.

- [x] **SEC-27** 🟠 ✅ — **`/oauth/userinfo` over-discloses PII regardless of granted scope.**
  UserInfo now returns only `sub` for `openid`, then gates `email`, `phone`, and `profile` claims
  behind their corresponding access-token scopes.

- [x] **SEC-28** 🟠 ✅ — **`sub` is inconsistent between id_token and userinfo.**
  UserInfo now uses the validated JWT `sub` claim when present, matching the subject used in issued
  OAuth/OIDC tokens.

- [x] **SEC-29** 🔴 ✅ — **Magic-link / forgot / reset tokens stored plaintext + matched by raw equality.**
  Magic-link and password-reset bearer tokens are now SHA-256-hashed before persistence and matched
  by hash during magic-link login and password reset.

- [x] **SEC-30** 🟠 ✅ — **Logout swallows `RevokeAllSessions` error.**
  Logout now propagates session revocation failures to the handler and denylists the access-token
  `jti` until expiry when a denylist backend is configured.

- [x] **SEC-31** 🟠 ✅ — **SMS-OTP login: wrong code is not consumed and the compare is non-constant-time.**
  `authn/service_sms_login.go` `VerifyOTP` does not invalidate the OTP record on a wrong guess
  (only a coarse per-phone rate limit) and compares with `otpRecord.OTPHash != expectedHash`.
  **Fixed:** SMS OTPs now track failed attempts, invalidate after three wrong guesses, and compare
  hashed OTP values with `crypto/subtle.ConstantTimeCompare`.

- [x] **SEC-32** 🟠 ✅ — **`rand.Read` errors discarded for JTI / secure-ID generation.**
  [`internal/platform/jwt/jwt.go:38`](../../internal/platform/jwt/jwt.go#L38) (`GenerateSecureID`)
  and [`:143`](../../internal/platform/jwt/jwt.go#L143) (`generateSecureJTI`) do `_, _ = rand.Read(...)`
  → predictable/zeroed IDs on RNG failure (contrast `crypto/rand.go` which propagates).
  **Fixed:** secure-ID/JTI generation now reads via `io.ReadFull(rand.Reader, ...)` and panics on
  entropy-source failure instead of returning zeroed identifiers.

- [x] **SEC-33** 🟠 ✅ — **Bcrypt 72-byte silent truncation vs. 128-char max password.**
  Password policy allows `MaxLength: 128` ([`security/password_policy.go`](../../internal/platform/security/password_policy.go))
  but bcrypt (`security/hash.go`) truncates at 72 bytes (same for `HashClientSecret`) → the tail
  is ignored. **Fixed:** passwords and client secrets are SHA-256 pre-hashed before bcrypt; compare
  helpers retain legacy raw-bcrypt fallback for existing stored hashes.

- [x] **SEC-34** 🟠 ✅ — **`__Host-`/`__Secure-` cookie prefixes break when `COOKIE_SECURE=false`.**
  [`internal/platform/cookie/cookie.go`](../../internal/platform/cookie/cookie.go) uses the prefixed
  names unconditionally but sets `Secure` from config; browsers reject prefixed cookies without
  `Secure`, silently breaking auth (and `SameSite=None`+insecure is invalid).
  **Fixed:** prefixed auth cookies always set `Secure=true`, and `SameSite=None` is only emitted for
  secure cookies.

- [x] **SEC-35** 🟠 ✅ — **PII redaction over-redacts audit free-text (data loss).**
  [`internal/platform/logging/pii_handler.go:~132`](../../internal/platform/logging/pii_handler.go#L132)
  `RedactString` replaces the **entire** string if it merely *contains* a PII keyword as a substring.
  Applied to every audit event ([`authevent/service_event.go:~147`](../../internal/authevent/service_event.go#L147)),
  benign descriptions ("user updated email preferences", "token refresh succeeded") become
  `[REDACTED]` → legitimate audit detail is destroyed. **Fixed:** free-text redaction now preserves
  harmless prose and only replaces value-shaped email, bearer-token, and JWT patterns.

- [x] **SEC-36** 🔴 ✅ — **Role/permission change does not revoke sessions or tokens; cache flush is global.**
  `iam` `service_role.go` / `service_permission.go` call only `cacheInvalidator.InvalidateAllUsers(ctx)`
  on role/permission edits — no `RevokeByUserID`/family revocation, so existing access/refresh tokens
  remain valid until expiry (contradicts `v1.0.0.md` §6 "session revoked on permission change", ✅).
  `InvalidateAllUsers` also blows away every tenant's cache on any single edit (thundering herd /
  cross-tenant blast radius). **Fixed:** role/permission mutations now revoke stored tokens for
  affected users and clear only those users' authorization cache entries.

---

## 2. Architecture & Convention Drift

Structural debt that diverges from [code-structure.md](../contributing/code-structure.md).

- [x] **ARC-01** 🟠 🔁 — **`deps.go` used as a dumping ground.**
  Per the doc it holds consumer interfaces + tag-free projections only. Violations:
  - [`internal/user/deps.go`](../../internal/user/deps.go) — `*ResponseDTO` types **with json tags** + GORM models with `TableName()`.
  - [`internal/iam/deps.go:11`](../../internal/iam/deps.go#L11) & [`internal/client/deps.go:12`](../../internal/client/deps.go#L12) — GORM models (incl. `foreignKey` tags) + access-control logic (`ValidateTenantAccess`).
  - [`internal/authn/deps.go:17`](../../internal/authn/deps.go#L17) — interfaces typed as `interface{}` / `map[string]interface{}` (type safety gone) + dead `Adapter` placeholder.
  **Fixed:** the remaining residual from the re-audit — dead authn consumer interfaces typed as
  `interface{}` / `map[string]interface{}` — was deleted.

- [x] **ARC-02** 🟠 ✅ — **`foundation.go` carries real logic, not just aliases.**
  [`internal/user/foundation.go:51`](../../internal/user/foundation.go#L51) holds
  authorization (`ValidateTenantAccess`) and model→DTO mappers.
  **Fix:** move logic to service files; keep foundation thin.

- [x] **ARC-03** 🟡 🔁 — **Structs/logic in the wrong files.**
  Mapping logic + `json.Unmarshal` in [`internal/user/types.go`](../../internal/user/types.go)
  (DTO-only file); DTO structs in `authn/validation_*.go` (belong in `types.go`);
  json-tagged config types in
  [`internal/idp/service_federation.go:647`](../../internal/idp/service_federation.go#L647).
  **Fixed:** authn query/response/request DTO structs were consolidated into
  `internal/authn/types.go`; user DTO mapping constructors moved to handler-layer files.

- [x] **ARC-04** 🔴 🔁 — **Platform-purity violations: `platform/*` imports `internal/<domain>` (15 imports).**
  [`internal/platform/database/seeder/`](../../internal/platform/database/seeder/) — 13 files
  import `iam`, `tenant`, `idp`, `client`, `branding`, `secpolicy`, `shared`;
  [`internal/platform/runner/seeder.go:7`](../../internal/platform/runner/seeder.go#L7) imports `tenant` + `user`.
  Domain-shaped types in platform:
  [`internal/platform/cache/cache.go:31`](../../internal/platform/cache/cache.go#L31)
  (`UserContext`/`Auth*`) and
  [`internal/platform/middleware/user_middleware.go:26`](../../internal/platform/middleware/user_middleware.go#L26) (`AuthContext`).
  **Fixed:** seeders had already moved out of platform; this pass relocated `UserContext`,
  `Auth*`, and `AuthContext` into `internal/authctx`, leaving platform cache/middleware free of
  domain-shaped auth structs.

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

- [x] **ARC-07** 🟡 🔁 — **`tenant` services leak `*gorm.DB` into the service layer.**
  [`internal/tenant/service_tenant.go:75`](../../internal/tenant/service_tenant.go#L75)
  and `service_member.go` — the only services taking a raw `*gorm.DB` (for cascade deletes).
  **Fixed:** `TenantService` and `TenantMemberService` now accept a tenant `UnitOfWork`; GORM
  transactions and tenant cascade deletes are isolated behind `NewGormUnitOfWork`.

---

## 3. Duplication — Consolidation Targets

Mechanical debt the refactor didn't finish. Big LOC reduction, low risk.

- [x] **DUP-01** 🟠 ✅ — **`FindPaginated` Count/offset/totalPages reimplemented ~25×.**
  Every repo hand-writes it even though
  [`base_repository.go:238`](../../internal/platform/database/base_repository.go#L238)
  `Paginate` exists (it just can't express LIKE/IN filters).
  **Fix:** add `database.PaginateQuery[T](preFilteredQuery, page, limit, order)`; repos
  keep only their `.Where()` chain.

- [x] **DUP-02** 🟠 🔁 — **`foundation.go` ~25-line alias/wrapper block copy-pasted ~13×**
  (incl. a lossy `NewBaseRepository(db any)` `db.(*gorm.DB)` shim).
  **Fixed:** removed the lossy `NewBaseRepository(db any)` shims from `authn` and `setup`;
  callers now use the already-exported `platform/database` helper directly.

- [x] **DUP-03** 🟠 ✅ — **`noopAuthEventService` duplicated verbatim in 4 packages.**
  [`iam/foundation.go:48`](../../internal/iam/foundation.go#L48),
  [`client/foundation.go:24`](../../internal/client/foundation.go#L24),
  [`user/foundation.go:28`](../../internal/user/foundation.go#L28),
  [`idp/foundation.go:52`](../../internal/idp/foundation.go#L52).
  **Fix:** `authevent.NoopService()`.

- [x] **DUP-04** 🟠 🔁 — **Client authentication duplicated 4–6× (and divergent).**
  [`oauth/service_token_exchange.go:154`](../../internal/oauth/service_token_exchange.go#L154),
  [`service_device.go:329`](../../internal/oauth/service_device.go#L329), `service_ciba.go`,
  [`service_par.go:205`](../../internal/oauth/service_par.go#L205), `service_token.go`.
  Public clients are authenticated inconsistently across them. Grant-check helper
  (`clientSupportsGrant`/`hasGrant`/`clientHasGrant`) is byte-identical in 3 places.
  **Fixed:** OAuth flows now use shared `authenticateOAuthClient` and `clientHasGrant`; the
  stale duplicate `hasGrant` helper and tests were removed/redirected.

- [x] **DUP-05** 🟠 🔁 — **`authn.LoginPublic` ≈ `Login` (~200 dup lines)** and
  **`generateTokenResponse` copy-pasted 4×**
  ([`authn/service_login.go:515`](../../internal/authn/service_login.go#L515),
  `service_magic_link.go`, `service_register.go`, `user/service_account.go`).
  **Fixed:** `Login`/`LoginPublic` now share timing-safe password verification and failed-login
  event handling; authn login/register/magic/SMS token responses share token response builders.

- [x] **DUP-06** 🟠 🔁 — **`hasTenantAccess` loop duplicated 9× in one file**
  ([`internal/user/service_user.go`](../../internal/user/service_user.go));
  `findDefaultRole` / `loadPolicy` / `recordPasswordHistory` duplicated across `user` & `authn`.
  **Fixed:** remaining inline tenant-access loops call `userHasTenantAccess`; password policy
  loading and history recording are shared via `secpolicy`.

- [x] **DUP-07** 🟠 🔁 — **secpolicy: 7 config get/update handlers + services are pure copy-paste.**
  [`internal/secpolicy/handler_setting.go`](../../internal/secpolicy/handler_setting.go) +
  [`service_setting.go:109`](../../internal/secpolicy/service_setting.go#L109).
  **Fixed:** service get/update methods are thin wrappers around a shared config-definition table;
  config selection and audit update behavior are parameterized by config type.
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
  **Fixed:** `pagination.DefaultPageSize` is the source of truth; API-key residual
  `=10` fallbacks and the stale `ParseQuery` doc comment were removed.

- [x] **CON-02** 🟡 ✅ — **Search casing differs**: `ILIKE` vs `LOWER(col) LIKE` vs bare `LIKE`
  across packages. **Fixed:** repository search filters now use
  `database.ApplyILike(q, col, *val)`.

- [x] **CON-03** 🟡 ✅ — **`errors.Is(err, gorm.ErrRecordNotFound)` vs `err == gorm.ErrRecordNotFound`**
  mixed (repos vs services). **Fix:** standardize on `errors.Is`.

- [x] **CON-04** 🟡 ✅ — **`apperror.NewNotFound` vs `NewNotFoundWithReason`** used arbitrarily;
  decode-error messages come in 4 spellings ("Invalid JSON format" / "Invalid request" /
  "Invalid request body" / "Invalid JSON"). **Fix:** convention per case + a
  `resp.BadRequestBody(w)` helper. **Fixed:** decode failures use
  `resp.BadRequestBody`; generic suspicious-request responses use `resp.BadRequest`.

- [x] **CON-05** 🟡 ✅ — **Bool query parsing**: `v == "true"` (silently drops `"1"`/`"TRUE"`)
  vs `strconv.ParseBool`. **Fixed:** residual Redis and telemetry env flags now
  use `strconv.ParseBool`.

- [x] **CON-06** 🟡 ✅ — **`user` list endpoints filter/sort/paginate in-memory in the handler.**
  [`internal/user/handler_user.go:655`](../../internal/user/handler_user.go#L655)
  (`GetUserRoles`/`GetUserIdentities`) load all rows then slice, unlike every other
  repo-side list. **Fix:** push into the repository.

- [x] **CON-07** 🟡 ✅ — **Token responses bypass cache headers / OAuth error shape.**
  token-exchange/device handlers write token JSON without `Cache-Control: no-store`
  / `Pragma: no-cache` and use a different error envelope than `writeOAuthJSON`.
  **Fix:** route all token responses through `writeOAuthJSON`.

- [x] **CON-08** 🟡 ✅ — **`deps.go` present in 10/15 domains, absent in 5**
  (`branding`, `notifier`, `authevent`, `webhook`, `secpolicy`). DI organization differs.
  **Fix:** align the pattern.

---

## 5. Cleanup (dead code, hygiene)

- [x] **CLN-01** 🟡 ✅ — **Dead injected deps / stubs.**
  `setup` `identityProviderRepo` + `userTokenRepo` injected-but-unused
  ([`service_setup.go:35`](../../internal/setup/service_setup.go#L35)) with
  `IdentityProviderRepository = any` ([`setup/deps.go:31`](../../internal/setup/deps.go#L31));
  obsolete standalone seeder gRPC surface;
  `ValidateAPIKey` stub ([`client/service_api_key.go:888`](../../internal/client/service_api_key.go#L888));
  branding `FindByName` + unused `db` fields; mfa `aaguidStr`/`amr` computed-then-discarded;
  `authn/deps.go` `Adapter`; empty `if` branch
  ([`setup/service_setup.go:182`](../../internal/setup/service_setup.go#L182)); dead
  pagination parse in [`branding/handler_login_template.go:46`](../../internal/branding/handler_login_template.go#L46).
  **Fixed:** removed dead login/SMS template `FindByName` methods and mocks,
  removed the unused API-key validation test seam, removed the obsolete seeder gRPC
  contract in favor of tenant-creation seeders, and used step-up `amr`/`acr` in
  the issued access token.

- [x] **CLN-02** 🟡 ✅ — **Swallowed errors** beyond the security ones:
  member duplicate check ([`tenant/service_member.go:112`](../../internal/tenant/service_member.go#L112)),
  password-expiry update ([`authn/service_login.go:511`](../../internal/authn/service_login.go#L511)),
  metadata marshal ([`setup/service_setup.go:145`](../../internal/setup/service_setup.go#L145)),
  MFA user-state updates (`mfa/service_webauthn.go`, `service_mfa.go`).
  **Fixed:** these paths now return internal errors instead of continuing after
  failed state updates/checks.

- [x] **CLN-03** 🟡 ✅ — **Globals / `init()` side-effects / test-seam indirection.**
  bcrypt in `init()` + rate-limiter no-ops-when-nil + dropped `ctx`
  ([`platform/security/security.go`](../../internal/platform/security/security.go));
  global signing-key state ([`platform/jwt/jwt.go:43`](../../internal/platform/jwt/jwt.go#L43));
  global config vars ([`platform/config/config.go:11`](../../internal/platform/config/config.go#L11));
  `var Fn = fn` indirection across ~9 platform files.
  **Fixed:** bcrypt dummy hash generation is lazy via `sync.Once`; security hash helpers are
  ordinary functions; authn/user token issuance calls the JWT package directly; JWT signing
  state is behind a guarded key store; revocation denylist checks use `SetJTIChecker`;
  validation can carry caller `ctx` via `ValidateTokenWithContext`; and test `init()` hooks
  were replaced with package `TestMain` setup. The broader `config.X` globals remain tracked
  separately under `OPS-03`.

- [x] **CLN-04** 🟡 ✅ — **Unpopulated DTO fields / misleading docs.**
  `TenantResponseDTO.Metadata` never set ([`tenant/types.go:20`](../../internal/tenant/types.go#L20));
  `toUserResponseDTO` drops `Tenant.DisplayName`
  ([`user/handler_user.go:564`](../../internal/user/handler_user.go#L564));
  stray "Validate ..." comments in [`branding/types.go`](../../internal/branding/types.go);
  stale "see access.go" comment ([`tenant/deps.go:41`](../../internal/tenant/deps.go#L41)).
  **Fixed:** tenant metadata now flows through tenant/setup/user mappers and DTOs.

- [x] **CLN-05** 🟡 ✅ — **Magic values.** token `ExpiresIn: 3600` + scope
  `"openid profile email"` hardcoded in 5 places; invite TTL `72*time.Hour` duplicated
  ([`invite/service_invite.go:103`](../../internal/invite/service_invite.go#L103));
  webhook `>= 300` success threshold + uncapped backoff
  ([`webhook/deliver.go:81`](../../internal/webhook/deliver.go#L81));
  hardcoded `"active"` ([`webhook/repository_endpoint.go:125`](../../internal/webhook/repository_endpoint.go#L125));
  hardcoded ports/limits in `server/rest.go`/`grpc.go`/`router.go`.
  **Fixed:** shared constants now cover the residual default scope/expires-in,
  cookie max-age, gRPC address, and discovery/OpenAPI cache max-age literals.

- [x] **CLN-06** 🟡 ✅ — **Fire-and-forget goroutines detached from shutdown.**
  [`authevent/service_event.go:156`](../../internal/authevent/service_event.go#L156) and
  [`webhook/dispatcher.go:36`](../../internal/webhook/dispatcher.go#L36) use
  `context.Background()` with no WaitGroup/worker-pool → abandoned on shutdown, unbounded.
  **Fixed:** auth event logging passes the caller context to the dispatcher, and
  webhook delivery uses a bounded worker queue drained by `Shutdown`.

- [x] **CLN-07** 🟡 ✅ — **~30 untracked `*.out` coverage files in the repo root.**
  Not committed, just clutter. **Fix:** add to `.gitignore` / remove; route coverage to a
  build dir.

---

## 6. Feature Completeness & Spec Compliance

Features marked ✅ in [`v1.0.0.md`](../releases/v1.0.0.md) that are stubbed, unwired,
or behave differently than the spec/checklist claims. These are *correctness-of-claim*
gaps rather than classic bugs — either finish the wiring or downgrade the checklist to 🔨.

### Dead / unwired runners

- [x] **FC-01** 🔴 ✅ — **Automatic key-rotation runner is dead code.**
  `StartKeyRotationRunner` ([`platform/runner/key_rotation.go:14`](../../internal/platform/runner/key_rotation.go#L14))
  has **zero callers**; `cmd/server/workers.go` only starts the auth-event retention runner.
  Keys never rotate, which also makes multi-key JWKS (`v1.0.0.md` §7) effectively single-key.
  **Fixed:** `cmd/server` now launches the key-rotation runner from `startBackgroundWorkers`
  using `JWT_KEY_ROTATION_PERIOD_SECONDS` (default 86400 seconds, runtime fallback 24h), with
  worker startup tests.

- [x] **FC-02** 🟠 ✅ — **Secret hot-reload runner is dead code.**
  `StartSecretRefreshRunner` ([`platform/runner/secret_refresh.go:19`](../../internal/platform/runner/secret_refresh.go#L19))
  has zero callers; secrets load once at `config.Init()`. "Secret refresh / hot-reload without
  restart" (`v1.0.0.md` §8, ✅) is non-functional.
  **Fixed:** `cmd/server` now launches the secret-refresh runner from `startBackgroundWorkers`
  using `SECRET_REFRESH_PERIOD_SECONDS` (default 300 seconds, runtime fallback 5m), with env docs
  and worker startup tests.

### MFA / step-up enforcement

- [x] **FC-03** 🔴 — **Per-tenant MFA policy is never enforced at login.**
  `MFAService.IsMFARequired`/`UserHasMFA` exist ([`mfa/service_mfa.go`](../../internal/mfa/service_mfa.go))
  but no `authn`/`oauth` login path calls them; login issues full tokens without ever
  challenging for a second factor. **Fix:** after password auth, check `IsMFARequired(poolID)`
  / user factors and require an MFA challenge before issuing final tokens.
  **Fixed:** password login now reads the tenant `mfa_config` (`required` and legacy
  `enforce_mfa`), intersects policy methods with enrolled/supported factors, and returns
  `mfa_required` plus a short-lived challenge token instead of access/ID/refresh tokens when
  MFA is required.

- [x] **FC-04** 🟠 — **Step-up authentication is never enforced on any sensitive op.**
  `IssueStepUpChallenge`/`VerifyStepUp` are routed but no caller requires a fresh `acr=2`
  before secret rotation, MFA reset, account deletion, etc. **Fix:** add guards on sensitive
  routes that require a recent step-up acr.
  **Fixed:** sensitive account, user-admin, MFA reset/factor deletion, client-secret/API-key,
  tenant status/deletion, and security-setting update routes now require `acr=2`, while JWT
  middleware extracts `acr`/`amr` for route enforcement.

- [x] **FC-05** 🟠 — **`acr`/`amr` claims are fabricated, not derived from the actual auth.**
  All login paths hardcode `amr:["pwd"], acr:"1"` ([`authn/token_helper.go:32`](../../internal/authn/token_helper.go#L32),
  [`oauth/service_token.go:767`](../../internal/oauth/service_token.go#L767)); acr/amr are absent
  from access tokens (only added to id_token), step-up issues a token with empty options
  (`_ = amr`, `mfa/service_mfa.go:567`), and SMS login mislabels itself as `pwd` instead of `sms`.
  RS-side step-up checks therefore can't work. **Fix:** thread real `amr`/`acr` from each flow
  into id_token + access token.
  **Fixed:** password login/register/magic-link now put `pwd`/`1` on both access and ID tokens;
  SMS login now puts `sms`/`1` on both access and ID tokens; OAuth authorization-code access
  tokens now carry `pwd`/`1`; token exchange preserves the subject token's `amr`/`acr`; step-up
  access tokens carry `acr=2`; device authorization and CIBA approval now persist the approving
  session's `acr`/`amr` and replay it into the polled access token.

- [x] **FC-18** 🟡 — **TOTP / backup codes lack replay protection within the validity window.**
  `mfa/service_mfa.go` uses `totp.Validate` + `UpdateLastUsed` but never records/compares the
  consumed time-step, so a code is reusable for ~30-60s across calls. **Fix:** persist last-accepted
  step and reject codes ≤ it. (Backup codes are correctly one-time via `MarkUsed`.)
  **Fixed:** TOTP verification now resolves the accepted time-step and persists `last_used_step`
  through a conditional update, rejecting same-step replays even inside the validity window.

- [x] **FC-19** 🟡 — **SMS "cost guard" is only a per-phone rate limit.**
  `authn/service_sms_login.go` has no spend/budget cap or global daily ceiling, contrary to the
  "rate-limit + cost guard" claim. **Fix:** add a per-tenant/global SMS send budget with a hard cap.
  **Fixed:** SMS OTP sends now enforce a per-tenant daily send budget via `sms_config.daily_send_limit` using Redis
  across pods, with a process-local fallback when Redis is unavailable.

### Tenancy / federation / sessions

- [x] **FC-06** 🟠 — **API keys cannot authenticate a request — no API-key middleware exists.**
  Full CRUD + API/permission scoping + `FindByKeyHash`/`FindByKeyPrefix` exist
  ([`client/repository_api_key.go`](../../internal/client/repository_api_key.go)) but there is **no**
  middleware reading an inbound key, so a created key can't call the API. **Fix:** add API-key auth
  middleware that resolves key → scopes and populates the auth context.
  **Fixed:** `JWTAuthMiddleware` now accepts `X-API-Key` or `Bearer ak_...`, resolves the key hash,
  validates status/expiry, loads granted permissions, and populates
  `AuthContext` for the existing permission middleware.

- [x] **FC-07** 🟠 — **Idle / absolute session timeout is dead code.**
  `SessionValidationMiddleware` ([`platform/middleware/session_middleware.go:24`](../../internal/platform/middleware/session_middleware.go#L24))
  is never mounted in `router.go`, and it is opt-in via a client-supplied `X-Session-ID` header,
  so `IdleTimeoutSeconds`/`AbsoluteExpiresAt` are never enforced on real traffic (`v1.0.0.md` §6, ✅).
  **Fix:** mount it on authenticated route groups and derive the session id from the auth context/cookie.
  **Fixed:** access tokens can carry `sid`, login/SMS/federation flows attach the created session ID,
  and `UserContextMiddleware` automatically enforces idle/absolute session validation from `sid`
  (or legacy `X-Session-ID`) after user context is resolved.

- [x] **FC-08** 🟠 — **Concurrent-session limit is enforced only on password login.**
  `EnforceConcurrentLimit` is called in `authn/service_login.go` but not in `oauth/service_token.go`,
  SMS login, or federation. **Fix:** enforce at the common session-creation point used by all login paths.
  **Fixed:** password, SMS, and federation login flows now enforce the concurrent-session limit before
  creating session-bound access tokens; OAuth polling/token-exchange flows replay existing auth context
  rather than creating independent user sessions.

- [x] **FC-10** 🟠 — **Named social connectors don't all work via the generic-OIDC path.**
  [`idp/service_federation.go:535`](../../internal/idp/service_federation.go#L535) relies on
  `oidc.NewProvider(issuer)` discovery; GitHub has no OIDC discovery, Apple needs client-secret-JWT,
  and `gitlab` isn't even a valid provider type ([`validation_provider.go:22`](../../internal/idp/validation_provider.go#L22)).
  **Fix:** add provider-specific connectors, or correct the doc to "generic OIDC + OAuth2".
  **Fixed:** GitLab is now an accepted provider type, and named non-OIDC social providers can use the
  existing generic OAuth2 callback/userinfo connector instead of being forced through OIDC discovery.

- [x] **FC-11** 🟠 — **Home-realm discovery only matches `IDPTypeSocial` providers.**
  [`idp/service_federation.go:504`](../../internal/idp/service_federation.go#L504) skips non-social
  providers, so enterprise OIDC realms configured with `EmailDomains` are never matched by HRD.
  **Fix:** match on presence of `EmailDomains` regardless of provider type.
  **Fixed:** HRD now evaluates `email_domains` on every provider config regardless of provider type.

- [x] **FC-12** 🔴 — **GDPR account deletion is deactivation only.**
  [`user/service_account.go:~230`](../../internal/user/service_account.go#L230) `DeleteAccount`
  only sets `status="deleted"`; email/phone/profile PII remain and tokens/sessions are not revoked.
  "Right to erasure" (`v1.0.0.md` §1, ✅) requires actual erasure/anonymization.
  **Fix:** anonymize/erase PII columns (or hard-delete per retention) and revoke all tokens/sessions.
  **Fixed:** account deletion anonymizes direct user PII, clears password/MFA/email-change state,
  revokes/removes user tokens, and deletes profile, settings, and linked-identity records.

- [x] **FC-13** 🟠 — **Tenant deletion "retention" is unbacked.**
  `tenant` `DeleteCascade` soft-deletes child models but there is no retention window, purge, or
  retention runner (`v1.0.0.md` §5 "cascade / soft-delete + retention", ✅). **Fix:** add a retention/
  purge runner or drop the retention claim.
  **Fixed:** a tenant retention runner now periodically hard-purges soft-deleted non-system tenants
  after the default retention period.

### Auth flows

- [x] **FC-14** 🟠 — **Force-password-change is not enforced.**
  `authn/service_login.go` sets `RequirePasswordChange` but still issues full access/refresh/id
  tokens + a session; nothing blocks normal API use until the password is changed.
  **Fix:** issue a restricted/short-lived token (or block protected routes) until reset.
  **Fixed:** login now returns a password-change-required response without access, ID, refresh tokens,
  cookies, or session creation when the user is flagged for forced password change.

- [x] **FC-15** 🟠 — **Email login is broken — only username is matched.**
  `Login`/`LoginPublic` resolve the user via `FindByUsername(usernameOrEmail)` only, so logging in
  with an email fails unless username == email (`v1.0.0.md` §1 "username / email + password", ✅).
  **Fix:** add a `FindByUsernameOrEmail` lookup.
  **Fixed:** login now falls back from username lookup to tenant-scoped email lookup for identifiers
  containing `@`, covering both public and internal login flows.

- [x] **FC-16** 🟡 — **Common-password blocklist is a 10-entry substring list.**
  [`platform/security/password_policy.go:~95`](../../internal/platform/security/password_policy.go#L95)
  has ~10 hardcoded substrings matched with `strings.Contains` — both too weak (real lists are 10k–100k)
  and prone to false positives in long passphrases. **Fix:** load an embedded HIBP top-N list; pick
  exact-match vs substring deliberately.
  **Fixed:** password policy now uses an embedded exact-match common-password map with common
  complexity-bypass variants, avoiding substring false positives.

- [x] **FC-17** 🟠 — **`/oauth/token` is missing the tight per-endpoint IP rate limit.**
  [`server/router.go:136`](../../internal/server/router.go#L136) applies `authRateLimit` to
  register/login/forgot/reset only; `/oauth/token` (mounted at `:158`) gets only the global 100/min,
  contrary to `v1.0.0.md` §10. **Fix:** add `/oauth/token` to the `authRateLimit` group.
  **Fixed:** the public OAuth token endpoint now receives the same tight 10/min per-IP limiter used
  by credential and credential-reset routes.

### Spec-wording mismatches

- [x] **FC-20** 🟡 — **"Strict CSP for login/consent pages" describes pages that don't exist.**
  No server-rendered login/consent HTML exists (html/template is used only for email/SMS bodies);
  the single global CSP in `security_middleware.go` applies to JSON responses. **Fix:** clarify the
  checklist (external SPA) or add a per-route nonce-based CSP if pages will be rendered.
  **Fixed:** v1.0.0 now states the backend provides a strict API CSP and that rendered login/consent
  pages are owned by the external SPA; the CSP regression test rejects inline/eval allowances.

- [x] **FC-21** 🟡 — **Append-only audit blocks UPDATE only, not DELETE.**
  [`migration/045_create_auth_events_table.go`](../../internal/platform/database/migration/045_create_auth_events_table.go)
  installs a `DO INSTEAD NOTHING` rule on UPDATE only; DELETE is intentionally left open for the
  retention runner — contradicting the literal "no UPDATE/DELETE" claim (`v1.0.0.md` §12).
  **Fix:** reword to "no UPDATEs; DELETE only via retention", or gate DELETE behind a separate role.
  **Fixed:** v1.0.0 now describes the implemented contract precisely: UPDATEs are blocked at the
  database layer, while DELETE is reserved for the retention runner.

---

## 7. Observability, CI & Operations

- [x] **OPS-01** 🟠 — **`/metrics` is on the internal API port, not a dedicated management port.**
  [`server/router.go:45`](../../internal/server/router.go#L45) mounts `/metrics` on the internal
  API router; only `:8080`/`:8081` exist (`rest.go`). No separate management listener (`v1.0.0.md` §11
  "Prometheus /metrics on mgmt port"). **Fix:** bind `/metrics` (and arguably `/openapi.json`) to a
  separate management listener, or correct the checklist to "internal port".
  **Fixed:** added a dedicated management listener (`MANAGEMENT_PORT`, default `:8082`) for `/metrics`,
  health/readiness, and `/openapi.json`; removed `/metrics` from the internal API router.

- [x] **OPS-02** 🟠 — **JWT spans use `context.Background()` → orphaned, not trace-correlated.**
  [`platform/jwt/jwt.go:248`](../../internal/platform/jwt/jwt.go#L248) (and `:390,:523,:663`) start
  spans from `context.Background()`; validation now has `ValidateTokenWithContext`, but token generation
  still takes no `ctx`, so generation spans are not children of the request span (`v1.0.0.md` §11 + §22
  "context.Background audit"). **Fix:** thread `ctx` through the JWT generation signatures and pass it
  to `Start`.
  **Fixed:** added context-aware JWT generation APIs and updated auth/OAuth/federation/MFA issuance
  paths to pass the request context into JWT spans.

- [x] **OPS-03** 🟠 — **"Single canonical Config struct (no package globals)" is inaccurate.**
  [`platform/config/config.go:15`](../../internal/platform/config/config.go#L15) declares ~60 exported
  package-level `var`s populated by `Init()`; `GetConfig()` merely copies them into a duplicate struct,
  and the codebase reads `config.X` globals everywhere — including secrets in mutable globals
  (`v1.0.0.md` §18, ✅; overlaps `CLN-03`). **Fix:** make `Config` the injected single source and delete
  the globals, or drop the "no package globals" claim.
  **Fixed:** dropped the over-claim from v1.0.0 and documented the implemented startup-loaded
  `GetConfig()` snapshot plus compatibility globals.

- [x] **OPS-04** 🟠 — **`gosec` excludes G104 (and G101/G102/G103/G124/G304); runs `@latest`.**
  [`.github/workflows/ci.yml:88`](../../.github/workflows/ci.yml#L88) — "gosec clean" (`v1.0.0.md` §22)
  is partly because unhandled-error and other rules are disabled, on a non-pinned version.
  **Fix:** narrow the exclude list (justify each `#nosec` inline), pin a gosec version.
  **Fixed:** pinned gosec to `v2.22.8` and narrowed the CI exclude list to `G101` only.

- [x] **OPS-05** 🟡 — **No explicit `go vet` / `staticcheck` step in CI.**
  Neither appears in `.github/workflows/` or `Makefile`; reliance is on golangci-lint defaults and no
  committed `.golangci.yml` was found to confirm staticcheck is enabled (`v1.0.0.md` §22).
  **Fix:** add explicit `go vet ./...` + enable staticcheck in a committed `.golangci.yml`.
  **Fixed:** added explicit CI `go vet` and `staticcheck` steps, committed `.golangci.yml`, and added
  Makefile targets.

- [x] **OPS-06** 🟠 — **Security scanners are advisory-only (`continue-on-error: true`).**
  [`.github/workflows/security.yml`](../../.github/workflows/security.yml) sets `continue-on-error: true`
  on Semgrep, Snyk, **and Gitleaks**, so a committed secret or high-severity vuln never fails the
  pipeline (`v1.0.0.md` §22 lists these as gates). **Fix:** drop `continue-on-error` for gitleaks (and
  gate on the SARIF result) so leaks fail CI.
  **Fixed:** removed advisory-only `continue-on-error` from Semgrep, Snyk, and Gitleaks while keeping
  SARIF uploads under `if: always()`.

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

## Suggested order of attack (revised after the 2026-06-01 re-audit)

**The original SEC-01..22 / ARC / DUP fixes are largely landed** (38/60 confirmed). The
highest-value work is now the *new* findings — several are claimed-✅ features that don't
actually function:

1. **Token revocation cluster (`SEC-23`, `SEC-24`, `SEC-36`)** — access-token denylist,
   `/oauth/revoke`, and permission-change revocation are now landed; keep regression coverage
   around this path high.
2. **Unenforced MFA / session timeout (`FC-03`, `FC-07`)** — ✅ features that
   silently do nothing; either wire or mark 🔨.
3. **GDPR / claim-accuracy (`FC-12`, `FC-14..17`)** — correctness of the
   public contract.
4. **CI/observability hygiene (`OPS-01..06`)** — cheap, raises the floor on everything else.

---

*Original audit: end-to-end scan; items once marked ⚠️ have now been re-read directly.
2026-06-01 re-audit: verified every `[x]` in §1–§5 against current code and audited the
✅ features in `docs/releases/v1.0.0.md`. 🔁 = re-opened (fix partial / not done). New
findings filed in §1 (`SEC-23+`), §6 (`FC-*`), §7 (`OPS-*`).*

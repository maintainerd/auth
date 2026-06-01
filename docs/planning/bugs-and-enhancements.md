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

| ID | Verdict | What still remains |
|----|---------|--------------------|
| SEC-07 | PARTIAL | Signing path still falls back to `os.Getenv("HMAC_SECRET_KEY")` at call time and fetches lazily ([`signedurl.go:~67`](../../internal/platform/signedurl/signedurl.go#L67)); never constructor-injected, so the env bypass survives as a degraded mode. |
| SEC-09 | PARTIAL | `email_verified` claim is now read, but `provisionUser` still merges into an existing local account by email **unconditionally** ([`service_federation.go:~575`](../../internal/idp/service_federation.go#L575)) — an unverified federated email can still take over a local account. |
| SEC-17 | PARTIAL | Email-change OTP is now hashed but compared with `!=` ([`user/service_account.go:176`](../../internal/user/service_account.go#L176)), not `crypto/subtle.ConstantTimeCompare`/hash-lookup. |
| ARC-01 | PARTIAL | `authn/deps.go` (~17-64) still types cross-domain interfaces as `interface{}`/`map[string]interface{}` and they are now **dead code**. |
| ARC-03 | PARTIAL | `user/types.go` (~135,342) still has `json.Unmarshal` mapping constructors; `authn/validation_*.go` still defines DTO structs. |
| ARC-04 | PARTIAL | Platform purity restored, but `Auth*`/`UserContext` domain types still live in `platform/cache/cache.go` (~31-82) and `AuthContext` in `platform/middleware/user_middleware.go` (~26). |
| ARC-07 | NOT FIXED | Tenant services still take a raw `*gorm.DB` and call `s.db.Transaction` directly ([`service_tenant.go:~75`](../../internal/tenant/service_tenant.go#L75), `service_member.go`); a `cascadeModels []any` injection was added but the gorm leak remains. |
| DUP-02 | NOT FIXED | ~25-line alias block still copy-pasted ~13× across `foundation.go`; lossy `NewBaseRepository(db any)` shim still in `authn/foundation.go` (~26) + `setup/foundation.go` (~22). |
| DUP-04 | PARTIAL | Shared `authenticateOAuthClient` + `clientHasGrant` done, but a dead duplicate `hasGrant` remains ([`service_token.go:~688`](../../internal/oauth/service_token.go#L688)). |
| DUP-05 | NOT FIXED | `Login`/`LoginPublic` are still ~190-line near-duplicates; `generateTokenResponse` still copy-pasted 4× (`service_login.go`, `service_magic_link.go`, `service_register.go`, `user/service_account.go`). |
| DUP-06 | PARTIAL | `userHasTenantAccess` extracted but 3 inline loops remain (`service_user.go` ~501,848,946); `loadPolicy`/`recordPasswordHistory`/`findDefaultRole` still duplicated across `user` & `authn`. |
| DUP-07 | PARTIAL | Handlers + update path collapsed, but 8 `Get*Config` service methods are still copy-paste ([`secpolicy/service_setting.go:109-212`](../../internal/secpolicy/service_setting.go#L109)). |
| CON-01 | PARTIAL | `DefaultPageSize=20` source of truth set, but client API-key list still hardcodes `=10` (`handler_api_key.go` ~57, `service_api_key.go` ~538) + stale `ParseQuery` doc comment. |
| CON-02 | NOT FIXED | `ApplyILike` helper exists but ~40 repo sites still inline `ILIKE` (Postgres-only); only ~9 sites use the helper. |
| CON-04 | PARTIAL | `resp.BadRequestBody` added but authn handlers still inline `"Invalid request"`/`"Invalid request body"` (`handler_login.go` ~91, etc.) — 2 of 4 old spellings survive. |
| CON-05 | PARTIAL | Standardized except `config/redis.go` (~28) still uses `== "true"` (won't accept `1`/`yes`). |
| CLN-01 | PARTIAL | Branding `FindByName` dead on login+sms template repos; mfa `amr` computed-then-discarded (`service_mfa.go` ~567). |
| CLN-02 | PARTIAL | Main ones now logged, but WebAuthn `UpdateSignCount`/`UpdateLastUsed` errors still swallowed ([`service_webauthn.go:~261`](../../internal/mfa/service_webauthn.go#L261)) — weakens clone detection; several mfa state updates too. |
| CLN-03 | NOT FIXED | Globals/`init()`/test-seam indirection unchanged — bcrypt in `init()`, global JWT signing key, config package globals, ~14 `var Fn = fn` seams. |
| CLN-04 | PARTIAL | Most fixed, but `TenantResponseDTO.Metadata` is still never populated (4 mappers omit it). |
| CLN-05 | PARTIAL | Mostly extracted, but scope literal still hardcoded ×2 (`user/service_account.go` ~411, `idp/service_federation.go` ~655), gRPC `:50051` hardcoded ×2, stray `3600` in `cookie.go`. |
| CLN-06 | PARTIAL | Now drained on shutdown via `WaitGroup`, but still **unbounded** (one goroutine per event) and authevent still uses `context.Background()` — no worker pool. |

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

- [ ] **SEC-07** 🟠 🔁 — **`signedurl` reads `os.Getenv` at call time, bypassing the secret manager.**
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

- [ ] **SEC-09** 🔴 🔁 — **Federation sets `email_verified` from email presence, not the claim.**
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

- [ ] **SEC-17** 🟠 🔁 — **Email-change / verification OTPs stored plaintext & compared with `!=`.**
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

- [ ] **SEC-23** 🔴 ✅ — **Access-token denylist is never populated (read-but-empty).**
  [`internal/platform/jwt/jwt.go:73`](../../internal/platform/jwt/jwt.go#L73) declares
  `var JTIChecker` and `ValidateToken` reads it ([`:731`](../../internal/platform/jwt/jwt.go#L731)),
  but **nothing ever assigns `JTIChecker`** and no revoke/logout/password-change/session
  path calls `cache.DenyJTI` for an access-token `jti` (the only `DenyJTI` caller is the
  separate `dpop:` keyspace). The Redis denylist (`v1.0.0.md` §2 Tokens, ✅) does nothing.
  **Fix:** wire `jwt.JTIChecker = cache.IsJTIDenied` at startup and call `cache.DenyJTI(jti, ttl)`
  on revoke / logout / password & session revocation.

- [ ] **SEC-24** 🔴 ✅ — **`POST /oauth/revoke` does not revoke access tokens.**
  `oauth` `service_token.go` `Revoke` only revokes refresh tokens; for access tokens it
  comments "we cannot revoke them server-side" and no-ops. Combined with SEC-23 the issued
  access JWT stays valid until expiry after a revoke. **Fix:** denylist the access `jti`.

- [ ] **SEC-25** 🔴 ✅ — **`client_secret_jwt` (RFC 7523) is broken — verifies against the bcrypt hash.**
  [`internal/oauth/authentication.go:115`](../../internal/oauth/authentication.go#L115)
  uses `client.SecretHash` (the bcrypt hash) as the HMAC key. The client signs with the
  plaintext secret, so HS256 verification can never match — the auth method is non-functional.
  **Fix:** store/retrieve a symmetric key (or the raw secret for these clients) and HMAC with it.

- [ ] **SEC-26** 🔴 ✅ — **Per-client `allowed_scopes` is never enforced.**
  `AllowedScopes` exists on the model ([`client/model_client.go:66`](../../internal/client/model_client.go#L66))
  but is referenced nowhere in `oauth` issuance — requested scopes flow unchecked into the
  authorize challenge, auth code, and all token grants → scope escalation.
  **Fix:** intersect/validate requested scope against `client.AllowedScopes` in authorize + every grant.

- [ ] **SEC-27** 🟠 ✅ — **`/oauth/userinfo` over-discloses PII regardless of granted scope.**
  `oauth/handler_userinfo.go` always returns email/email_verified/phone/name/picture; a token
  with only `openid` still gets full PII. **Fix:** filter claims by the token's `scope`
  (`email`→email, `profile`→name/picture, `phone`→phone).

- [ ] **SEC-28** 🟠 ✅ — **`sub` is inconsistent between id_token and userinfo.**
  id_token uses `userIdentity.Sub` (`service_token.go`) but userinfo returns
  `user.UserUUID.String()` (`handler_userinfo.go`) → same user, different `sub`, breaking
  OIDC clients that key on `sub`. **Fix:** use the identity `sub` consistently.

- [ ] **SEC-29** 🔴 ✅ — **Magic-link / forgot / reset tokens stored plaintext + matched by raw equality.**
  `authn/service_magic_link.go` and `service_forgot_password.go` persist `UserToken{Token: token}`
  in the clear and look them up with `Where("token = ?", token)`. Refresh tokens / auth codes /
  SMS OTP are SHA-256-hashed at rest; these email-bearer secrets are not → a DB read is account
  takeover. **Fix:** hash at rest (SHA-256) and look up by hash, like refresh tokens.

- [ ] **SEC-30** 🟠 ✅ — **Logout swallows `RevokeAllSessions` error.**
  `authn/service_login.go` `Logout` returns `nil` even when revoke fails → reports success on a
  failed revoke (and does not denylist the access `jti`). **Fix:** propagate the error; denylist jti.

- [ ] **SEC-31** 🟠 ✅ — **SMS-OTP login: wrong code is not consumed and the compare is non-constant-time.**
  `authn/service_sms_login.go` `VerifyOTP` does not invalidate the OTP record on a wrong guess
  (only a coarse per-phone rate limit) and compares with `otpRecord.OTPHash != expectedHash`.
  **Fix:** track per-OTP attempt count + invalidate after N failures; constant-time compare.

- [ ] **SEC-32** 🟠 ✅ — **`rand.Read` errors discarded for JTI / secure-ID generation.**
  [`internal/platform/jwt/jwt.go:38`](../../internal/platform/jwt/jwt.go#L38) (`GenerateSecureID`)
  and [`:143`](../../internal/platform/jwt/jwt.go#L143) (`generateSecureJTI`) do `_, _ = rand.Read(...)`
  → predictable/zeroed IDs on RNG failure (contrast `crypto/rand.go` which propagates).
  **Fix:** check and return/panic on the error. (Same class as SEC-20, different files.)

- [ ] **SEC-33** 🟠 ✅ — **Bcrypt 72-byte silent truncation vs. 128-char max password.**
  Password policy allows `MaxLength: 128` ([`security/password_policy.go`](../../internal/platform/security/password_policy.go))
  but bcrypt (`security/hash.go`) truncates at 72 bytes (same for `HashClientSecret`) → the tail
  is ignored. **Fix:** SHA-256 pre-hash before bcrypt, or cap effective length at 72 with a clear error.

- [ ] **SEC-34** 🟠 ✅ — **`__Host-`/`__Secure-` cookie prefixes break when `COOKIE_SECURE=false`.**
  [`internal/platform/cookie/cookie.go`](../../internal/platform/cookie/cookie.go) uses the prefixed
  names unconditionally but sets `Secure` from config; browsers reject prefixed cookies without
  `Secure`, silently breaking auth (and `SameSite=None`+insecure is invalid).
  **Fix:** force `Secure=true` whenever a prefixed name is used; reject `SameSite=None` without Secure.

- [ ] **SEC-35** 🟠 ✅ — **PII redaction over-redacts audit free-text (data loss).**
  [`internal/platform/logging/pii_handler.go:~132`](../../internal/platform/logging/pii_handler.go#L132)
  `RedactString` replaces the **entire** string if it merely *contains* a PII keyword as a substring.
  Applied to every audit event ([`authevent/service_event.go:~147`](../../internal/authevent/service_event.go#L147)),
  benign descriptions ("user updated email preferences", "token refresh succeeded") become
  `[REDACTED]` → legitimate audit detail is destroyed. **Fix:** redact structured field *keys* /
  value-shaped patterns (regex), not free-text substring scans.

- [ ] **SEC-36** 🔴 ✅ — **Role/permission change does not revoke sessions or tokens; cache flush is global.**
  `iam` `service_role.go` / `service_permission.go` call only `cacheInvalidator.InvalidateAllUsers(ctx)`
  on role/permission edits — no `RevokeByUserID`/family revocation, so existing access/refresh tokens
  remain valid until expiry (contradicts `v1.0.0.md` §6 "session revoked on permission change", ✅).
  `InvalidateAllUsers` also blows away every tenant's cache on any single edit (thundering herd /
  cross-tenant blast radius). **Fix:** revoke affected users' token families (or bump a token-version
  claim checked at validation); scope invalidation to affected users only.

---

## 2. Architecture & Convention Drift

Structural debt that diverges from [code-structure.md](../contributing/code-structure.md).

- [ ] **ARC-01** 🟠 🔁 — **`deps.go` used as a dumping ground.**
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

- [ ] **ARC-03** 🟡 🔁 — **Structs/logic in the wrong files.**
  Mapping logic + `json.Unmarshal` in [`internal/user/types.go`](../../internal/user/types.go)
  (DTO-only file); DTO structs in `authn/validation_*.go` (belong in `types.go`);
  json-tagged config types in
  [`internal/idp/service_federation.go:647`](../../internal/idp/service_federation.go#L647).

- [ ] **ARC-04** 🔴 🔁 — **Platform-purity violations: `platform/*` imports `internal/<domain>` (15 imports).**
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

- [ ] **ARC-07** 🟡 🔁 — **`tenant` services leak `*gorm.DB` into the service layer.**
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

- [ ] **DUP-02** 🟠 🔁 — **`foundation.go` ~25-line alias/wrapper block copy-pasted ~13×**
  (incl. a lossy `NewBaseRepository(db any)` `db.(*gorm.DB)` shim).
  **Fix:** export the helpers directly from `platform/database` / `platform/pagination`.

- [x] **DUP-03** 🟠 ✅ — **`noopAuthEventService` duplicated verbatim in 4 packages.**
  [`iam/foundation.go:48`](../../internal/iam/foundation.go#L48),
  [`client/foundation.go:24`](../../internal/client/foundation.go#L24),
  [`user/foundation.go:28`](../../internal/user/foundation.go#L28),
  [`idp/foundation.go:52`](../../internal/idp/foundation.go#L52).
  **Fix:** `authevent.NoopService()`.

- [ ] **DUP-04** 🟠 🔁 — **Client authentication duplicated 4–6× (and divergent).**
  [`oauth/service_token_exchange.go:154`](../../internal/oauth/service_token_exchange.go#L154),
  [`service_device.go:329`](../../internal/oauth/service_device.go#L329), `service_ciba.go`,
  [`service_par.go:205`](../../internal/oauth/service_par.go#L205), `service_token.go`.
  Public clients are authenticated inconsistently across them. Grant-check helper
  (`clientSupportsGrant`/`hasGrant`/`clientHasGrant`) is byte-identical in 3 places.
  **Fix:** one shared `authenticateClient` + one grant-check helper.

- [ ] **DUP-05** 🟠 🔁 — **`authn.LoginPublic` ≈ `Login` (~200 dup lines)** and
  **`generateTokenResponse` copy-pasted 4×**
  ([`authn/service_login.go:515`](../../internal/authn/service_login.go#L515),
  `service_magic_link.go`, `service_register.go`, `user/service_account.go`).
  **Fix:** extract a shared `authenticate(...)` and one token-response builder.

- [ ] **DUP-06** 🟠 🔁 — **`hasTenantAccess` loop duplicated 9× in one file**
  ([`internal/user/service_user.go`](../../internal/user/service_user.go));
  `findDefaultRole` / `loadPolicy` / `recordPasswordHistory` duplicated across `user` & `authn`.
  **Fix:** extract `userHasTenantAccess(...)`; share the policy helpers.

- [ ] **DUP-07** 🟠 🔁 — **secpolicy: 7 config get/update handlers + services are pure copy-paste.**
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

- [ ] **CON-01** 🟠 🔁 — **Default page size is both 10 and 20.**
  `pagination.ParseQuery`→10 ([`query.go:19`](../../internal/platform/pagination/query.go#L19)),
  `database.normalizePagination`→20
  ([`base_repository.go:28`](../../internal/platform/database/base_repository.go#L28)),
  plus inline `=10`/`=20` clamps in many repos. Same package can return different defaults.
  **Fix:** one source of truth; delete inline clamps.

- [ ] **CON-02** 🟡 🔁 — **Search casing differs**: `ILIKE` vs `LOWER(col) LIKE` vs bare `LIKE`
  across packages. **Fix:** shared `applyILike(q, col, *val)` helper.

- [x] **CON-03** 🟡 ✅ — **`errors.Is(err, gorm.ErrRecordNotFound)` vs `err == gorm.ErrRecordNotFound`**
  mixed (repos vs services). **Fix:** standardize on `errors.Is`.

- [ ] **CON-04** 🟡 🔁 — **`apperror.NewNotFound` vs `NewNotFoundWithReason`** used arbitrarily;
  decode-error messages come in 4 spellings ("Invalid JSON format" / "Invalid request" /
  "Invalid request body" / "Invalid JSON"). **Fix:** convention per case + a
  `resp.BadRequestBody(w)` helper.

- [ ] **CON-05** 🟡 🔁 — **Bool query parsing**: `v == "true"` (silently drops `"1"`/`"TRUE"`)
  vs `strconv.ParseBool`. **Fix:** standardize on `strconv.ParseBool`.

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

- [ ] **CLN-01** 🟡 🔁 — **Dead injected deps / stubs.**
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

- [ ] **CLN-02** 🟡 🔁 — **Swallowed errors** beyond the security ones:
  member duplicate check ([`tenant/service_member.go:112`](../../internal/tenant/service_member.go#L112)),
  password-expiry update ([`authn/service_login.go:511`](../../internal/authn/service_login.go#L511)),
  metadata marshal ([`setup/service_setup.go:145`](../../internal/setup/service_setup.go#L145)),
  MFA user-state updates (`mfa/service_webauthn.go`, `service_mfa.go`).

- [ ] **CLN-03** 🟡 🔁 — **Globals / `init()` side-effects / test-seam indirection.**
  bcrypt in `init()` + rate-limiter no-ops-when-nil + dropped `ctx`
  ([`platform/security/security.go`](../../internal/platform/security/security.go));
  global signing-key state ([`platform/jwt/jwt.go:43`](../../internal/platform/jwt/jwt.go#L43));
  global config vars ([`platform/config/config.go:11`](../../internal/platform/config/config.go#L11));
  `var Fn = fn` indirection across ~9 platform files.
  **Fix:** encapsulate in injected structs; prefer interfaces over mutable function vars.

- [ ] **CLN-04** 🟡 🔁 — **Unpopulated DTO fields / misleading docs.**
  `TenantResponseDTO.Metadata` never set ([`tenant/types.go:20`](../../internal/tenant/types.go#L20));
  `toUserResponseDTO` drops `Tenant.DisplayName`
  ([`user/handler_user.go:564`](../../internal/user/handler_user.go#L564));
  stray "Validate ..." comments in [`branding/types.go`](../../internal/branding/types.go);
  stale "see access.go" comment ([`tenant/deps.go:41`](../../internal/tenant/deps.go#L41)).

- [ ] **CLN-05** 🟡 🔁 — **Magic values.** token `ExpiresIn: 3600` + scope
  `"openid profile email"` hardcoded in 5 places; invite TTL `72*time.Hour` duplicated
  ([`invite/service_invite.go:103`](../../internal/invite/service_invite.go#L103));
  webhook `>= 300` success threshold + uncapped backoff
  ([`webhook/deliver.go:81`](../../internal/webhook/deliver.go#L81));
  hardcoded `"active"` ([`webhook/repository_endpoint.go:125`](../../internal/webhook/repository_endpoint.go#L125));
  hardcoded ports/limits in `server/rest.go`/`grpc.go`/`router.go`.

- [ ] **CLN-06** 🟡 🔁 — **Fire-and-forget goroutines detached from shutdown.**
  [`authevent/service_event.go:156`](../../internal/authevent/service_event.go#L156) and
  [`webhook/dispatcher.go:36`](../../internal/webhook/dispatcher.go#L36) use
  `context.Background()` with no WaitGroup/worker-pool → abandoned on shutdown, unbounded.
  **Fix:** server-scoped context + bounded worker pool.

- [x] **CLN-07** 🟡 ✅ — **~30 untracked `*.out` coverage files in the repo root.**
  Not committed, just clutter. **Fix:** add to `.gitignore` / remove; route coverage to a
  build dir.

---

## 6. Feature Completeness & Spec Compliance

Features marked ✅ in [`v1.0.0.md`](../releases/v1.0.0.md) that are stubbed, unwired,
or behave differently than the spec/checklist claims. These are *correctness-of-claim*
gaps rather than classic bugs — either finish the wiring or downgrade the checklist to 🔨.

### Dead / unwired runners

- [ ] **FC-01** 🔴 — **Automatic key-rotation runner is dead code.**
  `StartKeyRotationRunner` ([`platform/runner/key_rotation.go:14`](../../internal/platform/runner/key_rotation.go#L14))
  has **zero callers**; `cmd/server/workers.go` only starts the auth-event retention runner.
  Keys never rotate, which also makes multi-key JWKS (`v1.0.0.md` §7) effectively single-key.
  **Fix:** start it in `startBackgroundWorkers` with a configurable period.

- [ ] **FC-02** 🟠 — **Secret hot-reload runner is dead code.**
  `StartSecretRefreshRunner` ([`platform/runner/secret_refresh.go:19`](../../internal/platform/runner/secret_refresh.go#L19))
  has zero callers; secrets load once at `config.Init()`. "Secret refresh / hot-reload without
  restart" (`v1.0.0.md` §8, ✅) is non-functional. **Fix:** launch it from `startBackgroundWorkers`.

### MFA / step-up enforcement

- [ ] **FC-03** 🔴 — **Per-tenant MFA policy is never enforced at login.**
  `MFAService.IsMFARequired`/`UserHasMFA` exist ([`mfa/service_mfa.go`](../../internal/mfa/service_mfa.go))
  but no `authn`/`oauth` login path calls them; login issues full tokens without ever
  challenging for a second factor. **Fix:** after password auth, check `IsMFARequired(poolID)`
  / user factors and require an MFA challenge before issuing final tokens.

- [ ] **FC-04** 🟠 — **Step-up authentication is never enforced on any sensitive op.**
  `IssueStepUpChallenge`/`VerifyStepUp` are routed but no caller requires a fresh `acr=2`
  before secret rotation, MFA reset, account deletion, etc. **Fix:** add guards on sensitive
  routes that require a recent step-up acr.

- [ ] **FC-05** 🟠 — **`acr`/`amr` claims are fabricated, not derived from the actual auth.**
  All login paths hardcode `amr:["pwd"], acr:"1"` ([`authn/token_helper.go:32`](../../internal/authn/token_helper.go#L32),
  [`oauth/service_token.go:767`](../../internal/oauth/service_token.go#L767)); acr/amr are absent
  from access tokens (only added to id_token), step-up issues a token with empty options
  (`_ = amr`, `mfa/service_mfa.go:567`), and SMS login mislabels itself as `pwd` instead of `sms`.
  RS-side step-up checks therefore can't work. **Fix:** thread real `amr`/`acr` from each flow
  into id_token + access token.

- [ ] **FC-18** 🟡 — **TOTP / backup codes lack replay protection within the validity window.**
  `mfa/service_mfa.go` uses `totp.Validate` + `UpdateLastUsed` but never records/compares the
  consumed time-step, so a code is reusable for ~30-60s across calls. **Fix:** persist last-accepted
  step and reject codes ≤ it. (Backup codes are correctly one-time via `MarkUsed`.)

- [ ] **FC-19** 🟡 — **SMS "cost guard" is only a per-phone rate limit.**
  `authn/service_sms_login.go` has no spend/budget cap or global daily ceiling, contrary to the
  "rate-limit + cost guard" claim. **Fix:** add a per-tenant/global SMS send budget with a hard cap.

### Tenancy / federation / sessions

- [ ] **FC-06** 🟠 — **API keys cannot authenticate a request — no API-key middleware exists.**
  Full CRUD + API/permission scoping + `FindByKeyHash`/`FindByKeyPrefix` exist
  ([`client/repository_api_key.go`](../../internal/client/repository_api_key.go)) but there is **no**
  middleware reading an inbound key, so a created key can't call the API. **Fix:** add API-key auth
  middleware that resolves key → scopes and populates the auth context.

- [ ] **FC-07** 🟠 — **Idle / absolute session timeout is dead code.**
  `SessionValidationMiddleware` ([`platform/middleware/session_middleware.go:24`](../../internal/platform/middleware/session_middleware.go#L24))
  is never mounted in `router.go`, and it is opt-in via a client-supplied `X-Session-ID` header,
  so `IdleTimeoutSeconds`/`AbsoluteExpiresAt` are never enforced on real traffic (`v1.0.0.md` §6, ✅).
  **Fix:** mount it on authenticated route groups and derive the session id from the auth context/cookie.

- [ ] **FC-08** 🟠 — **Concurrent-session limit is enforced only on password login.**
  `EnforceConcurrentLimit` is called in `authn/service_login.go` but not in `oauth/service_token.go`,
  SMS login, or federation. **Fix:** enforce at the common session-creation point used by all login paths.

- [ ] **FC-09** 🟠 — **Per-tenant feature flags have zero consumers.**
  `tenant/service_setting.go` stores/gets/updates `feature_flags` JSON but nothing reads them to
  gate behavior. **Fix:** add a `feature.Enabled(ctx, tenantID, key)` helper and gate features, or mark 🔨.

- [ ] **FC-10** 🟠 — **Named social connectors don't all work via the generic-OIDC path.**
  [`idp/service_federation.go:535`](../../internal/idp/service_federation.go#L535) relies on
  `oidc.NewProvider(issuer)` discovery; GitHub has no OIDC discovery, Apple needs client-secret-JWT,
  and `gitlab` isn't even a valid provider type ([`validation_provider.go:22`](../../internal/idp/validation_provider.go#L22)).
  **Fix:** add provider-specific connectors, or correct the doc to "generic OIDC + OAuth2".

- [ ] **FC-11** 🟠 — **Home-realm discovery only matches `IDPTypeSocial` providers.**
  [`idp/service_federation.go:504`](../../internal/idp/service_federation.go#L504) skips non-social
  providers, so enterprise OIDC realms configured with `EmailDomains` are never matched by HRD.
  **Fix:** match on presence of `EmailDomains` regardless of provider type.

- [ ] **FC-12** 🔴 — **GDPR account deletion is deactivation only.**
  [`user/service_account.go:~230`](../../internal/user/service_account.go#L230) `DeleteAccount`
  only sets `status="deleted"`; email/phone/profile PII remain and tokens/sessions are not revoked.
  "Right to erasure" (`v1.0.0.md` §1, ✅) requires actual erasure/anonymization.
  **Fix:** anonymize/erase PII columns (or hard-delete per retention) and revoke all tokens/sessions.

- [ ] **FC-13** 🟠 — **Tenant deletion "retention" is unbacked.**
  `tenant` `DeleteCascade` soft-deletes child models but there is no retention window, purge, or
  retention runner (`v1.0.0.md` §5 "cascade / soft-delete + retention", ✅). **Fix:** add a retention/
  purge runner or drop the retention claim.

### Auth flows

- [ ] **FC-14** 🟠 — **Force-password-change is not enforced.**
  `authn/service_login.go` sets `RequirePasswordChange` but still issues full access/refresh/id
  tokens + a session; nothing blocks normal API use until the password is changed.
  **Fix:** issue a restricted/short-lived token (or block protected routes) until reset.

- [ ] **FC-15** 🟠 — **Email login is broken — only username is matched.**
  `Login`/`LoginPublic` resolve the user via `FindByUsername(usernameOrEmail)` only, so logging in
  with an email fails unless username == email (`v1.0.0.md` §1 "username / email + password", ✅).
  **Fix:** add a `FindByUsernameOrEmail` lookup.

- [ ] **FC-16** 🟡 — **Common-password blocklist is a 10-entry substring list.**
  [`platform/security/password_policy.go:~95`](../../internal/platform/security/password_policy.go#L95)
  has ~10 hardcoded substrings matched with `strings.Contains` — both too weak (real lists are 10k–100k)
  and prone to false positives in long passphrases. **Fix:** load an embedded HIBP top-N list; pick
  exact-match vs substring deliberately.

- [ ] **FC-17** 🟠 — **`/oauth/token` is missing the tight per-endpoint IP rate limit.**
  [`server/router.go:136`](../../internal/server/router.go#L136) applies `authRateLimit` to
  register/login/forgot/reset only; `/oauth/token` (mounted at `:158`) gets only the global 100/min,
  contrary to `v1.0.0.md` §10. **Fix:** add `/oauth/token` to the `authRateLimit` group.

### Spec-wording mismatches

- [ ] **FC-20** 🟡 — **"Strict CSP for login/consent pages" describes pages that don't exist.**
  No server-rendered login/consent HTML exists (html/template is used only for email/SMS bodies);
  the single global CSP in `security_middleware.go` applies to JSON responses. **Fix:** clarify the
  checklist (external SPA) or add a per-route nonce-based CSP if pages will be rendered.

- [ ] **FC-21** 🟡 — **Append-only audit blocks UPDATE only, not DELETE.**
  [`migration/056_auth_events_append_only.go`](../../internal/platform/database/migration/056_auth_events_append_only.go)
  installs a `DO INSTEAD NOTHING` rule on UPDATE only; DELETE is intentionally left open for the
  retention runner — contradicting the literal "no UPDATE/DELETE" claim (`v1.0.0.md` §12).
  **Fix:** reword to "no UPDATEs; DELETE only via retention", or gate DELETE behind a separate role.

---

## 7. Observability, CI & Operations

- [ ] **OPS-01** 🟠 — **`/metrics` is on the internal API port, not a dedicated management port.**
  [`server/router.go:45`](../../internal/server/router.go#L45) mounts `/metrics` on the internal
  API router; only `:8080`/`:8081` exist (`rest.go`). No separate management listener (`v1.0.0.md` §11
  "Prometheus /metrics on mgmt port"). **Fix:** bind `/metrics` (and arguably `/openapi.json`) to a
  separate management listener, or correct the checklist to "internal port".

- [ ] **OPS-02** 🟠 — **JWT spans use `context.Background()` → orphaned, not trace-correlated.**
  [`platform/jwt/jwt.go:248`](../../internal/platform/jwt/jwt.go#L248) (and `:390,:523,:663`) start
  spans from `context.Background()`; the generate/validate funcs take no `ctx`, so JWT spans are never
  children of the request span (`v1.0.0.md` §11 + §22 "context.Background audit"). **Fix:** thread `ctx`
  through the JWT signatures and pass it to `Start`.

- [ ] **OPS-03** 🟠 — **"Single canonical Config struct (no package globals)" is inaccurate.**
  [`platform/config/config.go:15`](../../internal/platform/config/config.go#L15) declares ~60 exported
  package-level `var`s populated by `Init()`; `GetConfig()` merely copies them into a duplicate struct,
  and the codebase reads `config.X` globals everywhere — including secrets in mutable globals
  (`v1.0.0.md` §18, ✅; overlaps `CLN-03`). **Fix:** make `Config` the injected single source and delete
  the globals, or drop the "no package globals" claim.

- [ ] **OPS-04** 🟠 — **`gosec` excludes G104 (and G101/G102/G103/G124/G304); runs `@latest`.**
  [`.github/workflows/ci.yml:88`](../../.github/workflows/ci.yml#L88) — "gosec clean" (`v1.0.0.md` §22)
  is partly because unhandled-error and other rules are disabled, on a non-pinned version.
  **Fix:** narrow the exclude list (justify each `#nosec` inline), pin a gosec version.

- [ ] **OPS-05** 🟡 — **No explicit `go vet` / `staticcheck` step in CI.**
  Neither appears in `.github/workflows/` or `Makefile`; reliance is on golangci-lint defaults and no
  committed `.golangci.yml` was found to confirm staticcheck is enabled (`v1.0.0.md` §22).
  **Fix:** add explicit `go vet ./...` + enable staticcheck in a committed `.golangci.yml`.

- [ ] **OPS-06** 🟠 — **Security scanners are advisory-only (`continue-on-error: true`).**
  [`.github/workflows/security.yml`](../../.github/workflows/security.yml) sets `continue-on-error: true`
  on Semgrep, Snyk, **and Gitleaks**, so a committed secret or high-severity vuln never fails the
  pipeline (`v1.0.0.md` §22 lists these as gates). **Fix:** drop `continue-on-error` for gitleaks (and
  gate on the SARIF result) so leaks fail CI.

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

1. **Token revocation that doesn't revoke (`SEC-23`, `SEC-24`, `SEC-36`)** — denylist is
   never populated, `/oauth/revoke` and permission changes leave access tokens live. Top priority.
2. **Plaintext bearer secrets (`SEC-29`)** + **broken `client_secret_jwt` (`SEC-25`)** +
   **unenforced `allowed_scopes` (`SEC-26`)** — exploitable / spec-breaking today.
3. **Dead runners (`FC-01`, `FC-02`)** and **unenforced MFA / session timeout
   (`FC-03`, `FC-07`)** — ✅ features that silently do nothing; either wire or mark 🔨.
4. **Re-opened residuals** — `SEC-07`, `SEC-09`, `SEC-17`, `ARC-07`, `DUP-05`, `CON-02`,
   `CLN-03` (see the Re-Audit table). `SEC-09` (federated account takeover) first.
5. **GDPR / claim-accuracy (`FC-12`, `SEC-27`, `SEC-28`, `FC-14..17`)** — correctness of the
   public contract.
6. **CI/observability hygiene (`OPS-01..06`)** — cheap, raises the floor on everything else.

---

*Original audit: end-to-end scan; items once marked ⚠️ have now been re-read directly.
2026-06-01 re-audit: verified every `[x]` in §1–§5 against current code and audited the
✅ features in `docs/releases/v1.0.0.md`. 🔁 = re-opened (fix partial / not done). New
findings filed in §1 (`SEC-23+`), §6 (`FC-*`), §7 (`OPS-*`).*

# maintainerd-auth — Code Quality Assessment

**Repository:** `github.com/maintainerd/maintainerd-auth`
**Assessment type:** File-by-file quality review across the dimensions requested: best practice, industry standard, security compliance, architecture, file/folder naming, symbol naming, data leakage, tenant isolation, and migrations/seeders (with attention to the per-tenant system-record model).
**Method:** Direct source reading. Every claim below — positive or negative — cites the specific file and, where useful, the exact code. This is an assessment of *quality*, not a bug list; it grades what exists and flags what would raise the grade.
**Verification pass:** A second source-reading pass was performed after the initial assessment to confirm every finding. Three initial findings (A-01, L-02, SD-01) were **retracted** after direct verification contradicted them; one new finding (S-07) was added from the verification pass.
**Not covered:** No compilation, test execution, linting, or dynamic analysis was performed (out of scope per your instruction). A handful of packages (CIBA internals, token-exchange internals, SAML request signing, the seeder permission catalog in full) were sampled rather than read line-by-line.

---

## Overall grade: A− (strong, production-oriented; a few systemic hardening gaps hold it back from A/A+)

This is one of the better-structured Go services of its kind. The clean-architecture layering is real and consistently enforced, the naming discipline is close to exemplary, the database schema is sophisticated, and the security fundamentals are implemented correctly rather than merely claimed. What separates it from a top-tier grade is a small number of *systemic* issues where safety depends on developer discipline rather than being enforced by the type system or the database — most importantly, tenant isolation is a per-query convention rather than a structural guarantee.

| Dimension | Grade | One-line summary |
|---|---|---|
| Architecture | A | Clean layering, one-way deps, consumer-interface pattern applied correctly including in `oauth`; webhook behavior misplaced. |
| File/folder naming | A+ | Role-first convention applied with near-perfect consistency across every package. |
| Symbol naming | A | Idiomatic Go; a `FindBy*`/`Get*` inconsistency and some brittle error-string matching. |
| Security compliance | A− | Correct crypto, PKCE, DPoP, revocation, SSRF defense; gaps in setup auth, default binding, and RSA key floor. |
| Data leakage | A− | Passwords never serialize, secrets never logged; 2 handlers leak raw error strings to clients. |
| Tenant isolation | B+ | Works and is tested, but enforced per-query; base repo exposes non-scoped finders. |
| Migrations | A | Advisory-locked, versioned, create-only with a documented freeze policy; numbering gaps. |
| Seeders | A+ | Single idempotent per-tenant source of truth; both bootstrap and admin paths are fully transactional. |
| Per-tenant system records | A | `is_system` scoped per tenant with per-tenant uniqueness; cleanly modeled. |

---

# 1. Architecture — Grade A

**What is done well (verified):**

- **Strict layered flow with interface boundaries.** Handlers → services → repositories → database, each depending only on the layer below through interfaces. Confirmed by the `deps.go` files in every feature package that declare the consumed interfaces, and by the composition root in `internal/app` (`repositories.go`, `services.go`, `application.go`) that wires concrete implementations.
- **Honest package taxonomy.** `cmd/server` is a thin bootstrap (`main.go` delegates to `run()`, with `bootstrap.go`, `telemetry.go`, `logging.go`, `workers.go`). `internal/app` is the composition root. `internal/server` owns transport (`rest.go`, `grpc.go`, `router.go`, `health.go`). `internal/platform/*` holds reusable infrastructure. `internal/<feature>` holds domain logic. This is the right shape and it is followed.
- **Transaction boundaries via `WithTx`.** Repositories support `WithTx(tx)` so services can compose multi-repo operations atomically. The refresh-token rotation path (`oauth/service_token.go` `exchangeRefreshToken`) and admin tenant creation (`tenant/service_tenant.go:231`) both use `s.db.Transaction(...)` with tx-scoped repos correctly.
- **Generics for shared CRUD.** `internal/platform/database/base_repository.go` provides a type-parameterized `BaseRepository[T]` that concrete repos embed, avoiding hand-written boilerplate while allowing domain-specific queries.
- **`oauth` is correctly decoupled from sibling domains.** `internal/oauth/deps.go` defines local projection types (`oauth.Tenant`, `oauth.Client`, `oauth.User`, `oauth.IdentityProvider`, etc.) and consumer-repository interfaces that reference only those local types, so the `oauth` package never imports sibling feature packages directly. The file's own comment states the intent explicitly: *"the oauth package declares them so it does not import those domains directly."* Concrete implementations are wired via adapters in `internal/app/`. This is the correct consumer-interface + composition-root pattern, and it is applied consistently.

**What would raise the grade (verified issues):**

- **A-01 · Oversized units.** `internal/idp/service_federation.go` is ~1,500 lines spanning OIDC exchange, JIT provisioning, identity linking, and discovery; `internal/mfa/service_mfa.go` and `internal/oauth/service_token.go` are similarly large. No correctness issue, but these are the files where the next subtle bug hides. *Fix: split by responsibility within the same package following the existing role-first naming.*
- **A-02 · Behavior in the composition root.** `internal/app/webhook_delivery.go` (404 lines, verified) contains full delivery behavior — `deliverToWebhooks`, `attemptOnce`, `doDeliveryRequest`, `computeWebhookSignature`, `buildDeliveryBody`, retry/replay helpers — which contradicts the documented rule that `internal/app` holds only wiring. *Fix: move behavior into `internal/webhook`, leave only wiring in `internal/app`.*

---

# 2. File & Folder Naming — Grade A+

This is the strongest dimension. The role-first convention documented in `docs/contributing/code-structure.md` is applied with near-perfect consistency across every feature package. Verified by listing all `.go` files in `authn`, `oauth`, `user`, `iam`, `tenant`, `mfa`, `idp`, `client`, and `webhook`:

- `handler_<name>.go`, `service_<name>.go`, `repository_<name>.go`, `model_<name>.go`, `validation_<name>.go`, `types.go`, `routes.go`, `deps.go`, `foundation.go` — used uniformly.
- gRPC handlers are consistently suffixed `_grpc.go` (`handler_user_grpc.go`, `handler_provider_grpc.go`).
- Domain-specific support files use honest descriptive names rather than being force-fit into a role prefix: `policy_evaluator.go`, `token_invalidator.go`, `redirect_match.go`, `config_mapping.go` (client); `saml_provider.go`, `http_client.go`, `encryption.go` (idp); `dispatcher.go`, `deliver.go`, `signer.go`, `payload.go`, `security_url.go` (webhook); `unit_of_work.go`, `retention.go`, `defaults_setting.go` (tenant); `sweeper.go`, `cleanup_runner.go`, `pkce_policy.go`, `startup_signing_key.go` (oauth).
- Migrations use `NNN_create_<table>_table.go`; seeders use `NNN_<entity>.go`. Both ordered and scannable.

**Only nitpick (A+ stands):** the numeric prefixes in `internal/setup/seeder/` have gaps (`001`, `003`, `004` — no `002`) and duplicate numbers across different entities (`011_security_setting.go` and `011_sms_template.go`). Harmless — Go ignores filenames for compilation — but the duplicate numeric prefixes on *different* entities slightly undercut the "ordered" intent. Consider unique numbers per entity.

---

# 3. Symbol Naming (functions, variables, types) — Grade A

**What is done well (verified):**

- Interfaces and implementations follow the idiomatic Go split: exported `FooService` interface + unexported `fooService` struct + `NewFooService(...)` constructor (seen throughout, e.g. `OAuthTokenService`/`oauthTokenService`/`NewOAuthTokenService`).
- Boolean helpers read naturally (`IsExpired`, `IsRevoked`, `clientHasGrant`, `isUnsafeWebhookIP`).
- Repository methods are descriptive and self-documenting about their scoping (`FindByUUIDAndTenantID`, `FindByTenantProviderAndSub`, `RevokeByUserAndClient`, `MarkUsed`).
- Package-local unexported helpers are appropriately scoped and commented.

**What would raise the grade (verified issues):**

- **N-01 · `FindBy*` vs `Get*` inconsistency.** The codebase mixes `FindBy*` (returns nil on miss) and `Get*` in different packages for conceptually identical read operations. *Fix: pick one verb convention — recommend `FindBy*` returns `(nil, nil)` on miss, `Get*` returns an error on miss — and document it in `code-structure.md`.*
- **N-02 · Brittle error-string matching (`internal/authn/handler_register.go` lines ~84–90 and ~180–186).** The register handler classifies errors by comparing `err.Error()` against literal strings (`"password is too weak"`, `"password must contain at least one uppercase letter"`, …). This couples control flow to human-readable message text: any wording change silently breaks the classification, and it duplicates the same block twice (once for the public route, once for the internal route). *Fix: use typed/sentinel errors (e.g. `var ErrWeakPassword = errors.New(...)`) checked with `errors.Is`, defined once in the security package.*

---

# 4. Security Compliance — Grade A−

**What is done well (verified line-by-line):**

- **Password hashing:** bcrypt cost 12 default (`security/hash.go`), with a policy-driven self-describing envelope supporting Argon2/scrypt/PBKDF2, plus HIBP breach checking (`security/hibp.go`) and a common-password list. Constant-time comparison.
- **JWT:** RS256/PS256 only; the verification key callback explicitly type-asserts the signing method and rejects anything else (`platform/jwt/jwt.go` ~line 1007) — correct algorithm-confusion defense. KID-based lookup supports rotation grace.
- **Token revocation / logout:** `jwt.SetJTIChecker` is wired at startup (`cmd/server/bootstrap.go:97`) and invoked inside `ValidateToken`. It is a two-tier check (Redis denylist → authoritative DB revocation store) and **fails closed** on error (`return nil, fmt.Errorf("token revocation check failed: %w", checkErr)`). Revoked access tokens are rejected before natural expiry.
- **PKCE:** mandatory on the auth-code grant; `ValidatePKCEChallenge` accepts S256 only, rejects plain.
- **DPoP (RFC 9449):** proof validation, `cnf.jkt` binding, and the §8 server-nonce gate implemented.
- **Client authentication:** four methods (`client_secret_basic/post`, `private_key_jwt`, `client_secret_jwt`) with `WithValidMethods` restricting algorithms, and `iss==sub==client_id` + `aud`/`exp` assertion validation.
- **Webhook SSRF (`internal/webhook/security_url.go`):** HTTPS-only, DNS resolution, rejects loopback/private/link-local/unspecified/multicast IPs, re-validated on every delivery **and** each redirect hop via `CheckRedirect`. This is a complete, correct implementation.
- **gRPC TLS:** refuses to start in production without certs; TLS 1.2 minimum; optional `RequireAndVerifyClientCert` mTLS.
- **Security headers (`middleware/security_middleware.go`):** full set — HSTS with preload, a restrictive CSP, `X-Frame-Options: DENY`, `nosniff`, `Referrer-Policy`, `Permissions-Policy`.
- **CORS:** wildcard never combined with credentials; `Vary: Origin` set.
- **SQL injection:** parameterized queries throughout; ORDER BY guarded by a column allowlist (`base_repository.go` `sanitizeOrder`); no string-interpolated SQL in query paths.
- **Login enumeration/timing:** generic `"invalid credentials"`; password verification runs even when the user is nil.

**What would raise the grade (verified issues):**

- **S-01 · Authorization-code redemption is not atomic** (`oauth/service_token.go`). Read→check-`Used`→`MarkUsed` with no transaction/lock allows a concurrent double-redemption of one code. `MarkUsed` itself (`oauth/repository_auth_code.go:57`) is an unconditional `UPDATE ... SET used=true` with no `WHERE used = false` guard, so two concurrent requests both passing the `Used` check before either write leaves no database-level backstop. The refresh path right below it is transactional; the auth-code path should match. *Highest-severity item. Fix: wrap the read+mark in a transaction with a conditional update — `UPDATE ... SET used=true, used_at=NOW() WHERE oauth_authorization_code_id=? AND used=false`, check `RowsAffected`, and reject if 0.*
- **S-02 · Setup endpoints have no authentication** (`setup/routes.go`). `create_tenant`/`create_admin` are unauthenticated; the idempotency lock in `ensureSetupOpen()` only fires after the system tenant reaches `status="active"` — it does not protect the window before the first records exist. Any network-reachable caller can take over first-boot. *Fix: gate with a `SETUP_BOOTSTRAP_TOKEN` env var checked in the handler or middleware.*
- **S-03 · Management port binds on all interfaces by default** (`docker-compose.yml`). Port mapping is `"8080:8080"` (binds `0.0.0.0`), while docs describe it as "VPN-only." *Fix: bind internal port to loopback by default — `"127.0.0.1:8080:8080"`.*
- **S-04 · No RSA key-size floor for client `private_key_jwt` keys** (`oauth/authentication.go` `findClientJWK`). The function reconstructs an `rsa.PublicKey` from the client's JWKS without checking `pubKey.N.BitLen()`. A client can register a weak (e.g. 1024-bit) key. *Fix: reject `pubKey.N.BitLen() < 2048`.*
- **S-05 · Upstream OIDC nonce check fails open on empty input** (`idp/service_federation.go:910`, `if nonce != ""`). Not currently exploitable (the broker always sets a nonce) but a fragile default for a security check. *Fix: fail closed — require a non-empty nonce and reject if absent.*
- **S-06 · Insecure cookie defaults in `.env.example`** (`COOKIE_SECURE=false`, `COOKIE_SAMESITE=lax`) contradict both the secure code default and the CSRF middleware's stated `SameSite=Strict` assumption. *Fix: ship secure example defaults; make the three sources consistent.*
- **S-07 · `ensureSetupOpen()` conflates DB error with "not found"** (`setup/service_setup.go:594`). The function calls `s.tenantRepo.FindSystem()` and on error returns `nil` (setup is treated as "still open"). A transient database error during the lock check therefore silently allows setup endpoints to proceed rather than failing safely. *Fix: return the error explicitly — `if err != nil { return err }` — so the handler rejects the request on DB failure.*

---

# 5. Data Leakage — Grade A−

**What is done well (verified):**

- **Passwords never serialize out.** `users.password` is tagged `json:"-"` in `internal/user/model_user.go` — it cannot leak through any JSON response by construction.
- **Secrets are never logged.** The secret-manager (`config/secret_manager.go`) logs only the provider name, region/prefix, and — at load — the key name and byte length (`"Loaded secret", "key", key, "bytes", len(secret)`), never the value. No `slog` call anywhere logs a password, token, or client secret value (verified by grep across `internal/`).
- **TOTP secrets encrypted at rest.** `mfa/service_mfa.go` calls `crypto.EncryptAtRest` (AES-256-GCM) before persisting; the plaintext base32 is never stored.
- **Internal errors don't leak through the main error path.** `response.HandleServiceError` logs internal/untyped errors server-side and returns only a generic `fallbackMsg` to the client; typed errors return controlled, intentional messages.
- **PII redaction handler exists** (`platform/logging/pii_handler.go`) for structured logs.
- **`handler_magic_link.go` does NOT leak errors to clients.** `err.Error()` at line 152 is embedded in the `Details` field of a server-side `SecurityEvent` log (never sent over the wire); the HTTP response is the fixed string `"Invalid or expired magic link"`.

**What would raise the grade (verified issues):**

- **L-01 · Two handlers return raw `err.Error()` as the HTTP client message**, bypassing `HandleServiceError`: `internal/authn/handler_account_link.go:43` and `internal/user/handler_user_consent.go:95` (`resp.Error(w, http.StatusBadRequest, err.Error())`). Low sensitivity (validation context) but inconsistent with the safe pattern and a potential vector for internal-detail disclosure if an unexpected error reaches them. *Fix: route these through `HandleServiceError` or map to a controlled message.*

---

# 6. Tenant Isolation — Grade B+

This is the dimension most worth your attention. It **works** and it is **tested**, but it is enforced by convention rather than structure, which is why it is the lowest grade here despite functioning correctly today.

**What is done well (verified):**

- **83 tenant-scoped finder methods** across the concrete repositories (`FindByUUIDAndTenantID`, `FindByIDAndTenantID`, `FindByTenantProviderAndSub`, etc.) — the intended access pattern carries the tenant predicate.
- **A cross-domain regression suite** (`tests/integration/tenant_isolation_test.go`) asserts, via SQL-matching mocks, that the primary UUID-addressed finders emit a `tenant_id` predicate — so dropping the predicate from one of those finders breaks the build.
- **Tenant-scoped uniqueness at the schema level**: `uq_users_tenant_email`, `uq_users_tenant_username`, `uq_services_tenant_name`, `uq_roles_tenant_name` — all `(tenant_id, name) WHERE deleted_at IS NULL`. The same email/username can exist independently per tenant ("separate worlds"), which is correct for this model.
- **Auth-code exchange double-checks tenant binding** (`oauth/service_token.go`: `if authCode.TenantID == 0 || authCode.TenantID != client.TenantID`).

**What would raise the grade (verified structural risk):**

- **T-01 · The generic `BaseRepository[T]` exposes non-tenant-scoped finders** (`FindByUUID`, `FindByID`, `UpdateByUUID`, `UpdateByID`, `DeleteByUUID`, `DeleteByID` in `base_repository.go`). Every concrete repo embeds these, so a bare `FindByUUID(id)` with **no tenant predicate** is always one call away on a tenant-scoped table, sitting right beside the safe `FindByUUIDAndTenantID`. At least one concrete repo already exposes a bare `FindByUUID` (`oauth/repository_oauth_authorize_request.go:36`). Nothing at the type level prevents a future handler from calling the unsafe method on a tenant-owned aggregate — the only thing catching it is the regression test, and that test only covers the finders it enumerates. **This is the single most important quality gap in the codebase.**
  - *Strongest fix:* add **PostgreSQL Row-Level Security** so tenant isolation becomes a database invariant. Enable RLS on every tenant-scoped table with a policy `USING (tenant_id = current_setting('app.tenant_id', true)::bigint)`, and `SET LOCAL app.tenant_id = <caller>` at the start of each request-scoped transaction. The codebase already uses the `set_config(...)` session-GUC pattern (for `maintainerd.allow_auth_event_delete` in `tenant`/`authevent` repos), so the mechanism is familiar. RLS is additive — keep the application predicates too.
  - *Cheaper mitigation if RLS is deferred:* make the generic mutating/finding methods on `BaseRepository[T]` unexported or remove them for tenant-scoped aggregates, forcing callers through explicit tenant-scoped methods; and expand the isolation regression suite to assert every finder on every tenant-scoped table includes the predicate.

---

# 7. Migrations — Grade A

**What is done well (verified):**

- **Concurrency-safe.** `internal/platform/runner/migration.go` takes a PostgreSQL session-level advisory lock (`advisoryLockKey = 7316949`) so only one pod runs migrations when many start at once.
- **Versioned and ordered.** Each migration is a `{Version, Fn}` entry in an explicit ordered slice; the version string is written to `schema_migrations` and documented as immutable once applied.
- **Documented create-only policy with a clear freeze point.** The runner comment and `docs/contributing/database-migrations.md` state: migrations are CREATE-ONLY (one canonical create per table, edited in place pre-release); new tables append at the bottom; the policy freezes to forward-only at first production deployment. This is a coherent, deliberately chosen strategy.
- **Schema quality is high.** Sampled tables (`tenants`, `users`, `oauth_refresh_tokens`) show partial unique indexes, CHECK constraints (`chk_users_status`, `chk_oauth_refresh_revoked`), GIN indexes on JSONB, composite indexes leading with `tenant_id`, `TIMESTAMPTZ` throughout, FK indexing, and even PL/pgSQL denormalization triggers (`sync_totp_flag`, `sync_webauthn_flag`) with documented rationale. Audit FKs are attached in a deferred `DO $$` block once `users` exists — a thoughtful ordering solution.

**What would raise the grade (verified issues):**

- **M-01 · Migration numbering gaps** (no `002`, `040`, `045`, `069`). Harmless with the versioned runner, but they look like accidental deletions. *Fix: add a one-line note in `database-migrations.md` that gaps are intentional/historical, or renumber while still pre-release.*
- **M-02 · No down/rollback migrations** (by design). Legitimate given the forward-only policy, but operators need it stated explicitly. *Fix: document "roll back via backup/PITR, not down-migrations" in the runbook, and add backup-before-migrate guidance.*

---

# 8. Seeders — Grade A+

**What is done well (verified):**

- **Single idempotent source of truth.** `internal/setup/seeder/seed_tenant.go` `SeedTenant(db, tenantID)` seeds the *entire* per-tenant baseline (service, API, permissions, control policy, IdP, clients, URIs, roles, role-permissions, registration flows, email/SMS templates, security settings, branding, tenant settings, event types) and is reused by **both** first-run bootstrap and admin-side tenant creation. This is exactly the right design — new tenants get the identical baseline the system tenant got.
- **Both seeding paths are transactional.** The bootstrap path (`setup/service_setup.go:155`) wraps the entire `CreateTenant` flow — including `setupRunSeeders(tx, "v0.1.0")` — in `db.Transaction(...)`, passing the live `tx` all the way down through `runner.RunSeeders` → `seeder.RunAll` → `seeder.SeedTenant`. Admin-side tenant creation (`tenant/service_tenant.go:231`) does the same. Both paths are all-or-nothing; partial seeding is not possible.
- **Idempotent.** Each seeder checks existence before creating (`Where(...).First(...)` + `errors.Is(err, gorm.ErrRecordNotFound)`), so re-running is safe.
- **Correctly tenant-scoped.** Every seeded record carries `TenantID`; comments explicitly note that even the "auth" service is per-tenant.

**Minor issue (grade stands):**

- **SD-01 · Version string duplication.** `seed_tenant.go` hardcodes `seedServiceVersion = "v0.1.0"` while `runner/seeder.go`'s `RunAll` ignores the `appVersion` parameter (`_ string`). The service version stamped on records does not reflect the actual running version. *Fix: thread a single version constant through both paths instead of hardcoding it in the seeder.*

---

# 9. Per-Tenant System Records — Grade A

You specifically flagged that you have a system record per tenant. This is modeled correctly.

**What is done well (verified):**

- **`is_system` records are created per tenant, not globally.** The seeded "auth" service (`001_service.go:32`), "auth" API (`003_api.go:36`), and the `registered` + `super-admin` roles (`008_role.go`) all set `IsSystem: true` **and** `TenantID: tenantID`. Every tenant — including the system tenant and every admin-created tenant — gets its own copy via the shared `SeedTenant` path.
- **Per-tenant uniqueness is enforced at the schema level.** `uq_services_tenant_name` and `uq_roles_tenant_name` are `(tenant_id, name) WHERE deleted_at IS NULL`, so each tenant has exactly one `auth` service and one `super-admin` role, and different tenants' system records never collide.
- **The system *tenant* itself is a guarded singleton.** `001_create_tenants_table.go` has `uq_tenants_single_system` — a partial unique index on `(is_system) WHERE is_system = TRUE AND deleted_at IS NULL` — guaranteeing at most one live system tenant (the root). System *records* are per-tenant, but the system *tenant* is unique.
- **Seeding branches on system-vs-regular tenant where appropriate.** `004_permission.go:50-54` reads the tenant's `is_system` flag and seeds a different permission set accordingly, so the root tenant can receive elevated/control permissions that regular tenants do not.

**Minor observation (grade stands):** because the per-tenant uniqueness constraints are `WHERE deleted_at IS NULL`, a soft-deleted system role/service could in principle coexist with a live one of the same name — which is intended (soft-delete + re-create), but worth being aware of when reasoning about "exactly one system record per tenant."

---

# Prioritized improvement list (quality-raising, in order)

1. **T-01 — Add PostgreSQL Row-Level Security** (tenant isolation as a DB invariant). Highest structural value; removes the "one bare `FindByUUID` away from a leak" risk. `L`.
2. **S-01 — Make authorization-code redemption atomic** (conditional `UPDATE ... WHERE used=false`, check `RowsAffected`). `M`.
3. **S-02 + S-03 + S-07 — Close the first-boot exposure** (setup bootstrap token; loopback-bind the management port; fix `ensureSetupOpen` to return error on DB failure). Do together. `M`.
4. **S-04, S-05 — Crypto/federation hardening** (RSA key-size floor in `findClientJWK`; fail-closed OIDC nonce check). `S` each.
5. **L-01 — Route the 2 raw-error handlers through `HandleServiceError`** (`handler_account_link.go:43`, `handler_user_consent.go:95`). `S`.
6. **S-06 — Fix cookie defaults** across `.env.example`, config default, and the CSRF comment so all three agree. `S`.
7. **A-02 — Move webhook delivery behavior out of `internal/app`** into `internal/webhook`. `M`.
8. **N-01, N-02 — Standardize `FindBy*`/`Get*`; replace error-string matching with sentinel/typed errors.** `M`.
9. **A-01 — Split the largest service files by responsibility** (`service_federation.go`, `service_mfa.go`, `service_token.go`). `M`.
10. **M-01, M-02, SD-01 — Documentation/consistency: migration gaps, forward-only policy, seed version source.** `S`.

---

# Appendix — Dimensions verified as already meeting the bar (no action)

- `oauth` domain coupling: consumer-interface + local-projection pattern applied correctly; oauth never imports sibling packages.
- Password serialization (`json:"-"`), secret logging (key-name + byte-length only), TOTP encryption at rest.
- Magic-link error handling: `err.Error()` goes to a server-side security log, not the HTTP response.
- Seeder transactionality: both bootstrap (setup service `CreateTenant`) and admin paths wrap `SeedTenant` in `db.Transaction(...)`.
- JWT algorithm pinning, token revocation denylist (fail-closed, two-tier), PKCE-mandatory, DPoP binding.
- Webhook SSRF defense (complete, re-checked on redirects) and per-delivery timeouts.
- gRPC production TLS enforcement.
- Security headers, safe CORS, parameterized SQL, ORDER BY allowlist.
- File/folder naming convention (near-perfect adherence).
- Seeder idempotency and single-source-of-truth design.
- Per-tenant system-record scoping and per-tenant uniqueness; system-tenant singleton guard.
- Schema quality (partial indexes, CHECK constraints, GIN JSONB, TIMESTAMPTZ, denormalization triggers, migration advisory lock).

*End of assessment.*

# Test Coverage Plan

Last updated: 2026-06-01 (re-audit) | Total coverage: **69.2%** (`-coverpkg=./internal/...`)

This document tracks the testing status for every package, defines the e2e and integration tests that need to be written, and serves as the single source of truth for test coverage planning.

> **2026-06-01 re-audit note.** A verification pass re-ran coverage and read the actual
> test files. Two things changed materially since the last update:
> 1. **Unit coverage is higher than recorded** — total is **69.2%**, and `authn`, `oauth`,
>    `platform/middleware`, and `platform/cache` are no longer critical gaps.
> 2. **The e2e and integration suites exist but largely do not meet the standard.** The
>    12 e2e files drive **inline stub handlers**, not the real app router; the 5 repository
>    integration files use **go-sqlmock against ad-hoc fake structs**, not the real
>    repositories on a live DB. Their per-flow / per-repository checklist boxes are
>    therefore left **unchecked** below until rewritten. Only the cache + middleware
>    integration tests are genuinely real.

---

## Test Standards Reference

All test code must follow [docs/contributing/testing.md](testing.md). Before writing any test, read:
- Handler test checklist (9-step flow: auth → authz → params → body → validation → deps → business rules → service → success)
- Service test coverage (success branches + every error path)
- Validation test rules (one sub-test per rule)
- Mock conventions (`mock_test.go`, `mock_repos_test.go`, function-field pattern)

---

## Overall Coverage Snapshot (go test -cover)

| Tier | Packages | Coverage |
|------|----------|----------|
| **Unit** (all tested) | 32 packages with tests | **69.2%** total |
| **Unit** (0% or no tests) | `mfa`, `platform/sms`, `platform/dpop`, `shared`, `platform/model`, `platform/templates/emailtemplate` | 0% |
| **Integration** | `tests/integration/` | 9 files — **cache + middleware are real**; the 5 repository files are sqlmock-based (do **not** meet the live-DB standard) |
| **E2E** | `tests/e2e/` | 12 files exist but are **stub-handler scaffolding**, not real contract tests (see E2E section) |

---

## Domain Package Coverage (ranked)

> Values updated against a measured `-coverpkg=./internal/...` profile run on 2026-06-01.
> ⬆ = moved up a tier since the last plan.

### Tier 1 — Good (≥85%)

| Package | Coverage | Handler Tests | Service Tests | Validation Tests | Cross-cutting |
|---------|----------|---------------|---------------|-----------------|--------------|
| `platform/apperror` | 100.0% | — | — | — | — |
| `platform/ptr` | 100.0% | — | — | — | — |
| `platform/valid` | 100.0% | — | — | — | — |
| `platform/middleware` ⬆ | **99.0%** | — | — | — | `auth_chain_test.go` (integration) |
| `platform/signedurl` | 98.6% | — | — | — | — |
| `platform/response` | 97.5% | — | — | — | — |
| `platform/cache` ⬆ | **96.5%** | — | — | — | redis (miniredis) integration |
| `platform/cookie` | 95.6% | — | — | — | — |
| `setup` | 95.5% | yes | yes | yes | — |
| `platform/config` | 92.6% | — | — | — | — |
| `authn` ⬆ | **88.3%** | yes (4 handlers) | yes (4 services) | yes (4 DTOs) | — |
| `platform/jsonutil` | 87.5% | — | — | — | — |

### Tier 2 — Solid (75-84%)

| Package | Coverage | Handler Tests | Service Tests | Validation Tests | Cross-cutting |
|---------|----------|---------------|---------------|-----------------|--------------|
| `branding` | 84.5% | yes (4 handlers) | yes (4 services) | yes (4 DTOs) | — |
| `tenant` | 84.0% | yes (2 handlers) | yes (3 services) | yes (2 DTOs) | `isolation_test.go` |
| `iam` | 83.6% | yes (5 handlers) | yes (5 services) | yes (5 DTOs) | — |
| `client` | 79.1% | yes (2 handlers) | yes (2 services) | yes (2 DTOs) | — |
| `platform/crypto` | 76.7% | — | — | — | — |
| `oauth` ⬆ | **75.2%** | yes (5 handlers) | yes (3 services) | yes (2 DTOs) | — |

### Tier 3 — Needs Work (55-74%)

| Package | Coverage | Handler Tests | Service Tests | Validation Tests | Notes |
|---------|----------|---------------|---------------|-----------------|-------|
| `secpolicy` | 73.3% | yes (2 handlers) | yes (2 services) | yes (2 DTOs) | |
| `invite` | 72.7% | yes | yes | yes | |
| `notifier` | 72.0% | yes (2 handlers) | yes (2 services) | yes (2 DTOs) | |
| `user` | 70.2% | yes (3 handlers) | yes (3 services) | yes (3 DTOs) | |
| `platform/jwt` | 69.6% | — | — | — | spans untested; `ctx` not threaded (OPS-02) |
| `authevent` | 69.5% | yes | yes | yes | |
| `platform/security` | 68.1% | — | — | — | |
| `webhook` | 65.2% | yes | yes | yes | SSRF guard + signer need explicit tests |
| `idp` | 62.1% | yes (2 handlers) | yes (2 services) | yes (2 DTOs) | federation branches thin |

### Tier 4 — Critical Need (<55%)

| Package | Coverage | Handler Tests | Service Tests | Validation Tests | Notes |
|---------|----------|---------------|---------------|-----------------|-------|
| `platform/logging` | 41.2% | — | — | — | `RedactString` logic untested (SEC-35) |
| `platform/telemetry` | 37.3% | — | — | — | |
| `platform/email` | 17.9% | — | — | — | provider factory untested |
| `platform/pagination` | 11.1% | — | — | — | core helper, very low |

### Tier 5 — Zero Coverage

| Package | Coverage | Status |
|---------|----------|--------|
| `mfa` | 0.0% | **No tests at all** — confirmed still 0% (recent commits did NOT add mfa tests). High priority: TOTP, WebAuthn, step-up, backup codes, SMS OTP. |
| `platform/sms` | 0.0% | No tests |
| `platform/dpop` | 0.0% | No tests — DPoP proof parsing/validation untested |
| `shared` | — | No test files |
| `platform/model` | — | No test files |
| `platform/templates/emailtemplate` | — | No test files |

### Excluded from Codecov (wiring/infra/generated)

| Package | Reason |
|---------|--------|
| `cmd/server` | Bootstrap wiring (now has `logging_test.go` — reconcile exclusion) |
| `internal/app` | DI composition root (now has `app_test.go` at ~10.8% — reconcile exclusion) |
| `internal/server` | Transport routing |
| `platform/database` | GORM-dependent |
| `platform/database/migration` | Schema-only |
| `platform/runner` | Process lifecycle |
| `platform/gen` | Generated protobuf |
| `setup/seeder` | DB seeding |

---

## E2E Test Plan

**12 e2e files now exist, but they do not test the application.** `newE2ERouter()` returns a
bare `chi.NewRouter()` and each test registers **inline stub `http.HandlerFunc`s** that return
hand-written canned responses. They never wire the real app router, handlers, or services, so
success/validation/not-found assertions are tautological (they assert against the stub the test
itself wrote). Examples found: `multi_tenant_test.go` only asserts 401 on stub routes (no real
isolation); `invite_test.go` has a "no body returns bad request" sub-test that asserts **201**.

**These must be rewritten to drive the real router before any box below is honestly checked.**

| Flow | Test File | Status | Priority |
|------|-----------|--------|----------|
| Login flow | `tests/e2e/auth_test.go` | ⚠️ File exists (stub-only) — rewrite | **P0** |
| Register flow | `tests/e2e/register_test.go` | ⚠️ Stub-only — rewrite | **P0** |
| Forgot password flow | `tests/e2e/forgot_password_test.go` | ⚠️ Stub-only — rewrite | **P0** |
| Reset password flow | `tests/e2e/reset_password_test.go` | ⚠️ Stub-only — rewrite | **P0** |
| OAuth authorize code grant | `tests/e2e/oauth_flow_test.go` | ⚠️ Stub-only — rewrite | **P0** |
| Tenant CRUD | `tests/e2e/tenant_test.go` | ⚠️ Stub-only — rewrite | **P1** |
| Multi-tenant isolation | `tests/e2e/multi_tenant_test.go` | ⚠️ Stub-only (only tests 401) — rewrite | **P1** |
| User profile | `tests/e2e/user_test.go` | ⚠️ Stub-only — rewrite | **P1** |
| IAM roles & permissions | `tests/e2e/iam_test.go` | ⚠️ Stub-only — rewrite | **P1** |
| Invite flow | `tests/e2e/invite_test.go` | ⚠️ Stub-only (misleading 201 assert) — rewrite | **P2** |
| Branding | `tests/e2e/branding_test.go` | ⚠️ Stub-only — rewrite | **P2** |
| Client management | `tests/e2e/client_test.go` | ⚠️ Stub-only — rewrite | **P2** |
| MFA flow | `tests/e2e/mfa_test.go` | ❌ Not written | **P2** |
| Webhook delivery | `tests/e2e/webhook_test.go` | ❌ Not written | **P2** |
| IDP federation | `tests/e2e/idp_test.go` | ❌ Not written | **P2** |

### E2E test coverage checklist per flow

These boxes are intentionally **unchecked**: the current stub-handler files do not satisfy any
of them against the real application. Each rewritten flow must drive the real router and cover:

- [ ] **Real router wiring** — test calls the actual `buildPublicRouter`/`buildInternalRouter`, not an inline stub
- [ ] **Authentication check** — request without token returns 401 (on a real protected route)
- [ ] **Authorization check** — request with insufficient permissions returns 403 (NOT covered by any current file)
- [ ] **Success path** — full round-trip through real handler + service + DB
- [ ] **Validation errors** — invalid/missing required fields return 400 via the real DTO `Validate()`
- [ ] **Not found** — request for non-existent resource returns 404 from the real store
- [ ] **Tenant isolation** — cross-tenant access returns 403/404 (NOT covered; `multi_tenant_test.go` only checks 401)

---

## Integration Test Plan

**9 integration files now exist; realness is split:**

- ✅ **Genuinely real (meet the standard):** `cache/user_context_test.go` and
  `cache/jti_denylist_test.go` exercise the real `cache` package via miniredis;
  `middleware/auth_chain_test.go` exercises the real middleware chain (JWTAuth, Permission,
  CSRF→403, content-type→415, CORS, session, security headers).
- ⚠️ **Not real (repository tier):** all 5 repository tests use **go-sqlmock with ad-hoc local
  structs** (e.g. a hand-rolled `testUser` with its own `TableName()`), not the project's real
  repository code. `testing.md` requires "real repository against live DB". No live Postgres,
  no real pagination/soft-delete/cascade/tenant-scoping is exercised.

| Layer | Test File | Status | Priority |
|-------|-----------|--------|----------|
| Cache — user context (Redis) | `tests/integration/cache/user_context_test.go` | ✅ Real | **P0** |
| Cache — JTI denylist (Redis) | `tests/integration/cache/jti_denylist_test.go` | ✅ Real | **P1** |
| Middleware chain | `tests/integration/middleware/auth_chain_test.go` | ✅ Real | **P0** |
| Tenant repository | `tests/integration/repository/tenant_test.go` | ⚠️ sqlmock + fake struct — rewrite for live DB | **P0** |
| User repository | `tests/integration/repository/user_test.go` | ⚠️ sqlmock — rewrite for live DB | **P0** |
| OAuth repository | `tests/integration/repository/oauth_test.go` | ⚠️ sqlmock — rewrite for live DB | **P0** |
| IAM repository | `tests/integration/repository/iam_test.go` | ⚠️ sqlmock — rewrite for live DB | **P1** |
| Client repository | `tests/integration/repository/client_test.go` | ⚠️ sqlmock — rewrite for live DB | **P1** |
| Auth event repository | `tests/integration/repository/authevent_test.go` | ❌ Not written | **P1** |

### Integration test coverage per repository

Unchecked: the sqlmock-based repository tests do not satisfy any of these against the real
repository code on a live DB (testcontainers-go). Each rewritten repository test must cover:

- [ ] **Uses the real repository** — calls the project's `NewXRepository(db)`, not a fake struct
- [ ] **Live DB** — runs against testcontainers Postgres, not sqlmock
- [ ] **CRUD operations** — create, read, update, delete against real DB
- [ ] **Pagination** — page/limit/sort with real data
- [ ] **Transaction boundaries** — commit on success, rollback on failure (real rollback, not a Begin/Commit assertion)
- [ ] **Soft delete** — verify record is excluded from normal queries (current tests issue hard `DELETE`)
- [ ] **Cascade behavior** — tenant delete cascades to related models
- [ ] **Tenant scoping** — queries scoped to tenant_id return correct subset

---

## New unit-test checklist (added 2026-06-01)

Packages still missing the standard unit-test trio or below the ≥80% target:

- [ ] **`mfa`** (0%) — full 9-step handler tests, per-branch service tests, per-rule validation tests. Cover: TOTP enroll/verify (+ replay within window, FC-18), WebAuthn register/assert (+ sign-count regression, SEC-15), step-up issue/verify (subject binding SEC-14), backup-code one-time use, SMS OTP, MFA reset, rate-limit/lockout paths (SEC-16).
- [ ] **`platform/dpop`** (0%) — unit tests for DPoP proof parsing, `htu`/`htm`/`iat` validation, jti binding, replay denylist.
- [ ] **`platform/sms`** (0%) — table-driven tests for the SMS sender + provider factory selection.
- [ ] **`platform/pagination`** (11.1%) — tests for `ParseQuery` defaults/clamps and the `DefaultPageSize` source of truth (CON-01).
- [ ] **`platform/email`** (17.9%) — provider factory + each provider adapter (SES/SendGrid/Postmark/Mailgun/Resend/SMTP) smoke tests.
- [ ] **`platform/telemetry`** (37.3%) — meter/tracer provider init, build-info gauge.
- [ ] **`platform/logging`** (41.2%) — **`RedactString` behavior** (currently over-redacts free-text, SEC-35) + PII handler field redaction.
- [ ] **`platform/jwt`** (69.6%) — span/`ctx` propagation (OPS-02), `JTIChecker` denylist read path (SEC-23), `rand.Read` error handling (SEC-32), multi-key JWKS / rotation.
- [ ] **`webhook`** (65.2%) — explicit tests for the SSRF URL guard (loopback/private/redirect re-check, SEC-18) and HMAC signer + replay window.
- [ ] **`platform/templates/emailtemplate`, `platform/model`, `shared`** — at minimum smoke/render tests where branching logic exists.

---

## Remediation Priority

### Immediate (P0)

1. **Rewrite the e2e suite to drive the real router.** The 12 existing files are stub-handler
   scaffolding and give false confidence — no real auth/authz/validation/isolation is tested.
2. **Rewrite the 5 repository integration tests against a live DB (testcontainers).** Current
   sqlmock + fake-struct tests don't exercise the real repos, soft-delete, cascade, or scoping.
3. **`mfa` (0%)** — security-critical and entirely untested; start with the 9-step handler + service trees.
4. **`platform/pagination` (11.1%)** — core helper used everywhere, near-zero coverage.

### Short-term (P1)

5. **`platform/dpop` (0%)** and **`platform/sms` (0%)** — platform packages without tests.
6. **`platform/email` (17.9%)**, **`platform/telemetry` (37.3%)**, **`platform/logging` (41.2%)** — raise to target; logging includes the SEC-35 redaction logic.
7. **`idp` (62.1%)** — federation branches (JIT merge, HRD, OIDC verify) are thin and tied to SEC-09/FC-10/FC-11.
8. **`authevent_test.go` integration** — the one repository integration file with no test at all.

### Nice-to-have (P2)

9. Remaining e2e flows with no file yet: **mfa, webhook, idp**.
10. `webhook` SSRF + signer explicit tests; `platform/jwt` span/ctx + denylist tests.
11. Reconcile the Codecov exclusion list with `internal/app` / `cmd/server` now having tests.

---

## What "Done" Looks Like Per Package

A package is considered **fully tested** when:

1. Every handler method follows the [9-step checklist](testing.md#handler-test-checklist)
2. Every service method has at least one success sub-test and one sub-test per error branch
3. Every `Validate()` method has one sub-test per validation rule
4. Cross-cutting invariants (isolation, nil safety) are tested in `<concern>_test.go`
5. Package coverage is ≥80%
6. Corresponding integration tests exist **against a live DB (not sqlmock)** if the package has repositories
7. Corresponding e2e tests exist **driving the real router (not stub handlers)** if the package exposes API endpoints

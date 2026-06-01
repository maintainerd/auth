# Test Coverage Plan

Last updated: 2026-06-01 | Total coverage: **60.0%**

This document tracks the testing status for every package, defines the e2e and integration tests that need to be written, and serves as the single source of truth for test coverage planning.

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
| **Unit** (all tested) | 30 packages with tests | 60.0% total |
| **Unit** (0% or no tests) | 13 packages | 0% |
| **Integration** | `tests/integration/` | 0 tests (placeholder) |
| **E2E** | `tests/e2e/` | 0 tests (placeholder) |

---

## Domain Package Coverage (ranked)

### Tier 1 — Good (≥85%)

| Package | Coverage | Handler Tests | Service Tests | Validation Tests | Cross-cutting |
|---------|----------|---------------|---------------|-----------------|--------------|
| `platform/apperror` | 100.0% | — | — | — | — |
| `platform/ptr` | 100.0% | — | — | — | — |
| `platform/valid` | 100.0% | — | — | — | — |
| `platform/signedurl` | 98.6% | — | — | — | — |
| `platform/response` | 97.5% | — | — | — | — |
| `platform/cookie` | 95.6% | — | — | — | — |
| `setup` | 95.5% | yes | yes | yes | — |
| `platform/config` | 92.6% | — | — | — | — |
| `platform/jsonutil` | 87.5% | — | — | — | — |

### Tier 2 — Solid (75-84%)

| Package | Coverage | Handler Tests | Service Tests | Validation Tests | Cross-cutting |
|---------|----------|---------------|---------------|-----------------|--------------|
| `branding` | 84.5% | yes (4 handlers) | yes (4 services) | yes (4 DTOs) | — |
| `tenant` | 84.0% | yes (2 handlers) | yes (3 services) | yes (2 DTOs) | `isolation_test.go` |
| `iam` | 83.6% | yes (5 handlers) | yes (5 services) | yes (5 DTOs) | — |
| `client` | 79.1% | yes (2 handlers) | yes (2 services) | yes (2 DTOs) | — |
| `platform/crypto` | 76.7% | — | — | — | — |

### Tier 3 — Needs Work (55-74%)

| Package | Coverage | Handler Tests | Service Tests | Validation Tests | Notes |
|---------|----------|---------------|---------------|-----------------|-------|
| `secpolicy` | 73.3% | yes (2 handlers) | yes (2 services) | yes (2 DTOs) | |
| `invite` | 72.7% | yes | yes | yes | |
| `notifier` | 72.0% | yes (2 handlers) | yes (2 services) | yes (2 DTOs) | |
| `authevent` | 69.5% | yes | yes | yes | |
| `platform/jwt` | 69.6% | — | — | — | |
| `platform/security` | 68.1% | — | — | — | |
| `user` | 65.2% | yes (3 handlers) | yes (3 services) | yes (3 DTOs) | |
| `webhook` | 65.2% | yes | yes | yes | |

### Tier 4 — Critical Need (<55%)

| Package | Coverage | Handler Tests | Service Tests | Validation Tests | Notes |
|---------|----------|---------------|---------------|-----------------|-------|
| `platform/middleware` | 59.8% | — | — | — | **Critical** — middleware is security surface |
| `platform/cache` | 58.1% | — | — | — | |
| `authn` | 57.3% | yes (4 handlers) | yes (4 services) | yes (4 DTOs) | **Critical** — login/register/forgot/reset |
| `idp` | 55.2% | yes (2 handlers) | yes (2 services) | yes (2 DTOs) | |
| `platform/logging` | 41.2% | — | — | — | |
| `oauth` | 41.1% | yes (5 handlers) | yes (3 services) | yes (2 DTOs) | **Critical** — OAuth 2.0 / OIDC core |
| `platform/telemetry` | 37.3% | — | — | — | |
| `platform/email` | 17.9% | — | — | — | |
| `platform/pagination` | 11.1% | — | — | — | |

### Tier 5 — Zero Coverage

| Package | Coverage | Status |
|---------|----------|--------|
| `mfa` | 0.0% | No tests at all |
| `platform/sms` | 0.0% | No tests |
| `platform/dpop` | 0.0% | No tests |
| `shared` | — | No test files |
| `platform/model` | — | No test files |
| `platform/templates/emailtemplate` | — | No test files |

### Excluded from Codecov (wiring/infra/generated)

| Package | Reason |
|---------|--------|
| `cmd/server` | Bootstrap wiring |
| `internal/app` | DI composition root |
| `internal/server` | Transport routing |
| `platform/database` | GORM-dependent |
| `platform/database/migration` | Schema-only |
| `platform/runner` | Process lifecycle |
| `platform/gen` | Generated protobuf |
| `setup/seeder` | DB seeding |

---

## E2E Test Plan

**All e2e tests are missing.** Only `tests/e2e/placeholder_test.go` exists.

| Flow | Test File | Status | HTTP Methods | Priority |
|------|-----------|--------|-------------|----------|
| Login flow | `tests/e2e/login_test.go` | ❌ Not written | POST /login | **P0** |
| Register flow | `tests/e2e/register_test.go` | ❌ Not written | POST /register | **P0** |
| Forgot password flow | `tests/e2e/forgot_password_test.go` | ❌ Not written | POST /forgot-password | **P0** |
| Reset password flow | `tests/e2e/reset_password_test.go` | ❌ Not written | POST /reset-password | **P0** |
| OAuth authorize code grant | `tests/e2e/oauth_flow_test.go` | ❌ Not written | POST /oauth/token, GET /oauth/authorize | **P0** |
| Tenant CRUD | `tests/e2e/tenant_test.go` | ❌ Not written | GET/POST/PUT/DELETE /tenants | **P1** |
| Multi-tenant isolation | `tests/e2e/multi_tenant_test.go` | ❌ Not written | Cross-tenant API calls | **P1** |
| User profile | `tests/e2e/user_test.go` | ❌ Not written | GET/PUT /user/profile | **P1** |
| IAM roles & permissions | `tests/e2e/iam_test.go` | ❌ Not written | CRUD /roles, /permissions | **P1** |
| Invite flow | `tests/e2e/invite_test.go` | ❌ Not written | POST /invite, accept flow | **P2** |
| MFA flow | `tests/e2e/mfa_test.go` | ❌ Not written | POST /mfa/* | **P2** |
| Webhook delivery | `tests/e2e/webhook_test.go` | ❌ Not written | CRUD /webhooks | **P2** |
| Branding | `tests/e2e/branding_test.go` | ❌ Not written | CRUD /branding/* | **P2** |
| IDP federation | `tests/e2e/idp_test.go` | ❌ Not written | CRUD /idp/* | **P2** |
| Client management | `tests/e2e/client_test.go` | ❌ Not written | CRUD /clients | **P2** |

### E2E test coverage checklist per flow

Each e2e test file must cover:

- [ ] **Authentication check** — request without token returns 401
- [ ] **Authorization check** — request with insufficient permissions returns 403
- [ ] **Success path** — full round-trip with valid credentials
- [ ] **Validation errors** — invalid/missing required fields return 400
- [ ] **Not found** — request for non-existent resource returns 404
- [ ] **Tenant isolation** — cross-tenant access returns 403 (for scoped resources)

---

## Integration Test Plan

**All integration tests are missing.** Only `tests/integration/repository/placeholder_test.go` exists.

| Layer | Test File | Status | Priority |
|-------|-----------|--------|----------|
| Tenant repository | `tests/integration/repository/tenant_test.go` | ❌ Not written | **P0** |
| User repository | `tests/integration/repository/user_test.go` | ❌ Not written | **P0** |
| OAuth repository | `tests/integration/repository/oauth_test.go` | ❌ Not written | **P0** |
| IAM repository | `tests/integration/repository/iam_test.go` | ❌ Not written | **P1** |
| Client repository | `tests/integration/repository/client_test.go` | ❌ Not written | **P1** |
| Auth event repository | `tests/integration/repository/authevent_test.go` | ❌ Not written | **P1** |
| Cache (Redis) | `tests/integration/cache/user_context_test.go` | ❌ Not written | **P0** |
| Cache (Redis) | `tests/integration/cache/jti_denylist_test.go` | ❌ Not written | **P1** |
| Middleware chain | `tests/integration/middleware/auth_chain_test.go` | ❌ Not written | **P0** |

### Integration test coverage per repository

Each integration test file must cover:

- [ ] **CRUD operations** — create, read, update, delete against real DB
- [ ] **Pagination** — page/limit/sort with real data
- [ ] **Transaction boundaries** — commit on success, rollback on failure
- [ ] **Soft delete** — verify record is excluded from normal queries
- [ ] **Cascade behavior** — tenant delete cascades to related models
- [ ] **Tenant scoping** — queries scoped to tenant_id return correct subset

---

## Remediation Priority

### Immediate (P0)

1. **oauth** (41.1%) — OAuth 2.0 / OIDC is the core protocol. Handler tests for token, authorize, consent, and discovery need full 9-step checklist coverage.
2. **authn** (57.3%) — Login, register, forgot/reset password are the primary user-facing flows. Service tests need error-branch expansion; handler tests need auth/authz checks added.
3. **middleware** (59.8%) — JWT, user context, and permission middleware are the security boundary. Every code path must be tested.
4. **e2e login + oauth flows** — The API contract must be verified end-to-end.

### Short-term (P1)

5. **idp** (55.2%) — Federation flows need coverage.
6. **user** (65.2%) — Profile and settings handlers.
7. **cache** (58.1%) — Redis interaction coverage.
8. **Integration tests for tenant + user repos + cache** — Foundation layer.

### Nice-to-have (P2)

9. **mfa** (0%) — Needs both unit and e2e tests.
10. **sms** (0%) / **dpop** (0%) — Platform packages without tests.
11. Remaining e2e flows (branding, IDP, webhook, client).

---

## What "Done" Looks Like Per Package

A package is considered **fully tested** when:

1. Every handler method follows the [9-step checklist](testing.md#handler-test-checklist)
2. Every service method has at least one success sub-test and one sub-test per error branch
3. Every `Validate()` method has one sub-test per validation rule
4. Cross-cutting invariants (isolation, nil safety) are tested in `<concern>_test.go`
5. Package coverage is ≥80%
6. Corresponding integration tests exist if the package has repositories
7. Corresponding e2e tests exist if the package exposes API endpoints

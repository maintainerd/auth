# Testing Guide

This document describes how to run, write, and organise tests when contributing to **Maintainerd Auth**.

---

## Table of Contents

- [Test Tiers and Placement](#test-tiers-and-placement)
- [Running Tests](#running-tests)
- [Test Layout](#test-layout)
- [Writing Unit Tests](#writing-unit-tests)
  - [Table-driven tests](#table-driven-tests)
  - [Assertions](#assertions)
- [Service Layer Tests](#service-layer-tests)
  - [Success cases](#success-cases)
  - [Error cases](#error-cases)
  - [Interaction tests](#interaction-tests)
- [Validation Tests](#validation-tests)
- [Handler (API) Layer Tests](#handler-api-layer-tests)
  - [Handler test checklist](#handler-test-checklist)
  - [POST/PUT/PATCH handlers (state-changing)](#postputpatch-handlers-state-changing)
  - [GET handlers (reads)](#get-handlers-reads)
  - [DELETE handlers](#delete-handlers)
  - [Unauthenticated routes](#unauthenticated-routes)
  - [Belongs-to-tenant checks](#belongs-to-tenant-checks)
  - [HTTP handler test helpers](#http-handler-test-helpers)
- [Middleware Tests](#middleware-tests)
- [Cross-cutting Tests](#cross-cutting-tests)
- [Mocking Strategy](#mocking-strategy)
- [Environment Variables in Tests](#environment-variables-in-tests)
- [Coverage](#coverage)
- [Integration Tests](#integration-tests)
- [End-to-End Tests](#end-to-end-tests)
- [What to Test When Adding Code](#what-to-test-when-adding-code)

---

## Test Coverage Checklist

Mark boxes as you bring each handler/service/validation/middleware up to the standard below.

### Middleware Tests

- [x] `RecoveryMiddleware` — panic recovery + normal path
- [x] `CORSMiddleware` — no-origin, wildcard, explicit match, origin-not-allowed, OPTIONS preflight
- [x] `CSRFMiddleware` — safe methods (GET/HEAD/OPTIONS/TRACE), non-safe: no-cookie, empty-cookie, missing-header, mismatch, success
- [x] `IPRateLimitMiddleware` — nil Redis, under limit, exceeds limit (429 + Retry-After), per-path keys, per-IP keys
- [x] `EnforceJSONContentType` — GET/HEAD/DELETE skip, JSON passes, charset passes, empty Content-Type (415), text/plain (415), OAuth form-encoded exemption
- [x] `SessionValidationMiddleware` — no header (pass-through), invalid UUID (400), no user (401), valid session, UnauthorizedError (401), generic error (500)
- [x] `JWTAuthMiddleware` — existing tests cover all branches
- [x] `UserContextMiddleware` — existing tests cover all branches
- [x] `PermissionMiddleware` — existing tests cover all branches
- [x] `LoggingMiddleware` — existing tests cover all branches
- [x] `SecurityHeadersMiddleware` / `SecurityContextMiddleware` / `RequestSizeLimitMiddleware` / `TimeoutMiddleware` / `IPWhitelistMiddleware` — existing tests cover all branches

---

### authn Package

#### Handler Tests (9-step checklist)

- [x] `handler_register.go` — RegisterPublic, Register, RegisterInvite, RegisterInvitePublic
- [x] `handler_login.go` — LoginPublic, Login, Logout (handler-level)
- [x] `handler_forgot_password.go` — ForgotPasswordPublic, ForgotPassword
- [x] `handler_reset_password.go` — ResetPasswordPublic, ResetPassword
- [x] `handler_magic_link.go` — SendMagicLinkPublic, SendMagicLink, VerifyMagicLink
- [x] `handler_sms_login.go` — SendOTP, VerifyOTP
- [x] `handler_email_verification.go` — SendVerificationEmailPublic, SendVerificationEmail, VerifyEmail

#### Service Tests

- [x] `service_register.go` — RegisterPublic, Register, RegisterInvite, RegisterInvitePublic (all error branches)
- [ ] `service_register.go` — true success paths (RegisterResponseDTO returned with valid tokens — currently UserIdentitySub is empty in tests)
- [x] `service_login.go` — LoginPublic, Login, GetUserByEmail (all error branches)
- [x] `service_login.go` — Logout (10 branches: empty token, nil sessionService, malformed JWT, no sub, empty sub, invalid UUID, user not found, lookup error, revoke error, success)
- [ ] `service_login.go` — checkPasswordExpiry branch (nil repo, nil PasswordChangedAt, expiry check triggered)
- [x] `service_forgot_password.go` — SendPasswordResetEmail (all error branches)
- [x] `service_reset_password.go` — ResetPassword (all error branches)
- [ ] `service_magic_link.go` — SendMagicLink, LoginWithMagicLink (all branches — rate-limit, client not found, user inactive, token expired, etc.)
- [ ] `service_sms_login.go` — SendOTP, VerifyOTP (all branches — rate-limit, FindByPhone, GenerateOTP, OTP hash mismatch, etc.)
- [ ] `service_email_verification.go` — SendVerificationEmail, VerifyEmail (all branches — user not found, already verified, token expired, etc.)
- [x] `service_session.go` — ListSessions (3), RevokeSession (4), RevokeAllSessions (2), CreateSession (2), EnforceConcurrentLimit (5), ValidateAndTouch (3)
- [ ] `service_password_policy.go` — loadPolicy, checkPasswordHistory, recordPasswordHistory standalone tests

#### Validation Tests

- [x] `validation_register.go` — RegisterRequestDTO, RegisterRequestDTO.ValidateForRegistration, RegisterQueryDTO, RegisterInviteQueryDTO
- [ ] `validation_register.go` — max-length rules (username >255, fullname >255, password >128)
- [x] `validation_login.go` — LoginRequestDTO, LoginQueryDTO
- [ ] `validation_login.go` — max-length rules (client_id >255, provider_id >255)
- [x] `validation_forgot_password.go` — ForgotPasswordRequestDTO
- [x] `validation_reset_password.go` — ResetPasswordRequestDTO, ResetPasswordQueryDTO
- [ ] `validation_reset_password.go` — max-length rules (token >500, client_id >100, etc.)
- [x] `validation_magic_link.go` — SendMagicLinkRequestDTO, VerifyMagicLinkRequestDTO
- [x] `validation_sms_login.go` — SMSLoginSendDTO, SMSLoginVerifyDTO
- [x] `validation_email_verification.go` — SendEmailVerificationRequestDTO, VerifyEmailRequestDTO

---

### oauth Package

#### Handler Tests (9-step checklist)

- [x] `handler_authorize.go` — Authorize, GetConsentChallenge, HandleConsent (existing tests cover full checklist)
- [x] `handler_token.go` — Token, Revoke, Introspect (existing tests cover full checklist)
- [ ] `handler_token.go` — DPoP proof validation path (non-nil dpopStore + valid/invalid DPoP proof)
- [x] `handler_discovery.go` — Discovery, JWKS (existing), AuthorizationServerMetadata (new)
- [x] `handler_consent.go` — ListGrants, RevokeGrant (existing tests cover full checklist)
- [x] `handler_userinfo.go` — UserInfo (existing tests cover full checklist)
- [ ] `handler_userinfo.go` — composeUserDisplayName edge cases (DisplayName whitespace, FirstName+LastName fallback, nil User)
- [x] `handler_ciba.go` — Initiate, ExchangeToken (validation), ApproveRequest, DenyRequest
- [x] `handler_device.go` — Authorize, VerifyUserCode, ExchangeDeviceToken (validation), DenyUserCode
- [x] `handler_par.go` — Push
- [x] `handler_register.go` — Register
- [x] `handler_session.go` — EndSession (GET + POST), BackchannelLogout
- [x] `handler_token_exchange.go` — Exchange (validation)

#### Service Tests

- [x] `service_authorize.go` — Authorize (17 sub-tests), GetConsentChallenge (6), HandleConsent (11)
- [x] `service_token.go` — Exchange (~26 sub-tests), Revoke (6), Introspect (8)
- [x] `service_consent.go` — ListGrants (4), RevokeGrant (5)
- [ ] `service_ciba.go` — Initiate, ExchangeToken, ApproveRequest, DenyRequest (0 tests)
- [ ] `service_device.go` — Authorize, VerifyUserCode, ExchangeToken, DenyUserCode (0 tests)
- [ ] `service_par.go` — Push, ConsumeRequestURI (0 tests)
- [ ] `service_register.go` — Register (0 tests)
- [ ] `service_session.go` — EndSession, BackchannelLogout (0 tests)
- [ ] `service_token_exchange.go` — Exchange (0 tests)
- [ ] `authentication.go` — authenticateOAuthClient: private_key_jwt + client_secret_jwt branches (0 tests)

#### Validation Tests

- [x] `validation_authorize.go` — OAuthAuthorizeRequestDTO, OAuthConsentDecisionDTO (existing + max-length tests added)
- [x] `validation_token.go` — OAuthTokenRequestDTO, OAuthRevokeRequestDTO, OAuthIntrospectRequestDTO
- [x] `validation_ciba.go` — OAuthCIBARequestDTO, OAuthCIBATokenRequestDTO
- [x] `validation_device.go` — OAuthDeviceAuthorizationRequestDTO, OAuthDeviceVerifyRequestDTO, OAuthDeviceTokenRequestDTO
- [x] `validation_par.go` — OAuthPARRequestDTO
- [x] `validation_register.go` — OAuthClientRegistrationRequestDTO
- [x] `validation_session.go` — OAuthEndSessionRequestDTO, OAuthBackchannelLogoutRequestDTO
- [x] `validation_token_exchange.go` — OAuthTokenExchangeRequestDTO

---

### E2E Tests

- [ ] `tests/e2e/login_test.go` — POST /login flow
- [ ] `tests/e2e/register_test.go` — POST /register flow
- [ ] `tests/e2e/forgot_password_test.go` — POST /forgot-password flow
- [ ] `tests/e2e/reset_password_test.go` — POST /reset-password flow
- [ ] `tests/e2e/oauth_flow_test.go` — Full OAuth authorization code grant
- [ ] `tests/e2e/tenant_test.go` — CRUD /tenants
- [ ] `tests/e2e/multi_tenant_test.go` — Cross-tenant isolation
- [ ] `tests/e2e/user_test.go` — User profile
- [ ] `tests/e2e/iam_test.go` — Roles and permissions
- [ ] `tests/e2e/invite_test.go` — Invite flow
- [ ] `tests/e2e/branding_test.go` — Branding CRUD
- [ ] `tests/e2e/client_test.go` — Client management

### Integration Tests

- [ ] `tests/integration/repository/tenant_test.go`
- [ ] `tests/integration/repository/user_test.go`
- [ ] `tests/integration/repository/oauth_test.go`
- [ ] `tests/integration/repository/iam_test.go`
- [ ] `tests/integration/repository/client_test.go`
- [ ] `tests/integration/cache/user_context_test.go`
- [ ] `tests/integration/cache/jti_denylist_test.go`
- [ ] `tests/integration/middleware/auth_chain_test.go`

---

## Test Tiers and Placement

The project has three test tiers. Know where each belongs:

| Tier | Location | Build Tag | Requires | What it verifies |
|------|----------|-----------|----------|-----------------|
| **Unit** | `internal/<domain>/` (beside source) | none (default) | nothing | Single function, handler, or behaviour in isolation. All external dependencies are mocked. |
| **Integration** | `tests/integration/<layer>/` | `integration` | Postgres + Redis running | Real repository against live DB, real cache against live Redis, middleware chain with real storage. |
| **End-to-End** | `tests/e2e/` | `e2e` | Full stack running | Real HTTP requests against the running server. Verifies full request lifecycle through middleware, handlers, services, repos, DB, and back. |

### Decision tree

```
Is the test sending an HTTP request to a running server?
├─ Yes → tests/e2e/ (build tag: e2e)
├─ No → Does it need a real Postgres or Redis?
│       ├─ Yes → tests/integration/<layer>/ (build tag: integration)
│       └─ No  → internal/<domain>/beside the source file (unit test)
```

### What goes where

- **Unit tests** (`internal/<domain>/`): Every public function. Mock everything external. Fast, no I/O, run on every `make test`.
- **Integration tests** (`tests/integration/`): Verify that repository queries work against a real schema, that cache serialization round-trips, that middleware chains compose correctly with real context resolution.
- **E2E tests** (`tests/e2e/`): Verify complete user-facing flows (login → JWT → call protected endpoint → response shape). These are the "contract" tests — if they pass, the API works.

---

## Running Tests

All commands below are available via `make` or raw `go test`.

| Goal                          | Command                                            |
| ----------------------------- | -------------------------------------------------- |
| Run all unit tests            | `make test` or `go test ./...`                     |
| Run with race detector        | `make test-race`                                   |
| Generate HTML coverage report | `make test-cover`                                  |
| Run tests for one package     | `go test ./internal/tenant/... -v`                 |
| Run a single test by name     | `go test ./internal/tenant/... -run TestTenantHandler_Update` |
| Run integration tests         | `go test -tags integration ./tests/integration/... -v` |
| Run e2e tests                 | `go test -tags e2e ./tests/e2e/... -v`             |

> **Before every pull request**, run `make test-race` to catch data races that the standard runner will miss.

---

## Test Layout

Follow the **same-package, beside-the-source-file** convention used by Ory Hydra, Casdoor, and the broader Go community.

```
internal/tenant/
├── handler_tenant.go
├── handler_tenant_test.go      ← unit test lives beside source
├── service_tenant.go
├── service_tenant_test.go      ← unit test lives beside source
├── validation_tenant.go
└── validation_tenant_test.go   ← unit test lives beside source
```

**Rules:**
- Test files use the same `package` declaration as the source file (`package tenant`, not `package tenant_test`). This lets tests access unexported helpers when needed.
- One test file per source file. Do not merge tests from multiple source files.
- Integration and e2e tests go under `tests/` with a build tag gate (see [Integration Tests](#integration-tests) and [End-to-End Tests](#end-to-end-tests)).

---

## Writing Unit Tests

### Table-driven tests

All tests use Go's table-driven pattern. This is mandatory for functions with more than two code paths.

```go
func TestGenerateOTP(t *testing.T) {
    tests := []struct {
        name    string
        length  int
        wantErr bool
    }{
        {"valid 6-digit OTP", 6, false},
        {"zero length returns error", 0, true},
        {"negative length returns error", -1, true},
    }
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            got, err := GenerateOTP(tc.length)
            if tc.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Len(t, got, tc.length)
        })
    }
}
```

### Assertions

Use **`testify`** — it is already a project dependency.

| Use | For |
|---|---|
| `assert.Equal` | Non-fatal equality checks; test continues on failure |
| `require.NoError` | Fatal guard; stop the test immediately if an error occurs |
| `require.NotNil` | Fatal guard before dereferencing a pointer |
| `assert.Contains` | Substring / slice membership checks |
| `assert.JSONEq` | JSON equality ignoring key order |
| `assert.NoError` | Verify an operation succeeded (non-fatal) |

Never use bare `t.Fatal` or `t.Error` — testify gives better output.

---

## Service Layer Tests

Service tests verify business logic in isolation. All external dependencies (repositories, user services, cache) are mocked.

### Success cases

Every service method must cover at minimum one **happy path** that exercises the full flow end-to-end:

```go
t.Run("success - creates tenant and returns DTO", func(t *testing.T) {
    repo := &mockTenantRepo{
        findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
            return nil, nil
        },
    }
    db, mock := newMockGormDB(t)
    mock.ExpectBegin()
    mock.ExpectExec(`INSERT INTO "tenants"`).WillReturnResult(sqlmock.NewResult(1, 0))
    mock.ExpectCommit()

    svc := NewTenantService(db, repo, nil)
    result, err := svc.Create(context.Background(), "acme", "Acme Corp", "desc", "active", false)
    require.NoError(t, err)
    assert.Equal(t, "acme", result.Name)
    assert.NotEqual(t, uuid.Nil, result.UUID)
    assert.NoError(t, mock.ExpectationsWereMet())
})
```

For methods with **conditional branches** (e.g. different behaviour depending on input state), add a sub-test for each branch:

```go
t.Run("success with all filters and rows covers filter+loop branches", ...)
t.Run("success with empty result set", ...)
t.Run("success when already deleted (no-op)", ...)
```

### Error cases

Service tests must cover every distinct error path the method can produce. Each error category gets its own sub-test:

| Error category | Example scenario | What to assert |
|---|---|---|
| **Not found** | Repository returns nil without error | `require.Error` + `assert.Contains(err.Error(), "not found")` |
| **Repository error** | Repository returns generic error | `require.Error` + `assert.Nil(result)` |
| **Validation error** | Input fails pre-condition check | `require.Error` + `assert.Contains(err.Error(), "validation")` |
| **Business rule error** | Operation violates a domain invariant | `require.Error` + assert specific error type or message |
| **Transaction rollback** | DB error inside a tx block | `require.Error` + `mock.ExpectationsWereMet()` verifies no commit |

```go
t.Run("not found → error", func(t *testing.T) {
    repo := &mockTenantRepo{
        findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
            return nil, nil  // found nothing
        },
    }
    svc := NewTenantService(nil, repo, nil)
    result, err := svc.GetByUUID(context.Background(), uuid.New())
    require.Error(t, err)
    assert.Nil(t, result)
})

t.Run("repo error → propagated", func(t *testing.T) {
    repo := &mockTenantRepo{
        findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
            return nil, errors.New("connection refused")
        },
    }
    svc := NewTenantService(nil, repo, nil)
    result, err := svc.GetByUUID(context.Background(), uuid.New())
    require.Error(t, err)
    assert.Nil(t, result)
})
```

### Interaction tests

When a service orchestrates multiple dependencies, verify the **sequence** of interactions — not just the final result. For example, a delete service that must fetch the tenant, check it is not a system tenant, then cascade-delete:

```go
t.Run("system tenant → error", func(t *testing.T) {
    repo := &mockTenantRepo{
        findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
            t := newTenant(1, "system")
            t.IsSystem = true
            return t, nil
        },
    }
    db, mock := newMockGormDB(t)
    mock.ExpectBegin()
    mock.ExpectRollback()  // tx must roll back, not commit

    svc := NewTenantService(db, repo, testCascadeModels())
    result, err := svc.DeleteByUUID(context.Background(), tenantUUID)
    require.Error(t, err)
    assert.Contains(t, err.Error(), "system tenant")
    assert.NoError(t, mock.ExpectationsWereMet())
})
```

---

## Validation Tests

DTO validation methods live in `validation_<name>.go` and are tested in `validation_<name>_test.go`. Cover:

1. **Valid input passes** — the happy path.
2. **Every required field missing** — one sub-test per field.
3. **Every field with format/semantic constraints** — one sub-test per invalid value class.
4. **Edge cases** — zero values, max-length strings, boundary limits.

```go
func TestTenantCreateDTO_Validate(t *testing.T) {
    tests := []struct {
        name    string
        dto     TenantCreateDTO
        wantErr string
    }{
        {"valid", TenantCreateDTO{Name: "acme", DisplayName: "Acme Corp", Description: "desc"}, ""},
        {"name required", TenantCreateDTO{Name: "", DisplayName: "Acme Corp"}, "name"},
        {"display_name required", TenantCreateDTO{Name: "acme", DisplayName: ""}, "display_name"},
        {"description too short", TenantCreateDTO{Name: "acme", DisplayName: "X", Description: "ab"}, "description"},
        {"name too long", TenantCreateDTO{Name: strings.Repeat("x", 256), DisplayName: "X", Description: "desc"}, "name"},
    }
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            err := tc.dto.Validate()
            if tc.wantErr == "" {
                require.NoError(t, err)
            } else {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tc.wantErr)
            }
        })
    }
}
```

---

## Handler (API) Layer Tests

Handler tests use `net/http/httptest` — no real server is started. Every handler method must cover the full request lifecycle through to the response status code and, for success cases, the response body.

### Handler test checklist

For **every** handler method, check each applicable category below. Not all apply to every method — skip categories that don't match the handler's logic.

The numbered order below matches the order the handler runs its checks. Write your test sub-cases in the same order so that reading the test file traces the handler's control flow.

#### 1. Authentication check

If the handler reads the user from context (caller is behind `JWTAuthMiddleware` in production), verify it returns **401** when no user is present:

```go
t.Run("no user returns 401", func(t *testing.T) {
    r := httptest.NewRequest(http.MethodDelete, "/", nil)
    r = withChiParam(r, "tenant_uuid", testResourceUUID.String())
    // deliberately no withUser() call
    w := httptest.NewRecorder()
    newTenantHandler(nil, nil).Delete(w, r)
    assert.Equal(t, http.StatusUnauthorized, w.Code)
})
```

#### 2. Authorization / permission check

If the handler enforces role-based or membership-based access (e.g. "only members of this tenant can update it"), verify it returns **403** when the user lacks the required role or membership:

```go
t.Run("not a member returns 403", func(t *testing.T) {
    ms := &mockTenantMemberService{
        isUserInTenantFn: func(int64, uuid.UUID) (bool, error) { return false, nil },
    }
    r := withUser(withChiParam(
        jsonReq(t, http.MethodPut, "/", validBody),
        "tenant_uuid", testResourceUUID.String(),
    ))
    w := httptest.NewRecorder()
    newTenantHandler(nil, ms).Update(w, r)
    assert.Equal(t, http.StatusForbidden, w.Code)
})
```

#### 3. URL parameter validation

If the handler reads chi URL parameters (e.g. `tenant_uuid`, `identifier`), test both:
- **Missing param** → 400
- **Invalid format** (e.g. invalid UUID) → 400

```go
t.Run("empty UUID param returns 400", func(t *testing.T) {
    r := httptest.NewRequest(http.MethodGet, "/", nil)
    // no chi param set
    w := httptest.NewRecorder()
    newTenantHandler(nil, nil).GetByUUID(w, r)
    assert.Equal(t, http.StatusBadRequest, w.Code)
})

t.Run("invalid UUID returns 400", func(t *testing.T) {
    r := withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "tenant_uuid", "bad")
    w := httptest.NewRecorder()
    newTenantHandler(nil, nil).GetByUUID(w, r)
    assert.Equal(t, http.StatusBadRequest, w.Code)
})
```

#### 4. Body parsing check (POST/PUT/PATCH only)

If the handler reads a JSON body, test:
- **Bad JSON** (malformed) → 400
- **Empty body** → 400 (if required fields are missing, through validation)

```go
t.Run("bad JSON returns 400", func(t *testing.T) {
    r := badJSONReq(t, http.MethodPost, "/tenants")
    w := httptest.NewRecorder()
    newTenantHandler(nil, nil).Create(w, r)
    assert.Equal(t, http.StatusBadRequest, w.Code)
})
```

#### 5. Request DTO validation

Every request DTO has a `Validate()` method. Test each distinct validation rule:

```go
t.Run("validation error returns 400", func(t *testing.T) {
    r := jsonReq(t, http.MethodPost, "/tenants", map[string]any{"name": ""})
    w := httptest.NewRecorder()
    newTenantHandler(nil, nil).Create(w, r)
    assert.Equal(t, http.StatusBadRequest, w.Code)
})
```

#### 6. Dependency lookup errors

If the handler calls a service to look up a resource before acting on it, test:
- **Lookup not found** → 404
- **Lookup service error** → 500 (or 404/400 depending on how `HandleServiceError` maps the error)

```go
t.Run("GetByUUID error returns 404", func(t *testing.T) {
    ts := &mockTenantService{
        getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
            return nil, errNotFound
        },
    }
    r := withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "tenant_uuid", testResourceUUID.String())
    w := httptest.NewRecorder()
    newTenantHandler(ts, nil).GetByUUID(w, r)
    assert.Equal(t, http.StatusNotFound, w.Code)
})
```

#### 7. Business rule errors

If the handler enforces domain rules beyond auth (e.g. "cannot delete a system tenant"), test each rule:

```go
t.Run("system tenant returns 403", func(t *testing.T) {
    ts := &mockTenantService{
        getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
            return &TenantServiceDataResult{IsSystem: true}, nil
        },
    }
    ms := &mockTenantMemberService{
        isUserInTenantFn: func(int64, uuid.UUID) (bool, error) { return true, nil },
    }
    r := withUser(withChiParam(
        httptest.NewRequest(http.MethodDelete, "/", nil),
        "tenant_uuid", testResourceUUID.String(),
    ))
    w := httptest.NewRecorder()
    newTenantHandler(ts, ms).Delete(w, r)
    assert.Equal(t, http.StatusForbidden, w.Code)
})
```

#### 8. Primary operation service error

The main service call made by the handler must have its error path covered:

```go
t.Run("service error returns 500", func(t *testing.T) {
    svc := &mockTenantService{
        createFn: func(n, dn, desc, s string, isPublic bool) (*TenantServiceDataResult, error) {
            return nil, errors.New("db error")
        },
    }
    r := jsonReq(t, http.MethodPost, "/tenants", validBody)
    w := httptest.NewRecorder()
    newTenantHandler(svc, nil).Create(w, r)
    assert.Equal(t, http.StatusInternalServerError, w.Code)
})
```

> Note: Use `errors.New("...")` for unknown/generic errors (maps to 500). Use `errNotFound`, `errValidation`, or typed apperrors to test specific status code mappings.

#### 9. Success — verify all three: status, headers, body

Success tests must assert:
1. **Status code** (200, 201, 204)
2. **Response body shape and values** (decode JSON, assert expected fields)
3. **Response headers** (Content-Type, Cache-Control, etc.) when the handler sets them

```go
t.Run("success returns 201 with response body", func(t *testing.T) {
    svc := &mockTenantService{
        createFn: func(n, dn, desc, s string, isPublic bool) (*TenantServiceDataResult, error) {
            return &TenantServiceDataResult{
                UUID:   uuid.New(),
                Name:   n,
                Status: shared.StatusActive,
            }, nil
        },
    }
    r := jsonReq(t, http.MethodPost, "/tenants", validBody)
    w := httptest.NewRecorder()
    newTenantHandler(svc, nil).Create(w, r)

    assert.Equal(t, http.StatusCreated, w.Code)
    assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

    var resp TenantResponseDTO
    require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
    assert.Equal(t, "my-tenant", resp.Name)
    assert.Equal(t, shared.StatusActive, resp.Status)
})
```

### POST/PUT/PATCH handlers (state-changing)

These handlers go through the **full checklist** above (steps 1–9). Auth/authorization checks run before any domain logic, so test them first in the file — matching the handler's actual control flow order.

For handlers with conditional access checks, test every branch:

```go
func TestTenantHandler_Update(t *testing.T) {
    // Auth checks (runs first)
    t.Run("no user returns 401", ...)
    // URL params
    t.Run("invalid UUID returns 400", ...)
    // Authorization
    t.Run("IsUserInTenant error returns 500", ...)
    t.Run("not a member returns 403", ...)
    // Body
    t.Run("bad JSON returns 400", ...)
    // Validation
    t.Run("validation error returns 400", ...)
    // Service
    t.Run("service error returns 500", ...)
    // Success
    t.Run("success returns 200", ...)
}
```

### GET handlers (reads)

GET handlers typically skip body-parsing checks (no request body). The checklist is shorter:

1. Authentication check (if behind JWT) → 401
2. URL parameter validation → 400
3. Query parameter / filter validation → 400
4. Dependency lookup errors → 404 / 500
5. Primary service error → 500
6. Success with full response body assertion

```go
func TestTenantHandler_Get(t *testing.T) {
    t.Run("validation error returns 400", ...)      // step 3
    t.Run("service error returns 500", ...)          // step 5
    t.Run("success with all filters covers filter+loop branches", ...)  // step 6
}
```

### DELETE handlers

DELETE handlers must additionally verify that **business rules protecting resources** are enforced before the delete proceeds:

```go
func TestTenantHandler_Delete(t *testing.T) {
    t.Run("no user returns 401", ...)
    t.Run("invalid UUID returns 400", ...)
    t.Run("IsUserInTenant error returns 500", ...)
    t.Run("not a member returns 403", ...)
    t.Run("GetByUUID error returns 404", ...)
    t.Run("system tenant returns 403", ...)   // <-- business rule
    t.Run("DeleteByUUID error returns 500", ...)
    t.Run("success returns 200", ...)
}
```

### Unauthenticated routes

Some routes are **not** behind JWTAuthMiddleware and do not read a user from context. Examples:
- `POST /oauth/token`
- `POST /oauth/revoke`
- `POST /oauth/par`
- `POST /forgot-password`
- `POST /reset-password`
- `GET /.well-known/openid-configuration`

For these handlers, **skip** authentication and authorization checks. The checklist still applies, but it starts from request validation (step 3):

1. ~~Authentication check~~ (skip — route is public)
2. ~~Authorization check~~ (skip — route is public)
3. Body parsing → 400
4. DTO validation → 400
5. Dependency errors → appropriate status
6. Service error → appropriate status
7. Success → full assertion

> OAuth error responses use the `error`/`error_description` JSON envelope form, not the standard `{ "error": { "code": ... } }` structure. Map them accordingly in OAuth handler tests.

```go
func TestOAuthTokenHandler_Token_Success(t *testing.T) {
    svc := &mockOAuthTokenService{
        exchangeFn: func(_ context.Context, _ OAuthTokenRequestDTO, _ OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
            return &OAuthTokenResult{
                AccessToken: "at-123",
                TokenType:   "Bearer",
                ExpiresIn:   900,
                Scope:       "openid",
            }, nil
        },
    }
    h := NewOAuthTokenHandler(svc, nil, nil)
    r := formReq(t, "/oauth/token", url.Values{
        "grant_type": {"authorization_code"},
        "code":       {"valid-code"},
    })
    w := httptest.NewRecorder()
    h.Token(w, r)

    assert.Equal(t, http.StatusOK, w.Code)
    assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
    assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))

    var resp OAuthTokenResponseDTO
    require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
    assert.Equal(t, "at-123", resp.AccessToken)
    assert.Equal(t, "Bearer", resp.TokenType)
    assert.Equal(t, int64(900), resp.ExpiresIn)
}
```

### Belongs-to-tenant checks

When a handler operates on a resource scoped to a tenant, verify that the handler enforces tenant-scoping at the service or handler level. The handler **must** check that the authenticated user belongs to the target tenant before performing the operation.

Pattern: handler reads the target tenant UUID from a URL parameter, calls `IsUserInTenant(userID, tenantUUID)` on the member service, and returns 403 if the user is not a member.

Cover these sub-cases:

```go
// 1. Membership check itself fails → 500
t.Run("IsUserInTenant error returns 500", func(t *testing.T) {
    ms := &mockTenantMemberService{
        isUserInTenantFn: func(int64, uuid.UUID) (bool, error) {
            return false, errors.New("db error")
        },
    }
    r := withUser(withChiParam(
        jsonReq(t, http.MethodPut, "/", validBody),
        "tenant_uuid", testResourceUUID.String(),
    ))
    w := httptest.NewRecorder()
    newTenantHandler(nil, ms).Update(w, r)
    assert.Equal(t, http.StatusInternalServerError, w.Code)
})

// 2. User is not in the target tenant → 403
t.Run("not a member returns 403", func(t *testing.T) {
    ms := &mockTenantMemberService{
        isUserInTenantFn: func(int64, uuid.UUID) (bool, error) { return false, nil },
    }
    r := withUser(withChiParam(...))
    w := httptest.NewRecorder()
    newTenantHandler(nil, ms).Update(w, r)
    assert.Equal(t, http.StatusForbidden, w.Code)
})

// 3. User is in the target tenant → proceed to success
t.Run("success returns 200", func(t *testing.T) {
    ms := &mockTenantMemberService{
        isUserInTenantFn: func(int64, uuid.UUID) (bool, error) { return true, nil },
    }
    ts := &mockTenantService{
        updateFn: func(id uuid.UUID, ...) (*TenantServiceDataResult, error) {
            return &TenantServiceDataResult{Name: "updated"}, nil
        },
    }
    r := withUser(withChiParam(...))
    w := httptest.NewRecorder()
    newTenantHandler(ts, ms).Update(w, r)
    assert.Equal(t, http.StatusOK, w.Code)
})
```

### HTTP handler test helpers

Each domain package with handler tests provides shared helpers in `http_testhelpers_test.go`. Use these consistently:

```go
// Inject chi URL parameters (e.g. tenant_uuid=xxx)
r := withChiParam(r, "tenant_uuid", testResourceUUID.String())

// Inject an authenticated user into context
r := withUser(r)

// Inject a full auth context (user + tenant)
r := withTenant(r)

// Create a request with JSON body
r := jsonReq(t, http.MethodPost, "/tenants", map[string]any{"name": "acme"})

// Create a request with intentionally malformed JSON
r := badJSONReq(t, http.MethodPost, "/tenants")
```

These helpers compose via chaining:
```go
r := withUser(withChiParam(jsonReq(t, http.MethodPut, "/", body), "tenant_uuid", uuid.String()))
```

---

## Middleware Tests

Middleware tests wrap a dummy handler and assert the middleware's effect on the request and response:

```go
func TestJWTAuthMiddleware(t *testing.T) {
    dummy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })
    handler := JWTAuthMiddleware(dummy)

    t.Run("no Authorization header returns 401", func(t *testing.T) {
        r := httptest.NewRequest(http.MethodGet, "/", nil)
        w := httptest.NewRecorder()
        handler.ServeHTTP(w, r)
        assert.Equal(t, http.StatusUnauthorized, w.Code)
    })

    t.Run("malformed Authorization header returns 401", func(t *testing.T) {
        r := httptest.NewRequest(http.MethodGet, "/", nil)
        r.Header.Set("Authorization", "Bearer")
        w := httptest.NewRecorder()
        handler.ServeHTTP(w, r)
        assert.Equal(t, http.StatusUnauthorized, w.Code)
    })

    t.Run("valid token passes through", func(t *testing.T) {
        r := httptest.NewRequest(http.MethodGet, "/", nil)
        r.Header.Set("Authorization", "Bearer valid-token")
        w := httptest.NewRecorder()
        handler.ServeHTTP(w, r)
        assert.Equal(t, http.StatusOK, w.Code)
    })
}
```

---

## Cross-cutting Tests

Some invariants span multiple types or files and do not map to a single source file. These live in `<concern>_test.go`:

- `isolation_test.go` — tenant isolation matrix: exhaustively verifies that no cross-tenant access leaks through.
- Property-based invariants (e.g. "ValidateTenantAccess and ValidateTenantAccessByID must always agree").
- Nil-safety guards (e.g. "nil user must not panic").

These tests use the same mock/stub infrastructure as unit tests but exercise invariants across the package, not a single method.

```go
func TestTenantIsolation_CrossTenantNeverLeaks(t *testing.T) {
    tenantIDs := []int64{2, 3, 4, 5, 6}
    for _, userTenant := range tenantIDs {
        for _, targetTenant := range tenantIDs {
            if userTenant == targetTenant {
                continue
            }
            t.Run(fmt.Sprintf("user_in_%d_cannot_access_%d", userTenant, targetTenant), func(t *testing.T) {
                user := buildUserWithIdentities([]AccessIdentity{
                    buildIdentity(userTenant, false),
                })
                err := ValidateTenantAccessByID(user, targetTenant)
                require.Error(t, err)
                assert.Contains(t, err.Error(), "access denied")
            })
        }
    }
}
```

---

## Mocking Strategy

### Database (GORM + go-sqlmock)

Services accept a `*gorm.DB`. Use `go-sqlmock` to assert that transactions are opened, committed, or rolled back without a real database.

```go
db, mock, err := sqlmock.New()
require.NoError(t, err)
gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
require.NoError(t, err)

mock.ExpectBegin()
mock.ExpectCommit()

// exercise the service ...

require.NoError(t, mock.ExpectationsWereMet())
```

### Repositories

Repositories are injected as interfaces. Define a minimal mock struct with function fields so each test only implements the methods it needs.

```go
type mockUserRepo struct {
    findByEmailFn func(email string) (*model.User, error)
}

func (m *mockUserRepo) FindByEmail(email string) (*model.User, error) {
    if m.findByEmailFn != nil {
        return m.findByEmailFn(email)
    }
    return nil, nil  // safe zero value
}
```

### Services

Services consumed by handlers follow the same pattern:

```go
type mockTenantService struct {
    getByUUIDFn func(uuid.UUID) (*TenantServiceDataResult, error)
    createFn    func(string, string, string, string, bool) (*TenantServiceDataResult, error)
}

func (m *mockTenantService) GetByUUID(_ context.Context, id uuid.UUID) (*TenantServiceDataResult, error) {
    if m.getByUUIDFn != nil {
        return m.getByUUIDFn(id)
    }
    return nil, nil
}
```

**Rules for mocks:**
- Every mock method returns a safe zero value when its function field is nil. This prevents panics when a test configures only some methods.
- Mocks live in `mock_test.go` (for services) and `mock_repos_test.go` (for repositories). Keep them in the package's root, not nested.
- Keep mock structs flat — one struct per interface, function fields named `<method>Fn`.
- Package-level `errNotFound` and `errValidation` variables (typed apperrors) serve as reusable error fixtures across all tests in the package.

### Package-level function injection

For simple leaf dependencies (e.g. `crypto.GenerateIdentifier`, `jwt.ValidateToken`), the codebase uses package-level function variables that tests can swap temporarily:

```go
orig := crypto.GenerateIdentifier
defer func() { crypto.GenerateIdentifier = orig }()
crypto.GenerateIdentifier = func(int) (string, error) { return "", errors.New("test error") }
```

Use this pattern sparingly — prefer interface injection for service/repository dependencies. Reserve function injection for stateless utility functions where an interface wrapper would be overkill.

---

## Environment Variables in Tests

Some utilities read environment variables at call time. Set them safely inside a test with `t.Setenv` — the value is restored automatically when the test ends.

```go
func TestGenerateSignedURL(t *testing.T) {
    t.Setenv("HMAC_SECRET_KEY", "test-secret")
    // ...
}
```

**Never** use `os.Setenv` in tests — it leaks state across tests run in parallel.

### JWT keys (service-layer tests)

Service tests that call `util.GenerateAccessToken` or `util.GenerateIDToken` require RSA keys to be initialised. Call the shared helper defined in `internal/service/login_service_test.go`:

```go
func TestMyService(t *testing.T) {
    initTestJWTKeysService(t) // generates a fresh RSA-2048 pair
    // ...
}
```

---

## Coverage

```bash
make test-cover        # generates coverage.html
```

Open `coverage.html` in a browser to see per-line coverage. Red lines are uncovered.

**Target:** every new public function must have at least one test. There is no hard percentage gate, but reviewers will ask for tests on any uncovered public function.

**Paths excluded from coverage** (`codecov.yml`):
- `cmd/server`, `internal/app`, `internal/server` — pure wiring, no branching logic
- `internal/gen` — generated code
- `internal/database`, `internal/runner`, `internal/model`, `internal/grpc`, `internal/rest`, `internal/templates` — infrastructure-dependent; covered by integration/e2e tiers

---

## Integration Tests

Integration tests require a running PostgreSQL database and Redis instance and are **not** run by `go test ./...`.

They live under `tests/integration/` and are gated by a build tag:

```go
//go:build integration

package repository
```

### Directory structure

```
tests/integration/
├── repository/          # Repository-layer tests against real Postgres schema
│   ├── placeholder_test.go
│   ├── tenant_test.go
│   └── user_test.go
├── cache/               # Cache-layer tests against real Redis
│   └── user_context_test.go
└── middleware/           # Full middleware chain with real context resolution
    └── auth_chain_test.go
```

Run them explicitly:

```bash
docker-compose up -d postgres-db redis-db
go test -tags integration ./tests/integration/... -v
```

### What to test at the integration level

- **Repository queries** against the real schema: CRUD operations, pagination, join correctness, soft-delete behaviour.
- **Transaction boundaries**: commit/rollback behaviour with real DB state.
- **Cache serialization**: round-trip a struct through Redis and back, verify field fidelity.
- **Middleware chain composition**: verify that `JWTAuth → UserContext → Permission` chain resolves correctly end-to-end with real cache population.

---

## End-to-End Tests

E2E tests require the full HTTP stack running (Postgres, Redis, and the server binary) and are gated by a build tag:

```go
//go:build e2e

package e2e_test
```

### Directory structure

```
tests/e2e/
├── placeholder_test.go
├── auth_test.go          # Login → JWT → call protected endpoint
├── tenant_test.go        # Create tenant → list tenants → delete tenant
├── oauth_flow_test.go    # Full OAuth authorization code grant flow
└── multi_tenant_test.go  # Cross-tenant isolation at the API level
```

Run them:

```bash
docker-compose up -d
go test -tags e2e ./tests/e2e/... -v
```

### What to test at the E2E level

E2E tests are **contract tests**. They verify that the API surface behaves the way clients expect. Each test file covers one user-facing flow:

1. Seed or authenticate to obtain credentials
2. Send a real HTTP request to the running server
3. Assert the status code, response headers, and response body shape

**Do not** test internal implementation details at the E2E level — those belong in unit or integration tests. E2E tests answer: "If a client sends this request, does the server respond correctly?"

---

## What to Test When Adding Code

| What you added | What you must test |
|---|---|
| A new utility function in `internal/platform/` | A `_test.go` file beside it; table-driven cases covering happy path, edge cases, and error path |
| A new service method | A test in `internal/<domain>/service_<name>_test.go`; mock the repo and DB transaction; cover success, each distinct error branch (not-found, repo error, validation error, business rule violation), and any conditional branching |
| A new validation method | A test in `internal/<domain>/validation_<name>_test.go`; one sub-test per validation rule and one for the valid case |
| A new REST handler | An `httptest`-based test in `internal/<domain>/handler_<name>_test.go`; cover the full checklist: auth, authorization, param/bad-JSON/validation errors, service errors, and success with full response body assertion |
| A new middleware | A test that wraps a dummy handler and asserts the middleware's effect on the request/response for each code path |
| A new repository method | Unit test the service that calls it (mock the repo); add an integration test in `tests/integration/repository/` against a real DB |
| A new API endpoint | Unit test the handler; add an E2E test in `tests/e2e/` that hits the endpoint with a real HTTP request |
| A config or env change | Verify the zero value or default does not panic; test `t.Setenv` for required variables |
| A cross-cutting invariant | A `<concern>_test.go` file exercising the invariant across multiple types/methods |

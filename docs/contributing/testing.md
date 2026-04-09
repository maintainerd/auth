# Testing Guide

This document describes how to run, write, and organise tests when contributing to **Maintainerd Auth**.

---

## Table of Contents

- [Running Tests](#running-tests)
- [Test Layout](#test-layout)
- [Writing Unit Tests](#writing-unit-tests)
- [Mocking Strategy](#mocking-strategy)
- [Environment Variables in Tests](#environment-variables-in-tests)
- [Coverage](#coverage)
- [Integration Tests](#integration-tests)
- [What to Test When Adding Code](#what-to-test-when-adding-code)

---

## Running Tests

All commands below are available via `make` or raw `go test`.

| Goal                          | Command                                            |
| ----------------------------- | -------------------------------------------------- |
| Run all unit tests            | `make test` or `go test ./...`                     |
| Run with race detector        | `make test-race`                                   |
| Generate HTML coverage report | `make test-cover`                                  |
| Run tests for one package     | `go test ./internal/util/... -v`                   |
| Run a single test by name     | `go test ./internal/util/... -run TestGenerateOTP` |
| Run with verbose output       | `go test ./... -v`                                 |

> **Before every pull request**, run `make test-race` to catch data races that the standard runner will miss.

---

## Test Layout

Follow the **same-package, beside-the-source-file** convention used by Ory Hydra, Casdoor, and the broader Go community.

```
internal/
├── util/
│   ├── jwt_util.go
│   └── jwt_util_test.go      ← unit test lives here
├── service/
│   ├── login_service.go
│   └── login_service_test.go ← unit test lives here
```

**Rules:**
- Test files use the same `package` declaration as the source file (`package util`, not `package util_test`).  
  This lets tests access unexported helpers when needed.
- One test file per source file. Do not merge tests from multiple source files.
- Integration tests that require a live database or network belong in `tests/integration/` (see [Integration Tests](#integration-tests)).

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

Never use bare `t.Fatal` or `t.Error` — testify gives better output.

### HTTP handler tests

Use `net/http/httptest` to avoid a real server.

```go
rr := httptest.NewRecorder()
req := httptest.NewRequest(http.MethodPost, "/login", body)
handler.ServeHTTP(rr, req)
assert.Equal(t, http.StatusOK, rr.Code)
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
    return m.findByEmailFn(email)
}
// ... other interface methods return zero values by default
```

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

**Target:** every new function in `internal/util/` and `internal/service/` must have at least one test. There is no hard percentage gate, but reviewers will ask for tests on any uncovered public function.

---

## Integration Tests

Integration tests require a running PostgreSQL database and Redis instance and are **not** run by `go test ./...`.

They live under `tests/integration/` and are gated by a build tag:

```go
//go:build integration

package integration
```

Run them explicitly:

```bash
go test -tags integration ./tests/integration/... -v
```

> The `tests/integration/` directory does not exist yet. If you are the first to add an integration test, create it with the build tag above and update this guide.

---

## What to Test When Adding Code

| What you added | What you must test |
|---|---|
| A new utility function in `internal/util/` | A `_test.go` file beside it; table-driven cases covering happy path, edge cases, and error path |
| A new service method | A test in `internal/service/<name>_test.go`; mock the repo and DB transaction; cover success, not-found, and auth-failure branches |
| A new REST handler | An `httptest`-based test in `internal/handler/resthandler/`; assert status code and JSON body |
| A new middleware | A test that wraps a dummy handler and asserts the middleware's effect on the request/response |
| A config or env change | Verify the zero value or default does not panic; test `t.Setenv` for required variables |


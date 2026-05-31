# Code Structure

This project uses role-first file names inside feature packages so related layers
are easy to scan and keep separate. This is a convention for code that actually
has those roles; it is not a requirement to invent empty model, handler,
repository, or validation files in packages that do not need them.

> Note on Go semantics: Go compiles every `.go` file in a directory as one
> package and ignores file names entirely. The naming scheme below is an
> organizational convention for humans, not a compiler requirement — any
> exported symbol is visible to the whole package regardless of which file it
> lives in. Split files to keep each role cohesive, not to create visibility
> boundaries.

## Feature Package File Naming

Use these prefixes consistently:

- `model_<name>.go` for GORM/domain persistence models, table names, and model hooks.
- `service_<name>.go` for service interfaces, service implementations, service filters/results, and business logic.
- `handler_<name>.go` for HTTP handlers and handler-only response mapping.
- `validation_<name>.go` for DTO validation methods and validation-only constants.
- `repository_<name>.go` for repository interfaces and persistence implementations for a specific aggregate or subresource. Keep the backend out of the filename when the package has only one persistence backend.
- `routes.go` for route registration.
- `types.go` for API DTOs and request/response shapes only.
- `<source>_test.go` mirrors the source file under test: `service_<name>_test.go`
  tests `service_<name>.go`, `handler_<name>_test.go` tests `handler_<name>.go`,
  `validation_<name>_test.go` tests `validation_<name>.go`. Name the test after
  the file the code lives in, not after the type it exercises.
- `<concern>_test.go` only for cross-cutting tests that do not map to a single
  source file (for example `isolation_test.go` for a tenant-isolation invariant).
- `mock_test.go`, `mock_repos_test.go`, `http_testhelpers_test.go`, and
  `testhelpers_test.go` for test-only mocks and shared helpers. Put helpers used
  by more than one test file here rather than in one test's file.

Apply only the roles a package truly has. A package with no database-owned
aggregate should not have a `model_*.go`; a package with no HTTP surface should
not have a `handler_*.go`; a package with no repository should not carry a
placeholder repository file. The structure exists to make real boundaries
visible, not to make every package look identical.

Composition and infrastructure packages use their own honest names:

- `cmd/server` is the executable bootstrap package. Keep `main.go` tiny: it
  should call a package-local `run(context.Context)` and translate a returned
  error into process exit. Use focused files such as `bootstrap.go` for the
  startup sequence, `logging.go` for logger configuration, `telemetry.go` for
  OpenTelemetry setup/shutdown, and `workers.go` for background workers. When a
  command package has multiple files, run it with `go run ./cmd/server`, not
  `go run cmd/server/main.go`. Document command-package helpers with short
  developer-facing comments that explain ordering and ownership decisions:
  why `main` delegates to `run`, why bootstrap logging exists before config,
  why telemetry shutdown is bundled, and why background workers share the
  bootstrap cancellation context. Every function in `cmd/server`, including
  unexported helpers and tests, should have a short purpose comment because this
  package is the executable startup map.
- `internal/app` is the composition root. It wires dependency graph objects; it
  does not own domain behavior, models, handlers, routes, or DTO validation. Use
  `app.go` for the exported runtime bundle, `repositories.go` for repository
  construction, `services.go` for service construction, `application.go` for
  adapting the app bundle to runtime/transport bundles, and
  `adapters_<consumer>[_<source-or-concern>].go` for cross-domain adapter
  implementations. Keep adapter model mappers beside the adapter family, for
  example `adapters_authn_user_models.go`, instead of collecting unrelated
  mappings in a broad `models.go` file.
- `internal/server` owns transport runtime concerns. Use `application.go` for
  the transport dependency bundle, `rest.go` and `grpc.go` for server lifecycle,
  `router.go` for route mounting and transport middleware, `handlers.go` for
  transport handler construction, `health.go` for liveness/readiness endpoints,
  `openapi.go` for OpenAPI serving, and `adapters_<concern>.go` for transport
  adapter glue. Avoid feature-package role prefixes here unless the server
  package actually grows a real domain model, repository, service, or handler
  layer of its own.
- `internal/platform/*` packages are reusable infrastructure. Prefer names that
  describe the utility or component (`jwt`, `middleware`, `database`,
  `telemetry`) rather than domain-layer prefixes.
- Feature packages may also contain cohesive support components that are not
  CRUD layers. For example `internal/webhook` keeps delivery-specific files such
  as `dispatcher.go`, `deliver.go`, `payload.go`, and `signer.go` because those
  are parts of the webhook delivery pipeline, not endpoint CRUD handlers,
  repositories, or DTO validation.

## Test File Naming

A test file is named after the **source file** it covers, not the type or
function it exercises. This keeps the source/test pairing obvious and survives
refactors: when code moves to a different file, its test moves with it under the
matching name.

| Source file              | Test file                     |
| ------------------------ | ----------------------------- |
| `service_<name>.go`      | `service_<name>_test.go`      |
| `handler_<name>.go`      | `handler_<name>_test.go`      |
| `validation_<name>.go`   | `validation_<name>_test.go`   |
| `repository_<name>.go`   | `repository_<name>_test.go`   |

Rules:

- Mirror the source file name. Tests for `TenantService` (in `service_tenant.go`)
  go in `service_tenant_test.go`, not `tenant_test.go`. Tests for DTO `Validate()`
  methods go in `validation_<name>_test.go`, not `<name>_types_test.go` — they
  exercise validation, not the `types.go` structs.
- Use `<concern>_test.go` only for a cross-cutting test that has no single source
  file (for example `isolation_test.go` for a tenant-isolation invariant that
  spans several types).
- When you move code between files, move its tests too so the pairing holds (for
  example access-control logic folded into `service_tenant.go` has its tests in
  `service_tenant_test.go`).
- A helper used by more than one test file goes in `testhelpers_test.go` (or the
  matching `*_testhelpers_test.go` / `mock_test.go`), never in a single test file
  that another test happens to depend on.

```text
internal/tenant/
  service_tenant.go          → service_tenant_test.go
  service_member.go          → service_member_test.go
  service_setting.go         → service_setting_test.go
  handler_tenant.go          → handler_tenant_test.go
  handler_setting.go         → handler_setting_test.go
  validation_tenant.go       → validation_tenant_test.go
  validation_setting.go      → validation_setting_test.go
  (cross-cutting invariant)  → isolation_test.go
  (shared mocks/helpers)     → mock_test.go, mock_repos_test.go,
                               http_testhelpers_test.go, testhelpers_test.go
```

## Supporting Files

Some packages need file roles beyond the per-resource layers above. Use these
names consistently:

- `deps.go` for consumer-defined interfaces and projection types a package
  declares for upstream domains so it does not import them directly (the
  decoupling pattern). Keep these out of `types.go`, which is API DTOs only.
- `foundation.go` for package-local type aliases and thin wrappers over
  `internal/platform/*` helpers. This is re-export glue only; reusable
  cross-domain logic still belongs under `internal/platform/<helper>`.

### Why `deps.go` is not part of `types.go`

`types.go` and `deps.go` are opposite-facing contracts, so they stay separate:

- `types.go` faces **out** — the HTTP wire format this domain exposes. Its
  structs carry `json` tags (for example `TenantResponseDTO`,
  `MemberUserResponseDTO`).
- `deps.go` faces **in** — what this domain needs *from upstream domains*: the
  consumer-defined interfaces (for example `UserReader`, `AccessActor`) plus the
  plain projection structs they pass (for example `MemberUser`,
  `AccessIdentity`), which have **no `json` tags** because they never touch the
  wire.

They also change for different reasons — `types.go` when the API changes,
`deps.go` when a cross-domain boundary is refactored. Watch for the giveaway: a
projection like `MemberUser` (no tags) belongs in `deps.go`, while its wire twin
`MemberUserResponseDTO` (json tags) belongs in `types.go`. Keeping them apart
makes clear which structs are the dependency contract and which are the API.

### Why `foundation.go` is not in `internal/platform`

The dependency only flows domain → platform, and `foundation.go` already points
*at* platform: it holds package-local type aliases and one-line wrappers over
`internal/platform/*` (for example a local `PaginationResult[Tenant]` alias, or a
`sanitizeOrder` that just calls `database.SanitizeOrder`). The reusable substance
already lives in platform; `foundation.go` is the domain-side binding layer that
adapts those generics to the domain's concrete types. Moving it into platform
would invert the dependency, and platform must never import a domain.

Rule of thumb:

- Glue and aliases over platform → stay in the domain's `foundation.go`.
- Real reusable logic that creeps into a `foundation.go` → move *that* into
  `internal/platform/<helper>`.

For example:

```text
internal/tenant/
  model_tenant.go
  model_member.go
  model_setting.go
  service_tenant.go
  service_member.go
  service_setting.go
  handler_tenant.go
  handler_member.go
  handler_setting.go
  validation_tenant.go
  validation_member.go
  validation_setting.go
  repository_tenant.go
  repository_member.go
  repository_setting.go
  routes.go
  types.go
  deps.go
  foundation.go
```

## Separation Rules

- Do not mix GORM model structs into service files.
- Do not put business logic in `model_*.go`; model files should stay limited to
  fields, table names, and model hooks.
- Do not put HTTP request parsing or response writing in service files.
- Keep DTO structs in `types.go`; keep DTO validation methods in
  `validation_*.go`.
- Split repositories by aggregate or subresource when one file starts mixing
  unrelated persistence concerns.
- Keep reusable cross-domain helpers under `internal/platform/<helper>` (see
  the next section for the boundary).
- Keep domain-specific behavior inside the domain package unless it is repeated
  by multiple packages and has no domain dependency.
- Domain business logic with no service struct of its own (for example
  cross-service access-control rules) lives in the relevant `service_<name>.go`,
  not in a separate file. Only extract a standalone file when the logic is
  genuinely domain-agnostic — in which case it belongs in `internal/platform`.

## What Belongs in `internal/platform`

`internal/platform/<helper>` is for reusable, **domain-agnostic** building blocks:
code that knows nothing about tenants, users, roles, clients, etc., and could be
dropped into an unrelated service unchanged. Existing examples set the bar:
`ptr`, `jsonutil`, `crypto`, `pagination`, `response`, `database`, `apperror`,
`jwt`, `cache`, `valid`.

Move code here when it is:

- A pure transformation, conversion, or formatting helper that takes and returns
  generic/standard-library or platform types (no domain structs).
- A generic data-structure, encoding, or I/O utility (pagination math, JSON
  helpers, pointer helpers, response writers, time/string formatting).
- Cross-cutting infrastructure (logging, telemetry, middleware, crypto, mailers)
  used by more than one domain.

Keep code in the domain package when it is:

- Tied to a domain type (`*Tenant`, `*User`, …) or expresses a business rule.
- A mapping between a domain model and that domain's DTOs/results — those stay in
  `service_*.go` / `handler_*.go`.

Litmus test: if a function's signature mentions a domain type, or its body
encodes a business rule, it stays in the domain. If you could publish it as a
standalone utility library with no domain import, it belongs in `internal/platform`.

Hard rule: **`internal/platform/*` must never import `internal/<domain>`.** The
dependency only flows domain → platform. If a helper needs a domain type, it is
not platform code.

## Go Conventions and Recommendations

These follow the Go team's published guidance (Effective Go and the Go Code
Review Comments / wiki). Apply them to new code; they are the intended direction
for existing code, not a mandate to rewrite working symbols all at once.

- **Define interfaces on the consumer side.** An interface belongs in the package
  that *uses* the values, not the package that implements them. This is exactly
  what `deps.go` does: a domain declares the narrow interface it needs from an
  upstream domain, and the composition root injects an adapter that satisfies it.
  Keep these interfaces small — the larger the interface, the weaker the
  abstraction.
- **Accept interfaces, return concrete types.** Constructors and functions should
  take the minimal interface they need and return concrete structs/results where
  practical, so callers are not forced to depend on an interface they do not own.
- **Use package-qualified names.** Since the package name already supplies domain
  context, `tenant.Service` is the stylistically preferred form for new code.
  However, many existing symbols use the stuttered form (`tenant.TenantService`)
  — the project intentionally accepts both forms. Do not rename working symbols
  solely to fix stutter.
- **Package names are short, lower-case, singular, no underscores.** The
  underscores in *file* names (`model_tenant.go`) are fine — they are not package
  names.
- **Test files mirror the code under test.** `foo_test.go` tests the concern in
  `foo.go`; when logic moves to another file, move or rename its test to track it
  so the pairing stays obvious. Test-only helpers use the shared
  `*_test.go` helper file names listed above.

## Before Updating Graphify

After code edits, run the affected package tests and any app-level tests touched
by the change. Fix failures before running `graphify update .`.

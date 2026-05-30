# Code Structure

This project uses role-first file names inside feature packages so related layers
are easy to scan and keep separate.

## Feature Package File Naming

Use these prefixes consistently:

- `model_<name>.go` for GORM/domain persistence models, table names, and model hooks.
- `service_<name>.go` for service interfaces, service implementations, service filters/results, and business logic.
- `handler_<name>.go` for HTTP handlers and handler-only response mapping.
- `validation_<name>.go` for DTO validation methods and validation-only constants.
- `repository_<name>.go` for repository interfaces and persistence implementations for a specific aggregate or subresource. Keep the backend out of the filename when the package has only one persistence backend.
- `routes.go` for route registration.
- `types.go` for API DTOs and request/response shapes only.
- `<concern>_test.go` for focused tests matching the code under test.
- `mock_test.go`, `mock_repos_test.go`, and `http_testhelpers_test.go` for test-only helpers.

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
- Keep reusable cross-domain helpers under `internal/platform/<helper>`.
- Keep domain-specific behavior inside the domain package unless it is repeated
  by multiple packages and has no domain dependency.

## Before Updating Graphify

After code edits, run the affected package tests and any app-level tests touched
by the change. Fix failures before running `graphify update .`.

## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

When the user types `/graphify`, invoke the `skill` tool with `skill: "graphify"` before doing anything else.

Rules:
- Follow the project file/layer structure in `docs/contributing/code-structure.md` when creating or reorganizing code.
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- Dirty graphify-out/ files are expected after hooks or incremental updates; dirty graph files are not a reason to skip graphify. Only skip graphify if the task is about stale or incorrect graph output, or the user explicitly says not to use it.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, check the app for compile/test errors first, at minimum with the package touched and any app-level package affected (for example `go test ./internal/tenant` and `go test ./internal/app` for tenant/app changes). Fix failures before running any Graphify update.
- After the app checks pass, run `graphify update .` to keep the current graph updated (AST-only, no API cost). Do not run commands that create timestamped report folders unless the user explicitly asks for a new report.

## testing

Always follow the test standards in [docs/contributing/testing.md](docs/contributing/testing.md). Key references:

- **Handler tests**: 9-step checklist (auth → authz → params → body → validation → deps → business rules → service → success). See [handler-test-checklist](docs/contributing/testing.md#handler-test-checklist).
- **Service tests**: one success sub-test per branch, one sub-test per error path. See [service-layer-tests](docs/contributing/testing.md#service-layer-tests).
- **Validation tests**: one sub-test per validation rule. See [validation-tests](docs/contributing/testing.md#validation-tests).
- **Mock conventions**: function-field pattern in `mock_test.go` / `mock_repos_test.go`. See [mocking-strategy](docs/contributing/testing.md#mocking-strategy).
- **Test placement**: unit tests beside source; integration tests in `tests/integration/` (tag: `integration`); e2e tests in `tests/e2e/` (tag: `e2e`). See [test-tiers](docs/contributing/testing.md#test-tiers-and-placement).
- **Coverage baseline**: total is ~69%. Target ≥80% per domain package. Track gaps in [docs/planning/test-coverage.md](docs/planning/test-coverage.md).

When writing or fixing tests, consult the coverage plan to see which packages need attention and what is missing.

## database migrations

Follow [docs/contributing/database-migrations.md](docs/contributing/database-migrations.md). Key rule: **migrations are create-only** while the project is pre-release / not deployed anywhere — there is one canonical `NNN_create_<table>_table.go` per table. To change a table's schema, **edit its original create migration in place** (and the matching GORM model in the owning package); do **not** add `*_add_*`/`*_alter_*`/`*_drop_*`/backfill migrations. Only brand-new tables get a new migration, appended to the registry in `internal/platform/runner/migration.go`. This rule freezes at first production deployment.

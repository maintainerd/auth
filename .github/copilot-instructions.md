# maintainerd/auth — Copilot Instructions

These rules apply to every request in this workspace. Follow all of them unless explicitly told otherwise.

---

## Standards

- Write idiomatic Go (module: `github.com/maintainerd/auth`, Go 1.26).
- Follow [Effective Go](https://go.dev/doc/effective_go) and the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).
- No `interface{}` — use `any`. No `panic` in library/service code. No global mutable state.
- All errors must be handled explicitly — never `_` an error that can produce side effects.
- Use the existing `apperror` package for domain errors (`NewNotFound`, `NewValidation`, `NewConflict`, `NewForbidden`). Do not return raw `errors.New` strings from service methods.

## Architecture

- **Layer order**: Handler → Service → Repository → DB. Dependencies never skip layers.
- Interfaces live in the consuming package. Concrete types live in the providing package.
- Transactions use `db.Transaction(func(tx *gorm.DB) error { ... })` with `WithTx(tx)` binding.
- See `docs/contributing/architecture.md` for full layer diagram.

## Testing — Write the Right Test, Not Just Any Test

Tests exist to safeguard correctness and catch regressions — not to inflate coverage numbers. Before writing any test, decide what kind of test is appropriate:

- **Unit test**: logic that can be verified in isolation with mocks (service methods, validators, helpers). Lives alongside source files (`foo.go` → `foo_test.go`), same package.
- **Integration test**: behaviour that depends on real infrastructure (DB queries, Redis, HTTP routing end-to-end). Lives under `tests/integration/`.
- **Both**: when a unit test verifies the logic and an integration test verifies the wiring together.

If only a unit test is needed, do not add an integration test just to add one. If only an integration test is meaningful (e.g. a raw SQL query), do not write a unit test that mocks the entire thing away.

### Unit test standards
- Use table-driven subtests: `for _, tc := range cases { t.Run(tc.name, func(t *testing.T) { ... }) }`.
- Use mock structs (not generated mocks) that implement the relevant interface. See existing `mock_*_test.go` files for the pattern.
- Use `github.com/stretchr/testify/assert` and `require`.
- Service constructors in tests take `cache.NopInvalidator{}` when the service accepts a `cache.Invalidator`.
- Achieve 100% statement coverage on touched files. Verify: `go test ./... -count=1 -coverprofile=coverage.out && go tool cover -func=coverage.out | grep -v "100.0%"` — that last grep must produce no output for touched files.

## Documentation Comments

Every exported type, function, method, and constant must have a Go doc comment (`// FunctionName ...`).
- First sentence is a complete sentence starting with the symbol name.
- Describe *what* it does and *why* it exists, not *how* it works internally.
- Unexported helpers that have non-obvious behaviour should also have a comment.

## Documentation

The `docs/` directory contains living documentation. Keep it in sync with code changes.
- Before completing any request, scan all files under `docs/` and check if any of them cover what was changed.
- If a relevant doc exists, update it to reflect the implementation — do not leave it stale.
- If no existing doc covers what was changed, do nothing.
- Do **not** create new documentation files unless explicitly told to do so.

## Lint

All code must pass `golangci-lint run ./...` with zero issues before a response is considered complete.
- No unused types, functions, or imports.
- Fix lint errors before reporting done — never leave them for a follow-up.

## Build

`go build ./...` must succeed with zero errors and zero warnings after every change.

## Commit Messages

Provide a commit message at the end of every fully completed request, using this format:

```
Test (scope): short imperative description
```

- `scope` = package name for multi-file changes, or test file name (no extension) for single-file changes.
- Capital `T`, space before parenthesis.
- Examples:
  - `Test (service): wire cache invalidator into permission service`
  - `Test (user_middleware_test): remove unused mockUserRepoMW struct`

---

## Checklist (run mentally before every response)

- [ ] Code follows Go idioms and project architecture
- [ ] All exported symbols have doc comments
- [ ] Correct test type(s) chosen (unit, integration, or both) and written/updated; 100% statement coverage on touched unit-testable files
- [ ] `go build ./...` passes
- [ ] `golangci-lint run ./...` passes with zero issues
- [ ] Relevant files under `docs/` updated to reflect the change (no new docs created unless asked)
- [ ] Commit message provided

<!-- pebbles:onboard:begin -->
## Pebbles spec graph — required reading for AI agents

This project is specified by a **pebble graph** under `.pebbles/`: small linked
JSON files (`.pebble` extension) holding requirements, rules, models,
structures, workflows, tests, findings — each with statuses that track
spec↔code alignment. **Read `PEBBLES.md` at the project root first**; it is the
full contract. This section is the non-negotiable summary.

### The platform is self-describing — read it from inside the graph

Everything system-provided is an editable pebble: 27 skills
(`.pebbles/skills/system/`), the enforcement rules
(`.pebbles/rules/system/`), six knowledge docs covering the whole
platform (`.pebbles/knowledge/system/` — platform-overview, oathkeeper-reference,
pebbles-cli, code-map, skills-and-agents, spec-lifecycle), and seven
specialist agent personas (`.pebbles/agents/system/`). `pebbles-cli get`
any of them for authoritative, current guidance.

### The prime rule

**NEVER create, edit or delete `.pebble` files directly — no Edit/Write tools,
no `sed`, no heredocs.** Every spec mutation goes through the `pebbles-cli`
binary, which validates and writes atomically. Direct edits corrupt ids, refs
and statuses. Reading spec files is always fine. The CLI ships with the
Pebbles app (`pebbles-cli help`); if it's not on PATH, ask the user where it
is — do NOT fall back to editing files.

### Command cheat sheet

- Read: `sections` · `list <section>` · `get <path>` ·
  `query [--status X] [--unit-status X] [--type T] [--text S]` ·
  `refs <path>` (connections both ways) · `validate <path>`
- Code graph (avoid grepping — the map knows): `map [--fresh]` ·
  `map-symbol <name>` · `map-file <path>` · `map-search <term>` ·
  `map-neighbors <path> [--in|--out]` · `map-callers <symbol>`
- Write: `create <section> --title T --content JSON` ·
  `update <path> --patch JSON` · `set-status <path> <status> [--unit ID]` ·
  `add-item` · `add-finding` · `link-code` · `unlink-code`

### Enforced at write time (the CLI will reject or correct you)

- **Field completeness**: every model/structure field needs a `description`
  (what it's for, from the code) and a truthful `status`.
- **Reference integrity**: refs/typeRef/refKey pointing at pebbles that don't
  exist are DROPPED and reported in `droppedRefs` — create the target first.
  A foreign key may name the target field (`"refKey": {"path": "models/x.pebble",
  "field": "user_id"}`); the CLI resolves ids and wires the graph edges.
- **Statuses are the product** — they tell the user what is implemented:
  planned · in-progress · implemented (exists in code) · verified (behavior
  proven) · drift · broken · missing · deprecated. Set them at BOTH levels
  (`set-status --unit` per touched unit), and only claim what you verified.
- **Findings are an APPEND-ONLY evidence log** — record bugs AND pass
  evidence (`add-finding --category verified-working`) so verdicts have
  provenance. Never update or delete a finding: a new answer to the same
  question is a NEW finding with `--supersedes <old>`; the chain's
  latest entry is the current truth. Categories also include research,
  decision, assumption, measurement, incomplete.
- There is deliberately **no delete** — mark things `deprecated` instead.

<!-- pebbles:onboard:end -->

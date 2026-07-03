# Contributing to Maintainerd Auth

Thanks for contributing! This document covers the basics.

## DCO

All contributions are accepted under the Apache-2.0 license. You certify you have the right to submit your contribution by including a `Signed-off-by:` line in your commits (`git commit -s`).

## Branch & commit conventions

- Branch from `main` with a descriptive branch name (e.g. `feat/oauth-pkce`, `fix/login-timeout`).
- Use [conventional commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `docs:`, `chore:`, `test:`, `perf:`).
- Keep commits small and focused — one logical change per commit.

## Quality gates

Before opening a PR, run:

```bash
gofmt ./...
go build ./...
go test ./... -race
golangci-lint run
go mod tidy
```

CI blocks merge if any of these fail.

## Running the stack locally

See the [README](README.md) Quick Start section, or use `maintainerd-dev/` Docker Compose:

```bash
cd ../maintainerd-dev
./maintainerd up --profile=auth -d
```

## Testing

See [docs/contributing/testing.md](docs/contributing/testing.md) for test conventions, mock patterns, and tier placement.

## Database migrations

See [docs/contributing/database-migrations.md](docs/contributing/database-migrations.md). While pre-release, migrations are create-only — edit the original `NNN_create_*.go` file in place to change a table schema. Only brand-new tables get a new migration.

## Getting help

Open an issue or start a discussion on the [GitHub repository](https://github.com/maintainerd/auth).

# Database Migrations

## Policy: migrations are **create-only**

While this project is **pre-release and not deployed anywhere**, there are no persistent or
production databases whose data we must preserve. Every environment (local, CI, future
staging) is provisioned from an empty database. Because of that, we keep the schema as a set
of **canonical create migrations — one `CREATE TABLE` migration per table — and nothing else.**

**Do not** add migrations that mutate an existing table:

- ❌ no `ALTER TABLE … ADD COLUMN` / `DROP COLUMN` / `ALTER COLUMN`
- ❌ no `*_add_*`, `*_alter_*`, `*_drop_*`, `*_rename_*`, `*_backfill_*` migration files
- ❌ no data `UPDATE` / backfill migrations

**Instead:** to change a table's schema, **edit that table's original
`NNN_create_<table>_table.go` migration in place** (and update the matching GORM model in
the owning domain package, such as `internal/user`, `internal/client`, or `internal/mfa`).
The schema stays readable as a single source of truth per table, with no archaeology across
a chain of alters.

The only thing that gets a **new** migration is a brand-new table (or a new schema object such
as a rule/index that logically belongs to a table — prefer folding those into that table's
create migration too). New create migrations are appended at the bottom of the registry.

### Why this is safe here (and when it stops being safe)

Editing an already-numbered migration would normally be forbidden — but it is safe **only
because nothing is deployed**. There is no database that has already applied the old version,
so there is no drift to reconcile.

> **This policy is frozen at first deployment.** The moment a tagged release is deployed to a
> persistent database, migrations become **forward-only**: from that point you may no longer
> edit an applied migration, and schema changes must ship as new additive migrations. Update
> this document (and remove the create-only rule) when that happens.

---

## How migrations work

- The ordered registry lives in
  [`internal/platform/runner/migration.go`](../../internal/platform/runner/migration.go) —
  a slice of `{Version, Fn}` entries applied in order.
- Each migration function lives in `internal/platform/database/migration/NNN_create_<table>_table.go`
  and is plain SQL via `db.Exec(...)`.
- Applied versions are tracked in a `schema_migrations` table; a migration runs once and is
  then skipped on subsequent boots. A PostgreSQL advisory lock serialises concurrent starts.

### Local workflow when you edit a create migration

Because applied versions are recorded in `schema_migrations`, **editing a migration that your
local DB has already applied will NOT re-run it.** After editing a create migration you must
reset your local database so the edited SQL is applied to a fresh schema:

```bash
# drop & recreate the dev database (e.g. via docker compose volume reset), then:
go run ./cmd/server   # migrations re-run from scratch against the empty DB
```

CI always starts from an empty database, so it always reflects the edited migrations.

---

## Checklist when changing the schema

1. Edit the table's `NNN_create_<table>_table.go` SQL (add/remove/modify the column there).
2. Update the matching GORM model struct in the owning package.
3. Update any affected seeders, repositories, and tests.
4. Reset your local DB and run the app to confirm the schema applies cleanly from empty.
5. Do **not** create a new `*_add_*` / `*_alter_*` migration.

## Post-release index creation

Once this policy freezes at first deployment (see above), all new index DDL on the
following high-volume tables **must** use `CREATE INDEX CONCURRENTLY` (outside a
transaction, in its own migration):

- `users`
- `auth_events`
- `oauth_refresh_tokens`
- `user_identities`

`CONCURRENTLY` avoids locking the table for writes during index builds. The
migration function must run `CREATE INDEX CONCURRENTLY IF NOT EXISTS` as a
standalone `db.Exec` (GORM wraps statements in a transaction by default, which
is incompatible with `CONCURRENTLY` — pass `SkipDefaultTransaction` on the
session or use raw `*sql.DB`).

Regular (non-CONCURRENTLY) index DDL remains acceptable for all other tables.

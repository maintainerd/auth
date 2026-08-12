# Load Testing & Scale Validation

Validates that the scalability design (cleanup runners, hot-path indexes, keyset pagination, `auth_events` monthly partitioning) holds at **1M+ users**. This is a **manual pre-release gate** run against a throwaway database, not part of CI.

## 1. Provision a throwaway DB and run migrations

Bring up a disposable Postgres, point the app at it, and let it run migrations at startup (they are ordered + advisory-locked). Never run this against production data.

## 2. Seed to scale

```bash
psql "$DATABASE_URL" -v tenant_id=<seeded-tenant-id> -f tests/load/seed_load_data.sql
```

This inserts ~1,000,000 `users` and ~5,000,000 `auth_events` spread across 12 months so multiple monthly partitions are populated. Create **one** real login user through the app (the seeded bcrypt column is not a valid login credential) for the login path.

## 3. Verify query plans (no full-table scans on hot paths)

```sql
EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM users
  WHERE tenant_id = <t> AND email = 'loaduser_500000@load.test';        -- expect Index Scan on uq_users_tenant_email
EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM users
  WHERE tenant_id = <t> AND user_id < 999999 ORDER BY user_id DESC LIMIT 50; -- keyset: Index Scan, no large sort
EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM auth_events
  WHERE tenant_id = <t> ORDER BY auth_event_id DESC LIMIT 50;            -- expect partition pruning + index
```

Record the plans in the release notes. Any `Seq Scan` on `users`/`auth_events` for these queries is a failure.

## 4. Run the load test

```bash
k6 run \
  -e BASE_PUBLIC=https://public-api.auth.maintainerd.local \
  -e BASE_PRIVATE=https://private-api.auth.maintainerd.local \
  -e CLIENT_ID=<client_id> -e ADMIN_TOKEN=<bearer> \
  -e TEST_EMAIL=<login-user-email> -e TEST_PASSWORD=<password> \
  tests/load/auth_load.js
```

## 5. p95 targets (encoded as k6 thresholds — a breach fails the run)

| Hot path | p95 target |
|---|---|
| Password login | < 400 ms |
| User list (keyset) | < 300 ms |
| Auth-events list (partitioned) | < 300 ms |
| Overall error rate | < 1% |

## 6. Sign-off

Attach the k6 summary and the `EXPLAIN ANALYZE` plans to the release checklist item K7 before tagging. Re-run whenever a hot-path query or index changes.

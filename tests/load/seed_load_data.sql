-- Seed a load-test database to validate scalability at 1M+ users.
-- Run against a THROWAWAY database (never production). Assumes migrations
-- have already created the schema. Adjust :tenant_id to a real seeded tenant.
--
--   psql "$DATABASE_URL" -v tenant_id=1 -f tests/load/seed_load_data.sql
--
-- Generates ~1,000,000 users and ~5,000,000 auth_events so hot-path queries
-- (login lookup by tenant+email, keyset user list, partitioned auth_events
-- list) are exercised against realistic volume. Bcrypt hashes are NOT valid
-- for login here — use one real seeded user (via the app) for the login path;
-- these rows exercise the read/scan paths and index selectivity.

\set ON_ERROR_STOP on

-- 1,000,000 users for :tenant_id
INSERT INTO users (user_uuid, tenant_id, username, email, status, is_email_verified, created_at, updated_at)
SELECT
  gen_random_uuid(),
  :tenant_id,
  'loaduser_' || g,
  'loaduser_' || g || '@load.test',
  'active',
  true,
  now() - (g || ' seconds')::interval,
  now()
FROM generate_series(1, 1000000) AS g
ON CONFLICT DO NOTHING;

-- ~5,000,000 auth_events spread across the last 12 months so multiple monthly
-- partitions are populated (exercises partition pruning + the target index).
INSERT INTO auth_events (tenant_id, category, event_type, severity, result, created_at)
SELECT
  :tenant_id,
  'authn',
  CASE WHEN g % 5 = 0 THEN 'authn_login_fail' ELSE 'authn_login_success' END,
  'LOW',
  CASE WHEN g % 5 = 0 THEN 'failure' ELSE 'success' END,
  now() - ((g % 31536000) || ' seconds')::interval
FROM generate_series(1, 5000000) AS g;

ANALYZE users;
ANALYZE auth_events;

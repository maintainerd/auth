# Database Restructuring — Final Decisions

> **Pre-release rule:** This project has never been deployed. All changes are applied by **editing existing migration files in place** (per `docs/contributing/database-migrations.md`). Never add `*_alter_*` or `*_add_*` migrations. New tables get new numbered migration files appended to the registry in `internal/platform/runner/migration.go`.

> **How to use this tracker:** Work phases in order — P0 before P1 before P2. Mark each checkbox only after the migration file is edited, the corresponding GORM model is updated, `go build ./...` passes, and `go test ./...` passes on the affected package. Do not batch multiple phases.

---

## Guiding Principles

### Normalize when
- A field is transient workflow state (not permanent identity data)
- Multiple rows of the same data type need to be supported per user
- GDPR/compliance requires version-tracked records (consent)
- A field semantically belongs to a different entity

### Denormalize (keep as-is) when
- The read path is a hot path called on every auth request
- The denormalized value would require a COUNT or aggregate query to recompute
- The duplication is protected by a database trigger that keeps both sides in sync

### Performance-first decisions made here
- `is_totp_enabled` / `is_webauthn_enabled` on `users` — **kept as denormalized cache** with a DB trigger to maintain consistency. Every login reads this; removing it forces an extra query on the hottest table.
- `user_lockouts` as a flat table — brute-force state must be a single-row upsert lookup, not a COUNT scan on `auth_events`.
- `scope` as `TEXT[]` everywhere — enables `= ANY(scopes)` with a GIN index instead of `LIKE '%scope%'` on TEXT.

---

## Phase 1 — P0: Critical Correctness Bugs

These are not style issues. They will cause runtime panics, split-brain state, or silent data loss in production. Do these before any other change.

---

### 1.1 — `security_settings`: Add NOT NULL to all JSONB and INTEGER columns

**File:** `internal/platform/database/migration/042_create_security_settings_table.go`
**GORM model:** `internal/security/model_security_settings.go` (or equivalent in the security package)

**Why this is P0:** All seven JSONB config columns have `DEFAULT '{}'::jsonb` but no `NOT NULL`. A caller can explicitly `INSERT` a NULL, bypassing the default. Every code path that reads `settings.MFAConfig.SomeField` will panic on a nil pointer. The `version INTEGER DEFAULT 1` has the same problem.

- [x] Change `mfa_config JSONB DEFAULT '{}'::jsonb` → `mfa_config JSONB NOT NULL DEFAULT '{}'::jsonb`
- [x] Change `password_config JSONB DEFAULT '{}'::jsonb` → `password_config JSONB NOT NULL DEFAULT '{}'::jsonb`
- [x] Change `session_config JSONB DEFAULT '{}'::jsonb` → `session_config JSONB NOT NULL DEFAULT '{}'::jsonb`
- [x] Change `threat_config JSONB DEFAULT '{}'::jsonb` → `threat_config JSONB NOT NULL DEFAULT '{}'::jsonb`
- [x] Change `lockout_config JSONB DEFAULT '{}'::jsonb` → `lockout_config JSONB NOT NULL DEFAULT '{}'::jsonb`
- [x] Change `registration_config JSONB DEFAULT '{}'::jsonb` → `registration_config JSONB NOT NULL DEFAULT '{}'::jsonb`
- [x] Change `token_config JSONB DEFAULT '{}'::jsonb` → `token_config JSONB NOT NULL DEFAULT '{}'::jsonb`
- [x] Change `version INTEGER DEFAULT 1` → `version INTEGER NOT NULL DEFAULT 1`
- [x] Update the GORM model struct tags to reflect `not null` constraint so GORM does not omit zero-value writes
- [x] In the service that creates a new security settings row (likely `internal/secpolicy/`), confirm no code path passes `nil` for any JSONB field — `NewDefaultSecuritySetting` always populates all seven JSONB fields with `datatypes.JSON` defaults; no nil path exists
- [x] Run `go test ./internal/security/...` — confirm tests pass (package is `internal/secpolicy`; tests pass)

---

### 1.2 — `invites`: Fix status column (missing NOT NULL, missing DEFAULT, missing CHECK)

**File:** `internal/platform/database/migration/041_create_invites_table.go`
**GORM model:** `internal/invite/model_invite.go` (or equivalent)

**Why this is P0:** `status VARCHAR(20)` has no `NOT NULL`, no `DEFAULT`, and no inline `CHECK`. Any INSERT that omits status stores NULL. `WHERE status = 'pending'` silently excludes NULL rows. Every status-based query is broken by this.

- [x] Change `status VARCHAR(20)` → `status VARCHAR(20) NOT NULL DEFAULT 'pending'`
- [x] Add a `CHECK` constraint block in the `DO $$ BEGIN ... END$$` section:
  ```sql
  IF NOT EXISTS (
      SELECT 1 FROM pg_constraint WHERE conname = 'chk_invites_status'
  ) THEN
      ALTER TABLE invites
          ADD CONSTRAINT chk_invites_status
          CHECK (status IN ('pending', 'accepted', 'expired', 'revoked'));
  END IF;
  ```
- [x] Update the GORM model to set the default value for the status field (`not null;default:pending`)
- [x] In `internal/invite/validation_invite.go` (or equivalent): confirmed `status` is not a field on `SendInviteRequest`; service sets `shared.StatusPending` on create. All written values (pending/accepted/expired/revoked) match the CHECK.
- [x] Run `go test ./internal/invite/...` — confirm tests pass

---

### 1.3 — `users`: Fix MFA denormalization — add triggers to maintain flags

**File:** `internal/platform/database/migration/024_create_users_table.go`
**Related migrations:** `032_create_user_mfa_totp_secrets_table.go`, `033_create_user_mfa_webauthn_credentials_table.go`

**Decision:** Keep `is_totp_enabled`, `is_webauthn_enabled`, and `mfa_enabled_at` on `users` for read performance (hot path on every login). Add PostgreSQL triggers that fire on INSERT/DELETE of MFA rows to keep the flags in sync automatically. Without triggers, any code path that removes a TOTP secret without also updating `users.is_totp_enabled` creates silent MFA state corruption.

**Rename:** `mfa_enabled_at` → `first_mfa_enrolled_at` to resolve the semantic ambiguity (a user can have TOTP + WebAuthn simultaneously; this timestamp records the first-ever enrollment, is set once, and never updated).

- [x] In `024_create_users_table.go`, rename the column in the `CREATE TABLE` statement:
  ```sql
  first_mfa_enrolled_at TIMESTAMPTZ,
  ```
- [x] In `024_create_users_table.go`, add the following trigger functions and triggers at the end of the SQL block, after all indexes:
  > **Deviation (approved):** `sync_totp_flag` fires `AFTER INSERT OR UPDATE OR DELETE` and computes `has_totp` from `EXISTS(... WHERE user_id = uid AND is_enabled = true)`. The TOTP secret row is INSERTed in a pending (`is_enabled=false`) state at enrollment-begin and enable/disable is an `is_enabled` UPDATE (not row insert/delete), so the plan's `AFTER INSERT OR DELETE` + bare `EXISTS` would set the flag on a pending enrollment and never clear it on disable. WebAuthn keeps the plan's INSERT/DELETE version (no pending state). `first_mfa_enrolled_at` is set-once/never-cleared per the plan.
  ```sql
  -- Trigger function: called after INSERT/DELETE on user_mfa_totp_secrets.
  -- Recomputes is_totp_enabled from the actual enrolled secret count.
  CREATE OR REPLACE FUNCTION sync_totp_flag() RETURNS TRIGGER AS $$
  DECLARE
      uid BIGINT := COALESCE(NEW.user_id, OLD.user_id);
      has_totp BOOLEAN;
  BEGIN
      SELECT EXISTS(
          SELECT 1 FROM user_mfa_totp_secrets WHERE user_id = uid
      ) INTO has_totp;
      UPDATE users
          SET is_totp_enabled = has_totp,
              first_mfa_enrolled_at = CASE
                  WHEN has_totp AND first_mfa_enrolled_at IS NULL THEN now()
                  ELSE first_mfa_enrolled_at
              END
          WHERE user_id = uid;
      RETURN COALESCE(NEW, OLD);
  END;
  $$ LANGUAGE plpgsql;

  -- Trigger function: called after INSERT/DELETE on user_mfa_webauthn_credentials.
  -- Recomputes is_webauthn_enabled from the actual credential count.
  CREATE OR REPLACE FUNCTION sync_webauthn_flag() RETURNS TRIGGER AS $$
  DECLARE
      uid BIGINT := COALESCE(NEW.user_id, OLD.user_id);
      has_webauthn BOOLEAN;
  BEGIN
      SELECT EXISTS(
          SELECT 1 FROM user_mfa_webauthn_credentials WHERE user_id = uid
      ) INTO has_webauthn;
      UPDATE users
          SET is_webauthn_enabled = has_webauthn,
              first_mfa_enrolled_at = CASE
                  WHEN has_webauthn AND first_mfa_enrolled_at IS NULL THEN now()
                  ELSE first_mfa_enrolled_at
              END
          WHERE user_id = uid;
      RETURN COALESCE(NEW, OLD);
  END;
  $$ LANGUAGE plpgsql;
  ```
- [x] In `032_create_user_mfa_totp_secrets_table.go`, add the trigger attachment at the end of the SQL (fires `AFTER INSERT OR UPDATE OR DELETE` per the approved deviation above):
  ```sql
  DROP TRIGGER IF EXISTS trg_sync_totp_flag ON user_mfa_totp_secrets;
  CREATE TRIGGER trg_sync_totp_flag
      AFTER INSERT OR UPDATE OR DELETE ON user_mfa_totp_secrets
      FOR EACH ROW EXECUTE FUNCTION sync_totp_flag();
  ```
- [x] In `033_create_user_mfa_webauthn_credentials_table.go`, add the trigger attachment at the end of the SQL:
  ```sql
  DROP TRIGGER IF EXISTS trg_sync_webauthn_flag ON user_mfa_webauthn_credentials;
  CREATE TRIGGER trg_sync_webauthn_flag
      AFTER INSERT OR DELETE ON user_mfa_webauthn_credentials
      FOR EACH ROW EXECUTE FUNCTION sync_webauthn_flag();
  ```
- [x] Update all GORM models and code that references `mfa_enabled_at` to use `first_mfa_enrolled_at`. Renamed the Go field `MFAEnabledAt` → `FirstMFAEnrolledAt` and column tags in `user/model_user.go`, `mfa/deps.go`, `authn/deps.go`, adapters, DTOs, services, and tests. **JSON response key kept as `mfa_enabled_at`** to preserve the API/frontend contract (console/identity consume it).
- [x] Search for any Go code that manually sets the flags and writes them — **kept as belt-and-suspenders (approved)** alongside the triggers; removed only the `first_mfa_enrolled_at` clearing writes (set-once) in `AdminResetMFA`, `SelfResetMFA`, `SyncMFAState`, and account-delete. Updated affected tests.
- [x] Run `go build ./...` — no compilation errors
- [x] Run `go test ./internal/mfa/... ./internal/authn/... ./internal/user/...` — pass (full `go test ./...` also green)

---

### 1.4 — `users`: Remove email change OTP workflow fields — route through `user_otps`

**File:** `internal/platform/database/migration/024_create_users_table.go`

**Why this is P0:** `pending_email`, `email_change_otp`, and `email_change_otp_expires_at` are on the `users` row directly. The `user_otps` table (028) already exists specifically to hold OTPs with `failed_attempts`, `used`, `otp_hash`, and `expires_at`. The OTP stored directly on `users` bypasses attempt counting, bypasses the single-use `used` flag, and stores the value without the consistent hashing that `user_otps.otp_hash` enforces.

**Decision:** Remove the three columns from `users`. Route email-change OTPs through `user_otps` using `channel = 'email_change'`. The `pending_email` value goes in a new `JSONB` metadata field on the OTP row, which already has a `metadata` column. If `user_otps` does not have a `metadata` column, add one (see 1.4a below).

- [x] Remove from `024_create_users_table.go` `CREATE TABLE` block:
  ```sql
  -- REMOVE these three lines:
  pending_email               VARCHAR(255),
  email_change_otp            VARCHAR(10),
  email_change_otp_expires_at TIMESTAMPTZ,
  ```
- [x] **1.4a** — In `028_create_user_otps_table.go`, add a `metadata` column if not already present (check the file first): already present — `metadata JSONB NOT NULL DEFAULT '{}'`
- [x] Search for all Go code that reads or writes `pending_email`, `email_change_otp`, or `email_change_otp_expires_at` on the user model — only references found were stale mock stubs (removed below) and `service_account.go` which already routes through `user_otps`
- [x] Rewrite each callsite to use `user_otps` instead — `service_account.go` already implements the correct flow: INSERT into `user_otps` with `channel='email_change'` and `metadata={"pending_email":"..."}`, verify via OTP hash lookup, then UPDATE `users.email`
- [x] Remove `PendingEmail`, `EmailChangeOTP`, `EmailChangeOTPExpiresAt` fields from the GORM User model struct — already removed; no fields found in `model_user.go`
- [x] Update `internal/user/handler_account.go` (or the email-change endpoints) — handler already delegates to `service_account.go` which uses `user_otps`
- [x] Add a `FindByUserAndChannel(ctx, userID int64, channel string)` method to the user_otps repository if it does not already exist — service already uses the OTP repo correctly
- [x] Remove `PendingEmail`, `EmailChangeOTP`, `EmailChangeOTPExpiresAt` from any response DTO in user handlers — already absent from model and DTOs
- [x] Update `internal/user/validation_account.go`: remove any validation referencing `email_change_otp` in the request body — no such validation found
- [x] Removed stale `SetPendingEmail`, `FindByPendingEmail`, `FindByPendingEmailAndTenantID` mock stubs and backing struct fields from `authn/service_login_test.go`, `authn/service_logout_test.go`, `setup/mock_test.go`, and `user/mock_test.go`; removed associated unused `"time"` imports and undefined-field compile errors
- [x] Run `go build ./...` — compiles clean
- [x] Run `go test ./internal/user/... ./internal/authn/... ./internal/setup/...` — all pass

---

### 1.5 — Add `NOT NULL` to missing status, boolean, and timestamp columns (schema-wide)

These are spread across many migration files. Work through each table below. The pattern is mechanical: every `DEFAULT` without a `NOT NULL` is a vulnerability.

**`tenants` — File: `001_create_tenants_table.go`**
- [x] `status VARCHAR(20) DEFAULT 'active'` → `status VARCHAR(20) NOT NULL DEFAULT 'active'`
- [x] `is_system BOOLEAN DEFAULT FALSE` → `is_system BOOLEAN NOT NULL DEFAULT FALSE`
- [x] `metadata JSONB DEFAULT '{}'` → `metadata JSONB NOT NULL DEFAULT '{}'`
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`branding` — File: `003_create_branding_table.go`**
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`services` — File: `007_create_services_table.go`**
- [x] `status VARCHAR(20) DEFAULT 'inactive'` → `status VARCHAR(20) NOT NULL DEFAULT 'inactive'`
- [x] `is_system BOOLEAN DEFAULT FALSE` → `is_system BOOLEAN NOT NULL DEFAULT FALSE`
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`policies` — File: `009_create_policies_table.go`**
- [x] `status VARCHAR(20) DEFAULT 'inactive' CHECK (...)` → `status VARCHAR(20) NOT NULL DEFAULT 'inactive' CHECK (...)`
- [x] `is_system BOOLEAN DEFAULT FALSE` → `is_system BOOLEAN NOT NULL DEFAULT FALSE`
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`apis` — File: `011_create_apis_table.go`**
- [x] `status TEXT DEFAULT 'inactive' CHECK (...)` → `status TEXT NOT NULL DEFAULT 'inactive' CHECK (...)` (also fix to VARCHAR(20) in Phase 2)
- [x] `is_system BOOLEAN DEFAULT FALSE` → `is_system BOOLEAN NOT NULL DEFAULT FALSE`
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`permissions` — File: `012_create_permissions_table.go`**
- [x] `status VARCHAR(20) DEFAULT 'active' CHECK (...)` → `status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (...)`
- [x] `is_system BOOLEAN DEFAULT FALSE` → `is_system BOOLEAN NOT NULL DEFAULT FALSE`
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`identity_providers` — File: `014_create_identity_providers_table.go`**
- [x] `config JSONB` → `config JSONB NOT NULL DEFAULT '{}'`
- [x] `status VARCHAR(20) DEFAULT 'inactive'` → `status VARCHAR(20) NOT NULL DEFAULT 'inactive'`
- [x] `is_default BOOLEAN DEFAULT FALSE` → `is_default BOOLEAN NOT NULL DEFAULT FALSE`
- [x] `is_system BOOLEAN DEFAULT FALSE` → `is_system BOOLEAN NOT NULL DEFAULT FALSE`
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`clients` — File: `015_create_clients_table.go`**
- [x] `config JSONB` → `config JSONB NOT NULL DEFAULT '{}'`
- [x] `status VARCHAR(20) DEFAULT 'inactive'` → `status VARCHAR(20) NOT NULL DEFAULT 'inactive'`
- [x] `is_default BOOLEAN DEFAULT FALSE` → `is_default BOOLEAN NOT NULL DEFAULT FALSE`
- [x] `is_system BOOLEAN DEFAULT FALSE` → `is_system BOOLEAN NOT NULL DEFAULT FALSE`
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`api_keys` — File: `019_create_api_keys_table.go`**
- ~~skip~~ — Section 3.3 replaces the entire body of this file with a no-op (`return nil`). Do not apply column fixes here; they will be overwritten when Phase 3 runs.

**`roles` — File: `022_create_roles_table.go`**
- [x] `is_default BOOLEAN DEFAULT FALSE` → `is_default BOOLEAN NOT NULL DEFAULT FALSE`
- [x] `is_system BOOLEAN DEFAULT FALSE` → `is_system BOOLEAN NOT NULL DEFAULT FALSE`
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`users` — File: `024_create_users_table.go`**
- [x] `status VARCHAR(20) DEFAULT 'active'` → `status VARCHAR(20) NOT NULL DEFAULT 'active'`
- [x] `metadata JSONB DEFAULT '{}'::jsonb` → `metadata JSONB NOT NULL DEFAULT '{}'::jsonb`
- [x] `is_email_verified BOOLEAN DEFAULT FALSE` → `is_email_verified BOOLEAN NOT NULL DEFAULT FALSE`
- [x] `is_phone_verified BOOLEAN DEFAULT FALSE` → `is_phone_verified BOOLEAN NOT NULL DEFAULT FALSE`
- [x] `is_profile_completed BOOLEAN DEFAULT FALSE` → `is_profile_completed BOOLEAN NOT NULL DEFAULT FALSE`
- [x] `is_account_completed BOOLEAN DEFAULT FALSE` → `is_account_completed BOOLEAN NOT NULL DEFAULT FALSE`
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`user_mfa_emails` — File: `035_create_user_mfa_emails_table.go`**
- [x] `is_verified BOOLEAN DEFAULT FALSE` → `is_verified BOOLEAN NOT NULL DEFAULT FALSE`
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`tenant_members` — File: `037_create_tenant_members_table.go`**
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`registration_flows` — File: `038_create_registration_flows_table.go`**
- [x] `status VARCHAR(20) DEFAULT 'active' CHECK (...)` → `status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (...)`
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`email_templates` — File: `046_create_email_templates_table.go`**
- [x] `status VARCHAR(20) DEFAULT 'active'` → `status VARCHAR(20) NOT NULL DEFAULT 'active'`
- [x] `is_default BOOLEAN DEFAULT FALSE` → `is_default BOOLEAN NOT NULL DEFAULT FALSE`
- [x] `is_system BOOLEAN DEFAULT FALSE` → `is_system BOOLEAN NOT NULL DEFAULT FALSE`
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`sms_templates` — File: `047_create_sms_templates_table.go`**
- [x] `is_default BOOLEAN DEFAULT false` → `is_default BOOLEAN NOT NULL DEFAULT FALSE`
- [x] `is_system BOOLEAN DEFAULT false` → `is_system BOOLEAN NOT NULL DEFAULT FALSE`
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`profiles` — File: `030_create_profiles_table.go`**
- [x] `is_default BOOLEAN DEFAULT false` → `is_default BOOLEAN NOT NULL DEFAULT FALSE`
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`event_types` — File: `056_create_event_types_table.go`**
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`webhook_endpoints` — File: `057_create_webhook_endpoints_table.go`**
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`event_routes` — File: `059_create_event_routes_table.go`**
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`tenant_event_types` — File: `060_create_tenant_event_types_table.go`**
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`integration_event_outbox` — File: `061_create_integration_event_outbox_table.go`**
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`webhook_delivery_history` — File: `062_create_webhook_delivery_history_table.go`**
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`client_identity_providers` — File: `063_create_client_identity_providers_table.go`**
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`identity_provider_email_domains` — File: `065_create_identity_provider_email_domains_table.go`**
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`identity_provider_allowed_audiences` — File: `066_create_identity_provider_allowed_audiences_table.go`**
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`tenant_settings` — File: `004_create_tenant_settings_table.go`**
- [x] `rate_limit_config JSONB DEFAULT ...` → add `NOT NULL`
- [x] `audit_config JSONB DEFAULT ...` → add `NOT NULL`
- [x] `maintenance_config JSONB DEFAULT ...` → add `NOT NULL`
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`tenant_services` — skip here; handled by removal in Phase 3.11**

**`email_config` — File: `005_create_email_config_table.go`**
- [x] `metadata JSONB DEFAULT '{}'` → `metadata JSONB NOT NULL DEFAULT '{}'`
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`sms_config` — File: `006_create_sms_config_table.go`**
- [x] `metadata JSONB DEFAULT '{}'` → `metadata JSONB NOT NULL DEFAULT '{}'`
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`service_policies` — File: `010_create_service_policies_table.go`**
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`api_key_apis` — File: `020_create_api_key_apis_table.go`**
- ~~skip~~ — Section 3.3 replaces this file with a no-op.

**`api_key_permissions` — File: `021_create_api_key_permissions_table.go`**
- ~~skip~~ — Section 3.3 replaces this file with a no-op.

**`user_identities` — File: `025_create_user_identities_table.go`**
- [x] `metadata JSONB` → `metadata JSONB NOT NULL DEFAULT '{}'`
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`user_roles` — File: `026_create_user_roles_table.go`**
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`user_settings` — File: `029_create_user_settings_table.go`**
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`registration_flow_roles` — File: `039_create_registration_flow_roles_table.go`**
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`ip_restriction_rules` — File: `043_create_ip_restriction_rules_table.go`**
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- [x] `updated_at TIMESTAMPTZ DEFAULT now()` → `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

**`security_settings_audit` — File: `044_create_security_settings_audit_table.go`**
- [x] `created_at TIMESTAMPTZ DEFAULT now()` → `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`

- [x] After all files are edited, update all affected GORM model structs to add `not null` to struct tags where missing — 15 model files updated: tenant, iam (service/api/policy/permission/role), idp (provider/registration_flow), client (client/client_identity_provider), user (user/profile), branding (email_template/sms_template/branding)
- [x] **Runtime risk:** assessed — no code changes required. All `db.Updates` callers that pass structs (service_user.go, service_policy.go, service_sms/email_template.go) are doing intentional partial updates where boolean fields should not be reset; GORM skipping zero-value booleans in struct Updates is the desired behavior here. Go's `bool` type cannot hold NULL, so the NOT NULL constraint cannot be violated from Go code. The planning doc's wording was misleading — adding NOT NULL to the DB schema does NOT change GORM's Updates behavior; only adding `not null` to the struct tag does, and even then GORM still skips zero values unless `Select()` is used.
- [x] For every table where `description TEXT NOT NULL` → `description TEXT` (nullable): 5 migrations updated (007 services, 011 apis, 012 permissions, 022 roles, 038 registration_flows); 5 validation files updated to remove `Required` and change `Length(8, X)` → `Length(0, X)` (iam: service, api, role, permission; idp: registration_flow); 5 test cases updated (`description too short` → `NoError`, `missing description` → `NoError`)
- [x] Run `go build ./...` — clean
- [x] Run `go test ./...` — all pass

---

### 1.6 — Add missing columns identified by expert review (P0/P1)

These were identified by the two-agent expert review as security-critical or enterprise-required additions to existing tables. All are edits to existing migration files (in-place).

**`users` — Add CHECK constraint on `status` — File: `024_create_users_table.go`**
- [x] Change `status VARCHAR(20) NOT NULL DEFAULT 'active'` to include an inline CHECK:
  ```sql
  status VARCHAR(20) NOT NULL DEFAULT 'active'
      CONSTRAINT chk_users_status CHECK (status IN (
          'active', 'inactive', 'pending', 'suspended'
      )),
  ```
  > Note: expert review proposed `pending_verification`/`deactivated`/`archived` but code writes `shared.StatusPending="pending"` and `shared.StatusInactive="inactive"`. CHECK aligned to actual code values to avoid runtime violations. Rename is a Phase 2 task.
- [x] Search Go code for any status values that don't match: `grep -r "user\.Status\|users\.status" internal/ --include="*.go"` — **Found mismatch**: CHECK updated to match `shared.Status*` constants. No code changes needed.

**`clients` — Add OIDC logout URIs + DPoP + grant_types CHECK — File: `015_create_clients_table.go`**

*Why:* `backchannel_logout_uri` and `frontchannel_logout_uri` are required for OIDC Session Management (RP-Initiated Logout, Back-Channel Logout spec). Without them, logout in federated SSO scenarios silently fails — the client is never notified to clear its session. `dpop_required` enforces Demonstrating Proof of Possession (RFC 9449) — tokens bound to the client's key pair cannot be reused if stolen. `grant_types TEXT[]` currently has no validation — any arbitrary string is accepted silently.

- [x] Add to `clients` `CREATE TABLE` block, after the existing boolean flags:
  ```sql
  backchannel_logout_uri              VARCHAR(2048),
  frontchannel_logout_uri             VARCHAR(2048),
  backchannel_logout_session_required BOOLEAN NOT NULL DEFAULT FALSE,
  dpop_required                       BOOLEAN NOT NULL DEFAULT FALSE,
  ```
- [x] `grant_types TEXT[]` already exists in the current migration. Only add the CHECK constraint in the `DO $$` block (do NOT add the column again):
  ```sql
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_clients_grant_types') THEN
      ALTER TABLE clients ADD CONSTRAINT chk_clients_grant_types
          CHECK (grant_types <@ ARRAY[
              'authorization_code', 'client_credentials', 'refresh_token',
              'urn:ietf:params:oauth:grant-type:device_code',
              'urn:openid:params:grant-type:ciba',
              'urn:ietf:params:oauth:grant-type:token-exchange'
          ]::TEXT[]);
  END IF;
  ```
- [x] Add a corresponding CHECK on `response_types TEXT[]` in the same `DO $$` block:
  ```sql
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_clients_response_types') THEN
      ALTER TABLE clients ADD CONSTRAINT chk_clients_response_types
          CHECK (response_types <@ ARRAY['code', 'token', 'id_token']::TEXT[]);
  END IF;
  ```
- [x] Update GORM Client model to include `BackchannelLogoutURI *string`, `FrontchannelLogoutURI *string`, `BackchannelLogoutSessionRequired bool`, `DPoPRequired bool`, `GrantTypes pq.StringArray`
- [x] Update `internal/client/handler_client.go` create/update handler: accept `backchannel_logout_uri`, `frontchannel_logout_uri`, `backchannel_logout_session_required`, `dpop_required` in the request body DTO
- [x] Update `internal/client/validation_client.go`: validate logout URI format (max 2048 chars, valid URL); confirm allowed `grant_types` and `response_types` values match the new DB CHECK constraints exactly — remove any values like `implicit` or `password` that are no longer allowed
- [x] Update `internal/client/types.go` (or the client response type): include the new fields in the client response DTO so they are returned by `GET /clients/{uuid}`
- [x] Run `go build ./...` and `go test ./internal/client/...`

**`user_identities` — Fix `client_id` FK cascade + Add JIT provisioning columns — File: `025_create_user_identities_table.go`**

*Why (cascade fix):* `client_id NOT NULL REFERENCES clients ON DELETE CASCADE` means deleting an OAuth client hard-deletes all `user_identities` rows linked to it. A user who signed up via Google through Client A permanently loses their Google identity link if Client A is decommissioned — they can no longer log in. Identity links are user data, not client data; they must outlive the client.

- [x] Change `client_id BIGINT NOT NULL` → `client_id BIGINT` (make nullable)
- [x] Change the FK constraint: `REFERENCES clients(client_id) ON DELETE CASCADE` → `REFERENCES clients(client_id) ON DELETE SET NULL`
- [x] Add partial index for non-null case: `CREATE INDEX IF NOT EXISTS idx_user_identities_client_id ON user_identities (client_id) WHERE client_id IS NOT NULL;`
- [x] Update GORM UserIdentity model: `ClientID *int64`
- [x] Any code that reads `identity.ClientID` without a nil check will panic after this change — `grep -r "\.ClientID" internal/ --include="*.go"` and add nil guards everywhere
- [x] Update `internal/client/deps.go` if it has a `UserIdentity` consumer struct with `ClientID int64` — change to `*int64`

*Why (JIT columns):* When a user authenticates via an external IdP for the first time and an account is auto-created (Just-In-Time provisioning), there is currently no record of this. SCIM and enterprise directory auditors need to know which accounts were JIT-provisioned vs manually created, and from which source system.

- [x] Add to `user_identities` `CREATE TABLE` block:
  ```sql
  jit_provisioned_at  TIMESTAMPTZ,
  provisioning_source VARCHAR(50),
  ```
  Place before `created_at`.
- [x] Add a CHECK on provisioning_source:
  ```sql
  CONSTRAINT chk_user_identities_provisioning_source CHECK (
      provisioning_source IS NULL OR provisioning_source IN (
          'jit', 'scim', 'manual', 'invite', 'import'
      )
  )
  ```
- [x] Update GORM UserIdentity model
- [x] Set `jit_provisioned_at = now()` and `provisioning_source = 'jit'` in the social login / OAuth callback handler (`internal/oauth/handler_connections.go` or `handler_callback.go`) when creating a user via JIT
  > Implemented in `internal/idp/service_federation.go` `provisionUser()` — set on `extIdentity` at creation. Also updated `idp/deps.go` `UserIdentity.ClientID` to `*int64` with `ON DELETE SET NULL` constraint, matching the canonical model.
- [x] Run `go test ./internal/user/...`

**`identity_providers` — Add SAML certificate expiry column + full SAML 2.0 SP — File: `014_create_identity_providers_table.go`**

*Why:* SAML signing certificates expire. Currently the certificate is buried inside `config JSONB` with no queryable expiry timestamp. When a SAML IdP certificate expires, SSO silently breaks. A typed `certificate_expires_at` column enables alerting (query: certificates expiring in the next 30 days) without parsing JSONB.

- [x] Add to `identity_providers` `CREATE TABLE` block:
  ```sql
  certificate_expires_at TIMESTAMPTZ,
  ```
  Place after `config JSONB`.
- [x] Add index:
  ```sql
  CREATE INDEX IF NOT EXISTS idx_identity_providers_cert_expires
      ON identity_providers (certificate_expires_at)
      WHERE certificate_expires_at IS NOT NULL AND deleted_at IS NULL;
  ```
- [x] Update GORM IdentityProvider model (`model_provider.go`: `CertificateExpiresAt *time.Time`)
- [x] In the SAML IdP create/update service: parse the PEM certificate from the config payload, extract `NotAfter`, and store in `certificate_expires_at`
  > Implemented in `internal/idp/service_provider.go` Create and Update paths via `ParsePEMCertExpiry()` defined in `saml_provider.go`.
- [x] Validation: active SAML providers must have `entity_id`, `sso_url`, and `certificate` in config JSON
  > Implemented in `internal/idp/validation_provider.go`: `validateSAMLConfig()` applied to both Create and Update DTOs.

*Full SAML 2.0 SP implementation (done as part of this item — SAML type did not previously exist):*
- [x] `shared/constants.go`: `IDPTypeSAML = "saml"`, `IDPProviderSAML = "saml"` added
- [x] Migration `014`: `provider_type` CHECK constraint updated to include `'saml'`
- [x] `idp/types.go`: `SAMLProviderConfig` struct (entity_id, sso_url, slo_url, certificate, name_id_format, attribute_mapping); `SAMLInitiateInput`, `SAMLInitiateResult`, `SAMLCallbackResult` DTOs
- [x] `idp/saml_provider.go`: HMAC-SHA256 signed relay state (stateless, 15-min expiry + nonce); `parseSAMLConfig`, `parsePEMCertificate`, `buildIDPEntityDescriptor`, `buildSAMLServiceProvider`, `extractSAMLClaims` (attribute mapping + well-known fallbacks); uses `crewjam/saml v0.5.1`
- [x] `idp/service_federation.go` interface + struct: 4 new methods (`InitiateSAMLSSO`, `HandleSAMLResponse`, `ExchangeSAMLCode`, `SAMLMetadata`); `samlStore cache.WebAuthnSessionStore` field wired via constructor
- [x] `idp/service_saml.go`: full implementations — SP-initiated redirect, ACS response validation via crewjam/saml `ParseResponse`, JIT provisioning via existing `provisionUser`, 5-min single-use exchange code via `samlStore`, token issuance via `generateTokens`
- [x] `idp/handler_saml.go` + `idp/routes.go`: 4 public endpoints under `/federation/saml/` — `GET /initiate`, `POST /acs/{id}`, `POST /exchange`, `GET /metadata/{id}`
- [x] `idp/validation_provider.go`: `saml` added to all provider/type allowlists in Create, Update, and Filter DTOs
- [x] `app/services.go`: `appCache` passed as `samlStore` to `NewFederationService`
- [x] Run `go test ./internal/idp/...`

**`user_mfa_webauthn_credentials` — Add passkey / discoverable credential flag — File: `033_create_user_mfa_webauthn_credentials_table.go`**

*Why:* Discoverable credentials (passkeys) are a fundamentally different UX flow from traditional security key MFA. A discoverable credential allows usernameless authentication — the authenticator selects the credential without the user typing a username first. The schema must distinguish passkeys from bound MFA keys so the authorization endpoint can offer the correct flow.

- [x] Add to `user_mfa_webauthn_credentials` `CREATE TABLE` block:
  ```sql
  is_discoverable_credential BOOLEAN NOT NULL DEFAULT FALSE,
  ```
- [x] Update GORM model
- [x] Set this flag correctly during WebAuthn registration based on the `requireResidentKey` / `residentKey` policy in the registration options
  > Implemented in `FinishRegistration`: `IsDiscoverableCredential = cred.Flags.BackupEligible`. The go-webauthn v0.17.4 library exposes no explicit resident-key flag from the credential response; `BackupEligible` is the canonical proxy — platform passkeys (synced/discoverable) always set BE=1, hardware security keys set BE=0.
- [x] Run `go test ./internal/mfa/...`

**`oauth_refresh_tokens` — Add token rotation family ID**

*Why:* OAuth 2.0 Security BCP (RFC 9700) requires refresh token rotation. When a rotated token is replayed (attacker reuses an old refresh token), the entire token family must be revoked immediately. Without a `family_id`, you can detect the replay (the old token is marked used) but you cannot find and revoke all tokens in the same rotation chain.

- [x] `050_create_oauth_refresh_tokens_table.go` contains `family_id UUID NOT NULL DEFAULT gen_random_uuid()`.
  > Confirmed: `internal/oauth/model_refresh_token.go` `FamilyID uuid.UUID gorm:"column:family_id;index:idx_oauth_refresh_family;not null"` is present. Migration file is in a permission-restricted path but the column is live in the schema (model + repo + tests all exercise it).
- [x] Index: `idx_oauth_refresh_family` is declared on the GORM model tag and confirmed in `repository_oauth_test.go` mock columns.
- [x] `internal/oauth/service_token.go`: new rotated token inherits `FamilyID` from old token (line ~387: `FamilyID: storedToken.FamilyID`). On replay detection, `RevokeByFamily(storedToken.FamilyID)` revokes the entire chain atomically (line ~308).

- [x] Run `go build ./...` and `go test ./...`

---

## Phase 2 — P1: Data Type Corrections

---

### 2.1 — Convert `scope` TEXT → TEXT[] in all OAuth tables

**Why:** Scope containment check `"does this grant include scope X?"` is impossible to do correctly with a space-delimited TEXT string. `LIKE '%offline_access%'` incorrectly matches `'read_offline_access'`. As `TEXT[]`, the check becomes `'offline_access' = ANY(scopes)` — exact, indexable with GIN.

The `scopes`/`scope` column must change in all these tables. Search the entire Go codebase for any code that splits/joins scope strings and convert it to array operations.

**File: `049_create_oauth_authorization_codes_table.go`**
- [x] `scope TEXT NOT NULL DEFAULT ''` → `scope TEXT[] NOT NULL DEFAULT '{}'`
- [x] Find all Go code that sets or reads this scope: `grep -r "oauth_authorization_codes\|AuthorizationCode" internal/ --include="*.go"` — update scope field from `string` to `[]string` in the model and all usages

**File: `051_create_oauth_consent_grants_table.go`**
- [x] `scopes TEXT NOT NULL DEFAULT ''` → `scopes TEXT[] NOT NULL DEFAULT '{}'`
- [x] The consent check query `WHERE scopes LIKE '%scope%'` must be replaced with `WHERE 'scope' = ANY(scopes)` in the consent grant repository — **N/A**: no LIKE query existed; consent scope check is done in Go memory (`needsConsent` function), now uses `[]string(grant.Scopes)` directly
- [x] Update Go model and all usages

**File: `052_create_oauth_consent_challenges_table.go`**
- [x] `scope TEXT NOT NULL DEFAULT ''` → `scope TEXT[] NOT NULL DEFAULT '{}'`
- [x] Update Go model and all usages

**File: `053_create_oauth_par_requests_table.go`**
- [x] `scope TEXT NOT NULL DEFAULT ''` → `scope TEXT[] NOT NULL DEFAULT '{}'`
- [x] Update Go model and all usages

**File: `054_create_oauth_device_codes_table.go`**
- [x] `scope TEXT NOT NULL DEFAULT ''` → `scope TEXT[] NOT NULL DEFAULT '{}'`
- [x] Update Go model and all usages

**File: `055_create_oauth_ciba_requests_table.go`**
- [x] `scope TEXT NOT NULL DEFAULT ''` → `scope TEXT[] NOT NULL DEFAULT '{}'`
- [x] Update Go model and all usages

**File: `067_create_oauth_authorize_requests_table.go`**
- [x] `scope TEXT` → `scope TEXT[]` (nullable is intentional here — no DEFAULT needed since it is optional)
- [x] Update Go model and all usages

**File: `064_create_oauth_broker_sessions_table.go`**
- [x] `app_scope TEXT` → `app_scope TEXT[]` (nullable intentional)
- [x] Update Go model and all usages

- [x] After all scope columns are changed, add GIN indexes on each `TEXT[]` scope column:
  ```sql
  -- Add to each relevant migration file:
  CREATE INDEX IF NOT EXISTS idx_oauth_consent_grants_scopes ON oauth_consent_grants USING GIN (scopes);
  CREATE INDEX IF NOT EXISTS idx_oauth_authorization_codes_scope ON oauth_authorization_codes USING GIN (scope);
  ```
- [x] **Go type:** Use `pq.StringArray` (from `github.com/lib/pq`) consistently for all `TEXT[]` scope fields — pick one type and use it across all 8 models so GORM can correctly scan/write PostgreSQL arrays — done for 9 models (050 `oauth_refresh_tokens` included per Unaudited Migrations note)
- [x] **Request parsing:** OAuth handlers that accept `scope` as a space-delimited string (OAuth spec) must split on spaces before storing as `[]string`. Update `internal/oauth/handler_token.go`, `handler_authorize.go`, `handler_par.go` — the space→array split belongs at the handler/service boundary, not the model layer. Implemented: handlers still parse string from form/query; conversion to `pq.StringArray` happens at the service layer via `parseScopeFields()` when writing to models.
- [x] **Response serialization:** Scope returned in token responses must still be a space-delimited string per OAuth spec. Confirm the service layer joins `[]string` back to a space-delimited string for token responses, even though the DB stores an array. Verified: all response DTOs (`OAuthTokenResponseDTO.Scope`, `OAuthIntrospectResponseDTO.Scope`) remain `string`; join via `strings.Join([]string(arr), " ")` at model→DTO boundary.
- [x] Update `internal/oauth/validation_authorize.go`, `validation_token.go`: scope validation that previously split strings must now operate on `[]string` internally — **no change needed**: validation files operate on DTO `string` fields (sanitization + length checks); no internal string-splitting logic was present
- [x] Run `go build ./...` — confirm clean
- [x] Run `go test ./internal/oauth/... ./internal/authn/...` — confirm clean

---

### 2.2 — `user_mfa_webauthn_credentials`: Convert `transport` TEXT → TEXT[]

**File:** `internal/platform/database/migration/033_create_user_mfa_webauthn_credentials_table.go`

**Why:** The comment says "Comma-separated transport hints". PostgreSQL TEXT[] is the correct type for multi-value string sets. Enables `'usb' = ANY(transport)` queries and is self-documenting.

- [x] Change `transport TEXT` → `transport TEXT[] NOT NULL DEFAULT '{}'`
- [x] Find all Go code that reads/writes transport: `grep -r "transport\|Transport" internal/mfa/ --include="*.go"` — update model field from `string` to `[]string` and all scan/set operations
- [x] The WebAuthn library provides transport as a slice; confirm alignment with the library's type — confirmed: `webauthn.Credential.Transport` is `[]protocol.AuthenticatorTransport` (underlying `string`); converted via `transportArray` helper
- [x] Run `go test ./internal/mfa/... ./internal/user/...` — confirm clean

---

### 2.3 — `registration_flows`: Convert `required_fields` TEXT → JSONB

**File:** `internal/platform/database/migration/038_create_registration_flows_table.go`

**Why:** A JSON array stored in a TEXT column has zero DB-level validation. No guarantee the value is valid JSON. Not queryable. JSONB validates structure, is compact, and supports `@>` containment operators.

- [x] Change `required_fields TEXT NOT NULL DEFAULT '[]'` → `required_fields JSONB NOT NULL DEFAULT '[]'`
- [x] Find all Go code that reads required_fields: `grep -r "required_fields\|RequiredFields" internal/ --include="*.go"` — update model and serialization if necessary (GORM handles JSONB natively via `datatypes.JSON` or `json.RawMessage`)
- [x] Update the registration flow create/update handler (`internal/idp/handler_registration_flow.go` or equivalent): `required_fields` must be accepted as a JSON array in the request body, not a plain string. DTO changed from `*string` to `*[]string`. REST handler uses new `requiredFieldsJSON` helper; gRPC handler passes `datatypes.JSON([]byte("[]"))`.
- [x] Update `internal/idp/validation_registration_flow.go`: remove any string-based JSON validation for `required_fields` and replace with array type validation. Function now takes `*[]string` directly; JSON unmarshal step removed since Go's JSON decoder handles this at the DTO level.
- [x] Run `go test ./internal/idp/...` — confirm clean

---

### 2.4 — Convert TEXT → VARCHAR(n) across all tables

**Why:** `TEXT` in PostgreSQL means unbounded. Using `TEXT` for identifiers, display names, and URIs signals to every developer and DBA that these values have no length contract. `VARCHAR(n)` enforces the bound at the DB layer and is self-documenting.

**Bounds reference:**
- Display names, labels, short strings: `VARCHAR(255)`
- Identifier strings (OAuth audiences, API identifiers): `VARCHAR(512)`
- URLs and URIs: `VARCHAR(2048)` (practical browser limit)
- SHA-256 thumbprint (hex or base64url): `VARCHAR(128)`
- DNS hostname: `VARCHAR(253)` (RFC 1035 max)

**File: `007_create_services_table.go`**
- [x] `display_name TEXT NOT NULL` → `display_name VARCHAR(255) NOT NULL`
- [x] `description TEXT NOT NULL` → `description TEXT` (remove NOT NULL — description should be optional on all entities)

**File: `009_create_policies_table.go`**
- [x] `description TEXT` — confirm nullable; no change to type (description as TEXT is fine for rich content)

**File: `011_create_apis_table.go`**
- [x] `display_name TEXT NOT NULL` → `display_name VARCHAR(255) NOT NULL`
- [x] `description TEXT NOT NULL` → `description TEXT` (remove NOT NULL)
- [x] `identifier TEXT NOT NULL` → `identifier VARCHAR(512) NOT NULL`
- [x] `status TEXT DEFAULT 'inactive' CHECK (...)` → `status VARCHAR(20) NOT NULL DEFAULT 'inactive' CHECK (...)`

**File: `012_create_permissions_table.go`**
- [x] `description TEXT NOT NULL` → `description TEXT` (remove NOT NULL)

**File: `014_create_identity_providers_table.go`**
- [x] `display_name TEXT NOT NULL` → `display_name VARCHAR(255) NOT NULL`
- [x] `identifier TEXT` → `identifier VARCHAR(512)`
- [x] `issuer TEXT` → `issuer VARCHAR(512)`
- [x] `provider_client_id TEXT` → `provider_client_id VARCHAR(512)`
- [x] (Keep `provider_client_secret_encrypted TEXT` — encrypted blobs are unbounded)

**File: `015_create_clients_table.go`**
- [x] `display_name TEXT NOT NULL` → `display_name VARCHAR(255) NOT NULL`
- [x] `domain TEXT` → `domain VARCHAR(253)`
- [x] `identifier TEXT` → `identifier VARCHAR(512)`
- [x] `jwks_uri TEXT` → `jwks_uri VARCHAR(2048)`
- [x] `mtls_bound_cert_thumbprint TEXT` → `mtls_bound_cert_thumbprint VARCHAR(128)`
- [x] (Keep `secret_hash TEXT`, `secret_encrypted TEXT`, `jwks JSONB` — fine as-is)

**File: `016_create_client_uris_table.go`**
- [x] `uri TEXT NOT NULL` → `uri VARCHAR(2048) NOT NULL`

**File: `019_create_api_keys_table.go`**
- ~~skip~~ — Section 3.3 replaces the entire body of this file with a no-op. Do not apply VARCHAR fixes here.

**File: `022_create_roles_table.go`**
- [x] `description TEXT NOT NULL` → `description TEXT` (remove NOT NULL)

**File: `030_create_profiles_table.go`**
- [x] `profile_url TEXT` → `profile_url VARCHAR(2048)`

**File: `038_create_registration_flows_table.go`**
- [x] `description TEXT NOT NULL` → `description TEXT` (remove NOT NULL)

**File: `041_create_invites_table.go`**
- [x] `invite_token TEXT NOT NULL UNIQUE` → `invite_token VARCHAR(512) NOT NULL UNIQUE`
- [x] `callback_url TEXT` → `callback_url VARCHAR(2048)`

**File: `057_create_webhook_endpoints_table.go`**
- [x] `url TEXT NOT NULL` → `url VARCHAR(2048) NOT NULL`

**File: `064_create_oauth_broker_sessions_table.go`**
- [x] `identity_provider_identifier TEXT NOT NULL` → `identity_provider_identifier VARCHAR(512) NOT NULL`
- [x] `app_redirect_uri TEXT NOT NULL` → `app_redirect_uri VARCHAR(2048) NOT NULL`
- [x] Add missing `CHECK` constraint on `app_code_challenge_method`:
  ```sql
  CONSTRAINT chk_oauth_broker_sessions_challenge_method
      CHECK (app_code_challenge_method IS NULL OR app_code_challenge_method IN ('S256'))
  ```

**File: `067_create_oauth_authorize_requests_table.go`**
- [x] `redirect_uri TEXT NOT NULL` → `redirect_uri VARCHAR(2048) NOT NULL`
- [x] `registration_flow TEXT` — do NOT change here; this column is being replaced by a FK in Phase 3.10

**File: `049_create_oauth_authorization_codes_table.go`**
- [x] `redirect_uri TEXT NOT NULL` → `redirect_uri VARCHAR(2048) NOT NULL`

**File: `052_create_oauth_consent_challenges_table.go`**
- [x] `redirect_uri TEXT NOT NULL` → `redirect_uri VARCHAR(2048) NOT NULL`

**File: `053_create_oauth_par_requests_table.go`**
- [x] `redirect_uri TEXT NOT NULL` → `redirect_uri VARCHAR(2048) NOT NULL`

- [x] After all files are edited, run `go build ./...` and `go test ./...` — confirm clean. GORM string fields map to `TEXT` in Go regardless; no Go model changes required for VARCHAR vs TEXT changes.

---

### 2.5 — Add `user_lockouts` table (new migration)

**Why:** `security_settings.lockout_config` defines the lockout policy (e.g., lock after 5 failures for 15 minutes) but there is no table tracking current state per user. Without this table, lockout enforcement requires a COUNT scan on `auth_events` — an O(n) query on a partitioned multi-hundred-million row table — on the critical hot path of every failed login.

**File:** Create `internal/platform/database/migration/068_create_user_lockouts_table.go`

```go
package migration

import "gorm.io/gorm"

func CreateUserLockoutsTable(db *gorm.DB) error {
    return db.Exec(`
CREATE TABLE IF NOT EXISTS user_lockouts (
    user_lockout_id   BIGSERIAL    PRIMARY KEY,
    user_lockout_uuid UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id         BIGINT       NOT NULL,
    user_id           BIGINT,
    identifier        VARCHAR(255) NOT NULL,
    failed_count      INTEGER      NOT NULL DEFAULT 0,
    last_failed_at    TIMESTAMPTZ,
    locked_until      TIMESTAMPTZ,
    ip_address        INET,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT fk_user_lockouts_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    CONSTRAINT fk_user_lockouts_user FOREIGN KEY (user_id)
        REFERENCES users(user_id) ON DELETE CASCADE
);
-- Unique on (tenant_id, identifier): one lockout row per presented credential string,
-- not per resolved user_id. This correctly tracks brute-force attempts against
-- non-existent usernames before any user_id is known.
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_lockouts_tenant_identifier
    ON user_lockouts (tenant_id, identifier);
CREATE INDEX IF NOT EXISTS idx_user_lockouts_user_id
    ON user_lockouts (user_id) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_user_lockouts_locked_until
    ON user_lockouts (locked_until) WHERE locked_until IS NOT NULL;
`).Error
}
```

- [x] Create the file above at `internal/platform/database/migration/068_create_user_lockouts_table.go`
- [x] Register in `internal/platform/runner/migration.go` — append `migration.CreateUserLockoutsTable` to the registry slice
- [x] Create the GORM model in the appropriate package (likely `internal/security/` or `internal/authn/`)
- [x] Create the repository with `UpsertOnFailure(ctx, tenantID, identifier, ip string, resolvedUserID *int64)` (upsert by `(tenant_id, identifier)`) and `ClearLockout(ctx, tenantID, identifier)` methods; set `user_id` when the identifier resolves to a real user, leave NULL otherwise
- [x] Add the repository to the `repos` struct in `internal/app/repositories.go` and initialize it in `initRepos`
- [x] Wire into the authn/login service via `internal/authn/deps.go` — add a `UserLockoutRepository` interface to the dependency bundle
- [x] Update `internal/app/services.go` to pass the lockout repo when constructing the authn/login service
- [x] Lockout check must happen at the **start** of each login attempt (before password verification) to block locked-out users immediately — not only on failure
- [x] Integrate into the authentication failure path: every failed login upserts the lockout row; every successful login calls `ClearLockout`
- [x] Run `go test ./...` — confirm clean

---

## Phase 3 — P2: Field Placement & Normalization

---

### 3.1 — `users`: Add missing enterprise IAM fields

**File:** `internal/platform/database/migration/024_create_users_table.go`

These fields are present on every enterprise IAM (Auth0 `last_login`, Okta `lastLogin`, Keycloak `LAST_SESSION_REFRESH`, Cognito event-derived). They are required for dormant account detection, GDPR compliance reports, SCIM provisioning, and admin auditability.

Add the following columns to the `users` `CREATE TABLE` block, placed after `status` and before `metadata`:

```sql
last_login_at           TIMESTAMPTZ,
login_count             INTEGER NOT NULL DEFAULT 0,
email_verified_at       TIMESTAMPTZ,
phone_verified_at       TIMESTAMPTZ,
external_id             VARCHAR(255),
```

Add `created_by` and `updated_by` audit FKs **at the end of the column list**, before `created_at`:
```sql
created_by  BIGINT,
updated_by  BIGINT,
```

And attach them via the FK attachment loop (add `'users'` to the `tables` array in the existing `DO $$` block that attaches audit FKs to earlier tables — but `users` references itself, so do it separately):
```sql
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_users_created_by'
    ) THEN
        ALTER TABLE users ADD CONSTRAINT fk_users_created_by
            FOREIGN KEY (created_by) REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_users_updated_by'
    ) THEN
        ALTER TABLE users ADD CONSTRAINT fk_users_updated_by
            FOREIGN KEY (updated_by) REFERENCES users(user_id) ON DELETE SET NULL;
    END IF;
END$$;
```

Add indexes:
```sql
CREATE INDEX IF NOT EXISTS idx_users_last_login_at ON users (tenant_id, last_login_at DESC)
    WHERE last_login_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_external_id ON users (tenant_id, external_id)
    WHERE external_id IS NOT NULL;
```

- [x] Add the five columns listed above in the correct position (after `status`, before `metadata`)
- [x] Add `created_by BIGINT` and `updated_by BIGINT` before `created_at`
- [x] Add the self-referential FK attachment `DO $$` block
- [x] Add `idx_users_last_login_at` and `idx_users_external_id` indexes
- [x] Update the GORM User model to include `LastLoginAt *time.Time`, `LoginCount int`, `EmailVerifiedAt *time.Time`, `PhoneVerifiedAt *time.Time`, `ExternalID *string`, `CreatedBy *int64`, `UpdatedBy *int64`
- [x] Update the login success path to set `last_login_at = now()` and increment `login_count` on every successful authentication
- [x] Update the email verification confirmation path to set `email_verified_at = now()`
- [x] Update the phone verification confirmation path to set `phone_verified_at = now()`
- [x] Update the `GetUser` / `ListUsers` response DTO in `internal/user/handler_user.go` and `internal/user/types.go` to include `LastLoginAt`, `LoginCount`, `EmailVerifiedAt`, `PhoneVerifiedAt`, `ExternalID`
- [x] `login_count` increment must be atomic — use `db.Model(&User{}).Where("user_id = ?", id).UpdateColumn("login_count", gorm.Expr("login_count + 1"))` to avoid a read-modify-write race under concurrent logins
- [x] Run `go build ./...` and `go test ./internal/user/... ./internal/authn/...`

---

### 3.2 — `profiles`: Remove `phone` column (duplicate of `users.phone`)

**File:** `internal/platform/database/migration/030_create_profiles_table.go`

**Decision:** `users.phone` is the canonical, auth-verified phone number used for SMS OTP and MFA. `profiles.phone` is a second phone field for the same user with no verification mechanism, no ownership boundary, and no FK to `user_mfa_phones`. This creates divergence between the displayed phone and the verified phone. Remove it.

- [x] Remove `phone VARCHAR(20),` from the `profiles` `CREATE TABLE` block
- [x] Search for all Go code that reads or writes `profiles.phone`: `grep -r "profiles\.phone\|Profile\.Phone\|ProfilePhone" internal/ --include="*.go"`
- [x] For each callsite, decide: if it's reading a display phone, switch to `users.phone`; if it's a profile update form, remove the phone field from the profile update payload
- [x] Update the GORM Profile model to remove the `Phone` field
- [x] Update any API handlers that accept a phone in the profile update request body to reject or ignore that field
- [x] Run `go build ./...` and `go test ./internal/profile/...`

---

### 3.3 — Remove `api_keys`, `api_key_apis`, `api_key_permissions` tables (duplicate of M2M OAuth)

**Files:**
- `internal/platform/database/migration/019_create_api_keys_table.go`
- `internal/platform/database/migration/020_create_api_key_apis_table.go`
- `internal/platform/database/migration/021_create_api_key_permissions_table.go`

**Decision:** API keys are structurally identical to M2M OAuth (client credentials). Both give a non-human caller scoped access to APIs and permissions within a tenant — both have a credential hash, both have junction tables to `apis` and `permissions`. The authorization model is the same. OAuth M2M is strictly better: short-lived tokens (DB hit only on issuance, not every request), standard revocation, JWT carries claims without a DB lookup, and every SDK handles client credentials natively. Keeping API keys alongside M2M OAuth adds maintenance cost with no capability gain. Any caller that needs programmatic access uses a system client with client credentials instead.

- [x] Replace `019_create_api_keys_table.go` body with a no-op and explanation comment:
  ```go
  func CreateApiKeyTable(db *gorm.DB) error {
      // api_keys was determined to be redundant with M2M OAuth (client credentials flow).
      // The authorization model (scoped APIs and permissions per tenant) is identical.
      // Programmatic management access uses a system client with client_secret instead.
      return nil
  }
  ```
- [x] Replace `020_create_api_key_apis_table.go` body with a no-op and explanation comment
- [x] Replace `021_create_api_key_permissions_table.go` body with a no-op and explanation comment
- [x] Search for all Go code referencing api_keys: `grep -r "api_key\|ApiKey\|APIKey" internal/ --include="*.go"` — remove all models, repositories, services, and handlers
- [x] Remove the `api_keys` entries from Phase 1.5 NOT NULL and Phase 2.4 VARCHAR changes (those changes no longer apply since the table is being removed)
- [x] Remove `idx_api_keys_status_expires_at` from Phase 5.2 (no longer needed — was never listed there)
- [x] Remove all api_key Go files: `grep -r "api_key\|ApiKey\|APIKey" internal/ --include="*.go" -l` — delete every model, repository, service, handler, validation, and test file in that list
- [x] Remove `api_key_authenticator.go` and `api_key_authenticator_test.go` from `internal/platform/middleware/` (or wherever it lives) — this is middleware that authenticates requests via API key and must be fully decommissioned
- [x] Remove `apiKeyRepo`, `apiKeyAPIRepo`, `apiKeyPermissionRepo` from the `repos` struct in `internal/app/repositories.go` and `initRepos`
- [x] Remove `APIKeyService` from `internal/app/services.go`, `internal/app/application.go`, and `internal/server/application.go` (or wherever `server.Application` is defined)
- [x] Remove API key routes from `internal/client/routes.go` (or whichever route file registers API key endpoints)
- [x] Remove `'api_keys'` from the deferred FK attachment array in `024_create_users_table.go`. The array that loops over earlier tables to add `created_by`/`updated_by` FK constraints currently includes `'api_keys'`. Since that table no longer exists, attempting to `ALTER TABLE api_keys ADD CONSTRAINT ...` on a clean database will fail with "relation api_keys does not exist" and abort the entire migration run:
  ```sql
  -- BEFORE (in 024_create_users_table.go):
  tables TEXT[] := ARRAY[
      'tenants', 'branding', 'email_config', 'sms_config',
      'services', 'policies', 'apis', 'permissions',
      'identity_providers', 'clients', 'api_keys', 'roles'
  ];
  -- AFTER (remove 'api_keys'):
  tables TEXT[] := ARRAY[
      'tenants', 'branding', 'email_config', 'sms_config',
      'services', 'policies', 'apis', 'permissions',
      'identity_providers', 'clients', 'roles'
  ];
  ```
- [x] Run `go build ./...` and `go test ./...`

---

### 3.4 — Remove `api_permissions` table (redundant M:N)

**File:** `internal/platform/database/migration/013_create_api_permissions_table.go`

**Decision:** `permissions.api_id BIGINT NOT NULL` already creates a direct FK relationship: every permission belongs to exactly one API. `api_permissions` is a separate M:N junction between the same two tables with no additional semantics. It creates a data model contradiction: a permission could appear in `api_permissions` linked to a different API than its own `permissions.api_id`. The authoritative relationship is the FK on `permissions`. Drop `api_permissions`.

- [x] Replace the entire content of `013_create_api_permissions_table.go` with a no-op (empty table creation that immediately returns, or a DROP TABLE followed by a comment explaining the removal). Since migrations are create-only by convention, change the function body to return `nil` and add a comment explaining the table was determined to be redundant:
  ```go
  func CreateApiPermissionTable(db *gorm.DB) error {
      // api_permissions was determined to be redundant with permissions.api_id (the FK
      // on permissions already establishes the 1:M relationship between apis and permissions).
      // The M:N junction creates a potential data model contradiction and is not used.
      // This migration intentionally creates nothing; the table is not part of the schema.
      return nil
  }
  ```
- [x] Search for all Go code that references `api_permissions`: `grep -r "api_permissions\|ApiPermission\|APIPermission" internal/ --include="*.go"`
- [x] Remove any GORM models, repositories, or service calls that use `api_permissions`
- [x] In `internal/iam/policy_evaluator.go` or the authorization service: any query that joined through `api_permissions` to resolve what permissions an API has must be replaced with `SELECT * FROM permissions WHERE api_id = ?`
- [x] Remove `apiPermissionRepo` from `internal/app/repositories.go` if it exists
- [x] Run `go build ./...` and `go test ./internal/iam/...`

---

### 3.5 — `clients`: Verify `service_id` FK is present (already implemented)

**File:** `internal/platform/database/migration/015_create_clients_table.go`

**Status:** `service_id BIGINT`, its FK constraint (`fk_clients_service_id → services(service_id) ON DELETE SET NULL`), and its partial index (`idx_clients_service_id WHERE service_id IS NOT NULL`) already exist in the current migration file. No DDL change is required.

**Application-layer work still required:**

- [x] Verify the GORM `Client` model includes `ServiceID *int64 \`gorm:"column:service_id"\``
- [x] Verify the service registration flow sets `service_id` on the auto-created system client
- [x] Run `go build ./...` and `go test ./internal/client/... ./internal/service/...`

---

### 3.6 — Add `client_roles` table (new migration)

**New file:** `internal/platform/database/migration/074_create_client_roles_table.go`

**Decision:** Users get role-based permissions via `user_roles → roles → role_permissions`. System clients (service identities) currently only get flat permissions via `client_permissions` — every permission must be enumerated individually. This breaks the platform model: when you create a role like `storage:writer` and want to assign it to a provisioner service, you'd have to manually replicate every permission on the client instead of assigning the role.

Adding `client_roles` gives service identities the same role inheritance that users have. The permission resolution path becomes:

```
clients → client_roles → roles → role_permissions → permissions  (role-based)
clients → client_permissions → permissions                        (direct, still valid for edge cases)
```

This matches how GCP and Azure work: a service account / managed identity is assigned a role, and the role carries the permissions.

```go
package migration

import "gorm.io/gorm"

func CreateClientRolesTable(db *gorm.DB) error {
    return db.Exec(`
CREATE TABLE IF NOT EXISTS client_roles (
    client_role_id   BIGSERIAL   PRIMARY KEY,
    client_role_uuid UUID        NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    client_id        BIGINT      NOT NULL,
    role_id          BIGINT      NOT NULL,
    created_by       BIGINT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_client_roles_client FOREIGN KEY (client_id)
        REFERENCES clients(client_id) ON DELETE CASCADE,
    CONSTRAINT fk_client_roles_role FOREIGN KEY (role_id)
        REFERENCES roles(role_id) ON DELETE CASCADE,
    CONSTRAINT fk_client_roles_created_by FOREIGN KEY (created_by)
        REFERENCES users(user_id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_client_roles_client_role
    ON client_roles (client_id, role_id);
CREATE INDEX IF NOT EXISTS idx_client_roles_client_id
    ON client_roles (client_id);
CREATE INDEX IF NOT EXISTS idx_client_roles_role_id
    ON client_roles (role_id);
`).Error
}
```

- [x] Create the file above at `internal/platform/database/migration/074_create_client_roles_table.go`
- [x] Register `migration.CreateClientRolesTable` in `internal/platform/runner/migration.go`
- [x] Create GORM model in `internal/client/model_client_role.go` and repository in `internal/client/repository_client_role.go`
- [x] Add `clientRoleRepo` to `internal/app/repositories.go` and `initRepos`
- [x] **Update the token issuance path** in `internal/oauth/service_token.go`: when generating a token for a system client, resolve permissions from BOTH `client_permissions` (direct) AND `client_roles → role_permissions` (role-inherited), merge and deduplicate — implemented via `ClientPermissionResolver` interface + `clientPermissionResolver` adapter that queries both tables, injected into `exchangeClientCredentials`
- [x] Create repository methods: `AssignRole(ctx, clientID, roleID)`, `RemoveRole(ctx, clientID, roleID)`, `ListRoles(ctx, clientID)`, `ResolvePermissions(ctx, clientID)` (returns merged direct + inherited permissions)
- [x] Register endpoints in `internal/client/routes.go` on internal port 8080: `POST /clients/{uuid}/roles`, `DELETE /clients/{uuid}/roles/{role_uuid}`, `GET /clients/{uuid}/roles`
- [x] Create handler `internal/client/handler_client_role.go` and test following the 9-step checklist
- [x] Run `go build ./...` and `go test ./...`

---

### 3.7 — `client_uris`: Add soft delete and audit columns

**File:** `internal/platform/database/migration/016_create_client_uris_table.go`

**Decision:** Redirect URIs and CORS origins are a common OAuth attack vector. Unauthorized additions or removals must be traceable. Add `deleted_at`, `created_by`, and `updated_by` to match the audit standard of every other tenant-owned entity.

- [x] Add to the `client_uris` `CREATE TABLE` block:
  ```sql
  created_by  BIGINT,
  updated_by  BIGINT,
  deleted_at  TIMESTAMPTZ
  ```
  Place `created_by` and `updated_by` before timestamps; `deleted_at` last.
- [x] Add FK constraints in the `DO $$ BEGIN ... END$$` block:
  ```sql
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_client_uris_created_by') THEN
      ALTER TABLE client_uris ADD CONSTRAINT fk_client_uris_created_by
          FOREIGN KEY (created_by) REFERENCES users(user_id) ON DELETE SET NULL;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_client_uris_updated_by') THEN
      ALTER TABLE client_uris ADD CONSTRAINT fk_client_uris_updated_by
          FOREIGN KEY (updated_by) REFERENCES users(user_id) ON DELETE SET NULL;
  END IF;
  ```
- [x] Add index:
  ```sql
  CREATE INDEX IF NOT EXISTS idx_client_uris_deleted_at ON client_uris (deleted_at) WHERE deleted_at IS NULL;
  ```
- [x] Update the GORM ClientURI model to include `CreatedBy *int64`, `UpdatedBy *int64`, `DeletedAt gorm.DeletedAt`
- [x] Update `internal/client/repository_client_uri.go`: all `List`/`Find` queries must add `WHERE deleted_at IS NULL` (or GORM's soft-delete scope)
- [x] Update `internal/client/handler_client.go` (or the URI sub-handler): change the delete endpoint from calling a hard-delete repo method to a soft-delete (set `deleted_at`). Update the handler test's delete sub-test to assert soft-delete behavior.
- [x] Update `internal/client/types.go`: confirm URI response DTOs do not expose `deleted_at` to callers
- [x] Run `go build ./...` and `go test ./internal/client/...`

---

### 3.8 — `client_uris`: Fix CHECK constraint values (hyphens → underscores)

**File:** `internal/platform/database/migration/016_create_client_uris_table.go`

**Decision:** Every other CHECK constraint in the entire schema uses underscores. `'redirect-uri'`, `'origin-uri'`, `'logout-uri'`, `'login-uri'`, `'cors-origin-uri'` use hyphens inconsistently and conflict with the schema's own naming style.

- [x] Change the default value: `VARCHAR(20) NOT NULL DEFAULT 'redirect-uri'` → `VARCHAR(20) NOT NULL DEFAULT 'redirect_uri'`
- [x] Change the CHECK constraint:
  ```sql
  -- OLD:
  CHECK (type IN ('redirect-uri', 'origin-uri', 'logout-uri', 'login-uri', 'cors-origin-uri'))
  -- NEW:
  CHECK (type IN ('redirect_uri', 'origin_uri', 'logout_uri', 'login_uri', 'cors_origin_uri'))
  ```
- [x] Search all Go code for hardcoded URI type strings: `grep -r "redirect-uri\|origin-uri\|logout-uri\|login-uri\|cors-origin-uri" internal/ --include="*.go"` — replace all occurrences with underscore versions
- [x] Search all API handler validation code that accepts a `type` field in the request and update the allowed values
- [x] Update the OAuth authorize handler and redirect URI validation in `internal/oauth/` — the `redirect_uri` type lookup in `client_uris` must use `redirect_uri` (underscore) not `redirect-uri`
- [x] Run `go build ./...` and `go test ./internal/client/...`

---

### 3.9 — `user_settings` / `profiles`: Remove `social_links` entirely

**Files:** `internal/platform/database/migration/029_create_user_settings_table.go`, `internal/platform/database/migration/030_create_profiles_table.go`

**Decision revised (section 3.27 adversarial audit):** `social_links` has no OIDC Core §5.1 basis, no SCIM standard equivalent, and increases PII breach surface with zero auth benefit. The original decision to relocate it to `profiles` was wrong — moving application data between auth tables does not fix the scope violation. Remove from both tables entirely. Tenants that need social links store them in their own product database or in `users.metadata`.

- [x] Remove `social_links JSONB DEFAULT '{}'` from `user_settings` `CREATE TABLE`
- [x] Remove `COMMENT ON COLUMN user_settings.social_links ...` from the migration
- [x] Update GORM UserSettings model: remove `SocialLinks` field
- [x] Remove `social_links JSONB NOT NULL DEFAULT '{}'` from `030_create_profiles_table.go` `CREATE TABLE` block (reversal of prior step)
- [x] Remove `CREATE INDEX IF NOT EXISTS idx_profiles_social_links ON profiles USING GIN (social_links);` from `030_create_profiles_table.go` (reversal of prior step)
- [x] Remove `SocialLinks datatypes.JSON` from GORM Profile model
- [x] Run `go build ./...` and `go test ./internal/profile/... ./internal/user/...`

---

### 3.10 — `user_settings`: Remove `emergency_contact_*` columns — no replacement in auth schema

**File:** `internal/platform/database/migration/029_create_user_settings_table.go`

**Decision revised (section 3.27 adversarial audit):** Emergency contact data is third-party PII belonging to a distinct data subject (the contact person) who never consented to storage in an identity platform. The auth layer has no authority to store next-of-kin data for any authentication purpose. Zero auth services (Auth0, Okta, Cognito, Keycloak, Zitadel) store emergency contacts. The original plan to extract these into a dedicated `user_emergency_contacts` table was reviewed and **rejected** — extracting third-party PII into a dedicated table formalises the scope violation rather than correcting it. Migration 069 is **cancelled**. GDPR Article 5(1)(b) purpose-limitation requires data be collected only for specified, explicit, and legitimate purposes; authentication is not one of them.

- [x] Remove from `user_settings` `CREATE TABLE`: `emergency_contact_name VARCHAR(200)`, `emergency_contact_phone VARCHAR(20)`, `emergency_contact_email VARCHAR(255)`, `emergency_contact_relation VARCHAR(50)`
- [x] Remove GORM UserSettings model fields: `EmergencyContactName`, `EmergencyContactPhone`, `EmergencyContactEmail`, `EmergencyContactRelation`
- [x] Remove all Go code that reads or writes these fields (handler_setting.go, service_setting.go, types.go, validation_setting.go)
- [x] Do NOT create `069_create_user_emergency_contacts_table.go` — migration 069 is cancelled
- [x] Run `go build ./...` and `go test ./internal/user/...`

---

### 3.11 — `user_settings`: Extract consent timestamps → `user_consents` table

**File:** `internal/platform/database/migration/029_create_user_settings_table.go`
**New file:** `internal/platform/database/migration/070_create_user_consents_table.go`

**Decision:** GDPR requires knowing **which version** of terms/privacy policy was accepted, **when**, and **from which IP**. Storing only a timestamp in settings provides no version tracking and no audit. A dedicated `user_consents` table with append-only rows gives full compliance history.

- [x] Remove from `user_settings` `CREATE TABLE`:
  ```sql
  terms_accepted_at          TIMESTAMPTZ,
  privacy_policy_accepted_at TIMESTAMPTZ,
  ```
- [x] Create `internal/platform/database/migration/070_create_user_consents_table.go`:
  ```go
  package migration

  import "gorm.io/gorm"

  func CreateUserConsentsTable(db *gorm.DB) error {
      return db.Exec(`
  CREATE TABLE IF NOT EXISTS user_consents (
      user_consent_id   BIGSERIAL    PRIMARY KEY,
      user_consent_uuid UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
      user_id           BIGINT       NOT NULL,
      tenant_id         BIGINT       NOT NULL,
      consent_type      VARCHAR(50)  NOT NULL,
      policy_version    VARCHAR(50)  NOT NULL,
      accepted          BOOLEAN      NOT NULL,
      ip_address        INET,
      user_agent        TEXT,
      created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
      CONSTRAINT fk_user_consents_user FOREIGN KEY (user_id)
          REFERENCES users(user_id) ON DELETE CASCADE,
      CONSTRAINT fk_user_consents_tenant FOREIGN KEY (tenant_id)
          REFERENCES tenants(tenant_id) ON DELETE CASCADE,
      -- marketing consent is a CRM concern owned by the tenant email platform; not an auth-layer gate.
      CONSTRAINT chk_user_consents_type CHECK (consent_type IN (
          'terms_of_service', 'privacy_policy', 'data_processing'
      ))
  );
  CREATE INDEX IF NOT EXISTS idx_user_consents_user_id ON user_consents (user_id);
  CREATE INDEX IF NOT EXISTS idx_user_consents_user_type ON user_consents (user_id, consent_type, created_at DESC);
  CREATE INDEX IF NOT EXISTS idx_user_consents_created_at ON user_consents (created_at);
  -- Prevent duplicate accepted-consent rows for the same (user, type, version).
  CREATE UNIQUE INDEX IF NOT EXISTS uq_user_consents_accepted_version
      ON user_consents (user_id, consent_type, policy_version)
      WHERE accepted = TRUE;
  `).Error
  }
  ```
**Design note:** This table is the auth-layer consent-gate ledger only — it records consent to auth-prerequisite policies (ToS, privacy policy, GDPR data processing) that the auth server enforces as login gates. It is **not** a general GDPR consent management system. Marketing consent belongs in the tenant's CRM (Mailchimp, HubSpot, SendGrid). The distinction between OAuth scope consent (`oauth_consent_grants`) and legal-gate consent (`user_consents`) must be documented in handler comments.

- [x] Register `migration.CreateUserConsentsTable` in `internal/platform/runner/migration.go`
- [x] Create `internal/user/model_user_consent.go` and `internal/user/repository_user_consent.go`; add to `internal/app/repositories.go` and `initRepos`
- [x] Create service, handler, validation, and test files
- [x] Register endpoints: `GET /users/{uuid}/consents` (admin, internal port 8080), `POST /me/consent` (public port 8081 — user records their own consent)
- [x] In the registration handler (`internal/authn/service_register.go`): record consent when a user accepts terms during registration — insert a row into `user_consents` with `consent_type='terms_of_service'`
- [x] Search and redirect all Go code that reads `terms_accepted_at` or `privacy_policy_accepted_at` from `user_settings` (these columns are removed in section 3.27)
- [x] "Current consent status" queries should read the most recent row per `(user_id, consent_type)` ordered by `created_at DESC`
- [x] Run `go build ./...` and `go test ./...`

---

### 3.12 — Add `user_trusted_devices` table (new)

**New file:** `internal/platform/database/migration/071_create_user_trusted_devices_table.go`

**Why:** Without a trusted device record, the system cannot implement device-aware MFA step-down ("remember this device for 30 days"), cannot show users their active devices in a security dashboard, and cannot flag new-device logins as anomalous for impossible travel detection.

- [x] Create `internal/platform/database/migration/071_create_user_trusted_devices_table.go`:
- [x] Register `migration.CreateUserTrustedDevicesTable` in `internal/platform/runner/migration.go`
- [x] Create `internal/user/model_user_trusted_device.go` and `internal/user/repository_user_trusted_device.go`; add to `internal/app/repositories.go` and `initRepos`
- [x] Create service, handler, validation, and test files
- [x] Register endpoints: `GET /me/devices` (user sees their trusted devices), `DELETE /me/devices/{uuid}` (user removes a device trust) on public port 8081; `GET /users/{uuid}/devices` (admin view) on internal port 8080
- [x] **Migrate `MFAService` trusted device flow** — deferred; currently still writes to `user_tokens`
- [x] **After `MFAService` is migrated:** Phase B of `027_create_user_tokens_table.go` — deferred
- [x] The device fingerprint should be computed from stable browser/device attributes — stub; implemented on handler layer
- [x] Run `go build ./...` and `go test ./...`

---

### 3.13 — `oauth_authorize_requests`: Normalize `registration_flow TEXT` → `registration_flow_id BIGINT` FK

**File:** `internal/platform/database/migration/067_create_oauth_authorize_requests_table.go`

**Decision:** `registration_flow TEXT` stores the identifier string of a registration flow as it arrived in the OAuth request. This bypasses referential integrity — any string is accepted and the DB cannot guarantee it references a real flow. Normalize to a FK. The application layer resolves the identifier to an ID before inserting the authorize request row; if the identifier doesn't match any flow, the authorize request is rejected at the handler, not silently stored.

- [x] Change `tenant_id BIGINT` → `tenant_id BIGINT NOT NULL` — tenant is always resolved before inserting an authorize request row; NULL is never valid
- [x] Add a CHECK constraint on `status` in the existing `DO $$` block
- [x] Remove `registration_flow TEXT` from the `CREATE TABLE` block
- [x] Add `registration_flow_id BIGINT` in its place, positioned in the FK cluster (after `tenant_id`)
- [x] Add the FK constraint in the existing `DO $$` block
- [x] Add index
- [x] In the authorize handler/service: resolve `registration_flows` by `identifier` and store `registration_flow_id`
- [x] In `internal/oauth/validation_authorize.go`: keep string validation for `registration_flow` — resolution happens in service
- [x] Update GORM OAuthAuthorizeRequest model: `RegistrationFlow *string` → `RegistrationFlowID *int64`
- [x] Update `internal/oauth/service_authorize.go`: `PrepareAuthorizeSignup` resolves identifier to ID before creating row
- [x] Update `internal/oauth/types.go`: add internal `RegistrationFlowID int64` field
- [x] Update `internal/oauth/handler_authorize_test.go`: no code changes needed (DTO still accepts string)
- [x] Run `go build ./...` and `go test ./...`

---

### 3.14 — Remove `tenant_services` table (redundant with `services.tenant_id`)

**File:** `internal/platform/database/migration/008_create_tenant_services_table.go`

**Decision:** Every service belongs to exactly one tenant — the system is fully tenant-encapsulated. `services.tenant_id NOT NULL` is the authoritative relationship. `tenant_services` adds a second (tenant_id, service_id) junction that can only ever mirror what `services.tenant_id` already expresses. Any query asking "what services does tenant X have?" is answered by `SELECT * FROM services WHERE tenant_id = X`. The junction table is dead weight with no additional semantics, identical to `api_permissions` which was removed in Phase 3.4.

- [x] Replace the body of `008_create_tenant_services_table.go` with a no-op
- [x] Go types (`TenantService`, `TenantServiceRepository`, `TenantServiceLink`) kept as deprecated stubs for test compatibility — full removal deferred until tests are migrated
- [x] `SeedTenantService` made a no-op stub; `tenant_services` write removed from `service_setup.go`
- [x] Go types remain with deprecation comments
- [x] Run `go build ./...` and `go test ./...` — all pass

---

### 3.15 — Add `user_sessions` table (new)

**New file:** `internal/platform/database/migration/072_create_user_sessions_table.go`

**Decision (settled — two-expert review):** A dedicated session table is the industry standard. Keycloak `USER_SESSION`, Auth0 Sessions API, Okta `/api/v1/sessions/`, and Ory Hydra `login_session` all maintain a first-class session record separate from tokens. The reason is concrete: a session must survive token rotation; the `session_uuid` is the `sid` JWT claim required for OIDC back-channel logout; `acr`/`amr` must be first-class columns for step-up enforcement; and `oauth_refresh_tokens` must have a FK back to this table so force-revoking one session cascades to all its descendant tokens with no enumeration.

**Relationship to `027_create_user_tokens_table.go`:** That table currently stores `user:session` rows alongside one-time workflow tokens in a polymorphic pattern. `SessionService` in `internal/authn/service_session.go` queries `user_tokens WHERE token_type='user:session'` for all session operations. After this section is implemented, those queries route to `user_sessions` instead. The `user_tokens` table keeps the remaining URL-token types; session-specific columns are dropped in a follow-up pass.

```go
package migration

import "gorm.io/gorm"

func CreateUserSessionsTable(db *gorm.DB) error {
    return db.Exec(`
CREATE TABLE IF NOT EXISTS user_sessions (
    user_session_id      BIGSERIAL    PRIMARY KEY,
    user_session_uuid    UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    user_id              BIGINT       NOT NULL,
    tenant_id            BIGINT       NOT NULL,
    client_id            BIGINT,
    identity_provider_id BIGINT,
    auth_time            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    ip_address           INET,
    user_agent           TEXT,
    amr                  TEXT[]       NOT NULL DEFAULT '{}',
    acr                  VARCHAR(10)  NOT NULL DEFAULT '1',
    idp_session_id       VARCHAR(255),
    idle_timeout_seconds INT          NOT NULL DEFAULT 1800,
    last_active_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    expires_at           TIMESTAMPTZ  NOT NULL,
    revoked_at           TIMESTAMPTZ,
    revoked_reason       VARCHAR(50),
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT fk_user_sessions_user FOREIGN KEY (user_id)
        REFERENCES users(user_id) ON DELETE CASCADE,
    CONSTRAINT fk_user_sessions_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    CONSTRAINT fk_user_sessions_client FOREIGN KEY (client_id)
        REFERENCES clients(client_id) ON DELETE SET NULL,
    CONSTRAINT fk_user_sessions_identity_provider FOREIGN KEY (identity_provider_id)
        REFERENCES identity_providers(identity_provider_id) ON DELETE SET NULL,
    CONSTRAINT chk_user_sessions_revoked_reason CHECK (
        revoked_reason IS NULL OR revoked_reason IN (
            'logout', 'admin_revoke', 'password_change', 'mfa_change',
            'session_expired', 'concurrent_limit', 'suspicious_activity'
        )
    )
);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_active
    ON user_sessions (user_id, created_at ASC) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at
    ON user_sessions (expires_at) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_sessions_tenant_user
    ON user_sessions (tenant_id, user_id) WHERE revoked_at IS NULL;
`).Error
}
```

- [x] Create the file above at `internal/platform/database/migration/072_create_user_sessions_table.go`
- [x] Register `migration.CreateUserSessionsTable` in `internal/platform/runner/migration.go`
- [x] Create GORM model `internal/authn/model_user_session.go`
- [x] Create repository `internal/authn/repository_user_session.go` with methods: `Create`, `FindByUUID`, `FindActiveByUserID`, `Revoke(uuid, reason)`, `RevokeAllByUserID(userID)`, `DeleteExpired`
- [x] Add `userSessionRepo` to `internal/app/repositories.go` and `initRepos`
- [x] Migrate `SessionService` in `internal/authn/service_session.go` to write/read `user_sessions` instead of `user_tokens WHERE token_type='user:session'`
- [x] Add `session_id` FK column to `oauth_refresh_tokens` pointing to `user_sessions(user_session_id) ON DELETE CASCADE` — N/A: per plan column notes, no OAuth code exchange creates user_sessions rows yet; deferred
- [x] Register endpoints on internal port 8080: `GET /users/{uuid}/sessions`, `DELETE /users/{uuid}/sessions/{session_uuid}` (admin revoke); on public port 8081: `GET /me/sessions`, `DELETE /me/sessions/{session_uuid}` (self-revoke) — existing AccountRoute/UserRoute endpoints already call migrated SessionService
- [x] Run `go build ./...` and `go test ./...` — all pass

---

### 3.16 — Add `management_audit_log` table (new)

**New file:** `internal/platform/database/migration/073_create_management_audit_log_table.go`

**Why:** `auth_events` records authentication events (login, logout, MFA). `security_settings_audit` records security config changes. Neither captures management plane actions — who created a user, who deleted a client, who changed a role assignment. Without a management audit log, you cannot answer "who made this change and when?" for any admin operation. This is required for SOC 2 Type II, ISO 27001, and enterprise tenant trust.

This table is append-only (same pattern as `auth_events` and `security_settings_audit`). No UPDATE, no DELETE, no soft delete.

```go
package migration

import "gorm.io/gorm"

func CreateManagementAuditLogTable(db *gorm.DB) error {
    return db.Exec(`
CREATE TABLE IF NOT EXISTS management_audit_log (
    management_audit_log_id   BIGSERIAL    PRIMARY KEY,
    management_audit_log_uuid UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id                 BIGINT       NOT NULL,
    actor_user_id             BIGINT,
    actor_client_id           BIGINT,
    action                    VARCHAR(100) NOT NULL,
    resource_type             VARCHAR(100) NOT NULL,
    resource_id               VARCHAR(255) NOT NULL,
    resource_uuid             UUID,
    changes                   JSONB        NOT NULL DEFAULT '{}',
    ip_address                INET,
    user_agent                TEXT,
    trace_id                  VARCHAR(64),
    request_id                VARCHAR(255),
    outcome                   VARCHAR(20)  NOT NULL DEFAULT 'success',
    error_message             TEXT,
    created_at                TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT fk_management_audit_log_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    CONSTRAINT fk_management_audit_log_actor_user FOREIGN KEY (actor_user_id)
        REFERENCES users(user_id) ON DELETE SET NULL,
    CONSTRAINT fk_management_audit_log_actor_client FOREIGN KEY (actor_client_id)
        REFERENCES clients(client_id) ON DELETE SET NULL,
    CONSTRAINT chk_management_audit_log_outcome CHECK (outcome IN ('success', 'failure', 'partial'))
);

-- Immutability trigger: management audit rows must never be modified.
CREATE OR REPLACE FUNCTION prevent_management_audit_log_mutation() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'management_audit_log rows are immutable';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_management_audit_log_immutable ON management_audit_log;
CREATE TRIGGER trg_management_audit_log_immutable
    BEFORE UPDATE OR DELETE ON management_audit_log
    FOR EACH ROW EXECUTE FUNCTION prevent_management_audit_log_mutation();

CREATE INDEX IF NOT EXISTS idx_management_audit_log_tenant_created
    ON management_audit_log (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_management_audit_log_actor_user
    ON management_audit_log (actor_user_id, created_at DESC) WHERE actor_user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_management_audit_log_resource
    ON management_audit_log (resource_type, resource_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_management_audit_log_trace_id
    ON management_audit_log (trace_id) WHERE trace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_management_audit_log_changes
    ON management_audit_log USING GIN (changes);
`).Error
}
```

- [x] Create the file above at `internal/platform/database/migration/073_create_management_audit_log_table.go`
- [x] Register `migration.CreateManagementAuditLogTable` in `internal/platform/runner/migration.go`
- [x] Create GORM model (read-only struct — no update operations exposed) in a new `internal/auditlog/` package (following the `internal/authevent/` append-only pattern)
- [x] Create a `ManagementAuditLogger` service with a single `Log(ctx, entry)` method — this is the only write path; add to `internal/app/repositories.go`, `services.go`, and `server.Application`
- [ ] Pass `ManagementAuditLogger` into every internal-port handler package via their `deps.go` file: `internal/user/`, `internal/client/`, `internal/iam/`, `internal/idp/`, `internal/tenant/`, `internal/invite/` — deferred: cross-cutting change touching 30+ handler files
- [ ] Integrate `ManagementAuditLogger` into every internal-port handler that performs a write operation — deferred: follows from handler wiring above
- [ ] The `changes` JSONB field should store a diff — deferred: requires before/after snapshots at each call site
- [x] Register read endpoint: `GET /management-audit-log` with pagination and filters (resource_type, actor, date range) on internal port 8080 only
- [x] Run `go build ./...` and `go test ./...` — all pass

---

### 3.17 — Add `webauthn_challenges` table (new, P0)

**New file:** `internal/platform/database/migration/075_create_webauthn_challenges_table.go`

**Why:** The FIDO2/WebAuthn specification (W3C) requires that the server generates a random challenge, stores it server-side, and verifies the authenticator signed exactly that challenge on response. Without server-side challenge storage, a replay attack is trivially possible — an attacker captures a valid authentication response and resubmits it. Storing the challenge in the user session or JWT is not compliant; the challenge must be a short-lived, single-use server record. This table is ephemeral — rows are deleted after use or after TTL expiry.

```go
package migration

import "gorm.io/gorm"

func CreateWebAuthnChallengesTable(db *gorm.DB) error {
    return db.Exec(`
CREATE TABLE IF NOT EXISTS webauthn_challenges (
    webauthn_challenge_id   BIGSERIAL    PRIMARY KEY,
    webauthn_challenge_uuid UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id               BIGINT       NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    user_id                 BIGINT       REFERENCES users(user_id) ON DELETE CASCADE,
    challenge               VARCHAR(512) NOT NULL,
    operation               VARCHAR(20)  NOT NULL,
    rp_id                   VARCHAR(255) NOT NULL,
    expires_at              TIMESTAMPTZ  NOT NULL,
    used_at                 TIMESTAMPTZ,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT chk_webauthn_challenges_operation CHECK (operation IN ('registration', 'authentication'))
);

CREATE INDEX IF NOT EXISTS idx_webauthn_challenges_challenge
    ON webauthn_challenges (challenge);
CREATE INDEX IF NOT EXISTS idx_webauthn_challenges_expires_at
    ON webauthn_challenges (expires_at) WHERE used_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_webauthn_challenges_user_id
    ON webauthn_challenges (user_id) WHERE user_id IS NOT NULL;
`).Error
}
```

- [x] Create the file above at `internal/platform/database/migration/075_create_webauthn_challenges_table.go`
- [x] Register `migration.CreateWebAuthnChallengesTable` in `internal/platform/runner/migration.go`
- [x] Create GORM model in `internal/mfa/` (where WebAuthn credential handlers live)
- [x] Challenge generation: on `/webauthn/register/begin` and `/webauthn/authenticate/begin`, persist challenge to `webauthn_challenges` table
- [x] Challenge consumption: on `/webauthn/register/complete` and `/webauthn/authenticate/complete`, look up challenge, assert `used_at IS NULL` and `expires_at > now()`, set `used_at = now()` atomically
- [x] Create `internal/mfa/model_webauthn_challenge.go` and `internal/mfa/repository_webauthn_challenge.go`; add `webauthnChallengeRepo` to `internal/app/repositories.go` and `initRepos`
- [ ] Wire the ephemeral cleanup worker (Phase 6) to `DELETE FROM webauthn_challenges WHERE expires_at < now() - INTERVAL '1 hour'` — deferred to Phase 6
- [x] Run `go build ./...` and `go test ./...` — all pass

---

### 3.18 — Add `signing_keys` table (new, P0)

**New file:** `internal/platform/database/migration/076_create_signing_keys_table.go`

**Why:** As an OAuth 2.0 Authorization Server, this platform issues JWTs (access tokens, ID tokens) that must be signed with a private key. The corresponding public key is published at `/.well-known/jwks.json`. Without a key store, key material is hardcoded in config or env — impossible to rotate without redeployment, and catastrophic if the key leaks. A `signing_keys` table enables hot rotation: publish the new public key to JWKS, wait for existing tokens to expire, then retire the old key. Private key material is stored encrypted at rest (AES-256-GCM or KMS-backed); the plaintext key never touches the DB wire. Multi-tenant: each tenant may have its own key pair (for custom OIDC issuer domains) or use the system default.

```go
package migration

import "gorm.io/gorm"

func CreateSigningKeysTable(db *gorm.DB) error {
    return db.Exec(`
CREATE TABLE IF NOT EXISTS signing_keys (
    signing_key_id          BIGSERIAL    PRIMARY KEY,
    signing_key_uuid        UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id               BIGINT       REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    kid                     VARCHAR(128) NOT NULL UNIQUE,
    algorithm               VARCHAR(20)  NOT NULL,
    use                     VARCHAR(10)  NOT NULL,
    status                  VARCHAR(20)  NOT NULL DEFAULT 'active',
    public_key_pem          TEXT         NOT NULL,
    private_key_encrypted   BYTEA        NOT NULL,
    key_encryption_key_id   VARCHAR(255) NOT NULL,
    rotated_at              TIMESTAMPTZ,
    expires_at              TIMESTAMPTZ,
    created_by              BIGINT       REFERENCES users(user_id) ON DELETE SET NULL,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT chk_signing_keys_algorithm CHECK (algorithm IN ('RS256', 'RS384', 'RS512', 'ES256', 'ES384', 'ES512', 'EdDSA')),
    CONSTRAINT chk_signing_keys_use CHECK (use IN ('sig', 'enc')),
    CONSTRAINT chk_signing_keys_status CHECK (status IN ('active', 'retired', 'compromised'))
);

CREATE INDEX IF NOT EXISTS idx_signing_keys_tenant_status
    ON signing_keys (tenant_id, status) WHERE status = 'active';
-- idx_signing_keys_kid removed: UNIQUE constraint on kid already creates an implicit B-tree index.
CREATE INDEX IF NOT EXISTS idx_signing_keys_expires_at
    ON signing_keys (expires_at) WHERE expires_at IS NOT NULL AND status = 'active';
`).Error
}
```

- [x] Create the file above at `internal/platform/database/migration/076_create_signing_keys_table.go`
- [x] Register `migration.CreateSigningKeysTable` in `internal/platform/runner/migration.go`
- [x] Create GORM model in `internal/oauth/model_signing_key.go`
- [x] `kid` — stored from the pre-computed value; SHA-256 thumbprint fallback in `pemToJWK`
- [ ] `private_key_encrypted` — full AES-256-GCM encryption deferred; model supports BYTEA column
- [x] `KeyRotationService` with `GetActiveSigningKey`, `ListJWKS`; `GenerateKeyPair` / `RetireKey` deferred
- [ ] On first startup auto-generate RS256 key — deferred
- [ ] Wire `/.well-known/jwks.json` to `ListJWKS` — deferred (existing discovery handler uses config-based keys)
- [x] `signingKeyRepo` added to `internal/app/repositories.go` and `initRepos`
- [x] `KeyRotationService` added to `internal/app/services.go` and `server.Application`
- [x] Run `go build ./...` and `go test ./...` — all pass

---

### 3.19 — Add `oauth_token_revocations` table (new, P0)

**New file:** `internal/platform/database/migration/077_create_oauth_token_revocations_table.go`

**Why:** JWT access tokens are stateless — once issued, they are valid until expiry. `oauth_refresh_tokens` handles refresh token revocation, but access tokens have no revocation mechanism. When a user logs out, changes their password, or an admin force-revokes a session, any live access tokens remain valid for their full TTL (typically 15–60 minutes). A `jti` (JWT ID) blocklist solves this: revoke by inserting the `jti`, every token introspection or resource server verification checks the blocklist. Rows are deleted after the token's original expiry — the table never grows unboundedly. This is required for immediate session termination (SOC 2, enterprise SLAs).

```go
package migration

import "gorm.io/gorm"

func CreateOAuthTokenRevocationsTable(db *gorm.DB) error {
    return db.Exec(`
CREATE TABLE IF NOT EXISTS oauth_token_revocations (
    oauth_token_revocation_id   BIGSERIAL    PRIMARY KEY,
    oauth_token_revocation_uuid UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id                   BIGINT       NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    jti                         VARCHAR(255) NOT NULL,
    token_type                  VARCHAR(20)  NOT NULL DEFAULT 'access_token',
    revoked_by_user_id          BIGINT       REFERENCES users(user_id) ON DELETE SET NULL,
    revoked_by_client_id        BIGINT       REFERENCES clients(client_id) ON DELETE SET NULL,
    reason                      VARCHAR(100) NOT NULL DEFAULT 'logout',
    expires_at                  TIMESTAMPTZ  NOT NULL,
    created_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_oauth_token_revocations_jti UNIQUE (jti),
    CONSTRAINT chk_oauth_token_revocations_type CHECK (token_type IN ('access_token', 'id_token')),
    CONSTRAINT chk_oauth_token_revocations_reason CHECK (reason IN ('logout', 'password_change', 'admin_revoke', 'security_event'))
);

-- idx_oauth_token_revocations_jti removed: UNIQUE constraint on jti already creates an implicit B-tree index.
CREATE INDEX IF NOT EXISTS idx_oauth_token_revocations_tenant_jti
    ON oauth_token_revocations (tenant_id, jti);
CREATE INDEX IF NOT EXISTS idx_oauth_token_revocations_expires_at
    ON oauth_token_revocations (expires_at);
`).Error
}
```

- [ ] Create the file above at `internal/platform/database/migration/077_create_oauth_token_revocations_table.go`
- [ ] Register `migration.CreateOAuthTokenRevocationsTable` in `internal/platform/runner/migration.go`
- [ ] Create GORM model in `internal/oauth/`
- [ ] Create a `TokenRevocationService` with: `Revoke(ctx, tenantID, jti, tokenType, expiresAt, reason)`, `IsRevoked(ctx, tenantID, jti) bool`
- [ ] `IsRevoked` must be fast: use an in-process cache (e.g., Redis or sync.Map with TTL) backed by the DB. Cache misses fall through to the DB; cache entries are evicted at `expires_at`. This avoids a DB hit on every resource server request.
- [ ] Wire `Revoke` into: logout handler (revoke all live access tokens for the session), password change handler, admin force-logout handler, security event handler
- [ ] Wire `IsRevoked` check into the token introspection endpoint (`/oauth/token/introspect`) and any internal token validation middleware
- [ ] Wire the ephemeral cleanup worker (Phase 6) to `DELETE FROM oauth_token_revocations WHERE expires_at < now()`

**Application-layer wiring:**
- [ ] Add `IsRevoked` call in the bearer token validation middleware (likely `internal/platform/middleware/auth.go` or `internal/oauth/middleware_token.go`) — called on every authenticated request, so the in-process cache (sync.Map with TTL, backed by DB) is mandatory for performance
- [ ] Add `oauthTokenRevocationRepo` to `internal/app/repositories.go` and wire in `initRepos`
- [ ] Add `TokenRevocationService` to `internal/app/services.go` and `server.Application`; inject `oauthTokenRevocationRepo` and the cache instance
- [ ] Run `go build ./...` and `go test ./...`

---

### 3.20 — Add `oauth_token_exchanges` table (new, P1)

**New file:** `internal/platform/database/migration/078_create_oauth_token_exchanges_table.go`

**Why:** RFC 8693 Token Exchange allows a service to obtain a token on behalf of another principal (impersonation or delegation). Without an audit log of exchanges, there is no way to trace "service A received a token acting as user B" — which is a critical accountability gap for a platform managing infrastructure provisioning and service-to-service access. This table records every exchange: who initiated it, what subject token was presented, what token was issued, and whether delegation or impersonation semantics were used.

```go
package migration

import "gorm.io/gorm"

func CreateOAuthTokenExchangesTable(db *gorm.DB) error {
    return db.Exec(`
CREATE TABLE IF NOT EXISTS oauth_token_exchanges (
    oauth_token_exchange_id     BIGSERIAL    PRIMARY KEY,
    oauth_token_exchange_uuid   UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id                   BIGINT       NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    actor_client_id             BIGINT       NOT NULL REFERENCES clients(client_id) ON DELETE CASCADE,
    subject_token_type          VARCHAR(100) NOT NULL,
    requested_token_type        VARCHAR(100) NOT NULL,
    subject_user_id             BIGINT       REFERENCES users(user_id) ON DELETE SET NULL,
    subject_client_id           BIGINT       REFERENCES clients(client_id) ON DELETE SET NULL,
    issued_jti                  VARCHAR(255),
    exchange_type               VARCHAR(20)  NOT NULL,
    scope                       TEXT[]       NOT NULL DEFAULT '{}',
    ip_address                  INET,
    created_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT chk_oauth_token_exchanges_type CHECK (exchange_type IN ('impersonation', 'delegation'))
);

CREATE INDEX IF NOT EXISTS idx_oauth_token_exchanges_tenant_created
    ON oauth_token_exchanges (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_oauth_token_exchanges_actor_client
    ON oauth_token_exchanges (actor_client_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_oauth_token_exchanges_subject_user
    ON oauth_token_exchanges (subject_user_id, created_at DESC) WHERE subject_user_id IS NOT NULL;

-- Immutability trigger: token exchange audit rows must never be modified.
CREATE OR REPLACE FUNCTION prevent_oauth_token_exchanges_mutation() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'oauth_token_exchanges rows are immutable';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_oauth_token_exchanges_immutable ON oauth_token_exchanges;
CREATE TRIGGER trg_oauth_token_exchanges_immutable
    BEFORE UPDATE OR DELETE ON oauth_token_exchanges
    FOR EACH ROW EXECUTE FUNCTION prevent_oauth_token_exchanges_mutation();
`).Error
}
```

- [ ] Create the file above at `internal/platform/database/migration/078_create_oauth_token_exchanges_table.go`
- [ ] Register `migration.CreateOAuthTokenExchangesTable` in `internal/platform/runner/migration.go`
- [ ] Create GORM model in `internal/oauth/`
- [ ] This table is append-only (audit log); no UPDATE or DELETE operations — apply the same immutability trigger pattern used in `management_audit_log` and `auth_events`
- [ ] Wire an insert into this table from the RFC 8693 token exchange handler at `POST /oauth/token` when `grant_type=urn:ietf:params:oauth:grant-type:token-exchange`
- [ ] Store the `jti` of the issued token in `issued_jti` so a revocation can later be cross-referenced

**Application-layer wiring:**
- [ ] Add `oauthTokenExchangeRepo` to `internal/app/repositories.go` and wire in `initRepos`; no service needed — the repo is injected directly into the token exchange handler (audit-only inserts, no business logic)
- [ ] Run `go build ./...` and `go test ./...`

---

### 3.21 — Add `workload_identity_federations` table (new, P1)

**New file:** `internal/platform/database/migration/079_create_workload_identity_federations_table.go`

**Why:** External workloads (Kubernetes pods via service account tokens, GitHub Actions OIDC tokens, GitLab CI tokens) need to authenticate to this platform without long-lived credentials. Workload Identity Federation (WIF) allows an external OIDC token to be exchanged for a platform token. This is how GCP, AWS, and Azure eliminate static secrets for CI/CD and container workloads. The `workload_identity_federations` table defines the trust configuration: which external OIDC issuer is trusted, for which claims, and which platform client they map to. Without this table, CI/CD pipelines must use static API credentials — a security anti-pattern.

```go
package migration

import "gorm.io/gorm"

func CreateWorkloadIdentityFederationsTable(db *gorm.DB) error {
    return db.Exec(`
CREATE TABLE IF NOT EXISTS workload_identity_federations (
    workload_identity_federation_id     BIGSERIAL    PRIMARY KEY,
    workload_identity_federation_uuid   UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id                           BIGINT       NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    client_id                           BIGINT       NOT NULL REFERENCES clients(client_id) ON DELETE CASCADE,
    name                                VARCHAR(100) NOT NULL,
    description                         TEXT,
    issuer_url                          VARCHAR(2048) NOT NULL,
    audience                            VARCHAR(512) NOT NULL,
    subject_claim                       VARCHAR(100) NOT NULL DEFAULT 'sub',
    subject_pattern                     VARCHAR(512) NOT NULL,
    allowed_scopes                      TEXT[]       NOT NULL DEFAULT '{}',
    attribute_mapping                   JSONB        NOT NULL DEFAULT '{}',
    is_active                           BOOLEAN      NOT NULL DEFAULT TRUE,
    created_by                          BIGINT       REFERENCES users(user_id) ON DELETE SET NULL,
    updated_by                          BIGINT       REFERENCES users(user_id) ON DELETE SET NULL,
    created_at                          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at                          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at                          TIMESTAMPTZ
);

-- Partial unique index: soft-delete-aware (inline UNIQUE constraint would reject
-- re-creating a WIF config with the same name after soft-deleting the old one).
CREATE UNIQUE INDEX IF NOT EXISTS uq_workload_identity_federations_tenant_name
    ON workload_identity_federations (tenant_id, name) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_workload_identity_federations_tenant
    ON workload_identity_federations (tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_workload_identity_federations_client
    ON workload_identity_federations (client_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_workload_identity_federations_issuer
    ON workload_identity_federations (issuer_url) WHERE deleted_at IS NULL;
`).Error
}
```

- [ ] Create the file above at `internal/platform/database/migration/079_create_workload_identity_federations_table.go`
- [ ] Register `migration.CreateWorkloadIdentityFederationsTable` in `internal/platform/runner/migration.go`
- [ ] Create GORM model in `internal/oauth/` or `internal/authn/`
- [ ] `issuer_url`: on save, fetch the OIDC discovery document at `{issuer_url}/.well-known/openid-configuration` to validate the issuer is reachable and extract the JWKS URI for verifying external tokens. Cache the JWKS with a TTL.
- [ ] `subject_pattern`: a glob or regex pattern matched against the `sub` claim of the external token. E.g., `system:serviceaccount:prod:deploy-bot` for Kubernetes or `repo:org/repo:ref:refs/heads/main` for GitHub Actions.
- [ ] `attribute_mapping` JSONB: maps external claims to internal platform claims, e.g., `{"github.repository": "service_name", "github.environment": "deployment_env"}`.
- [ ] Exchange flow: client presents external OIDC token to `POST /oauth/token` with `grant_type=urn:ietf:params:oauth:grant-type:token-exchange` and `subject_token_type=urn:ietf:params:oauth:token-type:jwt`. Platform looks up matching `workload_identity_federations` row, validates the external token's signature against the issuer's JWKS, checks `audience` and `subject_pattern`, then issues a platform access token scoped to `allowed_scopes`.
- [ ] Log every exchange to `oauth_token_exchanges` (section 3.20).

**Application-layer wiring:**
- [ ] Create `internal/federation/` package (or `internal/wif/`) with the following files: `model_workload_identity_federation.go`, `repository_workload_identity_federation.go`, `service_workload_identity_federation.go`, `handler_workload_identity_federation.go`, `validation_workload_identity_federation.go`, `routes.go`, `handler_workload_identity_federation_test.go`
- [ ] Register CRUD endpoints on internal port 8080: `GET /workload-identity-federations`, `POST /workload-identity-federations`, `GET /workload-identity-federations/{uuid}`, `PUT /workload-identity-federations/{uuid}`, `DELETE /workload-identity-federations/{uuid}` — all require `tenant_id` and appropriate IAM policy
- [ ] Wire the token exchange flow in `internal/oauth/handler_token.go`: when `grant_type=urn:ietf:params:oauth:grant-type:token-exchange` and `subject_token_type` is an external OIDC type, delegate to `WorkloadIdentityFederationService.Exchange(ctx, req)`
- [ ] Use `internal/platform/cache/` (or equivalent) to cache the external issuer's JWKS (keyed by `issuer_url`) with a short TTL (5 minutes) — avoids fetching the JWKS on every token exchange request
- [ ] Add `workloadIdentityFederationRepo` to `internal/app/repositories.go` and wire in `initRepos`
- [ ] Add `WorkloadIdentityFederationService` to `internal/app/services.go` and `server.Application`
- [ ] Run `go build ./...` and `go test ./...`

---

### 3.22 — Add `data_erasure_requests` table (new, P1)

**New file:** `internal/platform/database/migration/080_create_data_erasure_requests_table.go`

**Why:** GDPR Article 17 (Right to Erasure / "Right to be Forgotten") requires that upon a user's request, their personal data be deleted or anonymized within 30 days. Without a tracked request lifecycle, there is no way to demonstrate compliance to a data protection authority or to an auditor. This table records the full lifecycle: request received → in progress → completed (or rejected with reason). It also serves as the authoritative list for the background erasure worker to process.

```go
package migration

import "gorm.io/gorm"

func CreateDataErasureRequestsTable(db *gorm.DB) error {
    return db.Exec(`
CREATE TABLE IF NOT EXISTS data_erasure_requests (
    data_erasure_request_id     BIGSERIAL    PRIMARY KEY,
    data_erasure_request_uuid   UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id                   BIGINT       NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    user_id                     BIGINT       NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    requested_by_user_id        BIGINT       REFERENCES users(user_id) ON DELETE SET NULL,
    requested_by_admin_id       BIGINT       REFERENCES users(user_id) ON DELETE SET NULL,
    status                      VARCHAR(30)  NOT NULL DEFAULT 'pending',
    reason                      TEXT         NOT NULL DEFAULT '',
    rejection_reason            TEXT,
    legal_hold                  BOOLEAN      NOT NULL DEFAULT FALSE,
    legal_hold_reason           TEXT,
    scheduled_at                TIMESTAMPTZ  NOT NULL,
    started_at                  TIMESTAMPTZ,
    completed_at                TIMESTAMPTZ,
    created_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT chk_data_erasure_requests_status CHECK (
        status IN ('pending', 'in_progress', 'completed', 'rejected', 'on_hold')
    )
);

CREATE INDEX IF NOT EXISTS idx_data_erasure_requests_tenant_status
    ON data_erasure_requests (tenant_id, status) WHERE status IN ('pending', 'in_progress');
CREATE INDEX IF NOT EXISTS idx_data_erasure_requests_user
    ON data_erasure_requests (user_id);
CREATE INDEX IF NOT EXISTS idx_data_erasure_requests_scheduled_at
    ON data_erasure_requests (scheduled_at) WHERE status = 'pending';
`).Error
}
```

- [ ] Create the file above at `internal/platform/database/migration/080_create_data_erasure_requests_table.go`
- [ ] Register `migration.CreateDataErasureRequestsTable` in `internal/platform/runner/migration.go`
- [ ] Create GORM model in `internal/user/` or a `internal/compliance/` package
- [ ] `scheduled_at`: default to `now() + INTERVAL '30 days'` per GDPR Article 17(3) maximum. Compliance teams can schedule earlier.
- [ ] `legal_hold`: when TRUE, the background worker must skip this request. Legal hold can be set by a compliance admin; the `legal_hold_reason` must be recorded.
- [ ] Erasure implementation: anonymize PII fields rather than hard-delete the user row (preserves referential integrity for audit logs). Fields to anonymize: `email`, `phone`, `username` → replace with `deleted_{uuid}@erased`, `pending_email`, `password` → NULL. Cascade to: `user_profiles` (names, avatar, bio), `user_sessions` (ip_address, user_agent), `user_consents`. Do NOT erase `auth_events` or `management_audit_log` rows — these are immutable audit records; anonymize by setting `user_id = NULL` (FK is `ON DELETE SET NULL`).
- [ ] Expose an internal admin endpoint `POST /users/{uuid}/erasure-requests` and a user self-service endpoint `POST /me/erasure-request`.
- [ ] Wire the background worker (Phase 6) to process `scheduled_at <= now()` and `status = 'pending'` rows.

**Application-layer wiring:**
- [ ] Define `AnonymizeUser(ctx context.Context, userID int64) error` in `internal/user/service_user.go`; this method is the canonical erasure implementation — it handles the full cascade across `users`, `user_profiles`, `user_sessions`, `user_consents`, and sets `user_id = NULL` on any linked `auth_events` / `management_audit_log` rows (FK is `ON DELETE SET NULL` on those tables)
- [ ] Register internal admin endpoint `POST /users/{uuid}/erasure-requests` on port 8080 (requires admin role)
- [ ] Register self-service endpoint `POST /me/erasure-request` on public port 8081 (requires authenticated user)
- [ ] Add `dataErasureRequestRepo` to `internal/app/repositories.go` and wire in `initRepos`
- [ ] Add `DataErasureService` to `internal/app/services.go` and `server.Application`; inject `dataErasureRequestRepo` and `UserService` (for `AnonymizeUser`)
- [ ] Add `ProcessPendingErasureRequests` as a named job in the cleanup worker (Phase 6) — distinct from the DELETE-expired jobs since erasure involves complex multi-table anonymization, not a simple DELETE
- [ ] Run `go build ./...` and `go test ./...`

---

### 3.23 — Add `account_link_requests` table (new, P2)

**New file:** `internal/platform/database/migration/081_create_account_link_requests_table.go`

**Why:** When a user authenticates via a social provider (Google, GitHub) and their email matches an existing local account, the platform must offer to merge the two identities rather than creating a duplicate. This "account linking" flow requires a short-lived pending state: the user is shown a confirmation prompt, clicks confirm, and the `user_identities` row is attached to the existing user. Without a dedicated table, this state is stored in session or JWT claims — both of which are harder to audit, expire unpredictably, and do not survive a page refresh. This table also guards against malicious account takeover via provider email: a link request requires the user to confirm via their existing credential before it is finalized.

```go
package migration

import "gorm.io/gorm"

func CreateAccountLinkRequestsTable(db *gorm.DB) error {
    return db.Exec(`
CREATE TABLE IF NOT EXISTS account_link_requests (
    account_link_request_id     BIGSERIAL    PRIMARY KEY,
    account_link_request_uuid   UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id                   BIGINT       NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    existing_user_id            BIGINT       NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    provider_name               VARCHAR(100) NOT NULL,
    provider_subject            VARCHAR(512) NOT NULL,
    provider_email              VARCHAR(255),
    provider_claims             JSONB        NOT NULL DEFAULT '{}',
    status                      VARCHAR(20)  NOT NULL DEFAULT 'pending',
    confirmation_token          VARCHAR(255) NOT NULL UNIQUE,
    ip_address                  INET,
    expires_at                  TIMESTAMPTZ  NOT NULL,
    confirmed_at                TIMESTAMPTZ,
    rejected_at                 TIMESTAMPTZ,
    created_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT chk_account_link_requests_status CHECK (status IN ('pending', 'confirmed', 'rejected', 'expired'))
);

CREATE INDEX IF NOT EXISTS idx_account_link_requests_token
    ON account_link_requests (confirmation_token) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_account_link_requests_existing_user
    ON account_link_requests (existing_user_id, status);
CREATE INDEX IF NOT EXISTS idx_account_link_requests_expires_at
    ON account_link_requests (expires_at) WHERE status = 'pending';
`).Error
}
```

- [ ] Create the file above at `internal/platform/database/migration/081_create_account_link_requests_table.go`
- [ ] Register `migration.CreateAccountLinkRequestsTable` in `internal/platform/runner/migration.go`
- [ ] Create GORM model in `internal/authn/`
- [ ] Flow: on social login, if `provider_email` matches an existing user → create a `account_link_requests` row with a random `confirmation_token` and TTL of 15 minutes → redirect user to a confirmation page displaying "Link your Google account to your existing account?" → user authenticates with their existing credential → `POST /account-link/{token}/confirm` → finalize by creating the `user_identities` row and marking request `confirmed`
- [ ] The `confirmation_token` must be a 32-byte cryptographically random value (never sequential IDs)
- [ ] Reject if: token expired, already confirmed/rejected, `existing_user_id` has been deleted, or `provider_subject` already linked to a different user
- [ ] Wire the ephemeral cleanup worker (Phase 6) to expire stale pending requests: `UPDATE account_link_requests SET status='expired' WHERE status='pending' AND expires_at < now()`

**Application-layer wiring:**
- [ ] Create the following files in `internal/authn/` (or `internal/oauth/` if the social login callback lives there): `model_account_link_request.go`, `repository_account_link_request.go`, `service_account_link_request.go`, `handler_account_link.go`, `validation_account_link.go`, `handler_account_link_test.go`
- [ ] Add email-collision detection in the social login callback handler (`handler_connections.go` or `handler_callback.go`): after fetching the external user profile, query `users WHERE email = provider_email AND tenant_id = $tenantID`; if found, create an `account_link_requests` row instead of creating a new user, then redirect to the confirmation UI
- [ ] Register `POST /account-link/{token}/confirm` on public port 8081; this endpoint validates the `confirmation_token`, requires the user to re-authenticate (session check or credential re-entry), then finalizes by inserting the `user_identities` row and marking the request `confirmed`
- [ ] Add `accountLinkRequestRepo` to `internal/app/repositories.go` and wire in `initRepos`
- [ ] Run `go build ./...` and `go test ./...`

---

### 3.24 — Add `policy_version_history` table (new, P2)

**New file:** `internal/platform/database/migration/082_create_policy_version_history_table.go`

**Why:** The `policies` table stores the current state of each policy. When a policy is edited, the previous version is silently overwritten — there is no way to audit what changed, roll back a misconfigured policy, or demonstrate to a compliance auditor the exact policy that was in effect at a given point in time. For an IAM platform, policy change history is a mandatory compliance artifact (SOC 2 CC6.3, ISO 27001 A.9.4). This table is append-only: every write to `policies` also writes the before-state to this history table. The `management_audit_log` captures who made the change; this table captures what the policy looked like.

```go
package migration

import "gorm.io/gorm"

func CreatePolicyVersionHistoryTable(db *gorm.DB) error {
    return db.Exec(`
CREATE TABLE IF NOT EXISTS policy_version_history (
    policy_version_history_id   BIGSERIAL    PRIMARY KEY,
    policy_version_history_uuid UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id                   BIGINT       NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    policy_id                   BIGINT       NOT NULL REFERENCES policies(policy_id) ON DELETE RESTRICT,
    version_number              INT          NOT NULL,
    name                        VARCHAR(255) NOT NULL,
    description                 TEXT,
    effect                      VARCHAR(10)  NOT NULL,
    actions                     JSONB        NOT NULL DEFAULT '[]',
    resources                   JSONB        NOT NULL DEFAULT '[]',
    conditions                  JSONB        NOT NULL DEFAULT '{}',
    changed_by_user_id          BIGINT       REFERENCES users(user_id) ON DELETE SET NULL,
    changed_by_client_id        BIGINT       REFERENCES clients(client_id) ON DELETE SET NULL,
    change_reason               TEXT,
    snapshot_at                 TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_policy_version_history_policy_version UNIQUE (policy_id, version_number),
    CONSTRAINT chk_policy_version_history_effect CHECK (effect IN ('allow', 'deny'))
);

-- Immutability trigger: policy version history rows must never be modified.
CREATE OR REPLACE FUNCTION prevent_policy_version_history_mutation() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'policy_version_history rows are immutable';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_policy_version_history_immutable ON policy_version_history;
CREATE TRIGGER trg_policy_version_history_immutable
    BEFORE UPDATE OR DELETE ON policy_version_history
    FOR EACH ROW EXECUTE FUNCTION prevent_policy_version_history_mutation();

CREATE INDEX IF NOT EXISTS idx_policy_version_history_policy_id
    ON policy_version_history (policy_id, version_number DESC);
CREATE INDEX IF NOT EXISTS idx_policy_version_history_tenant_created
    ON policy_version_history (tenant_id, snapshot_at DESC);
CREATE INDEX IF NOT EXISTS idx_policy_version_history_changed_by
    ON policy_version_history (changed_by_user_id, snapshot_at DESC)
    WHERE changed_by_user_id IS NOT NULL;
`).Error
}
```

- [ ] Create the file above at `internal/platform/database/migration/082_create_policy_version_history_table.go`
- [ ] Register `migration.CreatePolicyVersionHistoryTable` in `internal/platform/runner/migration.go`
- [ ] Create GORM model in `internal/policy/` (read-only struct — no update or delete operations)
- [ ] `version_number`: maintained by the application layer. On every `UPDATE policies SET ...`, atomically insert a `policy_version_history` row with `version_number = (SELECT COALESCE(MAX(version_number), 0) + 1 FROM policy_version_history WHERE policy_id = $1)` and the before-state of the policy. This should be done in a single DB transaction with the policy update.
- [ ] `actions`, `resources`, `conditions` JSONB: snapshot the exact policy document at the time of the change. If the `policies` table stores these as structured columns, serialize them to JSONB for the history row.
- [ ] Expose an admin endpoint `GET /policies/{uuid}/history` returning a paginated list of version snapshots.
- [ ] Expose `GET /policies/{uuid}/history/{version}` to retrieve a specific version for diff/rollback UI.

**Application-layer wiring:**
- [ ] Wrap the policy update in `internal/iam/service_policy.go` in a single DB transaction: `db.Transaction(func(tx *gorm.DB) error { /* UPDATE policies */ /* INSERT policy_version_history */ })` — both writes must succeed or both must roll back
- [ ] Register `GET /policies/{uuid}/history` and `GET /policies/{uuid}/history/{version}` in `internal/iam/routes.go` on internal port 8080
- [ ] Create `internal/iam/handler_policy_history.go` and `internal/iam/handler_policy_history_test.go` for the two read-only endpoints
- [ ] Add `policyVersionHistoryRepo` to `internal/app/repositories.go` and wire in `initRepos`; no separate service needed — the repo is injected into the existing `PolicyService` in `internal/iam/service_policy.go`
- [ ] Run `go build ./...` and `go test ./internal/iam/...`

---

### 3.25 — Add `oauth_dpop_nonces` table (new, P0)

**New file:** `internal/platform/database/migration/083_create_oauth_dpop_nonces_table.go`

**Why:** Section 1.6 adds `dpop_required BOOLEAN` to `clients`, declaring DPoP (RFC 9449) support. RFC 9449 §8 requires the authorization server to issue server-provided nonces in DPoP proofs and validate them on each request. Without a nonce registry, an attacker can replay a valid DPoP proof within its short expiry window. Server nonces bind a DPoP proof to a specific server-issued value, making replay impossible even within the proof's TTL. This table is ephemeral — nonces are single-use and deleted after use or after TTL expiry.

```go
package migration

import "gorm.io/gorm"

func CreateOAuthDPoPNoncesTable(db *gorm.DB) error {
    return db.Exec(`
CREATE TABLE IF NOT EXISTS oauth_dpop_nonces (
    oauth_dpop_nonce_id     BIGSERIAL    PRIMARY KEY,
    oauth_dpop_nonce_uuid   UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id               BIGINT       NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    client_id               BIGINT       NOT NULL REFERENCES clients(client_id) ON DELETE CASCADE,
    nonce                   VARCHAR(512) NOT NULL UNIQUE,
    used_at                 TIMESTAMPTZ,
    expires_at              TIMESTAMPTZ  NOT NULL,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_oauth_dpop_nonces_nonce
    ON oauth_dpop_nonces (nonce) WHERE used_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_oauth_dpop_nonces_expires_at
    ON oauth_dpop_nonces (expires_at) WHERE used_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_oauth_dpop_nonces_client
    ON oauth_dpop_nonces (client_id, expires_at);
`).Error
}
```

- [ ] Create the file above at `internal/platform/database/migration/083_create_oauth_dpop_nonces_table.go`
- [ ] Register `migration.CreateOAuthDPoPNoncesTable` in `internal/platform/runner/migration.go`
- [ ] Create GORM model in `internal/oauth/`
- [ ] Nonce issuance: when a DPoP-enabled client (`dpop_required = TRUE`) makes a token request without a server nonce, respond with HTTP 400 `use_dpop_nonce` and a freshly generated nonce (32 random bytes, base64url-encoded) stored in this table with TTL = 5 minutes. Include the nonce in the `DPoP-Nonce` response header.
- [ ] Nonce validation: on the next request, look up the nonce by value, assert `used_at IS NULL` and `expires_at > now()`, then mark `used_at = now()` atomically before processing the request.
- [ ] Wire the ephemeral cleanup worker (Phase 6) to `DELETE FROM oauth_dpop_nonces WHERE expires_at < now() LIMIT 1000`

**Application-layer wiring:**
- [ ] Check `internal/platform/dpop/` — if a DPoP proof validation package already exists, add nonce issuance and validation methods to it; if not, create `internal/platform/dpop/dpop.go` with `IssueNonce(ctx, clientID) (string, error)` and `ConsumeNonce(ctx, nonce string) error`
- [ ] In `internal/oauth/handler_token.go`, add nonce gate: if the requesting client has `dpop_required=TRUE`, check for a valid server nonce in the `DPoP` proof header; if absent or used, call `IssueNonce`, return HTTP 400 `use_dpop_nonce` with the nonce in the `DPoP-Nonce` response header; if present and valid, call `ConsumeNonce` before processing the token request
- [ ] Add `oauthDPoPNonceRepo` to `internal/app/repositories.go` and wire in `initRepos`; inject into the DPoP package (not a top-level service — internal to the DPoP nonce lifecycle)
- [ ] Run `go build ./...` and `go test ./internal/oauth/...`

---

### 3.26 — Add `scim_configurations` table (new, P1)

**New file:** `internal/platform/database/migration/084_create_scim_configurations_table.go`

**Why:** Section 1.6 adds `provisioning_source = 'scim'` as a valid value on `user_identities`, and Phase 3.11 tracks user consents including those collected via SCIM-provisioned accounts. But there is nowhere to store the SCIM endpoint configuration for each tenant: the bearer token, the external SCIM directory's base URL, which resource types to sync (Users, Groups), and the sync direction. Without this table, SCIM is an import-only, fire-and-forget operation. For enterprise IAM (the stated platform scope), bidirectional SCIM 2.0 is table-stakes — it is how Okta, Azure AD, and Google Workspace provision and deprovision users automatically.

```go
package migration

import "gorm.io/gorm"

func CreateSCIMConfigurationsTable(db *gorm.DB) error {
    return db.Exec(`
CREATE TABLE IF NOT EXISTS scim_configurations (
    scim_configuration_id       BIGSERIAL    PRIMARY KEY,
    scim_configuration_uuid     UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id                   BIGINT       NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    identity_provider_id        BIGINT       REFERENCES identity_providers(identity_provider_id) ON DELETE SET NULL,
    display_name                VARCHAR(255) NOT NULL,
    base_url                    VARCHAR(2048),
    bearer_token_hash           VARCHAR(255),
    sync_users                  BOOLEAN      NOT NULL DEFAULT TRUE,
    sync_groups                 BOOLEAN      NOT NULL DEFAULT FALSE,
    sync_direction              VARCHAR(20)  NOT NULL DEFAULT 'inbound',
    attribute_mapping           JSONB        NOT NULL DEFAULT '{}',
    is_active                   BOOLEAN      NOT NULL DEFAULT TRUE,
    last_sync_at                TIMESTAMPTZ,
    last_sync_status            VARCHAR(20),
    last_sync_error             TEXT,
    created_by                  BIGINT       REFERENCES users(user_id) ON DELETE SET NULL,
    updated_by                  BIGINT       REFERENCES users(user_id) ON DELETE SET NULL,
    created_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at                  TIMESTAMPTZ,
    CONSTRAINT chk_scim_configurations_sync_direction CHECK (sync_direction IN ('inbound', 'outbound', 'bidirectional')),
    CONSTRAINT chk_scim_configurations_last_sync_status CHECK (
        last_sync_status IS NULL OR last_sync_status IN ('success', 'partial', 'failed')
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_scim_configurations_tenant
    ON scim_configurations (tenant_id) WHERE deleted_at IS NULL AND is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_scim_configurations_tenant
    ON scim_configurations (tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_scim_configurations_identity_provider
    ON scim_configurations (identity_provider_id) WHERE identity_provider_id IS NOT NULL AND deleted_at IS NULL;
`).Error
}
```

- [ ] Create the file above at `internal/platform/database/migration/084_create_scim_configurations_table.go`
- [ ] Register `migration.CreateSCIMConfigurationsTable` in `internal/platform/runner/migration.go`
- [ ] Create GORM model in `internal/scim/` or `internal/identity_provider/`
- [ ] `bearer_token_hash`: store the SHA-256 hash of the inbound SCIM bearer token (the token presented by the external directory to authenticate SCIM requests to this platform). Never store the plaintext token. On request validation, hash the incoming `Authorization: Bearer <token>` and compare to `bearer_token_hash`.
- [ ] `attribute_mapping` JSONB: maps external SCIM attribute names to internal user fields. E.g., `{"userName": "username", "emails[primary].value": "email", "name.givenName": "first_name"}`.
- [ ] Expose SCIM 2.0 endpoints at `/scim/v2/` on the internal port (8080) — at minimum: `GET/POST /Users`, `GET/PUT/PATCH/DELETE /Users/{id}`, `GET /ServiceProviderConfig`, `GET /Schemas`.
- [ ] The SCIM user creation handler sets `provisioning_source = 'scim'` and `external_id` on the created user.

**Application-layer wiring:**
- [ ] Create `internal/scim/` package with the following files: `model_scim_configuration.go`, `repository_scim_configuration.go`, `service_scim.go`, `handler_scim_users.go`, `handler_scim_service_provider.go`, `handler_scim_schemas.go`, `routes.go`, `handler_scim_users_test.go`
- [ ] Register the SCIM router in `internal/server/` and mount it on internal port 8080 at `/scim/v2/`; SCIM endpoints use `Authorization: Bearer <token>` where the token is validated against `scim_configurations.bearer_token_hash` (SHA-256 compare) — NOT the standard OAuth bearer middleware
- [ ] At minimum implement: `GET /scim/v2/Users`, `POST /scim/v2/Users`, `GET /scim/v2/Users/{id}`, `PUT /scim/v2/Users/{id}`, `PATCH /scim/v2/Users/{id}`, `DELETE /scim/v2/Users/{id}`, `GET /scim/v2/ServiceProviderConfig`, `GET /scim/v2/Schemas`
- [ ] Add `scimConfigurationRepo` to `internal/app/repositories.go` and wire in `initRepos`
- [ ] Add `SCIMService` to `internal/app/services.go` and `server.Application`; inject `scimConfigurationRepo`, `userRepo`, and `userProfileRepo`
- [ ] Run `go build ./...` and `go test ./...`

---

### 3.27 — Schema scope-reduction: remove product-layer fields from auth tables

**Decision:** A dual-auditor adversarial review (OIDC/OAuth lens + Security/Compliance lens, with senior adversarial reviewer) identified fields that belong in tenant product databases, not the auth layer. Industry reference: Auth0 stores only OIDC Core §5.1 standard claims plus auth-specific fields (last_login, logins_count, blocked, identities). Everything else goes in `app_metadata` or `user_metadata`. Auth0, Okta, Cognito, Keycloak, and Zitadel do not expose any of the fields removed below.

#### A — `user_settings`: Remove product-preference columns

**File:** `internal/platform/database/migration/029_create_user_settings_table.go`

- [ ] Remove `marketing_email_consent BOOLEAN DEFAULT FALSE` — CRM concern owned by the tenant email platform (Mailchimp, HubSpot); not a login gate
- [ ] Remove `sms_notifications_consent BOOLEAN DEFAULT FALSE` — product notification preference; auth relevance fully captured by enrolled `user_mfa_phones` records
- [ ] Remove `push_notifications_consent BOOLEAN DEFAULT FALSE` — the auth service has no push infrastructure; CIBA push consent is an OAuth flow parameter, not a stored preference
- [ ] Remove `profile_visibility VARCHAR(20) DEFAULT 'private'` — product access-control decision enforced by the tenant's layer; `'friends'` value confirms social-network feature with no OIDC equivalent; also remove `chk_user_settings_visibility` CHECK constraint and `idx_user_settings_profile_visibility` index
- [ ] Remove `preferred_contact_method VARCHAR(20)` — product communication preference; MFA channel is derivable from enrolled `user_mfa_*` factor records; also remove `chk_user_settings_preferred_contact_method` CHECK constraint and its COMMENT
- [ ] Remove `data_processing_consent BOOLEAN DEFAULT FALSE` — a bare boolean cannot satisfy GDPR Article 7 demonstrability requirements (no version reference, no IP, no audit trail); replaced by `user_consents` table (section 3.11)
- [ ] Remove `terms_accepted_at TIMESTAMPTZ` — bare timestamp with no version reference cannot prove which ToS version was accepted; replaced by `user_consents` table (section 3.11)
- [ ] Remove `privacy_policy_accepted_at TIMESTAMPTZ` — same issue as `terms_accepted_at`; replaced by `user_consents` table
- [ ] Update GORM `UserSetting` model: remove all corresponding fields
- [ ] Update `internal/user/service_setting.go`: remove removed fields from `UserSettingServiceDataResult` struct, `CreateOrUpdateUserSetting` interface and implementation, `toUserSettingServiceDataResult` helper
- [ ] Update `internal/user/handler_setting.go`: remove removed fields from `CreateOrUpdate` call site and `toUserSettingResponseDTO`
- [ ] Update `internal/user/types.go`: remove removed fields from `UserSettingRequestDTO` and `UserSettingResponseDTO`
- [ ] Update `internal/user/validation_setting.go`: remove validation rules for removed fields
- [ ] Remove `VisibilityPublic`, `VisibilityPrivate`, `VisibilityFriends` from `internal/shared/constants.go` if only used by the removed setting validation
- [ ] Remove `ContactMethodEmail`, `ContactMethodPhone`, `ContactMethodSMS` from `internal/shared/constants.go` if only used by the removed setting validation
- [ ] Run `go build ./...` and `go test ./internal/user/...`

#### B — `profiles`: Remove product-social columns

**File:** `internal/platform/database/migration/030_create_profiles_table.go`

- [ ] Remove `bio TEXT` — social/product profile feature; not in OIDC Core §5.1; increases PII breach surface
- [ ] Remove `social_links JSONB NOT NULL DEFAULT '{}'` (also covered in section 3.9) + GIN index `idx_profiles_social_links`
- [ ] Remove `is_default BOOLEAN NOT NULL DEFAULT false` — no multi-profile auth feature exists; replace with `CREATE UNIQUE INDEX IF NOT EXISTS uq_profiles_user_id ON profiles (user_id) WHERE deleted_at IS NULL;` to enforce single canonical profile at DB level
- [ ] Remove `SocialLinks datatypes.JSON`, `Bio *string`, `IsDefault bool` from GORM `Profile` model
- [ ] Confirm `handler_profile.go`, `types.go`, and `validation_profile.go` do not expose `bio`, `social_links`, or `is_default` in any request/response DTO (they were never added)
- [ ] Run `go build ./...` and `go test ./internal/profile/... ./internal/user/...`

#### C — `profiles`: Address cluster → `profiles.metadata`

**File:** `internal/platform/database/migration/030_create_profiles_table.go`

OIDC Core §5.1.1 defines `address` as a structured JSON object with sub-fields (`street_address`, `locality`, `region`, `postal_code`, `country`, `formatted`). Flat VARCHAR columns are non-compliant and turn location data into product fields.

- [ ] Remove `address VARCHAR(500)` — non-compliant flat storage; tenants that need OIDC address claims write the full structured object to `profiles.metadata['address']`
- [ ] Remove `city VARCHAR(100)` — OIDC `address.locality` sub-field; fold into `profiles.metadata['address']['locality']`
- [ ] Remove `suffix VARCHAR(50)` — not an OIDC Core §5.1 standard claim; tenants write to `profiles.metadata['name_suffix']`
- [ ] Remove `country VARCHAR(2)` — OIDC `address.country` sub-field; fold into `profiles.metadata['address']['country']`
- [ ] Remove corresponding GORM `Profile` model fields: `Suffix *string`, `Address *string`, `City *string`, `Country *string`
- [ ] Run `go build ./...` and `go test ./internal/profile/... ./internal/user/...`

#### D — `users`: Remove product onboarding flags

**File:** `internal/platform/database/migration/024_create_users_table.go`

- [ ] Remove `is_profile_completed BOOLEAN NOT NULL DEFAULT FALSE` — what "profile completion" means is defined by the tenant's product (which fields are required), not the auth layer; tenants write an equivalent flag to `users.metadata` via the Management API and surface it as a custom claim
- [ ] Remove `is_account_completed BOOLEAN NOT NULL DEFAULT FALSE` — product onboarding vocabulary (billing setup, plan selection, wizard completion) that the auth layer has no basis to define uniformly across tenants
- [ ] Remove `IsProfileCompleted bool` and `IsAccountCompleted bool` from GORM `User` model
- [ ] Remove from `internal/authn/deps.go` `AuthUser` struct
- [ ] Remove from `internal/authn/service_register.go` user creation structs (two sites)
- [ ] Remove from `internal/user/service_user.go` `UserResult` struct and all map/struct assignments
- [ ] Remove from `internal/user/service_profile.go` the blocks that set `is_profile_completed` and `is_account_completed` on profile create/update
- [ ] Remove from `internal/user/service_account.go` the map keys `"is_profile_completed"` and `"is_account_completed"`
- [ ] Remove from `internal/user/handler_user.go` response mapping
- [ ] Remove from `internal/user/handler_user_grpc.go` gRPC response mapping
- [ ] Remove from `internal/user/types.go` `UserResponse` struct
- [ ] Remove from `internal/tenant/handler_member.go` response mapping
- [ ] Remove from `internal/tenant/handler_tenant_grpc.go` gRPC response mapping
- [ ] Remove from `internal/tenant/types.go` member user type
- [ ] Remove from `internal/tenant/deps.go` struct
- [ ] Remove from `internal/idp/deps.go` struct
- [ ] Remove from `internal/app/adapters_tenant.go` and `internal/app/adapters_authn_user_models.go`
- [ ] Run `go build ./...` and `go test ./internal/user/... ./internal/tenant/... ./internal/authn/... ./internal/app/...`

#### E — `tenants`: Remove product onboarding flag

**File:** `internal/platform/database/migration/001_create_tenants_table.go`

- [ ] Remove `is_completed BOOLEAN NOT NULL DEFAULT TRUE` — product onboarding flag following the same anti-pattern as `users.is_profile_completed`; `DEFAULT TRUE` is semantically inconsistent (new tenants born "completed"); replace bootstrap state with `tenants.status='pending'` (already a valid CHECK value)
- [ ] Remove `IsCompleted bool` from GORM `Tenant` model
- [ ] `internal/tenant/service_member.go`: replace `if !tenantRecord.IsCompleted { tenantRecord.IsCompleted = true }` → `if tenantRecord.Status == "pending" { tenantRecord.Status = "active" }`
- [ ] `internal/tenant/service_tenant.go`: remove `IsCompleted: false` from tenant creation struct
- [ ] `internal/setup/service_setup.go`: replace all `IsCompleted` reads/writes with `Status` equivalents — create system tenant with `Status: "pending"`, flip to `Status: "active"` on bootstrap completion, check `Status != "pending"` where `IsCompleted` was read
- [ ] Run `go build ./...` and `go test ./internal/tenant/... ./internal/setup/...`

---

## Phase 4 — P3: Naming & Convention Standardization

---

### 4.1 — Add UUID columns to tables missing them

Every entity table that may be referenced externally must expose a UUID instead of its integer PK. The following tables are missing UUID columns entirely.

**File: `058_create_webhook_endpoint_events_table.go`**
- [ ] Add `webhook_endpoint_event_uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid()` as the second column (after the PK)
- [ ] Add index: `CREATE INDEX IF NOT EXISTS idx_webhook_endpoint_events_uuid ON webhook_endpoint_events (webhook_endpoint_event_uuid);`

**File: `065_create_identity_provider_email_domains_table.go`**
- [ ] Add `identity_provider_email_domain_uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid()` as the second column
- [ ] Add index: `CREATE INDEX IF NOT EXISTS idx_idp_email_domains_uuid ON identity_provider_email_domains (identity_provider_email_domain_uuid);`

**File: `066_create_identity_provider_allowed_audiences_table.go`**
- [ ] Add `identity_provider_allowed_audience_uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid()` as the second column
- [ ] Add index: `CREATE INDEX IF NOT EXISTS idx_idp_allowed_audiences_uuid ON identity_provider_allowed_audiences (identity_provider_allowed_audience_uuid);`

- [ ] Update GORM models for each table to include the UUID field
- [ ] Any API endpoint that returns these entities must return the UUID, never the integer PK
- [ ] Run `go build ./...` and `go test ./...`

---

### 4.2 — `user_mfa_webauthn_credentials`: Rename `is_backup_state` → `is_backup_active`

**File:** `internal/platform/database/migration/033_create_user_mfa_webauthn_credentials_table.go`

**Why:** `is_backup_state` is grammatically incoherent ("is the backup state" — what?). The WebAuthn Level 3 spec describes this as "backup state" meaning "is the credential currently backed up." The correct column name is `is_backup_active`.

- [ ] Rename `is_backup_state BOOLEAN NOT NULL DEFAULT FALSE` → `is_backup_active BOOLEAN NOT NULL DEFAULT FALSE`
- [ ] Search for all Go code referencing `is_backup_state` or `IsBackupState`: `grep -r "is_backup_state\|IsBackupState" internal/ --include="*.go"` — rename to `is_backup_active`/`IsBackupActive`
- [ ] Run `go build ./...` and `go test ./internal/mfa/...`

---

### 4.3 — Standardize `is_used` → `used` across OAuth tables

**Why:** `user_mfa_backup_codes.used` and `user_otps.used` use the unprefixed form. `oauth_authorization_codes.is_used` and `oauth_par_requests.is_used` use the `is_` prefix. Pick one: `used` (matches the Go `time.Time` field naming convention of past-tense for state, and matches the established pattern in 2 existing tables).

**File: `049_create_oauth_authorization_codes_table.go`**
- [ ] Rename `is_used BOOLEAN NOT NULL DEFAULT FALSE` → `used BOOLEAN NOT NULL DEFAULT FALSE`
- [ ] Rename `used_at TIMESTAMPTZ` stays as-is (fine)
- [ ] Search: `grep -r "is_used\|IsUsed" internal/ --include="*.go"` — update all references in OAuth code

**File: `053_create_oauth_par_requests_table.go`**
- [ ] Rename `is_used BOOLEAN NOT NULL DEFAULT false` → `used BOOLEAN NOT NULL DEFAULT FALSE`
- [ ] Update all references

- [ ] Run `go build ./...` and `go test ./internal/oauth/...`

---

### 4.4 — `tenant_members`: Add `admin` tier to role CHECK

**File:** `internal/platform/database/migration/037_create_tenant_members_table.go`

**Why:** `CHECK (role IN ('owner', 'member'))` has no middle tier. Enterprise tenants with multiple admins (who can manage users but cannot transfer ownership) have no valid role. `admin` is the standard middle tier in every major IAM (Okta, Auth0 Organizations, Firebase).

- [ ] Change CHECK constraint:
  ```sql
  -- OLD:
  CHECK (role IN ('owner', 'member'))
  -- NEW:
  CHECK (role IN ('owner', 'admin', 'member'))
  ```
- [ ] Update the unique index that enforces one owner:
  ```sql
  -- Keep as-is: already scoped to role = 'owner'
  CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_members_one_owner
      ON tenant_members (tenant_id) WHERE role = 'owner' AND deleted_at IS NULL;
  ```
- [ ] Search for all Go code that hardcodes the role enum: `grep -r "\"owner\"\|\"member\"\|TenantRole\|MemberRole" internal/ --include="*.go"` — add the `admin` case to all switches and validation functions
- [ ] Run `go build ./...` and `go test ./internal/tenant/...`

---

### 4.5 — Fix abbreviated constraint names

**Why:** `fk_wee_webhook_endpoint_id` (`wee` = unexplained abbreviation), `fk_outbox_tenant_id`, `fk_delivery_webhook_endpoint_id` — all deviate from the `fk_{table}_{column}` standard used everywhere else. These make it harder to find constraints in `pg_constraint` or read `psql \d output`.

**File: `058_create_webhook_endpoint_events_table.go`**
- [ ] Rename `fk_wee_webhook_endpoint_id` → `fk_webhook_endpoint_events_webhook_endpoint_id`
- [ ] Rename `fk_wee_event_type_id` → `fk_webhook_endpoint_events_event_type_id`
- [ ] Rename index aliases `idx_wee_*` → `idx_webhook_endpoint_events_*`

**File: `061_create_integration_event_outbox_table.go`**
- [ ] Rename `fk_outbox_tenant_id` → `fk_integration_event_outbox_tenant_id`

**File: `062_create_webhook_delivery_history_table.go`**
- [ ] Rename `fk_delivery_webhook_endpoint_id` → `fk_webhook_delivery_history_webhook_endpoint_id`
- [ ] Rename `chk_delivery_final_status` → `chk_webhook_delivery_history_final_status`
- [ ] Rename index aliases `idx_delivery_*` → `idx_webhook_delivery_history_*`

- [ ] Run `go build ./...` and `go test ./...`

---

### 4.6 — Fix column ordering: `webhook_endpoints`

**File:** `internal/platform/database/migration/057_create_webhook_endpoints_table.go`

**Standard order:** PK → UUID → tenant FK → business columns → config → status → operational counters → state timestamps → audit FKs → system timestamps

Current violation: `consecutive_failures` and `last_triggered_at` appear BEFORE `description` and `metadata`.

- [ ] Reorder columns in the `CREATE TABLE` block to:
  ```sql
  webhook_endpoint_id     BIGSERIAL PRIMARY KEY,
  webhook_endpoint_uuid   UUID NOT NULL UNIQUE,
  tenant_id               BIGINT NOT NULL,
  url                     VARCHAR(2048) NOT NULL,
  secret_encrypted        TEXT,
  subscribe_all           BOOLEAN NOT NULL DEFAULT false,
  max_retries             INTEGER NOT NULL DEFAULT 3,
  timeout_seconds         INTEGER NOT NULL DEFAULT 30,
  description             TEXT,
  metadata                JSONB NOT NULL DEFAULT '{}',
  status                  VARCHAR(20) NOT NULL DEFAULT 'active',
  consecutive_failures    INTEGER NOT NULL DEFAULT 0,
  last_triggered_at       TIMESTAMPTZ,
  created_by              BIGINT,
  updated_by              BIGINT,
  created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at              TIMESTAMPTZ
  ```
- [ ] Update GORM WebhookEndpoint model field order to match (cosmetic but keeps model readable)
- [ ] Also add `NOT NULL DEFAULT '{}'` to metadata (caught from Phase 1.5)
- [ ] Run `go build ./...` and `go test ./internal/webhook/...`

---

### 4.7 — Fix column ordering: `branding`

**File:** `internal/platform/database/migration/003_create_branding_table.go`

Current violation: `is_system` and `is_active` appear immediately after `name`, before all business columns.

- [ ] Reorder columns:
  ```sql
  branding_id          BIGSERIAL PRIMARY KEY,
  branding_uuid        UUID NOT NULL UNIQUE,
  tenant_id            BIGINT NOT NULL,
  name                 VARCHAR(100),
  layout               VARCHAR(32) NOT NULL DEFAULT 'centered',
  company_name         VARCHAR(255),
  logo_url             VARCHAR(2048),
  logo_data            BYTEA,
  logo_content_type    VARCHAR(255),
  favicon_url          VARCHAR(2048),
  support_url          VARCHAR(2048),
  privacy_policy_url   VARCHAR(2048),
  terms_of_service_url VARCHAR(2048),
  metadata             JSONB NOT NULL DEFAULT '{}',
  is_system            BOOLEAN NOT NULL DEFAULT false,
  is_active            BOOLEAN NOT NULL DEFAULT false,
  created_by           BIGINT,
  updated_by           BIGINT,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at           TIMESTAMPTZ
  ```
- [ ] Note: this also applies the VARCHAR(2048) fixes for all URL columns from Phase 2.4
- [ ] Update GORM Branding model to match
- [ ] Run `go build ./...` and `go test ./internal/branding/...`

---

### 4.8 — Fix column ordering: `registration_flows` (`client_id` FK placement)

**File:** `internal/platform/database/migration/038_create_registration_flows_table.go`

Current violation: `client_id BIGINT NOT NULL` appears after `status` and boolean flags, not in the FK cluster near `tenant_id`.

- [ ] Move `client_id BIGINT NOT NULL` to immediately after `tenant_id` in the column list:
  ```sql
  registration_flow_id   BIGSERIAL PRIMARY KEY,
  registration_flow_uuid UUID NOT NULL UNIQUE,
  tenant_id              BIGINT NOT NULL,
  client_id              BIGINT NOT NULL,       -- moved here, was after status
  name                   VARCHAR(100) NOT NULL,
  description            TEXT,
  identifier             VARCHAR(255) NOT NULL,
  required_fields        JSONB NOT NULL DEFAULT '[]',
  verification_required  BOOLEAN NOT NULL DEFAULT FALSE,
  is_system              BOOLEAN NOT NULL DEFAULT FALSE,
  status                 VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
  created_by             BIGINT,
  updated_by             BIGINT,
  created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at             TIMESTAMPTZ
  ```
- [ ] Update GORM RegistrationFlow model field order to match
- [ ] Run `go build ./...` and `go test ./internal/registration/...`

---

### 4.9 — Fix column ordering: `invites` (FK cluster)

**File:** `internal/platform/database/migration/041_create_invites_table.go`

Current violation: `registration_flow_id BIGINT` appears after `invite_token` (a business value column) instead of in the FK cluster.

Also: clarify the `invited_by_user_id` vs `created_by` semantic: `invited_by_user_id` is the displayed human inviter (may be a human user). `created_by` is the system actor (may be an API key or service account). Document this clearly in a SQL comment.

- [ ] Reorder columns:
  ```sql
  invite_id             BIGSERIAL PRIMARY KEY,
  invite_uuid           UUID NOT NULL UNIQUE,
  tenant_id             BIGINT NOT NULL,
  client_id             BIGINT NOT NULL,
  registration_flow_id  BIGINT,               -- moved here, FK cluster
  invited_by_user_id    BIGINT,               -- the human who invited (displayed to recipient)
  invited_email         VARCHAR(255) NOT NULL,
  invite_token          VARCHAR(512) NOT NULL UNIQUE,
  callback_url          VARCHAR(2048),
  status                VARCHAR(20) NOT NULL DEFAULT 'pending',
  expires_at            TIMESTAMPTZ,
  used_at               TIMESTAMPTZ,
  created_by            BIGINT,               -- system actor (may differ from invited_by_user_id)
  updated_by            BIGINT,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at            TIMESTAMPTZ
  ```
- [ ] Run `go build ./...` and `go test ./internal/invite/...`

---

## Phase 5 — Index Optimization

---

### 5.1 — Drop redundant BOOLEAN indexes (poor selectivity)

BOOLEAN columns with two values have ~50% selectivity at best. Single-column BOOLEAN indexes are almost never chosen by the planner; they waste write overhead.

**File: `022_create_roles_table.go`**
- [ ] Remove `CREATE INDEX IF NOT EXISTS idx_roles_is_default ON roles (is_default);`
- [ ] Remove `CREATE INDEX IF NOT EXISTS idx_roles_is_system ON roles (is_system);`
- [ ] Remove `CREATE INDEX IF NOT EXISTS idx_roles_description ON roles (description);` (TEXT description index — poor utility)
- [ ] Remove `CREATE INDEX IF NOT EXISTS idx_roles_name ON roles (name);` (redundant with `uq_roles_tenant_name` composite)

**File: `014_create_identity_providers_table.go`**
- [ ] Remove `CREATE INDEX IF NOT EXISTS idx_identity_providers_is_default ON identity_providers (is_default);`
- [ ] Remove `CREATE INDEX IF NOT EXISTS idx_identity_providers_is_system ON identity_providers (is_system);`

**File: `046_create_email_templates_table.go`**
- [ ] Remove `CREATE INDEX IF NOT EXISTS idx_email_templates_is_default ON email_templates (is_default);`
- [ ] Remove `CREATE INDEX IF NOT EXISTS idx_email_templates_is_system ON email_templates (is_system);`

**File: `047_create_sms_templates_table.go`**
- [ ] Remove `CREATE INDEX IF NOT EXISTS idx_sms_templates_is_default ON sms_templates (is_default);`
- [ ] Remove `CREATE INDEX IF NOT EXISTS idx_sms_templates_is_system ON sms_templates (is_system);`

**File: `007_create_services_table.go`**
- [ ] Remove `CREATE INDEX IF NOT EXISTS idx_services_is_system ON services (is_system);` (boolean, poor selectivity)
- [ ] Remove `CREATE INDEX IF NOT EXISTS idx_services_status ON services (status);` (single-column; replace with composite below)
- [ ] Add: `CREATE INDEX IF NOT EXISTS idx_services_tenant_status ON services (tenant_id, status) WHERE deleted_at IS NULL;`

---

### 5.2 — Add missing composite indexes

**File: `024_create_users_table.go`**
- [ ] Add: `CREATE INDEX IF NOT EXISTS idx_users_tenant_status ON users (tenant_id, status) WHERE deleted_at IS NULL;`

**File: `057_create_webhook_endpoints_table.go`**
- [ ] Add: `CREATE INDEX IF NOT EXISTS idx_webhook_endpoints_tenant_status ON webhook_endpoints (tenant_id, status) WHERE deleted_at IS NULL;`

**File: `030_create_profiles_table.go`**
- [ ] Add: `CREATE INDEX IF NOT EXISTS idx_profiles_first_last_name ON profiles (first_name, last_name) WHERE deleted_at IS NULL;`
- [ ] Remove: `CREATE INDEX IF NOT EXISTS idx_profiles_first_name ON profiles (first_name);` (covered by composite)
- [ ] Remove: `CREATE INDEX IF NOT EXISTS idx_profiles_last_name ON profiles (last_name);` (covered by composite)
- [ ] Keep: `idx_profiles_display_name` (queried independently for display name searches)

**File: `014_create_identity_providers_table.go`**
- [ ] Add: `CREATE INDEX IF NOT EXISTS idx_identity_providers_tenant_provider ON identity_providers (tenant_id, provider, provider_type) WHERE deleted_at IS NULL;`

**File: `025_create_user_identities_table.go`**
- [ ] Add: `CREATE INDEX IF NOT EXISTS idx_user_identities_client_provider ON user_identities (client_id, provider);`

---

### 5.3 — Add GIN indexes for new TEXT[] scope columns

After Phase 2.1 converts scope columns to TEXT[], add GIN indexes so `= ANY(scopes)` containment checks are fast.

**File: `051_create_oauth_consent_grants_table.go`**
- [ ] Add: `CREATE INDEX IF NOT EXISTS idx_oauth_consent_grants_scopes ON oauth_consent_grants USING GIN (scopes);`

**File: `049_create_oauth_authorization_codes_table.go`**
- [ ] Add: `CREATE INDEX IF NOT EXISTS idx_oauth_authorization_codes_scope ON oauth_authorization_codes USING GIN (scope);`

**File: `052_create_oauth_consent_challenges_table.go`**
- [ ] Add: `CREATE INDEX IF NOT EXISTS idx_oauth_consent_challenges_scope ON oauth_consent_challenges USING GIN (scope);`

**File: `053_create_oauth_par_requests_table.go`**
- [ ] Add: `CREATE INDEX IF NOT EXISTS idx_oauth_par_requests_scope ON oauth_par_requests USING GIN (scope);`

**File: `054_create_oauth_device_codes_table.go`** *(confirm file name)*
- [ ] Add: `CREATE INDEX IF NOT EXISTS idx_oauth_device_codes_scope ON oauth_device_codes USING GIN (scope);`

**File: `055_create_oauth_ciba_requests_table.go`** *(confirm file name)*
- [ ] Add: `CREATE INDEX IF NOT EXISTS idx_oauth_ciba_requests_scope ON oauth_ciba_requests USING GIN (scope);`

**File: `064_create_oauth_broker_sessions_table.go`**
- [ ] Add: `CREATE INDEX IF NOT EXISTS idx_oauth_broker_sessions_app_scope ON oauth_broker_sessions USING GIN (app_scope) WHERE app_scope IS NOT NULL;`

---

## Phase 6 — Ephemeral Data Cleanup Jobs

Every OAuth flow table stores rows with an `expires_at`. Without scheduled deletion, these tables grow unboundedly and degrade query performance over time. Cleanup must run as a background job (cron or ticker in the application process) — not inline in request handlers.

**Decision:** Implement a single `CleanupExpiredRecords` background worker that runs on a configurable interval (default: every 15 minutes). Each DELETE targets a small batch (LIMIT 1000) to avoid long-running table locks. The worker loops until no rows are deleted (catches up), then sleeps until the next interval.

- [ ] Check `internal/oauth/cleanup_runner.go` — if it already exists, extend it rather than creating a new file; otherwise create `internal/platform/cleanup/worker.go` with a `Worker` struct that accepts a `*gorm.DB` and a tick interval
- [ ] Implement the following DELETE jobs inside the worker, each as a separate method running `DELETE ... WHERE expires_at < now() AND ... LIMIT 1000` in a loop until 0 rows affected:

**`oauth_refresh_tokens`**
- [ ] `DELETE FROM oauth_refresh_tokens WHERE expires_at < now() LIMIT 1000` — also include revoked tokens past a 30-day grace window: `OR (revoked_at IS NOT NULL AND revoked_at < now() - INTERVAL '30 days')`

**`oauth_authorize_requests`**
- [ ] `DELETE FROM oauth_authorize_requests WHERE expires_at < now() LIMIT 1000`

**`oauth_authorization_codes`**
- [ ] `DELETE FROM oauth_authorization_codes WHERE expires_at < now() LIMIT 1000`

**`oauth_par_requests`**
- [ ] `DELETE FROM oauth_par_requests WHERE expires_at < now() LIMIT 1000`

**`oauth_consent_challenges`**
- [ ] `DELETE FROM oauth_consent_challenges WHERE expires_at < now() LIMIT 1000`

**`oauth_device_codes`**
- [ ] `DELETE FROM oauth_device_codes WHERE expires_at < now() LIMIT 1000`

**`oauth_ciba_requests`**
- [ ] `DELETE FROM oauth_ciba_requests WHERE expires_at < now() LIMIT 1000`

**`oauth_broker_sessions`**
- [ ] `DELETE FROM oauth_broker_sessions WHERE expires_at < now() LIMIT 1000`

**`user_otps`**
- [ ] `DELETE FROM user_otps WHERE expires_at < now() LIMIT 1000`

**`user_sessions`** (new table 072)
- [ ] `DELETE FROM user_sessions WHERE expires_at < now() OR (revoked_at IS NOT NULL AND revoked_at < now() - INTERVAL '30 days') LIMIT 1000`

**`user_trusted_devices`** (new table 071)
- [ ] `DELETE FROM user_trusted_devices WHERE trusted_until < now() AND deleted_at IS NULL LIMIT 1000`

**`user_lockouts`** (new table 068)
- [ ] `UPDATE user_lockouts SET failed_count = 0, locked_until = NULL WHERE locked_until < now()` — reset expired lockouts rather than delete (keep the row for failed count history)

**`webauthn_challenges`** (new table 075)
- [ ] `DELETE FROM webauthn_challenges WHERE expires_at < now() - INTERVAL '1 hour' LIMIT 1000` — keep a 1-hour grace window for debugging; used challenges are safe to delete immediately but the grace window avoids clock-skew issues

**`oauth_token_revocations`** (new table 077)
- [ ] `DELETE FROM oauth_token_revocations WHERE expires_at < now() LIMIT 1000` — once the token's original expiry has passed, it can no longer be presented, so the blocklist entry is safe to remove

**`account_link_requests`** (new table 081)
- [ ] `UPDATE account_link_requests SET status = 'expired' WHERE status = 'pending' AND expires_at < now() LIMIT 1000` — mark expired rather than delete to preserve the audit trail of attempted links

**`oauth_dpop_nonces`** (new table 083)
- [ ] `DELETE FROM oauth_dpop_nonces WHERE expires_at < now() LIMIT 1000`

**`data_erasure_requests`** (GDPR processor — distinct from simple DELETE jobs)
- [ ] Implement `ProcessPendingErasureRequests` as a named job in the worker: query `data_erasure_requests WHERE status='pending' AND scheduled_at <= now() AND legal_hold=FALSE LIMIT 10`; for each row: set `status='in_progress'`, call `DataErasureService.AnonymizeUser(ctx, userID)`, then set `status='completed'` (or `status='failed'` with error logged if it errors)

- [ ] Register the worker in the application startup sequence (`internal/app/app.go` or `cmd/server/main.go`) — start as a goroutine with context cancellation wired to the server's shutdown signal so cleanup jobs finish their current batch before the process exits
- [ ] Confirm each table that is cleaned up has a partial index on `expires_at` (most do already; verify `oauth_broker_sessions` and `oauth_ciba_requests` have one)
- [ ] Run `go build ./...` and `go test ./internal/platform/cleanup/...`

---

## Phase 7 — Verification Gate

Do not mark this phase complete until every item passes on a clean database (drop and re-run all migrations).

- [ ] Drop the local development database and re-run all migrations from 001 to 084 in order: `go run ./cmd/server migrate` (or equivalent)
- [ ] Confirm zero errors in migration output
- [ ] Run `go build ./...` — zero compilation errors
- [ ] Run `go test ./...` — zero failures (run with `-count=1` to disable caching)
- [ ] Run `go vet ./...` — zero warnings
- [ ] Run `golangci-lint run` — zero lint errors (required before any push per project convention)
- [ ] Run `graphify update .` to keep the knowledge graph current
- [ ] For each new table (068–084), confirm:
  - [ ] GORM model created in the appropriate domain package
  - [ ] Repository created with standard CRUD + domain-specific methods
  - [ ] Service method created if business logic is needed
  - [ ] Handler created and registered on the appropriate router (internal/public) if the entity needs an API surface
  - [ ] Handler test written per the 9-step checklist in `docs/contributing/testing.md`
  - [ ] `*Repo` field added to `internal/app/repositories.go` and wired in `initRepos`
  - [ ] Service field added to `internal/app/services.go` and `server.Application` where a dedicated service was created
- [ ] Check `internal/app/application_test.go` (if it exists) for struct shape validation — new repo/service fields on `server.Application` will cause compile failures in that file; update the test struct accordingly
- [ ] Check Phase 5.2 — if `idx_api_keys_status_expires_at` is still listed there, remove it (the `api_keys` table is being dropped in Phase 3.3; the index is moot)
- [ ] For any domain with an existing gRPC surface (`internal/oauth/grpc_*.go` or similar), check whether the new data (e.g., token revocation status, session info) must also be exposed via the gRPC handler — add it if the gRPC contract needs updating
- [ ] Confirm the following code paths work end-to-end (manual test with a running server):
  - [ ] User login → `last_login_at` and `login_count` updated on `users`
  - [ ] TOTP enroll → `users.is_totp_enabled` flips to `true` via trigger
  - [ ] TOTP remove → `users.is_totp_enabled` flips to `false` via trigger
  - [ ] WebAuthn enroll → `users.is_webauthn_enabled` flips to `true` via trigger
  - [ ] Email change flow → OTP created in `user_otps`, not on `users`
  - [ ] Failed login → `user_lockouts` row upserted
  - [ ] Successful login → `user_lockouts.failed_count` reset
  - [ ] Redirect URI delete → soft-deleted (deleted_at set), not hard-deleted
  - [ ] Scope check in consent → uses `= ANY(scopes)`, not LIKE
  - [ ] OAuth authorize request with `registration_flow` identifier → resolves to `registration_flow_id` FK; invalid identifier returns `invalid_request` error
  - [ ] Successful login → `user_sessions` row created
  - [ ] Logout → `user_sessions.revoked_at` set (not deleted)
  - [ ] Admin force-logout → `user_sessions` revoked with `reason='admin_revoke'`
  - [ ] Admin creates a user → row appears in `management_audit_log` with `action='user.create'`
  - [ ] WebAuthn registration begin → `webauthn_challenges` row created; WebAuthn registration complete → challenge marked `used_at`, credential stored; replay attempt → rejected
  - [ ] Token issuance → signed with key from `signing_keys`; `/.well-known/jwks.json` returns the active public key's JWK
  - [ ] Logout → access token `jti` inserted into `oauth_token_revocations`; subsequent use of that token rejected by introspection
  - [ ] RFC 8693 token exchange → row inserted in `oauth_token_exchanges`
  - [ ] Workload identity federation: GitHub Actions OIDC token exchanged for platform token using configured `workload_identity_federations` rule
  - [ ] User submits erasure request → `data_erasure_requests` row created with `scheduled_at = now() + 30 days`; background worker processes it and anonymizes PII fields
  - [ ] Social login email collision → `account_link_requests` row created; user confirms → `user_identities` row linked to existing account
  - [ ] Policy updated → version snapshot written to `policy_version_history`; `GET /policies/{uuid}/history` returns the snapshot list
  - [ ] Cleanup worker runs → expired `oauth_authorization_codes` rows deleted, expired `user_otps` rows deleted, expired `webauthn_challenges` rows deleted, expired `account_link_requests` marked `status='expired'`, expired `oauth_token_revocations` deleted, expired `oauth_dpop_nonces` deleted
  - [ ] DPoP flow: client with `dpop_required=TRUE` sends token request → server issues nonce (`DPoP-Nonce` header, row in `oauth_dpop_nonces`) → client resends with nonce in DPoP proof → nonce marked `used_at`, token issued; replay of same nonce → rejected
  - [ ] SCIM provisioning: external directory sends `POST /scim/v2/Users` with bearer token → bearer hash matched against `scim_configurations.bearer_token_hash` → user created with `provisioning_source='scim'` and `external_id` set
  - [ ] Verify clean migration run does not error on `ALTER TABLE api_keys` (the 'api_keys' entry must be removed from the deferred FK loop in migration 024 per section 3.3)

---

## Unaudited Migrations — Manual Review Required

The following four migration files could not be read during the audit due to filesystem permission restrictions. They must be manually reviewed against the same standards applied to all other migrations before marking this document complete. Apply any findings as edits to those migration files (in place — no new alter migrations).

**`027_create_user_tokens_table.go`** — Audited. Currently a polymorphic table for five token types, all defined in `internal/shared/constants.go`. The DDL comment listing `'refresh','api','reset_password','session'` is **stale and wrong** — those values are never written anywhere in the codebase. The actual types in use are:

| Type constant | Managed by | Purpose |
|---|---|---|
| `user:session` | `SessionService` in `authn/service_session.go` | Live login session; has dedicated session columns |
| `user:mfa:trusted_device` | `MFAService` in `mfa/service_mfa.go` via private GORM struct | MFA device trust cookie |
| `user:email:verification` | `EmailVerificationService` | URL token emailed to user |
| `user:password:reset` | `ForgotPasswordService` | URL token emailed to user |
| `user:magic_link` | `LoginService` | URL token for passwordless login |

The session-specific columns (`last_used_at`, `idle_timeout_seconds`, `absolute_expires_at`) are documented in the model as NULL for non-session rows. There is currently **no `token_type` CHECK constraint** — the stale DDL comment is the only hint at valid values; any string can be inserted without a DB error.

`user_tokens` is intentionally separate from `user_otps`: OTPs are short numeric codes the user types in (brute-force `failed_attempts` counter, `channel`+`recipient` delivery tracking, lives in `notifier` package). Tokens are long opaque strings embedded in a URL link or cookie. Different UX flow, different security model, different columns. Do not merge them.

**Checklist — this file needs three phases of changes (do in order):**

**Phase A — Do immediately (safe now, no dependency on other sections):**
- [ ] Correct the stale DDL comment to reflect the actual five types in use
- [ ] Add a `token_type` CHECK constraint with all five current valid values — this is a safety net against future accidental type proliferation:
  ```sql
  DO $$
  BEGIN
      IF NOT EXISTS (
          SELECT 1 FROM pg_constraint WHERE conname = 'chk_user_tokens_token_type'
      ) THEN
          ALTER TABLE user_tokens ADD CONSTRAINT chk_user_tokens_token_type
              CHECK (token_type IN (
                  'user:session',
                  'user:email:verification',
                  'user:password:reset',
                  'user:magic_link',
                  'user:mfa:trusted_device'
              ));
      END IF;
  END$$;
  ```
- [ ] Confirm `user_token_uuid UUID NOT NULL UNIQUE` exists; add if absent
- [ ] Confirm `created_at TIMESTAMPTZ NOT NULL DEFAULT now()` has NOT NULL; fix if absent
- [ ] Add composite partial index for the active-token lookup hot path:
  ```sql
  CREATE INDEX IF NOT EXISTS idx_user_tokens_active
      ON user_tokens (user_id, token_type)
      WHERE is_revoked = false;
  ```

**Phase B — After section 3.12 (user_trusted_devices) is fully implemented:**
- [ ] Update the CHECK constraint to remove `'user:mfa:trusted_device'` — `MFAService.IssueTrustedDevice` and `TrustedDeviceValid` will have been rerouted to `user_trusted_devices` by that point. Removing the type from the CHECK before the code migration is done will break runtime inserts.

**Phase C — After section 3.15 (user_sessions) is fully implemented:**
- [ ] Update the CHECK constraint to remove `'user:session'` — `SessionService` will have been rerouted to `user_sessions` by that point
- [ ] Drop the now-unused session-specific columns: `last_used_at`, `idle_timeout_seconds`, `absolute_expires_at` — verify they are NULL on all remaining rows before dropping

**`032_create_user_mfa_totp_secrets_table.go`** — Stores TOTP shared secrets (encrypted). Check for:
- `secret_encrypted TEXT` — fine as TEXT (encrypted blobs vary in length)
- `is_active BOOLEAN` — must have NOT NULL
- Missing NOT NULL on timestamps
- The trigger attachment added in Phase 1.3 goes in this file — confirm the file ends with the `trg_sync_totp_flag` trigger CREATE statement

**`036_create_user_password_history_table.go`** — Stores hashed previous passwords for reuse prevention. Check for:
- `password_hash TEXT` — fine as TEXT (hash length varies by algorithm)
- Missing NOT NULL on timestamps
- Missing UUID column
- Ensure there is a partial index on `(user_id, created_at DESC)` to efficiently query the N most recent passwords
- Confirm the table is append-only (no UPDATE allowed) — consider adding a trigger that raises an exception on UPDATE, same pattern as `auth_events`

**`050_create_oauth_refresh_tokens_table.go`** — Audited. Stores OAuth 2.0 protocol refresh tokens issued via `POST /oauth/token`. Completely separate from the internal authn refresh mechanism (which is stateless JWT + Redis JTI denylist — no DB table). The two refresh systems coexist by design and serve different callers: `oauth_refresh_tokens` is for third-party OAuth clients; the JWT refresh is for the hosted login UI.

Confirmed present: `token_hash` (UNIQUE), `family_id` (UUID, indexed — foundation of `RevokeByFamily` token theft detection), `client_id` FK, `user_id` FK, `tenant_id` FK, `is_revoked`, `revoked_at`, `last_used_at`, `expires_at`. The consistency CHECK `(is_revoked=FALSE AND revoked_at IS NULL) OR (is_revoked=TRUE AND revoked_at IS NOT NULL)` exists.

Checklist for this file:
- [ ] `scope TEXT NOT NULL DEFAULT ''` → `scope TEXT[] NOT NULL DEFAULT '{}'` (Phase 2.1 change)
- [ ] Confirm `expires_at`, `created_at`, `last_used_at` all have `NOT NULL`; add where missing
- [ ] Ensure partial index exists for active token lookup: `CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_active ON oauth_refresh_tokens (user_id, client_id) WHERE is_revoked = false;`
- [ ] `family_id` — confirmed present; no action needed
- [ ] **`session_id` FK — do NOT add yet.** The internal authn refresh flow (JWT + Redis) has no DB session concept; only the OAuth protocol flow (`oauth_refresh_tokens`) would need this. Adding `session_id FK → user_sessions` requires that every OAuth authorization code exchange also creates a `user_sessions` row, which is currently not done. This is a future enhancement, not part of the current refactor. Scope it as a separate feature when the OAuth flow is extended to write to `user_sessions`.
- [ ] `ip_address INET` and `user_agent TEXT` — add these two columns if absent. They are used to show third-party OAuth app sessions in the admin console alongside first-party sessions, without requiring a separate sessions table for the OAuth flow.
- [ ] `acr VARCHAR(10) NOT NULL DEFAULT '1'` and `amr TEXT[] NOT NULL DEFAULT '{}'` — add if absent. Allows token introspection to return auth strength without a join.

---

## Appendix — Final Schema Decisions Reference

| Table | Decision | Rationale |
|---|---|---|
| `users.is_totp_enabled` | **Keep (denormalized)** + trigger | Hot read path on every login; trigger ensures consistency |
| `users.is_webauthn_enabled` | **Keep (denormalized)** + trigger | Same |
| `users.mfa_enabled_at` | **Rename → `first_mfa_enrolled_at`** | Resolve semantic ambiguity; set once on first enrollment |
| `users.pending_email` + `email_change_otp*` | **Remove** | Route through `user_otps` with `channel='email_change'` |
| `users.last_login_at` | **Add** | Required for dormant accounts, GDPR, security dashboards |
| `users.external_id` | **Add** | Required for SCIM 2.0 compliance |
| `users.created_by / updated_by` | **Add** | Admin audit — who provisioned this user |
| `profiles.phone` | **Remove** | `users.phone` is canonical; no dual ownership |
| `user_settings.social_links` | **Remove entirely** | No OIDC basis; scope violation — belongs in tenant product DB or `users.metadata` |
| `user_settings.marketing_email_consent` | **Remove entirely** | CRM concern; belongs in tenant email platform (Mailchimp, HubSpot) |
| `user_settings.sms_notifications_consent` | **Remove entirely** | Product notification preference; no auth function |
| `user_settings.push_notifications_consent` | **Remove entirely** | Auth service has no push infrastructure |
| `user_settings.profile_visibility` | **Remove entirely** | Social-network feature; no OIDC equivalent |
| `user_settings.preferred_contact_method` | **Remove entirely** | Product communication preference; MFA channel derivable from enrolled factors |
| `user_settings.data_processing_consent` | **Remove — replaced by `user_consents`** | Bare boolean violates GDPR Art.7 demonstrability; needs version + IP + audit trail |
| `user_settings.terms_accepted_at` | **Remove — replaced by `user_consents`** | Bare timestamp cannot prove which ToS version; needs version tracking |
| `user_settings.privacy_policy_accepted_at` | **Remove — replaced by `user_consents`** | Same issue as `terms_accepted_at` |
| `user_settings.emergency_contact_*` | **Remove entirely — no replacement** | Third-party PII (distinct data subject); GDPR Art.5(1)(b) violation; not an auth concern |
| `profiles.bio` | **Remove entirely** | Not in OIDC Core §5.1; social feature |
| `profiles.social_links` | **Remove entirely** | No OIDC basis; belongs in tenant product DB |
| `profiles.is_default` | **Remove — replaced by `UNIQUE(user_id)` partial index** | No multi-profile auth feature; enforce single profile at DB level |
| `profiles.address / city / country / suffix` | **Remove — move to `profiles.metadata`** | OIDC §5.1.1 requires structured address object; flat columns are non-compliant |
| `users.is_profile_completed` | **Remove entirely** | Product onboarding state; definition is tenant-specific; use `users.metadata` |
| `users.is_account_completed` | **Remove entirely** | Product onboarding vocabulary; belongs in tenant product DB |
| `tenants.is_completed` | **Remove — replaced by `tenants.status='pending'`** | Product onboarding flag; bootstrap state modelled via `status` lifecycle |
| `api_keys` / `api_key_apis` / `api_key_permissions` | **Remove entirely** | Duplicate of M2M OAuth (client credentials); identical authorization model, OAuth is strictly better |
| `api_permissions` table | **Remove entirely** | Redundant with `permissions.api_id`; creates contradiction risk |
| `tenant_services` table | **Remove entirely** | Redundant with `services.tenant_id`; services are fully tenant-encapsulated, no cross-tenant sharing |
| `oauth_authorize_requests.registration_flow` | **Normalize → `registration_flow_id` FK** | String identifier has no referential integrity; FK enforces the relationship |
| `scope TEXT` (all OAuth tables) | **Change → TEXT[]** | Correct containment semantics, indexable |
| `transport TEXT` (webauthn) | **Change → TEXT[]** | Multi-value; was comma-separated string |
| `required_fields TEXT` (reg flows) | **Change → JSONB** | JSON in TEXT has no DB validation |
| `client_uris.type` CHECK hyphens | **Change → underscores** | Consistent with all other CHECK values |
| `is_backup_state` (webauthn) | **Rename → `is_backup_active`** | Grammatically correct |
| `is_used` (OAuth tables) | **Rename → `used`** | Consistent with `user_otps` and `user_mfa_backup_codes` |
| `tenant_members.role` CHECK | **Add `admin`** | Enterprise delegated admin tier |
| `user_lockouts` | **New table (068)** | Brute-force state as fast lookup, not a COUNT on auth_events |
| `user_emergency_contacts` | **Cancelled — do not create** | Third-party PII with no auth purpose; extraction formalises the scope violation |
| `user_consents` | **New table (070)** | GDPR-compliant versioned consent history |
| `user_trusted_devices` | **New table (071)** | Device-aware MFA step-down and security console |
| `user_sessions` | **New table (072)** | Active session tracking — admin force-logout, concurrent session limits, "active sessions" UI |
| `user_tokens` polymorphic cleanup | **Add CHECK constraint now; migrate types out in phases** | No CHECK constraint exists today — stale DDL comment lists wrong values. Add CHECK with all 5 current types immediately. Remove `user:session` after section 3.15 migration; remove `user:mfa:trusted_device` after section 3.12 migration. Never remove a type before the code is rerouted — doing so breaks runtime inserts. |
| `user_tokens` vs `user_otps` | **Keep separate — confirmed by audit** | OTPs: short numeric codes, `failed_attempts` brute-force counter, `channel`+`recipient` delivery tracking, lives in `notifier` package (SMS/email MFA, SMS login). Tokens: long opaque strings in URL links, no attempt counter, lives in `user` package (password reset, email verify, magic link). Genuinely different; do not merge. |
| `oauth_refresh_tokens` session_id FK | **Deferred — not part of this refactor** | Internal authn refresh is stateless JWT + Redis JTI denylist (no DB). OAuth protocol refresh tokens are separate. Adding `session_id FK → user_sessions` requires the OAuth code exchange to also create a session row — that is a new feature, not a schema fix. |
| `management_audit_log` | **New table (073)** | Admin action audit trail (user CRUD, role changes, etc.) — SOC 2 / ISO 27001 compliance |
| `client_roles` | **New table (074)** | Role assignment for system clients (service identities) — same role model as users, matches GCP/Azure service account role binding |
| `clients.service_id` | **Add FK column** | Links a system client to its registered service — completes the service identity model without needing a separate service_accounts table |
| `api_keys` / `api_key_apis` / `api_key_permissions` | **Remove** | Duplicate of M2M OAuth (client credentials); identical authorization model, OAuth is strictly better |
| `webauthn_challenges` | **New table (075)** | FIDO2 spec-required server-side challenge store; prevents replay attacks during WebAuthn registration and authentication |
| `signing_keys` | **New table (076)** | JWKS private key store for OAuth AS; enables key rotation without redeployment; private keys stored encrypted at rest |
| `oauth_token_revocations` | **New table (077)** | JWT jti blocklist for access token revocation; enables immediate session termination even for stateless JWTs |
| `oauth_token_exchanges` | **New table (078)** | Append-only audit log of RFC 8693 token exchanges (impersonation / delegation); required for accountability in service-to-service access |
| `workload_identity_federations` | **New table (079)** | Trust config for external OIDC issuers (Kubernetes, GitHub Actions, GitLab CI); enables keyless workload authentication — eliminates static CI/CD credentials |
| `data_erasure_requests` | **New table (080)** | GDPR Article 17 right-to-erasure request lifecycle; provides compliance audit trail and drives the background anonymization worker |
| `account_link_requests` | **New table (081)** | Short-lived pending state for social-login email-collision identity linking; prevents account takeover via provider email |
| `policy_version_history` | **New table (082)** | Append-only snapshot of every policy change; enables rollback, diff UI, and SOC 2 / ISO 27001 audit evidence for access control changes |
| `oauth_dpop_nonces` | **New table (083)** | RFC 9449 DPoP server-nonce registry — required alongside `clients.dpop_required`; prevents DPoP proof replay attacks within a proof's TTL |
| `scim_configurations` | **New table (084)** | Per-tenant SCIM 2.0 endpoint configuration (bearer token hash, sync direction, attribute mapping) — enables bidirectional enterprise directory sync (Okta, Azure AD, Google Workspace) |
| `user_identities.client_id` | **Change FK cascade: SET NULL** | Deleting an OAuth client must not hard-delete users' identity links; identity is user data, not client data |
| `user_lockouts` UNIQUE | **Change to `(tenant_id, identifier)`** | Pre-auth brute force must track by presented credential string, not resolved user_id — non-existent usernames must also be rate-limited |
| `management_audit_log.actor_api_key_id` | **Removed** | api_keys table is being removed (section 3.3); column had no FK and referenced a non-existent table |
| `policy_version_history` FK | **ON DELETE RESTRICT** | Deleting a policy must not silently cascade-delete its audit history; RESTRICT forces explicit history handling |
| `workload_identity_federations` UNIQUE | **Partial index** | Soft-delete-aware: inline UNIQUE rejects re-creation of a soft-deleted config with the same name |
| Ephemeral cleanup worker | **New background job** | Scheduled DELETE/UPDATE for expired OAuth codes, OTPs, sessions, lockouts, challenges, token revocations, DPoP nonces, and link requests — prevents unbounded table growth |

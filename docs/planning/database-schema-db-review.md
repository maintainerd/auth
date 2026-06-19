# DB Review: Full Schema Audit (53 Tables)
_Reviewed: 2026-06-18_

**Scope:** All 53 tables across 57 migration files. PostgreSQL + GORM. Multi-tenant auth platform.

---

## Good Practices

Things done correctly -- these should be preserved.

- **Dual-identity pattern (BIGSERIAL PK + UUID)**: Every entity table has `{entity}_id BIGSERIAL PK` for fast joins and `{entity}_uuid UUID NOT NULL UNIQUE` for external exposure. Prevents enumeration attacks. Matches enterprise PK+UUID standard.

- **Native PostgreSQL UUID type**: UUIDs stored as `UUID` (16 bytes) not `VARCHAR(36)`. Index-efficient, 4x smaller than string representation.

- **Partial unique indexes everywhere**: Uniqueness constraints like `uq_tenants_identifier (identifier) WHERE deleted_at IS NULL` correctly allow re-use of soft-deleted identifiers. Applied consistently across tenants, services, APIs, permissions, roles, clients, policies, auth flows, api keys, templates. This is an enterprise-grade pattern most apps miss.

- **Partial unique for system singleton**: `uq_tenants_single_system (is_system) WHERE is_system = TRUE AND deleted_at IS NULL` guarantees exactly one system tenant. Excellent constraint-level enforcement.

- **Partial unique for active branding**: `uq_branding_active_per_tenant (tenant_id) WHERE is_active AND deleted_at IS NULL` ensures one active branding per tenant at DB level. No race conditions.

- **JSONB + GIN indexes for flexible config**: `metadata`, `config`, `document` columns use JSONB with GIN indexes where queried. Correct PostgreSQL pattern for schema-flexible data.

- **CHECK constraints on status/enum columns**: All status fields and provider fields have explicit CHECK constraints (e.g., `CHECK provider IN ('smtp','ses','sendgrid','mailgun','postmark','resend')`). Prevents invalid data at DB level.

- **Immutability trigger on `auth_events`**: `protect_auth_events_immutable()` trigger blocks UPDATE and unauthorized DELETE. Audit logs must be append-only. This is SOC2 CC7.2 compliant.

- **Encrypted sensitive fields**: `password_encrypted`, `auth_token_encrypted`, `secret_encrypted` naming pattern. Secrets are never stored as plaintext. `secret_hash` for credential verification. `previous_secret_hash` + `previous_secret_encrypted` + `previous_secret_expires_at` for secret rotation on clients -- enterprise-grade credential lifecycle.

- **Password stored as hash**: `users.password` is TEXT (bcrypt/argon2 output). `user_password_history` tracks previous hashes for reuse prevention. Matches SOC2 CC6.2.

- **Soft deletes applied correctly**: Mutable business entities (tenants, users, clients, roles, etc.) have `deleted_at TIMESTAMPTZ`. Append-only tables (`auth_events`, `security_settings_audit`, `user_password_history`, OAuth tokens) correctly omit `deleted_at`. This matches the rule: soft-delete for lifecycle entities, never for logs.

- **Temporal fields consistently applied**: `created_at`, `updated_at` on every mutable table. `deleted_at` only where soft-delete is needed. OAuth flow tables (auth codes, refresh tokens, PAR, device codes, CIBA) use `expires_at` + `created_at` only (no `updated_at`) since they're effectively immutable after creation -- correct design.

- **`created_by` / `updated_by` audit columns**: Applied to all 12+ entity tables (tenants, branding, services, APIs, permissions, roles, clients, api keys, templates, email/sms configs, security settings, ip rules). Added retroactively via migration 024 with proper FK to users. Tracks who created/modified every record.

- **Tenant scoping on all business tables**: Every business entity has `tenant_id BIGINT NOT NULL` with FK to tenants ON DELETE CASCADE. This is row-level multi-tenancy done right.

- **Composite unique indexes include `tenant_id` first**: e.g., `uq_services_tenant_name (tenant_id, name)`, `uq_permissions_tenant_api_name (tenant_id, api_id, name)`. Ensures uniqueness is per-tenant and the index is useful for tenant-scoped queries. Matches the multi-tenancy composite index rule.

- **INET type for IP addresses**: `auth_events.ip_address`, `ip_restriction_rules.ip_address` use PostgreSQL `INET` type, enabling native IP comparison operators and CIDR matching. Better than VARCHAR(45).

- **Transactional outbox pattern**: `integration_event_outbox` with `is_published`, `claimed_at`, `published_at`, event versioning (`event_version`), `subject_uuid`/`subject_type`, `changed_fields` JSONB, `trace_id`, `request_id`. Multiple partial indexes for relay polling. `FOR UPDATE SKIP LOCKED` in the claim query -- correct concurrency pattern for at-least-once delivery without locking contention.

- **Webhook delivery with retry/dead-letter**: `webhook_delivery_history` tracks `attempt_count`, `response_status`, `final_status CHECK('pending','success','failed','dead_letter')`, `next_retry_time`. Enterprise-grade webhook reliability.

- **Complete MFA support**: TOTP secrets (`user_totp_secrets`), WebAuthn credentials (`user_webauthn_credentials`), backup codes (`user_backup_codes`), SMS phone (`user_sms_phones`). Covers all NIST 800-63B authenticator types.

- **WebAuthn credential storage is correct**: `credential_key_id` BYTEA UNIQUE, `public_key` BYTEA, `aaguid` UUID, `sign_count` INTEGER, `transport` TEXT[], `backup_eligible`/`backup_state` BOOLEAN. Matches WebAuthn Level 2 spec storage requirements.

- **OAuth flows are spec-compliant**: Authorization codes with PKCE (`code_challenge`, `code_challenge_method`), refresh token families (`family_id` UUID for rotation detection), PAR (`request_uri_hash`), device codes (`user_code` VARCHAR(9)), CIBA (`auth_req_id_hash`, `binding_message`, `interval`, `notification_sent_at`). All token values stored as hashes, not plaintext.

- **Refresh token rotation detection**: `family_id UUID` on `oauth_refresh_tokens` enables detecting token replay attacks (RFC 6819 Section 5.2.2.3). Combined with `is_revoked` partial index for fast family lookups.

- **Session management on `user_tokens`**: `idle_timeout_seconds`, `absolute_expires_at`, `last_used_at` fields enable both idle and absolute session timeout enforcement. Matches OWASP session management guidelines.

- **All raw SQL is parameterized**: Grep of all raw SQL queries confirms no string interpolation of user input. `set_config()` calls use parameterized binding. No SQL injection vectors found.

- **Idempotent seeders**: Seeder uses create-or-update patterns, scoped by tenant. Safe to run multiple times without duplicates.

- **Explicit FK + local key in GORM relationships**: e.g., `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`. Prevents the silent wrong-PK bug that occurs with GORM defaults on custom-PK models.

- **Array columns with GIN index**: `clients.grant_types TEXT[]`, `clients.response_types TEXT[]` with `CREATE INDEX idx_clients_grant_types ON clients USING GIN (grant_types)`. Enables `@>` (contains) queries for grant type filtering.

- **Security settings audit trail**: `security_settings_audit` logs every change to security config with `old_config`/`new_config` JSONB snapshots, `change_type`, `ip_address`, `user_agent`. Immutable (no updated_at, no deleted_at). SOC2 CC7.2 compliant.

- **Event types with versioning**: `event_types.version INTEGER DEFAULT 1` enables schema evolution of event payloads without breaking consumers.

---

## Bad Practices

Issues found, ordered by severity.

### CRITICAL -- Unscoped Base Repository Methods

**Table/File**: `internal/platform/database/base_repository.go` (all entity tables)
**What**: `FindByUUID`, `FindByID`, `FindAll`, `DeleteByUUID`, `DeleteByID`, `UpdateByUUID`, `UpdateByID` perform queries with **no `tenant_id` filter**. Any domain code calling these base methods can read/modify records belonging to other tenants.
**Why it's bad**: Cross-tenant data leakage. In a multi-tenant auth platform, this is the highest-severity vulnerability class. An attacker who guesses/enumerates a UUID could access another tenant's clients, roles, API keys, or security settings.
**Evidence**: All OAuth repositories (`oauth/repository_*.go`) inherit from `BaseRepository` and several operations use base methods without adding tenant scope. `secpolicy/repository_setting.go`, `invite/repository_invite.go` also rely on base `FindByUUID`.
**Recommendation**: Add a `tenant_id` parameter to all base repository methods and enforce it in the WHERE clause. Alternatively, implement a GORM scope/callback that automatically injects `tenant_id` from request context.

### CRITICAL -- Unscoped User Lookups

**Table/File**: `internal/user/repository_user.go:95-125`
**What**: `FindByUsername(username)`, `FindByEmail(email)`, `FindByPhone(phone)`, `FindByPendingEmail(email)` query the `users` table with **no `tenant_id`** scope. The code even has comments acknowledging this (`// Used for global uniqueness checks`).
**Why it's bad**: Returns users from any tenant. If used in auth flows (login, password reset), an attacker could authenticate as a user in a different tenant or trigger operations targeting wrong-tenant users.
**Evidence**: Scoped variants (`FindByEmailAndTenantID`, etc.) exist but unscoped methods remain in the interface and are callable by any service.
**Recommendation**: Remove unscoped methods from the repository interface entirely. If global uniqueness checks are needed (e.g., email uniqueness across all tenants), implement them as a separate, explicitly-named method with restricted access and ensure they never return the full user record.

### MAJOR -- Missing Composite Indexes on High-Frequency Query Columns

**Table/File**: `024_create_users_table.go`, multiple repository files
**What**: Several frequently-queried column combinations lack composite indexes:

| Missing Index | Queried At | Impact |
|---|---|---|
| `(email, tenant_id)` on `users` | `repository_user.go:130` | Full table scan on login-by-email |
| `(username, tenant_id)` on `users` | `repository_user.go:146` | Full table scan on login-by-username |
| `(phone, tenant_id)` on `users` | `repository_user.go:162` | Full table scan on login-by-phone |
| `(pending_email, tenant_id)` on `users` | `repository_user.go:178` | Full table scan on email-change check |
| `(user_id, client_id)` on `user_identities` | `repository_user_identity.go:78` | Scan on identity lookup per login |
| `(user_id, provider)` on `user_identities` | `repository_user_identity.go:102` | Scan on social login check |
| `(tenant_id, status)` on `ip_restriction_rules` | `repository_ip_restriction_rule.go:73` | Scan on every request IP check |
| `(tenant_id, created_at)` on `security_settings_audit` | `repository_settings_audit.go:56` | Scan on audit log listing |
| `(channel, recipient, used, expires_at)` on `user_otps` | `repository_user_otp.go:38` | Scan on OTP verification |
| `(event_type, tenant_id)` on `auth_events` | `repository_event.go:220` | Scan on event count queries |
| `(created_at, final_status)` on `webhook_delivery_history` | `repository_delivery_history.go:102` | Scan on delivery cleanup |

**Why it's bad**: At scale (100K+ users, millions of auth events), these queries degrade from milliseconds to seconds. Login and OTP verification are latency-sensitive hot paths.
**Recommendation**: Add composite indexes matching the query patterns. Put equality columns first, range columns last.

### MAJOR -- N+1 Query Pattern on Token Exchange

**Table/File**: `internal/user/repository_user.go:260-275`
**What**: `FindBySubAndClientID` issues 7+ separate SQL queries via nested Preloads:
```
Preload("UserIdentities.Tenant")
Preload("UserIdentities.Client.IdentityProvider.Tenant")
Preload("UserIdentities.Client.IdentityProvider")
Preload("UserIdentities.Client")
Preload("UserRoles.Role.RolePermissions.Permission")
Preload("Profile", "is_default = ?", true)
```
**Why it's bad**: Called on every token exchange/userinfo request. For a user with N identities and M roles, this fans out to 7+N+M queries. At high concurrency this becomes a database bottleneck.
**Recommendation**: Use a hand-crafted JOIN query that fetches the required data in 1-2 queries, or use GORM's `Joins` method with selective column loading. Alternatively, denormalize critical auth data into a materialized view or cache.

### MAJOR -- `users.email` VARCHAR(255) Not RFC-Compliant

**Table/File**: `024_create_users_table.go:22`
**What**: Email column is `VARCHAR(255)`. RFC 5321 specifies maximum email length as 254 characters.
**Why it's bad**: Minor data correctness issue, but also signals that email validation at the DB level is slightly off. More importantly, the column lacks `NOT NULL` -- email is nullable, which may or may not be intentional.
**Recommendation**: Change to `VARCHAR(254)`. If email is required for all users, add `NOT NULL`. If some users authenticate via phone-only or social providers without email, keep nullable but document the design decision.

### MAJOR -- Boolean Flags Instead of Timestamps for Verification

**Table/File**: `024_create_users_table.go:25-26`
**What**: Uses `is_email_verified BOOLEAN`, `is_phone_verified BOOLEAN` instead of `email_verified_at TIMESTAMPTZ`, `phone_verified_at TIMESTAMPTZ`.
**Why it's bad**: Loses the **when** of verification. SOC2 CC6.6 requires tracking email/phone verification events with timestamps. A boolean tells you it happened but not when, making compliance audits harder. Same issue with `is_totp_enabled` / `is_webauthn_enabled` vs. timestamps.
**Recommendation**: Replace boolean flags with nullable timestamps: `email_verified_at TIMESTAMPTZ NULL` (NULL = not verified, timestamp = verified). Keep `mfa_enabled_at` as-is (already correct). Add `email_verified_at`, `phone_verified_at`, `totp_enabled_at`, `webauthn_enabled_at`. The application code can derive the boolean from `IS NOT NULL`.

### MAJOR -- Missing `last_login_at` / `last_login_ip` on Users

**Table/File**: `024_create_users_table.go`
**What**: The `users` table has no `last_login_at` or `last_login_ip` columns.
**Why it's bad**: SOC2 CC6.1 requires session tracking including last login time and IP. While `auth_events` logs every login, querying a separate table for "when did this user last log in?" is expensive and requires a JOIN. The `last_login_at` column on users is a standard denormalization for dashboard/admin views.
**Recommendation**: Add `last_login_at TIMESTAMPTZ NULL` and `last_login_ip INET NULL` to the users table. Update them on each successful login alongside the auth_event insert.

### MEDIUM -- VARCHAR Status Columns Instead of PostgreSQL ENUMs

**Table/File**: Multiple migration files
**What**: Status fields use `VARCHAR(20) CHECK IN (...)` instead of PostgreSQL `CREATE TYPE ... AS ENUM (...)`. Examples: `tenants.status`, `services.status`, `clients.status`, `roles.status`, `permissions.status`, `apis.status`.
**Why it's bad**: VARCHAR + CHECK is functionally correct but:
1. No type safety at the PostgreSQL level (CHECK only validates on write, not on schema introspection)
2. Missing from `pg_type` catalog, so ORMs and schema tools can't discover valid values
3. Adding a new value requires `ALTER TABLE ... DROP CONSTRAINT ... ADD CONSTRAINT` instead of `ALTER TYPE ... ADD VALUE`
4. Slightly more storage (variable-length vs. 4-byte enum internal)
**Recommendation**: For a pre-production project, consider converting high-traffic status columns to PostgreSQL ENUMs. Lower priority since CHECK constraints are functionally equivalent for data integrity.

### MEDIUM -- `tenant_services` Junction Table Missing UUID

**Table/File**: `008_create_tenant_services_table.go`
**What**: `tenant_services` has no `{entity}_uuid` column. It's the only junction table that breaks the dual-identity pattern.
**Why it's bad**: If this table ever needs to be referenced in an API response (e.g., "list services for tenant"), there's no opaque external identifier. The integer PK would be exposed.
**Recommendation**: Add `tenant_service_uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid()`. Low priority since junction tables are rarely exposed directly.

### MEDIUM -- Paginated Listings Missing Preloads

**Table/File**: `internal/user/repository_user.go:350-393`, `internal/webhook/repository_endpoint.go:71`
**What**: `FindPaginated` on users only preloads `Profile`. If handlers access `user.UserRoles` or `user.UserIdentities`, GORM silently issues lazy queries per row. `FindByTenantID` on webhook endpoints has no preloads at all.
**Why it's bad**: N+1 queries on paginated lists. For 50 users per page, accessing roles generates 50 extra queries.
**Recommendation**: Preload all relationships that will be accessed in the response serialization. Or use a DTO pattern where the handler specifies which relations to load.

### MEDIUM -- No `anonymized_at` Column for GDPR Compliance

**Table/File**: `024_create_users_table.go`
**What**: No `anonymized_at TIMESTAMPTZ NULL` column on users or profiles.
**Why it's bad**: GDPR "right to be forgotten" requires either hard delete or anonymization. Soft delete alone is not sufficient -- the PII is still readable. An `anonymized_at` timestamp marks records that have been scrubbed in-place (email replaced with hash, name with "Deleted User", etc.) while preserving referential integrity.
**Recommendation**: Add `anonymized_at TIMESTAMPTZ NULL` to `users` and `profiles`. Implement an anonymization function that scrubs PII columns and sets the timestamp. Lower priority if GDPR is not yet a requirement.

### MEDIUM -- No `schema_version` on Tables with JSONB Config Blobs

**Table/File**: `security_settings`, `tenant_settings`, `email_config`, `sms_config`, `clients.config`, `identity_providers.config`
**What**: Multiple tables store complex configuration as JSONB but have no `schema_version` column.
**Why it's bad**: When the JSON structure evolves (e.g., adding a new key to `security_settings.mfa_config`), there's no way to distinguish v1 JSON from v2 JSON without inspecting the content. Migration of stored JSON becomes error-prone.
**Recommendation**: Add `config_version INTEGER NOT NULL DEFAULT 1` (or leverage the existing `version` column on `security_settings`). `security_settings` already has `version` -- confirm it tracks config schema version and not business version.

### MEDIUM -- Missing ON DELETE Action on Some FKs

**Table/File**: Various
**What**: Some FK relationships in GORM models specify `constraint:OnDelete:CASCADE` but the corresponding migration FK definitions are inconsistent:
- `client_permissions.client_api_id` FK to `client_apis` -- migration has CASCADE, correct
- `user_identities.identity_provider_id` -- nullable FK but model has no ON DELETE action specified
- Several `created_by`/`updated_by` FKs have `ON DELETE SET NULL` in migration but model has no constraint tag
**Why it's bad**: If the GORM model constraint and migration FK action disagree, GORM may try to auto-migrate and fail, or the actual DB behavior may differ from what the code expects.
**Recommendation**: Audit all FK constraints for consistency between migrations and GORM model tags. The migration is the source of truth; ensure model tags match.

### MINOR -- `email_change_otp` Stored as Plaintext

**Table/File**: `024_create_users_table.go:31`
**What**: `email_change_otp VARCHAR(10)` stores the OTP in plaintext. The `user_otps` table correctly stores `otp_hash` as a hash.
**Why it's bad**: Inconsistent security treatment. If the database is compromised, email change OTPs are readable. While OTPs are short-lived, the pattern should be consistent.
**Recommendation**: Store as `email_change_otp_hash TEXT` using the same hashing approach as `user_otps.otp_hash`.

### MINOR -- `user_backup_codes` Missing `is_used` Index

**Table/File**: `031_create_user_backup_codes_table.go`
**What**: `user_backup_codes` has `used BOOLEAN DEFAULT FALSE` and `used_at TIMESTAMPTZ` but no index on `(user_id, used)`.
**Why it's bad**: When verifying a backup code, the query `WHERE user_id = ? AND code_hash = ? AND used = false` benefits from an index. Without it, all backup codes for the user are scanned.
**Recommendation**: Add `CREATE INDEX idx_backup_codes_user_unused ON user_backup_codes (user_id, used) WHERE used = false`.

### MINOR -- Inconsistent `NOT NULL` on Description Columns

**Table/File**: Multiple
**What**: `services.description TEXT NOT NULL`, `apis.description TEXT NOT NULL`, `roles.description TEXT NOT NULL` but `policies.description TEXT` (nullable), `api_keys.description TEXT` (nullable). Some require a description, others don't.
**Why it's bad**: Inconsistent API contracts. Callers don't know which entities require descriptions.
**Recommendation**: Pick one convention. For admin-facing entities, `NOT NULL DEFAULT ''` is safer than nullable. Lower priority.

### MINOR -- `invites.invite_token` Stored as Plaintext

**Table/File**: `040_create_invites_table.go`
**What**: `invite_token TEXT NOT NULL UNIQUE` appears to store the token in cleartext rather than as a hash.
**Why it's bad**: If the database is compromised, invite tokens can be used to register on any tenant. Tokens should be hashed like `oauth_authorization_codes.code_hash`.
**Recommendation**: Store as `invite_token_hash TEXT NOT NULL UNIQUE` and only expose the raw token to the invited user via email.

---

## Recommendations

Ordered by priority.

1. **[CRITICAL]** Add `tenant_id` enforcement to `BaseRepository` methods -- every `FindByUUID`, `DeleteByUUID`, `UpdateByUUID` must require and filter by `tenant_id` to prevent cross-tenant access.

2. **[CRITICAL]** Remove or restrict unscoped `FindByEmail`/`FindByUsername`/`FindByPhone` from the user repository interface. Replace with tenant-scoped variants as the only public API.

3. **[MAJOR]** Add composite indexes on hot-path query columns: `users(email, tenant_id)`, `users(username, tenant_id)`, `users(phone, tenant_id)`, `user_identities(user_id, client_id)`, `user_identities(user_id, provider)`, `user_otps(channel, recipient, used, expires_at)`, `ip_restriction_rules(tenant_id, status)`.

4. **[MAJOR]** Optimize `FindBySubAndClientID` to reduce 7+ queries to 1-2 using JOINs or selective preloading. This is on the critical auth path.

5. **[MAJOR]** Replace boolean verification flags (`is_email_verified`, `is_phone_verified`) with nullable timestamps (`email_verified_at`, `phone_verified_at`) for SOC2 CC6.6 compliance.

6. **[MAJOR]** Add `last_login_at TIMESTAMPTZ` and `last_login_ip INET` to users table for SOC2 CC6.1 session tracking.

7. **[MAJOR]** Change `users.email` from `VARCHAR(255)` to `VARCHAR(254)` per RFC 5321.

8. **[MEDIUM]** Add `anonymized_at TIMESTAMPTZ NULL` to users and profiles for GDPR right-to-erasure support.

9. **[MEDIUM]** Add `config_version` / `schema_version` to tables with JSONB config blobs (security_settings, tenant_settings, email_config, sms_config).

10. **[MEDIUM]** Add Preloads to paginated list queries (users, webhook endpoints) or document which relationships are safe to access in handlers.

11. **[MEDIUM]** Audit FK constraint consistency between migration definitions and GORM model `constraint:` tags.

12. **[MINOR]** Hash `email_change_otp` and `invite_token` instead of storing plaintext.

13. **[MINOR]** Add UUID to `tenant_services` junction table for API consistency.

14. **[MINOR]** Add index on `user_backup_codes(user_id, used)` for faster backup code verification.

15. **[NICE-TO-HAVE]** Consider PostgreSQL ENUMs for high-traffic status columns to improve type safety and storage efficiency.

---

## Summary

The schema is **well above average** for a multi-tenant auth platform. The dual-identity PK+UUID pattern, partial unique indexes, JSONB+GIN indexing, immutable audit tables, complete MFA support, OAuth spec compliance, transactional outbox, and encrypted secret storage are all enterprise-grade patterns done correctly. The biggest risks are: **(1)** the unscoped `BaseRepository` methods that could allow cross-tenant data access, **(2)** missing composite indexes on login hot paths that will degrade at scale, and **(3)** the N+1 query fan-out on token exchange. A single PR addressing items 1-3 above would eliminate the critical security and performance risks. The remaining items are compliance hardening (SOC2 timestamps, GDPR anonymization) and consistency improvements.

# Audit Logging

> Two append-only, tenant-scoped trails: a structured **auth-event** security log (login/token/consent/session events, OWASP Logging Vocabulary) and a **management audit log** of control-plane config changes (who changed what, old→new).

| | |
|---|---|
| **Status** | Implemented |
| **Code** | `internal/authevent` (auth events), `internal/auditlog` (management audit log) |
| **Endpoints** | Control plane (`:8080`, VPN-only): `GET /api/v1/auth-events`, `/auth-events/count`, `/auth-events/export`, `/auth-events/{uuid}` (`auth_event:read`); `GET /api/v1/management-audit-log`, `/management-audit-log/{uuid}` (`audit:read`) |
| **Storage** | `auth_events` (migration 060, monthly RANGE-partitioned), `management_audit_log` (migration 057). Sibling `security_settings_audit` (migration 056) — see [security-settings](./security-settings.md) |
| **Config** | Per-tenant `tenant_settings.audit_config` JSON (`enabled`, `retention_days`, `pii_masking`, `log_level`, `event_types`). No dedicated env vars — retention interval/fallback are compile-time constants |

## Overview

"Audit logging" is two distinct, independently wired subsystems. Both are append-only, tenant-scoped, and readable only over the internal management plane (`:8080`, `internal/server/router.go:34` — VPN access only).

| Subsystem | Package | Table | Records | Written by |
|---|---|---|---|---|
| **Auth events** | `internal/authevent` | `auth_events` | Security events (authn/authz/session/user/system) following the OWASP Logging Vocabulary | Domain services call `AuthEventService.Log(ctx, input)` fire-and-forget |
| **Management audit log** | `internal/auditlog` | `management_audit_log` | Control-plane mutations (create/update/delete of resources), with old→new `changes` JSONB | REST handler call sites + a gRPC unary interceptor call `ManagementAuditLogger.Log(ctx, entry)` |

Both are complemented by (not replaced by) the OpenTelemetry pipeline: every `Log` also emits a span and a metric (`telemetry.RecordAuthEvent`, `telemetry.RecordAuditWriteFailure`) so operational dashboards see true rates even when a tenant's `audit_config` suppresses DB persistence.

> An earlier design/redesign proposal described the pre-060 `auth_logs` table and recommended the redesign. Where that proposal and the code disagree, the code below is authoritative — the redesign has shipped as `auth_events`, plus partitioning, per-tenant `audit_config`, PII masking, CSV/JSON export, and DB-level immutability triggers that the proposal did not specify.

## How it works

### Auth events (write path)

1. A domain service (login, token exchange, consent, session, MFA, IAM, user, client, SAML SLO, federation, …) constructs an `AuthEventInput` and calls `s.authEventService.Log(ctx, input)` (`internal/authevent/service_event.go:131`). The call is **fire-and-forget**: `Log` returns nothing and swallows errors so audit logging can never break a business flow (`service_event.go:60-63`).
2. `Log` opens an OTel span and **always** meters the event via `telemetry.RecordAuthEvent(ctx, category, event_type, result)` — regardless of whether it will be persisted (`service_event.go:132-142`).
3. It resolves the tenant's `auditConfig` (5s-cached read of `tenant_settings.audit_config`, `service_event.go:328-354`). If `enabled` is false, the `event_type` is not in the tenant's `event_types` allowlist (empty allowlist = allow all), or the event severity ranks below the tenant `log_level`, the event is **skipped** (metered but not stored) (`service_event.go:144-148`, `audit_config.go:135-164`).
4. The trace ID is extracted from the span context and stored on the row (`service_event.go:150-154`) so an admin can jump from a DB event to its full Datadog trace.
5. If `pii_masking` is on (default), `description`, `error_reason`, and `metadata` are redacted via `logging.RedactString` / `logging.RedactJSON` before insert (`service_event.go:159-163`).
6. The row is persisted synchronously and durably via `authEventRepo.Create` (`service_event.go:181-185`). Auth-event logging is **not** async and **not** fanned out to webhooks — they are not integration events (`service_event.go:78-82`).

### Auth events (read path)

- `GET /auth-events` → `AuthEventHandler.GetAll` → `FindPaginated`. Tenant is taken from the request auth context and is **mandatory** — the repository rejects any query without a `tenant_id` (`repository_event.go:92-94`), so events are strictly tenant-isolated. Filters: `category`, `event_type` (prefix `ILIKE`, `repository_event.go:117-123`), `ip_address` (INET `host()` prefix match, `:124-127`), `severity`, `result`, `user` (UUID → actor-or-target), `date_from`/`date_to`. Pagination is keyset (`database.PaginateKeyset`) with an exact tenant-scoped `COUNT` computed before the cursor is applied (`:148-160`).
- `GET /auth-events/{auth_event_uuid}` → `FindByUUID` (tenant-scoped).
- `GET /auth-events/count?event_type=…` → `CountByEventType`.
- `GET /auth-events/export?format=csv|json` → `Export` (default JSON), capped at `maxAuthEventExportLimit = 10000` rows (`audit_config.go:15`, `service_event.go:300-322`). **Exporting the trail is itself logged** as a `system_audit_export` event with the actor/IP (`handler_event.go:94-110`) — PCI 10.2.3 (access to the audit trail).

### Auth events (retention & partitioning)

- `auth_events` is **RANGE-partitioned by `created_at`** into monthly child tables `auth_events_yYYYYmMM` (migration `060_create_auth_events_table.go:54`). A shared sequence backs `auth_event_id` across partitions; uniqueness is on `(auth_event_uuid, created_at)` since the partition key must be part of any unique constraint.
- `StartPartitionManager` runs daily and pre-creates next month's partition so writes never hit a missing partition (`partition.go:41-61`, wired at `cmd/server/workers.go:46`).
- `StartRetentionRunner` runs every `DefaultRetentionInterval` (24h) (`service_retention.go`, wired at `workers.go:33`). Because `authEventService` implements `DeleteExpiredByAuditConfig`, the runner deletes rows per **each tenant's** `audit_config.retention_days` (falling back to `defaultAuditRetention = 90` days when missing/invalid) via a single SQL statement (`repository_event.go:214-244`, `service_retention.go:77-86`). It then drops whole partitions older than the runner's fixed window (`DefaultRetentionPeriod = 365` days) via `DropExpiredPartitions` (`service_retention.go:66-72`, `partition.go:65-98`).

> Note the two retention horizons: **row deletes** honor the per-tenant `retention_days` (default 90d); **partition drops** use the fixed 365-day `DefaultRetentionPeriod`. A partition is only dropped once its entire month is beyond 365 days.

### Management audit log (write path)

1. **REST call sites**: control-plane handlers (IAM policy/role/permission/api/service, client, tenant setting, federation, …) call `AuditLogger.Log(ctx, LogEntry{...})` after a mutation, supplying `Action`, `ResourceType`, `ResourceID`/`ResourceUUID`, and a `Changes` JSON diff (`internal/auditlog/service_management_audit_log.go:47`).
2. **gRPC interceptor**: `grpcAuditUnaryInterceptor` writes one row for **every mutating** RPC (Create/Update/Delete/Set/Assign/Remove/Rotate/Register), mapped from the method name (`internal/server/grpc_audit_interceptor.go:73-106`). Reads (Get/List/Introspect/Authorize) are not audited. This is the floor for the control plane so no handler can forget to log; it records who/what/outcome but not the resource UUID (that lives in the request body).
3. `Log` populates `ip_address`/`user_agent` from context, `trace_id` from the OTel span, and `request_id` from the security middleware, defaults `outcome` to `success`, and validates `Changes` is valid JSON (defaulting to `{}`) (`service_management_audit_log.go:51-89`).
4. The write is **best-effort**: on failure it emits `telemetry.RecordAuditWriteFailure`, logs the gap, and returns the error to the caller (which generally ignores it, since the business action already succeeded) (`service_management_audit_log.go:90-105`).

### Management audit log (read path)

- `GET /management-audit-log` → `List`. Tenant-scoped, filters `resource_type`, `action`, `outcome`, `actor_user_id`; offset pagination (default 20, `limit` allowlist-sorted to prevent ORDER BY injection, `repository_management_audit_log.go:126-141`). Read queries left-join `users`/`profiles`/`clients` to surface human-readable `actor_user_name` / `actor_client_name` (`repository_management_audit_log.go:33-63`).
- `GET /management-audit-log/{audit_log_uuid}` → `Get` (tenant-scoped).

## Implementation

### `internal/authevent`

| File | Role |
|---|---|
| `model_event.go` | `AuthEvent` GORM model → `auth_events`. `ip_address` is `INET NOT NULL`; `created_at` is part of the composite PK (partition key). `BeforeCreate` mints a UUID (`model_event.go:40-45`) |
| `service_event.go` | `AuthEventService` interface + impl: `Log`, `FindPaginated`, `FindByUUID`, `CountByEventType`, `DeleteOlderThan`, `DeleteExpiredByAuditConfig`, `Export`, `Shutdown` (no-op). `NoopService()` for tests |
| `service_event_constants.go` | Category (`AUTHN/AUTHZ/SESSION/USER/SYSTEM`), severity (`INFO/WARN/CRITICAL`), result (`success/failure`), and ~45 `event_type` string constants (login, token, MFA, OAuth authorize/consent/token, session, user, IAM policy, `system_audit_export`, `sys_*`, …) |
| `audit_config.go` | Per-tenant `auditConfig` parse/apply: `allowsEvent`, `allowsSeverity`, `masksPII`, `severityRank`. Constants: `defaultAuditRetention=90`, `legacyAuditRetention=365`, `auditConfigCacheTTL=5s`, `maxAuthEventExportLimit=10000` |
| `repository_event.go` | `AuthEventRepository`. **Update/CreateOrUpdate/DeleteByID/DeleteByUUID all return errors** — append-only enforced in code (`:70-88`). `DeleteOlderThan` / `DeleteExpiredByAuditConfig` set the `maintainerd.allow_auth_event_delete='retention'` GUC inside the txn so the DB trigger permits the delete (`:246-248`) |
| `handler_event.go` | `AuthEventHandler`: `GetAll`, `Export`, `Get`, `CountByType`; request→filter mapping + validation |
| `routes.go` | `/auth-events` router: `JWTAuthMiddleware` + `UserContextMiddleware` + optional rate-limit; every route gated by `auth_event:read` |
| `partition.go` | `EnsureNextPartition`, `StartPartitionManager`, `DropExpiredPartitions`, monthly partition naming |
| `service_retention.go` | `StartRetentionRunner` ticker; prefers per-tenant `DeleteExpiredByAuditConfig`, else `DeleteOlderThan(cutoff)` |
| `validation_event.go` / `types.go` | Filter validation + `AuthEventFilterDTO` / `AuthEventResponseDTO` (response `auth_event_id` = the UUID string) |

### `internal/auditlog`

| File | Role |
|---|---|
| `model_management_audit_log.go` | `ManagementAuditLog` GORM model → `management_audit_log`; read-only presentation fields `actor_user_name`/`actor_client_name` (`->`) |
| `service_management_audit_log.go` | `ManagementAuditLogger.Log(ctx, LogEntry)` — best-effort write with trace/request-id enrichment and JSON validation |
| `repository_management_audit_log.go` | `Create`, `FindPaginated` (offset), `FindByUUIDAndTenantID`; actor-label joins; sort allowlist |
| `handler_management_audit_log.go` | `List`, `Get` |
| `routes.go` | `/management-audit-log` router gated by `audit:read` |

### Cross-cutting

- **gRPC**: `internal/server/grpc_audit_interceptor.go` — mutating-RPC floor into `management_audit_log`.
- **Wiring**: `AuthEventService` is constructed in `internal/app/services.go:209` with the tenant-settings reader and injected into every logging domain service; retention/partition workers start in `cmd/server/workers.go`.
- **Permissions**: seeded in `internal/setup/seeder/004_permission.go:301` (`auth_event:read` — "Read and export auth events") and `:304` (`audit:read`).

### Migrations / DB tables

| Table | Migration | Notes |
|---|---|---|
| `auth_events` | `060_create_auth_events_table.go` | Partitioned parent; `CHECK` constraints on category/severity/result; FKs to `tenants` (CASCADE), `users` actor/target (SET NULL); partial indexes on `result='failure'` and `severity IN ('WARN','CRITICAL')`; **immutability trigger** `protect_auth_events_immutable` blocks all UPDATE and blocks DELETE unless the `maintainerd.allow_auth_event_delete` GUC is `retention` or `tenant_delete` |
| `management_audit_log` | `057_create_management_audit_log_table.go` | `BIGSERIAL` PK; FKs to `tenants`/`users`/`clients`; `outcome CHECK IN ('success','failure','partial')`; GIN index on `changes`; **immutability trigger** `prevent_management_audit_log_mutation` (same GUC scheme via `maintainerd.allow_management_audit_log_delete`) |
| `security_settings_audit` | `056_create_security_settings_audit_table.go` | Separate config-change trail owned by security-settings — see [security-settings](./security-settings.md) |

## Configuration

There are **no dedicated env vars** for audit logging. Behavior is governed per tenant by the `audit_config` object inside `tenant_settings` (read via `AuditConfigReader`, `audit_config.go:20-22`):

| Key | Type | Default (tenant) | Effect |
|---|---|---|---|
| `enabled` | bool | `true` | When false, auth events are metered but **not** persisted |
| `retention_days` | int | `90` (`defaultAuditRetention`) | Row-level retention used by `DeleteExpiredByAuditConfig`; invalid/missing → default |
| `pii_masking` | bool | `true` | Redacts `description`, `error_reason`, `metadata` before insert |
| `log_level` | string | `info` | Minimum severity persisted (`debug<info<warn<critical`, `severityRank`) |
| `event_types` | string[] | `[]` | Allowlist of `event_type`s; empty = allow all |

When no reader is wired or `tenant_id == 0`, a `legacyAuditConfig()` fallback applies: enabled, `retention_days=365`, PII masking on, `info` level (`audit_config.go:37-45`). Compile-time constants (not env-configurable): `DefaultRetentionPeriod=365d`, `DefaultRetentionInterval=24h` (`service_retention.go:12-18`), partition manager interval 24h (`workers.go:46`), `auditConfigCacheTTL=5s`, `maxAuthEventExportLimit=10000`.

## Security considerations

- **Append-only, enforced twice.** Immutability is enforced in Go (repository Update/Delete methods return errors) **and** at the database via `BEFORE UPDATE OR DELETE` triggers. UPDATE is always rejected; DELETE only succeeds when a transaction-local GUC marks the operation as sanctioned `retention` or `tenant_delete` (`060_...go:143-163`, `057_...go:34-58`). A stray application delete cannot silently rewrite history.
- **Strict tenant isolation.** Auth-event listing hard-fails without a `tenant_id` (`repository_event.go:92-94`); the exact `COUNT` is tenant-scoped to prevent cross-tenant row-count disclosure. Reads are gated by `auth_event:read` / `audit:read` on the VPN-only `:8080` plane, behind `JWTAuthMiddleware` + management-client audience guard.
- **PII minimization.** Default `pii_masking=true` redacts free-text and metadata; `log_level`/`event_types` let a tenant narrow what is retained; per-tenant `retention_days` bounds storage (GDPR storage limitation).
- **Access to the trail is itself audited.** Bulk export writes a `system_audit_export` auth event with actor + IP (`handler_event.go:94-110`).
- **Tamper-evident correlation.** Every row carries `trace_id` (and `request_id` for management logs), linking the DB record to the distributed trace.
- **Logging never breaks business flows.** Auth-event `Log` is fire-and-forget and error-swallowing; management-audit `Log` is best-effort and surfaces gaps via the `RecordAuditWriteFailure` metric rather than failing the request.
- **gRPC control plane cannot skip auditing.** The unary interceptor logs every mutating RPC by construction, closing the gap where gRPC handlers previously wrote nothing to `management_audit_log`.

## Related

- [authentication](./authentication.md) — login/token events feed `auth_events`
- [sessions](./sessions.md) — session lifecycle events
- [multi-factor-auth](./multi-factor-auth.md) — MFA enrollment/trusted-device events
- [security-settings](./security-settings.md) — the sibling `security_settings_audit` config-change trail
- [multi-tenancy](./multi-tenancy.md) — `tenant_settings.audit_config` and tenant-scoped isolation
- [events](./events.md) — integration/webhook events (deliberately distinct from audit events)

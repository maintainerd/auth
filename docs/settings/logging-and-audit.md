# Logging & Audit Architecture

## Overview

This document defines the logging and audit strategy for the maintainerd-auth service. It covers the three existing logging-related tables (`auth_logs`, `service_logs`, `security_settings_audit`), evaluates each against industry standards, and provides a concrete redesign plan grounded in OWASP, NIST, PCI DSS, and IETF specifications.

The core principle: **use the right tool for each logging concern**. Operational logs belong in a streaming pipeline (OpenTelemetry → Datadog). Security events belong in a queryable database table for compliance, admin investigation, and retention control. Configuration change diffs belong in dedicated audit tables.

---

## Standards & References

Every recommendation in this document traces to at least one of the following standards. Inline citations use the short name in brackets (e.g., `[OWASP-LOG]`).

| Short Name | Full Title | Key Section |
|---|---|---|
| **[OWASP-LOG]** | [OWASP Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html) | Which events to log, event attributes, data to exclude, storage, protection, retention |
| **[OWASP-VOCAB]** | [OWASP Logging Vocabulary Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Vocabulary_Cheat_Sheet.html) | Standardized event type names: `AUTHN`, `AUTHZ`, `SESSION`, `USER`, `PRIVILEGE`, etc. |
| **[OWASP-AUTHN]** | [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html) | §Logging and Monitoring: log all auth failures, password failures, account lockouts |
| **[OWASP-ASVS]** | [OWASP ASVS v4 §V7](https://owasp.org/www-project-application-security-verification-standard/) | Error Handling and Logging verification requirements |
| **[PCI-DSS]** | PCI DSS v4.0 Requirement 10 | Log and Monitor All Access to System Components and Cardholder Data |
| **[NIST-63B]** | NIST SP 800-63B §5.2.2 | Verifiers SHALL log authentication events including failures |
| **[NIST-92]** | [NIST SP 800-92](https://csrc.nist.gov/publications/detail/sp/800-92/final) — Guide to Computer Security Log Management | Centralized log management for operational logs; queryable store for security events |
| **[RFC-5424]** | [IETF RFC 5424](https://datatracker.ietf.org/doc/html/rfc5424) — The Syslog Protocol | Standard log format, severity levels (Emergency through Debug) |
| **[GDPR-5]** | GDPR Article 5(1)(e) | Storage limitation — personal data kept no longer than necessary |
| **[SOC2-CC6]** | SOC 2 Type II — CC6.1 | Logical and Physical Access Controls — audit trail requirements |

---

## Current State

The codebase contains three logging-related tables in varying states of completeness:

| Table | Migration | Model | Repository | Service | Handler | Wired | Status |
|---|:-:|:-:|:-:|:-:|:-:|:-:|---|
| `auth_events` | ✅ 044 | ✅ | ✅ (full) | ✅ | ✅ (read-only) | ✅ | **Active** — end-to-end integrated with OWASP Vocabulary |
| `auth_logs` | ❌ Dropped (045) | ❌ Removed | ❌ Removed | — | — | — | **Removed** — superseded by `auth_events` |
| `service_logs` | ❌ Dropped (045) | ❌ Removed | — | — | — | — | **Removed** — operational logs use OTel pipeline |
| `security_settings_audit` | ✅ 039 | ✅ | ✅ | ✅ | ✅ | ✅ | **Fully active** — end-to-end integrated |

In addition, the service has comprehensive OpenTelemetry instrumentation:

| Signal | Library | Scope |
|---|---|---|
| Inbound HTTP (REST) | `otelhttp` | Every request on admin `:8080` and public `:8081` |
| Inbound gRPC | `otelgrpc` | Every RPC |
| PostgreSQL | `otelgorm` | Every SQL query and transaction |
| Redis | `redisotel` | Every Redis command |
| SMTP | Manual spans | Every email send operation |
| Structured Logs | zerolog | `trace_id` / `span_id` injected via `LoggingMiddleware` |

---

## Three-Layer Logging Architecture

The recommended architecture uses three complementary layers, each serving a distinct purpose:

```
┌─────────────────────────────────────────────────────────────────────┐
│                 Three-Layer Logging Architecture                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Layer 1: Structured Logs (operational)                             │
│  ─────────────────────────────────────                              │
│  zerolog → stdout → OTel Collector → Datadog                       │
│  • HTTP/gRPC/SQL/Redis/SMTP traces                                 │
│  • Debug/info/warning/error messages                                │
│  • Performance metrics and latency                                  │
│  • Real-time alerting and dashboards                                │
│                                                                     │
│  Layer 2: Auth Events (security & compliance)                       │
│  ────────────────────────────────────────────                       │
│  Service code → PostgreSQL `auth_events` table                     │
│  • Authentication success/failure                                   │
│  • Authorization violations                                         │
│  • Session lifecycle                                                │
│  • User management operations                                       │
│  • Admin-queryable via REST API                                     │
│  • Retention-managed per compliance requirements                    │
│                                                                     │
│  Layer 3: Config Audits (change tracking)                           │
│  ────────────────────────────────────────                           │
│  Service code → PostgreSQL `*_audit` tables                        │
│  • Old value → new value JSONB diffs                                │
│  • Who changed it, when, from where                                 │
│  • Forensic investigation ("who turned off MFA?")                   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Why These Layers Are Complementary

| Concern | Layer 1: OTel → Datadog | Layer 2: DB `auth_events` | Layer 3: DB `*_audit` |
|---|:-:|:-:|:-:|
| Real-time alerting | ✅ | — | — |
| Distributed tracing | ✅ | via `trace_id` column | — |
| Admin dashboard / investigation | — | ✅ | ✅ |
| Compliance queryable store (who did what when?) | Awkward | ✅ | ✅ |
| Retention you control per regulation | Vendor-dependent | ✅ | ✅ |
| Feeds lockout / IP restriction decisions | Expensive API call | ✅ Fast local query | — |
| Config change forensics (old → new diff) | — | — | ✅ |

**Standards basis for the separation:**
- `[NIST-92] §3.1` — "Organizations should use centralized log management infrastructure" (= Layer 1)
- `[PCI-DSS] 10.2` — "Record audit trails for all individual user accesses" (= Layer 2)
- `[PCI-DSS] 10.2.7` — "Creation and deletion of system-level objects" (= Layer 3)
- `[OWASP-AUTHN]` — "Ensure all failures are logged and reviewed" (= Layer 2, queryable)

---

## Table 1: `auth_logs` — Assessment and Redesign

### Assessment: What's Wrong

The current `auth_logs` table has structural problems that prevent it from meeting compliance requirements:

#### 1. Too Few Event Types

The migration comment defines only four event types:

```
'login', 'logout', 'token_refresh', 'password_reset'
```

Per `[OWASP-VOCAB]`, an authentication service should capture events across **six categories** with **23+ defined event types**:

| Category | OWASP Vocabulary Events | Currently Captured |
|---|---|:-:|
| **Authentication [AUTHN]** | `authn_login_success`, `authn_login_fail`, `authn_login_fail_max`, `authn_login_lock`, `authn_login_successafterfail`, `authn_password_change`, `authn_password_change_fail`, `authn_token_created`, `authn_token_revoked`, `authn_token_reuse`, `authn_token_delete`, `authn_impossible_travel` | 4 of 12 |
| **Authorization [AUTHZ]** | `authz_fail`, `authz_change`, `authz_admin` | 0 of 3 |
| **Session [SESSION]** | `session_created`, `session_renewed`, `session_expired`, `session_use_after_expire` | 0 of 4 |
| **User Management [USER]** | `user_created`, `user_updated`, `user_archived`, `user_deleted` | 0 of 4 |
| **Privilege [PRIVILEGE]** | `privilege_permissions_changed` | 0 of 1 |
| **System [SYS]** | `sys_startup`, `sys_shutdown`, `sys_crash` | 0 of 3 |

#### 2. Missing Required Attributes

Per `[OWASP-LOG]` and `[PCI-DSS] 10.3`, every security event must include the "when, where, who, what" attributes. The current schema is missing critical columns:

| Required Attribute | Standard | Current Schema | Status |
|---|---|---|:-:|
| Result (success/failure) | `[PCI-DSS] 10.2` | Not captured | ❌ Missing |
| Severity (INFO/WARN/CRITICAL) | `[OWASP-VOCAB]` | Not captured | ❌ Missing |
| Event category (AUTHN/AUTHZ/SESSION/USER) | `[OWASP-VOCAB]` | Not captured | ❌ Missing |
| Target user (who was acted upon) | `[OWASP-LOG]` | Not captured | ❌ Missing |
| Error reason | `[OWASP-AUTHN]` | Not captured (free-text `description` only) | ❌ Missing |
| Trace ID (OTel correlation) | `[NIST-92]` | Not captured | ❌ Missing |
| Timestamp (ISO 8601 with UTC offset) | `[OWASP-VOCAB]` | `created_at TIMESTAMPTZ` | ✅ Present |
| IP address | `[OWASP-LOG]` | `ip_address VARCHAR(100)` | ✅ Present |
| User agent | `[OWASP-LOG]` | `user_agent TEXT` | ✅ Present |
| User identity | `[OWASP-LOG]` | `user_id INTEGER` | ✅ Present |
| Metadata (extensible) | `[OWASP-LOG]` | `metadata JSONB` | ✅ Present |

#### 3. Schema Issues

- **`updated_at` column** — Security events should be immutable. `[OWASP-LOG]` recommends "tamper detection" and "read-only copies." An `updated_at` column implies mutability.
- **`user_id NOT NULL`** — Failed login attempts by unknown users cannot be recorded because `user_id` is required. `[OWASP-AUTHN]` requires logging attempts with unknown identifiers.
- **`ip_address VARCHAR(100)` nullable** — IP address is a critical forensic attribute and should not be nullable. `VARCHAR(45)` covers IPv6 (`xxxx:xxxx:xxxx:xxxx:xxxx:xxxx:xxxx:xxxx` = 39 chars).
- **No partial indexes on failure/severity** — Most compliance queries are "show me failures" and "show me critical events." Without partial indexes, these queries scan the entire table.

#### 4. Wiring Gap

Despite having a complete model and a well-implemented repository (pagination, date range, retention, event type counting, SQL-injection-safe sort), the table is **never used**:
- No `AuthLogService` exists
- No REST handler exposes the data
- `NewAuthLogRepository()` is never called in `app/repositories.go`
- Permissions (`auth_log:read`, `auth_log:create`, `auth_log:update`, `auth_log:delete`) are seeded but never checked

### Recommendation: Redesign as `auth_events`

Rename the table to `auth_events` (industry standard naming — this is an event log, not a debug log) and redesign the schema to meet the standards gaps identified above.

#### Proposed Schema

```sql
CREATE TABLE IF NOT EXISTS auth_events (
    auth_event_id     BIGSERIAL     PRIMARY KEY,
    auth_event_uuid   UUID          NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    tenant_id         BIGINT        NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,

    -- WHO
    actor_user_id     BIGINT        REFERENCES users(user_id) ON DELETE SET NULL,
    target_user_id    BIGINT        REFERENCES users(user_id) ON DELETE SET NULL,
    ip_address        VARCHAR(45)   NOT NULL,
    user_agent        TEXT,

    -- WHAT  (aligns with OWASP Logging Vocabulary)
    category          VARCHAR(20)   NOT NULL,
    event_type        VARCHAR(60)   NOT NULL,
    severity          VARCHAR(10)   NOT NULL DEFAULT 'INFO',
    result            VARCHAR(10)   NOT NULL,
    description       TEXT,
    error_reason      VARCHAR(255),

    -- CONTEXT
    trace_id          VARCHAR(32),
    metadata          JSONB         DEFAULT '{}',

    -- WHEN  (immutable — no updated_at)
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    -- CONSTRAINTS
    CONSTRAINT chk_auth_events_category CHECK (category IN (
        'AUTHN', 'AUTHZ', 'SESSION', 'USER', 'SYSTEM'
    )),
    CONSTRAINT chk_auth_events_severity CHECK (severity IN (
        'INFO', 'WARN', 'CRITICAL'
    )),
    CONSTRAINT chk_auth_events_result CHECK (result IN (
        'success', 'failure'
    ))
);

-- Primary query patterns
CREATE INDEX idx_auth_events_tenant_created   ON auth_events (tenant_id, created_at DESC);
CREATE INDEX idx_auth_events_actor            ON auth_events (actor_user_id, created_at DESC);
CREATE INDEX idx_auth_events_event_type       ON auth_events (event_type, created_at DESC);

-- Compliance-focused partial indexes  (PCI DSS 10.2: "invalid logical access attempts")
CREATE INDEX idx_auth_events_failures         ON auth_events (result, created_at DESC)
    WHERE result = 'failure';
CREATE INDEX idx_auth_events_critical         ON auth_events (severity, created_at DESC)
    WHERE severity IN ('WARN', 'CRITICAL');
```

#### Key Design Decisions

| Decision | Rationale | Standard |
|---|---|---|
| **No `updated_at`** | Events are append-only / immutable. Prevents tampering. | `[OWASP-LOG]` — tamper protection |
| **`actor_user_id` nullable** | Failed logins by unknown users must be recorded. | `[OWASP-AUTHN]` — log all auth failures |
| **`actor_user_id` + `target_user_id`** | Separates "who performed the action" from "who was affected" (e.g., admin disabling another user). | `[OWASP-VOCAB]` USER events: `user_updated:joebob1,user1` |
| **`category` column** | Aligns with OWASP Vocabulary top-level namespaces. Enables category-level filtering and dashboards. | `[OWASP-VOCAB]` |
| **`event_type` column** | Uses the exact OWASP Vocabulary names (e.g., `authn_login_success`). Enables cross-application monitoring with standard keywords. | `[OWASP-VOCAB]` |
| **`severity` column** | Each OWASP Vocabulary event type has a defined severity (INFO, WARN, CRITICAL). | `[OWASP-VOCAB]`, `[RFC-5424]` |
| **`result` column** | `[PCI-DSS] 10.2` requires distinguishing success from failure in audit trails. | `[PCI-DSS] 10.2` |
| **`error_reason` column** | Structured failure reason (e.g., `bad_password`, `account_locked`, `mfa_fail`). Avoids parsing free-text descriptions. | `[OWASP-AUTHN]` |
| **`trace_id` column** | Bridges the DB event to the corresponding OTel trace in Datadog. Allows jumping from admin dashboard to full distributed trace. | `[NIST-92]` — correlation between log sources |
| **Partial indexes on `failure` and `WARN/CRITICAL`** | The most common compliance queries are "show me failures" and "show me critical events." Partial indexes keep these fast as the table grows. | Performance best practice |
| **`BIGSERIAL` / `BIGINT`** | Auth event tables grow indefinitely, they will exceed `INTEGER` range in high-traffic deployments. | Scalability |

#### OWASP Logging Vocabulary — Complete Event Type Mapping

These are the event types that should be recorded in the `auth_events` table, organized by category. The event type names follow the `[OWASP-VOCAB]` standard exactly.

##### Authentication [AUTHN]

| Event Type | When to Record | Severity | Result |
|---|---|:-:|:-:|
| `authn_login_success` | User successfully authenticates | INFO | success |
| `authn_login_fail` | Authentication attempt fails (bad credentials) | WARN | failure |
| `authn_login_fail_max` | User reaches max failed login attempts | WARN | failure |
| `authn_login_lock` | Account is locked due to policy (maxretries, suspicious, admin) | WARN | failure |
| `authn_login_successafterfail` | Successful login after previous failures (include retry count in metadata) | INFO | success |
| `authn_password_change` | User successfully changes their password | INFO | success |
| `authn_password_change_fail` | Password change attempt fails | CRITICAL | failure |
| `authn_token_created` | Access/refresh token pair is issued | INFO | success |
| `authn_token_revoked` | Token is explicitly revoked (logout, admin action) | INFO | success |
| `authn_token_reuse` | A previously revoked token is presented | CRITICAL | failure |
| `authn_token_delete` | API key or long-lived token is deleted | WARN | success |
| `authn_impossible_travel` | Login from geographically impossible location vs. last known | CRITICAL | failure |

##### Authorization [AUTHZ]

| Event Type | When to Record | Severity | Result |
|---|---|:-:|:-:|
| `authz_fail` | Access attempt to a resource the user is not authorized for | CRITICAL | failure |
| `authz_change` | User's role or permissions are changed | WARN | success |
| `authz_admin` | Any action performed by a privileged/admin user | WARN | success |

##### Session Management [SESSION]

| Event Type | When to Record | Severity | Result |
|---|---|:-:|:-:|
| `session_created` | New authenticated session is started | INFO | success |
| `session_renewed` | Session is extended / token is refreshed | INFO | success |
| `session_expired` | Session expires (timeout, logout, admin revocation). Store reason in `error_reason`. | INFO | success |
| `session_use_after_expire` | Attempt to use an expired session — potential session hijack | CRITICAL | failure |

##### User Management [USER]

| Event Type | When to Record | Severity | Result |
|---|---|:-:|:-:|
| `user_created` | New user account is registered or created by admin | WARN | success |
| `user_updated` | User profile or attributes are modified | WARN | success |
| `user_archived` | User account is soft-deleted / archived | WARN | success |
| `user_deleted` | User account is permanently deleted | WARN | success |

##### Privilege Changes [PRIVILEGE]

| Event Type | When to Record | Severity | Result |
|---|---|:-:|:-:|
| `privilege_permissions_changed` | Object-level or resource-level permission is changed | WARN | success |

##### System [SYSTEM]

| Event Type | When to Record | Severity | Result |
|---|---|:-:|:-:|
| `sys_startup` | Service instance starts | WARN | success |
| `sys_shutdown` | Service instance performs graceful shutdown | WARN | success |
| `sys_crash` | Unrecoverable error — store reason in `error_reason` | CRITICAL | failure |

#### Data Exclusions

Per `[OWASP-LOG]` — Data to Exclude, the following must **never** appear in any column (including `metadata` JSONB):

| Excluded Data | Reason |
|---|---|
| Session tokens / access tokens / refresh tokens | Credential theft via log access |
| Passwords (even hashed) | Offline brute-force via log access |
| Encryption keys, signing keys | Full system compromise via log access |
| Full credit card / bank account numbers | PCI DSS scope expansion |
| PII beyond what's strictly needed | `[GDPR-5]` storage limitation. Hash or mask emails if logged. |
| Database connection strings | Infrastructure compromise |

#### Implementation Layers

The `auth_events` table requires the full layer stack:

```
┌──────────────────────────────────────────────────────────────────┐
│  Handler Layer (REST API for admin queries)                      │
│  GET /auth-events          — paginated list with filters         │
│  GET /auth-events/:uuid    — single event detail                 │
├──────────────────────────────────────────────────────────────────┤
│  Service Layer (AuthEventService)                                │
│  Log(ctx, event)           — called by other services            │
│  FindPaginated(filter)     — admin query                         │
│  FindByUUID(uuid, tid)     — admin detail view                   │
│  CountByEventType(...)     — dashboard aggregations              │
│  DeleteOlderThan(cutoff)   — retention management                │
├──────────────────────────────────────────────────────────────────┤
│  Repository Layer (AuthEventRepository)                          │
│  (largely exists — refactor current AuthLogRepository)           │
├──────────────────────────────────────────────────────────────────┤
│  Database (PostgreSQL `auth_events` table)                       │
└──────────────────────────────────────────────────────────────────┘
```

**Integration points** — Other services call `AuthEventService.Log(ctx, event)`:

| Calling Service | Events Produced |
|---|---|
| LoginService | `authn_login_success`, `authn_login_fail`, `authn_login_fail_max`, `authn_login_lock`, `authn_login_successafterfail` |
| TokenService | `authn_token_created`, `authn_token_revoked`, `authn_token_reuse`, `authn_token_delete` |
| PasswordService | `authn_password_change`, `authn_password_change_fail` |
| SessionService | `session_created`, `session_renewed`, `session_expired`, `session_use_after_expire` |
| UserService | `user_created`, `user_updated`, `user_archived`, `user_deleted` |
| RoleService / PermissionService | `authz_change`, `privilege_permissions_changed` |
| Authorization Middleware | `authz_fail`, `authz_admin` |
| App Lifecycle (main.go) | `sys_startup`, `sys_shutdown`, `sys_crash` |

#### Retention Policy

Per `[GDPR-5]` and `[PCI-DSS] 10.7`:

- **Minimum retention**: 1 year (PCI DSS 10.7.1 requires at least 12 months of audit trail history)
- **Immediately available**: Last 3 months must be immediately available for analysis (PCI DSS 10.7.1)
- **Maximum retention**: Per your organization's data retention policy — do not keep beyond what's legally required
- **Implementation**: The existing `DeleteOlderThan(cutoff)` repository method is correct; wire it to a configurable retention period and a background job (cron or ticker)

---

## Table 2: `service_logs` — Assessment and Verdict

### Assessment

The `service_logs` table stores operational log messages (`INFO`, `WARN`, `ERROR`, `DEBUG`) in PostgreSQL:

```sql
-- Current schema (migration 012)
CREATE TABLE IF NOT EXISTS service_logs (
    service_log_id   SERIAL PRIMARY KEY,
    service_log_uuid UUID NOT NULL UNIQUE,
    service_id       INTEGER NOT NULL REFERENCES services(service_id) ON DELETE CASCADE,
    level            VARCHAR(20) NOT NULL CHECK (level IN ('INFO', 'WARN', 'ERROR', 'DEBUG')),
    message          TEXT NOT NULL,
    metadata         JSONB,
    created_at       TIMESTAMPTZ DEFAULT now()
);
```

### Verdict: Remove

Writing operational log messages to a relational database is an anti-pattern for three reasons:

| Problem | Explanation | Standard |
|---|---|---|
| **High volume** | Every request generates multiple log lines. This will bloat the database, degrade query performance, and inflate backup sizes. | — |
| **Wrong query tool** | Debugging operational issues requires full-text search, log correlation, and time-series visualization. SQL is the wrong interface — Datadog/Grafana are purpose-built for this. | `[NIST-92] §3.1` |
| **Already covered** | The OTel stack (`otelhttp`, `otelgrpc`, `otelgorm`, `redisotel`, manual SMTP spans) with `trace_id`/`span_id` injection already captures everything `service_logs` was intended to record, with better tooling. | `[NIST-92] §3.1` |

Per `[NIST-92] §3.1`: *"Organizations should use centralized log management infrastructure"* — that centralized infrastructure is your OTel → Datadog pipeline, not a custom database table.

**Action**: Remove the migration and model. No service, handler, or repository was ever built, so the blast radius is minimal.

---

## Table 3: `security_settings_audit` — Assessment and Verdict

### Assessment

The `security_settings_audit` table is **fully active** and well-designed:

```sql
-- Current schema (migration 039)
CREATE TABLE IF NOT EXISTS security_settings_audit (
    security_settings_audit_id   SERIAL PRIMARY KEY,
    security_settings_audit_uuid UUID NOT NULL UNIQUE,
    user_pool_id                 INTEGER NOT NULL REFERENCES user_pools(user_pool_id) ON DELETE CASCADE,
    security_setting_id          INTEGER NOT NULL REFERENCES security_settings(security_setting_id) ON DELETE CASCADE,
    change_type                  VARCHAR(50) NOT NULL,
    old_config                   JSONB,
    new_config                   JSONB,
    ip_address                   VARCHAR(50),
    user_agent                   TEXT,
    created_by                   INTEGER REFERENCES users(user_id) ON DELETE SET NULL,
    created_at                   TIMESTAMPTZ DEFAULT now(),
    updated_at                   TIMESTAMPTZ DEFAULT now()
);
```

**Integration**: Every `UpdateXConfig()` method in `SecuritySettingService` creates an audit row via the `updateConfig()` helper within a transaction. The audit record captures:
- **What changed** (`change_type`: `update_password_config`, `update_mfa_config`, etc.)
- **Before and after** (`old_config`, `new_config` as JSONB)
- **Who changed it** (`created_by` → user, `ip_address`, `user_agent`)
- **When** (`created_at`)

This pattern satisfies:
- `[PCI-DSS] 10.2.7` — "Creation and deletion of system-level objects"
- `[SOC2-CC6]` — Audit trail for access control configuration changes
- `[OWASP-LOG]` — "Higher-risk functionality: admin-level privilege use"

### Verdict: Keep As-Is, Expand the Pattern

The `security_settings_audit` table demonstrates the correct pattern for configuration change auditing. Consider applying the same pattern to other configuration entities that currently lack audit trails:

| Entity | Has Audit Trail | Security Relevance | Recommendation |
|---|:-:|---|---|
| Security Settings | ✅ | High — auth/session/lockout policies | Keep |
| Email Config | ❌ | High — SMTP credentials, sender address | Add audit table |
| SMS Config | ❌ | High — SMS provider credentials | Add audit table |
| Webhook Endpoints | ❌ | High — URL changes can exfiltrate data | Add audit table |
| Branding | ❌ | Low — cosmetic changes only | Not needed |
| Tenant Settings | ❌ | Medium — general, locale, notification, compliance configs | Consider for compliance sub-config |
| IP Restriction Rules | ❌ | High — firewall rule changes | Add audit table |

---

## Summary of Actions

### Must Do

| # | Action | Status | Standards Basis |
|---|---|:-:|---|
| 1 | Redesign `auth_logs` → `auth_events` with the schema above | ✅ | `[PCI-DSS] 10.2`, `[OWASP-VOCAB]`, `[OWASP-AUTHN]`, `[NIST-63B]` |
| 2 | Build `AuthEventService` with a `Log(ctx, event)` method | ✅ | Architecture layer pattern |
| 3 | Wire `Log()` calls into login service (remaining: token, password, session, user, role/permission) | 🔶 | `[OWASP-VOCAB]` event types |
| 4 | Build admin REST API (`GET /auth-events`, `GET /auth-events/:uuid`, `GET /auth-events/count`) | ✅ | `[PCI-DSS] 10.4.1` — able to review audit trail |
| 5 | Implement retention background job using `DeleteOlderThan` | ✅ | `[PCI-DSS] 10.7`, `[GDPR-5]` |
| 6 | Remove `service_logs` and `auth_logs` migrations, models, repos, and permissions | ✅ | `[NIST-92]` — use centralized log management |

### Should Do

| # | Action | Status | Standards Basis |
|---|---|:-:|---|
| 7 | Add audit tables for `email_config`, `sms_config`, `webhook_endpoints`, `ip_restriction_rules` | ☐ | `[PCI-DSS] 10.2.7`, `[SOC2-CC6]` |
| 8 | Emit OTel span attributes alongside every `auth_events` DB write (dual-write to both layers) | ☐ | `[NIST-92]` — correlation between log sources |

### Nice to Have

| # | Action | Status | Standards Basis |
|---|---|:-:|---|
| 9 | Implement `authn_impossible_travel` detection using IP geolocation | ☐ | `[OWASP-VOCAB]` |
| 10 | Build dashboard aggregation endpoints (events by type, failures over time) | ☐ | `[PCI-DSS] 10.4.1` |

---

## Appendix A: Current `auth_logs` Schema (for reference)

```sql
-- Migration 043 — current schema (to be replaced)
CREATE TABLE IF NOT EXISTS auth_logs (
    auth_log_id         SERIAL PRIMARY KEY,
    auth_log_uuid       UUID NOT NULL UNIQUE,
    tenant_id           INTEGER NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    user_id             INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    event_type          VARCHAR(100) NOT NULL,  -- 'login', 'logout', 'token_refresh', 'password_reset'
    description         TEXT,
    ip_address          VARCHAR(100),
    user_agent          TEXT,
    metadata            JSONB,
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ DEFAULT now()
);
```

## Appendix B: Current `auth_logs` Repository Methods (to be refactored)

The existing `AuthLogRepository` in `internal/repository/auth_audit_log.go` implements:

| Method | Description | Keep? |
|---|---|:-:|
| `FindPaginated(filter)` | Pagination with tenant/user/event type/date range filters, SQL-injection-safe sort | ✅ Refactor |
| `FindByUUIDAndTenantID(uuid, tenantID)` | Single record lookup | ✅ Keep |
| `FindByDateRange(tenantID, from, to)` | Date range query | ✅ Keep |
| `DeleteOlderThan(cutoff)` | Retention management | ✅ Keep |
| `CountByEventType(eventType, tenantID)` | Aggregation for dashboards | ✅ Keep |
| `WithTx(tx)` | Transaction binding | ✅ Keep |

The repository methods are well-implemented. The refactoring is primarily:
1. Rename to `AuthEventRepository`
2. Update model references from `AuthLog` to `AuthEvent`
3. Add `category`, `severity`, and `result` to the filter struct
4. Update `FindPaginated` to support the new filter fields

## Appendix C: OWASP Logging Vocabulary Quick Reference

Source: [OWASP Logging Vocabulary Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Vocabulary_Cheat_Sheet.html)

```
AUTHN  — Authentication events
  authn_login_success, authn_login_fail, authn_login_fail_max,
  authn_login_lock, authn_login_successafterfail,
  authn_password_change, authn_password_change_fail,
  authn_token_created, authn_token_revoked, authn_token_reuse,
  authn_token_delete, authn_impossible_travel

AUTHZ  — Authorization events
  authz_fail, authz_change, authz_admin

SESSION — Session management events
  session_created, session_renewed, session_expired,
  session_use_after_expire

USER   — User management events
  user_created, user_updated, user_archived, user_deleted

PRIVILEGE — Privilege change events
  privilege_permissions_changed

SYS    — System lifecycle events
  sys_startup, sys_shutdown, sys_restart, sys_crash,
  sys_monitor_disabled, sys_monitor_enabled

INPUT  — Input validation events
  input_validation_fail, input_validation_discrete_fail

EXCESS — Rate limiting events
  excess_rate_limit_exceeded

MALICIOUS — Malicious behavior events
  malicious_excess_404, malicious_extraneous,
  malicious_attack_tool, malicious_sqli,
  malicious_cors, malicious_direct_reference
```

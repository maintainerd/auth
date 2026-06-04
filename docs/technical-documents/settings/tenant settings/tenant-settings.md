# Tenant Settings

## Overview

Tenant settings is a centralized configuration resource that holds four major operational JSONB settings for the tenant: **rate limit configuration**, **audit configuration**, **maintenance configuration**, and **feature flags**. Each sub-config is independently readable and updatable via the admin API.

---

## 1. Rate Limit Configuration

### What It Is

Rate limiting restricts the number of requests a client can make within a time window. It protects against brute-force attacks, credential stuffing, denial-of-service, and resource exhaustion. In authentication services, rate limiting is especially critical because login endpoints are the #1 target for automated attacks.

### Relevant Standards

| Standard | Reference | Key Guidance |
|----------|-----------|-------------|
| **RFC 6585** | Additional HTTP Status Codes | Defines `429 Too Many Requests` — the standard response when a rate limit is exceeded. |
| **IETF draft-ietf-httpapi-ratelimit-headers** | RateLimit Header Fields | Proposed standard for `RateLimit-Limit`, `RateLimit-Remaining`, `RateLimit-Reset` response headers. |
| **OWASP API Security Top 10** | API4:2023 — Unrestricted Resource Consumption | Recommends rate limiting, cost quotas, and throttling on all API endpoints. |
| **OWASP ASVS v4** | V11.1.7 | Application should throttle by number of requests per time period. |
| **NIST SP 800-53 Rev 5** | SC-5 (Denial of Service Protection) | Recommends mechanisms to protect against excessive resource consumption. |
| **PCI DSS v4.0** | Req 6.2.4 | Applications must prevent common attacks including automated brute-force. |
| **CIS Controls v8** | Control 13.10 | Rate limit inbound connections to prevent brute-force attacks. |

### How Rate Limiting Typically Works

- **Fixed Window**: Count requests per calendar-aligned window (e.g., 100 req/minute). Simple but allows burst at window boundaries.
- **Sliding Window**: Count requests in a rolling window. Smoother but more state to manage.
- **Token Bucket**: A bucket fills at a steady rate; each request consumes a token. Allows bursts up to bucket capacity.
- **Leaky Bucket**: Requests enter a queue processed at a fixed rate. Smoothest but may add latency.
- **Per-Identity**: Limits applied per user, per IP, per API key, or per tenant.
- **Tiered**: Different limits for different operations (login: 5/min; read: 100/min; write: 20/min).

---

## 2. Audit Configuration

### What It Is

Audit configuration controls how the system records and retains security-relevant events. An audit trail is the immutable log of who did what, when, and from where. It is a mandatory requirement in virtually every compliance framework.

### Relevant Standards

| Standard | Reference | Key Guidance |
|----------|-----------|-------------|
| **SOC 2 Type II** | Trust Services Criteria (CC6, CC7) | Requires logging and monitoring of all access to systems containing sensitive data. |
| **GDPR Art. 30** | Records of Processing Activities | Requires maintaining records of all personal data processing activities. |
| **ISO 27001:2022** | A.8.15 — Logging | Requires producing, storing, protecting, and analyzing event logs. |
| **NIST SP 800-92** | Guide to Computer Security Log Management | Defines log generation, storage, protection, and analysis best practices. |
| **PCI DSS v4.0** | Req 10.1–10.7 | Extensive logging requirements: who, what, when, where, success/failure; retain 1+ year. |
| **HIPAA** | §164.312(b) — Audit Controls | Requires hardware/software/procedural mechanisms to record and examine system activity. |
| **OWASP Logging Cheat Sheet** | Logging Best Practices | Enumerate what to log, what NOT to log (secrets), and how to protect logs. |

### What to Audit in an Auth System

- All authentication events (login success/failure, logout, token refresh)
- All authorization decisions (access granted/denied)
- Account lifecycle (creation, deletion, suspension, role changes)
- Configuration changes (settings updates, rule changes)
- MFA enrollment/removal
- Password changes and resets
- Administrative actions (impersonation, bulk operations)
- API key creation/revocation

---

## 3. Maintenance Configuration

### What It Is

Maintenance mode allows administrators to temporarily take the authentication service offline for planned updates, database migrations, or emergency interventions. While active, all non-admin requests receive a standardized maintenance response, preventing user-facing errors during system changes.

### Relevant Standards & Patterns

| Standard | Reference | Key Guidance |
|----------|-----------|-------------|
| **RFC 9110** | HTTP 503 Service Unavailable | The standard status code for temporary unavailability; should include `Retry-After` header. |
| **ISO 27001:2022** | A.8.32 — Change Management | Requires planned, controlled, and documented changes to IT systems. |
| **ITIL Change Management** | Service Transition | Defines pre-approved (standard), normal, and emergency change types with approval workflows. |
| **SRE Best Practices** | Google SRE Book, Ch. 8 | Release engineering: gradual rollouts, feature flags as an alternative to full maintenance windows. |

### How Maintenance Mode Typically Works

1. Admin toggles maintenance mode via API or config.
2. All non-admin requests receive `503 Service Unavailable` with `Retry-After` header.
3. A human-readable maintenance page/message is returned for browser requests.
4. Admin/internal requests continue to work (bypass mode).
5. Planned maintenance can be scheduled (start/end time) with advance HTTP `Retry-After` headers.
6. Monitoring alerts are suppressed during planned windows.

---

## 4. Feature Flags

### What It Is

Feature flags (feature toggles) allow enabling or disabling specific functionality at runtime without deploying new code. In an auth service, they control which features are available — such as social login, MFA enforcement, passwordless auth, or new registration flows.

### Relevant Standards & Patterns

| Standard | Reference | Key Guidance |
|----------|-----------|-------------|
| **OpenFeature** | [openfeature.dev](https://openfeature.dev/) | CNCF project defining a vendor-neutral API for feature flag evaluation. |
| **Martin Fowler** | Feature Toggles (original article) | Categorizes toggles: release, experiment, ops, permission. Recommends minimizing toggle debt. |
| **OWASP ASVS v4** | V1.2.2 | Security controls should not be feature-flagged off in production. |
| **LaunchDarkly Patterns** | Feature Flag Best Practices | Lifecycle: create → enable → test → default-on → remove. Avoid permanent flags. |

### Feature Flag Categories

| Category | Description | Example |
|----------|-------------|---------|
| **Release toggle** | Control rollout of new features | `enable_passwordless_login` |
| **Ops toggle** | Circuit breaker for operational control | `enable_email_sending` |
| **Experiment toggle** | A/B testing for different flows | `enable_new_registration_flow` |
| **Permission toggle** | Feature availability per plan/tier | `enable_custom_domains` |

---

## Our Implementation

### Architecture

Tenant settings is a **tenant-level** singleton resource (one row per tenant) with four JSONB columns, each holding a structured configuration object. The row is enforced as unique per tenant via a unique index on `tenant_id`.

### Data Model

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `tenant_id` | UUID | Foreign key → tenant (unique index) |
| `rate_limit_config` | JSONB | Rate limiting configuration |
| `audit_config` | JSONB | Audit/logging configuration |
| `maintenance_config` | JSONB | Maintenance mode configuration |
| `feature_flags` | JSONB | Feature flag key-value pairs |
| `created_at` | timestamp | Creation time |
| `updated_at` | timestamp | Last update time |
| `deleted_at` | timestamp | Soft-delete time (nullable) |

**Source files:**
- Model: `internal/model/tenant_setting.go`
- Migration: `internal/database/migration/004_create_tenant_settings.go`

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/tenant-settings/rate-limit` | Get rate limit configuration |
| `PUT` | `/tenant-settings/rate-limit` | Update rate limit configuration |
| `GET` | `/tenant-settings/audit` | Get audit configuration |
| `PUT` | `/tenant-settings/audit` | Update audit configuration |
| `GET` | `/tenant-settings/maintenance` | Get maintenance configuration |
| `PUT` | `/tenant-settings/maintenance` | Update maintenance configuration |
| `GET` | `/tenant-settings/feature-flags` | Get feature flags |
| `PUT` | `/tenant-settings/feature-flags` | Update feature flags |

**Source files:**
- Handler: `internal/rest/tenant_setting_handler.go`
- Routes: `internal/rest/tenant_setting_routes.go`

### Service Layer

The service provides nine operations:
- `GetRateLimitConfig` / `UpdateRateLimitConfig`
- `GetAuditConfig` / `UpdateAuditConfig`
- `GetMaintenanceConfig` / `UpdateMaintenanceConfig`
- `GetFeatureFlags` / `UpdateFeatureFlags`
- `ensureTenantSetting` (internal helper — creates the row on first access)

All operations are traced with OpenTelemetry spans. The service uses a unified internal helper `updateConfig` that reads the current row, patches the specific JSONB column, and writes back within a transaction.

**Source files:**
- Service: `internal/service/tenant_setting_service.go`
- Repository: `internal/repository/tenant_setting_repository.go`
- DTO: `internal/dto/tenant_setting.go`

---

## Requirements Checklist

### Rate Limit Configuration
- [x] Get rate limit config
- [x] Update rate limit config
- [x] Store as JSONB for flexible schema
- [ ] Define rate limit config schema (requests per window, window duration, per-identity type)
- [ ] Rate limit middleware that reads config at runtime
- [ ] Per-endpoint rate limits (login vs. API vs. admin)
- [ ] Per-identity rate limits (per user, per IP, per API key)
- [ ] Rate limit response headers (`RateLimit-Limit`, `RateLimit-Remaining`, `RateLimit-Reset`)
- [ ] `429 Too Many Requests` response with `Retry-After` header
- [ ] Rate limit storage in Redis (sliding window counters)
- [ ] Exempt list (IPs or API keys exempt from rate limiting)
- [ ] Rate limit metrics (hits, rejects, by endpoint)
- [ ] Rate limit config validation (min/max values, sane defaults)

### Audit Configuration
- [x] Get audit config
- [x] Update audit config
- [x] Store as JSONB for flexible schema
- [ ] Define audit config schema (enabled events, retention period, log level, export format)
- [ ] Audit event producer (hook into auth/admin operations)
- [ ] Audit event storage (append-only table or external log service)
- [ ] Audit log retention policy enforcement
- [ ] Audit log query API (filter by user, action, date range)
- [ ] Audit log export (CSV, JSON)
- [ ] Audit log immutability protections (prevent modification/deletion)
- [ ] Configurable audit verbosity (minimal → verbose)
- [ ] PII handling in audit logs (mask sensitive fields)
- [ ] Audit config validation (valid event types, sane retention range)

### Maintenance Configuration
- [x] Get maintenance config
- [x] Update maintenance config
- [x] Store as JSONB for flexible schema
- [ ] Define maintenance config schema (enabled, message, start_time, end_time, bypass_roles)
- [ ] Maintenance middleware (intercept requests, return 503)
- [ ] `Retry-After` header in 503 responses
- [ ] Custom maintenance page/message
- [ ] Admin bypass (admin API unaffected during maintenance)
- [ ] Scheduled maintenance (auto-enable/disable at specified times)
- [ ] Maintenance notification to connected clients
- [ ] Maintenance mode audit log (who enabled/disabled, when)
- [ ] Health check endpoint excluded from maintenance block

### Feature Flags
- [x] Get feature flags
- [x] Update feature flags
- [x] Store as JSONB for flexible schema
- [ ] Define feature flag schema (key, value, description, created_at)
- [ ] Feature flag evaluation helper (check flag at runtime)
- [ ] Feature flag types (boolean, percentage rollout, user-segment)
- [ ] Default values (fallback when flag is undefined)
- [ ] Feature flag middleware integration
- [ ] Feature flag change audit log
- [ ] Prevent disabling security-critical features in production (OWASP ASVS V1.2.2)
- [ ] Feature flag cleanup / deprecation workflow
- [ ] OpenFeature SDK compatibility

### Shared / Cross-Cutting
- [x] Singleton per tenant (unique index on tenant_id)
- [x] Auto-create on first access (ensureTenantSetting helper)
- [x] OpenTelemetry tracing on all operations
- [x] Unit tests for service layer (100% coverage)
- [x] Unit tests for DTO validation
- [x] Unit tests for handler layer
- [ ] JSONB schema validation on each sub-config
- [ ] Default values for each sub-config on initial creation
- [ ] Config change audit trail (what changed, by whom, old value → new value)
- [ ] Integration tests with real database
- [ ] Config export / import (backup/restore)

---

## References

- [RFC 6585 — Additional HTTP Status Codes (429)](https://datatracker.ietf.org/doc/html/rfc6585)
- [IETF RateLimit Header Fields Draft](https://datatracker.ietf.org/doc/draft-ietf-httpapi-ratelimit-headers/)
- [OWASP API Security Top 10 (2023)](https://owasp.org/API-Security/editions/2023/en/0xa4-unrestricted-resource-consumption/)
- [NIST SP 800-92 — Guide to Computer Security Log Management](https://csrc.nist.gov/publications/detail/sp/800-92/final)
- [SOC 2 Trust Services Criteria](https://www.aicpa-cima.com/topic/audit-assurance/audit-and-assurance-greater-than-soc-2)
- [PCI DSS v4.0 — Requirement 10](https://www.pcisecuritystandards.org/)
- [OpenFeature — Feature Flag Standard](https://openfeature.dev/)
- [Martin Fowler — Feature Toggles](https://martinfowler.com/articles/feature-toggles.html)
- [Google SRE Book — Chapter 8: Release Engineering](https://sre.google/sre-book/release-engineering/)

# Security Settings

> **Scope**: User Pool · **API Prefix**: `/security-settings/*` · **Storage**: `security_settings` table (7 JSONB columns)

## Overview

Security settings control the authentication and session security posture for a user pool. Unlike other settings entities that use dedicated tables, all seven security sub-configurations live in a single `security_settings` row as JSONB columns — providing atomic reads and version-tracked, audited updates.

## Sub-Configurations

| Configuration | Description | API Endpoint | Status |
|--------------|-------------|:------------:|:------:|
| [Password](password-config.md) | Password policy (length, complexity, breach checking, rotation, history) | `GET/PUT /security-settings/password` | 🟡 Config API |
| [MFA](mfa-config.md) | Multi-factor authentication (TOTP, SMS, email OTP, trusted devices) | `GET/PUT /security-settings/mfa` | 🟡 Config API ⚠️ Bug |
| [Session](session-config.md) | Token lifetimes, concurrent sessions, idle/absolute timeouts, refresh rotation | `GET/PUT /security-settings/session` | 🟡 Config API |
| [Lockout](lockout-config.md) | Account lockout (max attempts, duration, progressive lockout, auto-unlock) | `GET/PUT /security-settings/lockout` | 🟡 Config API |
| [Registration](registration-config.md) | Self-registration, email/phone verification, domain allow/blocklists | `GET/PUT /security-settings/registration` | 🟡 Config API |
| [Threat Detection](threat-config.md) | Brute force, impossible travel, new device, velocity, risk-based step-up | `GET/PUT /security-settings/threat` | 🟡 Config API |
| [Token](token-config.md) | JWT clock skew, additional claims for access and ID tokens | `GET/PUT /security-settings/token` | 🟡 Config API |

**Status legend:**
- 🟡 Config API — Admin CRUD for the configuration is implemented; runtime enforcement is not yet built.

## Architecture

### Data Model

All seven configs share a single table row per user pool:

```
security_settings
├── security_setting_id     (PK, uint)
├── security_setting_uuid   (UUID, unique)
├── user_pool_id            (FK → user_pools)
├── password_config         (JSONB)
├── mfa_config              (JSONB)
├── session_config          (JSONB)
├── lockout_config          (JSONB)
├── registration_config     (JSONB)
├── threat_config           (JSONB)
├── token_config            (JSONB)
├── version                 (uint, auto-incremented on every update)
├── created_by              (FK → users)
├── updated_by              (FK → users)
├── created_at              (timestamp)
└── updated_at              (timestamp)
```

### Audit Trail

Every config update creates a row in `security_settings_audit`:

```
security_settings_audit
├── security_settings_audit_id      (PK, uint)
├── security_settings_audit_uuid    (UUID, unique)
├── user_pool_id                    (FK → user_pools)
├── security_setting_id             (FK → security_settings)
├── change_type                     (string: "update_password_config", "update_mfa_config", etc.)
├── old_config                      (JSONB — previous value of the changed config)
├── new_config                      (JSONB — new value)
├── ip_address                      (string — admin's IP)
├── user_agent                      (string — admin's browser)
├── created_by                      (FK → users)
├── created_at                      (timestamp)
└── updated_at                      (timestamp)
```

### Service Layer

The `SecuritySettingService` exposes 15 methods:

| Method | Description |
|--------|-------------|
| `GetByUserPoolID` | Returns the full security settings row (all 7 configs) |
| `GetPasswordConfig` | Returns only the `password_config` JSONB |
| `UpdatePasswordConfig` | Replaces `password_config`, increments version, creates audit record |
| `GetMFAConfig` | Returns only the `mfa_config` JSONB |
| `UpdateMFAConfig` | ⚠️ **Bug:** Passes `"general"` instead of `"mfa"` to internal switch |
| `GetSessionConfig` | Returns only the `session_config` JSONB |
| `UpdateSessionConfig` | Replaces `session_config`, increments version, creates audit record |
| `GetLockoutConfig` | Returns only the `lockout_config` JSONB |
| `UpdateLockoutConfig` | Replaces `lockout_config`, increments version, creates audit record |
| `GetRegistrationConfig` | Returns only the `registration_config` JSONB |
| `UpdateRegistrationConfig` | Replaces `registration_config`, increments version, creates audit record |
| `GetThreatConfig` | Returns only the `threat_config` JSONB |
| `UpdateThreatConfig` | Replaces `threat_config`, increments version, creates audit record |
| `GetTokenConfig` | Returns only the `token_config` JSONB |
| `UpdateTokenConfig` | Replaces `token_config`, increments version, creates audit record |

All update methods share an internal `updateConfig` helper that runs in a transaction:
1. Find or create the `security_settings` row for the user pool
2. Set the appropriate JSONB column
3. Increment `version`
4. Create a `security_settings_audit` record

### Known Issues

| Issue | Severity | Location | Description |
|-------|----------|----------|-------------|
| `UpdateMFAConfig` bug | **High** | `service/security_setting.go` | Passes `"general"` instead of `"mfa"` to the config type switch, causing all MFA updates to fail with `"invalid config type"` |
| No JSONB schema validation | Medium | `dto/security_setting.go` | DTO accepts `map[string]any` with only non-empty check — no field-level validation |
| No default values | Low | `service/security_setting.go` | Lazy-created rows have empty JSONB columns — no sane defaults |

## Source Files

| File | Description |
|------|-------------|
| `internal/model/security_setting.go` | `SecuritySetting` struct with 7 JSONB columns |
| `internal/model/security_settings_audit.go` | `SecuritySettingsAudit` struct for change tracking |
| `internal/dto/security_setting.go` | `SecuritySettingUpdateConfigRequestDTO` (`map[string]any`) |
| `internal/service/security_setting.go` | 15-method service with shared `updateConfig` helper |
| `internal/rest/security_setting_handler.go` | 14 HTTP handlers (7 GET + 7 PUT) |
| `internal/rest/security_setting_routes.go` | Route registration |
| `internal/repository/security_setting.go` | `FindByUserPoolID`, `IncrementVersion`, `WithTx` |
| `internal/repository/security_settings_audit.go` | `FindBySecuritySettingID`, `FindByUserPoolID` |
| `internal/database/migration/037_create_security_settings_table.go` | Main table migration |
| `internal/database/migration/039_create_security_settings_audit_table.go` | Audit table migration |

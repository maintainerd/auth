# Security Settings

> Per-tenant security policy stored as seven named JSONB config columns in one row, managed through an audited, step-up-gated control-plane REST surface and enforced at runtime by the auth flows that load each config as an "effective policy".

| | |
|---|---|
| **Status** | Implemented — both the admin CRUD surface **and** runtime enforcement are wired (each config has a `Load*Policy` / `ResolveEffective*Policy` consumer). |
| **Code** | `internal/secpolicy` (store, service, handlers, validation, defaults, per-config policy loaders); consumers in `internal/authn`, `internal/mfa`; step-up gate in `internal/platform/middleware`; enforced structs in `internal/platform/security` + `internal/platform/jwt` |
| **Endpoints** | `GET`/`PUT /api/v1/security-settings/{mfa,password,session,threat,lockout,registration,token}` (control-plane internal router). Sibling IP-allow/deny CRUD: `/api/v1/ip-restriction-rules` |
| **Storage** | `security_settings` (migration `054`), `security_settings_audit` (migration `056`) |
| **Config** | No env vars govern the store itself; every knob is per-tenant JSONB. Step-up freshness reads `mfa_config.step_up_ttl_minutes`. (`CAPTCHA_SECRET` gates the registration captcha field, which ships disabled.) |

## Overview

`security_settings` holds one row per tenant. Rather than a table per policy area, all seven sub-configs live as named JSONB columns on that single row (`mfa_config`, `password_config`, `session_config`, `threat_config`, `lockout_config`, `registration_config`, `token_config`), giving atomic reads and one version counter for the whole posture. `internal/secpolicy/model_setting.go:13`

Two distinct surfaces use the package:

1. **Admin CRUD** — `GET`/`PUT` per config on the control-plane management API. Reads auto-create the row with shipped defaults; writes validate, normalize, increment `version`, and append an audit row.
2. **Runtime enforcement** — auth flows call `secpolicy.Load*Policy(...)` / `ResolveEffective*Policy(...)` to turn stored JSONB into an enforced policy struct at the moment of login, registration, password change, MFA, and token issuance. These loaders **fail safe to the shipped defaults** (logged, never silently off) when settings are missing or unreadable.

The old "config API only, enforcement not built" note and the "`general` vs `mfa` config-type bug" are both stale — enforcement is live and the service uses the correct `mfa` key. `internal/secpolicy/service_setting.go:70,217`

## How it works

### Write path (`PUT /security-settings/{type}`)

1. Middleware chain: `JWTAuthMiddleware` → `UserContextMiddleware` → optional tenant rate-limit → `PermissionMiddleware(["security-setting:update"])` → **`RequireStepUp`**. `internal/secpolicy/routes.go:56`
2. Handler pulls the authenticated user + tenant from context, then decodes the body with `DecodeSecuritySettingUpdateConfig(type, body)` — a strict decode (`DisallowUnknownFields`, single JSON object) into the config-specific DTO, then per-field validation. `internal/secpolicy/handler_setting.go:188`, `validation_setting.go:57`
3. Service `updateConfig` runs a DB transaction: find-or-create the tenant row → `NormalizeSecuritySettingConfig` (merge shipped defaults + existing allowed keys + patch, re-validate the merged result) → marshal into the target JSONB column → `CreateOrUpdate` → `IncrementVersion` → insert a `security_settings_audit` row capturing old/new config, actor, IP, user agent. `internal/secpolicy/service_setting.go:285`
4. The row is re-read and returned; the response is the full normalized config for that type.

### Read path (`GET /security-settings/{type}`)

`getConfig` → `getOrCreateSecuritySetting` (creates a defaults row on first access) → `NormalizeSecuritySettingConfig(type, stored, nil)` fills any missing keys from defaults so the client always sees a complete config. `internal/secpolicy/service_setting.go:193,265`

### Enforcement path (per config)

| Config | Loader | Enforced struct | Where it fires |
|--------|--------|-----------------|----------------|
| `password` | `LoadPasswordPolicy` | `security.PasswordPolicy` | register, reset-password, login (`internal/authn/service_register.go:511`, `service_reset_password.go:149`, `service_login.go:624`) |
| `mfa` | `LoadMFAPolicy` → `MFAPolicy` | method gating, TOTP params, grace/trusted windows, step-up TTL | `internal/mfa/service_mfa.go` (many), `service_login.go:693` |
| `session` | `ResolveEffectiveSessionPolicy` | `EffectiveSessionPolicy` | token/session issuance (`internal/authn/service_security_policy.go:24`) |
| `token` | `ResolveEffectiveTokenPolicy` | `EffectiveTokenPolicy` | token issuance / PKCE gate (`service_security_policy.go:41`) |
| `lockout` | `LoadLockoutPolicy` | `security.RateLimitConfig` | login rate-limit (`service_login.go:229`) |
| `registration` | `LoadRegistrationPolicy` → `RegistrationPolicy` | self-signup gate, email-domain allow/block, initial status | register/login/email-verify (`service_register.go:444`, `service_login.go:167`) |
| `threat` | `LoadThreatPolicy` | `security.ThreatConfig` | login threat assessment (`service_login.go:254`, `service_sms_login.go:134`) |

All loaders normalize the stored JSONB through the same `NormalizeSecuritySettingConfig`, so a stored value always maps back to the same defaults+validation contract the write path enforced.

## Implementation

### Storage

`security_settings` — one row per tenant (`uq_security_settings_tenant_id` unique index), FK to `tenants` (cascade delete), `created_by`/`updated_by` FK to `users`. Seven `JSONB NOT NULL DEFAULT '{}'` config columns plus a `version INTEGER` bumped on every write. `internal/platform/database/migration/054_create_security_settings_table.go`

`security_settings_audit` — append-only change log (no `updated_at`, no soft delete): `change_type` (`update_<key>_config`), `old_config`/`new_config` JSONB, `ip_address INET`, `user_agent TEXT`, `created_by`. FKs to `tenants`, `security_settings`, `users`. `internal/platform/database/migration/056_create_security_settings_audit_table.go`, `model_settings_audit.go`

### Key files

| File | Role |
|------|------|
| `internal/secpolicy/model_setting.go` | `SecuritySetting` (7 JSONB columns, `version`, auditors) |
| `internal/secpolicy/model_settings_audit.go` | `SecuritySettingsAudit` append-only record |
| `internal/secpolicy/types.go` | Per-config DTOs + `Effective{Session,Token}Policy`, `SecuritySettingClientOverrides` |
| `internal/secpolicy/defaults_setting.go` | Shipped default map per config; `NewDefaultSecuritySetting`, `ApplySecuritySettingDefaults` |
| `internal/secpolicy/validation_setting.go` | Strict decode, `NormalizeSecuritySettingConfig`, per-field validators, `ResolveEffective{Session,Token}Policy` |
| `internal/secpolicy/service_setting.go` | `SecuritySettingService` (7 Get + 7 Update), transactional `updateConfig`, table-driven config definitions |
| `internal/secpolicy/handler_setting.go` | 14 HTTP handlers (7 GET + 7 PUT) |
| `internal/secpolicy/routes.go` | `SecuritySettingRoute`, `IPRestrictionRuleRoute` |
| `internal/secpolicy/repository_setting.go` | `FindByTenantID`, `IncrementVersion`, `WithTx` |
| `internal/secpolicy/{password,mfa,lockout,threat,registration}_policy.go`, `session_helpers.go` | Runtime policy loaders (fail-safe to defaults) |

> **UI labels vs. route keys.** The console labels `mfa` as **"General"** and `lockout` as **"IP"** (`handler_setting.go:12`), and some handler doc-comments say `/general` and `/ip`. The *routes* are authoritative: the actual paths are `/security-settings/mfa` and `/security-settings/lockout`. `internal/secpolicy/routes.go:70,94`

### Endpoints

All under the control-plane internal router `/api/v1` (`internal/server/router.go:91`), behind `RequireManagementClient`. Reads need `security-setting:read`; writes need `security-setting:update` **and** a fresh step-up token.

| Method | Path | Permission | Step-up |
|--------|------|-----------|:-------:|
| GET | `/security-settings/{mfa,password,session,threat,lockout,registration,token}` | `security-setting:read` | no |
| PUT | `/security-settings/{mfa,password,session,threat,lockout,registration,token}` | `security-setting:update` | **yes (acr=2)** |

## Configuration

No environment variables configure the store; every value is a per-tenant JSONB key. Shipped defaults come from `defaults_setting.go`. Below are the four configs owned by this doc; `session`, `mfa`, and `registration` have their own feature docs (see **Related**) — their defaults are summarized here for completeness only.

### `password_config`

| Key | Default | Validation |
|-----|---------|-----------|
| `min_length` | 12 | ≥ 1 (floor deliberately low; validator runs on read too — see `validation_setting.go:39`) |
| `max_length` | 128 | 64–128; `min_length ≤ max_length` |
| `require_uppercase`/`_lowercase`/`_number`/`_symbol` | false | bool |
| `reject_common_passwords` | true | bool |
| `check_hibp` | true | bool (breach check) |
| `password_history_count` | 5 | 0–24 |
| `max_age_days` | 0 (no rotation) | 0–3650 |
| `temporary_password_validity_hours` | 72 | 1–720 |
| `hash_algorithm` | `argon2id` | one of `argon2id, bcrypt, scrypt, pbkdf2` |
| `min_strength_score` | 2 | 0–4 |

Loader tolerates legacy short keys (`require_upper`, `history_count`, `expiry_days`, …) via `canonicalPasswordConfigAliases`; **the canonical key always wins** when both are present. `password_policy.go:128`

### `lockout_config`

| Key | Default | Validation |
|-----|---------|-----------|
| `enabled` | true | bool |
| `max_failed_attempts` | 5 | 1–100 |
| `lockout_duration_minutes` | 30 | ≥ 1 |
| `progressive_lockout` | true | bool |
| `auto_unlock` | true | bool |
| `reset_count_on_success` | true | bool |
| `observation_window_minutes` | 15 | ≥ 1 |
| `max_lockout_duration_minutes` | 60 | ≥ `lockout_duration_minutes` |
| `progression_reset_hours` | 24 | ≥ 1 |
| `notify_user_on_lockout` | true | bool |

`LoadLockoutPolicy` never returns nil on failure — the login path treats nil as "lockout off", so a read error falls back to the shipped defaults (logged) to keep the control on. `lockout_policy.go:23`

### `threat_config`

| Key | Default | Validation |
|-----|---------|-----------|
| `brute_force_detection_enabled` | true | bool |
| `impossible_travel_detection_enabled` | true | bool |
| `new_device_notification_enabled` | true | bool |
| `velocity_check_enabled` | true | bool |
| `risk_based_step_up_enabled` | false | bool |
| `compromised_credential_monitoring_enabled` | true | bool |
| `ip_reputation_check_enabled` | false | **rejected when `true`** — no provider ships (fail-open guard) |
| `block_tor_exit_nodes` | false | **rejected when `true`** — no Tor source ships (fail-open guard) |
| `risk_step_up_threshold` | 21 | 0–100; ≤ `risk_block_threshold` |
| `risk_block_threshold` | 81 | 0–100 |
| `velocity_failures_per_ip_per_hour` | 50 | ≥ 1 |
| `distinct_accounts_per_ip_per_hour` | 10 | ≥ 1 |

### `token_config`

| Key | Default | Validation |
|-----|---------|-----------|
| `clock_skew_leeway_seconds` | 30 | 0–300 |
| `additional_id_token_claims` | `["roles","tenant_id"]` | allowlist = `roles, tenant_id` only |
| `additional_access_token_claims` | `["roles","tenant_id"]` | allowlist = `roles, tenant_id` only |
| `signing_algorithm` | `RS256` | one of `RS256, PS256` — **ES256 rejected** (key store is RSA-only) |
| `require_pkce` | true | bool (public clients force this on regardless) |
| (whole config) | — | serialized `token_config` must stay < 4 KB |

### Summarized defaults for sibling configs

- `session_config`: `access_token_ttl_minutes` 15, `refresh_token_ttl_days` 30, `max_concurrent_sessions` 5, `idle_timeout_minutes` 30, `absolute_timeout_hours` 24, `rotate_refresh_tokens` true, `refresh_token_reuse_interval_seconds` 10, `cookie_secure` true, `cookie_http_only` true, `cookie_same_site` `Lax`, `revoke_sessions_on_password_change` true. → see ./sessions.md
- `mfa_config`: `mode` `optional`, `allowed_methods` `[totp, webauthn, recovery_code]`, `totp_issuer` `Maintainerd-Auth`, `trusted_device_period_days` 14, `grace_period_days` 30, `preferred_method` `webauthn`, `allow_sms` false, `allow_email_otp` false, `totp_digits` 6, `totp_period_seconds` 30, `recovery_codes_count` 10, `require_mfa_for_sensitive_actions` true, `admin_grace_period_days` 0, `step_up_ttl_minutes` 5. → see ./multi-factor-auth.md
- `registration_config`: `self_registration_enabled` true, `require_email_verification` true, `require_phone_verification` false, allow/block domain lists empty, `auto_confirm_enabled` false, `verification_token_ttl_hours` 24, `captcha_on_signup` **false** (deferred; no form emits a captcha token), `registration_rate_limit_per_ip_per_hour` 10. → see ./registration.md

## Security considerations

- **Step-up gate on every write.** All seven `PUT` routes require an elevated `acr=2` token issued within the tenant's step-up freshness window (`mfa_config.step_up_ttl_minutes`, default 300 s). Without it, a stolen `acr=1` admin session could disarm tenant-wide MFA or weaken any policy in a single request; the MFA write is called out as the most destructive because it can flip `mode` off and clear `require_mfa_for_sensitive_actions`. `routes.go:61`, `middleware/jwt_middleware.go:238`
- **Tenant isolation + management client.** The whole surface sits behind `RequireManagementClient` and tenant-scoped middleware; the tenant is taken from context, never the body. Reads/writes only ever touch the caller's own row.
- **Immutable audit trail.** Every write appends an `old_config`/`new_config`/actor/IP/UA row to `security_settings_audit` (append-only, cascade-scoped to the tenant), giving full change forensics.
- **Fail-safe, non-silent degradation.** Password, MFA, and lockout loaders fall back to shipped defaults on any read/normalize error and log a warning — a settings-table blip cannot silently turn a control off while the admin UI still shows it on. `password_policy.go:26`, `mfa_policy.go:39`, `lockout_policy.go:23`
- **"Don't advertise what you can't enforce."** Validation refuses fail-open configurations: `ip_reputation_check_enabled` and `block_tor_exit_nodes` are rejected while `true` (no data source ships), `signing_algorithm` is limited to the algorithms the RSA key store can actually sign (`RS256`/`PS256`; `ES256` rejected), and MFA bypass windows (`trusted_device_period_days`, `grace_period_days`) are bounded so `mode=enforced` cannot be neutered by an absurd value. `validation_setting.go:406,336,248`
- **Token-claim allowlist.** Only `roles` and `tenant_id` may be added to access/ID tokens — auth-context claims (`acr`, `amr`, `nonce`, …) and PII are excluded so an operator cannot forge authentication context or leak personal data through a config path. `validation_setting.go:567`
- **Public clients force PKCE.** `ResolveEffectiveTokenPolicy` escalates `require_pkce` to `true` for public (SPA/native) clients regardless of tenant or per-client config (RFC 9700 §2.1.1). Per-client session overrides can only *tighten* TTLs/timeouts, never loosen them. `validation_setting.go:485`
- **Enforced MFA raises required ACR.** When `mfa_config.mode == "enforced"`, `ResolveEffectiveSessionPolicy` sets the session's `RequiredACR` to `"2"`. `validation_setting.go:464`

## Related

- ./multi-factor-auth.md — MFA modes, methods, TOTP/WebAuthn, trusted devices, step-up (consumes `mfa_config`)
- ./sessions.md — session lifetimes, refresh-token rotation, cookie flags (consumes `session_config`)
- ./registration.md — self-signup, email/phone verification, domain allow/blocklists (consumes `registration_config`)
- ./cryptography-and-keys.md — JWT issuance, claims, PKCE, signing (consumes `token_config`)
- ./ip-restriction-rules.md — IP allow/deny rules served by the sibling handler in `internal/secpolicy`

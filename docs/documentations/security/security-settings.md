# Security Settings — System Reference

> **Status**: implemented and in use. This is the authoritative reference for the tenant security-settings subsystem: data model, configuration API (with request/response structures), every config field and its default, runtime enforcement, the client-override model, the auth-flow behaviors driven by these settings, and standards compliance.
>
> **Audience**: backend developers, frontend integrators, and AI agents extending or integrating this system. Everything needed to integrate or upgrade is here — you should not need to read the source to integrate.
>
> **Owning package**: `internal/secpolicy` · **Storage**: `security_settings` table (one row per tenant, 7 JSONB columns) · **Override layer**: `clients` table (`internal/client`).

---

## 1. Overview

Security settings define a tenant's authentication and session security posture. Each tenant has exactly one `security_settings` row holding seven independent JSONB configuration blocks:

| Block | Controls |
|-------|----------|
| `password_config` | Password length/complexity, breach & strength checks, history, expiry, hashing |
| `mfa_config` | MFA mode, allowed methods, TOTP/recovery params, trusted devices, sensitive-action step-up |
| `session_config` | Token lifetimes, idle/absolute timeouts, concurrency, refresh rotation, cookie flags |
| `token_config` | JWT clock-skew, signing algorithm, extra claims, PKCE requirement |
| `lockout_config` | Failed-login lockout, progressive escalation, auto-unlock, notification |
| `registration_config` | Self-registration, email/phone verification, domain allow/block, captcha |
| `threat_config` | Brute-force/velocity detection, risk-based step-up, new-device, compromised-credential monitoring |

Settings are **tenant-level**. A `client` (relying-party app) may override a deliberately limited, **tighten-only** subset (see §6). All seven blocks are seeded with secure, standards-aligned defaults at tenant creation (§8).

---

## 2. Data model

### `security_settings` (migration `041_create_security_settings_table.go`)

```
security_settings
├── security_setting_id    BIGSERIAL PK
├── security_setting_uuid  UUID UNIQUE
├── tenant_id              BIGINT  FK → tenants (UNIQUE, 1:1, ON DELETE CASCADE)
├── mfa_config             JSONB  DEFAULT '{}'
├── password_config        JSONB  DEFAULT '{}'
├── session_config         JSONB  DEFAULT '{}'
├── threat_config          JSONB  DEFAULT '{}'
├── lockout_config         JSONB  DEFAULT '{}'
├── registration_config    JSONB  DEFAULT '{}'
├── token_config           JSONB  DEFAULT '{}'
├── version                INTEGER DEFAULT 1   (incremented on every update)
├── created_by / updated_by BIGINT FK → users (ON DELETE SET NULL)
└── created_at / updated_at TIMESTAMPTZ
```

No soft delete — it cascade-deletes with the tenant. The GORM model is `secpolicy.SecuritySetting` (`internal/secpolicy/model_setting.go`).

### `security_settings_audit` (migration `043_create_security_settings_audit_table.go`)

Every config update writes one audit row inside the same transaction:

```
security_settings_audit
├── security_settings_audit_id   BIGSERIAL PK
├── security_settings_audit_uuid UUID UNIQUE
├── tenant_id                    FK → tenants
├── security_setting_id          FK → security_settings
├── change_type                  e.g. "update_password_config"
├── old_config                   JSONB (previous value of that block)
├── new_config                   JSONB (new value)
├── ip_address                   admin IP
├── user_agent                   admin client
├── created_by                   FK → users
└── created_at / updated_at      TIMESTAMPTZ
```

### Write path (service `updateConfig`, transactional)
1. Find-or-create the tenant's `security_settings` row (lazy-create uses the §8 defaults).
2. Replace the target JSONB column with the validated, normalized config.
3. Increment `version`.
4. Insert a `security_settings_audit` row capturing old + new + actor + IP + UA.

### Package layout
| Concern | File |
|---|---|
| Model | `internal/secpolicy/model_setting.go`, `model_settings_audit.go` |
| Typed DTOs | `internal/secpolicy/types.go` |
| Decode + validation + effective-policy resolvers | `internal/secpolicy/validation_setting.go` |
| Defaults + seeder helpers | `internal/secpolicy/defaults_setting.go` |
| Per-block policy loaders | `internal/secpolicy/{password,lockout,registration,threat,mfa,session}_policy.go` |
| Service | `internal/secpolicy/service_setting.go` |
| HTTP handler / routes | `internal/secpolicy/handler_setting.go`, `routes.go` |
| Repository | `internal/secpolicy/repository_setting.go`, `repository_settings_audit.go` |
| Seeder | `internal/setup/seeder/011_security_setting.go` |

---

## 3. Configuration API

All endpoints are tenant-scoped (tenant resolved from the auth context) and JSON over HTTPS.

### Endpoints

| Method | Path | Permission | Extra |
|--------|------|------------|-------|
| GET | `/security-settings/password` | `security-setting:read` | |
| PUT | `/security-settings/password` | `security-setting:update` | **step-up (acr=2)** |
| GET/PUT | `/security-settings/mfa` | read / update | PUT step-up |
| GET/PUT | `/security-settings/session` | read / update | PUT step-up |
| GET/PUT | `/security-settings/token` | read / update | PUT step-up |
| GET/PUT | `/security-settings/lockout` | read / update | PUT step-up |
| GET/PUT | `/security-settings/registration` | read / update | PUT step-up |
| GET/PUT | `/security-settings/threat` | read / update | PUT step-up |

Middleware chain: `JWTAuthMiddleware` → `UserContextMiddleware` → `PermissionMiddleware` → (PUT only) `RequireStepUp`. Every **PUT requires a fresh step-up token** (`acr=2`, issued within 5 minutes) — see §7.4.

> Note: a separate `/ip-restriction-rules` CRUD API (`security_settings` package) manages tenant IP allow/deny rules; it is related but out of scope for this document.

### Standard response envelope

All endpoints (and the whole API) use this envelope:

```jsonc
{
  "success": true,            // bool
  "data": { ... },            // payload (omitted on error)
  "message": "…",             // human-readable (omitted on error)
  "error": "…",               // human-readable error (only when success=false)
  "code": "step_up_required", // optional machine-readable code (branch on this, not the message)
  "details": { ... }          // optional structured detail (e.g. validation errors)
}
```

### GET — read a config block
`GET /security-settings/password` →
```jsonc
{
  "success": true,
  "message": "Password config retrieved successfully",
  "data": {
    "min_length": 12,
    "max_length": 128,
    "require_uppercase": false,
    "require_lowercase": false,
    "require_number": false,
    "require_symbol": false,
    "reject_common_passwords": true,
    "check_hibp": true,
    "password_history_count": 5,
    "max_age_days": 0,
    "temporary_password_validity_hours": 72,
    "hash_algorithm": "argon2id",
    "min_strength_score": 2
  }
}
```
`data` is the config object for that block exactly as stored (all keys present, because reads normalize against defaults).

### PUT — update a config block
Request body = a **partial or full** object of that block's fields (typed; unknown keys are rejected). Omitted fields keep their current value.
`PUT /security-settings/password`
```json
{ "min_length": 14, "reject_common_passwords": true }
```
Response = the full updated config block (same shape as GET):
```jsonc
{
  "success": true,
  "message": "Password config updated successfully",
  "data": { "min_length": 14, "max_length": 128, "...": "..." }
}
```

### Error responses
| HTTP | `error` / `code` | When |
|------|------------------|------|
| 400 | `"Invalid request body"` | malformed JSON, unknown field, or field-validation failure (§ each block's rules) |
| 401 | `"Tenant not found in context"` / unauthorized | missing/invalid auth |
| 403 | `code: "step_up_required"` | PUT without a fresh `acr=2` token |
| 403 | forbidden | missing `security-setting:*` permission |
| 500 | fallback message | internal error (logged server-side) |

**Validation contract**: the request body is decoded into a typed DTO per block with `DisallowUnknownFields` — a misspelled or unsupported key is a 400, never silently ignored. Each block validates types, ranges, enums, and cross-field rules (listed per block in §5).

---

## 4. Data types & units

- Tenant config stores **human-friendly units**: `*_minutes`, `*_hours`, `*_days`. Client override columns store **seconds**. The effective-policy resolvers (§6) normalize everything to seconds.
- Optional fields in update DTOs are pointers/omitempty — omitting a field preserves the stored value; an absent field on first write takes the seeded default.
- Booleans default per §8; reads always return the full set of keys.

---

## 5. Configuration blocks — fields, defaults, validation, enforcement

Defaults below are the seeded **Business SaaS** baseline (`internal/secpolicy/defaults_setting.go`). Validation rules are enforced in `validation_setting.go`. "Enforcement" = what the runtime actually does with the value (all verified implemented).

### 5.1 `password_config`

| Field | Type | Default | Validation |
|-------|------|---------|-----------|
| `min_length` | int | `12` | ≥1, ≤ `max_length` |
| `max_length` | int | `128` | ≥64, ≤128 |
| `require_uppercase` | bool | `false` | — |
| `require_lowercase` | bool | `false` | — |
| `require_number` | bool | `false` | — |
| `require_symbol` | bool | `false` | — |
| `reject_common_passwords` | bool | `true` | — |
| `check_hibp` | bool | `true` | — |
| `password_history_count` | int | `5` | ≥0 |
| `max_age_days` | int | `0` (no expiry) | ≥0 |
| `temporary_password_validity_hours` | int | `72` | ≥1 |
| `hash_algorithm` | string | `"argon2id"` | one of `argon2id`,`bcrypt`,`scrypt`,`pbkdf2` |
| `min_strength_score` | int | `2` | 0–4 (zxcvbn-like) |

**Enforcement** (register, password-reset): length + optional complexity; common-password blocklist (embedded `common_passwords.txt`, case-insensitive); HIBP k-anonymity check (SHA-1 5-char prefix range query, **fail-open** on network error); strength score threshold; password history reuse rejection; `max_age_days>0` → sets `ForcePasswordChange` at login (response carries `require_password_change`). Admin-created users receive a temporary initial password, are forced to change it on first login, and are blocked once `temporary_password_validity_hours` has elapsed. Hashing uses the configured KDF — argon2id (64 MiB / t=3 / p=4), bcrypt (cost 12), scrypt (N=2¹⁵), pbkdf2 (600k iters); KDF hashes carry a `$maintainerd$` envelope.

### 5.2 `mfa_config`

| Field | Type | Default | Validation |
|-------|------|---------|-----------|
| `mode` | string | `"optional"` | `disabled`,`optional`,`enforced` |
| `allowed_methods` | []string | `["totp","webauthn","recovery_code"]` | subset of `totp,webauthn,sms,email_otp,recovery_code` |
| `totp_issuer` | string | `"Lula"` | non-empty when `totp` allowed |
| `trusted_device_period_days` | int | `14` | ≥0 (0 disables) |
| `grace_period_days` | int | `30` | ≥0 |
| `preferred_method` | string | `"webauthn"` | must be in `allowed_methods` |
| `allow_sms` | bool | `false` | gates `sms` |
| `totp_digits` | int | `6` | 6 or 8 |
| `totp_period_seconds` | int | `30` | 30–90 |
| `recovery_codes_count` | int | `10` | 0 or 8–16 |
| `require_mfa_for_sensitive_actions` | bool | `true` | — |
| `admin_grace_period_days` | int | `0` | ≥0 |

**Enforcement**: `mode=disabled` hard-disables MFA challenges, trusted-device acceptance, WebAuthn ceremonies, step-up issuance, and factor verification through current tenant policy. At login, `mode=enforced` requires a usable primary factor or login is blocked after the applicable grace window; `admin_grace_period_days` applies only to tenant `super-admin` users, while normal users use `grace_period_days`. `allowed_methods` filters offered and usable factors across login, step-up, TOTP/backup-code generation, SMS, and WebAuthn; `preferred_method` is offered first when enrolled and allowed. SMS is rejected at **enrollment/send/verification** when `allow_sms=false` / not allowed. TOTP issuer/digits/period are read from config and **persisted per enrolled secret** (changing tenant policy never breaks existing enrollments). `recovery_codes_count` controls generated backup codes. `require_mfa_for_sensitive_actions=true` + an MFA-enrolled user → email-change requires a fresh step-up (§7.4). Trusted-device tokens let a remembered device skip MFA within `trusted_device_period_days`. `client.required_acr` can force step-up per client (§6).

### 5.3 `session_config`

| Field | Type | Default | Validation | Client override |
|-------|------|---------|-----------|-----------------|
| `access_token_ttl_minutes` | int | `15` | 1–60 | `access_token_ttl` (sec) |
| `refresh_token_ttl_days` | int | `30` | 1–365 | `refresh_token_ttl` (sec) |
| `max_concurrent_sessions` | int | `5` | ≥0 (0=unlimited) | — |
| `idle_timeout_minutes` | int | `30` | ≥1 | `session_idle_timeout` (sec) |
| `absolute_timeout_hours` | int | `24` | ≥1 | `session_absolute_timeout` (sec) |
| `rotate_refresh_tokens` | bool | `true` | — | — |
| `refresh_token_reuse_interval_seconds` | int | `10` | ≥0 | — |
| `cookie_secure` | bool | `true` | — | — |
| `cookie_http_only` | bool | `true` | — | — |
| `cookie_same_site` | string | `"Lax"` | `Strict`,`Lax`,`None` (None ⇒ secure) | — |
| `revoke_sessions_on_password_change` | bool | `true` | — | — |

**Enforcement**: access/refresh TTLs flow into issued JWTs (clamped by client overrides, §6). Idle + absolute timeouts are enforced on the server-side session at validate/refresh; concurrent sessions beyond the limit evict the oldest. Refresh rotation is single-use with reuse detection + family revocation (§7.5). Cookie flags are applied to auth cookies. Password change revokes other sessions when enabled.

### 5.4 `token_config`

| Field | Type | Default | Validation | Client override |
|-------|------|---------|-----------|-----------------|
| `clock_skew_leeway_seconds` | int | `30` | 0–300 | — |
| `additional_id_token_claims` | []string | `["roles","tenant_id"]` | known claim names only | — |
| `additional_access_token_claims` | []string | `["roles","tenant_id"]` | known claim names only | — |
| `signing_algorithm` | string | `"RS256"` | `RS256`,`ES256`,`PS256` | — |
| `require_pkce` | bool | `true` | — | `require_pkce` (strengthen-only) |

**Enforcement**: `clock_skew_leeway_seconds` is applied to JWT exp/nbf validation; `signing_algorithm` selects the JWT signing method (RS256/PS256 active; ES256 errors unless an EC key store is configured — never silently downgrades); additional access-token claims are injected from authoritative sources (e.g. `tenant_id` from the client record); `require_pkce` is enforced at `/authorize` and `/par` (S256 required, `plain` rejected). Known claim allow-list is enforced at validation; the 4 KB token-size guard rejects oversized configs.

### 5.5 `lockout_config`

| Field | Type | Default | Validation |
|-------|------|---------|-----------|
| `enabled` | bool | `true` | — |
| `max_failed_attempts` | int | `5` | 1–100 |
| `lockout_duration_minutes` | int | `30` | ≥1 |
| `progressive_lockout` | bool | `true` | — |
| `auto_unlock` | bool | `true` | — |
| `reset_count_on_success` | bool | `true` | — |
| `observation_window_minutes` | int | `15` | ≥1 |
| `max_lockout_duration_minutes` | int | `60` | ≥ `lockout_duration_minutes` |
| `progression_reset_hours` | int | `24` | ≥1 |
| `notify_user_on_lockout` | bool | `true` | — |

**Enforcement** (Redis-backed, keyed `tenantID:username`): failures counted within the observation window; at `max_failed_attempts` the account locks for `lockout_duration_minutes`. `progressive_lockout` escalates the duration on repeated lockouts up to `max_lockout_duration_minutes`, resetting after `progression_reset_hours`. `auto_unlock=false` persists the lock (admin unlock required). `reset_count_on_success` clears the counter on login. `notify_user_on_lockout` fires the `OnAccountLockout` hook (wire to notifier for email). Locked logins return a generic error (no account-existence disclosure).

### 5.6 `registration_config`

| Field | Type | Default | Validation |
|-------|------|---------|-----------|
| `self_registration_enabled` | bool | `true` | — |
| `require_email_verification` | bool | `true` | — |
| `require_phone_verification` | bool | `false` | — |
| `allowed_email_domains` | []string | `[]` | valid domains; no overlap with blocked |
| `blocked_email_domains` | []string | `[]` | valid domains |
| `auto_confirm_enabled` | bool | `false` | not both true with `require_email_verification` |
| `verification_token_ttl_hours` | int | `24` | ≥1 |
| `captcha_on_signup` | bool | `true` | — |
| `registration_rate_limit_per_ip_per_hour` | int | `10` | ≥1 |

**Enforcement** (public registration): `self_registration_enabled=false` → 403; email domain allow/block (case-insensitive, supports `*.domain`); captcha verified when `captcha_on_signup`; per-IP rate limit; new users start `pending` (so login is blocked) until email verified, unless `auto_confirm_enabled`/verification not required; role assignment is not configurable here and always uses the system `registered` role.

### 5.7 `threat_config`

| Field | Type | Default | Validation |
|-------|------|---------|-----------|
| `brute_force_detection_enabled` | bool | `true` | — |
| `impossible_travel_detection_enabled` | bool | `true` | — |
| `new_device_notification_enabled` | bool | `true` | — |
| `velocity_check_enabled` | bool | `true` | — |
| `risk_based_step_up_enabled` | bool | `false` | — |
| `compromised_credential_monitoring_enabled` | bool | `true` | — |
| `ip_reputation_check_enabled` | bool | `false` | — |
| `block_tor_exit_nodes` | bool | `false` | — |
| `risk_step_up_threshold` | int | `21` | ≥0, ≤ block threshold |
| `risk_block_threshold` | int | `81` | ≥0 |
| `velocity_failures_per_ip_per_hour` | int | `50` | ≥1 |

**Enforcement**: `AssessLoginThreat` runs **before** the credential check in all login paths (`LoginPublic`, internal `Login`, SMS) with tenant-scoped Redis keys. A score ≥ `risk_block_threshold` blocks the login; with `risk_based_step_up_enabled`, a score ≥ `risk_step_up_threshold` forces an MFA challenge even when MFA is otherwise optional. `RecordLoginThreatSuccess/Failure` track per-IP velocity, last-login (impossible-travel time/IP signal), and device fingerprints (new-device). `compromised_credential_monitoring_enabled` triggers an HIBP check with forced password change on a hit.

**Deferred sub-features** (config-present, behind their flags, need external data sources before they enforce): full GeoIP-distance impossible-travel, `ip_reputation_check`, `block_tor_exit_nodes`. These are accepted future upgrades, not regressions.

---

## 6. Client overrides & effective-policy resolution

Security settings are tenant-level. A `client` row may override a **limited, tighten-only** subset via dedicated columns (mirrored from the client `config` JSON by `internal/client/config_mapping.go`). Capability/credential policy (password, lockout, threat, registration, MFA methods) is **never** client-overridable.

| Client column (`clients`) | Unit | Overrides | Rule |
|---|---|---|---|
| `access_token_ttl` | seconds | `session.access_token_ttl_minutes` | tighten only (`min`) |
| `refresh_token_ttl` | seconds | `session.refresh_token_ttl_days` | tighten only (`min`) |
| `session_idle_timeout` | seconds | `session.idle_timeout_minutes` | tighten only (`min`) |
| `session_absolute_timeout` | seconds | `session.absolute_timeout_hours` | tighten only (`min`) |
| `required_acr` | `"1"`/`"2"` | MFA step-up demand | strengthen only (`max`) |
| `require_pkce` | bool | `token.require_pkce` | strengthen only (OR) |

Columns are NULL-when-absent ⇒ inherit the tenant default.

**Resolvers** (`internal/secpolicy/validation_setting.go`):
- `ResolveEffectiveSessionPolicy(tenantSession, tenantMFA, clientOverrides)` → `EffectiveSessionPolicy` (all seconds; clamped).
- `ResolveEffectiveTokenPolicy(tenantToken, clientOverrides)` → `EffectiveTokenPolicy`.

Resolution logic (the invariant: **a client can only make security stricter, never weaker**):
```
effective_idle      = min(tenant.idle_timeout_minutes*60,   client.session_idle_timeout      ?? +inf)
effective_absolute  = min(tenant.absolute_timeout_hours*3600, client.session_absolute_timeout ?? +inf)
effective_access_ttl= min(tenant.access_token_ttl_minutes*60, client.access_token_ttl        ?? +inf)
effective_refresh   = min(tenant.refresh_token_ttl_days*86400, client.refresh_token_ttl       ?? +inf)
effective_acr       = max(tenant_step_up_acr, client.required_acr ?? "1")   // "2" if MFA enforced or client demands
effective_pkce      = tenant.require_pkce OR client.require_pkce
```
A larger client value is simply ignored (the tenant ceiling wins). OAuth issuance (auth-code, refresh, token-exchange, CIBA, device) and the login/refresh session paths route token/session TTLs through these resolvers.

---

## 7. Auth flows driven by these settings (frontend integration)

These are the runtime behaviors and the request/response structures the frontend must handle. All use the §3 response envelope; auth token payloads are the DTOs below.

### 7.1 Token delivery (cookie vs bearer)
Auth responses return tokens in `data`. Send header `X-Token-Delivery: cookie` to also receive them as hardened cookies (`__Host-access_token`, `__Host-id_token`, `__Secure-refresh_token` scoped to `/api/v1/refresh-token`); otherwise consume the tokens from the JSON body. Cookie flags honor `session_config` (`cookie_secure`, `cookie_http_only`, `cookie_same_site`).

### 7.2 Login → possible MFA challenge
`POST /login` body `LoginRequestDTO`:
```json
{ "username": "...", "password": "...", "trusted_device_token": "optional" }
```
On success (no MFA), `data` is `LoginResponseDTO`:
```jsonc
{
  "access_token": "...", "id_token": "...", "refresh_token": "...",
  "expires_in": 900, "token_type": "Bearer", "issued_at": 1718000000,
  "session_id": "uuid",
  "require_password_change": false
}
```
When MFA is required (enrolled, enforced, or risk-based step-up), the **same endpoint** returns an MFA challenge instead of tokens:
```jsonc
{
  "success": true,
  "data": {
    "mfa_required": true,
    "mfa_challenge_token": "…",
    "mfa_allowed_methods": ["totp","webauthn","sms","backup_code"]
  }
}
```
Frontend must branch on `mfa_required`. Complete it via:
- `POST /login/mfa/verify` — `MFALoginVerifyRequestDTO`: `{ "mfa_challenge_token", "method", "code"?, "assertion"? (webauthn JSON), "remember_device"? }` → returns the full `LoginResponseDTO` (with tokens; `acr=2`). If `remember_device` was set, the response includes `trusted_device_token` to store and replay on future logins.
- `POST /login/mfa/sms` — `MFALoginChallengeRequestDTO` `{ "mfa_challenge_token" }` to send an SMS code.
- `POST /login/mfa/webauthn/begin` — `{ "mfa_challenge_token" }` → WebAuthn assertion options JSON.

`require_password_change: true` means the password expired (`max_age_days`) — route the user to change it.

### 7.3 SMS (passwordless) login
- `POST /login/sms/send` — `SMSLoginSendDTO` `{ "phone", "client_id", "provider_id" }`.
- `POST /login/sms/verify` — `SMSLoginVerifyDTO` `{ "phone", "otp", "client_id", "provider_id" }` → `LoginResponseDTO`.

### 7.4 Step-up authentication (`acr=2`) and sensitive actions
Sensitive operations require a **fresh** step-up: an access token with `acr=2` issued within the last **5 minutes**. This guards: all `PUT /security-settings/*`, MFA self-service (enroll/disable/reset), email change (when `require_mfa_for_sensitive_actions` and the user has MFA), session termination, and admin actions.
When not satisfied, the API returns:
```jsonc
{ "success": false, "error": "Step-up authentication required", "code": "step_up_required" }
```
HTTP `403`. Frontend should branch on `code === "step_up_required"`, run an MFA step-up ceremony to obtain a fresh `acr=2` token, then retry. (An expired step-up returns the same code with a "expired; please re-authenticate" message.)

### 7.5 Token refresh & rotation
`POST /api/v1/refresh-token` — `RefreshTokenRequestDTO` `{ "refresh_token" }` (or via the refresh cookie). Returns a fresh `LoginResponseDTO` with a **rotated** refresh token. Rotation is single-use; the consumed token is denylisted. Refresh tokens carry a family id (`rfid`): replaying an already-consumed token outside the `refresh_token_reuse_interval_seconds` grace window **revokes the entire family** (forces re-login) and logs a HIGH-severity event. Idle/absolute session limits still apply across refreshes.

### 7.6 Registration
`POST /register` (query `client_id`,`provider_id`) — `RegisterRequestDTO`:
```json
{ "username","fullname","email"?,"phone"?,"password","captcha_token"? }
```
Enforces the §5.6 registration policy (self-reg gate, domains, captcha, rate limit, password policy). On success returns `RegisterResponseDTO` (token set). If email verification is required, the account is created `pending` and login is blocked until verified.

### 7.7 Email verification
- `POST /email-verification/send` — `{ "email" }` → `{ "message","success" }`.
- `POST /email-verification/verify` — `{ "email","otp" }` → `{ "message","success" }`; flips the account to `active`.

### 7.8 Password reset
`POST /reset-password` (signed-URL query: `token`,`client_id`,`provider_id`,`expires`,`sig`) — `ResetPasswordRequestDTO` `{ "new_password" }`. Enforces the full password policy + history reuse rejection, updates `PasswordChangedAt`, and revokes existing sessions when `revoke_sessions_on_password_change`. Returns `{ "message","success" }`.

### 7.9 Machine-readable error codes summary
| `code` | HTTP | Meaning / frontend action |
|--------|------|---------------------------|
| `step_up_required` | 403 | Obtain fresh `acr=2` then retry |
| (none) `"login blocked by threat detection"` | 401 | Risk-blocked; surface generic failure |
| (none) lockout message | 401/429-style | Account locked; show generic retry-later |
| (none) `"Invalid request body"` | 400 | Field/validation error (see `details`) |

---

## 8. Default seed (every new tenant)

`internal/secpolicy/defaults_setting.go` is the single source of truth; the seeder (`011_security_setting.go`) writes all seven blocks at tenant creation and idempotently backfills any empty/missing keys for existing tenants. The seeded values are the **Business SaaS** baseline (the defaults listed per block in §5). `risk_based_step_up_enabled` and `ip_reputation_check_enabled` seed `false` pending the external integrations in §5.7. Stricter postures (healthcare/finance, high-security) are set per tenant via the §3 API.

---

## 9. Standards & compliance

| Area | Standards satisfied |
|------|---------------------|
| Passwords | **NIST SP 800-63B** (length-over-composition, no forced rotation by default, breach screening), **OWASP ASVS v4 §2.1**, **PCI DSS v4.0 §8.3** (12-char min available), OWASP Password Storage (argon2id/bcrypt/scrypt/pbkdf2 at/above OWASP params) |
| MFA | **NIST 800-63B** AAL2/AAL3 (TOTP RFC 6238, WebAuthn/FIDO2; SMS restricted & off by default), OWASP MFA Cheat Sheet |
| Sessions/Tokens | **OWASP ASVS v4 §3** (idle/absolute timeouts, rotation, revocation), **OAuth 2.0 Security BCP / OAuth 2.1** (short access tokens, refresh rotation + reuse detection, PKCE S256), **RFC 7519/9068** (JWT), **OIDC Core 1.0** |
| Lockout | **NIST 800-63B §5.2.2** (throttling, ≤100 cap), **OWASP** (3–5 attempts), **PCI DSS §8.3.4** (≥30-min lockout) |
| Registration | **NIST 800-63A** (IAL1 + email verification), **GDPR** data-minimization/consent, disposable-domain blocking |
| Threat detection | **MITRE ATT&CK** T1110 (brute force) / T1078 (valid accounts), **NIST CSF** Detect, **OWASP** Credential Stuffing Prevention |
| Audit | Every config change recorded in `security_settings_audit` (old/new, actor, IP, UA) for SOC 2 / ISO 27001 change tracking |

Per-config deep-dive rationale (standards background, protocol detail) lives in [../settings/security-settings/](../settings/security-settings/README.md).

---

## 10. Extending / upgrading (for future changes)

To add a field to a block: (1) add it to the block's typed DTO in `types.go`; (2) add its validation rule in `validation_setting.go`; (3) add its default in `defaults_setting.go`; (4) consume it in the relevant policy loader + enforcement path; (5) document it here. The typed-DTO + `DisallowUnknownFields` contract means nothing reaches storage unvalidated and the schema evolves in one place.

To add a new client override: add a nullable column to the `clients` migration/model, map it in `config_mapping.go`, and clamp it tighten-only inside the relevant resolver in `validation_setting.go`.

Known accepted upgrades (not bugs): full GeoIP impossible-travel, IP-reputation feed, Tor-exit blocking (§5.7); enlarge the common-password blocklist toward the full SecLists 10k (`common_passwords.txt`).

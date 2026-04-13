# Password Configuration

> **Scope**: User Pool · **API Prefix**: `/security-settings/password` · **Storage**: `security_settings.password_config` (JSONB)

## Overview

Password configuration defines the rules that govern how passwords are created, stored, validated, and rotated across the user pool. As the most common authenticator, the password policy directly determines the baseline security posture of every user account.

A well-designed password policy balances **security** (resist brute-force, credential stuffing, and password spraying) against **usability** (users should be able to create and remember strong passwords without excessive friction).

---

## Industry Standards & Background

### The Evolution of Password Policy

Password guidance has changed dramatically over the past decade. Legacy policies (uppercase + lowercase + number + symbol, rotate every 90 days) have been superseded by evidence-based recommendations:

- **2017**: NIST SP 800-63B drops composition rules and periodic rotation, recommending length-based policies and breach-checking instead.
- **2019**: Microsoft publishes "Your Password Doesn't Matter" — argues MFA matters more than password complexity.
- **2020**: OWASP ASVS v4 aligns with NIST, dropping composition requirements.
- **2023**: PCI DSS v4.0 still requires complexity + 90-day rotation for payment systems (a notable divergence from NIST).

### Relevant Standards

| Standard | Reference | Key Guidance |
|----------|-----------|-------------|
| **NIST SP 800-63B** | Digital Identity Guidelines: Authentication & Lifecycle Management | The definitive modern password standard. |
| **OWASP ASVS v4** | V2.1 — Password Security | Application-level password requirements. |
| **OWASP Authentication Cheat Sheet** | Password Strength Controls | Practical implementation guidance. |
| **PCI DSS v4.0** | Requirement 8.3 | Password controls for payment card environments. |
| **CIS Password Policy Guide** | CIS Benchmarks | Enterprise password policy baselines. |
| **GDPR Art. 32** | Security of Processing | Mandates "appropriate technical measures" — strong passwords are one component. |
| **ISO 27001:2022** | A.5.17 — Authentication Information | Requires controls over the quality of authentication secrets. |
| **FIPS 140-3** | Cryptographic Module Validation | Governs the hashing algorithms used for password storage. |

### NIST SP 800-63B — Detailed Requirements

NIST 800-63B is the most influential modern password standard. Key provisions:

#### Length Requirements
- **Minimum 8 characters** when combined with MFA (Authenticator Assurance Level 2).
- **Minimum 15 characters** when password is the sole authenticator (AAL1 without MFA).
- **Maximum at least 64 characters** — the system must accept passwords up to at least 64 chars. Longer is better.
- No maximum below 64 characters. Passphrases like "correct horse battery staple" must be supported.

#### Composition Rules — Removed
- **Do NOT require** uppercase, lowercase, numbers, or special characters.
- Evidence shows composition rules lead to predictable patterns (`Password1!`, `P@ssw0rd`).
- Users should be free to create any password meeting the length requirement.

#### Character Sets
- **Allow all printable ASCII** plus space.
- **Should accept Unicode** (accented characters, emoji, CJK). At minimum, do not reject them.
- **Do not silently truncate** — if the max is 64, reject (not truncate) passwords longer than 64.

#### Breach Checking
- **Check against known breached passwords** — NIST explicitly recommends this.
- The HaveIBeenPwned Passwords API is the de facto standard (10+ billion entries).
- Use the k-Anonymity model (send only first 5 hex chars of SHA-1 hash) for privacy.

#### Rotation Policy
- **Do NOT require periodic password changes** unless there is evidence of compromise.
- Forced rotation causes users to pick weaker, predictable passwords (`Spring2024!` → `Summer2024!`).
- **DO require** change when a breach is detected or when the password is known compromised.

#### Password History
- If the system requires password changes (e.g., on compromise), prevent reuse of the **last N** passwords.
- NIST doesn't specify N, but CIS recommends 24 and PCI requires at least 4.

### OWASP ASVS v4 — Password Requirements

| ASVS Req | Description |
|----------|-------------|
| V2.1.1 | Minimum 8 characters |
| V2.1.2 | Maximum at least 128 characters |
| V2.1.3 | No truncation of passwords |
| V2.1.4 | Allow any printable Unicode character including spaces |
| V2.1.5 | Users can change their own password |
| V2.1.6 | Password change requires current password |
| V2.1.7 | Passwords checked against at least 10,000 most common passwords |
| V2.1.8 | Password strength meter to encourage stronger passwords |
| V2.1.9 | No password composition rules |
| V2.1.10 | No periodic credential rotation requirements |
| V2.1.11 | Allow paste into password fields |
| V2.1.12 | Users can view their entered password (via toggle) |

### PCI DSS v4.0 — Divergent Requirements

PCI DSS maintains stricter (and more traditional) rules:
- Minimum **12 characters** (or 8 with compensating controls)
- **Require both numeric and alphabetic** characters
- **Change every 90 days** maximum
- **Password history of 4** — cannot reuse last 4 passwords
- Account lockout after **10 failed attempts** (covered in lockout config)

### CIS Benchmark Recommendations

- Minimum **14 characters**
- Password history of **24**
- Maximum age **365 days** (more lenient than PCI)
- No specific composition rules (aligned with NIST)

### Comparison Matrix

| Requirement | NIST 800-63B | OWASP ASVS v4 | PCI DSS v4.0 | CIS Benchmark |
|-------------|:------------:|:--------------:|:------------:|:-------------:|
| Min length | 8 (w/ MFA) / 15 | 8 | 12 | 14 |
| Max length | ≥ 64 | ≥ 128 | — | — |
| Composition rules | ❌ No | ❌ No | ✅ Alpha + Numeric | ❌ No |
| Breach check | ✅ Required | ✅ Top 10K | — | — |
| Periodic rotation | ❌ Only on compromise | ❌ No | ✅ Every 90 days | ✅ Every 365 days |
| Password history | If rotating | — | Last 4 | Last 24 |

### Password Storage — How It Should Be Done

This is about how the system stores passwords, not about what this config controls — but it's critical context:

| Algorithm | Status | Notes |
|-----------|--------|-------|
| **Argon2id** | Recommended (OWASP #1) | Memory-hard, resists GPU attacks. Params: 19 MiB, 2 iterations, 1 parallelism. |
| **bcrypt** | Acceptable | Work factor ≥ 10. Max input 72 bytes — truncates silently. |
| **scrypt** | Acceptable | CPU + memory hard. Less common than Argon2id. |
| **PBKDF2** | Legacy acceptable | NIST approved. ≥ 600,000 iterations with SHA-256. Slower than Argon2id. |
| **SHA-256/MD5** | ❌ Never | No key stretching. Trivially crackable. |

### Temporary Passwords

Temporary passwords (admin-generated, invite-based, or first-login) should:
- Expire within a short window (e.g., 24–72 hours)
- Force the user to set a permanent password on first use
- Be generated cryptographically (not predictable)
- Be transmitted securely (one-time link, not plaintext in email body)

---

## Our Implementation

### Data Model

The password configuration is stored as a JSONB column in the `security_settings` table, scoped to a user pool.

```
security_settings
├── security_setting_id (PK, SERIAL)
├── security_setting_uuid (UUID, UNIQUE)
├── user_pool_id (FK → user_pools)
├── password_config (JSONB, default '{}')  ← THIS CONFIG
├── ... (other configs)
├── version (INTEGER, auto-incremented on change)
├── created_by / updated_by (FK → users)
└── created_at / updated_at (TIMESTAMPTZ)
```

**JSONB fields** (schema-free — values are `map[string]any`):

| Field | Type | Description |
|-------|------|-------------|
| `min_length` | int | Minimum password length |
| `max_length` | int | Maximum password length |
| `require_uppercase` | bool | Require at least one uppercase letter |
| `require_lowercase` | bool | Require at least one lowercase letter |
| `require_number` | bool | Require at least one digit |
| `require_symbol` | bool | Require at least one special character |
| `reject_common_passwords` | bool | Check against common password list |
| `check_hibp` | bool | Check against HaveIBeenPwned API |
| `password_history_count` | int | Number of previous passwords to remember |
| `max_age_days` | int | Maximum password age before forced change |
| `temporary_password_validity_hours` | int | How long a temporary password remains valid |

### API Endpoints

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `GET` | `/security-settings/password` | `GetPasswordConfig` | Returns the current password configuration |
| `PUT` | `/security-settings/password` | `UpdatePasswordConfig` | Replaces the password configuration |

**Request body** (PUT): Any JSON object — stored as-is in the JSONB column.
```json
{
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
  "temporary_password_validity_hours": 72
}
```

**Response**: The full `SecuritySettingServiceDataResult` including all seven config sections, version, and audit fields.

### Service Layer

- **`GetPasswordConfig(ctx, userPoolID)`** — Calls `getOrCreateSecuritySetting` (lazy-creates the row if it doesn't exist), then unmarshals the `password_config` JSONB column into `map[string]any`.
- **`UpdatePasswordConfig(ctx, userPoolID, config, updatedBy, ipAddress, userAgent)`** — Calls the shared `updateConfig` helper with `configType = "password"`. Inside a transaction, it:
  1. Finds or creates the security setting row
  2. Sets the `password_config` column
  3. Increments the `version` counter
  4. Creates a `SecuritySettingsAudit` record with `change_type = "update_password_config"`, capturing old config, new config, IP address, and user agent

### Audit Trail

Every update creates a row in `security_settings_audit`:

| Field | Value |
|-------|-------|
| `change_type` | `"update_password_config"` |
| `old_config` | Previous JSONB value |
| `new_config` | New JSONB value |
| `ip_address` | Admin's IP address |
| `user_agent` | Admin's browser/client |
| `created_by` | Admin user ID |

### Validation

Currently minimal — the DTO only validates that the config map is non-empty. No field-level validation exists yet.

**Source files:**
- Model: `internal/model/security_setting.go`
- Audit Model: `internal/model/security_settings_audit.go`
- DTO: `internal/dto/security_setting.go`
- Service: `internal/service/security_setting.go`
- Handler: `internal/rest/security_setting_handler.go`
- Routes: `internal/rest/security_setting_routes.go`
- Repository: `internal/repository/security_setting.go`
- Audit Repository: `internal/repository/security_settings_audit.go`
- Migration: `internal/database/migration/037_create_security_settings_table.go`
- Audit Migration: `internal/database/migration/039_create_security_settings_audit_table.go`

---

## Requirements Checklist

### Configuration Management (Admin API)
- [x] Get password config via `GET /security-settings/password`
- [x] Update password config via `PUT /security-settings/password`
- [x] Stored as JSONB for flexible schema evolution
- [x] Version tracking (auto-incremented on each update)
- [x] Audit trail (old/new config, who changed, IP, user agent)
- [x] Lazy-create on first access (no manual initialization required)
- [x] OpenTelemetry span tracing on all operations
- [x] Unit tests for service layer
- [x] Unit tests for handler layer
- [ ] JSONB schema validation on update (reject unknown fields, enforce types)
- [ ] Default values on initial creation (currently `{}`)
- [ ] Validation: `min_length` must be ≥ 1 and ≤ `max_length`
- [ ] Validation: `max_length` must be ≤ 128 (OWASP max)
- [ ] Validation: `password_history_count` must be ≥ 0
- [ ] Validation: `max_age_days` must be ≥ 0
- [ ] Validation: `temporary_password_validity_hours` must be ≥ 1

### Length Enforcement
- [ ] Enforce `min_length` on user registration
- [ ] Enforce `min_length` on password change
- [ ] Enforce `min_length` on admin-initiated password reset
- [ ] Enforce `max_length` — reject (not truncate) passwords exceeding max
- [ ] Enforce `max_length` ≥ 64 minimum to comply with NIST
- [ ] Support Unicode characters in passwords (NIST requirement)
- [ ] Do not silently truncate (OWASP V2.1.3)

### Complexity Rules (Optional, Configurable)
- [ ] Enforce `require_uppercase` when enabled
- [ ] Enforce `require_lowercase` when enabled
- [ ] Enforce `require_number` when enabled
- [ ] Enforce `require_symbol` when enabled
- [ ] Allow disabling all composition rules (NIST-aligned default)
- [ ] Display active requirements to user on registration/change forms

### Breach & Common Password Checking
- [ ] Reject common passwords from a local list (top 100K) when `reject_common_passwords` is true
- [ ] Common password list bundled with the application (embedded or file-based)
- [ ] HaveIBeenPwned API integration when `check_hibp` is true
- [ ] Use k-Anonymity model (send first 5 hex chars of SHA-1, compare locally)
- [ ] Handle HIBP API unavailability gracefully (fail-open with warning, not fail-closed)
- [ ] Cache HIBP responses to reduce API calls
- [ ] Rate limit HIBP API calls to respect their fair-use policy

### Password History
- [ ] Store hashed history of last N passwords (per `password_history_count`)
- [ ] History stored in a dedicated table (not in the config JSONB)
- [ ] Hash with the same algorithm as the current password
- [ ] Reject password change if new password matches any of the last N
- [ ] Prune history entries older than N (keep only the configured count)

### Password Aging & Rotation
- [ ] Track password last-changed date per user
- [ ] Enforce max age on login (redirect to password-change if expired) when `max_age_days > 0`
- [ ] Grace period before hard-lock (warn user for N days before expiry)
- [ ] Send email notification N days before password expiry
- [ ] Do NOT enforce periodic rotation when `max_age_days = 0` (NIST-aligned default)
- [ ] Force password change on detected breach (regardless of `max_age_days`)

### Temporary Passwords
- [ ] Admin can generate a temporary password for a user
- [ ] Temporary password expires after `temporary_password_validity_hours`
- [ ] User must set a permanent password on first login with temp password
- [ ] Temporary password generated cryptographically (CSPRNG, min 12 chars)
- [ ] Temporary password flag stored on user record

### Password Strength Estimation
- [ ] Server-side password strength scoring (zxcvbn-like)
- [ ] Return strength score + feedback on registration/change API
- [ ] Configurable minimum strength score threshold
- [ ] Reject passwords below minimum strength even if they meet length requirements

### Password Change Flow
- [ ] Require current password verification on password change (OWASP V2.1.6)
- [ ] Invalidate all existing sessions after password change (except current)
- [ ] Send notification email on password change
- [ ] Log password change event for audit

### Integration & Testing
- [ ] Integration test: password policy enforcement on registration endpoint
- [ ] Integration test: password policy enforcement on password change endpoint
- [ ] Integration test: password history enforcement
- [ ] Integration test: HIBP API integration (with mock)
- [ ] Integration test: audit trail recording on config change
- [ ] Load test: HIBP check latency impact on registration flow

---

## References

- [NIST SP 800-63B — Digital Identity Guidelines: Authentication and Lifecycle Management](https://pages.nist.gov/800-63-3/sp800-63b.html)
- [OWASP ASVS v4.0 — Password Security](https://owasp.org/www-project-application-security-verification-standard/)
- [OWASP Authentication Cheat Sheet — Password Strength Controls](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html#implement-proper-password-strength-controls)
- [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
- [PCI DSS v4.0 — Requirement 8.3](https://www.pcisecuritystandards.org/)
- [CIS Password Policy Guide](https://www.cisecurity.org/insights/white-papers/cis-password-policy-guide)
- [HaveIBeenPwned Passwords API](https://haveibeenpwned.com/API/v3#PwnedPasswords)
- [Microsoft: Your Password Doesn't Matter](https://techcommunity.microsoft.com/t5/Azure-Active-Directory-Identity/Your-Pa-word-doesn-t-matter/ba-p/731984)
- [Dropbox zxcvbn — Realistic Password Strength Estimation](https://github.com/dropbox/zxcvbn)

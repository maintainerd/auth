# Account Lockout Configuration

> **Scope**: User Pool · **API Prefix**: `/security-settings/lockout` · **Storage**: `security_settings.lockout_config` (JSONB)

## Overview

Account lockout configuration controls the system's response to repeated failed authentication attempts. This is the primary defense against **brute-force** and **credential stuffing** attacks — automated attacks that try thousands of password combinations against user accounts.

The core tension in lockout design:
- **Too aggressive** → Attackers can deliberately lock out legitimate users (denial-of-service)
- **Too lenient** → Attackers have more time and attempts to guess passwords
- **Progressive** → Best of both worlds: gentle at first, severe for persistent attacks

---

## Industry Standards & Background

### Attack Types That Lockout Defends Against

| Attack | Description | Scale | Lockout Effectiveness |
|--------|-------------|-------|----------------------|
| **Brute force** | Try all possible passwords against one account | Single account | ✅ High — stops after N attempts |
| **Credential stuffing** | Use stolen credential pairs from breaches against your service | Thousands of accounts | ⚠️ Medium — per-account lockout helps, but attackers distribute across accounts |
| **Password spraying** | Try a few common passwords against many accounts | Many accounts, few attempts each | ❌ Low — stays under per-account threshold. Rate limiting by IP is more effective |
| **Reverse brute force** | Try one password against many accounts | Many accounts, 1 attempt each | ❌ None — never triggers account lockout. Need IP/velocity detection (threat config) |

**Key insight:** Account lockout is effective against brute force but insufficient against distributed attacks. It must work in conjunction with threat detection (see [Threat Configuration](threat-config.md)).

### Relevant Standards

| Standard | Reference | Key Guidance |
|----------|-----------|-------------|
| **NIST SP 800-63B §5.2.2** | Throttling Mechanisms | Limit effectiveness of online guessing. 100 consecutive failed attempts per account is the absolute upper bound. Prefer throttling (delays) over hard lockout. |
| **OWASP Authentication Cheat Sheet** | Account Lockout | 3–5 failed attempts before lockout. Lockout duration: 30 minutes or increasing. Consider progressive lockout. |
| **OWASP ASVS v4** | V2.2.1 | Anti-automation controls for credential recovery and login |
| **PCI DSS v4.0 Req 8.3.4** | Lockout | Lock out after **no more than 10** failed attempts. Lockout for minimum 30 minutes or until admin unlocks. |
| **CIS Controls v8** | Control 4.10 | Configure lockout after a maximum of 5 failed attempts |
| **MITRE ATT&CK** | T1110 | Brute Force technique — lockout is a primary mitigation |
| **ISO 27001:2022** | A.8.5 | Secure authentication including lockout mechanisms |

### NIST SP 800-63B — Throttling vs. Hard Lockout

NIST's approach has evolved significantly:

**Old approach (pre-2017):** Lock the account after N failures. Require admin unlock or wait.

**Current NIST recommendation:** Prefer **throttling** (increasing delays) over hard lockout:

| After Failures | NIST Recommendation |
|---------------|---------------------|
| 1–3 | No delay |
| 4–10 | Increasingly longer delays between attempts (exponential backoff) |
| 10+ | Require CAPTCHA or secondary verification |
| 100 | Account must be locked (hard cap) |

**Why throttling over lockout:**
- Hard lockout enables **lock-out attacks** — deliberately failing to lock a victim's account
- Throttling slows attackers without denying legitimate users access
- Rate limiting by IP address catches distributed attacks

### OWASP Authentication Cheat Sheet — Lockout Guidance

OWASP recommends a practical middle ground:

> "Lock accounts after 3-5 failed login attempts. Use a lockout duration of at least 30 minutes. Consider progressive lockout where the duration increases with each lockout."

**Additional OWASP guidance:**
- Log all failed authentication attempts
- Alert on unusual patterns (many accounts being locked)
- Do not reveal whether the account exists ("wrong username or password")
- Reset the failed attempt counter on successful login
- Consider account lockout notification (email the user)

### PCI DSS v4.0 Requirement 8.3.4

PCI is the most prescriptive:

> "If a user account is locked out due to failed authentication attempts, the lockout is for a minimum of 30 minutes or until the user's identity is confirmed by an administrator."

- Maximum 10 failed attempts before lockout
- Minimum 30-minute lockout duration
- Admin override allowed
- Applies to all accounts with access to cardholder data

### Progressive Lockout Explained

Progressive lockout increases severity with each violation:

```
Attempt 1-3:   No delay, normal login
Attempt 4:     5-second delay before next attempt
Attempt 5:     Account locked for 1 minute
Attempt 6-7:   Account locked for 5 minutes
Attempt 8-10:  Account locked for 15 minutes
Attempt 11+:   Account locked for 30 minutes
                (cap to prevent indefinite lockout)
```

**Benefits:**
- Legitimate users who mistype 3 times aren't affected
- Automated attacks are slowed exponentially
- Persistent attackers face longer lockout without admin intervention
- Reduces lock-out attack effectiveness (eventually unlocks)

### Lock-Out Attack Mitigation

An attacker who knows a username can deliberately trigger lockout (denial of service). Mitigations:

| Mitigation | How It Works |
|-----------|-------------|
| **Auto-unlock after duration** | Account unlocks after `lockout_duration_minutes` — attacker must continuously attack |
| **CAPTCHA on lockout** | Instead of hard lock, show CAPTCHA — legitimate user can solve it |
| **Progressive duration** | First lockout is short, making DoS less effective |
| **IP-based lockout (not account)** | Lock the IP, not the account — attacker just locked themselves out |
| **Invisible lockout** | Account appears locked but responds "wrong password" — attacker doesn't know |
| **Notification** | Email the account owner — they know someone is attacking their account |

---

## Our Implementation

### Data Model

Stored in `security_settings.lockout_config` (JSONB column). The config is schema-free (`map[string]any`).

**Expected JSONB fields:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | true | Whether account lockout is enabled |
| `max_failed_attempts` | int | 5 | Consecutive failed attempts before lockout |
| `lockout_duration_minutes` | int | 30 | Duration of account lockout |
| `progressive_lockout` | bool | false | Whether lockout duration increases with repeated violations |
| `auto_unlock` | bool | true | Whether accounts auto-unlock after `lockout_duration_minutes` |
| `reset_count_on_success` | bool | true | Reset failed attempt counter on successful login |

### API Endpoints

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `GET` | `/security-settings/lockout` | `GetLockoutConfig` | Returns the current lockout configuration |
| `PUT` | `/security-settings/lockout` | `UpdateLockoutConfig` | Replaces the lockout configuration |

**Example request body (PUT):**
```json
{
  "enabled": true,
  "max_failed_attempts": 5,
  "lockout_duration_minutes": 30,
  "progressive_lockout": true,
  "auto_unlock": true,
  "reset_count_on_success": true
}
```

### Service Layer

- **`GetLockoutConfig(ctx, userPoolID)`** — Lazy-creates the security setting row, then returns the `lockout_config` JSONB.
- **`UpdateLockoutConfig(ctx, userPoolID, config, updatedBy, ipAddress, userAgent)`** — Calls `updateConfig` with `"lockout"` config type.

The update runs in a transaction:
1. Find or create `security_settings` row for the user pool
2. Replace the `lockout_config` JSONB column
3. Increment `version`
4. Create a `security_settings_audit` row with `change_type: "update_lockout_config"`

### Audit Trail

Every update creates a `security_settings_audit` row:
- `change_type`: `"update_lockout_config"`
- `old_config`: Previous lockout JSONB
- `new_config`: New lockout JSONB
- `ip_address`, `user_agent`, `created_by`: Admin context

### Validation

Currently minimal — only validates the config map is non-empty. No schema validation.

**Source files:**
- Model: `internal/model/security_setting.go`
- Service: `internal/service/security_setting.go`
- Handler: `internal/rest/security_setting_handler.go`
- Routes: `internal/rest/security_setting_routes.go`
- DTO: `internal/dto/security_setting.go`

---

## Recommended Defaults by Application Type

| Setting | Consumer App | Business SaaS | Healthcare/Finance | High Security |
|---------|:------------:|:-------------:|:-----------------:|:-------------:|
| `enabled` | true | true | true | true |
| `max_failed_attempts` | 10 | 5 | 5 | 3 |
| `lockout_duration_minutes` | 15 | 30 | 30 | 60 |
| `progressive_lockout` | false | true | true | true |
| `auto_unlock` | true | true | true | false (admin only) |
| `reset_count_on_success` | true | true | true | true |

---

## Requirements Checklist

### Configuration Management (Admin API)
- [x] Get lockout config via `GET /security-settings/lockout`
- [x] Update lockout config via `PUT /security-settings/lockout`
- [x] Stored as JSONB for flexible schema evolution
- [x] Version tracking (auto-incremented on each update)
- [x] Audit trail (old/new config, who changed, IP, user agent)
- [x] Lazy-create on first access
- [x] OpenTelemetry span tracing
- [x] Unit tests for service layer
- [x] Unit tests for handler layer
- [ ] Validation: `max_failed_attempts` must be 1–100 (NIST upper bound)
- [ ] Validation: `lockout_duration_minutes` must be ≥ 1 (PCI: ≥ 30)
- [ ] Validation: `auto_unlock` and `lockout_duration_minutes` must be coherent (if `auto_unlock` is false, duration is irrelevant)
- [ ] Sane defaults on creation

### Account Lockout Enforcement
- [ ] Track failed login attempts per user
- [ ] Increment counter on authentication failure
- [ ] Lock account when counter reaches `max_failed_attempts`
- [ ] Locked account returns generic "Invalid credentials" (not "account locked")
- [ ] Locked account cannot authenticate even with correct password
- [ ] Auto-unlock after `lockout_duration_minutes` when `auto_unlock` is true
- [ ] Do not auto-unlock when `auto_unlock` is false (admin must unlock)
- [ ] Reset counter on successful login when `reset_count_on_success` is true

### Progressive Lockout
- [ ] Implement escalating lockout durations when `progressive_lockout` is true
- [ ] Define progression tiers (e.g., 1 min → 5 min → 15 min → 30 min → 60 min cap)
- [ ] Track lockout count per user (separate from failed attempt count)
- [ ] Reset lockout progression after a configurable cool-down period (e.g., 24 hours)
- [ ] Cap maximum lockout duration to prevent indefinite lockout

### Admin Tools
- [ ] Admin can view locked accounts
- [ ] Admin can manually unlock an account
- [ ] Admin can view failed attempt history for an account
- [ ] Admin can reset failed attempt counter for an account
- [ ] Bulk unlock (unlock all currently locked accounts)

### Lock-Out Attack Mitigation
- [ ] CAPTCHA integration as an alternative to hard lockout
- [ ] IP-based rate limiting in conjunction with account lockout
- [ ] Lockout notification email to the account owner
- [ ] Monitor for mass lockout patterns (many accounts locked simultaneously)

### Logging & Alerting
- [ ] Log every failed authentication attempt (user, IP, timestamp, user agent)
- [ ] Log account lockout events
- [ ] Log account unlock events (auto-unlock and admin-unlock)
- [ ] Alert when lockout rate exceeds threshold (potential attack)
- [ ] Alert when a single IP causes multiple account lockouts

### Integration with Threat Detection
- [ ] Coordinate with `threat_config` brute force detection
- [ ] IP-based lockout for distributed attacks (credential stuffing)
- [ ] Velocity checks across accounts from same IP
- [ ] Feed lockout events into risk scoring

### Integration & Testing
- [ ] Unit tests for lockout config service methods
- [ ] Integration test: account locks after N failures
- [ ] Integration test: auto-unlock after duration
- [ ] Integration test: progressive lockout escalation
- [ ] Integration test: counter reset on success
- [ ] Integration test: admin unlock
- [ ] Integration test: audit trail with lockout config changes

---

## References

- [NIST SP 800-63B §5.2.2 — Throttling Mechanisms](https://pages.nist.gov/800-63-3/sp800-63b.html#throttle)
- [OWASP Authentication Cheat Sheet — Account Lockout](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html#account-lockout)
- [OWASP Credential Stuffing Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Credential_Stuffing_Prevention_Cheat_Sheet.html)
- [OWASP ASVS v4.0 — V2.2 General Authenticator Security](https://owasp.org/www-project-application-security-verification-standard/)
- [PCI DSS v4.0 Requirement 8.3.4](https://www.pcisecuritystandards.org/)
- [CIS Controls v8 — Control 4.10](https://www.cisecurity.org/controls/)
- [MITRE ATT&CK T1110 — Brute Force](https://attack.mitre.org/techniques/T1110/)

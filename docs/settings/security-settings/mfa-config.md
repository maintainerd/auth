# Multi-Factor Authentication (MFA) Configuration

> **Scope**: User Pool · **API Prefix**: `/security-settings/mfa` · **Storage**: `security_settings.mfa_config` (JSONB)

## Overview

MFA configuration controls whether and how users are required to provide a second authentication factor beyond their password. MFA is the single most effective defense against credential-based attacks — Microsoft estimates it blocks **99.9% of account compromises**.

This configuration governs:
- Whether MFA is disabled, optional, or enforced
- Which MFA methods are available (TOTP, SMS, email)
- Trusted device behavior (remember this device)
- Grace period for enforcement rollout

---

## Industry Standards & Background

### Why MFA Exists

Passwords alone are fundamentally insecure:
- **65%** of people reuse passwords across multiple sites (Google/Harris Poll 2019)
- **15 billion** credentials are available on the dark web (Digital Shadows 2020)
- **Credential stuffing** — automated attacks using breach data — accounts for the majority of login attacks
- **Password spraying** — trying a few common passwords against many accounts — bypasses per-account lockout

MFA adds a layer requiring something the attacker does not have: a physical device, a biometric, or a hardware key.

### Authentication Assurance Levels (NIST SP 800-63B)

NIST defines three assurance levels that map directly to MFA policy:

| Level | Requirements | Use Case |
|-------|-------------|----------|
| **AAL1** | Single factor (password only) | Low-risk, non-sensitive data |
| **AAL2** | Two factors (password + TOTP/SMS/push) | Most applications, personal data, financial |
| **AAL3** | Two factors, one must be hardware-bound (FIDO2) | Government, critical infrastructure, high-value targets |

**Key NIST guidance on SMS:**
- SMS is classified as a **"restricted"** authenticator
- Acceptable for AAL2 but with additional requirements
- Risk: SIM swapping (carrier social engineering) and SS7 protocol interception
- NIST recommends offering alternatives (TOTP, FIDO2) alongside SMS

### Relevant Standards

| Standard | Reference | Key Guidance |
|----------|-----------|-------------|
| **NIST SP 800-63B** | Digital Identity Guidelines | Defines AAL1/2/3. SMS is "restricted." TOTP and FIDO2 are preferred. |
| **RFC 6238** | TOTP: Time-Based One-Time Password | The algorithm: `HMAC-SHA1(secret, floor(time/30))` → 6-digit code. 30-second step. |
| **RFC 4226** | HOTP: HMAC-Based OTP | Counter-based OTP — the foundation RFC that TOTP extends. |
| **RFC 6287** | OCRA: OATH Challenge-Response Algorithm | Extension of HOTP/TOTP for challenge-response scenarios. |
| **W3C WebAuthn** | Web Authentication API | Browser API for FIDO2/Passkeys. Phishing-resistant, hardware-bound. |
| **FIDO2 / CTAP2** | Client-to-Authenticator Protocol | How authenticators (YubiKey, Face ID, Windows Hello) communicate. |
| **OWASP MFA Cheat Sheet** | Multifactor Authentication | Practical guidance: when to require, how to implement, recovery flows. |
| **OWASP ASVS v4** | V2.7 — Out-of-Band Verifier | OTP via SMS/email: time-limited, single-use, minimum 6 digits. |
| **PCI DSS v4.0** | Req 8.4 | MFA required for all personnel with administrative access to cardholder data. |
| **GDPR Art. 32** | Security of Processing | MFA is considered "appropriate technical measure" for personal data processing. |
| **ISO 27001:2022** | A.8.5 — Secure Authentication | Multi-factor authentication where appropriate based on risk assessment. |

### TOTP — How It Works (RFC 6238)

TOTP (Time-based One-Time Password) is the most common MFA method:

```
1. ENROLLMENT:
   Server generates a random 20-byte secret key
   Secret is encoded as Base32 and embedded in an otpauth:// URI
   URI is displayed as a QR code for the user to scan
   
   Format: otpauth://totp/{issuer}:{account}?secret={base32}&issuer={issuer}&digits=6&period=30

2. CODE GENERATION (every 30 seconds):
   T = floor(current_unix_time / 30)
   code = HMAC-SHA1(secret, T) → truncate to 6 digits

3. VERIFICATION:
   Server accepts codes from time windows: [T-1, T, T+1]
   This provides ±30 seconds clock skew tolerance
   Code is single-use — mark as consumed after successful verification
```

**Security properties:**
- Codes are valid for 30 seconds (configurable: 30–90 seconds)
- Codes are 6 digits (1,000,000 possibilities — secure given rate limiting + expiry)
- The shared secret never leaves the device after enrollment
- Works offline (no network required on the device)
- Supported by: Google Authenticator, Authy, Microsoft Authenticator, 1Password, Bitwarden, etc.

### SMS OTP — How It Works

```
1. ENROLLMENT:
   User provides phone number
   System sends a verification code to confirm number ownership
   
2. AUTHENTICATION:
   Server generates a random 6-8 digit code
   Code is sent via SMS (using the SMS config provider)
   User enters the code within the TTL window (typically 5-10 minutes)
   
3. SECURITY CONSIDERATIONS:
   - SIM swap attacks: attacker convinces carrier to port the number
   - SS7 vulnerabilities: network-level SMS interception
   - Message preview: code visible on lock screen
   - Same-device risk: SMS received on the device being authenticated
```

### Email OTP — How It Works

Similar to SMS OTP but delivered via email:
- Generally considered **weaker** than SMS (email passwords are often the same as app passwords)
- NIST does not classify email as a valid out-of-band authenticator for AAL2
- Useful as a fallback or for low-risk applications

### FIDO2 / WebAuthn / Passkeys — The Gold Standard

```
1. ENROLLMENT:
   Browser calls navigator.credentials.create()
   Authenticator generates a public/private key pair
   Public key is sent to the server and stored
   Private key stays on the device (TPM, Secure Enclave, YubiKey)

2. AUTHENTICATION:
   Server sends a random challenge
   Browser calls navigator.credentials.get()
   Authenticator signs the challenge with the private key
   Server verifies the signature with the stored public key

3. SECURITY PROPERTIES:
   - Phishing-resistant: the origin (domain) is bound to the credential
   - No shared secret: asymmetric cryptography
   - Hardware-bound: private key cannot be extracted
   - Passkeys: sync across devices via iCloud/Google Password Manager
```

### MFA Method Comparison — Detailed

| Criterion | TOTP | SMS OTP | Email OTP | FIDO2/Passkeys |
|-----------|:----:|:-------:|:---------:|:--------------:|
| **NIST AAL2** | ✅ | ⚠️ Restricted | ❌ | ✅ |
| **NIST AAL3** | ❌ | ❌ | ❌ | ✅ (hardware) |
| **Phishing-resistant** | ❌ | ❌ | ❌ | ✅ |
| **Works offline** | ✅ | ❌ | ❌ | ✅ |
| **User friction** | Low | Low | Medium | Very Low |
| **Setup complexity** | Medium (QR scan) | Low (phone #) | Low (already have email) | Medium (device prompt) |
| **Infrastructure cost** | None | Per-message | Included in email | None |
| **SIM swap risk** | ❌ None | ✅ High | ❌ None | ❌ None |
| **Same-device risk** | Low | Medium | Medium | None |
| **Recovery complexity** | Medium | Low | Low | High |
| **Enterprise adoption** | High | Declining | Low | Growing rapidly |

### Trusted Devices

After MFA verification, systems commonly offer "trust this device" to reduce friction:
- A secure cookie (HttpOnly, Secure, SameSite=Strict) is set with a unique device token
- On subsequent logins from the trusted device, MFA is skipped
- Trust expires after a configurable period (7–30 days typical)
- Trusted devices should be revocable by the user

### Grace Period for Enforcement Rollout

When enabling mandatory MFA for an existing user base:
- **Grace period**: Users can log in without MFA for N days (e.g., 14 days)
- During grace period: show a banner/notification urging MFA enrollment
- After grace period: block login until MFA is enrolled
- Admin users may have a shorter or zero grace period

### Recovery Codes

Recovery codes are a critical safety net:
- Generated at MFA enrollment time (typically 8–10 codes, 8 chars each)
- Each code is single-use
- Stored hashed (like passwords) — never shown again after generation
- Must be stored securely by the user (printed, password manager)
- Running out of recovery codes should prompt regeneration

---

## Our Implementation

### Data Model

Stored in `security_settings.mfa_config` (JSONB column). The config is schema-free (`map[string]any`).

**Expected JSONB fields:**

| Field | Type | Description |
|-------|------|-------------|
| `mode` | string | MFA policy mode: `disabled`, `optional`, `enforced` |
| `allowed_methods` | []string | Allowed MFA methods: `totp`, `sms`, `email_otp` |
| `totp_issuer` | string | Issuer name displayed in authenticator apps (e.g., "MyApp") |
| `trusted_device_period_days` | int | How many days a device remains trusted after MFA |
| `grace_period_days` | int | Days users have to enroll MFA after enforcement is enabled |

### API Endpoints

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `GET` | `/security-settings/mfa` | `GetMFAConfig` | Returns the current MFA configuration |
| `PUT` | `/security-settings/mfa` | `UpdateMFAConfig` | Replaces the MFA configuration |

**Example request body (PUT):**
```json
{
  "mode": "optional",
  "allowed_methods": ["totp", "sms"],
  "totp_issuer": "Maintainerd",
  "trusted_device_period_days": 14,
  "grace_period_days": 30
}
```

### Service Layer

- **`GetMFAConfig(ctx, userPoolID)`** — Lazy-creates the security setting row, then returns the `mfa_config` JSONB.
- **`UpdateMFAConfig(ctx, userPoolID, config, updatedBy, ipAddress, userAgent)`** — Calls `updateConfig` with the config type.

### ⚠️ Known Bug

`UpdateMFAConfig` passes `"general"` as the config type to the internal `updateConfig` method instead of `"mfa"`. This causes it to hit the `default` switch case, returning `apperror.NewValidation("invalid config type")`. **MFA config updates are broken until this is fixed.** The fix is a one-line change: `"general"` → `"mfa"`.

### Audit Trail

Every update creates a `security_settings_audit` row:
- `change_type`: `"update_mfa_config"` (once the bug is fixed)
- `old_config`: Previous MFA JSONB
- `new_config`: New MFA JSONB
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

## Requirements Checklist

### Configuration Management (Admin API)
- [x] Get MFA config via `GET /security-settings/mfa`
- [x] Update MFA config via `PUT /security-settings/mfa` (**⚠️ Bug: returns "invalid config type"**)
- [x] Stored as JSONB for flexible schema evolution
- [x] Version tracking (auto-incremented on each update)
- [x] Audit trail (old/new config, who changed, IP, user agent)
- [x] Lazy-create on first access
- [x] OpenTelemetry span tracing
- [x] Unit tests for service layer
- [x] Unit tests for handler layer
- [ ] **Fix bug**: `UpdateMFAConfig` passes `"general"` instead of `"mfa"` to `updateConfig`
- [ ] JSONB schema validation (validate `mode` enum, `allowed_methods` values)
- [ ] Default values on creation (`mode: "disabled"`, `allowed_methods: []`)
- [ ] Validation: `mode` must be `disabled`, `optional`, or `enforced`
- [ ] Validation: `allowed_methods` must contain only `totp`, `sms`, `email_otp`
- [ ] Validation: `totp_issuer` required when `totp` is in `allowed_methods`
- [ ] Validation: `trusted_device_period_days` must be > 0
- [ ] Validation: `grace_period_days` must be ≥ 0

### TOTP Enrollment & Verification
- [ ] Generate TOTP secret (20 bytes, cryptographically random)
- [ ] Generate `otpauth://` URI with issuer, account, secret
- [ ] QR code generation for the URI (or return URI for client-side rendering)
- [ ] Verify first TOTP code during enrollment (confirm secret was received)
- [ ] Store TOTP secret encrypted per user
- [ ] TOTP verification on login (accept T-1, T, T+1 windows)
- [ ] Rate limit TOTP verification attempts (5 per minute)
- [ ] Mark TOTP code as used (prevent replay within same 30s window)
- [ ] Support configurable TOTP parameters (6 vs 8 digits, 30 vs 60 second period)

### SMS OTP Enrollment & Verification
- [ ] Phone number entry and E.164 normalization
- [ ] Phone ownership verification (send code to number during enrollment)
- [ ] 6–8 digit OTP generation (CSPRNG)
- [ ] OTP delivery via SMS config provider
- [ ] OTP verification with configurable TTL (default 5 minutes)
- [ ] OTP single-use enforcement
- [ ] Rate limiting (max 5 SMS OTP per number per hour)
- [ ] SMS OTP fallback when TOTP is primary

### Email OTP Verification
- [ ] 6–8 digit OTP generation (CSPRNG)
- [ ] OTP delivery via email config
- [ ] OTP verification with configurable TTL (default 10 minutes)
- [ ] OTP single-use enforcement
- [ ] Rate limiting (max 5 email OTP per address per hour)

### FIDO2 / WebAuthn / Passkeys
- [ ] WebAuthn registration ceremony (navigator.credentials.create)
- [ ] WebAuthn authentication ceremony (navigator.credentials.get)
- [ ] Credential storage (public key, credential ID, sign count)
- [ ] Support multiple credentials per user
- [ ] Credential management (list, rename, delete)
- [ ] Passkey support (platform authenticators — Face ID, Windows Hello)
- [ ] Cross-device authentication (Bluetooth-mediated)
- [ ] Attestation verification (optional)

### Recovery Codes
- [ ] Generate 8–10 recovery codes on MFA enrollment
- [ ] Recovery codes: 8 alphanumeric characters each
- [ ] Store recovery codes hashed (bcrypt/argon2)
- [ ] Single-use: each code can only be used once
- [ ] Recovery code verification on login (MFA bypass)
- [ ] Recovery code regeneration (invalidate old, generate new)
- [ ] Show recovery codes only once at generation time
- [ ] Warn user when running low on recovery codes

### Trusted Device Management
- [ ] Set trusted device cookie on "Trust this device" consent
- [ ] Trusted device token: cryptographically random, stored hashed
- [ ] Respect `trusted_device_period_days` expiry
- [ ] Skip MFA for recognized trusted devices
- [ ] User can view their trusted devices
- [ ] User can revoke specific trusted devices
- [ ] Admin can revoke all trusted devices for a user
- [ ] Trusted device metadata (browser, OS, last used, IP)

### MFA Enforcement
- [ ] `disabled` mode: MFA not available, hide enrollment
- [ ] `optional` mode: Users can enroll voluntarily
- [ ] `enforced` mode: Users must enroll; block login without MFA after grace period
- [ ] Grace period enforcement during rollout
- [ ] Grace period banner/notification on login
- [ ] Separate enforcement for admin users (can be stricter)
- [ ] MFA bypass for service accounts / API keys
- [ ] MFA requirement for sensitive operations (password change, email change)

### MFA Lifecycle Management
- [ ] User can enroll a new MFA method
- [ ] User can disable an MFA method (with reauthentication)
- [ ] User can switch primary MFA method
- [ ] Admin can reset a user's MFA (emergency)
- [ ] Admin can force MFA enrollment for a specific user
- [ ] MFA enrollment webhook event
- [ ] MFA verification webhook event (success/failure)

### Integration & Testing
- [ ] Integration test: TOTP enrollment flow end-to-end
- [ ] Integration test: TOTP verification on login
- [ ] Integration test: SMS OTP flow
- [ ] Integration test: recovery code flow
- [ ] Integration test: trusted device flow
- [ ] Integration test: MFA enforcement mode transitions
- [ ] Integration test: grace period expiry behavior
- [ ] Integration test: audit trail with MFA config changes

---

## References

- [NIST SP 800-63B — Digital Identity Guidelines: Authentication](https://pages.nist.gov/800-63-3/sp800-63b.html)
- [RFC 6238 — TOTP: Time-Based One-Time Password Algorithm](https://datatracker.ietf.org/doc/html/rfc6238)
- [RFC 4226 — HOTP: HMAC-Based OTP Algorithm](https://datatracker.ietf.org/doc/html/rfc4226)
- [W3C Web Authentication (WebAuthn)](https://www.w3.org/TR/webauthn-2/)
- [FIDO2 Specifications](https://fidoalliance.org/fido2/)
- [OWASP Multifactor Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Multifactor_Authentication_Cheat_Sheet.html)
- [OWASP ASVS v4.0 — V2.7 Out-of-Band Verifier](https://owasp.org/www-project-application-security-verification-standard/)
- [Google: Your Password Doesn't Matter — 99.9%](https://techcommunity.microsoft.com/t5/Azure-Active-Directory-Identity/Your-Pa-word-doesn-t-matter/ba-p/731984)
- [Passkeys.dev — FIDO2/WebAuthn Resource](https://passkeys.dev/)

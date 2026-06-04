# Registration Configuration

> **Scope**: User Pool · **API Prefix**: `/security-settings/registration` · **Storage**: `security_settings.registration_config` (JSONB)

## Overview

Registration configuration controls how new user accounts are created — who can register, what verification is required, and what domains are allowed or blocked. This is the first line of defense in the user lifecycle: getting registration wrong means everything downstream (authentication, authorization, MFA) is built on a compromised foundation.

Registration policy answers three fundamental questions:
1. **Who can create an account?** (self-registration vs. invite-only)
2. **How do we verify identity?** (email verification, phone verification, admin approval)
3. **What restrictions exist?** (domain allowlists, blocklists, auto-confirmation)

---

## Industry Standards & Background

### Identity Proofing Assurance Levels (NIST SP 800-63A)

NIST defines three levels of identity assurance — how confident are you that the person registering is who they claim to be?

| Level | Requirements | Evidence | Use Case |
|-------|-------------|----------|----------|
| **IAL1** | Self-asserted identity, no verification | None | Consumer apps where real identity doesn't matter |
| **IAL2** | Remote identity proofing — verify government-issued ID, knowledge-based verification, or biometric | Photo ID, address verification, credit file check | Financial services, healthcare, regulated industries |
| **IAL3** | In-person identity proofing — physical presence required | In-person verification with biometric capture | Government, law enforcement, critical infrastructure |

**Most SaaS applications operate at IAL1** — self-registration with email verification. This configuration supports IAL1 with hooks for IAL2 (domain restrictions, admin approval).

### Relevant Standards

| Standard | Reference | Key Guidance |
|----------|-----------|-------------|
| **NIST SP 800-63A** | Identity Proofing | IAL1/2/3 levels. Self-asserted identity acceptable for many use cases. |
| **OWASP ASVS v4** | V2.5 — Credential Recovery | Ties into registration: email verification flow must be secure |
| **OWASP ASVS v4** | V2.1.1** | Minimum password length of 12+ characters at registration |
| **GDPR Art. 6 & 7** | Legal Basis & Consent | User must consent to data processing. Email verification is legitimate interest. |
| **GDPR Art. 5(1)(c)** | Data Minimization | Collect only necessary data during registration |
| **CAN-SPAM Act** | U.S. Email Regulations | Verification emails are transactional (not marketing) — exempt from opt-out |
| **ePrivacy Directive** | EU Cookie/Email | Verification emails are necessary for service provision — no separate consent needed |
| **COPPA** | Children's Privacy | Users under 13 require parental consent (U.S.) |
| **ISO 27001:2022** | A.5.16 — Identity Management | Identity verification before granting access |

### Email Verification — Why It Matters

Email verification serves multiple security purposes:

1. **Proof of ownership**: Confirms the user controls the email address they registered with
2. **Account recovery**: Establishes a verified channel for password reset
3. **Communication**: Ensures security notifications (lockout, new device, password change) reach the right person
4. **Spam prevention**: Unverified accounts are easy to create in bulk for abuse
5. **Compliance**: Many regulations require a verified communication channel

**Verification flow security requirements (OWASP):**
- Verification token must be cryptographically random (128+ bits entropy)
- Token must be time-limited (24 hours maximum, 1 hour recommended)
- Token must be single-use
- Token must be bound to the specific email address
- Verification link must use HTTPS
- Do not reveal whether an email exists ("If the email is registered, you'll receive a verification link")

### Phone Verification — Considerations

Phone verification provides a second channel for identity assurance:
- More resistant to throwaway accounts than email alone
- Enables SMS-based MFA and notifications
- **Privacy tension**: phone numbers are PII and harder to change than email
- **Cost**: SMS delivery costs real money per message
- **Accessibility**: not everyone has a phone number (especially international users)

### Domain Allow/Block Lists

Domain restrictions are powerful gatekeeping tools:

**Allowlists** (whitelist):
- Only users with specific email domains can register
- Common in B2B: `@acme.com`, `@subsidiary.acme.com`
- Ensures only employees/partners can self-register
- Combined with SSO/SAML for enterprise identity

**Blocklists** (blacklist):
- Block known disposable/temporary email domains
- Common domains: `mailinator.com`, `guerrillamail.com`, `tempmail.com`
- There are 3,000+ known disposable email domains
- Reduces spam/throwaway account creation

### Auto-Confirmation

Auto-confirmation skips the verification step:
- **When useful**: Development environments, internal tools, trusted SSO sources
- **When dangerous**: Public-facing apps where email ownership matters
- **OWASP recommendation**: Limit unverified account capabilities (no password reset, no sensitive operations)

### Bot & Abuse Prevention at Registration

Registration endpoints are prime targets for bot abuse:

| Threat | Description | Mitigation |
|--------|-------------|------------|
| **Bulk account creation** | Bots create thousands of fake accounts | CAPTCHA, rate limiting, email verification |
| **Email bombing** | Attacker triggers verification emails to arbitrary addresses | Rate limit per email, per IP |
| **Enumeration** | Attacker discovers which emails are registered | Generic responses ("check your email") |
| **Disposable emails** | Throwaway accounts for abuse | Domain blocklisting |
| **Typosquatting** | Register with similar domain (faceb00k.com) | Domain allowlisting |

---

## Our Implementation

### Data Model

Stored in `security_settings.registration_config` (JSONB column). The config is schema-free (`map[string]any`).

**Expected JSONB fields:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `self_registration_enabled` | bool | true | Whether users can register themselves (false = invite-only) |
| `require_email_verification` | bool | true | Require email verification before account is active |
| `require_phone_verification` | bool | false | Require phone number verification before account is active |
| `allowed_email_domains` | []string | [] | If non-empty, only these email domains can register |
| `blocked_email_domains` | []string | [] | These email domains are blocked from registration |
| `auto_confirm_enabled` | bool | false | Skip verification — account is active immediately |

### API Endpoints

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `GET` | `/security-settings/registration` | `GetRegistrationConfig` | Returns the current registration configuration |
| `PUT` | `/security-settings/registration` | `UpdateRegistrationConfig` | Replaces the registration configuration |

**Example request body (PUT):**
```json
{
  "self_registration_enabled": true,
  "require_email_verification": true,
  "require_phone_verification": false,
  "allowed_email_domains": ["acme.com", "partner.acme.com"],
  "blocked_email_domains": [],
  "auto_confirm_enabled": false
}
```

### Service Layer

- **`GetRegistrationConfig(ctx, userPoolID)`** — Lazy-creates the security setting row, then returns the `registration_config` JSONB.
- **`UpdateRegistrationConfig(ctx, userPoolID, config, updatedBy, ipAddress, userAgent)`** — Calls `updateConfig` with `"registration"` config type.

The update runs in a transaction:
1. Find or create `security_settings` row for the user pool
2. Replace the `registration_config` JSONB column
3. Increment `version`
4. Create a `security_settings_audit` row with `change_type: "update_registration_config"`

### Audit Trail

Every update creates a `security_settings_audit` row:
- `change_type`: `"update_registration_config"`
- `old_config`: Previous registration JSONB
- `new_config`: New registration JSONB
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

| Setting | Consumer App | Business SaaS | Enterprise (B2B) | High Security |
|---------|:------------:|:-------------:|:-----------------:|:-------------:|
| `self_registration_enabled` | true | true | false (invite-only) | false |
| `require_email_verification` | true | true | true | true |
| `require_phone_verification` | false | false | false | true |
| `allowed_email_domains` | [] | [] | ["company.com"] | ["company.com"] |
| `blocked_email_domains` | [disposable list] | [disposable list] | [] (allowlist suffices) | [] |
| `auto_confirm_enabled` | false | false | false | false |

---

## Requirements Checklist

### Configuration Management (Admin API)
- [x] Get registration config via `GET /security-settings/registration`
- [x] Update registration config via `PUT /security-settings/registration`
- [x] Stored as JSONB for flexible schema evolution
- [x] Version tracking (auto-incremented on each update)
- [x] Audit trail (old/new config, who changed, IP, user agent)
- [x] Lazy-create on first access
- [x] OpenTelemetry span tracing
- [x] Unit tests for service layer
- [x] Unit tests for handler layer
- [ ] Validation: `allowed_email_domains` and `blocked_email_domains` must not overlap
- [ ] Validation: `auto_confirm_enabled` and `require_email_verification` must not both be true
- [ ] Validation: domain values must be valid domain format
- [ ] Sane defaults on creation

### Self-Registration Flow
- [ ] Registration endpoint respects `self_registration_enabled`
- [ ] When disabled, return 403 with clear message
- [ ] When enabled, create user in `pending` state if verification required
- [ ] Rate limit registration attempts per IP (max 5 per hour)
- [ ] CAPTCHA integration (optional, based on threat assessment)
- [ ] Generic error responses ("If the email is available, you'll receive a verification link")

### Email Verification
- [ ] Send verification email on registration when `require_email_verification` is true
- [ ] Verification token: cryptographically random, 128+ bits entropy
- [ ] Token TTL: configurable, default 24 hours
- [ ] Token is single-use — consumed on verification
- [ ] Token is bound to the specific email address
- [ ] Verification link uses HTTPS
- [ ] Resend verification endpoint with rate limiting (max 3 per address per hour)
- [ ] Account is inactive until verified (cannot login)
- [ ] Verification success activates account
- [ ] Verification expiry: send reminder or clean up pending accounts

### Phone Verification
- [ ] Request phone number in E.164 format
- [ ] Send verification code via SMS when `require_phone_verification` is true
- [ ] Verification code: 6 digits, CSPRNG
- [ ] Code TTL: 10 minutes
- [ ] Code single-use
- [ ] Rate limit SMS sends per number (max 3 per hour)
- [ ] Phone can be verified during or after registration

### Domain Restrictions
- [ ] Enforce `allowed_email_domains` during registration
- [ ] Enforce `blocked_email_domains` during registration
- [ ] Case-insensitive domain matching
- [ ] Support wildcard subdomains (e.g., `*.acme.com`)
- [ ] Update enforcement when config changes (existing users unaffected)
- [ ] Include a built-in list of known disposable email domains (optional)

### Auto-Confirmation
- [ ] Skip email verification when `auto_confirm_enabled` is true
- [ ] Account is immediately active
- [ ] Still send welcome email (but no verification required)
- [ ] Warn admin when enabling auto-confirm (audit log is sufficient)

### Anti-Abuse
- [ ] Rate limiting on registration endpoint (per IP)
- [ ] Rate limiting on verification email sends (per address and per IP)
- [ ] CAPTCHA support (optional)
- [ ] Honeypot fields (optional)
- [ ] Block known malicious IPs (integrate with threat config)
- [ ] Monitor for bulk registration patterns

### Account Enumeration Prevention
- [ ] Registration response is generic ("Check your email if the address is available")
- [ ] Timing-consistent responses (don't return faster for existing accounts)
- [ ] Login error is generic ("Invalid email or password")

### Invite-Only Registration
- [ ] When `self_registration_enabled` is false, only invited users can register
- [ ] Invite creates a pending record with a secure invite token
- [ ] Invite token has configurable TTL
- [ ] Invite token is single-use
- [ ] Invited user sets their password during registration
- [ ] Admin can manage invites (create, revoke, list, resend)

### Integration & Testing
- [ ] Unit tests for registration config service methods
- [ ] Integration test: self-registration flow with email verification
- [ ] Integration test: domain allowlist enforcement
- [ ] Integration test: domain blocklist enforcement
- [ ] Integration test: invite-only registration flow
- [ ] Integration test: auto-confirm registration flow
- [ ] Integration test: audit trail with registration config changes

---

## References

- [NIST SP 800-63A — Digital Identity Guidelines: Enrollment and Identity Proofing](https://pages.nist.gov/800-63-3/sp800-63a.html)
- [OWASP ASVS v4.0 — V2 Authentication](https://owasp.org/www-project-application-security-verification-standard/)
- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
- [GDPR Art. 6 — Lawfulness of Processing](https://gdpr-info.eu/art-6-gdpr/)
- [GDPR Art. 7 — Conditions for Consent](https://gdpr-info.eu/art-7-gdpr/)
- [GDPR Art. 5 — Principles of Processing](https://gdpr-info.eu/art-5-gdpr/)
- [COPPA — Children's Online Privacy Protection Act](https://www.ftc.gov/legal-library/browse/rules/childrens-online-privacy-protection-rule-coppa)
- [Disposable Email Domain List (GitHub)](https://github.com/disposable-email-domains/disposable-email-domains)

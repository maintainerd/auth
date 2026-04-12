# SMS Configuration

## Overview

SMS configuration defines how the authentication service sends text messages for multi-factor authentication (MFA), phone number verification, and one-time passwords (OTPs). It specifies which SMS provider to use, authentication credentials, sender identity, and delivery options.

## Industry Standards & Background

### What Is SMS in Authentication?

SMS-based messaging is widely used in authentication for delivering one-time passwords (OTPs), verification codes, and account notifications. While not the strongest second factor (NIST deprecates it for high-assurance scenarios), it remains the most accessible MFA method globally — nearly every user has a phone that can receive SMS.

### Relevant Standards

| Standard | Reference | Key Guidance |
|----------|-----------|-------------|
| **NIST SP 800-63B** | Digital Identity Guidelines — Authentication | Classifies SMS OTP as a "restricted" authenticator; acceptable for many use cases but not for AAL3 (highest assurance). Warns of SIM-swap and interception risks. |
| **ITU-T E.164** | International Telephone Numbering Plan | Defines the international phone number format (e.g., `+1234567890`) that all SMS providers require. |
| **3GPP TS 23.040** | SMS Technical Specification | The core GSM standard defining how SMS works at the network level. |
| **OWASP MFA Cheat Sheet** | Multifactor Authentication | Lists SMS as a valid MFA method with known limitations; recommends TOTP or FIDO2 as stronger alternatives. |
| **OWASP ASVS v4** | V2.7 — Out-of-Band Verifier | Defines requirements for out-of-band verification (SMS/push); recommends time-limited single-use codes. |
| **PCI DSS v4.0** | Req 8.4 | Requires MFA for personnel with administrative access; SMS is one accepted method. |
| **GDPR Art. 32** | Security of Processing | Requires appropriate technical measures, which may include MFA for data access. |
| **RFC 4226** | HOTP: An HMAC-Based One-Time Password Algorithm | Defines counter-based OTP generation (used as basis for SMS OTP code generation). |
| **RFC 6238** | TOTP: Time-Based One-Time Password Algorithm | Extends HOTP with time windows; relevant context for SMS OTP TTL choices. |

### How SMS Authentication Typically Works

1. **Provider selection**: Organizations choose an SMS gateway provider (Twilio, AWS SNS, Vonage, MessageBird) or multiple providers for redundancy.
2. **Number provisioning**: A phone number or sender ID is provisioned for sending outbound messages.
3. **Code generation**: The application generates a random 6–8 digit OTP using a cryptographically secure random number generator.
4. **Message delivery**: The OTP is sent via the provider's API. Most providers offer REST APIs with delivery receipts.
5. **Code verification**: The user submits the code within a time window (typically 5–10 minutes). The code is single-use and expires.
6. **Rate limiting**: Protections prevent abuse (e.g., max 5 SMS per phone number per hour).
7. **Delivery receipts**: Providers report delivery status (queued, sent, delivered, failed, undeliverable).

### Common Providers

| Provider | Protocol | Best For |
|----------|----------|----------|
| **Twilio** | REST API | Market leader, global coverage, excellent DX |
| **AWS SNS** | AWS SDK/API | AWS-native workloads, cost-effective |
| **Vonage** (Nexmo) | REST API | Strong in EMEA/APAC, good number management |
| **MessageBird** | REST API | European presence, omnichannel (SMS + WhatsApp + Voice) |
| **Sinch** | REST API | Telecom-grade, carrier-direct routes |

### Known Risks of SMS-Based MFA

| Risk | Description | Mitigation |
|------|-------------|------------|
| **SIM Swapping** | Attacker convinces carrier to transfer victim's number | Offer TOTP/FIDO2 as primary; SMS as fallback |
| **SS7 Interception** | Network-level interception of SMS messages | Use encrypted channels (WhatsApp, push) where possible |
| **Smishing** | Phishing via SMS to trick users into revealing codes | Education; ensure OTP messages don't contain clickable links |
| **Delivery Failures** | SMS not delivered due to network issues, roaming, or carrier filtering | Retry logic, multi-provider fallback, delivery receipt monitoring |
| **Cost Abuse** | Attackers trigger mass SMS sends to expensive destinations (toll fraud) | Rate limiting, geo-restrictions, anomaly detection |

---

## Our Implementation

### Architecture

SMS configuration is a **tenant-level** singleton resource (one config per tenant). It is stored in PostgreSQL and managed through the admin API (port 8080). The service layer reads this configuration to determine how to send SMS at runtime.

### Data Model

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `tenant_id` | UUID | Foreign key → tenant |
| `provider` | enum | SMS provider: `twilio`, `sns`, `vonage`, `messagebird` |
| `account_sid` | string | Provider account identifier |
| `auth_token_encrypted` | string | Provider auth token (encrypted at rest, excluded from JSON responses) |
| `from_number` | string | Sender phone number (E.164 format) |
| `sender_id` | string | Alphanumeric sender ID (where supported by carrier) |
| `test_mode` | bool | When true, SMS are logged but not sent |
| `status` | enum | `active` or `inactive` |
| `metadata` | JSONB | Provider-specific additional configuration |
| `created_at` | timestamp | Creation time |
| `updated_at` | timestamp | Last update time |
| `deleted_at` | timestamp | Soft-delete time (nullable) |

**Source files:**
- Model: `internal/model/sms_config.go`
- Migration: `internal/database/migration/006_create_sms_configs.go`

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/sms-config` | Get current SMS configuration |
| `PUT` | `/sms-config` | Update SMS configuration |

**Source files:**
- Handler: `internal/rest/sms_config_handler.go`
- Routes: `internal/rest/sms_config_routes.go`

### Service Layer

The service provides two operations:
- `Get` — retrieve the current SMS configuration for a tenant
- `Update` — validate and persist new SMS configuration

All operations are traced with OpenTelemetry spans. Credential fields (`auth_token_encrypted`) are excluded from API responses via `json:"-"` tags.

**Source files:**
- Service: `internal/service/sms_config_service.go`
- Repository: `internal/repository/sms_config_repository.go`

### ⚠️ Missing Components

- **DTO file**: No dedicated DTO/validation file exists for SMS config. Validation is currently handled inline or not enforced at the DTO layer.

---

## Requirements Checklist

### Core Operations
- [x] Get SMS configuration
- [x] Update SMS configuration
- [ ] Test SMS configuration (send a test message to a specified number)
- [ ] Reset to default configuration
- [ ] Verify sender number/ID with provider

### Provider Support
- [x] Twilio provider support
- [x] AWS SNS provider support
- [x] Vonage (Nexmo) provider support
- [x] MessageBird provider support
- [ ] Provider-specific validation (e.g., Twilio requires Account SID + Auth Token)
- [ ] Provider health check / connectivity test
- [ ] Automatic failover to backup provider
- [ ] Multi-provider routing (primary + fallback)

### Data Validation
- [x] Provider enum validation (twilio/sns/vonage/messagebird)
- [ ] **DTO validation file** (currently missing)
- [ ] Account SID format validation per provider
- [ ] From number E.164 format validation
- [ ] Sender ID character/length validation (max 11 alphanumeric per GSM standard)
- [ ] Metadata schema validation per provider

### Credential Security
- [x] Auth token excluded from JSON API responses (`json:"-"`)
- [x] Auth token stored encrypted at rest
- [ ] Credential rotation mechanism
- [ ] AWS IAM role-based auth for SNS (no stored credentials)
- [ ] Credential validation on save (test auth before persisting)
- [ ] API key support in addition to Account SID + Auth Token

### SMS Delivery Features
- [x] Test mode (log SMS instead of sending)
- [x] Active/inactive status toggle
- [ ] SMS sending implementation (actual provider API client)
- [ ] OTP code generation (cryptographically secure, 6–8 digits)
- [ ] OTP verification with TTL (configurable, default 5 minutes)
- [ ] OTP single-use enforcement
- [ ] OTP rate limiting per phone number
- [ ] Delivery receipt tracking
- [ ] Retry logic with provider failover
- [ ] SMS queue (async sending via background worker)
- [ ] Template support for SMS messages

### Phone Number Management
- [ ] E.164 phone number normalization
- [ ] Country code validation
- [ ] Number type detection (mobile vs. landline)
- [ ] Opt-out / unsubscribe tracking
- [ ] Number verification before sending

### Anti-Abuse & Rate Limiting
- [ ] Max SMS per phone number per hour
- [ ] Max SMS per tenant per day
- [ ] Geographic sending restrictions (block expensive destinations)
- [ ] Toll fraud detection
- [ ] Anomaly detection (sudden spike in sends)
- [ ] CAPTCHA before SMS send for public endpoints

### Monitoring & Observability
- [x] OpenTelemetry tracing on service operations
- [ ] SMS send success/failure metrics
- [ ] Delivery rate dashboard
- [ ] Provider cost tracking
- [ ] Alert on delivery rate drop
- [ ] SMS audit log (who triggered what SMS, when, to which number)

### Testing
- [x] Unit tests for service layer (100% coverage)
- [x] Unit tests for handler layer
- [ ] Unit tests for DTO validation (blocked: DTO file missing)
- [ ] Integration tests with provider sandbox/test mode
- [ ] SMS delivery end-to-end test

---

## References

- [NIST SP 800-63B — Digital Identity Guidelines: Authentication and Lifecycle Management](https://pages.nist.gov/800-63-3/sp800-63b.html)
- [ITU-T E.164 — The International Public Telecommunication Numbering Plan](https://www.itu.int/rec/T-REC-E.164)
- [RFC 4226 — HOTP: An HMAC-Based One-Time Password Algorithm](https://datatracker.ietf.org/doc/html/rfc4226)
- [RFC 6238 — TOTP: Time-Based One-Time Password Algorithm](https://datatracker.ietf.org/doc/html/rfc6238)
- [OWASP Multifactor Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Multifactor_Authentication_Cheat_Sheet.html)
- [Twilio SMS API Documentation](https://www.twilio.com/docs/sms)
- [AWS SNS Developer Guide](https://docs.aws.amazon.com/sns/latest/dg/Welcome.html)

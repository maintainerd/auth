# Email Configuration

## Overview

Email configuration defines how the authentication service sends transactional emails — verification codes, password resets, welcome messages, invitations, and other system notifications. It specifies which email provider to use, authentication credentials, sender identity, and delivery options.

## Industry Standards & Background

### What Is Transactional Email?

Transactional email is email sent to an individual in response to a specific action or event — as opposed to marketing/bulk email. Authentication services are one of the highest-volume senders of transactional email (every password reset, every verification, every MFA code).

### Relevant Standards

| Standard | Reference | Key Guidance |
|----------|-----------|-------------|
| **RFC 5321** | Simple Mail Transfer Protocol (SMTP) | Defines the SMTP protocol for email transmission between servers. All email ultimately flows over SMTP. |
| **RFC 5322** | Internet Message Format | Defines the format of email messages (headers, body, addressing). |
| **RFC 8314** | Cleartext Considered Obsolete: Use of TLS for Email | Mandates TLS (implicit or STARTTLS) for all email submission; deprecates cleartext SMTP. |
| **RFC 6376** | DomainKeys Identified Mail (DKIM) | Cryptographic signing of emails to prove domain authenticity and message integrity. |
| **RFC 7208** | Sender Policy Framework (SPF) | DNS-based mechanism declaring which IPs may send email for a domain. |
| **RFC 7489** | Domain-based Message Authentication, Reporting and Conformance (DMARC) | Policy layer that combines SPF + DKIM for domain-level email authentication. |
| **RFC 8461** | SMTP MTA Strict Transport Security (MTA-STS) | Mechanism for mail servers to declare TLS requirements and prevent downgrade attacks. |
| **CAN-SPAM Act** | US Federal Trade Commission | Requires clear sender identification, opt-out mechanisms, and honest subject lines for commercial email. |
| **GDPR Art. 6, 7** | EU General Data Protection Regulation | Requires lawful basis for processing personal data (including email addresses) and clear consent. |
| **NIST SP 800-177 Rev 1** | Trustworthy Email | Comprehensive guide on SPF, DKIM, DMARC, DANE, and TLS for email security. |

### How Transactional Email Typically Works

1. **Provider selection**: Organizations choose between self-hosted SMTP, cloud SMTP relays (Amazon SES, SendGrid, Mailgun, Postmark), or hybrid approaches.
2. **Authentication**: The sending application authenticates to the email provider using credentials (username/password, API keys, or IAM roles).
3. **Encryption**: All modern email should be sent over TLS (port 465 implicit TLS, or port 587 with STARTTLS).
4. **Domain verification**: The sending domain is verified via DNS records (SPF, DKIM, DMARC) to prove ownership and improve deliverability.
5. **Bounce handling**: Providers track bounces; persistent hard bounces should suppress future sends.
6. **Rate limiting**: Providers enforce sending limits; applications must respect them.
7. **Test mode**: Development environments use test/sandbox modes to avoid sending real emails.

### Common Providers

| Provider | Protocol | Best For |
|----------|----------|----------|
| **SMTP (self-hosted)** | SMTP | Full control, on-premise environments |
| **Amazon SES** | SMTP/API | AWS-native workloads, cost-effective at scale |
| **SendGrid** (Twilio) | SMTP/API | Developer-friendly, excellent deliverability analytics |
| **Mailgun** (Sinch) | SMTP/API | Transactional focus, good logging/webhooks |
| **Postmark** | SMTP/API | Highest deliverability rates, transactional-only |
| **Resend** | API | Modern DX, React Email templates |

---

## Our Implementation

### Architecture

Email configuration is a **tenant-level** singleton resource (one config per tenant). It is stored in PostgreSQL and managed through the admin API (port 8080). The service layer reads this configuration to determine how to send emails at runtime.

### Data Model

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `tenant_id` | UUID | Foreign key → tenant |
| `provider` | enum | Email provider: `smtp`, `ses`, `sendgrid`, `mailgun`, `postmark`, `resend` |
| `host` | string | SMTP server hostname |
| `port` | int | SMTP port (typically 25, 465, 587) |
| `username` | string | SMTP authentication username |
| `password_encrypted` | string | SMTP password (encrypted at rest, excluded from JSON responses) |
| `from_address` | string | Default sender email address |
| `from_name` | string | Default sender display name |
| `reply_to` | string | Reply-to email address |
| `encryption` | enum | `tls`, `ssl`, `none` |
| `test_mode` | bool | When true, emails are logged but not sent |
| `status` | enum | `active` or `inactive` |
| `metadata` | JSONB | Provider-specific additional configuration |
| `created_at` | timestamp | Creation time |
| `updated_at` | timestamp | Last update time |
| `deleted_at` | timestamp | Soft-delete time (nullable) |

**Source files:**
- Model: `internal/model/email_config.go`
- Migration: `internal/database/migration/005_create_email_configs.go`

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/email-config` | Get current email configuration |
| `PUT` | `/email-config` | Update email configuration |

**Source files:**
- Handler: `internal/rest/email_config_handler.go`
- Routes: `internal/rest/email_config_routes.go`

### Service Layer

The service provides two operations:
- `Get` — retrieve the current email configuration for a tenant
- `Update` — validate and persist new email configuration

All operations are traced with OpenTelemetry spans. Credential fields (`password_encrypted`) are excluded from API responses via `json:"-"` tags.

**Source files:**
- Service: `internal/service/email_config_service.go`
- Repository: `internal/repository/email_config_repository.go`

### Validation Rules

| Field | Rules |
|-------|-------|
| `provider` | Required, must be one of: `smtp`, `ses`, `sendgrid`, `mailgun`, `postmark`, `resend` |
| `host` | Required when provider is `smtp` |
| `port` | Required when provider is `smtp`, must be between 1–65535 |
| `from_address` | Required, must be a valid email format |
| `from_name` | Optional, max 255 characters |
| `reply_to` | Optional, must be valid email format if provided |
| `encryption` | Required, must be one of: `tls`, `ssl`, `none` |

**Source files:**
- DTO: `internal/dto/email_config.go`

---

## Requirements Checklist

### Core Operations
- [x] Get email configuration
- [x] Update email configuration
- [ ] Test email configuration (send a test email to a specified address)
- [ ] Reset to default configuration
- [ ] Verify domain ownership (SPF/DKIM/DMARC check)

### Provider Support
- [x] SMTP (generic) provider support
- [x] Amazon SES provider support
- [x] SendGrid provider support
- [x] Mailgun provider support
- [x] Postmark provider support
- [x] Resend provider support
- [ ] Provider-specific validation (e.g. SES requires region, SendGrid requires API key)
- [ ] Provider health check / connectivity test
- [ ] Automatic failover to backup provider

### Data Validation
- [x] Provider enum validation
- [x] Port range validation (1–65535)
- [x] From address email format validation
- [x] Encryption enum validation (tls/ssl/none)
- [ ] Host reachability validation (DNS lookup)
- [ ] Port-encryption consistency (warn if port 465 + `none`, or port 25 + `tls`)
- [ ] Reply-to email format validation on update
- [ ] From address domain verification against DNS

### Credential Security
- [x] Password excluded from JSON API responses (`json:"-"`)
- [x] Password stored encrypted at rest
- [ ] Credential rotation mechanism (update password without downtime)
- [ ] API-key-based auth for providers that support it (SendGrid, Postmark, Resend)
- [ ] AWS IAM role-based auth for SES (no stored credentials)
- [ ] Credential validation on save (test auth before persisting)

### Email Delivery Features
- [x] Test mode (log emails instead of sending)
- [x] Active/inactive status toggle
- [ ] Email sending implementation (actual SMTP/API client)
- [ ] Template rendering engine integration
- [ ] Rate limiting per provider
- [ ] Bounce handling and suppression list
- [ ] Email delivery webhooks (delivery, open, click, bounce, complaint)
- [ ] Retry logic with exponential backoff
- [ ] Email queue (async sending via background worker)

### Domain Authentication
- [ ] SPF record validation helper
- [ ] DKIM key generation and DNS record guidance
- [ ] DMARC policy validation helper
- [ ] Domain verification status tracking
- [ ] Automated DNS record check

### Monitoring & Observability
- [x] OpenTelemetry tracing on service operations
- [ ] Email send success/failure metrics
- [ ] Delivery rate dashboard
- [ ] Bounce rate monitoring
- [ ] Alert on high bounce/complaint rate
- [ ] Email audit log (who triggered what email, when, to whom)

### Testing
- [x] Unit tests for service layer (100% coverage)
- [x] Unit tests for DTO validation
- [x] Unit tests for handler layer
- [ ] Integration tests with real SMTP server (e.g., MailHog)
- [ ] Integration tests with provider APIs (sandbox mode)
- [ ] Email template rendering tests

---

## References

- [RFC 5321 — Simple Mail Transfer Protocol](https://datatracker.ietf.org/doc/html/rfc5321)
- [RFC 5322 — Internet Message Format](https://datatracker.ietf.org/doc/html/rfc5322)
- [RFC 8314 — Use of TLS for Email Submission/Access](https://datatracker.ietf.org/doc/html/rfc8314)
- [RFC 6376 — DomainKeys Identified Mail (DKIM)](https://datatracker.ietf.org/doc/html/rfc6376)
- [RFC 7208 — Sender Policy Framework (SPF)](https://datatracker.ietf.org/doc/html/rfc7208)
- [RFC 7489 — DMARC](https://datatracker.ietf.org/doc/html/rfc7489)
- [NIST SP 800-177 Rev 1 — Trustworthy Email](https://csrc.nist.gov/publications/detail/sp/800-177/rev-1/final)
- [Amazon SES Developer Guide](https://docs.aws.amazon.com/ses/latest/dg/Welcome.html)
- [SendGrid API Documentation](https://docs.sendgrid.com/)

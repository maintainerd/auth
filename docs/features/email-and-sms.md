# Email & SMS Delivery

> Tenant-scoped transactional messaging: email is delivered over SMTP only, SMS through a small set of HTTP/SDK provider adapters, both driven by DB-stored per-tenant config and DB-stored templates.

| | |
|---|---|
| **Status** | Implemented (email SMTP-only). SMS **Partial** — see [SMS provider gaps](#sms-provider-gaps); `test_mode` and `sender_id` are stored but not enforced. |
| **Code** | `internal/platform/email`, `internal/platform/sms`, `internal/notifier` (config CRUD) |
| **Endpoints** | `GET/PUT /api/v1/email-config`, `GET /api/v1/email-config/status`, `GET/PUT /api/v1/sms-config`, `GET /api/v1/sms-config/status` (control plane :8080) |
| **Storage** | Tables `email_config`, `sms_config`, `email_templates`, `sms_templates`, `user_otps` (migrations 004, 005, 058, 059) |
| **Config** | Per-tenant DB rows only (no SMTP env vars). Secrets decrypt via `APP_ENCRYPTION_KEY` (+ `APP_ENCRYPTION_KEYS_PREVIOUS`); SNS uses ambient AWS config (`AWS_REGION`, etc.). |

## Overview

Both channels deliver transactional messages triggered by auth flows — email verification, password reset, magic links, invitations, MFA step-up/enroll, phone verification, SMS login OTP, email-change OTP, CIBA and device-approval notifications. Every send is **tenant-scoped**: the sender adapter is built at send time from the calling tenant's config row, falling back to the **system tenant** (`tenants.is_system = true`) when the tenant has none.

There are two distinct layers:

- **Delivery config** (`internal/notifier`) — the CRUD surface an admin uses to set SMTP host/credentials or SMS provider/credentials, exposed on the management API.
- **Delivery adapters** (`internal/platform/email`, `internal/platform/sms`) — the runtime that reads that config, decrypts secrets, renders the DB template, and pushes the message to the transport.

Email is **SMTP-only**. There are no dedicated email-provider (SES / SendGrid / Mailgun / Postmark / Resend) API integrations in the code — any such relay is reached through its SMTP endpoint. The former multi-provider email surface has been removed; provider validation, the factory, and the DB `CHECK` constraint all now permit `smtp` exclusively.

## How it works

### Email send path
1. A caller (e.g. `internal/authn/service_email_verification.go:415`) invokes `email.SendEmail(ctx, db, SendEmailParams{TenantID, To, Subject, BodyHTML, BodyPlain, ...})`.
2. `sendEmail` (`internal/platform/email/email.go:26`) returns a no-op when `db == nil` (test seam), otherwise builds a provider via `NewProviderFromDB`.
3. `NewProviderFromDB` (`factory.go:15`) reads the active `email_config` row for the tenant; on `ErrRecordNotFound` it re-queries joined to the system tenant. The stored `password_encrypted` is decrypted with `crypto.DecryptAtRest` (the `k1:<key-id>:` at-rest envelope — not `DecryptString`).
4. `NewProvider` (`factory.go:79`) switches on `provider`: `smtp` or empty → SMTP provider; **any other value → error** `unsupported provider %q (only smtp is supported)`.
5. `smtpProvider.Send` (`smtp.go:32`) builds a `gomail` message (plain + HTML alternative when `BodyPlain` set), dials with a TLS 1.2-minimum config (`ServerName = host`), and `DialAndSend`s. An OTel span records host/port/to/subject.
6. Callers render the body first via `email.RenderTemplate(db, name, tenantID, data)` and pull the branding logo via `email.GetLogoURL(ctx, db, tenantID)`.

### SMS send path
1. A caller (e.g. `internal/authn/service_sms_login.go:180`) builds a provider via `sms.NewProviderFromDB(ctx, db, tenantID)`.
2. `NewProviderFromDB` (`internal/platform/sms/factory.go:13`) reads the active `sms_config` row (tenant → system-tenant fallback), decrypts `auth_token_encrypted` with `crypto.DecryptAtRest`, and maps the row into `ProviderConfig` — populating **only** `Provider`, `TwilioSID` (from `account_sid`), `TwilioToken`, and `TwilioFrom` (from `from_number`).
3. `NewProvider` (`factory.go:66`) switches: `twilio` → Twilio REST, `sns` → AWS SNS, `vonage` → Vonage/Nexmo REST, `log`/empty → no-op logging provider, anything else → `unknown provider` error.
4. `provider.Send(ctx, to, body)` performs the HTTP POST (Twilio/Vonage, via the shared 15s-timeout `httpClient`) or SNS `Publish`. Non-2xx → error; OTel span records `sms.to`.
5. Callers render the message first via `sms.RenderTemplate(db, name, tenantID, data)`.

### Templates
Both channels store templates per tenant in DB and cache them in Redis for 15 min.
- Email: `email.RenderTemplate` (`template.go:41`) reads `email_templates` (`subject`, `body_html`, `body_plain`) and executes with **`html/template`** (auto-escaping). Cache key `email_tpl:<tenant>:<name>`.
- SMS: `sms.RenderTemplate` (`template.go:31`) reads `sms_templates` (`message`) and executes with **`text/template`**. Cache key `sms_tpl:<tenant>:<name>`.
- `InvalidateTemplateCache(tenantID, name)` drops the cached entry. Redis is optional — `RedisClient == nil` falls straight through to the DB.

Template names in use include `user:email:verification`, `user:mfa:stepup`, `user:mfa:enroll`, `user:ciba:notification`, `user:device:approved`, `user:email:change`; SMS `sms:login:otp`, `sms:mfa:stepup`, `sms:mfa:enroll`, `sms:phone:verify`.

## Implementation

### Delivery adapters
| File | Role |
|------|------|
| `internal/platform/email/email.go` | `SendEmail` entrypoint (swappable var), `GetLogoURL` |
| `internal/platform/email/factory.go` | `NewProviderFromDB`, `NewProvider` (SMTP-only switch) |
| `internal/platform/email/smtp.go` | `smtpProvider.Send` (gomail, TLS 1.2 min) |
| `internal/platform/email/provider.go` | `Provider` interface, `ProviderConfig`, `ResolveFrom` |
| `internal/platform/email/template.go` | `RenderTemplate`, Redis cache, `InvalidateTemplateCache` |
| `internal/platform/sms/factory.go` | `NewProviderFromDB`, `NewProvider`, `logProvider` |
| `internal/platform/sms/twilio.go` | Twilio REST (`api.twilio.com/.../Messages.json`, basic auth) |
| `internal/platform/sms/sns.go` | AWS SNS `Publish` (uses `LoadDefaultConfig` + region) |
| `internal/platform/sms/vonage.go` | Vonage/Nexmo REST (`rest.nexmo.com/sms/json`) |
| `internal/platform/sms/http_client.go` | Shared `http.Client{Timeout: 15s}` |
| `internal/platform/sms/template.go` | SMS `RenderTemplate` (text/template) |

### Config CRUD (`internal/notifier`)
| File | Role |
|------|------|
| `model_email_config.go` / `model_sms_config.go` | GORM models for `email_config` / `sms_config` |
| `service_email_config.go` / `service_sms_config.go` | Get / GetStatus / Update (upsert), secret encryption |
| `handler_email_config.go` / `handler_sms_config.go` | HTTP handlers |
| `validation_email_config.go` / `validation_sms_config.go` | Request validation |
| `routes.go` | `EmailConfigRoute`, `SMSConfigRoute` |
| `model_user_otp.go` / `repository_user_otp.go` | `user_otps` (hashed OTP store shared by OTP flows) |

Routes are mounted on the management/control plane at `/api/v1` behind `RequireManagementClient`, `JWTAuthMiddleware`, `UserContextMiddleware`, tenant rate-limit, and permission middleware (`internal/server/router.go:97-98`). Tenant is resolved from the auth context, not the request body.

| Method / path | Permission | Handler |
|---|---|---|
| `GET /api/v1/email-config` | `email-config:read` | `EmailConfigHandler.Get` |
| `GET /api/v1/email-config/status` | `email-config:read` | `EmailConfigHandler.Status` |
| `PUT /api/v1/email-config` | `email-config:update` | `EmailConfigHandler.Update` |
| `GET /api/v1/sms-config` | `sms-config:read` | `SMSConfigHandler.Get` |
| `GET /api/v1/sms-config/status` | `sms-config:read` | `SMSConfigHandler.Status` |
| `PUT /api/v1/sms-config` | `sms-config:update` | `SMSConfigHandler.Update` |

`Update` is an upsert: a secret (`password` / `auth_token`) is written only when non-empty (blank preserves the stored one); if the provider changes and no new secret is supplied, the stored secret is **cleared** rather than silently reused (`service_email_config.go:176`, `service_sms_config.go:150`). Secrets are encrypted with `crypto.EncryptAtRest` before write and are `json:"-"` (never serialized back out).

`GET .../status` reports a lightweight `{configured, provider, status}` with no secrets. Email is "configured" when the row is active and has a provider, `from_address`, and SMTP `host`; SMS when active with a provider, a sender (`from_number` or `sender_id`), and an `auth_token`.

### Storage

**`email_config`** (migration `004_create_email_config_table.go`) — one row per tenant. Columns: `provider` (`CHECK IN ('smtp')`), `host`, `port`, `username`, `password_encrypted`, `from_address` (NOT NULL), `from_name`, `reply_to`, `encryption` (`CHECK IN ('tls','ssl','none')`), `logo_url`, `test_mode`, `status` (`active|inactive`), `metadata` JSONB, audit cols. FK `tenant_id → tenants ON DELETE CASCADE`.

**`sms_config`** (migration `005_create_sms_config_table.go`). Columns: `provider` (`CHECK IN ('twilio','sns','vonage','messagebird','log')`), `account_sid`, `auth_token_encrypted`, `from_number`, `sender_id`, `test_mode`, `daily_send_limit` (default 1000), `status`, `metadata`, audit cols. FK `tenant_id → tenants ON DELETE CASCADE`.

**`email_templates`** (058) / **`sms_templates`** (059) — per-tenant, unique `(tenant_id, name) WHERE deleted_at IS NULL`, `is_default`/`is_system` flags, status `active|inactive`.

**`user_otps`** (`model_user_otp.go`) — hashed OTP records (`otp_hash`, `channel`, `recipient`, `expires_at`, `used`, `failed_attempts`) backing the OTP-based flows.

### SMS provider gaps

The SMS layer is **Partial** — the config surface is broader than the runtime:

1. **`messagebird` is accepted but not implemented.** It passes request validation (`validation_sms_config.go:10`) and the DB `CHECK` (`005`), but `NewProvider` has no `messagebird` case, so a send with that provider fails at runtime with `sms: unknown provider "messagebird"`.
2. **Only Twilio-shaped fields flow from the DB.** `NewProviderFromDB` populates `Provider`, `TwilioSID`, `TwilioToken`, `TwilioFrom` only. `SNSRegion`, `VonageAPIKey`, `VonageAPISecret`, `VonageFrom` on `ProviderConfig` are settable only through the direct `NewProvider` path, never from an `sms_config` row. Consequences:
   - **Vonage via DB config** builds with empty API key/secret/from → provider auth fails.
   - **SNS via DB config** gets an empty region, so `awsconfig.LoadDefaultConfig` falls back to ambient AWS config (`AWS_REGION`/instance role); `from_number`/`sender_id` are unused (SNS sends to the recipient `PhoneNumber`).
   - **Twilio via DB config** is the fully wired path.
3. **`sender_id`** is stored and returned but is not read by any provider `Send`.
4. **`test_mode`** (both `email_config` and `sms_config`) is stored and exposed via the API but is **not** consulted anywhere in the send paths — there is no "log-only when test_mode" branch in the adapters. (This corrects the older doc claim that `test_mode` suppresses real sends.)
5. **`daily_send_limit`** is enforced for the **SMS login OTP** path only, via `security.CheckAndRecordSMSDailyBudget` under a `"global"` scope (Redis counter with in-memory fallback, `service_sms_login.go:141`; limit read from `sms_config.daily_send_limit`). Other SMS senders (MFA step-up/enroll, phone verification) do not apply this budget.

## Configuration

There are **no SMTP or SMS provider env vars** — all delivery config lives in the per-tenant DB rows above, edited through the config endpoints. The only ambient runtime inputs are:

| Env | Used by | Purpose |
|-----|---------|---------|
| `APP_ENCRYPTION_KEY` | `crypto.EncryptAtRest` / `DecryptAtRest` | Encrypts/decrypts `password_encrypted` and `auth_token_encrypted`. |
| `APP_ENCRYPTION_KEYS_PREVIOUS` | `crypto.DecryptAtRest` | Retired decrypt-only keys for key rotation (`k1:<key-id>:` envelope selects the right one). |
| `AWS_REGION` / standard AWS credential chain | SMS `sns` provider | Region + credentials for `sns.Publish` (since region isn't sourced from the DB row). |

Per-tenant settings (via `PUT /email-config` / `PUT /sms-config`): email — `provider` (`smtp`), `host`, `port`, `username`, `password`, `from_address`, `from_name`, `reply_to`, `encryption`, `logo_url`, `test_mode`; SMS — `provider`, `account_sid`, `auth_token`, `from_number`, `sender_id`, `daily_send_limit`, `test_mode`. Templates (subject/body/message) are managed as their own per-tenant resources and cached in Redis.

## Security considerations

- **Secrets encrypted at rest.** SMTP passwords and SMS auth tokens are stored `*_encrypted` using AES via `crypto.EncryptAtRest`, tagged with a `k1:<key-id>:` envelope so a rotated key still decrypts its own historical rows. They are `json:"-"` and never returned by any read endpoint; `/status` exposes only a boolean + provider + status.
- **TLS-enforced SMTP.** The dialer pins `MinVersion: TLS 1.2` with `ServerName = host`, so the SMTP session negotiates verified TLS.
- **Tenant isolation with system fallback.** Config lookups are scoped by `tenant_id` and only fall back to the `is_system` tenant when the tenant has no active row — a tenant never reads another tenant's secrets.
- **Template injection surface.** Email bodies render through `html/template` (context-aware auto-escaping). SMS renders through `text/template` (no escaping — appropriate for plaintext SMS, but template content is tenant-authored and trusted).
- **Least-privilege API.** Config changes require the `email-config:update` / `sms-config:update` permissions on the management plane behind a management-client gate; reads require the `:read` permissions.
- **Abuse throttling.** SMS login OTP is gated by a hard daily budget (`CheckAndRecordSMSDailyBudget`) plus login-threat assessment that fails silently to avoid enumeration; note this budget does not currently cover every SMS sender (see gaps).
- **Provider-switch hygiene.** Changing provider without supplying a new secret clears the stored secret rather than reusing a credential that belonged to the previous provider.

## Related
- `./multi-factor-auth.md` — SMS/email step-up and enrollment consumers
- `./authentication.md` — magic-link email and SMS-login OTP consumers
- `./secret-management.md` — the `EncryptAtRest`/`DecryptAtRest` at-rest envelope and key rotation
- `./branding-and-templates.md` — the `email_templates` / `sms_templates` management surface
- `./multi-tenancy.md` — tenant vs. system-tenant config resolution

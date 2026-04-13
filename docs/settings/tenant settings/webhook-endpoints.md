# Webhook Endpoints

## Overview

Webhook endpoints allow external systems to receive real-time HTTP notifications when events occur in the authentication service — such as user registration, login, password change, MFA enrollment, or role changes. Webhooks are the primary extensibility mechanism, enabling integrations without polling.

## Industry Standards & Background

### What Are Webhooks?

Webhooks are user-defined HTTP callbacks. When a specific event occurs, the source system makes an HTTP POST request to a pre-configured URL with a JSON payload describing the event. This "don't call us, we'll call you" pattern is the standard for event-driven integrations across the modern web.

### Relevant Standards

| Standard | Reference | Key Guidance |
|----------|-----------|-------------|
| **Standard Webhooks** | [standardwebhooks.com](https://www.standardwebhooks.com/) | Industry consortium (Svix, Kong, Twilio, etc.) defining standard webhook behaviors: HMAC signing, retry policies, timestamp tolerance, content types. |
| **RFC 9110** | HTTP Semantics | Defines the HTTP methods, status codes, and headers used in webhook delivery. |
| **RFC 6585** | Additional HTTP Status Codes (429) | Defines `429 Too Many Requests` for rate limiting — receivers may return this to throttle delivery. |
| **RFC 2104** | HMAC: Keyed-Hashing for Message Authentication | Defines HMAC construction used for webhook signature verification. |
| **RFC 6234** | US Secure Hash Algorithms (SHA-256) | Defines SHA-256 used in HMAC-SHA256 webhook signatures. |
| **CloudEvents v1.0** | [cloudevents.io](https://cloudevents.io/) | CNCF specification for describing event data in a common format (type, source, id, time). |
| **OWASP Webhook Security** | Server-Side Request Forgery (SSRF) Prevention | Guidance on validating webhook URLs to prevent SSRF attacks. |
| **PCI DSS v4.0** | Req 10.2, 10.3 | Requires audit trails of security-relevant events — webhooks can supplement this. |

### How Webhooks Typically Work

1. **Registration**: The consumer registers a webhook URL + a shared secret with the provider.
2. **Event occurs**: Something happens in the system (e.g., user created).
3. **Payload construction**: A JSON payload is created with event details, a timestamp, and an event type.
4. **Signature generation**: The payload is signed with HMAC-SHA256 using the shared secret, and the signature is sent in a header (typically `Webhook-Signature`).
5. **Delivery**: An HTTP POST is sent to the registered URL with the payload and signature.
6. **Verification**: The consumer verifies the signature by recomputing the HMAC with their copy of the secret.
7. **Acknowledgment**: The consumer returns `2xx` to acknowledge receipt. Any other response triggers retries.
8. **Retry**: On failure, the provider retries with exponential backoff (e.g., 5s, 30s, 2m, 15m, 1h, 6h, 24h).
9. **Disabling**: After repeated failures, the endpoint is automatically disabled and the admin is notified.

### Standard Webhook Signature Format

Per the Standard Webhooks specification:
```
Webhook-Id: msg_abc123
Webhook-Timestamp: 1234567890
Webhook-Signature: v1,base64(hmac-sha256(secret, "${webhook_id}.${webhook_timestamp}.${body}"))
```

### Common Event Categories

| Category | Example Events |
|----------|---------------|
| **User lifecycle** | `user.created`, `user.updated`, `user.deleted`, `user.verified` |
| **Authentication** | `auth.login.success`, `auth.login.failure`, `auth.logout`, `auth.mfa.verified` |
| **Password** | `password.changed`, `password.reset.requested`, `password.reset.completed` |
| **MFA** | `mfa.enrolled`, `mfa.disabled`, `mfa.challenge.success`, `mfa.challenge.failure` |
| **Session** | `session.created`, `session.revoked`, `session.expired` |
| **Admin** | `role.assigned`, `permission.changed`, `settings.updated`, `api_key.created` |
| **Security** | `account.locked`, `suspicious_activity.detected`, `ip.blocked` |

---

## Our Implementation

### Architecture

Webhook endpoints are a **tenant-level** resource — each tenant can configure multiple webhook endpoints, each subscribing to specific event types. They are stored in PostgreSQL and managed through the admin API (port 8080).

### Data Model

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `tenant_id` | UUID | Foreign key → tenant |
| `url` | string | The HTTPS URL to receive webhook POST requests |
| `secret_encrypted` | string | HMAC shared secret (encrypted at rest, excluded from JSON responses) |
| `events` | JSONB | Array of event types this endpoint subscribes to |
| `max_retries` | int | Maximum number of retry attempts on failure |
| `timeout_seconds` | int | HTTP request timeout for each delivery attempt |
| `status` | enum | `active`, `inactive`, or `failed` |
| `description` | string | Human-readable description |
| `metadata` | JSONB | Additional key-value metadata |
| `last_triggered_at` | timestamp | When the endpoint was last sent an event |
| `created_at` | timestamp | Creation time |
| `updated_at` | timestamp | Last update time |
| `deleted_at` | timestamp | Soft-delete time (nullable) |

**Source files:**
- Model: `internal/model/webhook_endpoint.go`
- Migration: `internal/database/migration/007_create_webhook_endpoints.go`

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/webhook-endpoints` | List all webhook endpoints for the tenant |
| `GET` | `/webhook-endpoints/{uuid}` | Get a single webhook endpoint |
| `POST` | `/webhook-endpoints` | Create a new webhook endpoint |
| `PUT` | `/webhook-endpoints/{uuid}` | Update an existing webhook endpoint |
| `DELETE` | `/webhook-endpoints/{uuid}` | Soft-delete a webhook endpoint |
| `PATCH` | `/webhook-endpoints/{uuid}/status` | Toggle endpoint status |

**Source files:**
- Handler: `internal/rest/webhook_endpoint_handler.go`
- Routes: `internal/rest/webhook_endpoint_routes.go`

### Service Layer

The service provides six operations:
- `List` — list all endpoints for a tenant
- `GetByID` — single endpoint lookup
- `Create` — validation + insert
- `Update` — validation + update
- `Delete` — soft delete
- `UpdateStatus` — toggle active/inactive/failed

All operations are traced with OpenTelemetry spans. Credential fields (`secret_encrypted`) are excluded from API responses via `json:"-"` tags.

**Source files:**
- Service: `internal/service/webhook_endpoint_service.go`
- Repository: `internal/repository/webhook_endpoint_repository.go`

### ⚠️ Missing Components

- **DTO file**: No dedicated DTO/validation file exists for webhook endpoints. Validation is handled inline or not enforced at the DTO layer.
- **Dispatcher**: No webhook dispatcher/delivery engine exists yet — endpoints can be configured but events are not actually sent.

---

## Requirements Checklist

### Core CRUD Operations
- [x] Create webhook endpoint
- [x] Read single webhook endpoint by ID
- [x] List webhook endpoints
- [x] Update webhook endpoint
- [x] Soft-delete webhook endpoint
- [x] Toggle endpoint status (active/inactive)

### Data Validation
- [ ] **DTO validation file** (currently missing)
- [ ] URL format validation (HTTPS required)
- [ ] URL SSRF prevention (block private/loopback IPs: 127.0.0.0/8, 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)
- [ ] Events array validation (must be non-empty, each event must be a known type)
- [ ] Max retries validation (reasonable range, e.g., 0–10)
- [ ] Timeout validation (reasonable range, e.g., 1–60 seconds)
- [ ] Description max length validation
- [ ] Duplicate URL prevention per tenant

### Webhook Signature & Security
- [x] Secret stored encrypted at rest
- [x] Secret excluded from JSON API responses (`json:"-"`)
- [ ] HMAC-SHA256 signature generation per Standard Webhooks spec
- [ ] Signature header format: `Webhook-Signature: v1,{base64_hmac}`
- [ ] Include `Webhook-Id` header (unique per delivery)
- [ ] Include `Webhook-Timestamp` header (Unix epoch)
- [ ] Timestamp tolerance window (reject replays older than 5 minutes)
- [ ] Secret rotation mechanism (generate new secret, grace period for old)
- [ ] Secret generation (auto-generate cryptographically random secret on create)

### Event Dispatch Engine
- [ ] Event dispatcher service (produce events from auth operations)
- [ ] HTTP POST delivery to registered endpoints
- [ ] Payload format (JSON with event type, timestamp, data)
- [ ] CloudEvents-compatible payload format (optional)
- [ ] Per-endpoint event filtering (only deliver subscribed events)
- [ ] Concurrent delivery to multiple endpoints
- [ ] Idempotency key in payload (for consumer deduplication)

### Retry & Reliability
- [x] Configurable max retries per endpoint
- [x] Configurable timeout per endpoint
- [ ] Exponential backoff retry schedule (e.g., 5s, 30s, 2m, 15m, 1h)
- [ ] Success criteria (any 2xx response = success)
- [ ] Automatic endpoint disabling after N consecutive failures
- [ ] Notification to admin when endpoint is auto-disabled
- [ ] Dead letter queue for undeliverable events
- [ ] Manual retry of specific events

### Delivery Logging
- [x] `last_triggered_at` tracking
- [ ] Delivery attempt log (timestamp, status code, duration, response body snippet)
- [ ] Delivery history API (list recent deliveries for an endpoint)
- [ ] Delivery statistics (success rate, avg latency, failure count)
- [ ] Event replay (re-send a past event to an endpoint)

### Event Registry
- [ ] Defined list of all event types
- [ ] Event type documentation (payload schema per event)
- [ ] Event type versioning (e.g., `user.created.v1`)
- [ ] Wildcard subscription (`*` = all events, `user.*` = all user events)
- [ ] Event type CRUD (admin can define custom events)

### Testing & Tooling
- [x] Unit tests for service layer (100% coverage)
- [x] Unit tests for handler layer
- [ ] Unit tests for DTO validation (blocked: DTO file missing)
- [ ] Test endpoint functionality (send sample event to an endpoint)
- [ ] Webhook debugger (show recent payloads, signatures, responses)
- [ ] Integration tests with mock HTTP receiver
- [ ] Load tests for high-throughput event delivery

### Monitoring & Observability
- [x] OpenTelemetry tracing on service operations
- [ ] Delivery success/failure metrics
- [ ] Endpoint health dashboard
- [ ] Alert on endpoint failure threshold
- [ ] Webhook audit log (who configured what, when)

---

## References

- [Standard Webhooks Specification](https://www.standardwebhooks.com/)
- [CloudEvents v1.0 Specification](https://cloudevents.io/)
- [RFC 9110 — HTTP Semantics](https://datatracker.ietf.org/doc/html/rfc9110)
- [RFC 2104 — HMAC](https://datatracker.ietf.org/doc/html/rfc2104)
- [RFC 6234 — SHA-256](https://datatracker.ietf.org/doc/html/rfc6234)
- [OWASP SSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html)
- [Svix Webhook Best Practices](https://docs.svix.com/receiving/introduction)
